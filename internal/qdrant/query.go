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

type searchRequest struct {
	Vector      any  `json:"vector"`
	Limit       int  `json:"limit"`
	WithPayload bool `json:"with_payload"`
	WithVector  bool `json:"with_vector"`
}

func (c *Client) Query(ctx context.Context, collection string, vectorName string, vector []float64, limit int) ([]SearchPoint, error) {
	if collection == "" {
		collection = c.cfg.QdrantCollection
	}

	if collection == "" {
		return nil, fmt.Errorf("qdrant collection is missing")
	}

	if limit <= 0 {
		limit = c.cfg.QdrantTopK
	}

	if vectorName == "" {
		vectorName = c.cfg.QdrantVectorName
	}

	queryBody := queryRequest{
		Query:       vector,
		Using:       vectorName,
		Limit:       limit,
		WithPayload: true,
		WithVector:  false,
	}

	queryURL := fmt.Sprintf("%s/collections/%s/points/query", c.cfg.QdrantURL, collection)
	statusCode, respBody, err := c.postQdrantJSON(ctx, queryURL, queryBody)
	if err != nil {
		return nil, err
	}

	if statusCode >= 200 && statusCode < 300 {
		return parseSearchPoints(respBody)
	}

	if statusCode != http.StatusNotFound && statusCode != http.StatusMethodNotAllowed {
		return nil, fmt.Errorf("qdrant returned HTTP %d: %s", statusCode, string(respBody))
	}

	searchVector := any(vector)
	if vectorName != "" {
		searchVector = map[string]any{
			"name":   vectorName,
			"vector": vector,
		}
	}

	searchBody := searchRequest{
		Vector:      searchVector,
		Limit:       limit,
		WithPayload: true,
		WithVector:  false,
	}

	searchURL := fmt.Sprintf("%s/collections/%s/points/search", c.cfg.QdrantURL, collection)
	statusCode, respBody, err = c.postQdrantJSON(ctx, searchURL, searchBody)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("qdrant returned HTTP %d: %s", statusCode, string(respBody))
	}

	return parseSearchPoints(respBody)
}

func (c *Client) postQdrantJSON(ctx context.Context, requestURL string, body any) (int, []byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.QdrantAPIKey != "" {
		req.Header.Set("api-key", c.cfg.QdrantAPIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, respBody, nil
}

func parseSearchPoints(respBody []byte) ([]SearchPoint, error) {
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}

	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("could not parse qdrant response: %w\nraw response: %s", err, string(respBody))
	}

	var queryResult struct {
		Points []SearchPoint `json:"points"`
	}

	if err := json.Unmarshal(envelope.Result, &queryResult); err == nil && queryResult.Points != nil {
		return queryResult.Points, nil
	}

	var searchResult []SearchPoint
	if err := json.Unmarshal(envelope.Result, &searchResult); err == nil {
		return searchResult, nil
	}

	return nil, fmt.Errorf("could not parse qdrant search points\nraw response: %s", string(respBody))
}
