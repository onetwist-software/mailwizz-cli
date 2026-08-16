package apiclient_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/onetwist-software/mailwizz-cli/internal/apiclient"
)

func TestGetSendsAuthHeaderAndQuery(t *testing.T) {
	t.Parallel()

	var gotPath, gotQuery, gotAPIKey, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAPIKey = r.Header.Get("X-Api-Key")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, "my-api-key")

	resp, err := client.Get(context.Background(), "/lists", url.Values{"page": {"2"}, "per_page": {"10"}})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}

	if gotPath != "/lists" {
		t.Errorf("path = %q, want /lists", gotPath)
	}

	if gotQuery != "page=2&per_page=10" {
		t.Errorf("query = %q, want page=2&per_page=10", gotQuery)
	}

	if gotAPIKey != "my-api-key" {
		t.Errorf("api key header = %q, want my-api-key", gotAPIKey)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	if !resp.IsSuccess() {
		t.Errorf("IsSuccess() = false, want true")
	}

	if string(resp.Body) != `{"status":"success"}` {
		t.Errorf("body = %q", resp.Body)
	}
}

func TestPostFormSendsMultipartBody(t *testing.T) {
	t.Parallel()

	var gotContentType, gotName string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")

		if err := r.ParseMultipartForm(1 << 20); err != nil { //nolint:gosec // test fixture, bounded literal
			t.Errorf("ParseMultipartForm: %v", err)
		}

		gotName = r.FormValue("general[name]")

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"success","list_uid":"abc123"}`))
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, "key")

	form := url.Values{"general[name]": {"My List"}}

	resp, err := client.PostForm(context.Background(), "/lists", form)
	if err != nil {
		t.Fatalf("PostForm: %v", err)
	}

	if want := "multipart/form-data"; gotContentType == "" || gotContentType[:len(want)] != want {
		t.Errorf("content type = %q, want prefix %q", gotContentType, want)
	}

	if gotName != "My List" {
		t.Errorf("form field general[name] = %q, want %q", gotName, "My List")
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
}

func TestPutFormAndDelete(t *testing.T) {
	t.Parallel()

	var gotMethods []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethods = append(gotMethods, r.Method)

		if r.Method == http.MethodPut {
			if err := r.ParseMultipartForm(1 << 20); err != nil { //nolint:gosec // test fixture, bounded literal
				t.Errorf("ParseMultipartForm: %v", err)
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, "key")

	if _, err := client.PutForm(context.Background(), "/lists/abc", url.Values{"general[name]": {"Updated"}}); err != nil {
		t.Fatalf("PutForm: %v", err)
	}

	if _, err := client.Delete(context.Background(), "/lists/abc"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	want := []string{http.MethodPut, http.MethodDelete}
	if len(gotMethods) != len(want) || gotMethods[0] != want[0] || gotMethods[1] != want[1] {
		t.Errorf("methods = %v, want %v", gotMethods, want)
	}
}

func TestNonSuccessStatusIsNotAnError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"error","error":"invalid list_uid"}`))
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, "key")

	resp, err := client.Get(context.Background(), "/lists/does-not-exist", nil)
	if err != nil {
		t.Fatalf("Get returned an error for a 400 response: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	if resp.IsSuccess() {
		t.Errorf("IsSuccess() = true, want false")
	}
}

func TestGetReturnsErrorOnNetworkFailure(t *testing.T) {
	t.Parallel()
	// A closed server guarantees connection failures without relying on
	// unreachable-network assumptions in CI sandboxes.
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close()

	client := apiclient.New(srv.URL, "key")

	if _, err := client.Get(context.Background(), "/lists", nil); err == nil {
		t.Fatal("expected an error when the server is unreachable")
	}
}

func TestBaseURLTrailingSlashIsTrimmed(t *testing.T) {
	t.Parallel()

	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "{}")
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL+"/", "key")

	if _, err := client.Get(context.Background(), "/lists", nil); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if gotPath != "/lists" {
		t.Errorf("path = %q, want /lists (no double slash)", gotPath)
	}
}
