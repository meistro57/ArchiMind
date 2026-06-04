# AGENTS.md

Practical guide for coding agents working in ArchiMind.

## Repository snapshot

- Language: Go 1.22
- Runtime: HTTP server + static web UI
- Services: Qdrant, Redis, OpenRouter, optional Ollama embeddings
- Entry point: `main.go`

## Fast orientation

1. Read `internal/config/config.go` for env keys/defaults.
2. Read `main.go` for wiring and startup behavior.
3. Read `internal/rag/rag.go` before changing answer behavior.
4. Read `internal/server/server.go` before changing API behavior.

## Validation commands

```bash
go test ./...
go build ./...
gofmt -w .
```

CI exists at `.github/workflows/tests.yml` and runs tests on push/PR.

## Architecture flow

1. `POST /api/chat` handled by `internal/server/server.go`.
2. Server calls `rag.Engine.Ask(...)`.
3. `Ask` does embedding, retrieval, source shaping, prompt build, LLM call, and memory write.
4. For `meta_reflections`, retrieval fans out to `mb_chunks`, merges/deduplicates/re-ranks, then filters dead reflection points before context assembly.
5. Retrieval can run through hybrid search (`HYBRID_SEARCH=true`) and Qdrant worker-pool dispatch.
6. Response returns answer + sources + diagnostics metadata.

Background report flow:

1. `POST /api/report` accepted by server.
2. `reporter.Agent.Generate(...)` runs in a goroutine.
3. Report worker uses `google/gemini-2.0-flash-001`.
4. Report is written to `reports/<topic>_<timestamp>.md`.

## Key files

- `main.go`: config load, provider setup, server lifecycle
- `internal/config/config.go`: env parsing/validation
- `internal/server/server.go`: HTTP handlers + route surface
- `internal/rag/rag.go`: core RAG orchestration + signal heuristics
- `internal/rag/compare.go`: collection comparison logic
- `internal/rag/framework.go`: framework extraction
- `internal/rag/review.go`: answer review/self-audit
- `internal/rag/export.go`: session export helpers
- `internal/qdrant/query.go`: vector query operations
- `internal/qdrant/worker.go`: worker-pool query dispatch
- `internal/qdrant/collections.go`: collection/vector introspection
- `internal/reporter/agent.go`: background report generation
- `web/app.js`: front-end request wiring and rendering

## API surface

- `POST /api/chat`
- `POST /api/compare`
- `POST /api/framework`
- `POST /api/review/last`
- `POST /api/export/markdown`
- `POST /api/export/json`
- `POST /api/report`
- `GET /api/health`
- `GET /api/collection`
- `GET /api/collections`
- `GET /api/models`
- `POST /api/model`

## Environment keys in use

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
- `HYBRID_SEARCH`
- `ARCHIMIND_STRICTNESS` (`strict|balanced|exploratory`)

## Behavioral constraints to preserve

- Factual claims should stay grounded in retrieved context.
- Unsupported claims should be surfaced as uncertainty.
- Citation format should remain bracket-index style (`[1]`).
- Retrieval diagnostics/signal logging should remain intact.

## Safe edit playbook

1. Read full files before edits.
2. Keep API responses backward compatible unless requested otherwise.
3. For shared logic, inspect all call sites before changing signatures.
4. Run `go test ./...` after each meaningful change.
5. Before handoff: `gofmt -w .`, `go test ./...`, `go build ./...`.
