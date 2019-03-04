package bot

import (
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"go.uber.org/zap"
)

type Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

type Bot struct {
	client *twitter.Client
}

func (b *Bot) Start(creds *Credentials, logger *zap.Logger) *Bot {
	logger.Info("", zap.Any("Creds", creds))

	config := oauth1.NewConfig(creds.ConsumerKey, creds.ConsumerSecret)
	token := oauth1.NewToken(creds.AccessToken, creds.AccessTokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	b.client = twitter.NewClient(httpClient)
	return b
}

func (b *Bot) Post(s string) error {
	_, _, err := b.client.Statuses.Update(s, nil)
	return err
}

func New() *Bot {
	return &Bot{}
}
