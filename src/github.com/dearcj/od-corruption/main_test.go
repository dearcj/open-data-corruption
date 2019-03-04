package main_test

import (
	"github.com/dearcj/od-corruption/miner"
	"go.uber.org/zap"
	"testing"
	"time"

	m "github.com/dearcj/od-corruption"
)

func TestFilter(t *testing.T) {
	logger, _ := zap.NewProduction()

	d := &miner.Data{
		Records: []miner.Record{
			{
				Date: miner.DateFormat{Time: time.Now()},
			},
			{
				Date: miner.DateFormat{Time: time.Now()},
			},
			{
				Date: miner.DateFormat{Time: time.Now()},
			},
			{
				Date:       miner.DateFormat{Time: time.Now().Add(time.Hour)},
				FIO:        "Yuriy Nikolaevich",
				Punishment: "Some punishment",
				JobPost:    "Cashier",
			},
			{
				Date: miner.DateFormat{Time: time.Now().Add(time.Hour)},
			},
		},
	}
	m.FilterRecords(logger, d, time.Now())

	if len(d.Records) != 1 {
		t.Error("Records aren't filtered correctly")
	} else {
		if d.Records[0].FIO == "" {
			t.Error("Records aren't filtered correctly")
		}
	}

}

func TestSort(t *testing.T) {
	d := &miner.Data{
		Records: []miner.Record{
			{
				Date: miner.DateFormat{Time: time.Now()},
			},
			{
				Date: miner.DateFormat{Time: time.Now().Add(time.Hour)},
			},
		},
	}
	m.SortRecords(d)

	if d.Records[0].Date.After(d.Records[1].Date.Time) {
		t.Error("Records sorted in a wrong order")
	}
}
