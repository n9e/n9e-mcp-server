package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/n9e/n9e-mcp-server/pkg/client"
)

// TestSetDatasourceStatusPostsOneRequestPerDatasource pins the wire format of
// /datasource/status/update: Nightingale binds a single models.Datasource
// ({id, status}) and updates that one row, so posting {"ids": [...]} binds a
// zero id, updates nothing and still answers HTTP 200.
func TestSetDatasourceStatusPostsOneRequestPerDatasource(t *testing.T) {
	var bodies []map[string]any

	c := newBatchClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/n9e/datasource/status/update" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		bodies = append(bodies, body)
		// Nightingale answers through center/router.Render.
		_, _ = w.Write([]byte(`{"data":{"id":1,"status":"disabled"},"error":""}`))
	})
	getClient := func(context.Context) *client.Client { return c }

	res := callTool(t, setDatasourceStatusTool(getClient), `{"ids":[1,2],"status":"disabled"}`)
	if res.IsError {
		t.Fatalf("tool errored: %s", resultText(res))
	}

	if len(bodies) != 2 {
		t.Fatalf("issued %d status updates, want one per datasource (2)", len(bodies))
	}
	for i, wantId := range []float64{1, 2} {
		if got := bodies[i]["id"]; got != wantId {
			t.Errorf("request %d: id = %v, want %v", i, got, wantId)
		}
		if got := bodies[i]["status"]; got != "disabled" {
			t.Errorf("request %d: status = %v, want disabled", i, got)
		}
		if _, hasIds := bodies[i]["ids"]; hasIds {
			t.Errorf("request %d: must not send the batch ids field", i)
		}
		if len(bodies[i]) != 2 {
			t.Errorf("request %d: body = %v, want only {id, status}", i, bodies[i])
		}
	}
}

// TestSetDatasourceStatusSurfacesEnvelopeError covers the other half of the bug:
// a business error in the data envelope used to be swallowed, so the tool
// reported success while no row had been updated.
func TestSetDatasourceStatusSurfacesEnvelopeError(t *testing.T) {
	c := newBatchClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"message":"name already exists"}}`))
	})
	getClient := func(context.Context) *client.Client { return c }

	res := callTool(t, setDatasourceStatusTool(getClient), `{"ids":[7],"status":"disabled"}`)
	if !res.IsError {
		t.Fatalf("expected a tool error, got: %s", resultText(res))
	}
	if !strings.Contains(resultText(res), "name already exists") {
		t.Fatalf("error should carry the n9e message: %s", resultText(res))
	}
	if !strings.Contains(resultText(res), "datasource 7") {
		t.Fatalf("error should name the datasource: %s", resultText(res))
	}
}

// TestSetDatasourceStatusValidatesInput keeps the local guards ahead of any call.
func TestSetDatasourceStatusValidatesInput(t *testing.T) {
	c := newBatchClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be issued for invalid input")
	})
	getClient := func(context.Context) *client.Client { return c }

	cases := []struct {
		name string
		args string
		want string
	}{
		{"empty ids", `{"ids":[],"status":"disabled"}`, "ids must be non-empty"},
		{"unknown status", `{"ids":[1],"status":"paused"}`, "status must be"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := callTool(t, setDatasourceStatusTool(getClient), tc.args)
			if !res.IsError {
				t.Fatal("expected a tool error")
			}
			if !strings.Contains(resultText(res), tc.want) {
				t.Fatalf("error = %q, want it to contain %q", resultText(res), tc.want)
			}
		})
	}
}
