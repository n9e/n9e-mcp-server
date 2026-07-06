package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/n9e/n9e-mcp-server/pkg/client"
)

// newProxyClient stands up a fake n9e datasource proxy and returns a client
// pointed at it.
func newProxyClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := client.NewClient("test-token", srv.URL, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestDoPromGetSuccess pins the reason doPromGet exists: the proxy returns the
// native Prometheus envelope (NOT the n9e {dat, err} envelope), and the tool
// must get the "data" payload out of it — DoGet would silently return nil.
func TestDoPromGetSuccess(t *testing.T) {
	c := newProxyClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/n9e/proxy/1/api/v1/query" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "up" {
			t.Errorf("query param = %q, want up", got)
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"up"},"value":[1751800000,"1"]}]}}`))
	})

	params := url.Values{}
	params.Set("query", "up")
	got, err := doPromGet(c, context.Background(), "/api/n9e/proxy/1/api/v1/query", params)
	if err != nil {
		t.Fatalf("doPromGet: %v", err)
	}
	data, ok := got.(map[string]any)
	if !ok || data["resultType"] != "vector" {
		t.Fatalf("data = %#v, want vector payload", got)
	}
	if n := len(data["result"].([]any)); n != 1 {
		t.Fatalf("result length = %d, want 1", n)
	}
}

// TestDoPromGetPrometheusError surfaces a TSDB-side error (bad PromQL etc.) as
// a tool error instead of an empty result.
func TestDoPromGetPrometheusError(t *testing.T) {
	c := newProxyClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"error","errorType":"bad_data","error":"parse error: unexpected end of input"}`))
	})

	_, err := doPromGet(c, context.Background(), "/api/n9e/proxy/1/api/v1/query", nil)
	if err == nil {
		t.Fatal("expected error for status=error response")
	}
	if !strings.Contains(err.Error(), "bad_data") || !strings.Contains(err.Error(), "parse error") {
		t.Fatalf("error should carry errorType and message: %v", err)
	}
}

// TestDoPromGetN9eEnvelopeError covers the proxy failing before it reaches the
// TSDB (e.g. unknown ds_id): n9e answers with its own {err} envelope and 200.
func TestDoPromGetN9eEnvelopeError(t *testing.T) {
	c := newProxyClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"dat":null,"err":"no such datasource"}`))
	})

	_, err := doPromGet(c, context.Background(), "/api/n9e/proxy/99/api/v1/query", nil)
	if err == nil {
		t.Fatal("expected error for n9e envelope error")
	}
	if !strings.Contains(err.Error(), "no such datasource") {
		t.Fatalf("error should carry the n9e err message: %v", err)
	}
}

// TestDoGetRawVerbatim proves DoGetRaw hands back the body untouched.
func TestDoGetRawVerbatim(t *testing.T) {
	const body = `{"status":"success","data":["a","b"]}`
	c := newProxyClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-Token") != "test-token" {
			t.Errorf("token header missing")
		}
		_, _ = w.Write([]byte(body))
	})

	raw, err := client.DoGetRaw(c, context.Background(), "/api/n9e/proxy/1/api/v1/label/job/values", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Fatalf("raw = %s, want verbatim body", raw)
	}
	var pr map[string]json.RawMessage
	if err := json.Unmarshal(raw, &pr); err != nil {
		t.Fatalf("raw should stay valid JSON: %v", err)
	}
}
