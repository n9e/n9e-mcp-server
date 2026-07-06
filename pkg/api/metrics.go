package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// defaultMaxPoints caps how many time-series points a range query may return per series
// before downsampling. Models tend to choke (and burn tokens) on tens of thousands of points.
const defaultMaxPoints = 1000

// promResponse is the native Prometheus API envelope. n9e's datasource proxy
// (/api/n9e/proxy/{ds_id}/*) forwards the TSDB response verbatim — it is NOT
// wrapped in the n9e {dat, err} envelope, so DoGet would silently decode it
// into an empty Dat. Metrics tools must go through doPromGet instead.
type promResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
}

// doPromGet fetches a datasource-proxy path and decodes the native Prometheus
// envelope: it returns the "data" payload on success, surfaces Prometheus
// {"status":"error"} responses as errors, and recognises the n9e {err} envelope
// the proxy emits when it fails before reaching the TSDB (e.g. unknown ds_id).
func doPromGet(c *client.Client, ctx context.Context, path string, params url.Values) (any, error) {
	raw, err := client.DoGetRaw(c, ctx, path, params)
	if err != nil {
		return nil, err
	}

	var pr promResponse
	if err := json.Unmarshal(raw, &pr); err != nil {
		return nil, fmt.Errorf("failed to decode datasource proxy response: %w, response preview: %s", err, preview(raw))
	}

	switch pr.Status {
	case "success":
		var data any
		if err := json.Unmarshal(pr.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to decode prometheus data: %w, response preview: %s", err, preview(raw))
		}
		return data, nil
	case "error":
		return nil, fmt.Errorf("prometheus error (%s): %s", pr.ErrorType, pr.Error)
	default:
		// Not a Prometheus envelope: most likely the n9e proxy rejected the
		// request before dialing the TSDB and answered with its own envelope.
		var env struct {
			Err string `json:"err"`
		}
		if json.Unmarshal(raw, &env) == nil && env.Err != "" {
			return nil, fmt.Errorf("n9e proxy error: %s", env.Err)
		}
		return nil, fmt.Errorf("unexpected datasource proxy response: %s", preview(raw))
	}
}

// preview truncates a response body for error messages.
func preview(raw []byte) string {
	if len(raw) > 200 {
		return string(raw[:200]) + "..."
	}
	return string(raw)
}

// RegisterMetricsToolset registers the metrics-query toolset.
// Tools wrap n9e's path-agnostic datasource proxy at /api/n9e/proxy/{ds_id}/*url
// to expose Prometheus-compatible queries (instant, range, label_values, series).
func RegisterMetricsToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("metrics", "Metrics query tools (Prometheus-compatible) via n9e datasource proxy")

	ts.AddReadTools(
		queryInstantTool(getClient),
		queryRangeTool(getClient),
		queryLabelValuesTool(getClient),
		querySeriesTool(getClient),
	)

	group.AddToolset(ts)
}

type queryInstantInput struct {
	DsId  int64   `json:"ds_id"`
	Query string  `json:"query"`
	Time  float64 `json:"time,omitempty"`
}

func queryInstantTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "query_instant",
			Description: "Run a Prometheus instant query (PromQL) against a datasource via n9e proxy.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Prometheus Instant Query",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"ds_id", "query"},
				Properties: map[string]*jsonschema.Schema{
					"ds_id": {Type: "integer", Description: "Datasource ID"},
					"query": {Type: "string", Description: "PromQL expression"},
					"time":  {Type: "number", Description: "Optional evaluation time (Unix seconds)"},
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

			params := url.Values{}
			params.Set("query", input.Query)
			if input.Time > 0 {
				params.Set("time", strconv.FormatFloat(input.Time, 'f', -1, 64))
			}

			path := fmt.Sprintf("/api/n9e/proxy/%d/api/v1/query", input.DsId)
			result, err := doPromGet(c, ctx, path, params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

type queryRangeInput struct {
	DsId      int64   `json:"ds_id"`
	Query     string  `json:"query"`
	Start     float64 `json:"start"`
	End       float64 `json:"end"`
	Step      float64 `json:"step,omitempty"`
	MaxPoints int     `json:"max_points,omitempty"`
}

func queryRangeTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "query_range",
			Description: "Run a Prometheus range query against a datasource. The 'step' is auto-adjusted upward when the result would exceed max_points (default 1000) per series, and 'truncated' is set in the response.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Prometheus Range Query",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"ds_id", "query", "start", "end"},
				Properties: map[string]*jsonschema.Schema{
					"ds_id":      {Type: "integer", Description: "Datasource ID"},
					"query":      {Type: "string", Description: "PromQL expression"},
					"start":      {Type: "number", Description: "Start time (Unix seconds)"},
					"end":        {Type: "number", Description: "End time (Unix seconds)"},
					"step":       {Type: "number", Description: "Resolution step in seconds (auto if omitted)"},
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

			step := input.Step
			truncated := false
			if step <= 0 {
				step = (input.End - input.Start) / float64(maxPoints)
				if step < 1 {
					step = 1
				}
			}
			// If user-supplied step would still exceed max_points, bump it.
			if pts := (input.End - input.Start) / step; pts > float64(maxPoints) {
				step = (input.End - input.Start) / float64(maxPoints)
				truncated = true
			}

			params := url.Values{}
			params.Set("query", input.Query)
			params.Set("start", strconv.FormatFloat(input.Start, 'f', -1, 64))
			params.Set("end", strconv.FormatFloat(input.End, 'f', -1, 64))
			params.Set("step", strconv.FormatFloat(step, 'f', -1, 64))

			path := fmt.Sprintf("/api/n9e/proxy/%d/api/v1/query_range", input.DsId)
			result, err := doPromGet(c, ctx, path, params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(map[string]any{
				"effective_step": step,
				"max_points":     maxPoints,
				"truncated":      truncated,
				"data":           result,
			}), nil
		}),
	)
}

type queryLabelValuesInput struct {
	DsId  int64  `json:"ds_id"`
	Label string `json:"label"`
}

func queryLabelValuesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "query_label_values",
			Description: "List all values seen for a Prometheus label.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Prometheus Label Values",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"ds_id", "label"},
				Properties: map[string]*jsonschema.Schema{
					"ds_id": {Type: "integer"},
					"label": {Type: "string", Description: "Label name (e.g. 'instance')"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input queryLabelValuesInput) (*mcp.CallToolResult, error) {
			if input.DsId <= 0 || input.Label == "" {
				return toolset.NewToolResultError("ds_id and label are required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			path := fmt.Sprintf("/api/n9e/proxy/%d/api/v1/label/%s/values", input.DsId, url.PathEscape(input.Label))
			result, err := doPromGet(c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

type querySeriesInput struct {
	DsId  int64    `json:"ds_id"`
	Match []string `json:"match"`
	Start float64  `json:"start,omitempty"`
	End   float64  `json:"end,omitempty"`
}

func querySeriesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "query_series",
			Description: "Find series matching one or more matchers (Prometheus /series).",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Prometheus Series",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"ds_id", "match"},
				Properties: map[string]*jsonschema.Schema{
					"ds_id": {Type: "integer"},
					"match": {Type: "array", Items: &jsonschema.Schema{Type: "string"}, Description: "Series selectors, e.g. up{job='node'}"},
					"start": {Type: "number"},
					"end":   {Type: "number"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input querySeriesInput) (*mcp.CallToolResult, error) {
			if input.DsId <= 0 || len(input.Match) == 0 {
				return toolset.NewToolResultError("ds_id and match are required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			params := url.Values{}
			for _, m := range input.Match {
				params.Add("match[]", m)
			}
			if input.Start > 0 {
				params.Set("start", strconv.FormatFloat(input.Start, 'f', -1, 64))
			}
			if input.End > 0 {
				params.Set("end", strconv.FormatFloat(input.End, 'f', -1, 64))
			}
			path := fmt.Sprintf("/api/n9e/proxy/%d/api/v1/series", input.DsId)
			result, err := doPromGet(c, ctx, path, params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}
