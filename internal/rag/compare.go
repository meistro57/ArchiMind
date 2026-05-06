package rag

import (
	"context"
	"fmt"
	"strings"

	"archimind/internal/llm"
)

type CollectionInsights struct {
	Collection      string            `json:"collection"`
	Sources         []Source          `json:"sources"`
	Themes          []Theme           `json:"themes,omitempty"`
	Contradictions  []Contradiction   `json:"contradictions,omitempty"`
	SourceInfluence []SourceInfluence `json:"source_influence,omitempty"`
	StrongClaims    []StrongClaim     `json:"strong_claims,omitempty"`
}

type CompareResult struct {
	Answer string             `json:"answer"`
	Left   CollectionInsights `json:"left"`
	Right  CollectionInsights `json:"right"`
}

func (e *Engine) CompareCollections(ctx context.Context, sessionID string, leftCollection string, rightCollection string, question string, requestedMode string) (CompareResult, error) {
	leftCollection = strings.TrimSpace(leftCollection)
	rightCollection = strings.TrimSpace(rightCollection)
	question = strings.TrimSpace(question)

	if leftCollection == "" || rightCollection == "" {
		return CompareResult{}, fmt.Errorf("both collections are required for comparison")
	}
	if question == "" {
		return CompareResult{}, fmt.Errorf("comparison question is required")
	}

	_ = e.mem.SaveTurn(ctx, sessionID, "user", fmt.Sprintf("[compare] %s | left=%s right=%s", question, leftCollection, rightCollection), e.cfg.ChatHistoryTurns)

	vector, err := e.getEmbedding(ctx, question)
	if err != nil {
		return CompareResult{}, err
	}

	leftPoints, err := e.getQdrantResults(ctx, leftCollection, question, vector)
	if err != nil {
		return CompareResult{}, err
	}
	rightPoints, err := e.getQdrantResults(ctx, rightCollection, question, vector)
	if err != nil {
		return CompareResult{}, err
	}

	leftSources := pointsToSources(leftPoints)
	rightSources := pointsToSources(rightPoints)

	if len(leftSources) == 0 && len(rightSources) == 0 {
		answer := "I could not find relevant material in either collection for that comparison."
		_ = e.mem.SaveTurn(ctx, sessionID, "assistant", answer, e.cfg.ChatHistoryTurns)
		return CompareResult{Answer: answer, Left: CollectionInsights{Collection: leftCollection}, Right: CollectionInsights{Collection: rightCollection}}, nil
	}

	leftThemes := ExtractRecurringThemes(leftSources, 5)
	rightThemes := ExtractRecurringThemes(rightSources, 5)
	leftContradictions := ExtractContradictions(leftSources, 3)
	rightContradictions := ExtractContradictions(rightSources, 3)
	leftInfluence := RankSourceInfluence(leftSources, leftThemes, leftContradictions, 5)
	rightInfluence := RankSourceInfluence(rightSources, rightThemes, rightContradictions, 5)
	leftClaims := RankStrongClaims(leftSources, leftThemes, leftContradictions, 5)
	rightClaims := RankStrongClaims(rightSources, rightThemes, rightContradictions, 5)

	signal := BuildSignal(question, append(append([]Source{}, leftSources...), rightSources...), requestedMode)
	signal.Strictness = normalizeStrictness(e.cfg.Strictness)

	system := buildCompareSystemPrompt(signal, leftCollection, rightCollection, leftThemes, rightThemes, leftContradictions, rightContradictions)
	user := buildCompareUserPrompt(question, leftCollection, rightCollection, leftSources, rightSources)

	answer, err := e.chat.Chat(ctx, []llm.Message{
		{Role: "system", Content: strings.TrimSpace(system)},
		{Role: "user", Content: user},
	})
	if err != nil {
		return CompareResult{}, err
	}

	_ = e.mem.SaveTurn(ctx, sessionID, "assistant", answer, e.cfg.ChatHistoryTurns)

	return CompareResult{
		Answer: answer,
		Left: CollectionInsights{
			Collection:      leftCollection,
			Sources:         leftSources,
			Themes:          leftThemes,
			Contradictions:  leftContradictions,
			SourceInfluence: leftInfluence,
			StrongClaims:    leftClaims,
		},
		Right: CollectionInsights{
			Collection:      rightCollection,
			Sources:         rightSources,
			Themes:          rightThemes,
			Contradictions:  rightContradictions,
			SourceInfluence: rightInfluence,
			StrongClaims:    rightClaims,
		},
	}, nil
}

func buildCompareSystemPrompt(signal RetrievalSignal, leftCollection string, rightCollection string, leftThemes []Theme, rightThemes []Theme, leftContradictions []Contradiction, rightContradictions []Contradiction) string {
	lines := []string{
		"You are ArchiMind, a retrieval-grounded comparison assistant for Qdrant collections.",
		"Rules:",
		"- Compare only using supplied context chunks.",
		"- Keep boundaries explicit: state whether each claim belongs to left, right, both, or neither collection.",
		"- Cite supporting chunks with [L1], [L2] for left and [R1], [R2] for right.",
		"- If evidence is insufficient in either side, say so directly.",
		fmt.Sprintf("- Active mode: %s. Strictness: %s.", signal.Mode, signal.Strictness),
		fmt.Sprintf("- Comparing collections: left=%s, right=%s.", leftCollection, rightCollection),
	}

	leftThemeSummary := buildThemeSummary(leftThemes)
	if leftThemeSummary != "" {
		lines = append(lines, strings.Replace(leftThemeSummary, "retrieval", leftCollection, 1))
	}
	rightThemeSummary := buildThemeSummary(rightThemes)
	if rightThemeSummary != "" {
		lines = append(lines, strings.Replace(rightThemeSummary, "retrieval", rightCollection, 1))
	}

	leftContradictionSummary := buildContradictionSummary(leftContradictions)
	if leftContradictionSummary != "" {
		lines = append(lines, strings.Replace(leftContradictionSummary, "signals", leftCollection+" signals", 1))
	}
	rightContradictionSummary := buildContradictionSummary(rightContradictions)
	if rightContradictionSummary != "" {
		lines = append(lines, strings.Replace(rightContradictionSummary, "signals", rightCollection+" signals", 1))
	}

	return strings.Join(lines, "\n")
}

func buildCompareUserPrompt(question string, leftCollection string, rightCollection string, leftSources []Source, rightSources []Source) string {
	leftContext := buildTaggedContext("L", leftSources)
	rightContext := buildTaggedContext("R", rightSources)

	if leftContext == "" {
		leftContext = "No relevant sources returned."
	}
	if rightContext == "" {
		rightContext = "No relevant sources returned."
	}

	return fmt.Sprintf(
		"Comparison question:\n%s\n\nLeft collection (%s):\n%s\n\nRight collection (%s):\n%s\n\nRespond with:\n1) Agreements\n2) Disagreements\n3) Unique strengths per side\n4) Uncertainties and missing evidence",
		question,
		leftCollection,
		leftContext,
		rightCollection,
		rightContext,
	)
}

func buildTaggedContext(prefix string, sources []Source) string {
	if len(sources) == 0 {
		return ""
	}
	blocks := make([]string, 0, len(sources))
	for _, src := range sources {
		blocks = append(blocks, fmt.Sprintf("[%s%d] %s\nScore: %.4f\nText:\n%s", prefix, src.Index, src.Title, src.Score, src.Text))
	}
	return strings.Join(blocks, "\n\n---\n\n")
}
