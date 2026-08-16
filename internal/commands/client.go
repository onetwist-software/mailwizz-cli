package commands

import (
	"errors"
	"fmt"

	"github.com/onetwist-software/mailwizz-cli/internal/apiclient"
	"github.com/onetwist-software/mailwizz-cli/internal/config"
	"github.com/onetwist-software/mailwizz-cli/internal/generated/api"
)

// ErrNotConfigured is returned by ResolveClient when neither the config
// file nor the environment variables provide both an API URL and an API
// key.
var ErrNotConfigured = errors.New(
	"mailwizz-cli is not configured; run `mailwizz-cli config set --api-url <url> --api-key <key>`",
)

// ResolveClient builds an API client from the resolved mailwizz-cli
// configuration (~/.mailwizz-cli.json, overridden by MAILWIZZ_API_URL and
// MAILWIZZ_API_KEY). Every generated command calls this at the start of
// its Action, so configuration is always read fresh for each invocation.
func ResolveClient() (*api.Client, error) {
	cfg, err := config.Resolve()
	if err != nil {
		return nil, fmt.Errorf("resolve config: %w", err)
	}

	if cfg.APIURL == "" || cfg.APIKey == "" {
		return nil, ErrNotConfigured
	}

	return api.NewClient(apiclient.New(cfg.APIURL, cfg.APIKey)), nil
}
