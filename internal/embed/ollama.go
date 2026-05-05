// internal/embed/ollama.go
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

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaProvider(cfg config.Config) *OllamaProvider {
	return &OllamaProvider{
		baseURL: cfg.OllamaURL,
		model:   cfg.OllamaEmbedModel,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *OllamaProvider) ModelName() string {
	return p.model
}

func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	body := map[string]any{
		"model":  p.model,
		"prompt": text,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+"/api/embeddings",
		bytes.NewReader(raw),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed struct {
		Embedding []float64 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama embeddings returned HTTP %d", resp.StatusCode)
	}

	if len(parsed.Embedding) == 0 {
		return nil, fmt.Errorf("ollama returned empty embedding")
	}

	return parsed.Embedding, nil
}
