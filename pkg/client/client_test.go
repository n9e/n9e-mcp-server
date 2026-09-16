package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type testPayload struct {
	A int `json:"a"`
}

// newEnvelopeClient returns a client whose every response is `body`.
func newEnvelopeClient(t *testing.T, body string) *Client {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("test-token", srv.URL, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestDecodeResponseHandlesBothEnvelopes pins the two response spellings
// Nightingale uses. The datasource router answers with the second one, which a
// dat/err-only decoder read as an empty payload.
func TestDecodeResponseHandlesBothEnvelopes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want testPayload
	}{
		{
			name: "ginx dat envelope",
			body: `{"dat":{"a":1},"err":"","request_id":"r1"}`,
			want: testPayload{A: 1},
		},
		{
			name: "router data envelope",
			body: `{"data":{"a":2},"error":""}`,
			want: testPayload{A: 2},
		},
		{
			name: "dat wins when both are present",
			body: `{"dat":{"a":3},"data":{"a":4},"err":"","error":""}`,
			want: testPayload{A: 3},
		},
		{
			name: "null payload is not an error",
			body: `{"data":null,"error":""}`,
			want: testPayload{},
		},
		{
			name: "empty payload is not an error",
			body: `{}`,
			want: testPayload{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newEnvelopeClient(t, tc.body)

			got, err := DoPost[testPayload](c, context.Background(), "/api/n9e/anything", map[string]any{"x": 1})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("payload = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestDecodeResponseReadsDataEnvelopePayload covers the array payloads that
// /datasource/list and /datasource/desc answer with.
func TestDecodeResponseReadsDataEnvelopePayload(t *testing.T) {
	c := newEnvelopeClient(t, `{"data":[{"a":1},{"a":2}],"error":""}`)

	got, err := DoGet[[]testPayload](c, context.Background(), "/api/n9e/datasource/list", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].A != 1 || got[1].A != 2 {
		t.Fatalf("payload = %+v, want [{1} {2}]", got)
	}
}

// TestDecodeResponseSurfacesErrors pins that a business error carried on an
// HTTP 200 response becomes an *APIError for both envelopes. Before this, the
// data envelope reported success while nothing had been applied.
func TestDecodeResponseSurfacesErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "ginx err field",
			body: `{"dat":null,"err":"no such datasource id: 99"}`,
			want: "no such datasource id: 99",
		},
		{
			name: "router error message",
			body: `{"error":{"message":"name already exists"}}`,
			want: "name already exists",
		},
		{
			name: "router bare error string",
			body: `{"error":"unauthorized"}`,
			want: "unauthorized",
		},
		{
			// n9e forwards Go errors unchanged, so they arrive marshalled.
			name: "marshalled go error",
			body: `{"error":{"message":{"Number":1062,"Message":"Duplicate entry 'sa-prom' for key 'datasource.name'"}}}`,
			want: "Duplicate entry 'sa-prom' for key 'datasource.name'",
		},
		{
			// gin bind failures answer with an empty message object; it still has
			// to read as a failure rather than a silent success.
			name: "router error without a usable message",
			body: `{"error":{"message":{}}}`,
			want: `{"message":{}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newEnvelopeClient(t, tc.body)

			_, err := DoPost[testPayload](c, context.Background(), "/api/n9e/datasource/upsert", nil)
			if err == nil {
				t.Fatal("expected an error")
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error type = %T, want *APIError", err)
			}
			if apiErr.ErrMsg != tc.want {
				t.Fatalf("ErrMsg = %q, want %q", apiErr.ErrMsg, tc.want)
			}
			// APIError formats the message with %q, so compare against the quoted form.
			if !strings.Contains(apiErr.Error(), strconv.Quote(tc.want)) {
				t.Fatalf("formatted error %q does not carry the message", apiErr.Error())
			}
			if apiErr.Path != "/api/n9e/datasource/upsert" {
				t.Fatalf("Path = %q", apiErr.Path)
			}
		})
	}
}

// TestDecodeResponseEmptyErrorIsNotAnError covers the success shape
// router.Render emits: {"data":...,"error":""}.
func TestDecodeResponseEmptyErrorIsNotAnError(t *testing.T) {
	c := newEnvelopeClient(t, `{"data":{"a":7},"error":""}`)

	got, err := DoPut[testPayload](c, context.Background(), "/api/n9e/anything", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.A != 7 {
		t.Fatalf("payload = %+v, want {A:7}", got)
	}
}

// TestDecodeResponseCoversEveryVerb guards the shared decoder: all five request
// helpers must unwrap the data envelope, not just the ones a test happens to use.
func TestDecodeResponseCoversEveryVerb(t *testing.T) {
	const body = `{"data":{"a":9},"error":""}`
	want := testPayload{A: 9}

	verbs := map[string]func(*Client) (testPayload, error){
		"DoGet": func(c *Client) (testPayload, error) {
			return DoGet[testPayload](c, context.Background(), "/api/n9e/x", nil)
		},
		"DoGetLarge": func(c *Client) (testPayload, error) {
			return DoGetLarge[testPayload](c, context.Background(), "/api/n9e/x", nil, 0)
		},
		"DoPost": func(c *Client) (testPayload, error) {
			return DoPost[testPayload](c, context.Background(), "/api/n9e/x", nil)
		},
		"DoPut": func(c *Client) (testPayload, error) {
			return DoPut[testPayload](c, context.Background(), "/api/n9e/x", nil)
		},
		"DoDelete": func(c *Client) (testPayload, error) {
			return DoDelete[testPayload](c, context.Background(), "/api/n9e/x", nil)
		},
	}

	for name, call := range verbs {
		t.Run(name, func(t *testing.T) {
			c := newEnvelopeClient(t, body)

			got, err := call(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != want {
				t.Fatalf("payload = %+v, want %+v", got, want)
			}
		})
	}
}

// TestNon2xxIsAnError covers that a non-2xx response is rejected before
// decoding. ginx answers errors with a plain-text body, so this path must not
// depend on the body being JSON.
func TestNon2xxIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("fields empty"))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("test-token", srv.URL, "test-agent")
	if err != nil {
		t.Fatal(err)
	}

	_, err = DoPut[testPayload](c, context.Background(), "/api/n9e/anything", nil)
	if err == nil {
		t.Fatal("expected an error for HTTP 400")
	}
	if !strings.Contains(err.Error(), "fields empty") {
		t.Fatalf("error = %q, want it to carry the response body", err)
	}
}
