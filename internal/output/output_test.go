package output_test

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/onetwist-software/mailwizz-cli/internal/apiclient"
	"github.com/onetwist-software/mailwizz-cli/internal/output"
)

func TestPrintIndentsJSONAndReturnsExitCode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	resp := &apiclient.Response{StatusCode: http.StatusOK, Body: []byte(`{"status":"success"}`)}

	code, err := output.Print(&buf, resp)
	if err != nil {
		t.Fatalf("Print: %v", err)
	}

	if code != 0 {
		t.Errorf("code = %d, want 0", code)
	}

	want := "{\n  \"status\": \"success\"\n}\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

func TestPrintReturnsNonZeroExitCodeForErrorStatus(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	resp := &apiclient.Response{StatusCode: http.StatusBadRequest, Body: []byte(`{"status":"error"}`)}

	code, err := output.Print(&buf, resp)
	if err != nil {
		t.Fatalf("Print: %v", err)
	}

	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
}

func TestPrintFallsBackToRawBodyForInvalidJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	resp := &apiclient.Response{StatusCode: http.StatusOK, Body: []byte("not json")}

	if _, err := output.Print(&buf, resp); err != nil {
		t.Fatalf("Print: %v", err)
	}

	if buf.String() != "not json\n" {
		t.Errorf("output = %q, want %q", buf.String(), "not json\n")
	}
}

func TestPrintHandlesEmptyBody(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	resp := &apiclient.Response{StatusCode: http.StatusOK, Body: nil}

	if _, err := output.Print(&buf, resp); err != nil {
		t.Fatalf("Print: %v", err)
	}

	if buf.String() != "{}\n" {
		t.Errorf("output = %q, want %q", buf.String(), "{}\n")
	}
}

func TestHandleReturnsErrRequestFailedForErrorStatus(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	resp := &apiclient.Response{StatusCode: http.StatusNotFound, Body: []byte(`{"status":"error"}`)}

	err := output.Handle(&buf, resp)
	if !errors.Is(err, output.ErrRequestFailed) {
		t.Fatalf("Handle err = %v, want ErrRequestFailed", err)
	}

	if buf.Len() == 0 {
		t.Errorf("expected the response body to still be printed")
	}
}

func TestHandleReturnsNilForSuccessStatus(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	resp := &apiclient.Response{StatusCode: http.StatusCreated, Body: []byte(`{"status":"success"}`)}

	if err := output.Handle(&buf, resp); err != nil {
		t.Fatalf("Handle: %v", err)
	}
}
