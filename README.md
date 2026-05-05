# ArchiMind

ArchiMind is a Go web chatbot that answers questions against a Qdrant collection using retrieval-augmented generation (RAG).

## What it uses

- **Chat model:** OpenRouter (`internal/llm/openrouter.go`)
- **Embeddings:** Ollama or OpenRouter (`internal/embed/`)
- **Vector retrieval:** Qdrant (`internal/qdrant/`)
- **Memory/cache:** Redis (`internal/memory/redis.go`)
- **UI:** Static browser app in `web/`
- **Background reports:** Reporter agent (`internal/reporter/agent.go`) using Qdrant + OpenRouter

## Quick start

## 1) Configure environment

Set these variables (for local development they can live in `.env`, loaded automatically by `godotenv` in `internal/config/config.go`):

### Core

- `APP_PORT` (default `8090`)
- `OPENROUTER_API_KEY` (required for chat and OpenRouter embeddings)
- `OPENROUTER_MODEL` (default `deepseek/deepseek-r1`)
- `OPENROUTER_SITE_URL` (default `http://localhost:8090`)
- `OPENROUTER_SITE_NAME` (default `ArchiMind`)

### Qdrant

- `QDRANT_URL` (default `http://localhost:6333`)
- `QDRANT_API_KEY` (optional)
- `QDRANT_COLLECTION` (required for normal usage)
- `QDRANT_VECTOR_NAME` (required for named-vector collections)
- `QDRANT_TOP_K` (default `8`)

### Embeddings

- `EMBED_PROVIDER` (`ollama` default, or `openrouter`)
- `OLLAMA_URL` (default `http://localhost:11434`)
- `OLLAMA_EMBED_MODEL` (default `nomic-embed-text:latest`)
- `OPENROUTER_EMBED_BASE_URL` (default `https://openrouter.ai/api/v1`)
- `OPENROUTER_EMBED_MODEL` (default `openai/text-embedding-3-small`)

### Redis / memory

- `REDIS_ADDR` (default `localhost:6379`)
- `REDIS_PASSWORD` (optional)
- `REDIS_DB` (default `0`)
- `REDIS_TTL_SECONDS` (default `3600`, used for JSON cache entries)
- `CHAT_HISTORY_TURNS` (default `12`)
- `CACHE_EMBEDDINGS` (default `true`)
- `CACHE_QDRANT_RESULTS` (default `true`)

### Prompt strictness

- `ARCHIMIND_STRICTNESS` (default `balanced`)
- Accepted values currently in code: `strict`, `balanced`, `exploratory`

## 2) Run

```bash
go run main.go
```

Then open: `http://localhost:8090`

## Development commands

```bash
# Format
gofmt -w .

# Dependency cleanup
go mod tidy

# Compile all packages
go build ./...

# Run tests
go test ./...
```

## HTTP API

### `POST /api/chat`

Request:

```json
{
  "session_id": "optional-session-id",
  "message": "your question",
  "collection": "optional-collection-override"
}
```

Response:

```json
{
  "answer": "assistant response",
  "sources": [
    {
      "index": 1,
      "score": 0.95,
      "title": "source title",
      "source": "optional source",
      "text": "retrieved text"
    }
  ]
}
```

### `POST /api/report`

Starts asynchronous report generation.

Request:

```json
{
  "topic": "history of retrieval architecture"
}
```

Response:

```json
{
  "message": "report generation started",
  "output_path": "reports/history_of_retrieval_architecture_20260505_120000.md"
}
```

### `GET /api/health`

Returns service status.

### `GET /api/collection?name=<collection>`

Returns raw Qdrant collection info (or configured default collection if `name` omitted).

## Project layout

- `main.go` - wiring for config, providers, Redis, Qdrant, RAG engine, HTTP server
- `internal/config/` - environment loading and validation
- `internal/server/` - HTTP handlers + static web serving
- `internal/rag/` - retrieval + prompt assembly + source extraction
- `internal/qdrant/` - Qdrant API client (collection info, vector size, query)
- `internal/embed/` - embedding provider implementations
- `internal/llm/` - chat provider interface + OpenRouter chat implementation
- `internal/memory/` - Redis chat history and cache storage
- `internal/reporter/` - background report generation agent
- `web/` - browser client

## Notes and gotchas

- Chat history is stored per session key: `chat:<session_id>:history`.
- `CHAT_HISTORY_TURNS` trims retained turns.
- History key expiration is currently fixed to **24h** in `SaveTurn`; cache TTL uses `REDIS_TTL_SECONDS`.
- Embedding and Qdrant result caching include provider/model/vector names in cache keys to avoid collisions across provider/model changes.
- On startup, the app attempts to fetch Qdrant vector size for configured `QDRANT_COLLECTION` + `QDRANT_VECTOR_NAME` and logs dimension mismatch warnings later in RAG execution.
- If no retrieval hits are returned, the assistant responds with a clear "could not find anything relevant" message.
- `/api/report` runs in a goroutine and writes markdown reports to `reports/<topic>_<timestamp>.md`.
