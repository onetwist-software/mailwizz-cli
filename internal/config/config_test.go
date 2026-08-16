package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/onetwist-software/mailwizz-cli/internal/config"
)

func TestLoadWithoutExistingFileReturnsZeroValue(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.APIURL != "" || cfg.APIKey != "" {
		t.Errorf("expected zero-value config, got %+v", cfg)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := config.Save(config.Config{APIURL: "https://a.example/api", APIKey: "key-1"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.APIURL != "https://a.example/api" || cfg.APIKey != "key-1" {
		t.Errorf("got %+v, want APIURL=https://a.example/api APIKey=key-1", cfg)
	}
}

func TestSaveMergesRatherThanOverwritesUnsetFields(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := config.Save(config.Config{APIURL: "https://a.example/api", APIKey: "key-1"}); err != nil {
		t.Fatalf("Save #1: %v", err)
	}

	if _, err := config.Save(config.Config{APIKey: "key-2"}); err != nil {
		t.Fatalf("Save #2: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.APIURL != "https://a.example/api" {
		t.Errorf("APIURL = %q, want it preserved from the first save", cfg.APIURL)
	}

	if cfg.APIKey != "key-2" {
		t.Errorf("APIKey = %q, want key-2", cfg.APIKey)
	}
}

func TestSaveWritesOwnerOnlyPermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := config.Save(config.Config{APIKey: "key-1"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	path, err := config.Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("permissions = %o, want 0600", perm)
	}
}

func TestPathJoinsHomeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := config.Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}

	if want := filepath.Join(home, config.FileName); path != want {
		t.Errorf("Path() = %q, want %q", path, want)
	}
}

func TestResolveEnvVarsOverrideFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := config.Save(config.Config{APIURL: "https://file.example/api", APIKey: "file-key"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	t.Setenv(config.EnvAPIURL, "https://env.example/api")
	t.Setenv(config.EnvAPIKey, "env-key")

	cfg, err := config.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if cfg.APIURL != "https://env.example/api" {
		t.Errorf("APIURL = %q, want the env var value", cfg.APIURL)
	}

	if cfg.APIKey != "env-key" {
		t.Errorf("APIKey = %q, want the env var value", cfg.APIKey)
	}
}

func TestResolveFallsBackToFileWhenEnvVarsAreUnset(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(config.EnvAPIURL, "")
	t.Setenv(config.EnvAPIKey, "")

	if _, err := config.Save(config.Config{APIURL: "https://file.example/api", APIKey: "file-key"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg, err := config.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if cfg.APIURL != "https://file.example/api" || cfg.APIKey != "file-key" {
		t.Errorf("got %+v, want file values preserved", cfg)
	}
}

func TestMaskAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "empty", key: "", want: ""},
		{name: "shorter than visible prefix", key: "ab", want: "****"},
		{name: "exact visible length", key: "abcd", want: "****"},
		{name: "typical key", key: "abcd1234efgh5678", want: "abcd****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := config.MaskAPIKey(tt.key); got != tt.want {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}
