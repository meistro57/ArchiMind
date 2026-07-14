# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project follows Semantic Versioning.

## [Unreleased]

### Added
- Runtime chat-model switching API: `GET /api/models` and `POST /api/model`.
- Web model selector that loads available OpenRouter models and switches active chat model live.
- Assistant response copy-to-clipboard controls in the web UI.
- Qdrant local worker pool for retrieval jobs (`internal/qdrant/worker.go`).
- Unit tests for worker pool behavior and OpenRouter embedding fallback parsing.

### Changed
- `meta_reflections` retrieval now fans out to `mb_chunks`, merges/deduplicates/re-ranks, and caps to `QDRANT_TOP_K`.
- Reflection points with `reflection_confidence == 0` or `is_empty_reflection == true` are filtered before context assembly.
- OpenRouter embedding parsing now handles nested/variant response shapes and surfaces provider errors more clearly.
- Background report worker now uses `google/gemini-2.0-flash-001` as its chat model.
- Retrieval now resolves vector configuration per selected collection and validates embedding dimensions before querying Qdrant.
- Chat/compare errors now return precise mismatch diagnostics (including Qdrant status/detail/endpoint) and use HTTP 400 for dimension/vector request issues.
- Framework UI validation now only requires a non-empty message, and allows blank collection to use `.env` defaults.
- Documentation refreshed to reflect API, retrieval, worker, and runtime model-switch behavior.

## [0.1.0] - 2026-05-05

### Added
- Initial ArchiMind release with Go HTTP server, RAG pipeline, Redis memory/cache, Qdrant integration, and static web UI.
