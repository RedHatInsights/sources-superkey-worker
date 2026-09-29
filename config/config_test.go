package config

import (
	"os"
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
			os.Setenv("SOURCES_SCHEME", tt.envValue)
			defer os.Unsetenv("SOURCES_SCHEME")

			// Set required env vars to avoid side effects.
			os.Setenv("SOURCES_HOST", "localhost")
			os.Setenv("SOURCES_PORT", "8000")
			os.Setenv("SOURCES_REQUEST_MAX_ATTEMPTS", "1")
			defer os.Unsetenv("SOURCES_HOST")
			defer os.Unsetenv("SOURCES_PORT")
			defer os.Unsetenv("SOURCES_REQUEST_MAX_ATTEMPTS")

			cfg := Get()

			if cfg.SourcesScheme != tt.expected {
				t.Errorf("expected SourcesScheme=%q, got %q", tt.expected, cfg.SourcesScheme)
			}
		})
	}
}
