package httpclient

import (
	"bytes"
	"net/http"
	"payment-service/config"
	"time"

	"github.com/rs/zerolog/log"
)

type Client interface {
	CallURL(method string, url string, header map[string]string, rawData []byte) (*http.Response, error)
}

type client struct {
	http *http.Client
}

func NewClient(cfg *config.Config) Client {
	httpClient := &http.Client{
		Timeout: time.Duration(cfg.App.ServerTimeOut) * time.Second,
		Transport: &loggingTransport{
			base: http.DefaultTransport,
		},
	}

	return &client{
		http: httpClient,
	}
}

func (c *client) CallURL(method string, url string, header map[string]string, rawData []byte) (*http.Response, error) {
	req, err := http.NewRequest(
		method,
		url,
		bytes.NewBuffer(rawData),
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", method).
			Str("url", url).
			Msg("failed create request")

		return nil, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", method).
			Str("url", url).
			Msg("failed do request")

		return nil, err
	}

	return resp, nil
}
