// internal/llm/provider.go
package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Provider interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}
