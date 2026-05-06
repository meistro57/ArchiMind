package rag

import (
	"strings"
	"testing"
	"time"

	"archimind/internal/memory"
)

func TestInferAnswerMode(t *testing.T) {
	tests := []struct {
		name     string
		question string
		want     AnswerMode
	}{
		{name: "diagnostic hints", question: "debug this retrieval failure", want: AnswerModeDiagnostic},
		{name: "skeptical hints", question: "give a skeptical critique", want: AnswerModeSkeptical},
		{name: "synthesis hints", question: "synthesize themes across sources", want: AnswerModeSynthesis},
		{name: "default normal", question: "what does this source claim", want: AnswerModeNormal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InferAnswerMode(tt.question); got != tt.want {
				t.Fatalf("InferAnswerMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildSignalHonorsRequestedMode(t *testing.T) {
	signal := BuildSignal("what does this say", nil, "skeptical")
	if signal.Mode != AnswerModeSkeptical {
		t.Fatalf("BuildSignal() mode = %q, want %q", signal.Mode, AnswerModeSkeptical)
	}
}

func TestBuildSignalIgnoresInvalidRequestedMode(t *testing.T) {
	signal := BuildSignal("debug this", nil, "invalid")
	if signal.Mode != AnswerModeDiagnostic {
		t.Fatalf("BuildSignal() mode = %q, want %q", signal.Mode, AnswerModeDiagnostic)
	}
}

func TestExtractRecurringThemes(t *testing.T) {
	sources := []Source{
		{Title: "Retrieval discipline", Text: "Retrieval quality improves with better retrieval diagnostics and source boundaries."},
		{Title: "Diagnostics", Text: "Clear diagnostics reduce retrieval confusion and improve retrieval integrity."},
		{Title: "Reasoning", Text: "Reasoning discipline depends on retrieval evidence and diagnostics."},
	}

	themes := ExtractRecurringThemes(sources, 3)
	if len(themes) == 0 {
		t.Fatal("ExtractRecurringThemes() returned no themes")
	}
	if themes[0].Count < 2 {
		t.Fatalf("top theme count = %d, want >= 2", themes[0].Count)
	}
}

func TestExtractContradictions(t *testing.T) {
	sources := []Source{
		{Index: 1, Title: "Policy", Text: "Teams should centralize release governance for quality."},
		{Index: 2, Title: "Policy", Text: "Teams should not centralize release governance because it slows delivery."},
		{Index: 3, Title: "Notes", Text: "A distributed process can work if tooling is consistent."},
	}

	contradictions := ExtractContradictions(sources, 3)
	if len(contradictions) == 0 {
		t.Fatal("ExtractContradictions() returned no contradictions")
	}
	if contradictions[0].Supporting == 0 || contradictions[0].Opposing == 0 {
		t.Fatalf("expected both sides, got +%d/-%d", contradictions[0].Supporting, contradictions[0].Opposing)
	}
}

func TestBuildCompareUserPrompt(t *testing.T) {
	left := []Source{{Index: 1, Title: "Left source", Score: 0.91, Text: "Left context text"}}
	right := []Source{{Index: 1, Title: "Right source", Score: 0.88, Text: "Right context text"}}

	prompt := buildCompareUserPrompt("What differs?", "left_col", "right_col", left, right)
	if prompt == "" {
		t.Fatal("buildCompareUserPrompt() returned empty prompt")
	}
	if !containsAll(prompt, "[L1]", "[R1]", "left_col", "right_col") {
		t.Fatalf("buildCompareUserPrompt() missing expected markers: %s", prompt)
	}
}

func TestFormatHistoryMarkdown(t *testing.T) {
	history := []memory.Turn{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "world"},
	}
	output := formatHistoryMarkdown("session-a", history, time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC))
	if !containsAll(output, "# ArchiMind Chat Export", "Session: `session-a`", "## 1. User", "## 2. Assistant") {
		t.Fatalf("formatHistoryMarkdown() missing expected content: %s", output)
	}
}

func TestAnalyzeAnswerDiscipline(t *testing.T) {
	sources := []Source{{Index: 1, Text: "Evidence text"}}
	signal := RetrievalSignal{HighRiskSynthesis: true, TopScore: 0.35}
	answer := "This always works in every case. It might fail in rare scenarios [1]."

	diagnostics := AnalyzeAnswerDiscipline(answer, sources, signal)
	if diagnostics.UnsupportedClaims == 0 {
		t.Fatal("AnalyzeAnswerDiscipline() expected unsupported claims")
	}
	if diagnostics.UnsupportedLeapRisk == "low" {
		t.Fatalf("AnalyzeAnswerDiscipline() risk = %q, want medium/high", diagnostics.UnsupportedLeapRisk)
	}
}

func TestRankSourceInfluence(t *testing.T) {
	sources := []Source{
		{Index: 1, Title: "A", Score: 0.70, Text: "retrieval quality and diagnostics"},
		{Index: 2, Title: "B", Score: 0.66, Text: "retrieval quality and contradiction diagnostics"},
	}
	themes := []Theme{{Label: "retrieval", Count: 2}, {Label: "diagnostics", Count: 2}}
	contradictions := []Contradiction{{Topic: "contradiction", Supporting: 1, Opposing: 1, MentionedIn: 2}}

	ranked := RankSourceInfluence(sources, themes, contradictions, 2)
	if len(ranked) != 2 {
		t.Fatalf("RankSourceInfluence() len = %d, want 2", len(ranked))
	}
	if ranked[0].Influence < ranked[1].Influence {
		t.Fatalf("expected descending influence, got %.2f < %.2f", ranked[0].Influence, ranked[1].Influence)
	}
}

func TestRankStrongClaims(t *testing.T) {
	sources := []Source{
		{Index: 1, Title: "A", Score: 0.75, Text: "Retrieval discipline improves answer quality in production systems."},
		{Index: 2, Title: "B", Score: 0.62, Text: "Diagnostics reduce uncertainty and improve retrieval reliability."},
	}
	themes := []Theme{{Label: "retrieval", Count: 2}, {Label: "diagnostics", Count: 2}}
	contradictions := []Contradiction{{Topic: "uncertainty", Supporting: 1, Opposing: 1, MentionedIn: 2}}

	claims := RankStrongClaims(sources, themes, contradictions, 2)
	if len(claims) != 2 {
		t.Fatalf("RankStrongClaims() len = %d, want 2", len(claims))
	}
	if claims[0].Confidence < claims[1].Confidence {
		t.Fatalf("expected descending confidence, got %.2f < %.2f", claims[0].Confidence, claims[1].Confidence)
	}
	if len(claims[0].SourceIndexes) == 0 {
		t.Fatal("expected source indexes in strong claim")
	}
}

func TestRankStrongClaimsSelectsAssertiveSentence(t *testing.T) {
	sources := []Source{
		{Index: 1, Title: "C", Score: 0.70, Text: "Background details only. This demonstrates measurable gains in reliability across teams."},
	}

	claims := RankStrongClaims(sources, nil, nil, 1)
	if len(claims) != 1 {
		t.Fatalf("RankStrongClaims() len = %d, want 1", len(claims))
	}
	if !strings.Contains(claims[0].Text, "demonstrates measurable gains") {
		t.Fatalf("expected strongest sentence to be selected, got %q", claims[0].Text)
	}
}

func TestBuildSelfAuditFromHistory(t *testing.T) {
	history := []memory.Turn{
		{Role: "user", Content: "What should I do?"},
		{Role: "assistant", Content: "You must always do X."},
	}
	report, err := buildSelfAuditFromHistory("session-1", history)
	if err != nil {
		t.Fatalf("buildSelfAuditFromHistory() error = %v", err)
	}
	if report.SessionID != "session-1" {
		t.Fatalf("session id = %q, want session-1", report.SessionID)
	}
	if report.Diagnostics.UnsupportedClaims == 0 {
		t.Fatal("expected unsupported claims in self-audit")
	}
}

func TestBuildFrameworkComponents(t *testing.T) {
	themes := []Theme{{Label: "retrieval", Count: 3}, {Label: "diagnostics", Count: 2}}
	sources := []Source{
		{Index: 1, Title: "retrieval notes", Text: "retrieval diagnostics and quality"},
		{Index: 2, Title: "diagnostics guide", Text: "diagnostics checklist"},
	}
	components := BuildFrameworkComponents("topic", themes, sources, 3)
	if len(components) < 2 {
		t.Fatalf("BuildFrameworkComponents() len = %d, want >= 2", len(components))
	}
	if len(components[0].SupportingSources) == 0 {
		t.Fatal("expected supporting source indexes")
	}
}

func TestBuildFrameworkSummary(t *testing.T) {
	components := []FrameworkComponent{{Name: "Retrieval pillar", Principle: "Use retrieval discipline."}}
	contradictions := []Contradiction{{Topic: "precision", Supporting: 1, Opposing: 1, MentionedIn: 2}}
	influence := []SourceInfluence{{Index: 1, Title: "Source A", Influence: 1.2}}

	summary := BuildFrameworkSummary("my topic", components, contradictions, influence)
	if !containsAll(summary, "Framework draft for my topic", "Primary tension to resolve", "Most influential source") {
		t.Fatalf("BuildFrameworkSummary() missing expected parts: %s", summary)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
