# README.md
# ArchiMind

ArchiMind is a Go-based retrieval app for querying Qdrant collections with source-cited answers.

## Stack

- Go 1.22+
- Qdrant (retrieval)
- Redis (chat history + cache)
- OpenRouter (chat model)
- Ollama or OpenRouter (embeddings)
- Static web UI (`web/`)

## Features

- Source-cited chat with retrieval diagnostics
- Answer modes: `normal`, `skeptical`, `synthesis`, `diagnostic`
- Collection + vector-aware querying (`vector_name` on chat/compare/framework)
- Collection comparison (`/api/compare`)
- Framework extraction (`/api/framework`)
- Last-answer review (`/api/review/last`)
- Session export to Markdown/JSON
- Background report generation (`/api/report`)
- Dynamic collection/vector discovery in the web UI (`/api/collections`)
- Guided preset question chips per selected collection

---

## What is ArchiMind?

**ArchiMind** is a browser-based chatbot for exploring embedded knowledge stored in **Qdrant** collections.

It uses retrieval-augmented generation to search your archive, pull relevant chunks, and answer with cited source context. It is designed for more than simple “chat with docs” behaviour: ArchiMind aims to separate **evidence-grounded claims**, **reasonable synthesis**, and **speculative interpretation** so your archive does not turn into a glitter cannon of confident nonsense.

In short:

> **ArchiMind is a source-aware retrieval cockpit for your ideas, documents, notes, reports, and research collections.**

---

## Core Stack

| Layer | Tool | Purpose |
|---|---|---|
| **Backend** | Go | Fast, clean local web server |
| **Chat model** | OpenRouter | Generates source-aware answers |
| **Embeddings** | Ollama or OpenRouter | Converts questions into searchable vectors |
| **Vector database** | Qdrant | Stores and searches embedded archive chunks |
| **Memory/cache** | Redis | Stores chat history and cached retrieval results |
| **Frontend** | Static HTML/CSS/JS | Lightweight browser interface |

---

## What it uses

- **Chat model:** OpenRouter (`internal/llm/openrouter.go`)
- **Embeddings:** Ollama or OpenRouter (`internal/embed/`)
- **Vector retrieval:** Qdrant (`internal/qdrant/`)
- **Memory/cache:** Redis (`internal/memory/redis.go`)
- **RAG logic:** Source-aware prompt assembly (`internal/rag/`)
- **UI:** Static browser app in `web/`
- **Background reports:** Reporter agent (`internal/reporter/agent.go`) using Qdrant + OpenRouter

---

## Interface Preview

<p align="center">
  <img
    width="1138" height="1811" alt="image" src="https://github.com/user-attachments/assets/22ad90ac-376c-4591-8c85-6f2cc04030a9">
</p>

---

## Why ArchiMind exists

Most RAG tools can retrieve chunks and produce an answer.

ArchiMind is being built to do something slightly fussier and more useful:

- search Qdrant collections by semantic meaning
- preserve source citations in answers
- use Redis for recent chat context and caching
- inspect collection/vector settings before querying
- avoid mixing unrelated retrieved chunks into one dramatic mega-theory
- distinguish grounded evidence from speculative synthesis
- support both practical archive Q&A and deeper pattern analysis

It is meant to help explore archives without losing track of **what the sources actually support**.

---

## Quick start

1. Copy env template and set values:

```bash
cp .env.example .env
```

2. Required settings in `.env`:

```env
APP_PORT=8090

OPENROUTER_API_KEY=sk-or-...
OPENROUTER_MODEL=deepseek/deepseek-r1
OPENROUTER_SITE_URL=http://localhost:8090
OPENROUTER_SITE_NAME=ArchiMind

## 2) Start local infrastructure (Qdrant + Redis)

Use the helper script:

```bash
./scripts/docker/install_qdrant_redis.sh
```

This script will:
- verify Docker availability,
- pull current images,
- create/reuse `archimind-qdrant` and `archimind-redis` containers,
- mount named volumes for persistent local data.

## 3) Run
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=
QDRANT_COLLECTION=your_collection
QDRANT_VECTOR_NAME=claims_vec
QDRANT_TOP_K=8

EMBED_PROVIDER=openrouter
OPENROUTER_EMBED_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_EMBED_MODEL=openai/text-embedding-3-small

# or EMBED_PROVIDER=ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_EMBED_MODEL=nomic-embed-text:latest

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_TTL_SECONDS=3600
CHAT_HISTORY_TURNS=12
CACHE_EMBEDDINGS=true
CACHE_QDRANT_RESULTS=true

ARCHIMIND_STRICTNESS=balanced
```

3. Run:

```bash
./scripts/rebuild-and-start.sh
```

(Alternative: `go run .`)

4. Open:

```text
http://localhost:8090
```

## Development commands

```bash
gofmt -w .
go test ./...
go build ./...
go mod tidy
./scripts/rebuild-and-start.sh
```

## Continuous integration (GitHub Actions)

CI is configured in `.github/workflows/ci.yml` and runs on pushes and pull requests. The workflow performs:
- formatting check (`gofmt -l .`),
- dependency download,
- unit tests (`go test ./... -v`),
- full build (`go build ./...`).

## Versioning and release notes

- ArchiMind uses **Semantic Versioning** (`MAJOR.MINOR.PATCH`).
- Changes are tracked in `CHANGELOG.md`.
- Add all unreleased work under the `## [Unreleased]` section, then cut a dated version section when releasing.

## Web UI highlights

- Top controls auto-load available collections from `GET /api/collections`
- Vector selector appears only when named vectors exist in a selected collection
- Empty-state preset chips can prefill high-signal starter questions
- Chat failures return structured diagnostics (`error`, `code`, `hint`) and are surfaced in the UI

## HTTP API

### `POST /api/chat`

Request:

```json
{
  "session_id": "optional-session-id",
  "message": "your question",
  "collection": "optional-collection-override",
  "vector_name": "optional-vector-name",
  "mode": "normal"
}
```

Success response:

```json
{
  "answer": "assistant response",
  "sources": [],
  "themes": [],
  "contradictions": [],
  "source_influence": [],
  "strong_claims": [],
  "diagnostics": {
    "grounded_claims": 0,
    "speculative_claims": 0,
    "unsupported_claims": 0,
    "unsupported_leap_risk": "low",
    "self_audit_checklist": []
  }
}
```

Failure response:

```json
{
  "error": "embedding and collection vector dimensions do not match",
  "code": "embedding_dimension_mismatch",
  "hint": "Verify QDRANT_VECTOR_NAME and selected embedding model dimensions."
}
```

### `POST /api/compare`

```json
{
  "session_id": "optional-session-id",
  "message": "compare these",
  "left_collection": "collection_a",
  "right_collection": "collection_b",
  "vector_name": "optional-vector",
  "mode": "synthesis"
}
```

### `POST /api/framework`

```json
{
  "session_id": "optional-session-id",
  "message": "extract a framework",
  "collection": "optional-collection",
  "vector_name": "optional-vector"
}
```

### `POST /api/review/last`

```json
{
  "session_id": "optional-session-id"
}
```

### `POST /api/export/markdown`

```json
{
  "session_id": "optional-session-id"
}
```

### `POST /api/export/json`

```json
{
  "session_id": "optional-session-id"
}
```

### `POST /api/report`

```json
{
  "topic": "history of retrieval architecture"
}
```

Returns:

```json
{
  "message": "report generation started",
  "output_path": "reports/history_of_retrieval_architecture_20260505_120000.md"
}
```

### `GET /api/health`

Returns service status and app version.

### `GET /api/collection?name=<collection>`

Returns raw collection info for a specific or default collection.

### `GET /api/collections`

Returns all collection names and discovered vector names.
Collection listing follows Qdrant pagination (`next_page_offset`) so large installs and older/newer response shapes are fully enumerated.

```json
{
  "collections": [
    {
      "name": "my_collection",
      "vectors": ["claims_vec", "chunks_vec"]
    }
  ]
}
```

## Project structure

```text
ArchiMind/
├── main.go
├── internal/
│   ├── config/
│   ├── embed/
│   ├── llm/
│   ├── logging/
│   ├── memory/
│   ├── qdrant/
│   ├── rag/
│   ├── reporter/
│   ├── server/
│   └── skills/
├── web/
└── .github/workflows/tests.yml
```
