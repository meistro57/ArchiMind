// internal/config/config_test.go
package config

import "testing"

func TestGetEnvInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	if got := getEnvInt("TEST_INT", 1); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	t.Setenv("TEST_INT", "not-a-number")
	if got := getEnvInt("TEST_INT", 7); got != 7 {
		t.Fatalf("expected fallback 7, got %d", got)
	}
}

func TestGetEnvBool(t *testing.T) {
	cases := []struct {
		value    string
		expected bool
	}{
		{value: "true", expected: true},
		{value: "1", expected: true},
		{value: "yes", expected: true},
		{value: "y", expected: true},
		{value: "false", expected: false},
	}

	for _, tc := range cases {
		t.Setenv("TEST_BOOL", tc.value)
		if got := getEnvBool("TEST_BOOL", false); got != tc.expected {
			t.Fatalf("value %q expected %t, got %t", tc.value, tc.expected, got)
		}
	}
}

func TestLoadNormalizesStrictness(t *testing.T) {
	t.Setenv("ARCHIMIND_STRICTNESS", "invalid")
	cfg := Load()
	if cfg.Strictness != "balanced" {
		t.Fatalf("expected balanced strictness, got %q", cfg.Strictness)
	}
}
