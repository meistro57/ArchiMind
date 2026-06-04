package server

import (
	"errors"
	"testing"
)

func TestSupportsTextCompletion(t *testing.T) {
	tests := []struct {
		name     string
		modality string
		want     bool
	}{
		{name: "text to text", modality: "text->text", want: true},
		{name: "multimodal to text", modality: "image+text->text", want: true},
		{name: "embedding only", modality: "text->embedding", want: false},
		{name: "unknown text", modality: "text", want: true},
		{name: "empty defaults to include", modality: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := supportsTextCompletion(tt.modality); got != tt.want {
				t.Fatalf("supportsTextCompletion(%q) = %t, want %t", tt.modality, got, tt.want)
			}
		})
	}
}

func TestEnsureModelOption(t *testing.T) {
	models := []modelOption{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}
	withExisting := ensureModelOption(models, "a")
	if len(withExisting) != 2 {
		t.Fatalf("expected unchanged length, got %d", len(withExisting))
	}

	withMissing := ensureModelOption(models, "c")
	if len(withMissing) != 3 {
		t.Fatalf("expected prepended model, got %d", len(withMissing))
	}
	if withMissing[0].ID != "c" {
		t.Fatalf("expected active model to be prepended, got %q", withMissing[0].ID)
	}
}

func TestClassifyChatError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "dimension mismatch", err: errors.New("embedding dimension mismatch: expected 1536 got 1024"), code: "embedding_dimension_mismatch"},
		{name: "collection missing", err: errors.New("qdrant collection is missing"), code: "collection_missing"},
		{name: "vector missing", err: errors.New("vector \"claims_vec\" not found in collection \"docs\""), code: "vector_not_found"},
		{name: "http error", err: errors.New("qdrant returned HTTP 401: unauthorized"), code: "qdrant_http_error"},
		{name: "parse error", err: errors.New("could not parse qdrant response: bad json"), code: "qdrant_parse_error"},
		{name: "fallback", err: errors.New("something else failed"), code: "retrieval_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostic := classifyChatError(tt.err)
			if diagnostic.Code != tt.code {
				t.Fatalf("classifyChatError() code = %q, want %q", diagnostic.Code, tt.code)
			}
			if diagnostic.Error == "" {
				t.Fatal("classifyChatError() error should not be empty")
			}
			if diagnostic.Hint == "" {
				t.Fatal("classifyChatError() hint should not be empty")
			}
		})
	}
}
