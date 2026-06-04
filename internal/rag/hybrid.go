// internal/rag/hybrid.go
//
// Hybrid retrieval: BM25 lexical search fused with dense vector results via
// Reciprocal Rank Fusion (RRF).
//
// Drop-in for ArchiMind — zero new dependencies. BM25 is implemented natively.
//
// Why hybrid?
//   Dense vectors are great at meaning and synonyms, but miss exact rare terms:
//   Sanskrit words, proper nouns, entity names, book-specific jargon (Ra Contact,
//   Sefer Yetzirah, etc.). BM25 nails those exact matches. RRF fuses both ranked
//   lists without any score normalization headaches.
//
// Usage (see rag.go):
//   corpus  := buildBM25Corpus(points)
//   lexRank := bm25Search(corpus, query, topK)
//   fused   := rrfFuse(denseRanking, lexRank, topK)

package rag

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"archimind/internal/qdrant"
)

// ── BM25 tuning knobs ────────────────────────────────────────────────────────

const (
	bm25K1 = 1.5  // term-frequency saturation  (1.2–2.0 typical)
	bm25B  = 0.75 // document-length normalization (0.75 standard)
	rrfK   = 60   // RRF constant — higher = flatter fusion curve
)

// ── BM25 corpus ──────────────────────────────────────────────────────────────

// bm25Doc holds the pre-computed data for one Qdrant point.
type bm25Doc struct {
	pointID any
	tokens  []string
	tf      map[string]int
}

// bm25Corpus holds the full indexed corpus for a single retrieval call.
type bm25Corpus struct {
	docs   []bm25Doc
	df     map[string]int // document frequency per token
	avgLen float64
}

// buildBM25Corpus indexes a slice of Qdrant points for BM25 scoring.
// It extracts the same text fields used by pointsToSources so the lexical
// index matches what the LLM actually sees.
func buildBM25Corpus(points []qdrant.SearchPoint) bm25Corpus {
	docs := make([]bm25Doc, 0, len(points))
	df := map[string]int{}

	totalTokens := 0

	for _, p := range points {
		raw := extractPointText(p)
		tokens := bm25Tokenize(raw)
		tf := map[string]int{}
		seen := map[string]bool{}
		for _, t := range tokens {
			tf[t]++
			if !seen[t] {
				df[t]++
				seen[t] = true
			}
		}
		totalTokens += len(tokens)
		docs = append(docs, bm25Doc{
			pointID: p.ID,
			tokens:  tokens,
			tf:      tf,
		})
	}

	avgLen := 0.0
	if len(docs) > 0 {
		avgLen = float64(totalTokens) / float64(len(docs))
	}

	return bm25Corpus{docs: docs, df: df, avgLen: avgLen}
}

// bm25Search scores every document in the corpus against the query and
// returns a ranked slice of (pointID → score) pairs limited to topK.
func bm25Search(corpus bm25Corpus, query string, topK int) []bm25Hit {
	queryTokens := bm25Tokenize(query)
	n := float64(len(corpus.docs))
	if n == 0 || len(queryTokens) == 0 {
		return nil
	}

	type scored struct {
		idx   int
		score float64
	}

	scores := make([]scored, len(corpus.docs))
	for i, doc := range corpus.docs {
		docLen := float64(len(doc.tokens))
		score := 0.0
		for _, qt := range queryTokens {
			tfVal := float64(doc.tf[qt])
			if tfVal == 0 {
				continue
			}
			dfVal := float64(corpus.df[qt])
			// IDF with smoothing to avoid log(0)
			idf := math.Log((n-dfVal+0.5)/(dfVal+0.5) + 1)
			// BM25 TF component
			tf := (tfVal * (bm25K1 + 1)) /
				(tfVal + bm25K1*(1-bm25B+bm25B*(docLen/corpus.avgLen)))
			score += idf * tf
		}
		scores[i] = scored{idx: i, score: score}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if topK > 0 && len(scores) > topK {
		scores = scores[:topK]
	}

	hits := make([]bm25Hit, 0, len(scores))
	for rank, s := range scores {
		if s.score <= 0 {
			break // remaining docs have no lexical match
		}
		hits = append(hits, bm25Hit{
			pointID: corpus.docs[s.idx].pointID,
			score:   s.score,
			rank:    rank + 1,
		})
	}
	return hits
}

// bm25Hit is a lightweight result from the BM25 pass.
type bm25Hit struct {
	pointID any
	score   float64
	rank    int
}

// ── RRF fusion ───────────────────────────────────────────────────────────────

// rrfFuse combines a dense-vector ranking ([]qdrant.SearchPoint in score order)
// with a BM25 lexical ranking and returns a re-ranked slice of SearchPoints.
//
// RRF formula:  RRF(d) = Σ  1 / (k + rank_i(d))
//
// Operating on ranks (not raw scores) makes dense and lexical contributions
// directly comparable — no normalization required.
func rrfFuse(densePoints []qdrant.SearchPoint, lexHits []bm25Hit, topK int) []qdrant.SearchPoint {
	if len(densePoints) == 0 {
		return densePoints
	}

	// Build lookup: pointID → dense rank (1-based)
	denseRank := map[any]int{}
	for i, p := range densePoints {
		denseRank[p.ID] = i + 1
	}

	// Build lookup: pointID → lexical rank (1-based)
	lexRank := map[any]int{}
	for _, h := range lexHits {
		lexRank[h.pointID] = h.rank
	}

	// Collect all unique point IDs (dense may contain points not in lexHits and vice-versa)
	seen := map[any]bool{}
	allIDs := make([]any, 0, len(densePoints)+len(lexHits))
	for _, p := range densePoints {
		if !seen[p.ID] {
			seen[p.ID] = true
			allIDs = append(allIDs, p.ID)
		}
	}
	for _, h := range lexHits {
		if !seen[h.pointID] {
			seen[h.pointID] = true
			allIDs = append(allIDs, h.pointID)
		}
	}

	// Penalise missing rank: one position beyond the last seen rank in each list
	densePenalty := len(densePoints) + 1
	lexPenalty := len(lexHits) + 1
	if lexPenalty < 2 {
		lexPenalty = densePenalty // no lex results → treat as pure dense
	}

	type fusedScore struct {
		id    any
		score float64
	}

	fused := make([]fusedScore, 0, len(allIDs))
	for _, id := range allIDs {
		dr, ok := denseRank[id]
		if !ok {
			dr = densePenalty
		}
		lr, ok := lexRank[id]
		if !ok {
			lr = lexPenalty
		}
		score := 1.0/float64(rrfK+dr) + 1.0/float64(rrfK+lr)
		fused = append(fused, fusedScore{id: id, score: score})
	}

	sort.Slice(fused, func(i, j int) bool {
		return fused[i].score > fused[j].score
	})

	if topK > 0 && len(fused) > topK {
		fused = fused[:topK]
	}

	// Rebuild point slice in fused order, preserving original payload.
	pointByID := map[any]qdrant.SearchPoint{}
	for _, p := range densePoints {
		pointByID[p.ID] = p
	}

	result := make([]qdrant.SearchPoint, 0, len(fused))
	for rank, fs := range fused {
		p, ok := pointByID[fs.id]
		if !ok {
			continue // point only in lex results — no payload available, skip
		}
		// Overwrite Score with normalised RRF rank score so downstream
		// logging and signal analysis reflect the fused order.
		p.Score = 1.0 / float64(rank+1) // simple 1/rank for display
		result = append(result, p)
	}

	return result
}

// ── Text helpers ─────────────────────────────────────────────────────────────

// extractPointText pulls the best available text from a point's payload.
// Mirrors the priority in pointsToSources so the lexical index is consistent.
func extractPointText(p qdrant.SearchPoint) string {
	keys := []string{"text", "chunk", "content", "page_content", "body", "message", "summary", "claim"}
	for _, k := range keys {
		if v, ok := p.Payload[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

// bm25Tokenize lowercases and splits text into alpha-numeric tokens,
// filtering stop words and very short tokens.
func bm25Tokenize(text string) []string {
	lower := strings.ToLower(text)
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, lower)

	fields := strings.Fields(cleaned)
	tokens := make([]string, 0, len(fields))
	for _, tok := range fields {
		if len(tok) < 3 {
			continue
		}
		if _, blocked := bm25Stopwords[tok]; blocked {
			continue
		}
		tokens = append(tokens, tok)
	}
	return tokens
}

// bm25Stopwords — common English function words that carry no retrieval signal.
// Keep this lean; over-filtering hurts recall on esoteric corpora.
var bm25Stopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "are": {}, "but": {}, "not": {}, "you": {},
	"all": {}, "can": {}, "her": {}, "was": {}, "one": {}, "our": {}, "out": {},
	"day": {}, "get": {}, "has": {}, "him": {}, "his": {}, "how": {}, "man": {},
	"new": {}, "now": {}, "old": {}, "see": {}, "two": {}, "way": {}, "who": {},
	"did": {}, "its": {}, "let": {}, "may": {}, "put": {}, "say": {}, "she": {},
	"too": {}, "use": {}, "with": {}, "that": {}, "this": {}, "they": {}, "from": {},
	"have": {}, "been": {}, "were": {}, "said": {}, "each": {}, "which": {}, "their": {},
	"will": {}, "what": {}, "there": {}, "when": {}, "about": {}, "would": {},
}
