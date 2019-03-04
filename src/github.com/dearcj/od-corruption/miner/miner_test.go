package miner_test

import (
	"github.com/dearcj/od-corruption/miner"
	"go.uber.org/zap"
	"testing"
)

const OpenDataLink = "https://data.gov.ua/dataset/78a73a00-4b71-4d03-8fcd-e71909c29731/resource/77848dfa-3c43-45db-bea4-0ee8871645af/download/24-ex_xml_corrupt.zip"

func TestDownloadAndUnzip(t *testing.T) {
	logger, _ := zap.NewProduction()
	c := make(chan *miner.Data)

	go func() {
		err := miner.ExecMining(logger, OpenDataLink, c)
		if err != nil {
			t.Error(err)
		}
	}()

	_ = <-c
}
