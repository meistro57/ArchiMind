// internal/rag/rag.go
package rag

import (
	"context"
	"fmt"
	"log"
	"strings"

	"archimind/internal/config"
	"archimind/internal/embed"
	"archimind/internal/llm"
	"archimind/internal/memory"
	"archimind/internal/qdrant"
)

type Engine struct {
	cfg                config.Config
	qdr                *qdrant.Client
	chat               llm.Provider
	embedder           embed.Provider
	mem                *memory.RedisMemory
	logger             *log.Logger
	expectedVectorSize int
}

type Source struct {
	Index  int     `json:"index"`
	Score  float64 `json:"score"`
	Title  string  `json:"title"`
	Page   string  `json:"page,omitempty"`
	Chunk  string  `json:"chunk,omitempty"`
	Source string  `json:"source,omitempty"`
	Text   string  `json:"text"`
}

type AnswerMode string

const (
	AnswerModeKnowledge AnswerMode = "knowledge"
	AnswerModeAdvisory  AnswerMode = "advisory"
	AnswerModeCreative  AnswerMode = "creative"
)

type RetrievalSignal struct {
	Mode              AnswerMode
	HighRiskSynthesis bool
	TopScore          float64
	ScoreGap          float64
	SimilaritySpread  float64
	Cluster           string
	Strictness        string
}

func NewEngine(
	cfg config.Config,
	qdr *qdrant.Client,
	chat llm.Provider,
	embedder embed.Provider,
	mem *memory.RedisMemory,
	logger *log.Logger,
	expectedVectorSize int,
) *Engine {
	return &Engine{
		cfg:                cfg,
		qdr:                qdr,
		chat:               chat,
		embedder:           embedder,
		mem:                mem,
		logger:             logger,
		expectedVectorSize: expectedVectorSize,
	}
}

func (e *Engine) Ask(ctx context.Context, sessionID string, collection string, question string) (string, []Source, error) {
	_ = e.mem.SaveTurn(ctx, sessionID, "user", question, e.cfg.ChatHistoryTurns)

	vector, err := e.getEmbedding(ctx, question)
	if err != nil {
		return "", nil, err
	}

	points, err := e.getQdrantResults(ctx, collection, question, vector)
	if err != nil {
		return "", nil, err
	}

	if len(points) == 0 {
		answer := "I could not find anything relevant in that Qdrant collection."
		_ = e.mem.SaveTurn(ctx, sessionID, "assistant", answer, e.cfg.ChatHistoryTurns)
		return answer, nil, nil
	}

	sources := pointsToSources(points)
	signal := BuildSignal(question, sources)
	signal.Strictness = normalizeStrictness(e.cfg.Strictness)
	e.logger.Printf("rag signal mode=%s high_risk=%t cluster=%s strictness=%s top_score=%.4f score_gap=%.4f spread=%.4f", signal.Mode, signal.HighRiskSynthesis, signal.Cluster, signal.Strictness, signal.TopScore, signal.ScoreGap, signal.SimilaritySpread)

	contextBlocks := make([]string, 0, len(sources))

	for _, src := range sources {
		contextBlocks = append(contextBlocks, fmt.Sprintf(
			"[%d] %s\nScore: %.4f\nText:\n%s",
			src.Index,
			src.Title,
			src.Score,
			src.Text,
		))
	}

	history, _ := e.mem.GetHistory(ctx, sessionID)
	historyText := formatHistory(history)

	system := buildSystemPrompt(signal)

	user := fmt.Sprintf(
		"Recent chat history:\n%s\n\nQuestion:\n%s\n\nQdrant context:\n%s\n\nAnswer:",
		historyText,
		question,
		strings.Join(contextBlocks, "\n\n---\n\n"),
	)

	answer, err := e.chat.Chat(ctx, []llm.Message{
		{Role: "system", Content: strings.TrimSpace(system)},
		{Role: "user", Content: user},
	})
	if err != nil {
		return "", nil, err
	}

	_ = e.mem.SaveTurn(ctx, sessionID, "assistant", answer, e.cfg.ChatHistoryTurns)

	return answer, sources, nil
}

func InferAnswerMode(question string) AnswerMode {
	lower := strings.ToLower(strings.TrimSpace(question))

	creativeHints := []string{
		"brainstorm", "creative", "story", "poem", "tagline", "slogan", "name ideas", "imagine", "pitch", "metaphor", "script",
	}
	for _, hint := range creativeHints {
		if strings.Contains(lower, hint) {
			return AnswerModeCreative
		}
	}

	advisoryHints := []string{
		"how should", "what should", "should i", "recommend", "advice", "plan", "steps", "strategy", "improve", "optimize", "fix", "debug", "troubleshoot", "best way", "risk", "tradeoff", "trade-off",
	}
	for _, hint := range advisoryHints {
		if strings.Contains(lower, hint) {
			return AnswerModeAdvisory
		}
	}

	return AnswerModeKnowledge
}

func BuildSignal(question string, sources []Source) RetrievalSignal {
	signal := RetrievalSignal{
		Mode:              InferAnswerMode(question),
		HighRiskSynthesis: isHighRiskSynthesis(question),
		Cluster:           classifyCluster(sources),
	}

	if len(sources) == 0 {
		return signal
	}

	signal.TopScore = sources[0].Score
	signal.SimilaritySpread = sources[0].Score - sources[len(sources)-1].Score
	if len(sources) > 1 {
		signal.ScoreGap = sources[0].Score - sources[1].Score
	} else {
		signal.ScoreGap = sources[0].Score
	}

	return signal
}

func buildSystemPrompt(signal RetrievalSignal) string {
	strictnessRules := []string{
		"- Strictness profile: balanced.",
		"- Prefer grounded synthesis only when multiple retrieved chunks align.",
		"- If support is weak or conflicting, state uncertainty directly.",
	}
	if signal.Strictness == "strict" {
		strictnessRules = []string{
			"- Strictness profile: strict.",
			"- Treat retrieval as hard boundaries: only state claims explicitly supported by citations.",
			"- If evidence is incomplete, respond that the context does not support a confident answer.",
		}
	}
	if signal.Strictness == "exploratory" {
		strictnessRules = []string{
			"- Strictness profile: exploratory.",
			"- You may propose hypotheses or options, but mark them clearly as speculative.",
			"- Keep factual statements grounded in cited context.",
		}
	}

	modeRules := []string{
		"- Answer mode: knowledge. Prioritize concise factual explanation with citations.",
	}
	if signal.Mode == AnswerModeAdvisory {
		modeRules = []string{
			"- Answer mode: advisory. Give practical, step-by-step guidance.",
			"- Separate grounded recommendations from assumptions when evidence is partial.",
		}
	}
	if signal.Mode == AnswerModeCreative {
		modeRules = []string{
			"- Answer mode: creative. Structure a useful creative response.",
			"- Keep any factual grounding tied to citations, and label imaginative additions clearly.",
		}
	}

	clusterRules := []string{
		"- Cluster profile: mixed. Do not overfit to one style; explain source limits.",
	}
	if signal.Cluster == "faq" {
		clusterRules = []string{
			"- Cluster profile: faq. Favor atomic, citation-dense answers with explicit bullet points.",
		}
	}
	if signal.Cluster == "narrative" {
		clusterRules = []string{
			"- Cluster profile: narrative. Allow coherent synthesis across related chunks, but keep citations for key claims.",
		}
	}

	riskRules := []string{}
	if signal.HighRiskSynthesis {
		riskRules = []string{
			"- High-risk synthesis detected. Cross-check claims across multiple chunks before concluding.",
			"- If the evidence does not converge, explain the conflict and avoid definitive recommendations.",
		}
	}

	signalSnapshot := fmt.Sprintf("- Retrieval signal snapshot: mode=%s, high_risk=%t, top_score=%.4f, score_gap=%.4f, spread=%.4f, cluster=%s, strictness=%s.", signal.Mode, signal.HighRiskSynthesis, signal.TopScore, signal.ScoreGap, signal.SimilaritySpread, signal.Cluster, signal.Strictness)

	lines := []string{
		"You are ArchiMind, a retrieval-grounded chat interface for Qdrant collections.",
		"",
		"Rules:",
		"- Use only the supplied Qdrant context for factual claims.",
		"- If the context does not support the answer, say that plainly.",
		"- Cite retrieved context with bracket citations like [1], [2].",
		"- Be clear, practical, and useful.",
		"- Do not invent sources.",
	}
	lines = append(lines, strictnessRules...)
	lines = append(lines, modeRules...)
	lines = append(lines, clusterRules...)
	lines = append(lines, riskRules...)
	lines = append(lines, signalSnapshot)

	return strings.Join(lines, "\n")
}

func normalizeStrictness(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return "strict"
	case "exploratory":
		return "exploratory"
	default:
		return "balanced"
	}
}

func isHighRiskSynthesis(question string) bool {
	lower := strings.ToLower(strings.TrimSpace(question))
	hints := []string{
		"synthesize", "synthesis", "compare", "across", "overall", "tradeoff", "trade-off", "strategy", "roadmap", "forecast", "predict", "recommend", "why", "root cause", "best approach", "long-term",
	}
	for _, hint := range hints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

func classifyCluster(sources []Source) string {
	if len(sources) == 0 {
		return "mixed"
	}

	faqVotes := 0
	narrativeVotes := 0

	for _, src := range sources {
		text := strings.ToLower(strings.TrimSpace(src.Text))
		title := strings.ToLower(strings.TrimSpace(src.Title))
		combined := title + " " + text
		wordCount := len(strings.Fields(text))

		hasMetadata := strings.TrimSpace(src.Page) != "" || strings.TrimSpace(src.Chunk) != "" || strings.TrimSpace(src.Source) != ""
		if hasMetadata {
			faqVotes++
		}

		if strings.Contains(combined, "faq") || strings.Contains(combined, "question") || strings.Contains(combined, "answer:") || strings.HasPrefix(text, "q:") {
			faqVotes++
		}

		if wordCount >= 170 {
			narrativeVotes++
		}

		if strings.Contains(combined, "chapter") || strings.Contains(combined, "section") || strings.Contains(combined, "story") || strings.Contains(combined, "timeline") || strings.Contains(combined, "narrative") {
			narrativeVotes++
		}
	}

	if faqVotes >= narrativeVotes+2 {
		return "faq"
	}
	if narrativeVotes >= faqVotes+2 {
		return "narrative"
	}
	return "mixed"
}

func (e *Engine) getEmbedding(ctx context.Context, question string) ([]float64, error) {
	cacheKey := "embedding:" + e.cfg.EmbedProvider + ":" + e.embedder.ModelName() + ":" + memory.HashKey(question)

	if e.cfg.CacheEmbeddings {
		var cached []float64
		found, err := e.mem.GetJSON(ctx, cacheKey, &cached)
		if err == nil && found && len(cached) > 0 {
			return cached, nil
		}
	}

	vector, err := e.embedder.Embed(ctx, question)
	if err != nil {
		return nil, err
	}

	if e.cfg.CacheEmbeddings {
		_ = e.mem.SetJSON(ctx, cacheKey, vector)
	}

	return vector, nil
}

func (e *Engine) getQdrantResults(ctx context.Context, collection string, question string, vector []float64) ([]qdrant.SearchPoint, error) {
	if collection == "" {
		collection = e.cfg.QdrantCollection
	}

	e.logger.Printf("embedding dimension: %d (provider=%s model=%s)", len(vector), e.cfg.EmbedProvider, e.embedder.ModelName())

	if e.expectedVectorSize > 0 && len(vector) != e.expectedVectorSize {
		return nil, fmt.Errorf(
			"embedding dimension mismatch: Qdrant vector %q expects %d, but embedder %q returned %d",
			e.cfg.QdrantVectorName,
			e.expectedVectorSize,
			e.embedder.ModelName(),
			len(vector),
		)
	}

	cacheKey := "qdrant:" + collection + ":" + e.cfg.QdrantVectorName + ":" + e.cfg.EmbedProvider + ":" + e.embedder.ModelName() + ":" + memory.HashKey(question)

	if e.cfg.CacheQdrant {
		var cached []qdrant.SearchPoint
		found, err := e.mem.GetJSON(ctx, cacheKey, &cached)
		if err == nil && found && len(cached) > 0 {
			return cached, nil
		}
	}

	points, err := e.qdr.Query(ctx, collection, vector, e.cfg.QdrantTopK)
	if err != nil {
		return nil, err
	}

	if e.cfg.CacheQdrant {
		_ = e.mem.SetJSON(ctx, cacheKey, points)
	}

	return points, nil
}

func pointsToSources(points []qdrant.SearchPoint) []Source {
	sources := make([]Source, 0, len(points))

	for i, point := range points {
		payload := point.Payload

		text := firstString(payload,
			"text",
			"chunk",
			"content",
			"page_content",
			"body",
			"message",
			"summary",
			"claim",
		)

		title := firstString(payload,
			"title",
			"source",
			"file",
			"filename",
			"document",
			"url",
		)

		if title == "" {
			title = fmt.Sprintf("Qdrant point %v", point.ID)
		}

		sources = append(sources, Source{
			Index:  i + 1,
			Score:  point.Score,
			Title:  title,
			Page:   valueString(payload["page"]),
			Chunk:  valueString(payload["chunk_id"]),
			Source: firstString(payload, "source", "filename", "file", "url"),
			Text:   text,
		})
	}

	return sources
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}

		result := valueString(value)
		if strings.TrimSpace(result) != "" {
			return strings.TrimSpace(result)
		}
	}

	return ""
}

func valueString(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func formatHistory(turns []memory.Turn) string {
	if len(turns) == 0 {
		return "No previous turns."
	}

	lines := make([]string, 0, len(turns))

	for _, turn := range turns {
		lines = append(lines, fmt.Sprintf("%s: %s", turn.Role, turn.Content))
	}

	return strings.Join(lines, "\n")
}
