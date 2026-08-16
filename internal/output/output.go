// Package output renders MailWizz API responses for the CLI and derives
// the process exit code from the HTTP status code the API returned.
package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/onetwist-software/mailwizz-cli/internal/apiclient"
)

// ErrRequestFailed is returned by Handle when the API responded with a
// non-2xx status. By the time it is returned, the response body has
// already been printed, so callers should exit(1) without printing this
// error's message again.
var ErrRequestFailed = errors.New("mailwizz api request failed")

// Handle prints resp to w (see Print) and turns a non-2xx status into
// ErrRequestFailed, so a generated command's Action can simply:
//
//	return output.Handle(cmd.Root().Writer, resp)
func Handle(w io.Writer, resp *apiclient.Response) error {
	code, err := Print(w, resp)
	if err != nil {
		return err
	}

	if code != 0 {
		return ErrRequestFailed
	}

	return nil
}

// Print writes resp.Body to w as indented JSON. If the body is not valid
// JSON it is written verbatim rather than dropped, so unexpected API
// responses are still visible to the caller.
//
// It returns the process exit code that the command should use: 0 for a
// 2xx status, 1 otherwise. Most callers should use Handle instead, which
// turns that exit code into an error.
func Print(w io.Writer, resp *apiclient.Response) (int, error) {
	if err := writeJSON(w, resp.Body); err != nil {
		return 1, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 1, nil
	}

	return 0, nil
}

func writeJSON(w io.Writer, body []byte) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		if _, err := fmt.Fprintln(w, "{}"); err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		return nil
	}

	var indented bytes.Buffer
	if err := json.Indent(&indented, trimmed, "", "  "); err != nil {
		if _, ferr := fmt.Fprintln(w, string(trimmed)); ferr != nil {
			return fmt.Errorf("write output: %w", ferr)
		}

		return nil
	}

	if _, err := fmt.Fprintln(w, indented.String()); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}
