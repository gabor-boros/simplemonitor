package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// Resource represents an endpoint to be monitored.
type Resource struct {
	Timeout            time.Duration
	URL                *url.URL
	ExpectedStatusCode int
}

func (r *Resource) Ping(ctx context.Context) error {
	slog.Info("pinging resource", "url", r.URL.String())

	client := &http.Client{Timeout: r.Timeout}
	resp, err := client.Get(r.URL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body []byte
	if resp.Body != nil {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}

	if resp.StatusCode != r.ExpectedStatusCode {
		slog.Error("unexpected status code", "url", r.URL.String(), "status_code", resp.StatusCode, "body", string(body))
		return fmt.Errorf("unexpected status code: got %d, want %d", resp.StatusCode, r.ExpectedStatusCode)
	}

	slog.Info("response received", "url", r.URL.String(), "status_code", resp.StatusCode, "body", string(body))
	return nil
}

func NewResource(endpoint string, status int, timeout time.Duration) *Resource {
	if endpoint == "" {
		panic("endpoint cannot be empty")
	}

	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		panic(fmt.Sprintf("invalid endpoint: %s", err))
	}

	return &Resource{
		Timeout:            timeout * time.Second,
		URL:                parsedURL,
		ExpectedStatusCode: status,
	}
}
