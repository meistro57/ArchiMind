// internal/embed/provider.go
package embed

import "context"

type Provider interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	ModelName() string
}
