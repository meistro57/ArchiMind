# ArchiMind Roadmap

<p align="center">
  <strong>From source-aware chatbot to archive intelligence system.</strong>
</p>

<p align="center">
  <em>Ask the archive. Trace the sources. Forge the signal.</em>
</p>

---

## Grand Vision

**ArchiMind** is evolving into a disciplined knowledge exploration system for Qdrant archives, research collections, notes, reports, books, and idea systems.

It is not just a chatbot.

It is a retrieval cockpit for:

- asking questions across structured and unstructured archives
- retrieving relevant source material from Qdrant
- preserving source boundaries
- separating evidence from synthesis
- detecting contradictions, drift, and conceptual clusters
- helping users turn archives into maps, frameworks, reports, tools, and action

> **ArchiMind becomes a disciplined archive intelligence system: part analyst, part librarian, part synthesis engine, part research workbench.**

---

## Core Identity

### What ArchiMind is

A **Go-based RAG application** that lets users query Qdrant collections through a clean browser interface, with:

- **OpenRouter** for chat and embeddings
- **Qdrant** for semantic retrieval
- **Redis** for short-term memory and caching
- **Go** for backend orchestration
- **Static HTML/CSS/JS** for a lightweight browser UI

### What makes it different

ArchiMind is not trying to be another generic “chat with docs” tool.

Its differentiators are:

- source-aware answers
- retrieval discipline
- clear boundary between grounded evidence and speculative synthesis
- cluster-aware reasoning
- archive-as-system thinking
- practical diagnostics for vector collections and retrieval quality

---

## North Star Goals

### 1. Trustworthy Retrieval

A user should be able to trust that ArchiMind:

- searched the right collection
- used the correct vector name
- used the correct embedding dimensions
- cited actual retrieved sources
- did not silently invent unsupported structure

### 2. Useful Synthesis

A user should be able to ask:

- “What are the recurring ideas?”
- “What is the strongest original contribution here?”
- “What contradictions exist?”
- “What can be built from this archive?”

…and get answers that are structured, useful, and honest.

### 3. Knowledge Workbench

A user should be able to move beyond Q&A and use ArchiMind to:

- compare collections
- extract frameworks
- generate outlines
- build reports
- detect source clusters
- identify actionable ideas
- track evolving themes over time

### 4. Multi-Mode Thinking

A user should be able to switch between answer modes such as:

- normal
- skeptical
- synthesis
- diagnostic
- compare
- builder
- research

---

## Product Pillars

### Pillar 1: Retrieval Integrity

Make sure the system is actually searching properly before it tries to sound clever.

#### Capabilities

- named vector support
- vector dimension verification
- collection inspection
- provider/model validation
- cache separation by provider/model/vector name
- retrieval diagnostics

#### Success looks like

- fewer retrieval errors
- fewer confident nonsense answers
- reliable source matching
- clear local errors before Qdrant throws cryptic failures

---

### Pillar 2: Source-Aware Reasoning

Make the answer logic smarter, cleaner, and more honest.

#### Capabilities

- evidence-grounded claims
- speculative sections clearly labeled
- cluster-aware synthesis
- contradiction detection
- source-weighting heuristics
- prompt strictness modes

#### Success looks like

- cleaner answers
- less over-fusion
- more trust in the output
- better distinction between literal mechanism and metaphor

---

### Pillar 3: Archive Intelligence

Turn archives into systems, not just answer blobs.

#### Capabilities

- recurring theme extraction
- strongest-claim analysis
- contradiction mapping
- worldview summarization
- concept graph generation
- framework extraction
- “what can be built from this?” mode

#### Success looks like

- answers become reusable
- users can see patterns across collections
- archives become design material

---

### Pillar 4: Workflow Power

Make ArchiMind a serious daily-use thinking tool.

#### Capabilities

- session memory
- saved chats
- saved prompts
- export reports
- collection switching
- prompt presets
- analyst modes
- side-by-side comparison

#### Success looks like

- people use it daily
- it supports real research and development
- it becomes a cockpit, not a novelty

---

## Roadmap Phases

---

## Phase 1: Foundation

**Theme:** Make it solid.

### Objectives

- stable Go web app
- working OpenRouter chat
- working OpenRouter/Ollama embeddings
- Qdrant retrieval
- Redis memory/cache
- browser UI
- source display
- collection inspection
- dimension mismatch protection

### Deliverables

- [x] Go server
- [x] Qdrant query pipeline
- [x] Redis chat history
- [x] basic web UI
- [x] source citations
- [x] named vector support
- [x] embedding provider support
- [x] vector dimension diagnostics
- [ ] improved retrieval diagnostic messaging
- [ ] polished documentation

### Definition of Done

ArchiMind is reliable enough to use without hand-holding every five minutes.

---

## Phase 2: Reasoning Discipline

**Theme:** Make it trustworthy.

### Objectives

- reduce over-synthesis
- detect cluster mismatch
- classify claims
- add answer modes
- improve prompt architecture

### Deliverables

- [x] answer mode system
  - [x] normal
  - [x] skeptical
  - [x] synthesis
  - [x] diagnostic
- [x] cluster heuristic labeling
- [x] prompt strictness setting
  - [x] strict
  - [x] balanced
  - [x] exploratory
- [x] evidence vs speculation separation
- [x] unsupported leap detection
- [x] “review your last answer” self-audit capability

### Definition of Done

ArchiMind can explain **why** it answered a certain way and where it may be overreaching.

---

## Phase 3: Archive Intelligence Layer

**Theme:** Make it insightful.

### Objectives

- analyze archives as systems
- detect patterns across retrieved material
- produce structured, reusable outputs

### Deliverables

- [ ] recurring theme analysis
- [ ] contradiction map
- [x] strongest claims ranking
- [ ] concept clustering
- [x] source influence ranking
- [ ] worldview summarizer
- [ ] original contribution detector
- [x] framework extractor
- [ ] book/report outline generator

### Definition of Done

ArchiMind becomes a real synthesis instrument, not just a question-answer box.

---

## Phase 4: Multi-Collection Intelligence

**Theme:** Connect islands into maps.

### Objectives

- compare collections
- synthesize across source groups
- detect overlap and divergence

### Deliverables

- [ ] compare two collections
- [ ] compare claims across collections
- [ ] collection similarity analysis
- [ ] cross-collection worldview map
- [ ] “where do these archives agree/disagree?” mode
- [ ] source family detection
- [ ] unified thesis builder with boundary warnings

### Example Use Cases

- compare Seth vs Dolores
- compare a user-authored book vs archive notes
- compare metaphysical vs practical/engineering content
- compare `claims_vec` vs `summary_vec`

### Definition of Done

ArchiMind can move from **archive lookup** to **archive comparison**.

---

## Phase 5: Research Workbench

**Theme:** Make it operational.

### Objectives

Turn ArchiMind into a daily-use research and thinking environment.

### Deliverables

- [ ] saved sessions
- [ ] saved question presets
- [ ] export to Markdown
- [ ] export to JSON
- [ ] report generator
- [ ] analyst workspaces
- [ ] collection dashboard
- [ ] session tagging
- [ ] query history
- [ ] retrieval replay / inspection

### Definition of Done

A user can conduct and preserve real research sessions inside ArchiMind.

---

## Phase 6: Skill System

**Theme:** Give it tools.

### Objectives

Add a structured skill layer for actions beyond basic answering.

### Deliverables

- [ ] skill registry
- [ ] `inspect_collection` skill
- [ ] `compare_collections` skill
- [ ] `summarize_sources` skill
- [ ] `contradiction_report` skill
- [ ] `extract_framework` skill
- [ ] `generate_outline` skill
- [ ] `build_report` skill
- [ ] `source_gap_analysis` skill

### Longer-Term Ideas

- [ ] MCP integration
- [ ] tool routing
- [ ] skill selection by answer mode
- [ ] scheduled/background analysis jobs

### Definition of Done

ArchiMind can choose structured tools to perform useful work, not just generate prose.

---

## Phase 7: Visual Knowledge Layer

**Theme:** Let users see the archive.

### Objectives

Make patterns visible.

### Deliverables

- [ ] source cluster map
- [ ] concept graph
- [ ] retrieval heatmap
- [ ] collection similarity graph
- [ ] contradiction network
- [ ] timeline view
- [ ] session memory map
- [ ] vector neighborhood explorer

### Definition of Done

A user can visually inspect the shape of an archive rather than only reading answers.

---

## Phase 8: Platform Layer

**Theme:** ArchiMind becomes a knowledge engine.

### Objectives

Turn ArchiMind into a platform for archive exploration and disciplined synthesis.

### Deliverables

- [ ] multiple project workspaces
- [ ] multiple users/profiles
- [ ] pluggable providers
- [ ] ingestion pipeline integration
- [ ] batch archive analysis
- [ ] automated insight reports
- [ ] source health checks
- [ ] archive maturity score
- [ ] “best next question” suggestion engine
- [ ] strategic briefing mode

### Definition of Done

ArchiMind becomes a serious platform for personal, creative, and research-grade knowledge work.

---

## High-Value Feature Ideas

### Analyst Features

- “What are the strongest recurring claims?”
- “What contradictions are present?”
- “Which sources are doing the most work?”
- “What is the simplest honest thesis here?”
- “What is original vs derivative?”
- “What is metaphorical vs literal?”
- “Where does this archive confuse mechanism and metaphor?”

### Builder Features

- “Turn this archive into a framework.”
- “Turn this archive into a product idea.”
- “Turn this archive into a report.”
- “Turn this archive into a GitHub README.”
- “Turn this archive into a research outline.”

### Diagnostic Features

- “Why did this answer overreach?”
- “What clusters were retrieved?”
- “Was this a good retrieval set?”
- “What should not have been merged?”
- “What was missing from retrieval?”

### Comparative Features

- “Compare these two collections.”
- “What overlaps?”
- “Where do they disagree?”
- “What worldview emerges from both?”

---

## Technical Direction

### Backend

- Go remains the orchestration layer
- clean package boundaries
- graceful shutdown
- structured logging
- stronger config validation

### Retrieval

- Qdrant Query API
- named vectors
- multi-vector support
- collection introspection
- retrieval inspection tools

### Providers

- OpenRouter chat
- OpenRouter embeddings
- Ollama optional fallback
- future pluggable provider interface

### Memory

Redis supports:

- chat turns
- cached embeddings
- cached retrieval results
- temporary state
- session context

### UI

Keep it lightweight but powerful:

- source visibility
- collection switching
- answer modes
- saved sessions
- export buttons
- diagnostic views
- visual exploration

---

## Success Metrics

### Product Quality

- answers cite sources correctly
- fewer unsupported synthesis errors
- fewer retrieval mismatches
- higher answer consistency

### User Usefulness

- faster time to useful answer
- more continued sessions
- more exports/reports generated
- more collections successfully explored

### Intelligence Quality

ArchiMind should consistently distinguish:

- grounded claims
- reasonable synthesis
- speculation
- contradiction
- metaphor
- literal mechanism
- unsupported leap

---

## Immediate Next Steps

### Right Now

- [x] finish discipline upgrade in RAG prompt logic
- [x] add answer modes
- [x] add cluster heuristics
- [x] improve collection inspection output
- [x] add clearer retrieval diagnostics
- [x] add `ROADMAP.md` to the repo
- [ ] polish README and screenshots

### Next

- [x] compare-collections mode
- [x] export answer/report to Markdown
- [x] source ranking
- [x] contradiction finder
- [x] framework extraction mode

---

## The Big Statement

> **ArchiMind is evolving from a chatbot into an archive intelligence system.**  
> First it answers questions.  
> Then it explains its sources.  
> Then it maps patterns.  
> Then it compares systems.  
> Then it helps build new ones.

---

## Working Principle

ArchiMind should remain useful, imaginative, and exploratory without losing sight of the source boundary.

The central line it must preserve is:

```text
What the sources say
What the model can reasonably connect
What is speculative
What is unsupported
```

That line is the difference between a knowledge engine and soup with footnotes.
