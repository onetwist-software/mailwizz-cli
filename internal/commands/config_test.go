package commands_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/onetwist-software/mailwizz-cli/internal/commands"
	"github.com/onetwist-software/mailwizz-cli/internal/config"
)

// failingWriter always returns an error, so tests can verify that a
// failure writing command output is surfaced as an error rather than
// silently discarded.
type failingWriter struct{}

var errWriteFailed = errors.New("write failed")

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, errWriteFailed
}

func newTestApp(out *bytes.Buffer) *cli.Command {
	return &cli.Command{
		Name:     "mailwizz-cli",
		Writer:   out,
		Commands: []*cli.Command{commands.ConfigCommand()},
	}
}

func TestConfigSetRequiresAValue(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var out bytes.Buffer

	app := newTestApp(&out)

	err := app.Run(context.Background(), []string{"mailwizz-cli", "config", "set"})
	if err == nil {
		t.Fatal("expected an error when neither --api-url nor --api-key is set")
	}
}

func TestConfigSetAndGet(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(config.EnvAPIURL, "")
	t.Setenv(config.EnvAPIKey, "")

	var setOut bytes.Buffer

	setApp := newTestApp(&setOut)

	args := []string{"mailwizz-cli", "config", "set", "--api-url", "https://example.com/api", "--api-key", "topsecret1234"}
	if err := setApp.Run(context.Background(), args); err != nil {
		t.Fatalf("config set: %v", err)
	}

	path, err := config.Path()
	if err != nil {
		t.Fatalf("config.Path: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist at %s: %v", path, err)
	}

	var getOut bytes.Buffer

	getApp := newTestApp(&getOut)
	if err := getApp.Run(context.Background(), []string{"mailwizz-cli", "config", "get"}); err != nil {
		t.Fatalf("config get: %v", err)
	}

	got := getOut.String()
	if !strings.Contains(got, "https://example.com/api") {
		t.Errorf("expected output to contain the api url, got %q", got)
	}

	if strings.Contains(got, "topsecret1234") {
		t.Errorf("expected api key to be masked, got %q", got)
	}

	if !strings.Contains(got, filepath.Base(path)) {
		t.Errorf("expected output to reference the config file path, got %q", got)
	}
}

func TestConfigSetSurfacesWriteErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	app := &cli.Command{
		Name:     "mailwizz-cli",
		Writer:   failingWriter{},
		Commands: []*cli.Command{commands.ConfigCommand()},
	}

	args := []string{"mailwizz-cli", "config", "set", "--api-url", "https://example.com/api"}

	err := app.Run(context.Background(), args)
	if !errors.Is(err, errWriteFailed) {
		t.Fatalf("expected the write failure to be returned as an error, got %v", err)
	}
}

func TestConfigGetSurfacesWriteErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, err := config.Save(config.Config{APIURL: "https://example.com/api"}); err != nil {
		t.Fatalf("config.Save: %v", err)
	}

	app := &cli.Command{
		Name:     "mailwizz-cli",
		Writer:   failingWriter{},
		Commands: []*cli.Command{commands.ConfigCommand()},
	}

	err := app.Run(context.Background(), []string{"mailwizz-cli", "config", "get"})
	if !errors.Is(err, errWriteFailed) {
		t.Fatalf("expected the write failure to be returned as an error, got %v", err)
	}
}
