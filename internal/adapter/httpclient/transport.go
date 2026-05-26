package httpclient

import (
	"bytes"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

type loggingTransport struct {
	base http.RoundTripper
}

func (lt *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Info().
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Interface("headers", req.Header).
		Msg("outgoing http request")

	if req.Body != nil {
		reqBody, err := io.ReadAll(req.Body)
		if err == nil {
			log.Debug().
				Bytes("body", reqBody).
				Msg("http request body")

			req.Body = io.NopCloser(
				bytes.NewBuffer(reqBody),
			)
		}
	}

	resp, err := lt.base.RoundTrip(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Msg("http request failed")

		return nil, err
	}

	log.Info().
		Int("status_code", resp.StatusCode).
		Str("status", resp.Status).
		Interface("headers", resp.Header).
		Msg("incoming http response")

	if resp.Body != nil {
		respBody, err := io.ReadAll(resp.Body)
		if err == nil {
			log.Debug().
				Bytes("body", respBody).
				Msg("http response body")

			resp.Body = io.NopCloser(
				bytes.NewBuffer(respBody),
			)
		}
	}

	return resp, nil
}
