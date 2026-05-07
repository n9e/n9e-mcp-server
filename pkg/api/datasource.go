package api

import (
	"context"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
	"github.com/n9e/n9e-mcp-server/pkg/types"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListDatasourcesInput represents datasources list query parameters
type ListDatasourcesInput struct {
	Limit int `json:"limit,omitempty"`
	Page  int `json:"p,omitempty"`
}

// RegisterDatasourceToolset registers datasource toolset.
//
// Note: n9e's datasource API is unusual — endpoints are POST even for reads,
// and create/update share /datasource/upsert (which also runs the connectivity
// check). There is no separate test-connection endpoint.
func RegisterDatasourceToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("datasource", "Datasource management tools (Prometheus/VictoriaMetrics/Loki/ES/...)")

	ts.AddReadTools(
		listDatasourcesTool(getClient),
		listDatasourcesFullTool(getClient),
		getDatasourceTool(getClient),
		listDatasourcePluginsTool(getClient),
	)

	ts.AddWriteTools(
		upsertDatasourceTool(getClient),
		setDatasourceStatusTool(getClient),
	)

	group.AddToolset(ts)
}

func listDatasourcesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_datasources",
			Description: "List all available datasources (sanitized brief view — no auth secrets)",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Datasources",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"limit": {Type: "integer", Description: "Page size (default 20)"},
					"p":     {Type: "integer", Description: "Page number (starts from 1)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListDatasourcesInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			result, err := client.DoGet[[]types.Datasource](c, ctx, "/api/n9e/datasource/brief", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			items, total := toolset.SlicePage(result, input.Page, input.Limit)
			return toolset.MarshalResult(types.PageResp[types.Datasource]{List: items, Total: total}), nil
		}),
	)
}

type listDatasourcesFullInput struct {
	Body map[string]any `json:"body,omitempty"`
}

func listDatasourcesFullTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_datasources_full",
			Description: "List datasources with full configuration including auth (admin perspective). Pass an optional filter body matching n9e's /datasource/list payload.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Datasources (Full)",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"body": {Type: "object", Description: "Optional filter body (e.g. plugin types, name search)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input listDatasourcesFullInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			body := input.Body
			if body == nil {
				body = map[string]any{}
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/datasource/list", body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

type getDatasourceInput struct {
	Id int64 `json:"id"`
}

func getDatasourceTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_datasource",
			Description: "Get full configuration of a single datasource by ID (POST /datasource/desc).",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Datasource",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer", Description: "Datasource ID"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input getDatasourceInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/datasource/desc", map[string]any{"id": input.Id})
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

func listDatasourcePluginsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_datasource_plugins",
			Description: "List supported datasource plugin types (Prometheus, Loki, ES, Tencent CLS, ...).",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Datasource Plugins",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{Type: "object"},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/datasource/plugin/list", map[string]any{})
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

type upsertDatasourceInput struct {
	Datasource map[string]any `json:"datasource"`
}

func upsertDatasourceTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "upsert_datasource",
			Description: "Create or update a datasource. n9e runs a connectivity check as part of upsert — a successful response implies the datasource is reachable. Omit 'id' to create.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Upsert Datasource",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"datasource"},
				Properties: map[string]*jsonschema.Schema{
					"datasource": {Type: "object", Description: "Datasource body (matches the shape returned by get_datasource)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input upsertDatasourceInput) (*mcp.CallToolResult, error) {
			if len(input.Datasource) == 0 {
				return toolset.NewToolResultError("datasource body is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/datasource/upsert", input.Datasource)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result}), nil
		}),
	)
}

type setDatasourceStatusInput struct {
	Ids    []int64 `json:"ids"`
	Status string  `json:"status"` // "enabled" or "disabled"
}

func setDatasourceStatusTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "set_datasource_status",
			Description: "Enable or disable one or more datasources (POST /datasource/status/update).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Set Datasource Status",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"ids", "status"},
				Properties: map[string]*jsonschema.Schema{
					"ids":    {Type: "array", Items: &jsonschema.Schema{Type: "integer"}, Description: "Datasource IDs"},
					"status": {Type: "string", Description: "'enabled' or 'disabled'"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input setDatasourceStatusInput) (*mcp.CallToolResult, error) {
			if len(input.Ids) == 0 {
				return toolset.NewToolResultError("ids must be non-empty"), nil
			}
			if input.Status != "enabled" && input.Status != "disabled" {
				return toolset.NewToolResultError("status must be 'enabled' or 'disabled'"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			body := map[string]any{"ids": input.Ids, "status": input.Status}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/datasource/status/update", body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result}), nil
		}),
	)
}

