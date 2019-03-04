package main

import (
	"encoding/json"
	"flag"
	botpackage "github.com/dearcj/od-corruption/bot"
	"github.com/dearcj/od-corruption/miner"
	"github.com/go-redis/redis"
	"github.com/karrick/tparse/v2"
	"go.uber.org/zap"
	"os"
	"sort"
	"time"
)

const OpenDataLink = "https://data.gov.ua/dataset/78a73a00-4b71-4d03-8fcd-e71909c29731/resource/77848dfa-3c43-45db-bea4-0ee8871645af/download/24-ex_xml_corrupt.zip"

type CachedPosts struct {
	PostIds []int `json:"post_ids"`
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	accessToken := flag.String("ACCESS_TOKEN", "", "access token")
	accessTokenSecret := flag.String("ACCESS_TOKEN_SECRET", "", "access token")
	consumerKey := flag.String("CONSUMER_KEY", "", "access token")
	consumerSecret := flag.String("CONSUMER_SECRET", "", "access token")
	redisAddress := flag.String("REDIS_ADDRESS", "localhost:6379", "redis database server")
	redisPassword := flag.String("REDIS_PASSWORD", "", "redis database server")
	delay := flag.String("DELAY", "30s", "redis database server")

	flag.Parse()

	restime, err := tparse.AddDuration(time.Unix(0, 0), *delay)
	if err != nil {
		logger.Error("Could not parse bot delay")
		panic(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     *redisAddress,
		Password: *redisPassword,
		DB:       0,
	})

	_, err = client.Ping().Result()
	if err != nil {
		panic(err)
	}

	val, _ := client.Get("cached_data").Result()
	cachedPosts, err := cachedDataFromBytea([]byte(val))
	if err != nil {
		panic(err)
	}

	defer saveCache(logger, client, cachedPosts)

	if *accessToken == "" || *accessTokenSecret == "" || *consumerKey == "" || *consumerSecret == "" {
		logger.Error("No access tokens")
		os.Exit(0)
	}

	bot := botpackage.New().Start(&botpackage.Credentials{
		AccessToken:       *accessToken,
		AccessTokenSecret: *accessTokenSecret,
		ConsumerKey:       *consumerKey,
		ConsumerSecret:    *consumerSecret,
	}, logger)

	if err != nil {
		logger.Error("Can't create a twitter bot", zap.Error(err))
	}

	_, parsed := miner.StartMining(logger, OpenDataLink, time.Duration(restime.UnixNano()))

	for {
		data := <-parsed
		FilterRecords(logger, data, time.Date(2019, 2, 20, 0, 0, 0, 0, time.UTC))
		SortRecords(data)
		excludeSent(data, cachedPosts.PostIds)
		if len(data.Records) > 0 {
			rec := data.Records[0]
			cachedPosts.PostIds = append(cachedPosts.PostIds, rec.RegNum)
			saveCache(logger, client, cachedPosts)
			tweet := rec.ToTweet()
			logger.Info("Posting", zap.String("tweet", tweet))

			err := bot.Post(tweet)
			if err != nil {
				logger.Error("Can't update status", zap.Error(err))
			}
		} else {
			logger.Error("Nothing to post")
		}
	}
}

func excludeSent(d *miner.Data, toexclude []int) {
	var recs []miner.Record
	var exclude = make(map[int]struct{})
	for _, v := range toexclude {
		exclude[v] = struct{}{}
	}

	for _, x := range d.Records {
		if _, ok := exclude[x.RegNum]; ok {
			continue
		} else {
			recs = append(recs, x)
		}
	}

	d.Records = recs
}

func FilterRecords(logger *zap.Logger, d *miner.Data, afterTime time.Time) {
	var recs []miner.Record
	lackDataCount := 0
	tooOldCount := 0

	for _, x := range d.Records {
		if !x.Date.After(afterTime) {
			tooOldCount++
			continue
		}

		if !x.EnoughData() {
			lackDataCount++
			continue
		}

		recs = append(recs, x)
	}

	logger.Info("", zap.Int("Records left", len(recs)), zap.Int("Excluded records by date", tooOldCount), zap.Int("Excluded records by lack of data", lackDataCount))

	d.Records = recs
}

func cachedDataFromBytea(b []byte) (*CachedPosts, error) {
	var cachedPosts = &CachedPosts{}

	if len(b) != 0 {
		err := json.Unmarshal(b, cachedPosts)
		if err != nil {
			return nil, err
		}
	}

	return cachedPosts, nil
}

func saveCache(logger *zap.Logger, client *redis.Client, posts *CachedPosts) error {
	bytes, err := json.Marshal(posts)
	if err != nil {
		return err
	} else {
		_, err := client.Set("cached_data", bytes, 0).Result()
		if err != nil {
			logger.Error("Can't save cached data to redis", zap.Error(err))
			return err
		}
	}
	logger.Info("Cache saved to redis")
	return nil
}

func SortRecords(d *miner.Data) {
	sort.Slice(d.Records, func(i, j int) bool { return d.Records[i].Date.Before(d.Records[j].Date.Time) })
}
