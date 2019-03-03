package main

import (
	"encoding/json"
	"flag"
	botpackage "github.com/dearcj/od-corruption/bot"
	"github.com/dearcj/od-corruption/miner"
	"go.uber.org/zap"
	"io/ioutil"
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

	f, err := os.OpenFile("cached_data.json", os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		logger.Error("Can't open cache file", zap.Error(err))
	}

	b, err := ioutil.ReadFile("cached_data.json") // just pass the file name
	if err != nil {
		logger.Error("Can't open cache file", zap.Error(err))
	}

	cachedPosts, _ := readCachedPosts(b)

	defer f.Close()
	defer saveCache(f, cachedPosts)

	accessToken := flag.String("ACCESS_TOKEN", "", "access token")
	accessTokenSecret := flag.String("ACCESS_TOKEN_SECRET", "", "access token")
	consumerKey := flag.String("CONSUMER_KEY", "", "access token")
	consumerSecret := flag.String("CONSUMER_SECRET", "", "access token")
	flag.Parse()

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

	_, parsed := miner.StartMining(logger, OpenDataLink, 3*time.Hour)

	for {
		data := <-parsed
		filterRecords(data, time.Date(2019, 2, 20, 0, 0, 0, 0, time.UTC))
		sortRecords(data)
		excludeSent(data, cachedPosts.PostIds)
		if len(data.Records) > 0 {
			rec := data.Records[0]
			err := bot.Post(rec.ToTweet(), logger)
			if err == nil {
				cachedPosts.PostIds = append(cachedPosts.PostIds, rec.RegNum)
				saveCache(f, cachedPosts)
			}
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

func filterRecords(d *miner.Data, afterTime time.Time) {
	var recs []miner.Record
	for _, x := range d.Records {
		if x.EnoughData() && x.Date.After(afterTime) {
			recs = append(recs, x)
		}
	}

	d.Records = recs
}

func readCachedPosts(b []byte) (*CachedPosts, error) {
	var cachedPosts = &CachedPosts{}

	if len(b) != 0 {
		err := json.Unmarshal(b, cachedPosts)
		if err != nil {
			return nil, err
		}
	}

	return cachedPosts, nil
}

func saveCache(f *os.File, posts *CachedPosts) error {
	bytea, err := json.Marshal(posts)
	if err != nil {
		return err
	} else {
		_, err = f.Write(bytea)
		if err != nil {
			return err
		} else {
			return nil
		}
	}
}

func sortRecords(d *miner.Data) {
	sort.Slice(d.Records, func(i, j int) bool { return d.Records[i].Date.After(d.Records[j].Date.Time) })
}
