// main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"archimind/internal/config"
	"archimind/internal/embed"
	"archimind/internal/llm"
	"archimind/internal/logging"
	"archimind/internal/memory"
	"archimind/internal/qdrant"
	"archimind/internal/rag"
	"archimind/internal/server"
)

func buildEmbedder(cfg config.Config) embed.Provider {
	switch cfg.EmbedProvider {
	case "openrouter":
		return embed.NewOpenRouterProvider(cfg)
	default: // "ollama" or unset
		return embed.NewOllamaProvider(cfg)
	}
}

func main() {
	logger := logging.New()

	cfg := config.Load()

	redisMemory := memory.NewRedisMemory(cfg)
	if err := redisMemory.Ping(context.Background()); err != nil {
		logger.Printf("Redis warning: %v", err)
	} else {
		logger.Println("Redis connected.")
	}

	embedder := buildEmbedder(cfg)
	chatProvider := llm.NewOpenRouterProvider(cfg)
	qdrantClient := qdrant.NewClient(cfg)

	logger.Printf("Embedding provider : %s", cfg.EmbedProvider)
	logger.Printf("Embedding model    : %s", embedder.ModelName())
	logger.Printf("Qdrant collection  : %s", cfg.QdrantCollection)
	logger.Printf("Qdrant vector name : %s", cfg.QdrantVectorName)
	logger.Printf("Prompt strictness  : %s", cfg.Strictness)

	ragEngine := rag.NewEngine(
		cfg,
		qdrantClient,
		chatProvider,
		embedder,
		redisMemory,
		logger,
	)

	appServer := server.New(cfg, ragEngine, qdrantClient, chatProvider, redisMemory, logger)

	go func() {
		if err := appServer.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	logger.Printf("ArchiMind running at http://localhost:%s", cfg.AppPort)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Println("Shutting down ArchiMind...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := appServer.Shutdown(ctx); err != nil {
		logger.Printf("shutdown error: %v", err)
	}

	ragEngine.Close()

	if err := redisMemory.Close(); err != nil {
		logger.Printf("redis close error: %v", err)
	}

	logger.Println("Shutdown complete.")
}
