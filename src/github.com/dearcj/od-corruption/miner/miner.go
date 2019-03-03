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
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type Datas struct {
	XMLName xml.Name `xml:"DATA"`
	Data    Data     `xml:"DATA"`
}

type Data struct {
	XMLName xml.Name   `xml:"DATA"`
	Records RecordList `xml:"RECORD"`
}

type Codex struct {
	XMLName xml.Name `xml:"CODEX_ST"`
	Article string   `xml:"CODEXARTICLE"`
}

type DateFormat struct {
	time.Time
}

type RecordList []Record

func (c RecordList) GetRandom() *Record {
	inx := rand.Int() % len(c)
	println(inx)
	return &c[inx]
}

func RemoveInsideBrackets(s string) string {
	inside := false

	space := regexp.MustCompile(`\s+`)
	s = space.ReplaceAllString(s, " ")
	space = regexp.MustCompile(`\'+`)
	s = space.ReplaceAllString(s, "'")

	s = strings.Replace(s, "\n", s, -1)
	s = strings.Replace(s, "  .", ".", -1)
	s = strings.Replace(s, ". ", " ", -1)
	s = strings.Replace(s, " ,", ",", -1)
	s = strings.Replace(s, " грн", "грн", -1)
	s = strings.Replace(s, " гривень", "грн", -1)
	s = strings.Replace(s, ",00", "", -1)
	s = strings.Replace(s, " неоподатковуваних", " ", -1)
	s = strings.Replace(s, " неоподаткованих", " ", -1)

	s = strings.Replace(s, " -", " ", -1)

	newstr := ""
	for _, v := range s {
		if v == '(' || v == '/' {
			inside = true
			continue
		}

		if !inside {
			newstr = newstr + string(v)
		}
		if v == ')' || v == '/' {
			inside = false
		}

	}

	return newstr
}

func ShortStatement(s string) string {
	s = strings.Replace(s, " Стаття ", "с", -1)
	s = strings.Replace(s, " cтаття ", "с", -1)
	return s
}

func ShortJob(s string) string {
	res := strings.Split(s, " ")
	if len(res) > 3 {
		return res[0] + " " + res[1] + " " + res[2]
	} else {
		return s
	}
}

func RemoveLastName(s string) string {
	res := strings.Split(s, " ")
	if len(res) > 2 {
		return res[0] + " " + res[1]
	} else {
		return s
	}
}

func (r *Record) EnoughData() bool {
	name := r.FIO
	jp := r.JobPost
	article := r.Codex.Article
	st := r.Punishment
	return len(name) > 0 && len(jp) > 0 && len(article) > 0 && len(st) > 0
}

func (r *Record) ToTweet() string {
	name := r.FIO
	jp := r.JobPost
	article := r.Codex.Article
	st := r.Punishment
	result := RemoveInsideBrackets(RemoveInsideBrackets(RemoveLastName(name)) + " " +
		ShortJob(RemoveInsideBrackets(jp)) + " " +
		RemoveInsideBrackets(ShortStatement(article)) + " " +
		RemoveInsideBrackets(st))

	if utf8.RuneCountInString(result) > 240 {
		return string([]rune(result)[:238]) + ".."
	} else {
		return result
	}
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
	rand.Seed(time.Now().Unix())
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
