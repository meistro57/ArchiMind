// internal/skills/registry.go
package skills

import "fmt"

type Registry struct {
	skills map[string]Skill
}

func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]Skill),
	}
}

func (r *Registry) Register(skill Skill) {
	r.skills[skill.Name()] = skill
}

func (r *Registry) Get(name string) (Skill, error) {
	skill, ok := r.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	return skill, nil
}

func (r *Registry) List() []Skill {
	out := make([]Skill, 0, len(r.skills))

	for _, skill := range r.skills {
		out = append(out, skill)
	}

	return out
}
