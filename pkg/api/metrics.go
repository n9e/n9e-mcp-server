package api

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// defaultMaxPoints caps how many time-series points a range query may return per series
// before downsampling. Models tend to choke (and burn tokens) on tens of thousands of points.
const defaultMaxPoints = 1000

// The metrics tools go through n9e's batch query APIs
// (/api/n9e/query-instant-batch, /api/n9e/query-range-batch) rather than the
// datasource proxy: the batch APIs answer with the standard {dat, err}
// envelope (so the shared client decoding applies) and are served by n9e's own
// Prometheus clients instead of a byte-for-byte reverse proxy. dat carries one
// result per submitted query — these tools always submit exactly one.

// RegisterMetricsToolset registers the metrics-query toolset: PromQL instant
// and range queries against Prometheus-compatible datasources.
func RegisterMetricsToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("metrics", "Metrics query tools (PromQL instant/range) via n9e query APIs")

	ts.AddReadTools(
		queryInstantTool(getClient),
		queryRangeTool(getClient),
	)

	group.AddToolset(ts)
}

// firstResult unwraps the batch envelope: dat is one entry per submitted
// query, and the tools submit exactly one. A missing entry (nil dat on an
// error the envelope didn't carry) degrades to null rather than erroring —
// the caller has already passed the envelope's err check.
func firstResult(dat []json.RawMessage) any {
	if len(dat) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(dat[0], &v); err != nil {
		return string(dat[0])
	}
	return v
}

type queryInstantInput struct {
	DsId  int64  `json:"ds_id"`
	Query string `json:"query"`
	Time  int64  `json:"time,omitempty"`
}

func queryInstantTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "query_instant",
			Description: "Run a PromQL instant query against a Prometheus-compatible datasource. Returns the result series array (each item: metric labels + [timestamp, value]).",
			Annotations: &mcp.ToolAnnotations{
				Title:        "PromQL Instant Query",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"ds_id", "query"},
				Properties: map[string]*jsonschema.Schema{
					"ds_id": {Type: "integer", Description: "Datasource ID"},
					"query": {Type: "string", Description: "PromQL expression"},
					"time":  {Type: "integer", Description: "Optional evaluation time (Unix seconds, default now)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input queryInstantInput) (*mcp.CallToolResult, error) {
			if input.DsId <= 0 {
				return toolset.NewToolResultError("ds_id is required"), nil
			}
			if input.Query == "" {
				return toolset.NewToolResultError("query is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			// A zero time is not rejected by the batch API — it silently
			// evaluates at the epoch and returns an empty vector — so "now"
			// must be stamped client-side.
			evalTime := input.Time
			if evalTime <= 0 {
				evalTime = time.Now().Unix()
			}

			body := map[string]any{
				"datasource_id": input.DsId,
				"queries": []map[string]any{
					{"time": evalTime, "query": input.Query},
				},
			}
			dat, err := client.DoPost[[]json.RawMessage](c, ctx, "/api/n9e/query-instant-batch", body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(firstResult(dat)), nil
		}),
	)
}

type queryRangeInput struct {
	DsId      int64  `json:"ds_id"`
	Query     string `json:"query"`
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	Step      int64  `json:"step,omitempty"`
	MaxPoints int    `json:"max_points,omitempty"`
}

func queryRangeTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "query_range",
			Description: "Run a PromQL range query against a Prometheus-compatible datasource. The 'step' is auto-adjusted upward when the result would exceed max_points (default 1000) per series, and 'truncated' is set in the response.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "PromQL Range Query",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"ds_id", "query", "start", "end"},
				Properties: map[string]*jsonschema.Schema{
					"ds_id":      {Type: "integer", Description: "Datasource ID"},
					"query":      {Type: "string", Description: "PromQL expression"},
					"start":      {Type: "integer", Description: "Start time (Unix seconds)"},
					"end":        {Type: "integer", Description: "End time (Unix seconds)"},
					"step":       {Type: "integer", Description: "Resolution step in seconds (auto if omitted)"},
					"max_points": {Type: "integer", Description: "Cap points per series (default 1000)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input queryRangeInput) (*mcp.CallToolResult, error) {
			if input.DsId <= 0 || input.Query == "" || input.Start <= 0 || input.End <= 0 {
				return toolset.NewToolResultError("ds_id, query, start, end are required"), nil
			}
			if input.End <= input.Start {
				return toolset.NewToolResultError("end must be > start"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			maxPoints := input.MaxPoints
			if maxPoints <= 0 {
				maxPoints = defaultMaxPoints
			}

			// The batch API takes step as whole seconds; ceil keeps the point
			// count at or under max_points.
			window := input.End - input.Start
			step := input.Step
			truncated := false
			if step <= 0 {
				step = int64(math.Ceil(float64(window) / float64(maxPoints)))
			} else if window/step > int64(maxPoints) {
				step = int64(math.Ceil(float64(window) / float64(maxPoints)))
				truncated = true
			}
			if step < 1 {
				step = 1
			}

			body := map[string]any{
				"datasource_id": input.DsId,
				"queries": []map[string]any{
					{"start": input.Start, "end": input.End, "step": step, "query": input.Query},
				},
			}
			dat, err := client.DoPost[[]json.RawMessage](c, ctx, "/api/n9e/query-range-batch", body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(map[string]any{
				"effective_step": step,
				"max_points":     maxPoints,
				"truncated":      truncated,
				"data":           firstResult(dat),
			}), nil
		}),
	)
}
