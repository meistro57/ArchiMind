package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"archimind/internal/memory"
)

func (e *Engine) SessionHistory(ctx context.Context, sessionID string) ([]memory.Turn, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = "default"
	}
	return e.mem.GetHistory(ctx, sessionID)
}

func (e *Engine) ExportSessionMarkdown(ctx context.Context, sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = "default"
	}

	history, err := e.SessionHistory(ctx, sessionID)
	if err != nil {
		return "", err
	}

	return formatHistoryMarkdown(sessionID, history, time.Now().UTC()), nil
}

func formatHistoryMarkdown(sessionID string, history []memory.Turn, exportedAt time.Time) string {
	lines := []string{
		"# ArchiMind Chat Export",
		"",
		fmt.Sprintf("- Session: `%s`", sessionID),
		fmt.Sprintf("- Exported at: %s", exportedAt.Format(time.RFC3339)),
		"",
	}

	if len(history) == 0 {
		lines = append(lines, "_No chat history found for this session._")
		return strings.Join(lines, "\n")
	}

	for i, turn := range history {
		role := strings.TrimSpace(turn.Role)
		if role == "" {
			role = "Unknown"
		} else {
			role = strings.ToUpper(role[:1]) + role[1:]
		}
		content := strings.TrimSpace(turn.Content)
		if content == "" {
			content = "(empty)"
		}
		lines = append(lines,
			fmt.Sprintf("## %d. %s", i+1, role),
			"",
			content,
			"",
		)
	}

	return strings.Join(lines, "\n")
}
