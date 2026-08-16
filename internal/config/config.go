// Package config manages mailwizz-cli's persisted configuration file,
// stored by default at ~/.mailwizz-cli.json, and its environment variable
// overrides.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the name of the config file inside the user's home
// directory.
const FileName = ".mailwizz-cli.json"

// filePerm restricts the config file to the owner only, since it stores
// an API key.
const filePerm = 0o600

// Environment variable names that override values stored in the config
// file.
const (
	EnvAPIURL = "MAILWIZZ_API_URL"
	EnvAPIKey = "MAILWIZZ_API_KEY" //nolint:gosec // this is a variable name, not a credential
)

// Config holds the settings needed to talk to a MailWizz instance.
type Config struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
}

// Path returns the absolute path to the config file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(home, FileName), nil
}

// Load reads the config file. A missing file is not an error; it yields
// a zero-value Config so that first-time users see empty fields rather
// than a failure.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is derived from the user's own home directory
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}

	return &cfg, nil
}

// Save merges non-empty fields of updates into the config currently on
// disk and writes the result back with 0600 permissions, since it holds
// an API key.
func Save(updates Config) (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	if updates.APIURL != "" {
		cfg.APIURL = updates.APIURL
	}

	if updates.APIKey != "" {
		cfg.APIKey = updates.APIKey
	}

	path, err := Path()
	if err != nil {
		return nil, err
	}

	// The config file intentionally stores the user's own API key; that is
	// the whole point of `mailwizz-cli config set`.
	data, err := json.MarshalIndent(cfg, "", "  ") //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, filePerm); err != nil {
		return nil, fmt.Errorf("write config file %s: %w", path, err)
	}

	return cfg, nil
}

// Resolve loads the config file and applies environment variable
// overrides (MAILWIZZ_API_URL, MAILWIZZ_API_KEY), which take precedence
// over values stored on disk.
func Resolve() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	if v := os.Getenv(EnvAPIURL); v != "" {
		cfg.APIURL = v
	}

	if v := os.Getenv(EnvAPIKey); v != "" {
		cfg.APIKey = v
	}

	return cfg, nil
}

// MaskAPIKey returns a redacted form of an API key suitable for display,
// keeping only enough characters to help a user recognize which key is
// configured.
func MaskAPIKey(key string) string {
	const visible = 4

	if key == "" {
		return ""
	}

	if len(key) <= visible {
		return "****"
	}

	return key[:visible] + "****"
}
