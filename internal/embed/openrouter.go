// internal/embed/openrouter.go
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	var parsed openRouterEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil {
			return nil, fmt.Errorf("openrouter embedding error: %s", parsed.Error.Message)
		}
		return nil, fmt.Errorf("openrouter embedding returned HTTP %d", resp.StatusCode)
	}

	if len(parsed.Data) == 0 || len(parsed.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("openrouter returned empty embedding")
	}

	return parsed.Data[0].Embedding, nil
}
