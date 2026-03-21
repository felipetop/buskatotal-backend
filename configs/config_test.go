package configs

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Unset any environment variables that could interfere with defaults.
	envKeys := []string{"PORT", "AUTH_MODE", "AUTH_HEADER", "APP_BASE_URL", "CATALOG_MARKUP", "USE_MOCK_DB"}
	saved := make(map[string]string)
	for _, k := range envKeys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.AuthMode != "jwt" {
		t.Errorf("AuthMode = %q, want %q", cfg.AuthMode, "jwt")
	}
	if cfg.AuthHeader != "X-User-Id" {
		t.Errorf("AuthHeader = %q, want %q", cfg.AuthHeader, "X-User-Id")
	}
	if cfg.AppBaseURL != "http://localhost:8080" {
		t.Errorf("AppBaseURL = %q, want %q", cfg.AppBaseURL, "http://localhost:8080")
	}
	if cfg.UseMockDB != false {
		t.Errorf("UseMockDB = %v, want false", cfg.UseMockDB)
	}
	if cfg.CatalogMarkup != 2.0 {
		t.Errorf("CatalogMarkup = %f, want 2.0", cfg.CatalogMarkup)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("PORT")

	cfg := Load()
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"y", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"", false},
		{"anything", false},
	}

	for _, tc := range tests {
		got := parseBool(tc.input)
		if got != tc.want {
			t.Errorf("parseBool(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"2.5", 2.5},
		{"10.0", 10.0},
		{"1.0", 1.0},
		// invalid strings fall back to 3.0
		{"invalid", 3.0},
		{"", 3.0},
		// zero and negative fall back to 3.0
		{"0", 3.0},
		{"-1.5", 3.0},
	}

	for _, tc := range tests {
		got := parseFloat(tc.input)
		if got != tc.want {
			t.Errorf("parseFloat(%q) = %f, want %f", tc.input, got, tc.want)
		}
	}
}
