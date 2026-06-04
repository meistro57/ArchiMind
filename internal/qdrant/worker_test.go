package qdrant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"archimind/internal/config"
)

func TestWorkerPoolQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections/demo/points/query" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"points": []map[string]any{{"id": 7, "score": 0.77, "payload": map[string]any{"text": "worker"}}},
			},
		})
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	client := NewClient(config.Config{QdrantURL: ts.URL, QdrantTopK: 3})
	pool := NewWorkerPool(client, 2, 8)
	defer pool.Close()

	points, err := pool.Query(context.Background(), "demo", "", []float64{0.2, 0.3}, 1)
	if err != nil {
		t.Fatalf("pool.Query() error = %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("pool.Query() len = %d, want 1", len(points))
	}
}

func TestWorkerPoolClose(t *testing.T) {
	client := NewClient(config.Config{QdrantURL: "http://localhost:65535", QdrantTopK: 3})
	pool := NewWorkerPool(client, 1, 1)
	pool.Close()

	_, err := pool.Query(context.Background(), "demo", "", []float64{0.1}, 1)
	if !errors.Is(err, ErrWorkerPoolClosed) {
		t.Fatalf("pool.Query() err = %v, want %v", err, ErrWorkerPoolClosed)
	}
}
