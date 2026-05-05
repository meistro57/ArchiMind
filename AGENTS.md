# AGENTS.md

This file is a practical guide for coding agents working in the ArchiMind repository.

## Repository snapshot

- Language: **Go** (`go.mod` uses Go `1.22`)
- Runtime: HTTP server with static web UI
- Storage/services:
  - Qdrant for retrieval
  - Redis for history/caching
  - OpenRouter for chat
  - Ollama or OpenRouter for embeddings
- Entry point: `main.go`

## First-run checklist for agents

1. Read `internal/config/config.go` for current env keys and defaults.
2. Read `main.go` for wiring decisions and startup behavior.
3. Read `internal/rag/rag.go` before touching answer behavior.
4. Validate with:

```bash
go test ./...
go build ./...
```

5. Format Go code after edits:

```bash
gofmt -w .
```

## Observed commands

No Makefile/Taskfile/CI config was found in this repo. Use native Go commands.

```bash
# Run app
go run main.go

# Format
gofmt -w .

# Dependency cleanup
go mod tidy

# Test all packages
go test ./...

# Build all packages
go build ./...
```

## Architecture and request flow

1. `internal/server/server.go` receives `POST /api/chat`.
2. Server calls `rag.Engine.Ask(...)`.
3. `Ask` flow in `internal/rag/rag.go`:
   - Save user turn to Redis
   - Generate embedding (`internal/embed` provider)
   - Query Qdrant (`internal/qdrant/query.go`)
   - Convert points to citation sources (`pointsToSources`)
   - Build system + user prompts
   - Call chat model (`internal/llm/openrouter.go`)
   - Save assistant turn to Redis
4. Response includes `answer` and `sources` to UI.

## Key files and responsibilities

- `main.go`
  - Config load, provider selection, dependency wiring
  - Startup logging and graceful shutdown
- `internal/config/config.go`
  - `.env` + environment loading
  - defaults and value validation
- `internal/rag/rag.go`
  - RAG orchestration, retrieval signal heuristics, prompt construction
- `internal/qdrant/query.go`
  - Vector query to Qdrant `/collections/{name}/points/query`
- `internal/qdrant/collections.go`
  - Collection info + named vector size lookup
- `internal/memory/redis.go`
  - Chat history storage and JSON cache helpers
- `internal/embed/`
  - `ollama.go` and `openrouter.go` embedding providers
- `internal/llm/openrouter.go`
  - Chat completion provider
- `web/app.js`
  - Front-end chat interactions and source rendering

## Current prompt/rag behavior to preserve

From observed code in `internal/rag/rag.go`:

- Retrieval signal is inferred from question + retrieved sources:
  - mode (`knowledge|advisory|creative`)
  - high-risk synthesis flag
  - cluster (`faq|narrative|mixed`)
  - top score, score gap, spread
  - strictness override from config
- Prompt rules include:
  - use only supplied context for factual claims
  - plain uncertainty when unsupported
  - bracket citations like `[1]`
- Engine logs retrieval signal snapshot.
- Server additionally logs chat debug signal from question + returned sources.

When changing RAG behavior, keep these guarantees unless explicitly asked to change them.

## Environment keys currently used

From `internal/config/config.go`:

- `APP_PORT`
- `OPENROUTER_API_KEY`
- `OPENROUTER_MODEL`
- `OPENROUTER_SITE_URL`
- `OPENROUTER_SITE_NAME`
- `QDRANT_URL`
- `QDRANT_API_KEY`
- `QDRANT_COLLECTION`
- `QDRANT_VECTOR_NAME`
- `QDRANT_TOP_K`
- `EMBED_PROVIDER`
- `OLLAMA_URL`
- `OLLAMA_EMBED_MODEL`
- `OPENROUTER_EMBED_BASE_URL`
- `OPENROUTER_EMBED_MODEL`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `REDIS_TTL_SECONDS`
- `CHAT_HISTORY_TURNS`
- `CACHE_EMBEDDINGS`
- `CACHE_QDRANT_RESULTS`
- `ARCHIMIND_STRICTNESS` (`strict|balanced|exploratory`, default `balanced`)

## Data contracts / API shape

### `/api/chat`

- Request JSON:
  - `session_id` (optional; defaults to `default`)
  - `message` (required)
  - `collection` (optional override)
- Response JSON:
  - `answer` string
  - `sources` array of `rag.Source`:
    - `index`, `score`, `title`, `page`, `chunk`, `source`, `text`

### `/api/health`

- Returns status JSON (`status`, `app`).

### `/api/collection`

- Reads optional query param `name`.
- Returns Qdrant collection info passthrough payload.

## Coding conventions seen in repo

- Keep packages small and focused by domain under `internal/`.
- Prefer explicit error returns with context (`fmt.Errorf(...)`).
- HTTP client timeouts are explicit in provider/client constructors.
- Log with `logger.Printf` for operational visibility.
- Avoid introducing new third-party dependencies unless needed.

## Gotchas and implementation details

- Chat history expiration is fixed to `24*time.Hour` in `SaveTurn` (not using `REDIS_TTL_SECONDS`).
- Cache entries for embeddings/Qdrant use `REDIS_TTL_SECONDS`.
- Qdrant query uses `with_payload=true` and `with_vector=false`.
- Named-vector dimension validation happens at startup (`VectorSize`) and again before query result use.
- If no points are returned, `Ask` returns a fixed fallback sentence and empty sources.
- `web/app.js` renders source snippets in a `<details>` section; long answers/sources should remain readable plain text.

## Safe edit playbook for future agents

1. Read full target files before editing.
2. For RAG changes, inspect both:
   - `internal/rag/rag.go`
   - `internal/server/server.go` (extra logging/debug signal usage)
3. Keep API response fields backward compatible unless user asks for breaking changes.
4. Run after each meaningful change:

```bash
go test ./...
```

5. Final validation before handoff:

```bash
gofmt -w .
go mod tidy
go build ./...
```
