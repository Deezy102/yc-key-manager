package logger

import (
	"bytes"
	"net/http"

	"github.com/sirupsen/logrus"
)

const stream = "key-manager"

type HttpHook struct {
	url string
}

func NewHook(url string) HttpHook {
	return HttpHook{url: url}
}

func (h HttpHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h HttpHook) Fire(entry *logrus.Entry) error {
	entry = entry.WithField("stream_name", stream)
	bytesData, err := entry.Bytes()
	if err != nil {
		return err
	}

	b := bytes.NewReader(bytesData)
	resp, err := http.Post(h.url, "application/json", b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
