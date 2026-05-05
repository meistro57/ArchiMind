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
  <em>Ask your collections. Trace the sources. Keep the speculation on a leash.</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.22+" />
  <img src="https://img.shields.io/badge/Qdrant-Vector_Search-DC244C?style=for-the-badge" alt="Qdrant" />
  <img src="https://img.shields.io/badge/OpenRouter-LLM_Provider-7C3AED?style=for-the-badge" alt="OpenRouter" />
  <img src="https://img.shields.io/badge/Redis-Memory_&_Cache-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis" />
</p>

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

---

## Interface Preview

<p align="center">
  <img
    src="https://github.com/user-attachments/assets/f7016739-5658-43dd-9931-d8dc091d7556"
    alt="ArchiMind interface screenshot"
    width="900"
  />
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

### 1) Configure environment

Set these variables. For local development, they can live in `.env`, loaded automatically by `godotenv` in `internal/config/config.go`.

#### Core

```env
APP_PORT=8090

OPENROUTER_API_KEY=your_openrouter_key_here
OPENROUTER_MODEL=deepseek/deepseek-r1
OPENROUTER_SITE_URL=http://localhost:8090
OPENROUTER_SITE_NAME=ArchiMind
Qdrant
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=
QDRANT_COLLECTION=your_collection_name
QDRANT_VECTOR_NAME=claims_vec
QDRANT_TOP_K=8
Embeddings

Use OpenRouter embeddings:

EMBED_PROVIDER=openrouter
OPENROUTER_EMBED_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_EMBED_MODEL=openai/text-embedding-3-small

Or use Ollama embeddings:

EMBED_PROVIDER=ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_EMBED_MODEL=nomic-embed-text:latest

The embedding model must match the vector dimensions used when the Qdrant collection was created. If Qdrant expects 1536, do not query it with a 768-dimension embedding model unless you enjoy being bonked by vector goblins.

Redis / memory
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_TTL_SECONDS=3600
CHAT_HISTORY_TURNS=12
CACHE_EMBEDDINGS=true
CACHE_QDRANT_RESULTS=true
Prompt strictness
ARCHIMIND_STRICTNESS=balanced

Accepted values:

strict
balanced
exploratory
2) Run
export GOTOOLCHAIN=local
go run .

Then open:

http://localhost:8090
Development commands
# Format
gofmt -w .

# Dependency cleanup
go mod tidy

# Compile all packages
go build ./...

# Run tests
go test ./...
HTTP API
POST /api/chat

Request:

{
  "session_id": "optional-session-id",
  "message": "your question",
  "collection": "optional-collection-override"
}

Response:

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
GET /api/health

Returns service status.

GET /api/collection?name=<collection>

Returns raw Qdrant collection info. If name is omitted, ArchiMind uses the configured default collection.

Project layout
ArchiMind/
├── main.go
├── internal/
│   ├── config/      # Environment loading and validation
│   ├── embed/       # Embedding providers
│   ├── llm/         # Chat provider interface and OpenRouter implementation
│   ├── memory/      # Redis chat history and cache storage
│   ├── qdrant/      # Qdrant API client
│   ├── rag/         # Retrieval, prompt assembly, source extraction
│   ├── server/      # HTTP handlers and static web serving
│   └── skills/      # Future skill registry
└── web/             # Browser interface
Notes and gotchas
Chat history is stored per session key:
chat:<session_id>:history
CHAT_HISTORY_TURNS controls how many recent turns Redis keeps.
Embedding cache keys include provider and model names to avoid stale vector reuse.
Qdrant result cache keys include collection, vector name, provider, and model.
Named-vector collections require QDRANT_VECTOR_NAME.
On startup, ArchiMind attempts to fetch Qdrant vector dimensions and warns when embedding dimensions do not match.
If no relevant retrieval hits are returned, ArchiMind gives a clear “could not find anything relevant” response.
Philosophy

ArchiMind is not trying to be an all-knowing oracle.

It is a retrieval instrument: part archive lantern, part source clerk, part suspicious little analyst with a clipboard.

Its job is to help you ask better questions of your own knowledge systems while keeping the line visible between:

What the sources say
What the model can reasonably connect
What is speculative
What is unsupported

That line matters. Without it, every archive eventually becomes soup with footnotes.
