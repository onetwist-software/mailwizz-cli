package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// runCLI runs the root command with args (excluding the program name,
// which is added automatically) and returns the exit code and captured
// stdout/stderr.
func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer

	code := run(context.Background(), append([]string{binaryName}, args...), &outBuf, &errBuf)

	return code, outBuf.String(), errBuf.String()
}

func TestRunPrintsHelpAndSucceeds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	code, stdout, _ := runCLI(t, "--help")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	if !strings.Contains(stdout, "mailwizz-cli") {
		t.Errorf("help output missing program name: %q", stdout)
	}
}

func TestRunPrintsVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	code, _, _ := runCLI(t, "--version")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRunFailsWhenNotConfigured(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MAILWIZZ_API_URL", "")
	t.Setenv("MAILWIZZ_API_KEY", "")

	code, _, stderr := runCLI(t, "lists", "list")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	if !strings.Contains(stderr, "config set") {
		t.Errorf("stderr should point the user at `config set`, got %q", stderr)
	}
}

func TestConfigSetAndGetRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	code, stdout, _ := runCLI(t, "config", "set", "--api-url", "https://example.com/api", "--api-key", "abc123")
	if code != 0 {
		t.Fatalf("config set: exit code = %d, want 0, stdout=%q", code, stdout)
	}

	code, stdout, _ = runCLI(t, "config", "get")
	if code != 0 {
		t.Fatalf("config get: exit code = %d, want 0", code)
	}

	if !strings.Contains(stdout, "https://example.com/api") {
		t.Errorf("config get output missing api url: %q", stdout)
	}

	if strings.Contains(stdout, "abc123") {
		t.Errorf("config get should mask the api key, got %q", stdout)
	}
}

// newMailWizzMock starts an httptest server that mimics just enough of
// the real MailWizz API for the representative commands exercised below,
// and points the CLI's configuration at it.
func newMailWizzMock(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	t.Setenv("HOME", t.TempDir())
	t.Setenv("MAILWIZZ_API_URL", srv.URL)
	t.Setenv("MAILWIZZ_API_KEY", "test-key")
}

//nolint:paralleltest // uses t.Setenv, which is not safe to combine with t.Parallel
func TestListsCRUD(t *testing.T) {
	newMailWizzMock(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Api-Key"); got != "test-key" {
			t.Errorf("api key header = %q, want test-key", got)
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/lists":
			_, _ = w.Write([]byte(`{"status":"success","data":{"records":[]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/lists":
			if err := r.ParseMultipartForm(1 << 20); err != nil { //nolint:gosec // test fixture, bounded literal
				t.Fatalf("ParseMultipartForm: %v", err)
			}

			if got := r.FormValue("general[name]"); got != "My List" {
				t.Errorf("general[name] = %q, want %q", got, "My List")
			}

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"success","list_uid":"abc123"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/lists/abc123":
			_, _ = w.Write([]byte(`{"status":"success","data":{"list_uid":"abc123"}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/lists/abc123":
			if err := r.ParseMultipartForm(1 << 20); err != nil { //nolint:gosec // test fixture, bounded literal
				t.Fatalf("ParseMultipartForm: %v", err)
			}

			_, _ = w.Write([]byte(`{"status":"success"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/lists/abc123":
			_, _ = w.Write([]byte(`{"status":"success"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	if code, _, _ := runCLI(t, "lists", "list"); code != 0 {
		t.Errorf("lists list: exit code = %d, want 0", code)
	}

	code, stdout, _ := runCLI(t, "lists", "create",
		"--general-name", "My List",
		"--general-description", "desc",
		"--defaults-from-name", "John",
		"--defaults-from-email", "john@example.com",
		"--defaults-reply-to", "john@example.com",
	)
	if code != 0 {
		t.Fatalf("lists create: exit code = %d, want 0, stdout=%q", code, stdout)
	}

	if !strings.Contains(stdout, "abc123") {
		t.Errorf("lists create output missing list_uid: %q", stdout)
	}

	if code, _, _ := runCLI(t, "lists", "view", "--list-uid", "abc123"); code != 0 {
		t.Errorf("lists view: exit code = %d, want 0", code)
	}

	if code, _, _ := runCLI(t, "lists", "update", "--list-uid", "abc123", "--general-name", "Renamed"); code != 0 {
		t.Errorf("lists update: exit code = %d, want 0", code)
	}

	if code, _, _ := runCLI(t, "lists", "delete", "--list-uid", "abc123"); code != 0 {
		t.Errorf("lists delete: exit code = %d, want 0", code)
	}
}

//nolint:paralleltest // uses t.Setenv, which is not safe to combine with t.Parallel
func TestListsSubscribersCreateSendsBracketedAndRequiredFields(t *testing.T) {
	newMailWizzMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/lists/list-1/subscribers" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil { //nolint:gosec // test fixture, bounded literal
			t.Fatalf("ParseMultipartForm: %v", err)
		}

		if got := r.FormValue("EMAIL"); got != "jane@example.com" {
			t.Errorf("EMAIL = %q, want jane@example.com", got)
		}

		if got := r.FormValue("details[status]"); got != "confirmed" {
			t.Errorf("details[status] = %q, want confirmed", got)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"success","subscriber_uid":"sub-1"}`))
	})

	code, stdout, _ := runCLI(t, "lists", "subscribers", "create",
		"--list-uid", "list-1",
		"--email", "jane@example.com",
		"--details-status", "confirmed",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stdout=%q", code, stdout)
	}

	if !strings.Contains(stdout, "sub-1") {
		t.Errorf("output missing subscriber_uid: %q", stdout)
	}
}

//nolint:paralleltest // uses t.Setenv, which is not safe to combine with t.Parallel
func TestListsSubscribersCreateRejectsInvalidEnumValue(t *testing.T) {
	newMailWizzMock(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("the API should not be called when flag validation fails")
	})

	code, _, stderr := runCLI(t, "lists", "subscribers", "create",
		"--list-uid", "list-1",
		"--email", "jane@example.com",
		"--details-status", "not-a-real-status",
	)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	if !strings.Contains(stderr, "details-status must be one of") {
		t.Errorf("stderr should explain the allowed values, got %q", stderr)
	}
}

//nolint:paralleltest // uses t.Setenv, which is not safe to combine with t.Parallel
func TestListsSubscribersSearchByEmail(t *testing.T) {
	newMailWizzMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/lists/list-1/subscribers/search-by-email" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		if got := r.URL.Query().Get("EMAIL"); got != "jane@example.com" {
			t.Errorf("EMAIL query param = %q, want jane@example.com", got)
		}

		_, _ = w.Write([]byte(`{"status":"success","data":{"subscriber_uid":"sub-1"}}`))
	})

	code, stdout, _ := runCLI(t, "lists", "subscribers", "search-by-email",
		"--list-uid", "list-1",
		"--email", "jane@example.com",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stdout=%q", code, stdout)
	}
}

//nolint:paralleltest // uses t.Setenv, which is not safe to combine with t.Parallel
func TestCampaignsStats(t *testing.T) {
	newMailWizzMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/campaigns/camp-1/stats" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		_, _ = w.Write([]byte(`{"status":"success","data":{"processed":10}}`))
	})

	code, stdout, _ := runCLI(t, "campaigns", "stats", "--campaign-uid", "camp-1")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0, stdout=%q", code, stdout)
	}

	if !strings.Contains(stdout, "processed") {
		t.Errorf("output missing stats data: %q", stdout)
	}
}

//nolint:paralleltest // uses t.Setenv, which is not safe to combine with t.Parallel
func TestCampaignsPauseUnpauseUsesOverriddenCommandName(t *testing.T) {
	newMailWizzMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/campaigns/camp-1/pause-unpause" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		_, _ = w.Write([]byte(`{"status":"success"}`))
	})

	code, _, _ := runCLI(t, "campaigns", "pause-unpause", "--campaign-uid", "camp-1")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

//nolint:paralleltest // uses t.Setenv, which is not safe to combine with t.Parallel
func TestRunReturnsNonZeroWithoutDoublePrintingAPIErrors(t *testing.T) {
	newMailWizzMock(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","error":"The list does not exist"}`))
	})

	code, stdout, stderr := runCLI(t, "lists", "view", "--list-uid", "does-not-exist")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	if !strings.Contains(stdout, "The list does not exist") {
		t.Errorf("stdout should contain the API's JSON error body, got %q", stdout)
	}

	if stderr != "" {
		t.Errorf("stderr should be empty (no redundant error message), got %q", stderr)
	}
}

//nolint:paralleltest // uses t.Setenv, which is not safe to combine with t.Parallel
func TestRunFailsForMissingRequiredFlag(t *testing.T) {
	newMailWizzMock(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("the API should not be called when a required flag is missing")
	})

	code, _, stderr := runCLI(t, "lists", "subscribers", "create", "--list-uid", "list-1")
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	if !strings.Contains(stderr, "email") {
		t.Errorf("stderr should mention the missing --email flag, got %q", stderr)
	}
}
