// internal/embed/openrouter.go
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"archimind/internal/config"
)

type OpenRouterProvider struct {
	apiKey   string
	baseURL  string
	model    string
	siteURL  string
	siteName string
	client   *http.Client
}

func NewOpenRouterProvider(cfg config.Config) *OpenRouterProvider {
	return &OpenRouterProvider{
		apiKey:   cfg.OpenRouterAPIKey,
		baseURL:  cfg.OpenRouterEmbedBaseURL,
		model:    cfg.OpenRouterEmbedModel,
		siteURL:  cfg.OpenRouterSiteURL,
		siteName: cfg.OpenRouterSiteName,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *OpenRouterProvider) ModelName() string {
	return p.model
}

type openRouterEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openRouterEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

func (p *OpenRouterProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is missing")
	}

	body := openRouterEmbeddingRequest{
		Model: p.model,
		Input: text,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+"/embeddings",
		bytes.NewReader(raw),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	if p.siteURL != "" {
		req.Header.Set("HTTP-Referer", p.siteURL)
	}

	if p.siteName != "" {
		req.Header.Set("X-OpenRouter-Title", p.siteName)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed openRouterEmbeddingResponse
	if err := json.Unmarshal(rawResp, &parsed); err != nil {
		return nil, err
	}

	if parsed.Error != nil && parsed.Error.Message != "" {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("openrouter embedding error: %s", parsed.Error.Message)
		}
		return nil, fmt.Errorf("openrouter embedding response error: %s", parsed.Error.Message)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openrouter embedding returned HTTP %d", resp.StatusCode)
	}

	embedding := extractEmbedding(parsed, rawResp)
	if len(embedding) == 0 {
		return nil, fmt.Errorf("openrouter returned empty embedding")
	}

	return embedding, nil
}

func extractEmbedding(parsed openRouterEmbeddingResponse, rawResp []byte) []float64 {
	if len(parsed.Data) > 0 && len(parsed.Data[0].Embedding) > 0 {
		return parsed.Data[0].Embedding
	}

	var payload map[string]any
	if err := json.Unmarshal(rawResp, &payload); err != nil {
		return nil
	}

	candidates := []any{
		payload["embedding"],
		payload["vector"],
		payload["embeddings"],
	}

	if data, ok := payload["data"].([]any); ok && len(data) > 0 {
		candidates = append(candidates, data[0])
	}

	for _, candidate := range candidates {
		if vector := extractVector(candidate); len(vector) > 0 {
			return vector
		}
	}

	return nil
}

func extractVector(value any) []float64 {
	switch typed := value.(type) {
	case []float64:
		return typed
	case []any:
		vector := make([]float64, 0, len(typed))
		for _, item := range typed {
			number, ok := item.(float64)
			if !ok {
				return nil
			}
			vector = append(vector, number)
		}
		return vector
	case map[string]any:
		nestedKeys := []string{"embedding", "vector", "values", "data", "embeddings"}
		for _, key := range nestedKeys {
			if nested, ok := typed[key]; ok {
				if vector := extractVector(nested); len(vector) > 0 {
					return vector
				}
			}
		}
	}

	return nil
}
