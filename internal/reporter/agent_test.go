package reporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"archimind/internal/config"
	"archimind/internal/embed"
	"archimind/internal/llm"
	"archimind/internal/logging"
	"archimind/internal/qdrant"
)

func TestAgentGenerate(t *testing.T) {
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		t.Skip("OPENROUTER_API_KEY is empty")
	}

	cfg := config.Load()
	if cfg.QdrantCollection == "" || cfg.QdrantURL == "" {
		t.Skip("Qdrant is not configured")
	}

	logger := logging.New()
	qdrClient := qdrant.NewClient(cfg)
	embedder := embed.NewOpenRouterProvider(cfg)
	chatProvider := llm.NewOpenRouterProvider(cfg)
	agent := NewAgent(cfg, qdrClient, chatProvider, embedder, logger)

	output := filepath.Join(t.TempDir(), "report.md")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err := agent.Generate(ctx, ReportRequest{
		Topic:      "Brief retrieval summary test",
		TokenLimit: 1200,
		OutputPath: output,
	})
	if err != nil {
		if strings.Contains(err.Error(), "no relevant chunks found") {
			t.Skipf("report skipped: %v", err)
		}
		t.Fatalf("generate report: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output report: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("report output is empty")
	}
}
