package config

import (
	"testing"
)

func TestGet_NormalizesSourcesScheme(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{"lowercase http", "http", "http"},
		{"lowercase https", "https", "https"},
		{"uppercase HTTP", "HTTP", "http"},
		{"uppercase HTTPS", "HTTPS", "https"},
		{"mixed case Https", "Https", "https"},
		{"with whitespace", "  https  ", "https"},
		{"empty defaults to http", "", "http"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SOURCES_SCHEME", tt.envValue)

			// Set required env vars to avoid side effects.
			t.Setenv("SOURCES_HOST", "localhost")
			t.Setenv("SOURCES_PORT", "8000")
			t.Setenv("SOURCES_REQUEST_MAX_ATTEMPTS", "1")

			cfg := Get()

			if cfg.SourcesScheme != tt.expected {
				t.Errorf("expected SourcesScheme=%q, got %q", tt.expected, cfg.SourcesScheme)
			}
		})
	}
}
