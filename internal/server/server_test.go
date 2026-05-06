package server

import (
	"errors"
	"testing"
)

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
