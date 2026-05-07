package api

import (
	"context"
	"fmt"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// defaultLogLimit caps the number of log lines returned per query.
const defaultLogLimit = 200

// maxLogRangeSeconds rejects log queries spanning more than 7 days to prevent runaway costs.
const maxLogRangeSeconds = 7 * 24 * 60 * 60

// RegisterLogsToolset registers the logs-query toolset.
// Backed by n9e's plugin-dispatched /api/n9e/logs-query (Loki/ES/OS).
func RegisterLogsToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("logs", "Logs query tools (Loki/Elasticsearch/OpenSearch) via n9e logs-query")

	ts.AddReadTools(
		queryLogsTool(getClient),
		listLogIndicesTool(getClient),
		listLogFieldsTool(getClient),
	)

	group.AddToolset(ts)
}

type queryLogsInput struct {
	Body  map[string]any `json:"body"`
	Limit int            `json:"limit,omitempty"`
	Start int64          `json:"start,omitempty"`
	End   int64          `json:"end,omitempty"`
}

func queryLogsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "query_logs",
			Description: "Query logs from a datasource (Loki/ES/OS) via n9e's plugin-dispatched /logs-query endpoint. Pass 'body' as the n9e query payload (must include datasource_id and the engine-specific query). Time range > 7 days is rejected.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Query Logs",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body":  {Type: "object", Description: "Log query body (matches n9e's /logs-query payload)"},
					"limit": {Type: "integer", Description: "Cap on returned lines (default 200)"},
					"start": {Type: "integer", Description: "Optional Unix-second start (used only for range validation)"},
					"end":   {Type: "integer", Description: "Optional Unix-second end (used only for range validation)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input queryLogsInput) (*mcp.CallToolResult, error) {
			if len(input.Body) == 0 {
				return toolset.NewToolResultError("body is required"), nil
			}
			if input.Start > 0 && input.End > 0 {
				if input.End-input.Start > maxLogRangeSeconds {
					return toolset.NewToolResultError(fmt.Sprintf("time range exceeds max of %d seconds (~7 days)", maxLogRangeSeconds)), nil
				}
			}

			body := input.Body
			limit := input.Limit
			if limit <= 0 {
				limit = defaultLogLimit
			}
			// Inject limit if the caller didn't set one (n9e plugins inspect this hint differently).
			if _, ok := body["limit"]; !ok {
				body["limit"] = limit
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/logs-query", body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{
				"limit": limit,
				"data":  result,
			}), nil
		}),
	)
}

type logIndicesInput struct {
	Body   map[string]any `json:"body"`
	Engine string         `json:"engine,omitempty"` // "es" (default) or "os" (OpenSearch)
}

func listLogIndicesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_log_indices",
			Description: "List indices for an Elasticsearch (default) or OpenSearch datasource. Body must include datasource_id.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Log Indices",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body":   {Type: "object", Description: "Body (must include datasource_id)"},
					"engine": {Type: "string", Description: "'es' (default) or 'os' for OpenSearch"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input logIndicesInput) (*mcp.CallToolResult, error) {
			if len(input.Body) == 0 {
				return toolset.NewToolResultError("body is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			path := "/api/n9e/indices"
			if input.Engine == "os" {
				path = "/api/n9e/os-indices"
			}
			result, err := client.DoPost[any](c, ctx, path, input.Body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

func listLogFieldsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_log_fields",
			Description: "List fields for an Elasticsearch (default) or OpenSearch index. Body must include datasource_id and the index name.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Log Fields",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body":   {Type: "object"},
					"engine": {Type: "string", Description: "'es' (default) or 'os' for OpenSearch"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input logIndicesInput) (*mcp.CallToolResult, error) {
			if len(input.Body) == 0 {
				return toolset.NewToolResultError("body is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			path := "/api/n9e/fields"
			if input.Engine == "os" {
				path = "/api/n9e/os-fields"
			}
			result, err := client.DoPost[any](c, ctx, path, input.Body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}
