package embed

import "testing"

func TestExtractVector(t *testing.T) {
	direct := extractVector([]any{0.1, 0.2, 0.3})
	if len(direct) != 3 {
		t.Fatalf("extractVector direct len = %d, want 3", len(direct))
	}

	nested := extractVector(map[string]any{
		"embedding": map[string]any{
			"values": []any{1.5, 2.5},
		},
	})
	if len(nested) != 2 {
		t.Fatalf("extractVector nested len = %d, want 2", len(nested))
	}
}

func TestExtractEmbeddingFallback(t *testing.T) {
	parsed := openRouterEmbeddingResponse{}
	raw := []byte(`{"data":[{"embedding":{"values":[0.11,0.22,0.33]}}]}`)

	vector := extractEmbedding(parsed, raw)
	if len(vector) != 3 {
		t.Fatalf("extractEmbedding fallback len = %d, want 3", len(vector))
	}
}
