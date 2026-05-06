package rag

import (
	"fmt"
	"sort"
	"strings"
)

type StrongClaim struct {
	Text          string   `json:"text"`
	Confidence    float64  `json:"confidence"`
	SourceIndexes []int    `json:"source_indexes"`
	Themes        []string `json:"themes,omitempty"`
}

func RankStrongClaims(sources []Source, themes []Theme, contradictions []Contradiction, limit int) []StrongClaim {
	if limit <= 0 {
		limit = 5
	}

	themeTokens := map[string]struct{}{}
	for _, theme := range themes {
		themeTokens[strings.ToLower(strings.TrimSpace(theme.Label))] = struct{}{}
	}
	contradictionTokens := map[string]struct{}{}
	for _, contradiction := range contradictions {
		contradictionTokens[strings.ToLower(strings.TrimSpace(contradiction.Topic))] = struct{}{}
	}

	claims := make([]StrongClaim, 0, len(sources))
	for _, src := range sources {
		claimText := strongestSentence(src.Text)
		if claimText == "" {
			claimText = strings.TrimSpace(src.Text)
		}
		if claimText == "" {
			continue
		}

		tokens := tokenizeThemeText(src.Title + " " + claimText)
		claimThemes := make([]string, 0, 3)
		themeScore := 0.0
		for _, token := range tokens {
			if _, ok := themeTokens[token]; ok {
				themeScore += 0.05
				if len(claimThemes) < 3 {
					claimThemes = append(claimThemes, token)
				}
			}
			if _, blocked := contradictionTokens[token]; blocked {
				themeScore -= 0.03
			}
		}

		confidence := src.Score + themeScore
		if confidence < 0 {
			confidence = 0
		}
		if confidence > 1.5 {
			confidence = 1.5
		}

		claims = append(claims, StrongClaim{
			Text:          truncateClaim(claimText, 220),
			Confidence:    confidence,
			SourceIndexes: []int{src.Index},
			Themes:        dedupeStrings(claimThemes),
		})
	}

	sort.Slice(claims, func(i, j int) bool {
		if claims[i].Confidence == claims[j].Confidence {
			return claims[i].Text < claims[j].Text
		}
		return claims[i].Confidence > claims[j].Confidence
	})

	if len(claims) > limit {
		claims = claims[:limit]
	}

	return claims
}

func strongestSentence(text string) string {
	sentences := splitSentences(text)
	best := ""
	bestScore := -1
	for _, sentence := range sentences {
		lower := strings.ToLower(strings.TrimSpace(sentence))
		if lower == "" {
			continue
		}
		score := 0
		if containsAny(lower, assertiveMarkers) {
			score += 2
		}
		if citationPattern.MatchString(sentence) {
			score += 1
		}
		if len(strings.Fields(sentence)) >= 8 {
			score += 1
		}
		if score > bestScore {
			bestScore = score
			best = strings.TrimSpace(sentence)
		}
	}
	return best
}

func truncateClaim(text string, max int) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= max {
		return trimmed
	}
	if max < 4 {
		return trimmed[:max]
	}
	return fmt.Sprintf("%s...", strings.TrimSpace(trimmed[:max-3]))
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
