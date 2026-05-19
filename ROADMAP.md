# ArchiMind Roadmap

This roadmap tracks what is done, what is in progress, and what is next.

## Current state (v0.6.x)

### Completed

- Source-cited chat over Qdrant (`/api/chat`)
- Retrieval discipline signals and diagnostics
- Answer modes (`normal`, `skeptical`, `synthesis`, `diagnostic`)
- Collection comparison (`/api/compare`)
- Framework extraction (`/api/framework`)
- Last-answer review (`/api/review/last`)
- Session export to Markdown/JSON (`/api/export/*`)
- Background report generation (`/api/report`)
- Named vector support and vector dimension validation
- Collection listing pagination compatibility across Qdrant response shapes

### In progress

- Tightening retrieval diagnostics UX in the web client
- Improving documentation and examples

## Near-term priorities

1. Retrieval quality tooling
   - Better “why this answer” explanation in UI
   - Stronger unsupported-claim warnings

2. Multi-collection intelligence
   - Richer compare output structure
   - Better overlap/divergence summaries

3. Report and export workflow
   - Report status visibility in UI
   - Cleaner export naming and metadata

4. Session workflow
   - Saved sessions and quick reload
   - Better history browsing for long runs

## Mid-term priorities

- Collection-level dashboards (health, vectors, coverage)
- Contradiction map improvements
- Theme and claim ranking refinements
- Optional skill routing from `internal/skills`

## Long-term direction

- Visual archive maps and cluster exploration
- More pluggable provider support
- Workspace-level project organization

## Definition of success

- Answers remain source-grounded and auditable
- Retrieval errors are obvious and actionable
- Users can move from Q&A to repeatable analysis workflows
