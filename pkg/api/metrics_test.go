package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newBatchClient stands up a fake n9e batch-query API and returns a client
// pointed at it.
func newBatchClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := client.NewClient("test-token", srv.URL, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// callTool drives a ServerTool's handler the way the MCP server would.
func callTool(t *testing.T, st toolset.ServerTool, args string) *mcp.CallToolResult {
	t.Helper()
	req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(args)}}
	res, err := st.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("tool handler: %v", err)
	}
	return res
}

func resultText(res *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

// TestQueryInstantTool pins the batch-API contract: POST
// /api/n9e/query-instant-batch with datasource_id + one query (time stamped
// client-side — the API silently evaluates a zero time at the epoch and
// returns an empty vector), and the tool unwraps dat[0] — the single query's
// vector — instead of echoing the whole batch envelope.
func TestQueryInstantTool(t *testing.T) {
	var gotBody map[string]any
	c := newBatchClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/n9e/query-instant-batch" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		_, _ = w.Write([]byte(`{"dat":[[{"metric":{"__name__":"up"},"value":[1751800000,"1"]}]],"err":""}`))
	})
	getClient := func(context.Context) *client.Client { return c }

	res := callTool(t, queryInstantTool(getClient), `{"ds_id":1,"query":"up"}`)
	if res.IsError {
		t.Fatalf("tool errored: %s", resultText(res))
	}

	if gotBody["datasource_id"] != float64(1) {
		t.Fatalf("datasource_id = %v, want 1", gotBody["datasource_id"])
	}
	queries := gotBody["queries"].([]any)
	if len(queries) != 1 {
		t.Fatalf("queries length = %d, want 1", len(queries))
	}
	q := queries[0].(map[string]any)
	if q["query"] != "up" {
		t.Fatalf("query = %v, want up", q["query"])
	}
	if ts, _ := q["time"].(float64); ts <= 0 {
		t.Fatalf("time must default to now (nonzero), got %v", q["time"])
	}

	// dat[0] unwrapped: the text is the vector array itself.
	var vector []map[string]any
	if err := json.Unmarshal([]byte(resultText(res)), &vector); err != nil {
		t.Fatalf("result should be the unwrapped vector array: %v; text=%s", err, resultText(res))
	}
	if len(vector) != 1 || vector[0]["metric"].(map[string]any)["__name__"] != "up" {
		t.Fatalf("unexpected vector: %s", resultText(res))
	}
}

// TestQueryInstantToolEnvelopeError surfaces the n9e {err} envelope (e.g.
// unknown datasource id) as a tool error instead of an empty result.
func TestQueryInstantToolEnvelopeError(t *testing.T) {
	c := newBatchClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"dat":null,"err":"no such datasource id: 99"}`))
	})
	getClient := func(context.Context) *client.Client { return c }

	res := callTool(t, queryInstantTool(getClient), `{"ds_id":99,"query":"up"}`)
	if !res.IsError {
		t.Fatal("expected tool error for envelope err")
	}
	if !strings.Contains(resultText(res), "no such datasource") {
		t.Fatalf("error should carry the n9e err message: %s", resultText(res))
	}
}

// TestQueryRangeToolStepAutoAdjust pins the downsampling contract on the batch
// API: step is whole seconds, auto-derived when omitted and bumped (with
// truncated=true) when a user-supplied step would exceed max_points.
func TestQueryRangeToolStepAutoAdjust(t *testing.T) {
	var gotQuery map[string]any
	c := newBatchClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/n9e/query-range-batch" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		gotQuery = body["queries"].([]any)[0].(map[string]any)
		_, _ = w.Write([]byte(`{"dat":[[{"metric":{"__name__":"up"},"values":[[1751800000,"1"]]}]],"err":""}`))
	})
	getClient := func(context.Context) *client.Client { return c }

	// 2h window, no step: ceil(7200/1000) = 8s.
	res := callTool(t, queryRangeTool(getClient),
		`{"ds_id":1,"query":"up","start":1751800000,"end":1751807200}`)
	if res.IsError {
		t.Fatalf("tool errored: %s", resultText(res))
	}
	if gotQuery["step"] != float64(8) {
		t.Fatalf("auto step = %v, want 8", gotQuery["step"])
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(resultText(res)), &out); err != nil {
		t.Fatal(err)
	}
	if out["truncated"] != false || out["effective_step"] != float64(8) {
		t.Fatalf("auto-step result meta wrong: %s", resultText(res))
	}
	if _, ok := out["data"].([]any); !ok {
		t.Fatalf("data should be the unwrapped matrix: %s", resultText(res))
	}

	// Same window with step=1 (7200 points): bumped to 8 and flagged.
	res = callTool(t, queryRangeTool(getClient),
		`{"ds_id":1,"query":"up","start":1751800000,"end":1751807200,"step":1}`)
	if res.IsError {
		t.Fatalf("tool errored: %s", resultText(res))
	}
	if gotQuery["step"] != float64(8) {
		t.Fatalf("bumped step = %v, want 8", gotQuery["step"])
	}
	if err := json.Unmarshal([]byte(resultText(res)), &out); err != nil {
		t.Fatal(err)
	}
	if out["truncated"] != true {
		t.Fatalf("oversized user step must set truncated: %s", resultText(res))
	}

	// A short window keeps a small user step untouched.
	res = callTool(t, queryRangeTool(getClient),
		`{"ds_id":1,"query":"up","start":1751800000,"end":1751800600,"step":15}`)
	if res.IsError {
		t.Fatalf("tool errored: %s", resultText(res))
	}
	if gotQuery["step"] != float64(15) {
		t.Fatalf("in-budget step = %v, want 15 untouched", gotQuery["step"])
	}
}

// TestQueryRangeToolValidation rejects impossible windows before any request.
func TestQueryRangeToolValidation(t *testing.T) {
	called := false
	c := newBatchClient(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	getClient := func(context.Context) *client.Client { return c }

	res := callTool(t, queryRangeTool(getClient),
		`{"ds_id":1,"query":"up","start":1751807200,"end":1751800000}`)
	if !res.IsError {
		t.Fatal("expected error for end <= start")
	}
	if called {
		t.Fatal("no request should be made for invalid input")
	}
}

// TestMetricsToolsetSurface pins the toolset to exactly the two query tools:
// the proxy-based label_values/series tools are gone.
func TestMetricsToolsetSurface(t *testing.T) {
	group := toolset.NewToolsetGroup(false)
	RegisterMetricsToolset(group, func(context.Context) *client.Client { return nil })
	if err := group.EnableToolsets([]string{"metrics"}); err != nil {
		t.Fatal(err)
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "t", Version: "0"}, nil)
	group.RegisterAll(server)

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	if len(names) != 2 || !names["query_instant"] || !names["query_range"] {
		t.Fatalf("metrics toolset should expose exactly query_instant and query_range, got %v", names)
	}
}
