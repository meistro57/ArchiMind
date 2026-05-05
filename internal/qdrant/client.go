// internal/qdrant/client.go
package qdrant

import (
	"net/http"
	"time"

	"archimind/internal/config"
)

type Client struct {
	cfg    config.Config
	client *http.Client
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}
