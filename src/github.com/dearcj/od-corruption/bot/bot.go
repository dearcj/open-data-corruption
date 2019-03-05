package bot

import (
	"github.com/dearcj/od-corruption/miner"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"go.uber.org/zap"
	"time"
)

type Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

type Bot struct {
	client  *twitter.Client
	records []miner.Record
}

func (b *Bot) Start(creds *Credentials, logger *zap.Logger, delay time.Duration, saveID chan int) (*Bot, chan []miner.Record) {
	logger.Info("", zap.Any("Creds", creds))

	config := oauth1.NewConfig(creds.ConsumerKey, creds.ConsumerSecret)
	token := oauth1.NewToken(creds.AccessToken, creds.AccessTokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	b.client = twitter.NewClient(httpClient)
	addRecords := make(chan []miner.Record)

	timer := time.NewTimer(delay)
	go func() {
		for {
			select {
			case newRecs := <-addRecords:
				b.records = append(b.records, newRecs...)
				break
			case <-timer.C:
				if len(b.records) > 0 {
					tweet := b.records[0].ToTweet()
					logger.Info("Posting", zap.String("tweet", tweet))
					err := b.Post(tweet)
					if err != nil {
						logger.Error("Can't update status", zap.Error(err))
						break
					}

					saveID <- b.records[0].RegNum
					b.records = b.records[1:]

				}
				timer.Reset(delay)
				break
			}
		}
	}()

	return b, addRecords
}

func (b *Bot) Post(s string) error {
	_, _, err := b.client.Statuses.Update(s, nil)
	return err
}

func New() *Bot {
	return &Bot{}
}
