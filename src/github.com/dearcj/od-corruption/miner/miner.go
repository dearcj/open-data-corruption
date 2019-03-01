package miner

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/text/encoding/charmap"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Datas struct {
	XMLName xml.Name `xml:"DATA"`
	Data    Data     `xml:"DATA"`
}

type Data struct {
	XMLName xml.Name  `xml:"DATA"`
	Records []*Record `xml:"RECORD"`
}

type Codex struct {
	XMLName xml.Name `xml:"CODEX_ST"`
	Article string   `xml:"CODEXARTICLE"`
}

type DateFormat struct {
	time.Time
}

func (c *DateFormat) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	const shortForm = "2006-01-02T15:04:05.000Z" // yyyymmdd date format
	var v string
	d.DecodeElement(&v, &start)
	parse, err := time.Parse(shortForm, v+"T00:00:00.000Z")
	if err != nil {
		return err
	}
	*c = DateFormat{parse}
	return nil
}

type Record struct {
	Date       DateFormat `xml:"JUDGMENTDATE"`
	XMLName    xml.Name   `xml:"RECORD"`
	Codex      Codex      `xml:"CODEX_ST"`
	RegNum     int        `xml:"REG_NUM"`
	FIO        string     `xml:"FIO"`
	Punishment string     `xml:"PUNISHMENT"`
	JobPlace   string     `xml:"JOBPLACE"`
	JobPost    string     `xml:"JOBPOST"`
}

func execMining(logger *zap.Logger, link string, output chan *Data) {
	response, err := http.Get(link)
	if err != nil {
		println(response)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Error("Failed to read response.body", zap.Error(err))
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		logger.Error("Failed to create zip reader", zap.Error(err))
		return
	}

	for _, zipFile := range zipReader.File {
		fmt.Println("Reading file:", zipFile.Name)
		rc, err := zipFile.Open()
		if err != nil {
			log.Println(err)
			return
		}
		unzippedBytes, err := ioutil.ReadAll(rc)
		rc.Close()
		var data Data

		d := xml.NewDecoder(bytes.NewReader(unzippedBytes))
		d.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
			switch charset {
			case "windows-1251":
				return charmap.Windows1251.NewDecoder().Reader(input), nil
			default:
				return nil, fmt.Errorf("unknown charset: %s", charset)
			}
		}
		err = d.Decode(&data)

		if err != nil {
			logger.Error("Can't parse xml", zap.Error(err))
			return
		}

		output <- &data
	}
}

func StartMining(logger *zap.Logger, link string, interval time.Duration) (chan struct{}, chan *Data) {
	var done = make(chan struct{})
	var onRecieve = make(chan *Data, 1)

	defer logger.Debug("Mining ended")
	go func() {
		execMining(logger, link, onRecieve)

		timer := time.NewTimer(interval)
	outer:
		for {
			select {
			case <-done:
				return
				break outer
			case <-timer.C:
				execMining(logger, link, onRecieve)
				timer.Reset(interval)
				break

			}
		}

	}()

	return done, onRecieve
}
