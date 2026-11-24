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
			Description: "Discover datasource IDs and plugin types for metric or log queries using Nightingale's redacted catalog. Use list_datasources_full to inspect connection configuration across sources or get_datasource for one source's admin configuration. This read-only lookup returns {list, total} in descending ID order, paginated locally at 20 per page by default, with datasource visibility determined by Nightingale.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Datasources",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"limit": {Type: "integer", Description: "Number of records per page. Defaults to 20."},
					"p":     {Type: "integer", Description: "1-based page number. Defaults to 1."},
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
			Description: "Inspect connection and plugin configuration across datasources, optionally filtered by plugin type or category. Use list_datasources for redacted ID discovery or get_datasource for one known source's admin configuration. This read-only lookup returns an unpaginated configuration array in descending ID order after server-side visibility filtering. Nightingale redacts secrets for non-admin callers and may expose credentials to admins.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Datasources (Full)",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"body": {Type: "object", Description: "Optional filter object. Omit for an unfiltered request. Use plugin_type and category values from list_datasource_plugins. This Nightingale source compares name for equality after adding percent signs, so leave name unset for ordinary names and filter the returned array."},
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
			Description: "Inspect one datasource's saved connection, authentication and plugin settings by ID. Use list_datasources for redacted discovery or list_datasources_full to inspect configurations across multiple sources. This read-only lookup requires Nightingale admin access and returns a configuration object that can contain credentials.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Datasource",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer", Description: "Positive datasource ID from list_datasources."},
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
			Description: "Discover the datasource plugin types configured in Nightingale before filtering sources or choosing a plugin for upsert_datasource. Use list_datasources to find existing datasource instances. This read-only lookup returns an unpaginated catalog with plugin IDs, categories, type keys and display names.",
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
			Description: "Save a new datasource or update an existing datasource's connection settings as a Nightingale admin. Use set_datasource_status for a status-only change. Nightingale may test connectivity depending on plugin and force_save. Inspect result for the datasource ID and verification status.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Upsert Datasource",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"datasource"},
				Properties: map[string]*jsonschema.Schema{
					"datasource": {Type: "object", Description: "Datasource object with name, plugin_type, category and plugin-specific settings, http and auth. Omit id to create or use an ID from list_datasources to update. Use get_datasource for the existing shape and list_datasource_plugins for plugin keys. Set force_save to true to skip normal connectivity checks where supported."},
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
			Description: "Submit an admin request to set datasource status to enabled or disabled. Use upsert_datasource to edit connection settings. This tool sends an ids array, so Nightingale versions that require a single id do not support this batch request format.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Set Datasource Status",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"ids", "status"},
				Properties: map[string]*jsonschema.Schema{
					"ids":    {Type: "array", Items: &jsonschema.Schema{Type: "integer"}, Description: "Nonempty array of datasource IDs from list_datasources."},
					"status": {Type: "string", Description: "Use enabled to activate or disabled to deactivate the selected sources."},
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
