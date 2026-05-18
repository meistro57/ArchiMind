package qdrant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"archimind/internal/config"
)

func TestParseVectorsNamed(t *testing.T) {
	raw := map[string]any{
		"result": map[string]any{
			"config": map[string]any{
				"params": map[string]any{
					"vectors": map[string]any{
						"claims_vec":  map[string]any{"size": 1536, "distance": "Cosine"},
						"summary_vec": map[string]any{"size": 1024, "distance": "Dot"},
					},
				},
			},
		},
	}

	vectors := parseVectors(raw)
	if len(vectors) != 2 {
		t.Fatalf("parseVectors() len = %d, want 2", len(vectors))
	}
	if vectors["claims_vec"].Size != 1536 {
		t.Fatalf("claims_vec size = %d, want 1536", vectors["claims_vec"].Size)
	}
	if vectors["summary_vec"].Distance != "Dot" {
		t.Fatalf("summary_vec distance = %q, want %q", vectors["summary_vec"].Distance, "Dot")
	}
}

func TestParseVectorsSingle(t *testing.T) {
	raw := map[string]any{
		"result": map[string]any{
			"config": map[string]any{
				"params": map[string]any{
					"vectors": map[string]any{"size": 768, "distance": "Cosine"},
				},
			},
		},
	}

	vectors := parseVectors(raw)
	if len(vectors) != 1 {
		t.Fatalf("parseVectors() len = %d, want 1", len(vectors))
	}
	if vectors["default"].Size != 768 {
		t.Fatalf("default size = %d, want 768", vectors["default"].Size)
	}
}

func TestListCollectionsHandlesPagination(t *testing.T) {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collections" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		offset := r.URL.Query().Get("offset")
		limit := r.URL.Query().Get("limit")
		if limit != "" {
			t.Fatalf("unexpected limit query param %q", limit)
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)

		switch offset {
		case "":
			_ = enc.Encode(map[string]any{
				"result": map[string]any{
					"collections": []map[string]string{
						{"name": "col_a"},
						{"name": "col_b"},
					},
					"next_page_offset": "page_2",
				},
			})
		case "page_2":
			_ = enc.Encode(map[string]any{
				"result": map[string]any{
					"collections": []map[string]string{
						{"name": "col_c"},
					},
				},
			})
		default:
			t.Fatalf("unexpected offset %q", offset)
		}
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	c := NewClient(config.Config{QdrantURL: ts.URL})
	names, err := c.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections() error = %v", err)
	}

	got := strings.Join(names, ",")
	if got != "col_a,col_b,col_c" {
		t.Fatalf("ListCollections() = %q, want %q", got, "col_a,col_b,col_c")
	}
}

func TestListCollectionsHandlesTopLevelNextOffset(t *testing.T) {
	t.Helper()

	requestCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)

		switch requestCount {
		case 1:
			_ = enc.Encode(map[string]any{
				"result": map[string]any{
					"collections": []map[string]string{{"name": "one"}},
				},
				"next_page_offset": "token-2",
			})
		case 2:
			if got := r.URL.Query().Get("offset"); got != "token-2" {
				t.Fatalf("offset = %q, want %q", got, "token-2")
			}
			_ = enc.Encode(map[string]any{
				"result": map[string]any{
					"collections": []map[string]string{{"name": "two"}},
				},
			})
		default:
			t.Fatalf("unexpected request count %d", requestCount)
		}
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	c := NewClient(config.Config{QdrantURL: ts.URL})
	names, err := c.ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections() error = %v", err)
	}

	got := strings.Join(names, ",")
	if got != "one,two" {
		t.Fatalf("ListCollections() = %q, want %q", got, "one,two")
	}
}

func TestParseCollectionsOffset(t *testing.T) {
	tests := []struct {
		name  string
		in    any
		want  string
		valid bool
	}{
		{name: "string", in: "abc", want: "abc", valid: true},
		{name: "json number", in: json.Number("42"), want: "42", valid: true},
		{name: "float", in: 5.0, want: "5", valid: true},
		{name: "int", in: 8, want: "8", valid: true},
		{name: "nil", in: nil, want: "", valid: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseCollectionsOffset(tc.in)
			if ok != tc.valid {
				t.Fatalf("parseCollectionsOffset() ok = %v, want %v", ok, tc.valid)
			}
			if got != tc.want {
				t.Fatalf("parseCollectionsOffset() = %q, want %q", got, tc.want)
			}
		})
	}
}
