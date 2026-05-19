// internal/rag/rag_test.go
package rag

import (
	"strings"
	"testing"
)

func TestInferAnswerMode(t *testing.T) {
	tests := []struct {
		question string
		expected AnswerMode
	}{
		{question: "Brainstorm a slogan for our app", expected: AnswerModeCreative},
		{question: "What should I do to improve onboarding?", expected: AnswerModeAdvisory},
		{question: "What is Redis used for?", expected: AnswerModeKnowledge},
	}

	for _, tc := range tests {
		got := InferAnswerMode(tc.question)
		if got != tc.expected {
			t.Fatalf("question %q expected %q, got %q", tc.question, tc.expected, got)
		}
	}
}

func TestBuildSignalMetrics(t *testing.T) {
	sources := []Source{
		{Index: 1, Score: 0.9, Title: "A", Text: "First"},
		{Index: 2, Score: 0.7, Title: "B", Text: "Second"},
	}
	signal := BuildSignal("Compare options", sources)

	if signal.TopScore != 0.9 {
		t.Fatalf("expected top score 0.9, got %f", signal.TopScore)
	}
	if diff := signal.ScoreGap - 0.2; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("expected score gap 0.2, got %f", signal.ScoreGap)
	}
	if diff := signal.SimilaritySpread - 0.2; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("expected spread 0.2, got %f", signal.SimilaritySpread)
	}
	if !signal.HighRiskSynthesis {
		t.Fatalf("expected high-risk synthesis to be true")
	}
}

func TestBuildSystemPromptIncludesCoreRules(t *testing.T) {
	signal := RetrievalSignal{Mode: AnswerModeAdvisory, Strictness: "strict", Cluster: "faq"}
	prompt := buildSystemPrompt(signal)

	required := []string{
		"Use only the supplied Qdrant context for factual claims.",
		"Cite retrieved context with bracket citations like [1], [2].",
		"Strictness profile: strict.",
		"Answer mode: advisory.",
		"Cluster profile: faq.",
	}

	for _, fragment := range required {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf("prompt missing required fragment: %q", fragment)
		}
	}
}

func TestClassifyCluster(t *testing.T) {
	faqSources := []Source{{
		Title:  "FAQ: Setup",
		Text:   "Q: How do I run this? A: Use go run main.go",
		Page:   "1",
		Chunk:  "2",
		Source: "manual",
	}}
	if got := classifyCluster(faqSources); got != "faq" {
		t.Fatalf("expected faq cluster, got %q", got)
	}
}
