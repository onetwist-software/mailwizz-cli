// Package commands contains the hand-written mailwizz-cli commands that
// are not generated from the OpenAPI schema: the root application and the
// "config" command used to manage ~/.mailwizz-cli.json.
package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/onetwist-software/mailwizz-cli/internal/config"
)

// ErrMissingConfigValue is returned by "config set" when neither --api-url
// nor --api-key was passed.
var ErrMissingConfigValue = errors.New("at least one of --api-url or --api-key must be set")

// ConfigCommand returns the "config" command, which reads and writes the
// mailwizz-cli configuration file.
func ConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage the mailwizz-cli configuration file",
		Commands: []*cli.Command{
			configSetCommand(),
			configGetCommand(),
		},
	}
}

func configSetCommand() *cli.Command {
	return &cli.Command{
		Name:  "set",
		Usage: "Save the MailWizz API URL and/or API key",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "api-url",
				Usage: "Base URL of the MailWizz API, e.g. https://example.com/api",
			},
			&cli.StringFlag{
				Name:  "api-key",
				Usage: "MailWizz API key",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			apiURL := cmd.String("api-url")

			apiKey := cmd.String("api-key")
			if apiURL == "" && apiKey == "" {
				return ErrMissingConfigValue
			}

			if _, err := config.Save(config.Config{APIURL: apiURL, APIKey: apiKey}); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			path, err := config.Path()
			if err != nil {
				return fmt.Errorf("resolve config path: %w", err)
			}

			if _, err := fmt.Fprintf(cmd.Root().Writer, "Configuration saved to %s\n", path); err != nil {
				return fmt.Errorf("write output: %w", err)
			}

			return nil
		},
	}
}

func configGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Print the resolved configuration (API key is masked)",
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg, err := config.Resolve()
			if err != nil {
				return fmt.Errorf("resolve config: %w", err)
			}

			path, err := config.Path()
			if err != nil {
				return fmt.Errorf("resolve config path: %w", err)
			}

			output := fmt.Sprintf(
				"config file: %s\napi_url:     %s\napi_key:     %s\n",
				path, cfg.APIURL, config.MaskAPIKey(cfg.APIKey),
			)

			if _, err := fmt.Fprint(cmd.Root().Writer, output); err != nil {
				return fmt.Errorf("write output: %w", err)
			}

			return nil
		},
	}
}
