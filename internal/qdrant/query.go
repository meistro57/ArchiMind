// internal/qdrant/query.go
package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type SearchPoint struct {
	ID      any            `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
}

type queryRequest struct {
	Query       []float64 `json:"query"`
	Using       string    `json:"using,omitempty"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
	WithVector  bool      `json:"with_vector"`
}

type queryResponse struct {
	Result struct {
		Points []SearchPoint `json:"points"`
	} `json:"result"`
	Status any `json:"status"`
	Time   any `json:"time"`
}

func (c *Client) Query(ctx context.Context, collection string, vector []float64) ([]SearchPoint, error) {
	if collection == "" {
		collection = c.cfg.QdrantCollection
	}

	if collection == "" {
		return nil, fmt.Errorf("qdrant collection is missing")
	}

	body := queryRequest{
		Query:       vector,
		Using:       c.cfg.QdrantVectorName,
		Limit:       c.cfg.QdrantTopK,
		WithPayload: true,
		WithVector:  false,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/collections/%s/points/query", c.cfg.QdrantURL, collection)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if c.cfg.QdrantAPIKey != "" {
		req.Header.Set("api-key", c.cfg.QdrantAPIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("qdrant returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed queryResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("could not parse qdrant response: %w\nraw response: %s", err, string(respBody))
	}

	return parsed.Result.Points, nil
}
