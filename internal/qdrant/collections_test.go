package qdrant

import "testing"

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
