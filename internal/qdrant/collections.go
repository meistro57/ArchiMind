// internal/qdrant/collections.go
package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type CollectionInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Raw    any    `json:"raw"`
}

type vectorConfig struct {
	Size     int    `json:"size"`
	Distance string `json:"distance"`
}

// collectionInfoResponse covers the Qdrant /collections/{name} response shape.
type collectionInfoResponse struct {
	Result struct {
		Config struct {
			Params struct {
				// RawMessage lets us handle both named-vector maps and single-vector objects.
				Vectors json.RawMessage `json:"vectors"`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
	Status any `json:"status"`
	Time   any `json:"time"`
}

func (c *Client) CollectionInfo(ctx context.Context, collection string) (*CollectionInfo, error) {
	if collection == "" {
		collection = c.cfg.QdrantCollection
	}

	if collection == "" {
		return nil, fmt.Errorf("qdrant collection is missing")
	}

	url := fmt.Sprintf("%s/collections/%s", c.cfg.QdrantURL, collection)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if c.cfg.QdrantAPIKey != "" {
		req.Header.Set("api-key", c.cfg.QdrantAPIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("qdrant collection info returned HTTP %d", resp.StatusCode)
	}

	return &CollectionInfo{
		Name:   collection,
		Status: "ok",
		Raw:    raw,
	}, nil
}

// VectorSize returns the expected vector dimension for a named vector in a Qdrant collection.
// It makes a GET request to /collections/{collection} and parses result.config.params.vectors.
func (c *Client) VectorSize(ctx context.Context, collection, vectorName string) (int, error) {
	if collection == "" {
		collection = c.cfg.QdrantCollection
	}
	if vectorName == "" {
		vectorName = c.cfg.QdrantVectorName
	}
	if collection == "" {
		return 0, fmt.Errorf("qdrant collection is missing")
	}
	if vectorName == "" {
		return 0, fmt.Errorf("qdrant vector name is missing")
	}

	url := fmt.Sprintf("%s/collections/%s", c.cfg.QdrantURL, collection)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	if c.cfg.QdrantAPIKey != "" {
		req.Header.Set("api-key", c.cfg.QdrantAPIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("qdrant collection info returned HTTP %d", resp.StatusCode)
	}

	var parsed collectionInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, fmt.Errorf("could not parse collection info: %w", err)
	}

	if parsed.Result.Config.Params.Vectors == nil {
		return 0, fmt.Errorf("no vectors config found in collection %q", collection)
	}

	// Try to parse as named-vector map: {"claims_vec": {"size": 1536, ...}, ...}
	var named map[string]vectorConfig
	if err := json.Unmarshal(parsed.Result.Config.Params.Vectors, &named); err != nil {
		return 0, fmt.Errorf("could not parse named vectors config for collection %q: %w", collection, err)
	}

	vc, ok := named[vectorName]
	if !ok {
		return 0, fmt.Errorf("vector %q not found in collection %q", vectorName, collection)
	}

	if vc.Size == 0 {
		return 0, fmt.Errorf("vector %q has size 0 in collection %q — check collection config", vectorName, collection)
	}

	return vc.Size, nil
}
