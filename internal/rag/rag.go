// internal/rag/rag.go
package rag

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"unicode"

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

type Theme struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type Contradiction struct {
	Topic       string `json:"topic"`
	Supporting  int    `json:"supporting"`
	Opposing    int    `json:"opposing"`
	MentionedIn int    `json:"mentioned_in"`
}

type AnswerMode string

const (
	AnswerModeNormal     AnswerMode = "normal"
	AnswerModeSkeptical  AnswerMode = "skeptical"
	AnswerModeSynthesis  AnswerMode = "synthesis"
	AnswerModeDiagnostic AnswerMode = "diagnostic"
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

type AnswerDiagnostics struct {
	GroundedClaims      int      `json:"grounded_claims"`
	SpeculativeClaims   int      `json:"speculative_claims"`
	UnsupportedClaims   int      `json:"unsupported_claims"`
	UnsupportedLeapRisk string   `json:"unsupported_leap_risk"`
	UnsupportedSignals  []string `json:"unsupported_signals,omitempty"`
	SelfAuditChecklist  []string `json:"self_audit_checklist"`
}

type SourceInfluence struct {
	Index     int      `json:"index"`
	Title     string   `json:"title"`
	Score     float64  `json:"score"`
	Influence float64  `json:"influence"`
	Reasons   []string `json:"reasons,omitempty"`
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

func (e *Engine) Ask(ctx context.Context, sessionID string, collection string, question string, requestedMode string) (string, []Source, []Theme, []Contradiction, []SourceInfluence, error) {
	_ = e.mem.SaveTurn(ctx, sessionID, "user", question, e.cfg.ChatHistoryTurns)

	vector, err := e.getEmbedding(ctx, question)
	if err != nil {
		return "", nil, nil, nil, nil, err
	}

	points, err := e.getQdrantResults(ctx, collection, question, vector)
	if err != nil {
		return "", nil, nil, nil, nil, err
	}

	if len(points) == 0 {
		answer := "I could not find anything relevant in that Qdrant collection."
		_ = e.mem.SaveTurn(ctx, sessionID, "assistant", answer, e.cfg.ChatHistoryTurns)
		return answer, nil, nil, nil, nil, nil
	}

	sources := pointsToSources(points)
	themes := ExtractRecurringThemes(sources, 5)
	contradictions := ExtractContradictions(sources, 3)
	sourceInfluence := RankSourceInfluence(sources, themes, contradictions, 5)
	signal := BuildSignal(question, sources, requestedMode)
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

	system := buildSystemPrompt(signal, themes, contradictions)

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
		return "", nil, nil, nil, nil, err
	}

	_ = e.mem.SaveTurn(ctx, sessionID, "assistant", answer, e.cfg.ChatHistoryTurns)

	return answer, sources, themes, contradictions, sourceInfluence, nil
}

func normalizeAnswerMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(AnswerModeNormal):
		return string(AnswerModeNormal)
	case string(AnswerModeSkeptical):
		return string(AnswerModeSkeptical)
	case string(AnswerModeSynthesis):
		return string(AnswerModeSynthesis)
	case string(AnswerModeDiagnostic):
		return string(AnswerModeDiagnostic)
	default:
		return ""
	}
}

func InferAnswerMode(question string) AnswerMode {
	lower := strings.ToLower(strings.TrimSpace(question))

	diagnosticHints := []string{
		"diagnose", "diagnostic", "debug", "what went wrong", "root cause", "inspect", "analyze failure", "why did",
	}
	for _, hint := range diagnosticHints {
		if strings.Contains(lower, hint) {
			return AnswerModeDiagnostic
		}
	}

	skepticalHints := []string{
		"skeptical", "critique", "counterargument", "weakness", "challenge this", "doubt", "what might be wrong",
	}
	for _, hint := range skepticalHints {
		if strings.Contains(lower, hint) {
			return AnswerModeSkeptical
		}
	}

	synthesisHints := []string{
		"synthesize", "synthesis", "combine", "across", "framework", "themes", "compare", "unify", "big picture",
	}
	for _, hint := range synthesisHints {
		if strings.Contains(lower, hint) {
			return AnswerModeSynthesis
		}
	}

	return AnswerModeNormal
}

func BuildSignal(question string, sources []Source, requestedMode string) RetrievalSignal {
	mode := InferAnswerMode(question)
	normalizedMode := normalizeAnswerMode(requestedMode)
	if normalizedMode != "" {
		mode = AnswerMode(normalizedMode)
	}

	signal := RetrievalSignal{
		Mode:              mode,
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

func buildSystemPrompt(signal RetrievalSignal, themes []Theme, contradictions []Contradiction) string {
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
		"- Answer mode: normal. Give a clear, direct answer grounded in citations.",
	}
	if signal.Mode == AnswerModeSkeptical {
		modeRules = []string{
			"- Answer mode: skeptical. Stress-test claims, surface assumptions, and note weak support.",
			"- Distinguish between well-supported claims, questionable inferences, and missing evidence.",
		}
	}
	if signal.Mode == AnswerModeSynthesis {
		modeRules = []string{
			"- Answer mode: synthesis. Integrate recurring patterns across sources into a coherent structure.",
			"- Separate grounded synthesis from speculative bridging and label speculative parts clearly.",
		}
	}
	if signal.Mode == AnswerModeDiagnostic {
		modeRules = []string{
			"- Answer mode: diagnostic. Explain likely causes, constraints, and failure points grounded in evidence.",
			"- Provide a practical verification checklist and call out uncertainty when support is incomplete.",
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
	themeSummary := buildThemeSummary(themes)
	contradictionSummary := buildContradictionSummary(contradictions)

	lines := []string{
		"You are ArchiMind, a retrieval-grounded chat interface for Qdrant collections.",
		"",
		"Rules:",
		"- Use only the supplied Qdrant context for factual claims.",
		"- If the context does not support the answer, say that plainly.",
		"- Cite retrieved context with bracket citations like [1], [2].",
		"- Separate grounded evidence from speculation with explicit labels.",
		"- Flag possible unsupported leaps before concluding.",
		"- Be clear, practical, and useful.",
		"- Do not invent sources.",
	}
	lines = append(lines, strictnessRules...)
	lines = append(lines, modeRules...)
	lines = append(lines, clusterRules...)
	lines = append(lines, riskRules...)
	if themeSummary != "" {
		lines = append(lines, themeSummary)
	}
	if contradictionSummary != "" {
		lines = append(lines, contradictionSummary)
	}
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

func ExtractRecurringThemes(sources []Source, limit int) []Theme {
	if limit <= 0 {
		limit = 5
	}

	countByToken := map[string]int{}
	for _, src := range sources {
		tokens := tokenizeThemeText(src.Title + " " + src.Text)
		seen := map[string]struct{}{}
		for _, token := range tokens {
			if _, exists := seen[token]; exists {
				continue
			}
			seen[token] = struct{}{}
			countByToken[token]++
		}
	}

	themes := make([]Theme, 0, len(countByToken))
	for token, count := range countByToken {
		if count < 2 {
			continue
		}
		themes = append(themes, Theme{Label: token, Count: count})
	}

	sort.Slice(themes, func(i, j int) bool {
		if themes[i].Count == themes[j].Count {
			return themes[i].Label < themes[j].Label
		}
		return themes[i].Count > themes[j].Count
	})

	if len(themes) > limit {
		themes = themes[:limit]
	}

	return themes
}

func tokenizeThemeText(text string) []string {
	lower := strings.ToLower(text)
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, lower)

	fields := strings.Fields(cleaned)
	tokens := make([]string, 0, len(fields))
	for _, token := range fields {
		if len(token) < 4 {
			continue
		}
		if _, blocked := themeStopwords[token]; blocked {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func buildThemeSummary(themes []Theme) string {
	if len(themes) == 0 {
		return ""
	}
	parts := make([]string, 0, len(themes))
	for _, theme := range themes {
		parts = append(parts, fmt.Sprintf("%s(%d)", theme.Label, theme.Count))
	}
	return "- Recurring themes from retrieval: " + strings.Join(parts, ", ") + "."
}

func ExtractContradictions(sources []Source, limit int) []Contradiction {
	if limit <= 0 {
		limit = 3
	}

	type stanceCounter struct {
		supporting map[int]struct{}
		opposing   map[int]struct{}
		mentioned  map[int]struct{}
	}

	counts := map[string]*stanceCounter{}
	for _, src := range sources {
		sentences := splitSentences(src.Text)
		for _, sentence := range sentences {
			stance := classifySentenceStance(sentence)
			if stance == 0 {
				continue
			}
			tokens := tokenizeThemeText(sentence)
			topic := ""
			for _, token := range tokens {
				if _, blocked := contradictionStopwords[token]; blocked {
					continue
				}
				topic = token
				break
			}
			if topic == "" {
				continue
			}

			counter, exists := counts[topic]
			if !exists {
				counter = &stanceCounter{
					supporting: map[int]struct{}{},
					opposing:   map[int]struct{}{},
					mentioned:  map[int]struct{}{},
				}
				counts[topic] = counter
			}
			counter.mentioned[src.Index] = struct{}{}
			if stance > 0 {
				counter.supporting[src.Index] = struct{}{}
			} else {
				counter.opposing[src.Index] = struct{}{}
			}
		}
	}

	contradictions := make([]Contradiction, 0, len(counts))
	for topic, counter := range counts {
		supporting := len(counter.supporting)
		opposing := len(counter.opposing)
		if supporting == 0 || opposing == 0 {
			continue
		}
		contradictions = append(contradictions, Contradiction{
			Topic:       topic,
			Supporting:  supporting,
			Opposing:    opposing,
			MentionedIn: len(counter.mentioned),
		})
	}

	sort.Slice(contradictions, func(i, j int) bool {
		left := contradictions[i].Supporting + contradictions[i].Opposing
		right := contradictions[j].Supporting + contradictions[j].Opposing
		if left == right {
			return contradictions[i].Topic < contradictions[j].Topic
		}
		return left > right
	})

	if len(contradictions) > limit {
		contradictions = contradictions[:limit]
	}

	return contradictions
}

func splitSentences(text string) []string {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == '\n'
	})
	sentences := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		sentences = append(sentences, trimmed)
	}
	return sentences
}

func classifySentenceStance(sentence string) int {
	lower := strings.ToLower(strings.TrimSpace(sentence))
	if lower == "" {
		return 0
	}

	for _, phrase := range opposingPhrases {
		if strings.Contains(lower, phrase) {
			return -1
		}
	}
	for _, phrase := range supportingPhrases {
		if strings.Contains(lower, phrase) {
			return 1
		}
	}
	return 0
}

func buildContradictionSummary(contradictions []Contradiction) string {
	if len(contradictions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(contradictions))
	for _, contradiction := range contradictions {
		parts = append(parts, fmt.Sprintf("%s(+%d/-%d)", contradiction.Topic, contradiction.Supporting, contradiction.Opposing))
	}
	return "- Potential contradiction signals: " + strings.Join(parts, ", ") + "."
}

func AnalyzeAnswerDiscipline(answer string, sources []Source, signal RetrievalSignal) AnswerDiagnostics {
	diagnostics := AnswerDiagnostics{
		UnsupportedLeapRisk: "low",
		SelfAuditChecklist: []string{
			"Verify every factual claim has a nearby citation.",
			"Confirm speculative statements are explicitly labeled.",
			"Check whether conclusions merge conflicting sources.",
		},
	}

	sentences := splitSentences(answer)
	for _, sentence := range sentences {
		lower := strings.ToLower(strings.TrimSpace(sentence))
		if lower == "" {
			continue
		}

		hasCitation := citationPattern.MatchString(sentence)
		isSpeculative := containsAny(lower, speculativeMarkers)
		isAssertive := containsAny(lower, assertiveMarkers)

		if isSpeculative {
			diagnostics.SpeculativeClaims++
		}
		if hasCitation {
			diagnostics.GroundedClaims++
		}

		if !isSpeculative && !hasCitation && isAssertive {
			diagnostics.UnsupportedClaims++
			if len(diagnostics.UnsupportedSignals) < 5 {
				diagnostics.UnsupportedSignals = append(diagnostics.UnsupportedSignals, strings.TrimSpace(sentence))
			}
		}
	}

	riskScore := diagnostics.UnsupportedClaims
	if signal.HighRiskSynthesis {
		riskScore++
	}
	if signal.TopScore < 0.45 && len(sources) > 0 {
		riskScore++
	}

	switch {
	case riskScore >= 4:
		diagnostics.UnsupportedLeapRisk = "high"
	case riskScore >= 2:
		diagnostics.UnsupportedLeapRisk = "medium"
	default:
		diagnostics.UnsupportedLeapRisk = "low"
	}

	return diagnostics
}

func RankSourceInfluence(sources []Source, themes []Theme, contradictions []Contradiction, limit int) []SourceInfluence {
	if limit <= 0 {
		limit = 5
	}

	themeTokens := map[string]struct{}{}
	for _, theme := range themes {
		themeTokens[strings.ToLower(strings.TrimSpace(theme.Label))] = struct{}{}
	}
	contradictionTokens := map[string]struct{}{}
	for _, contradiction := range contradictions {
		contradictionTokens[strings.ToLower(strings.TrimSpace(contradiction.Topic))] = struct{}{}
	}

	ranked := make([]SourceInfluence, 0, len(sources))
	for _, src := range sources {
		tokens := tokenizeThemeText(src.Title + " " + src.Text)
		themeHits := 0
		contradictionHits := 0
		for _, token := range tokens {
			if _, ok := themeTokens[token]; ok {
				themeHits++
			}
			if _, ok := contradictionTokens[token]; ok {
				contradictionHits++
			}
		}

		influence := src.Score + (float64(themeHits) * 0.05) + (float64(contradictionHits) * 0.07)
		reasons := []string{}
		if themeHits > 0 {
			reasons = append(reasons, fmt.Sprintf("theme overlap x%d", themeHits))
		}
		if contradictionHits > 0 {
			reasons = append(reasons, fmt.Sprintf("contradiction overlap x%d", contradictionHits))
		}
		if len(reasons) == 0 {
			reasons = append(reasons, "base retrieval score")
		}

		ranked = append(ranked, SourceInfluence{
			Index:     src.Index,
			Title:     src.Title,
			Score:     src.Score,
			Influence: influence,
			Reasons:   reasons,
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Influence == ranked[j].Influence {
			return ranked[i].Index < ranked[j].Index
		}
		return ranked[i].Influence > ranked[j].Influence
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}

var themeStopwords = map[string]struct{}{
	"about": {}, "after": {}, "again": {}, "also": {}, "been": {}, "being": {}, "between": {},
	"could": {}, "does": {}, "each": {}, "from": {}, "have": {}, "into": {}, "just": {}, "like": {},
	"many": {}, "more": {}, "most": {}, "other": {}, "over": {}, "same": {}, "some": {}, "such": {},
	"than": {}, "that": {}, "their": {}, "there": {}, "these": {}, "they": {}, "this": {}, "those": {},
	"very": {}, "what": {}, "when": {}, "where": {}, "which": {}, "while": {}, "with": {}, "would": {},
	"your": {}, "you": {}, "using": {}, "used": {}, "should": {}, "must": {}, "will": {},
}

var contradictionStopwords = map[string]struct{}{
	"should": {}, "avoid": {}, "never": {}, "always": {}, "cannot": {}, "can't": {}, "not": {},
	"might": {}, "could": {}, "would": {}, "maybe": {}, "likely": {}, "unlikely": {},
}

var supportingPhrases = []string{
	"should", "recommended", "works", "effective", "improves", "benefit", "helps", "success", "best",
}

var opposingPhrases = []string{
	"should not", "not recommended", "avoid", "does not", "doesn't", "fails", "failure", "risk", "harm", "worse",
}

var citationPattern = regexp.MustCompile(`\[[0-9]+\]`)

var speculativeMarkers = []string{
	"might", "could", "may", "possibly", "likely", "unlikely", "hypothesis", "speculative", "guess",
}

var assertiveMarkers = []string{
	"is", "are", "will", "must", "always", "never", "proves", "shows", "demonstrates", "therefore",
}

func containsAny(text string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
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
