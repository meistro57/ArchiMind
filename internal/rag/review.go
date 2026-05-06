package rag

import (
	"context"
	"fmt"
	"strings"

	"archimind/internal/memory"
)

type SelfAuditReport struct {
	SessionID         string            `json:"session_id"`
	LastUserMessage   string            `json:"last_user_message,omitempty"`
	LastAssistantText string            `json:"last_assistant_text,omitempty"`
	Diagnostics       AnswerDiagnostics `json:"diagnostics"`
}

func (e *Engine) ReviewLastAnswer(ctx context.Context, sessionID string) (SelfAuditReport, error) {
	history, err := e.SessionHistory(ctx, sessionID)
	if err != nil {
		return SelfAuditReport{}, err
	}
	return buildSelfAuditFromHistory(sessionID, history)
}

func buildSelfAuditFromHistory(sessionID string, history []memory.Turn) (SelfAuditReport, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = "default"
	}

	var lastUser string
	var lastAssistant string
	for i := len(history) - 1; i >= 0; i-- {
		turn := history[i]
		role := strings.ToLower(strings.TrimSpace(turn.Role))
		if role == "assistant" && lastAssistant == "" {
			lastAssistant = strings.TrimSpace(turn.Content)
			continue
		}
		if role == "user" && lastUser == "" {
			lastUser = strings.TrimSpace(turn.Content)
		}
		if lastUser != "" && lastAssistant != "" {
			break
		}
	}

	if lastAssistant == "" {
		return SelfAuditReport{}, fmt.Errorf("no assistant answer available for self-audit")
	}

	signal := BuildSignal(lastUser, nil, "")
	diagnostics := AnalyzeAnswerDiscipline(lastAssistant, nil, signal)

	return SelfAuditReport{
		SessionID:         sessionID,
		LastUserMessage:   lastUser,
		LastAssistantText: lastAssistant,
		Diagnostics:       diagnostics,
	}, nil
}
