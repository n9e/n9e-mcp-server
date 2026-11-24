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
			Description: "Retrieve matching log entries from a Nightingale log datasource, such as Loki, Elasticsearch or OpenSearch. Use list_log_indices or list_log_fields to discover Elasticsearch or OpenSearch metadata. This read-only query returns data with log rows in list and engine-reported counts summed in total, or an error when no rows match. When both top-level timestamps are supplied, the tool rejects windows longer than seven days.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Query Logs",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body":  {Type: "object", Description: "Supply numeric datasource_id from list_datasources, cate matching its plugin_type and query as an array of engine-specific query objects. For Loki, use a query object with a LogQL query, start, end and limit. For Elasticsearch or OpenSearch search, use index, date_field, filter, start, end and limit. Use Unix seconds for these time bounds."},
					"limit": {Type: "integer", Description: "Optional top-level limit hint, default 200. Inserted into body only if body.limit is absent and echoed as limit in the response. Set the engine-specific row limit inside body.query to limit results."},
					"start": {Type: "integer", Description: "Optional positive Unix-second start used with end for the seven-day validation. Set the actual start in body.query."},
					"end":   {Type: "integer", Description: "Optional positive Unix-second end used with start for the seven-day validation. Set the actual end in body.query."},
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
			Description: "Discover available index names on an Elasticsearch or OpenSearch datasource. Use list_log_fields to inspect an index's mapped fields or query_logs to retrieve entries. This read-only lookup returns the engine's index names without pagination and errors if the datasource cannot be resolved.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Log Indices",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body":   {Type: "object", Description: "Supply numeric datasource_id from list_datasources and cate matching its plugin_type (elasticsearch or opensearch)."},
					"engine": {Type: "string", Description: "Use es for Elasticsearch (default) or os for OpenSearch. Match the engine to the datasource plugin."},
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
			Description: "Discover mapped field names for an Elasticsearch or OpenSearch index or pattern before building a log query. Use list_log_indices to find index names and query_logs to retrieve log entries. This read-only lookup returns a list of field names from the selected engine and errors if the datasource cannot be resolved.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Log Fields",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body":   {Type: "object", Description: "Supply numeric datasource_id from list_datasources and cate matching its plugin_type (elasticsearch or opensearch). Set index to an index name from list_log_indices or an index pattern."},
					"engine": {Type: "string", Description: "Use es for Elasticsearch (default) or os for OpenSearch. Match the engine to the datasource plugin."},
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
