package qdrant

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"archimind/internal/config"
)

func TestQueryUsesPointsQueryEndpoint(t *testing.T) {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections/demo/points/query" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got := body["using"]; got != "claims_vec" {
			t.Fatalf("using = %v, want claims_vec", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"points": []map[string]any{{"id": 1, "score": 0.9, "payload": map[string]any{"text": "ok"}}},
			},
		})
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	c := NewClient(config.Config{QdrantURL: ts.URL, QdrantVectorName: "claims_vec", QdrantTopK: 3})
	points, err := c.Query(context.Background(), "demo", "", []float64{0.1, 0.2}, 0)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("Query() len = %d, want 1", len(points))
	}
}

func TestQueryFallsBackToSearchEndpoint(t *testing.T) {
	t.Helper()

	queryCalls := 0
	searchCalls := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/demo/points/query":
			queryCalls++
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"status":"error","result":"Not found"}`))
		case "/collections/demo/points/search":
			searchCalls++

			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			var body struct {
				Vector map[string]any `json:"vector"`
			}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("unmarshal body: %v", err)
			}
			if got := body.Vector["name"]; got != "claims_vec" {
				t.Fatalf("vector.name = %v, want claims_vec", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{{"id": "a", "score": 0.8, "payload": map[string]any{"text": "legacy"}}},
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	c := NewClient(config.Config{QdrantURL: ts.URL, QdrantTopK: 5})
	points, err := c.Query(context.Background(), "demo", "claims_vec", []float64{0.4, 0.5}, 2)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if queryCalls != 1 || searchCalls != 1 {
		t.Fatalf("queryCalls=%d searchCalls=%d, want 1/1", queryCalls, searchCalls)
	}
	if len(points) != 1 {
		t.Fatalf("Query() len = %d, want 1", len(points))
	}
}
