package reporter

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"archimind/internal/config"
	"archimind/internal/embed"
	"archimind/internal/llm"
	"archimind/internal/qdrant"
)

const defaultReportTokenLimit = 6000
const defaultQueryLimit = 100

type Agent struct {
	cfg      config.Config
	qdr      *qdrant.Client
	chat     llm.Provider
	embedder embed.Provider
	logger   *log.Logger
}

type ReportRequest struct {
	Topic      string
	SourceID   string
	TokenLimit int
	OutputPath string
}

func NewAgent(cfg config.Config, qdr *qdrant.Client, chat llm.Provider, embedder embed.Provider, logger *log.Logger) *Agent {
	return &Agent{
		cfg:      cfg,
		qdr:      qdr,
		chat:     chat,
		embedder: embedder,
		logger:   logger,
	}
}

func (a *Agent) Generate(ctx context.Context, req ReportRequest) error {
	req.Topic = strings.TrimSpace(req.Topic)
	req.SourceID = strings.TrimSpace(req.SourceID)
	req.OutputPath = strings.TrimSpace(req.OutputPath)
	if req.Topic == "" {
		return fmt.Errorf("topic is required")
	}
	if req.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}
	if req.TokenLimit <= 0 {
		req.TokenLimit = defaultReportTokenLimit
	}

	a.logger.Printf("report generation started topic=%q source_id=%q token_limit=%d output=%s", req.Topic, req.SourceID, req.TokenLimit, req.OutputPath)

	vector, err := a.embedder.Embed(ctx, req.Topic)
	if err != nil {
		return fmt.Errorf("embed topic: %w", err)
	}

	points, err := a.qdr.Query(ctx, a.cfg.QdrantCollection, vector, defaultQueryLimit)
	if err != nil {
		return fmt.Errorf("query qdrant: %w", err)
	}

	var contextBuilder strings.Builder
	totalTokens := 0
	chunksUsed := 0

	for _, point := range points {
		if req.SourceID != "" && strings.TrimSpace(valueString(point.Payload["source_id"])) != req.SourceID {
			continue
		}

		tokenEstimate := estimateTokens(point.Payload["token_est"])
		if totalTokens+tokenEstimate > req.TokenLimit {
			break
		}

		chapter := strings.TrimSpace(valueString(point.Payload["chapter"]))
		if chapter == "" {
			chapter = "Unknown Chapter"
		}
		text := strings.TrimSpace(valueString(point.Payload["text"]))
		if text == "" {
			continue
		}

		contextBuilder.WriteString("## Chapter: ")
		contextBuilder.WriteString(chapter)
		contextBuilder.WriteString("\n")
		contextBuilder.WriteString(text)
		contextBuilder.WriteString("\n\n")

		totalTokens += tokenEstimate
		chunksUsed++
	}

	if chunksUsed == 0 {
		return fmt.Errorf("no relevant chunks found for topic %q", req.Topic)
	}

	a.logger.Printf("report context assembled chunks=%d estimated_tokens=%d", chunksUsed, totalTokens)

	systemPrompt := strings.TrimSpace(`You are an archival researcher.
Write a comprehensive Markdown report strictly grounded in the provided archive context.
Use clear section headings and bullet points where useful.
Do not fabricate facts or citations beyond the supplied context.
When the context is insufficient for a claim, state that explicitly.`)
	userPrompt := fmt.Sprintf("Topic:\n%s\n\nArchive context:\n%s\nWrite the full report in Markdown.", req.Topic, strings.TrimSpace(contextBuilder.String()))

	reportBody, err := a.chat.Chat(ctx, []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return fmt.Errorf("generate report with llm: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	header := fmt.Sprintf("# Report: %s\n\nGenerated: %s\n\n", req.Topic, time.Now().Format(time.RFC3339))
	if err := os.WriteFile(req.OutputPath, []byte(header+strings.TrimSpace(reportBody)+"\n"), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	a.logger.Printf("report generation completed output=%s chunks=%d estimated_tokens=%d", req.OutputPath, chunksUsed, totalTokens)
	return nil
}

func estimateTokens(value any) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return 300
}

func valueString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}
