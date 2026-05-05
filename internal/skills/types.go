// internal/skills/types.go
package skills

import "context"

type Skill interface {
	Name() string
	Description() string
	Run(ctx context.Context, input map[string]any) (any, error)
}
