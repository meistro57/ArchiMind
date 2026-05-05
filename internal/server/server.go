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
	"archimind/internal/qdrant"
	"archimind/internal/rag"
)

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
}

type ChatResponse struct {
	Answer  string       `json:"answer"`
	Sources []rag.Source `json:"sources"`
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

	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	if req.SessionID == "" {
		req.SessionID = "default"
	}

	answer, sources, err := s.rag.Ask(r.Context(), req.SessionID, req.Collection, req.Message)
	if err != nil {
		s.logger.Printf("chat error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	signal := rag.BuildSignal(req.Message, sources)
	signal.Strictness = cfgStrictness(s.cfg.Strictness)
	s.logger.Printf("chat debug mode=%s high_risk=%t cluster=%s strictness=%s top_score=%.4f score_gap=%.4f spread=%.4f", signal.Mode, signal.HighRiskSynthesis, signal.Cluster, signal.Strictness, signal.TopScore, signal.ScoreGap, signal.SimilaritySpread)

	writeJSON(w, ChatResponse{
		Answer:  answer,
		Sources: sources,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"status": "ok",
		"app":    "ArchiMind",
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

func withBasicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
