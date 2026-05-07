package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
	"github.com/n9e/n9e-mcp-server/pkg/types"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// dashboardMaxResponseSize allows board configs to exceed the default 10MB cap
// when fetching panel JSON via /board/{bid}/pure.
const dashboardMaxResponseSize = 64 * 1024 * 1024

// RegisterDashboardsToolset registers the dashboards (boards) toolset.
func RegisterDashboardsToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("dashboards", "Dashboard (board) management: list/get/create/update/clone")

	ts.AddReadTools(
		listDashboardsTool(getClient),
		getDashboardTool(getClient),
		getDashboardPureTool(getClient),
	)

	ts.AddWriteTools(
		createDashboardTool(getClient),
		updateDashboardMetaTool(getClient),
		updateDashboardPanelsTool(getClient),
		setDashboardPublicTool(getClient),
		cloneDashboardTool(getClient),
	)

	group.AddToolset(ts)
}

// --- Read ---

type listDashboardsInput struct {
	GroupId int64  `json:"group_id,omitempty"`
	Gids    string `json:"gids,omitempty"`
	Query   string `json:"query,omitempty"`
}

func listDashboardsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_dashboards",
			Description: "List dashboards. Pass group_id for a single business group, or gids (comma-separated) to query multiple.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Dashboards",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Business group ID (single)"},
					"gids":     {Type: "string", Description: "Comma-separated business group IDs (multi)"},
					"query":    {Type: "string", Description: "Optional name search"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input listDashboardsInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			params := url.Values{}
			if input.Query != "" {
				params.Set("query", input.Query)
			}

			if input.GroupId > 0 {
				path := fmt.Sprintf("/api/n9e/busi-group/%d/boards", input.GroupId)
				result, err := client.DoGet[any](c, ctx, path, params)
				if err != nil {
					return toolset.NewToolResultError(err.Error()), nil
				}
				return toolset.MarshalResult(result), nil
			}

			if input.Gids != "" {
				params.Set("gids", input.Gids)
			}
			result, err := client.DoGet[any](c, ctx, "/api/n9e/busi-groups/boards", params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

type getDashboardInput struct {
	Bid int64 `json:"bid"`
}

func getDashboardTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_dashboard",
			Description: "Get a dashboard's metadata by ID. Use get_dashboard_pure if you also need the panel configs.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Dashboard",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid"},
				Properties: map[string]*jsonschema.Schema{
					"bid": {Type: "integer", Description: "Board ID"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input getDashboardInput) (*mcp.CallToolResult, error) {
			if input.Bid <= 0 {
				return toolset.NewToolResultError("bid is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[types.Board](c, ctx, fmt.Sprintf("/api/n9e/board/%d", input.Bid), nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

func getDashboardPureTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_dashboard_pure",
			Description: "Get a dashboard including its full panel JSON. Uses an extended response size cap (64MB) — large boards return their entire layout.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Dashboard (Full Panels)",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid"},
				Properties: map[string]*jsonschema.Schema{
					"bid": {Type: "integer", Description: "Board ID"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input getDashboardInput) (*mcp.CallToolResult, error) {
			if input.Bid <= 0 {
				return toolset.NewToolResultError("bid is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGetLarge[types.Board](c, ctx, fmt.Sprintf("/api/n9e/board/%d/pure", input.Bid), nil, dashboardMaxResponseSize)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

// --- Write ---

type createDashboardInput struct {
	GroupId int64          `json:"group_id"`
	Board   map[string]any `json:"board"`
}

func createDashboardTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_dashboard",
			Description: "Create a new dashboard inside a business group.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Dashboard",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "board"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer"},
					"board":    {Type: "object", Description: "Board body (name, ident, tags, configs, ...)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input createDashboardInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			path := fmt.Sprintf("/api/n9e/busi-group/%d/boards", input.GroupId)
			result, err := client.DoPost[any](c, ctx, path, input.Board)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result, "message": "Dashboard created"}), nil
		}),
	)
}

type updateDashboardMetaInput struct {
	Bid   int64          `json:"bid"`
	Patch map[string]any `json:"patch"`
}

func updateDashboardMetaTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_dashboard_meta",
			Description: "Update a dashboard's name and tags (PUT /board/{bid}). Use update_dashboard_panels to change the panel JSON.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Dashboard Meta",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid", "patch"},
				Properties: map[string]*jsonschema.Schema{
					"bid":   {Type: "integer"},
					"patch": {Type: "object", Description: "Fields to update (name, tags, ident)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateDashboardMetaInput) (*mcp.CallToolResult, error) {
			if input.Bid <= 0 {
				return toolset.NewToolResultError("bid is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/board/%d", input.Bid), input.Patch)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Bid, "message": "Dashboard meta updated"}), nil
		}),
	)
}

type updateDashboardPanelsInput struct {
	Bid     int64  `json:"bid"`
	Configs string `json:"configs"`
}

func updateDashboardPanelsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_dashboard_panels",
			Description: "Replace a dashboard's panel JSON (PUT /board/{bid}/configs). 'configs' must be a JSON string.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Dashboard Panels",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid", "configs"},
				Properties: map[string]*jsonschema.Schema{
					"bid":     {Type: "integer"},
					"configs": {Type: "string", Description: "Panel JSON as a string (n9e stores it as text)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateDashboardPanelsInput) (*mcp.CallToolResult, error) {
			if input.Bid <= 0 {
				return toolset.NewToolResultError("bid is required"), nil
			}
			if input.Configs == "" {
				return toolset.NewToolResultError("configs is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/board/%d/configs", input.Bid), map[string]any{"configs": input.Configs})
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Bid, "message": "Dashboard panels updated"}), nil
		}),
	)
}

type setDashboardPublicInput struct {
	Bid        int64 `json:"bid"`
	Public     int   `json:"public"`
	PublicCate int   `json:"public_cate,omitempty"`
}

func setDashboardPublicTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "set_dashboard_public",
			Description: "Toggle a dashboard's public visibility (PUT /board/{bid}/public).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Set Dashboard Public",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid", "public"},
				Properties: map[string]*jsonschema.Schema{
					"bid":         {Type: "integer"},
					"public":      {Type: "integer", Description: "0 = private, 1 = public"},
					"public_cate": {Type: "integer", Description: "Visibility category (n9e enum)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input setDashboardPublicInput) (*mcp.CallToolResult, error) {
			if input.Bid <= 0 {
				return toolset.NewToolResultError("bid is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			body := map[string]any{"public": input.Public}
			if input.PublicCate != 0 {
				body["public_cate"] = input.PublicCate
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/board/%d/public", input.Bid), body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Bid, "public": input.Public}), nil
		}),
	)
}

type cloneDashboardInput struct {
	GroupId int64 `json:"group_id"`
	Bid     int64 `json:"bid"`
}

func cloneDashboardTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "clone_dashboard",
			Description: "Clone a dashboard into the same business group (POST /busi-group/{gid}/board/{bid}/clone).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Clone Dashboard",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "bid"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer"},
					"bid":      {Type: "integer"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input cloneDashboardInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 || input.Bid <= 0 {
				return toolset.NewToolResultError("group_id and bid are required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			path := fmt.Sprintf("/api/n9e/busi-group/%d/board/%d/clone", input.GroupId, input.Bid)
			result, err := client.DoPost[any](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result}), nil
		}),
	)
}
