// internal/config/config.go
package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string

	OpenRouterAPIKey   string
	OpenRouterModel    string
	OpenRouterSiteURL  string
	OpenRouterSiteName string

	QdrantURL        string
	QdrantAPIKey     string
	QdrantCollection string
	QdrantVectorName string
	QdrantTopK       int

	EmbedProvider          string
	OllamaURL              string
	OllamaEmbedModel       string
	OpenRouterEmbedBaseURL string
	OpenRouterEmbedModel   string

	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	RedisTTLSeconds  int
	ChatHistoryTurns int
	CacheEmbeddings  bool
	CacheQdrant      bool
	Strictness       string
}

func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		AppPort: getEnv("APP_PORT", "8090"),

		OpenRouterAPIKey:   getEnv("OPENROUTER_API_KEY", ""),
		OpenRouterModel:    getEnv("OPENROUTER_MODEL", "deepseek/deepseek-r1"),
		OpenRouterSiteURL:  getEnv("OPENROUTER_SITE_URL", "http://localhost:8090"),
		OpenRouterSiteName: getEnv("OPENROUTER_SITE_NAME", "ArchiMind"),

		QdrantURL:        strings.TrimRight(getEnv("QDRANT_URL", "http://localhost:6333"), "/"),
		QdrantAPIKey:     getEnv("QDRANT_API_KEY", ""),
		QdrantCollection: getEnv("QDRANT_COLLECTION", ""),
		QdrantVectorName: getEnv("QDRANT_VECTOR_NAME", ""),
		QdrantTopK:       getEnvInt("QDRANT_TOP_K", 8),

		EmbedProvider:          strings.ToLower(getEnv("EMBED_PROVIDER", "ollama")),
		OllamaURL:              strings.TrimRight(getEnv("OLLAMA_URL", "http://localhost:11434"), "/"),
		OllamaEmbedModel:       getEnv("OLLAMA_EMBED_MODEL", "nomic-embed-text:latest"),
		OpenRouterEmbedBaseURL: strings.TrimRight(getEnv("OPENROUTER_EMBED_BASE_URL", "https://openrouter.ai/api/v1"), "/"),
		OpenRouterEmbedModel:   getEnv("OPENROUTER_EMBED_MODEL", "openai/text-embedding-3-small"),

		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:          getEnvInt("REDIS_DB", 0),
		RedisTTLSeconds:  getEnvInt("REDIS_TTL_SECONDS", 3600),
		ChatHistoryTurns: getEnvInt("CHAT_HISTORY_TURNS", 12),
		CacheEmbeddings:  getEnvBool("CACHE_EMBEDDINGS", true),
		CacheQdrant:      getEnvBool("CACHE_QDRANT_RESULTS", true),
		Strictness:       strings.ToLower(strings.TrimSpace(getEnv("ARCHIMIND_STRICTNESS", "balanced"))),
	}

	if cfg.OpenRouterAPIKey == "" {
		log.Println("WARNING: OPENROUTER_API_KEY is empty")
	}

	if cfg.QdrantCollection == "" {
		log.Println("WARNING: QDRANT_COLLECTION is empty")
	}

	if cfg.QdrantVectorName == "" {
		log.Println("WARNING: QDRANT_VECTOR_NAME is empty. Named-vector collections require it.")
	}

	if cfg.Strictness != "strict" && cfg.Strictness != "balanced" && cfg.Strictness != "exploratory" {
		log.Printf("WARNING: ARCHIMIND_STRICTNESS=%q is invalid; using balanced", cfg.Strictness)
		cfg.Strictness = "balanced"
	}

	return cfg
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}

	return value == "true" || value == "1" || value == "yes" || value == "y"
}
