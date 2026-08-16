// Package apiclient implements a small HTTP client for the MailWizz REST
// API.
//
// It knows how to authenticate requests and encode query parameters and
// multipart form bodies, and it returns the raw bytes the API responds
// with. It intentionally does not decode response bodies into typed
// structs: the CLI prints the API's JSON response verbatim, so any field
// MailWizz adds in the future is preserved without requiring a client
// update.
package apiclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultTimeout is used when no timeout is configured on the Client.
const DefaultTimeout = 30 * time.Second

// Client talks to the MailWizz REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a Client for the given MailWizz API base URL and API key.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// Response is the raw result of an API call. The CLI decides how to
// render it and which exit code to use based on StatusCode; this package
// does not treat non-2xx responses as errors, since MailWizz returns a
// well-formed JSON error body that the caller should display as-is.
type Response struct {
	StatusCode int
	Body       []byte
}

// IsSuccess reports whether the response status code is in the 2xx range.
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices
}

// Get sends a GET request with the given query parameters.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, query, nil)
}

// Delete sends a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// PostForm sends a POST request with a multipart/form-data body.
func (c *Client) PostForm(ctx context.Context, path string, form url.Values) (*Response, error) {
	return c.doForm(ctx, http.MethodPost, path, form)
}

// PutForm sends a PUT request with a multipart/form-data body.
func (c *Client) PutForm(ctx context.Context, path string, form url.Values) (*Response, error) {
	return c.doForm(ctx, http.MethodPut, path, form)
}

// requestBody carries a pre-encoded body together with the content type
// header it must be sent with.
type requestBody struct {
	reader      io.Reader
	contentType string
}

func (c *Client) doForm(ctx context.Context, method, path string, form url.Values) (*Response, error) {
	body, contentType, err := encodeMultipart(form)
	if err != nil {
		return nil, fmt.Errorf("encode form body: %w", err)
	}

	return c.do(ctx, method, path, nil, &requestBody{reader: body, contentType: contentType})
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body *requestBody) (*Response, error) {
	requestURL := c.baseURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		reader = body.reader
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", body.contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return &Response{StatusCode: resp.StatusCode, Body: respBody}, nil
}

func encodeMultipart(form url.Values) (*bytes.Buffer, string, error) {
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)

	writeErr := writeMultipartFields(writer, form)

	// writer.Close only appends the closing boundary to buf; call it
	// unconditionally so it runs on every path, including when writing a
	// field above failed, rather than only on the success path.
	closeErr := writer.Close()

	if writeErr != nil {
		return nil, "", writeErr
	}

	if closeErr != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", closeErr)
	}

	return buf, writer.FormDataContentType(), nil
}

func writeMultipartFields(writer *multipart.Writer, form url.Values) error {
	for key, values := range form {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return fmt.Errorf("write field %q: %w", key, err)
			}
		}
	}

	return nil
}
