// internal/llm/openrouter.go
package llm

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
	model    string
	siteURL  string
	siteName string
	client   *http.Client
}

func NewOpenRouterProvider(cfg config.Config) *OpenRouterProvider {
	return &OpenRouterProvider{
		apiKey:   cfg.OpenRouterAPIKey,
		model:    cfg.OpenRouterModel,
		siteURL:  cfg.OpenRouterSiteURL,
		siteName: cfg.OpenRouterSiteName,
		client: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func (p *OpenRouterProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("OPENROUTER_API_KEY is missing")
	}

	body := chatRequest{
		Model:       p.model,
		Messages:    messages,
		Temperature: 0.2,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://openrouter.ai/api/v1/chat/completions",
		bytes.NewReader(raw),
	)
	if err != nil {
		return "", err
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
		return "", err
	}
	defer resp.Body.Close()

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil {
			return "", fmt.Errorf("openrouter error: %s", parsed.Error.Message)
		}
		return "", fmt.Errorf("openrouter returned HTTP %d", resp.StatusCode)
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openrouter returned no choices")
	}

	return parsed.Choices[0].Message.Content, nil
}
