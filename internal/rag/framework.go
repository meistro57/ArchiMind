package rag

import (
	"context"
	"fmt"
	"strings"
)

type FrameworkComponent struct {
	Name              string `json:"name"`
	Principle         string `json:"principle"`
	SupportingSources []int  `json:"supporting_sources,omitempty"`
}

type FrameworkResult struct {
	Topic           string               `json:"topic"`
	Collection      string               `json:"collection"`
	Summary         string               `json:"summary"`
	Components      []FrameworkComponent `json:"components"`
	Themes          []Theme              `json:"themes,omitempty"`
	Contradictions  []Contradiction      `json:"contradictions,omitempty"`
	SourceInfluence []SourceInfluence    `json:"source_influence,omitempty"`
	StrongClaims    []StrongClaim        `json:"strong_claims,omitempty"`
	Sources         []Source             `json:"sources,omitempty"`
}

func (e *Engine) ExtractFramework(ctx context.Context, sessionID string, collection string, topic string) (FrameworkResult, error) {
	sessionID = strings.TrimSpace(sessionID)
	collection = strings.TrimSpace(collection)
	topic = strings.TrimSpace(topic)

	if sessionID == "" {
		sessionID = "default"
	}
	if topic == "" {
		return FrameworkResult{}, fmt.Errorf("topic is required")
	}

	_ = e.mem.SaveTurn(ctx, sessionID, "user", "[framework] "+topic, e.cfg.ChatHistoryTurns)

	vector, err := e.getEmbedding(ctx, topic)
	if err != nil {
		return FrameworkResult{}, err
	}

	points, err := e.getQdrantResults(ctx, collection, topic, vector)
	if err != nil {
		return FrameworkResult{}, err
	}
	if len(points) == 0 {
		return FrameworkResult{}, fmt.Errorf("no relevant chunks found for framework extraction")
	}

	sources := pointsToSources(points)
	themes := ExtractRecurringThemes(sources, 6)
	contradictions := ExtractContradictions(sources, 4)
	influence := RankSourceInfluence(sources, themes, contradictions, 6)
	strongClaims := RankStrongClaims(sources, themes, contradictions, 6)
	components := BuildFrameworkComponents(topic, themes, sources, 4)
	summary := BuildFrameworkSummary(topic, components, contradictions, influence)

	_ = e.mem.SaveTurn(ctx, sessionID, "assistant", summary, e.cfg.ChatHistoryTurns)

	return FrameworkResult{
		Topic:           topic,
		Collection:      collection,
		Summary:         summary,
		Components:      components,
		Themes:          themes,
		Contradictions:  contradictions,
		SourceInfluence: influence,
		StrongClaims:    strongClaims,
		Sources:         sources,
	}, nil
}

func BuildFrameworkComponents(topic string, themes []Theme, sources []Source, limit int) []FrameworkComponent {
	if limit <= 0 {
		limit = 4
	}

	components := make([]FrameworkComponent, 0, limit)
	for _, theme := range themes {
		if len(components) >= limit {
			break
		}
		name := toTitleWord(theme.Label) + " pillar"
		principle := fmt.Sprintf("Use %s as a repeated organizing principle across the archive (%d supporting chunks).", theme.Label, theme.Count)
		support := supportingSourceIndexes(theme.Label, sources, 3)
		components = append(components, FrameworkComponent{
			Name:              name,
			Principle:         principle,
			SupportingSources: support,
		})
	}

	if len(components) == 0 {
		fallback := FrameworkComponent{
			Name:              "Core grounding pillar",
			Principle:         fmt.Sprintf("Ground the framework for %s in the highest-confidence source evidence.", topic),
			SupportingSources: []int{},
		}
		if len(sources) > 0 {
			fallback.SupportingSources = []int{sources[0].Index}
		}
		components = append(components, fallback)
	}

	return components
}

func supportingSourceIndexes(themeLabel string, sources []Source, limit int) []int {
	if limit <= 0 {
		limit = 3
	}
	labelTokens := tokenizeThemeText(themeLabel)
	result := make([]int, 0, limit)

	for _, src := range sources {
		if len(result) >= limit {
			break
		}
		text := strings.ToLower(src.Title + " " + src.Text)
		for _, token := range labelTokens {
			if strings.Contains(text, token) {
				result = append(result, src.Index)
				break
			}
		}
	}
	return result
}

func BuildFrameworkSummary(topic string, components []FrameworkComponent, contradictions []Contradiction, influence []SourceInfluence) string {
	lines := []string{fmt.Sprintf("Framework draft for %s:", topic)}
	for i, component := range components {
		lines = append(lines, fmt.Sprintf("%d) %s — %s", i+1, component.Name, component.Principle))
	}
	if len(contradictions) > 0 {
		first := contradictions[0]
		lines = append(lines, fmt.Sprintf("Primary tension to resolve: %s (+%d/-%d).", first.Topic, first.Supporting, first.Opposing))
	}
	if len(influence) > 0 {
		lines = append(lines, fmt.Sprintf("Most influential source: [%d] %s (%.2f).", influence[0].Index, influence[0].Title, influence[0].Influence))
	}
	return strings.Join(lines, "\n")
}

func toTitleWord(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "Core"
	}
	return strings.ToUpper(trimmed[:1]) + trimmed[1:]
}
