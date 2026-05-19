// internal/server/server.go
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"archimind/internal/config"
	"archimind/internal/embed"
	"archimind/internal/llm"
	"archimind/internal/qdrant"
	"archimind/internal/rag"
	"archimind/internal/reporter"
)

const appVersion = "0.6.0"

type Server struct {
	cfg    config.Config
	rag    *rag.Engine
	qdr    *qdrant.Client
	logger *log.Logger
	http   *http.Server
}

type ChatRequest struct {
	SessionID  string `json:"session_id"`
	Message    string `json:"message"`
	Collection string `json:"collection,omitempty"`
	Mode       string `json:"mode,omitempty"`
}

type ChatResponse struct {
	Answer          string                `json:"answer"`
	Sources         []rag.Source          `json:"sources"`
	Themes          []rag.Theme           `json:"themes,omitempty"`
	Contradictions  []rag.Contradiction   `json:"contradictions,omitempty"`
	SourceInfluence []rag.SourceInfluence `json:"source_influence,omitempty"`
	StrongClaims    []rag.StrongClaim     `json:"strong_claims,omitempty"`
	Diagnostics     rag.AnswerDiagnostics `json:"diagnostics"`
}

type CompareRequest struct {
	SessionID       string `json:"session_id"`
	Message         string `json:"message"`
	LeftCollection  string `json:"left_collection"`
	RightCollection string `json:"right_collection"`
	Mode            string `json:"mode,omitempty"`
}

type CompareResponse struct {
	Answer string                 `json:"answer"`
	Left   rag.CollectionInsights `json:"left"`
	Right  rag.CollectionInsights `json:"right"`
}

type FrameworkRequest struct {
	SessionID  string `json:"session_id"`
	Message    string `json:"message"`
	Collection string `json:"collection,omitempty"`
}

type FrameworkResponse struct {
	Topic           string                   `json:"topic"`
	Collection      string                   `json:"collection"`
	Summary         string                   `json:"summary"`
	Components      []rag.FrameworkComponent `json:"components"`
	Themes          []rag.Theme              `json:"themes,omitempty"`
	Contradictions  []rag.Contradiction      `json:"contradictions,omitempty"`
	SourceInfluence []rag.SourceInfluence    `json:"source_influence,omitempty"`
	StrongClaims    []rag.StrongClaim        `json:"strong_claims,omitempty"`
	Sources         []rag.Source             `json:"sources,omitempty"`
}

type SessionExportRequest struct {
	SessionID string `json:"session_id"`
}

type ReportStartRequest struct {
	Topic string `json:"topic"`
}

type ReportStartResponse struct {
	Message    string `json:"message"`
	OutputPath string `json:"output_path"`
}

func New(cfg config.Config, ragEngine *rag.Engine, qdrantClient *qdrant.Client, logger *log.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		rag:    ragEngine,
		qdr:    qdrantClient,
		logger: logger,
	}

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(filepath.Join(".", "web"))))
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/compare", s.handleCompare)
	mux.HandleFunc("/api/framework", s.handleFramework)
	mux.HandleFunc("/api/review/last", s.handleReviewLast)
	mux.HandleFunc("/api/export/markdown", s.handleExportMarkdown)
	mux.HandleFunc("/api/export/json", s.handleExportJSON)
	mux.HandleFunc("/api/report", s.handleReport)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/collection", s.handleCollection)

	s.http = &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.AppPort),
		Handler:           withBasicMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	err := s.http.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ChatRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Message = strings.TrimSpace(req.Message)
	req.Collection = strings.TrimSpace(req.Collection)
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Mode = strings.TrimSpace(req.Mode)

	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	if req.SessionID == "" {
		req.SessionID = "default"
	}

	answer, sources, themes, contradictions, sourceInfluence, err := s.rag.Ask(r.Context(), req.SessionID, req.Collection, req.Message, req.Mode)
	if err != nil {
		s.logger.Printf("chat error: %v", err)
		diag := classifyChatError(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": diag.Error,
			"code":  diag.Code,
			"hint":  diag.Hint,
		})
		return
	}

	signal := rag.BuildSignal(req.Message, sources, req.Mode)
	signal.Strictness = cfgStrictness(s.cfg.Strictness)
	s.logger.Printf("chat debug mode=%s high_risk=%t cluster=%s strictness=%s top_score=%.4f score_gap=%.4f spread=%.4f", signal.Mode, signal.HighRiskSynthesis, signal.Cluster, signal.Strictness, signal.TopScore, signal.ScoreGap, signal.SimilaritySpread)

	diagnostics := rag.AnalyzeAnswerDiscipline(answer, sources, signal)
	strongClaims := rag.RankStrongClaims(sources, themes, contradictions, 5)

	writeJSON(w, ChatResponse{
		Answer:          answer,
		Sources:         sources,
		Themes:          themes,
		Contradictions:  contradictions,
		SourceInfluence: sourceInfluence,
		StrongClaims:    strongClaims,
		Diagnostics:     diagnostics,
	})
}

func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CompareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Message = strings.TrimSpace(req.Message)
	req.LeftCollection = strings.TrimSpace(req.LeftCollection)
	req.RightCollection = strings.TrimSpace(req.RightCollection)
	req.Mode = strings.TrimSpace(req.Mode)

	if req.SessionID == "" {
		req.SessionID = "default"
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}
	if req.LeftCollection == "" || req.RightCollection == "" {
		writeError(w, http.StatusBadRequest, "left_collection and right_collection are required")
		return
	}

	result, err := s.rag.CompareCollections(r.Context(), req.SessionID, req.LeftCollection, req.RightCollection, req.Message, req.Mode)
	if err != nil {
		s.logger.Printf("compare error: %v", err)
		diag := classifyChatError(err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": diag.Error,
			"code":  diag.Code,
			"hint":  diag.Hint,
		})
		return
	}

	writeJSON(w, CompareResponse{
		Answer: result.Answer,
		Left:   result.Left,
		Right:  result.Right,
	})
}

func (s *Server) handleFramework(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req FrameworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.SessionID = strings.TrimSpace(req.SessionID)
	req.Message = strings.TrimSpace(req.Message)
	req.Collection = strings.TrimSpace(req.Collection)
	if req.SessionID == "" {
		req.SessionID = "default"
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	result, err := s.rag.ExtractFramework(r.Context(), req.SessionID, req.Collection, req.Message)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, FrameworkResponse{
		Topic:           result.Topic,
		Collection:      result.Collection,
		Summary:         result.Summary,
		Components:      result.Components,
		Themes:          result.Themes,
		Contradictions:  result.Contradictions,
		SourceInfluence: result.SourceInfluence,
		StrongClaims:    result.StrongClaims,
		Sources:         result.Sources,
	})
}

func (s *Server) handleReviewLast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SessionExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	report, err := s.rag.ReviewLastAnswer(r.Context(), strings.TrimSpace(req.SessionID))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, report)
}

func (s *Server) handleExportMarkdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SessionExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	markdown, err := s.rag.ExportSessionMarkdown(r.Context(), strings.TrimSpace(req.SessionID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=archimind_chat_export.md")
	_, _ = w.Write([]byte(markdown))
}

func (s *Server) handleExportJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SessionExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "default"
	}

	history, err := s.rag.SessionHistory(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=archimind_chat_export.json")
	writeJSON(w, map[string]any{
		"session_id": sessionID,
		"turns":      history,
	})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ReportStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Topic = strings.TrimSpace(req.Topic)
	if req.Topic == "" {
		writeError(w, http.StatusBadRequest, "topic is required")
		return
	}

	timestamp := time.Now().UTC().Format("20060102_150405")
	outputPath := filepath.Join("reports", sanitizeReportTopic(req.Topic)+"_"+timestamp+".md")

	reportAgent := reporter.NewAgent(
		s.cfg,
		s.qdr,
		llm.NewOpenRouterProvider(s.cfg),
		buildReportEmbedder(s.cfg),
		s.logger,
	)

	go func(topic string, path string) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		err := reportAgent.Generate(ctx, reporter.ReportRequest{
			Topic:      topic,
			TokenLimit: 6000,
			OutputPath: path,
		})
		if err != nil {
			s.logger.Printf("report generation error topic=%q path=%s err=%v", topic, path, err)
			return
		}
		s.logger.Printf("report generation finished topic=%q path=%s", topic, path)
	}(req.Topic, outputPath)

	writeJSON(w, ReportStartResponse{
		Message:    "report generation started",
		OutputPath: outputPath,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"status":      "ok",
		"app":         "ArchiMind",
		"app_version": appVersion,
	})
}

func (s *Server) handleCollection(w http.ResponseWriter, r *http.Request) {
	collection := strings.TrimSpace(r.URL.Query().Get("name"))

	info, err := s.qdr.CollectionInfo(r.Context(), collection)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, info)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": message,
	})
}

func buildReportEmbedder(cfg config.Config) embed.Provider {
	if strings.ToLower(strings.TrimSpace(cfg.EmbedProvider)) == "openrouter" {
		return embed.NewOpenRouterProvider(cfg)
	}
	return embed.NewOllamaProvider(cfg)
}

func sanitizeReportTopic(topic string) string {
	trimmed := strings.ToLower(strings.TrimSpace(topic))
	if trimmed == "" {
		return "report"
	}

	slug := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '-' || r == '_':
			return '_'
		default:
			return -1
		}
	}, trimmed)

	slug = strings.Trim(slug, "_")
	if slug == "" {
		return "report"
	}
	return slug
}

func cfgStrictness(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return "strict"
	case "exploratory":
		return "exploratory"
	default:
		return "balanced"
	}
}

type chatErrorDiagnostic struct {
	Error string
	Code  string
	Hint  string
}

func classifyChatError(err error) chatErrorDiagnostic {
	message := strings.ToLower(strings.TrimSpace(err.Error()))

	diagnostic := chatErrorDiagnostic{
		Error: "retrieval request failed",
		Code:  "retrieval_failed",
		Hint:  "Check QDRANT_URL, QDRANT_COLLECTION, and provider settings.",
	}

	switch {
	case strings.Contains(message, "embedding dimension mismatch"):
		diagnostic.Error = "embedding and collection vector dimensions do not match"
		diagnostic.Code = "embedding_dimension_mismatch"
		diagnostic.Hint = "Verify QDRANT_VECTOR_NAME and selected embedding model dimensions."
	case strings.Contains(message, "qdrant collection is missing"):
		diagnostic.Error = "qdrant collection is missing"
		diagnostic.Code = "collection_missing"
		diagnostic.Hint = "Set QDRANT_COLLECTION or pass collection in /api/chat request."
	case strings.Contains(message, "vector") && strings.Contains(message, "not found in collection"):
		diagnostic.Error = "configured vector name was not found in collection"
		diagnostic.Code = "vector_not_found"
		diagnostic.Hint = "Update QDRANT_VECTOR_NAME to an existing vector in the target collection."
	case strings.Contains(message, "qdrant returned http") || strings.Contains(message, "qdrant collection info returned http"):
		diagnostic.Error = "qdrant request returned non-success status"
		diagnostic.Code = "qdrant_http_error"
		diagnostic.Hint = "Validate QDRANT_URL, QDRANT_API_KEY, and collection name permissions."
	case strings.Contains(message, "could not parse qdrant response"):
		diagnostic.Error = "qdrant response could not be parsed"
		diagnostic.Code = "qdrant_parse_error"
		diagnostic.Hint = "Inspect collection payload schema and ensure point payload fields are valid JSON."
	}

	return diagnostic
}

func withBasicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
