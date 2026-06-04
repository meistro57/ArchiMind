# ArchiMind

ArchiMind is a Go-based retrieval cockpit for querying Qdrant collections with source-cited answers, retrieval diagnostics, and runtime model controls.

## Stack

- Go 1.22+
- Qdrant (vector retrieval)
- Redis (chat history + caches)
- OpenRouter (chat + optional embeddings)
- Ollama (optional embeddings)
- Static web UI (`web/`)

## Core capabilities

- Source-cited chat with diagnostics (`/api/chat`)
- Answer modes: `normal`, `skeptical`, `synthesis`, `diagnostic`
- Collection comparison (`/api/compare`)
- Framework extraction (`/api/framework`)
- Last-answer review (`/api/review/last`)
- Session export (`/api/export/markdown`, `/api/export/json`)
- Background report generation (`/api/report`)
- Runtime chat-model switching (`GET /api/models`, `POST /api/model`)
- Collection/vector discovery in UI (`GET /api/collections`)
- Assistant response copy-to-clipboard actions in the web UI
- Hybrid retrieval (dense + BM25 RRF) via `HYBRID_SEARCH`
- Meta-reflection fan-out retrieval (`meta_reflections` -> `mb_chunks`) for exact keyword recall
- Dead reflection filtering (`reflection_confidence == 0` or `is_empty_reflection == true`)
- Local Qdrant worker pool for faster retrieval job handling

## Retrieval behavior highlights

- `meta_reflections` queries fan out to `mb_chunks`, then merge/deduplicate/re-rank before prompting.
- Reflection points with zero confidence or empty-reflection flag are dropped before context assembly.
- Hybrid retrieval fuses dense ranks with BM25 lexical ranks when `HYBRID_SEARCH=true`.
- Worker pool executes Qdrant query jobs locally (goroutine pool + queue) to reduce request-path stalls.

## Runtime model switching

- The UI model dropdown loads options from `GET /api/models`.
- Changing the model calls `POST /api/model` and takes effect on the next chat request.
- The list is cached in Redis at key `openrouter:models`.
- Background report generation uses a dedicated worker model: `google/gemini-2.0-flash-001`.

## Quick start

1. Copy env template:

```bash
cp .env.example .env
```

2. Start local infra (Qdrant + Redis):

```bash
./scripts/docker/install_qdrant_redis.sh
```

3. Run app:

```bash
./scripts/rebuild-and-start.sh
```

4. Open:

```text
http://localhost:8090
```

## Environment

See `.env.example` for all supported keys. Minimum commonly-used keys:

```env
APP_PORT=8090
OPENROUTER_API_KEY=sk-or-...
OPENROUTER_MODEL=deepseek/deepseek-r1
QDRANT_URL=http://localhost:6333
QDRANT_COLLECTION=meta_reflections
QDRANT_VECTOR_NAME=claims_vec
QDRANT_TOP_K=8
EMBED_PROVIDER=openrouter
OPENROUTER_EMBED_MODEL=openai/text-embedding-3-small
REDIS_ADDR=localhost:6379
CACHE_EMBEDDINGS=true
CACHE_QDRANT_RESULTS=true
HYBRID_SEARCH=true
```

## HTTP API

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

## Development commands

```bash
gofmt -w .
go test ./...
go build ./...
go mod tidy
```

CI runs in `.github/workflows/tests.yml` on push/PR.
