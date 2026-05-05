<h1 align="center">ArchiMind</h1>

<p align="center">
  <img
    src="https://github.com/user-attachments/assets/d94afabb-29e8-4265-9b1e-a1e941031102"
    alt="ArchiMind logo"
    width="420"
  />
</p>

<p align="center">
  <strong>A Go-powered RAG chatbot for querying Qdrant archives with OpenRouter, Redis memory, and source-aware synthesis.</strong>
</p>

<p align="center">
  <em>Ask your collections. Trace the sources. Keep speculation on a leash.</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.22+" />
  <img src="https://img.shields.io/badge/Qdrant-Vector_Search-DC244C?style=for-the-badge" alt="Qdrant" />
  <img src="https://img.shields.io/badge/OpenRouter-LLM_Provider-7C3AED?style=for-the-badge" alt="OpenRouter" />
  <img src="https://img.shields.io/badge/Redis-Memory_&_Cache-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis" />
</p>

---

## What is ArchiMind?

ArchiMind is a browser-based chatbot for exploring embedded knowledge stored in Qdrant collections.

It uses retrieval-augmented generation to search your archive, pull relevant chunks, and answer with source-backed context. The runtime emphasizes a clear line between evidence, synthesis, and speculation.

> **ArchiMind is a source-aware retrieval cockpit for notes, documents, reports, and research collections.**

---

## Core stack

| Layer | Tool | Purpose |
|---|---|---|
| Backend | Go | Local HTTP server and orchestration |
| Chat model | OpenRouter | Source-aware completion |
| Embeddings | Ollama or OpenRouter | Vectorize user questions |
| Vector database | Qdrant | Retrieve semantically relevant chunks |
| Memory/cache | Redis | Chat history and retrieval cache |
| Frontend | Static HTML/CSS/JS | Lightweight browser UI |

---

## Key capabilities

- Source-cited answers (`[1]`, `[2]`, ...)
- Retrieval signal analysis for safer response behavior
- Redis-backed per-session history and optional caching
- Startup vector-dimension checks against Qdrant collection config
- Async report generation via `POST /api/report`

---

## Interface preview

<p align="center">
  <img
    src="https://github.com/user-attachments/assets/f7016739-5658-43dd-9931-d8dc091d7556"
    alt="ArchiMind interface screenshot"
    width="900"
  />
</p>

---

## Quick start

### 1) Configure environment

Create `.env` (or set shell env vars directly):

```env
APP_PORT=8090

OPENROUTER_API_KEY=your_openrouter_key_here
OPENROUTER_MODEL=deepseek/deepseek-r1
OPENROUTER_SITE_URL=http://localhost:8090
OPENROUTER_SITE_NAME=ArchiMind

QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=
QDRANT_COLLECTION=your_collection_name
QDRANT_VECTOR_NAME=claims_vec
QDRANT_TOP_K=8

# Embeddings (choose one provider)
EMBED_PROVIDER=openrouter
OPENROUTER_EMBED_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_EMBED_MODEL=openai/text-embedding-3-small

# or
# EMBED_PROVIDER=ollama
# OLLAMA_URL=http://localhost:11434
# OLLAMA_EMBED_MODEL=nomic-embed-text:latest

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_TTL_SECONDS=3600
CHAT_HISTORY_TURNS=12
CACHE_EMBEDDINGS=true
CACHE_QDRANT_RESULTS=true

ARCHIMIND_STRICTNESS=balanced
```

`ARCHIMIND_STRICTNESS` accepted values: `strict`, `balanced`, `exploratory`.

### 2) Run

```bash
export GOTOOLCHAIN=local
go run .
```

Open `http://localhost:8090`.

---

## Development commands

```bash
# Format
gofmt -w .

# Dependency cleanup
go mod tidy

# Run tests
go test ./...

# Build
go build ./...
```

---

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

Returns raw Qdrant collection info. If `name` is omitted, ArchiMind uses the configured default collection.

---

## Project layout

```text
ArchiMind/
├── main.go
├── internal/
│   ├── config/      # Environment loading and validation
│   ├── embed/       # Embedding providers
│   ├── llm/         # Chat provider interface and OpenRouter implementation
│   ├── logging/     # Logger setup
│   ├── memory/      # Redis chat history and cache storage
│   ├── qdrant/      # Qdrant API client
│   ├── rag/         # Retrieval, prompt assembly, source extraction
│   ├── reporter/    # Async report generation
│   ├── server/      # HTTP handlers and static web serving
│   └── skills/      # Skill registry
└── web/             # Browser interface
```

---

## Notes and gotchas

- Chat history key: `chat:<session_id>:history`
- `CHAT_HISTORY_TURNS` controls retained turn count
- `SaveTurn` currently uses fixed 24h expiration for history keys
- Cache TTL uses `REDIS_TTL_SECONDS`
- Named-vector collections require `QDRANT_VECTOR_NAME`
- If retrieval returns no relevant hits, ArchiMind returns a clear fallback response

---

## Philosophy

ArchiMind is not trying to be an all-knowing oracle.

Its job is to help you ask better questions of your own knowledge while keeping the line visible between what the sources say, what the model can connect, and what remains unsupported.