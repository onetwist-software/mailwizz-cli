// Command mailwizz-cli is a command line client for the MailWizz API.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/onetwist-software/mailwizz-cli/internal/commands"
	generatedcli "github.com/onetwist-software/mailwizz-cli/internal/generated/cli"
	"github.com/onetwist-software/mailwizz-cli/internal/output"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

// binaryName is the program name shown in --help and used as args[0] in
// tests.
const binaryName = "mailwizz-cli"

func main() {
	os.Exit(run(context.Background(), os.Args, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	root := &cli.Command{
		Name:    binaryName,
		Usage:   "Command line client for the MailWizz API",
		Version: version,
		Writer:  stdout,
		Commands: append(
			[]*cli.Command{commands.ConfigCommand()},
			generatedcli.Commands()...,
		),
	}

	err := root.Run(ctx, args)
	if err == nil {
		return 0
	}

	// output.Handle already printed the API's own JSON error body; avoid
	// printing a second, redundant error line for it.
	if !errors.Is(err, output.ErrRequestFailed) {
		// This is the final error-reporting step before the process exits;
		// if writing to stderr itself fails there is nowhere left to
		// report that failure to, so the write error is deliberately
		// discarded here rather than silently ignored elsewhere.
		_, _ = fmt.Fprintln(stderr, err)
	}

	return 1
}
