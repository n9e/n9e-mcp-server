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
			Description: "Find dashboards by owning business group, name or tags and obtain their IDs. Use get_dashboard for one dashboard with its saved panels or get_dashboard_pure for the stored metadata row. This read-only lookup returns an unpaginated metadata array. Nightingale checks access to requested groups and defaults to the caller's groups, or all groups for admins.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Dashboards",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Positive business group ID from list_busi_groups. A positive value selects the single-group lookup and takes precedence over gids."},
					"gids":     {Type: "string", Description: "Comma-separated business group IDs from list_busi_groups. Used when group_id is omitted or nonpositive. Omit both for the caller's default group scope."},
					"query":    {Type: "string", Description: "Optional space-separated terms that match the dashboard name or tags. Prefix a term with - to exclude matching dashboards."},
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
			Description: "Load a dashboard as served to viewers, including its saved panel configuration in configs when present. Use this to inspect panels or back up their configuration, and use get_dashboard_pure for the stored dashboard row without loading the separate panel payload. This read-only lookup applies Nightingale's dashboard visibility rules and uses a 10 MiB response budget.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Dashboard",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid"},
				Properties: map[string]*jsonschema.Schema{
					"bid": {Type: "integer", Description: "Positive dashboard ID from list_dashboards. Nightingale tries the numeric value as an identifier before looking up the row ID."},
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
			Description: "Read the stored dashboard metadata row directly by numeric ID, including name, tags and visibility. Use get_dashboard for the viewer payload with saved panels. This authenticated, read-only lookup omits configs and the creator and updater names.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Dashboard (Full Panels)",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid"},
				Properties: map[string]*jsonschema.Schema{
					"bid": {Type: "integer", Description: "Positive dashboard ID from list_dashboards."},
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
			Description: "Save a dashboard in a writable business group, with metadata and optional panel configuration. Use clone_dashboard to copy a saved dashboard or update_dashboard_panels to replace the panels on an existing dashboard. Returns the created dashboard in result.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Dashboard",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "board"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Positive business group ID from list_busi_groups."},
					"board":    {Type: "object", Description: "Dashboard object with name and optional ident, tags and note as strings. Supply configs as a JSON-encoded string to save an initial panel configuration."},
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
			Description: "Replace the stored name, identifier, tags and note of a dashboard. Use update_dashboard_panels to change saved panels or set_dashboard_public to change sharing. Requires write access to the owning group and returns the dashboard ID on success.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Dashboard Meta",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid", "patch"},
				Properties: map[string]*jsonschema.Schema{
					"bid":   {Type: "integer", Description: "Positive dashboard ID from list_dashboards."},
					"patch": {Type: "object", Description: "Object with name, ident, tags and note as strings. Include values you want to keep because omitted fields are sent to storage as empty values."},
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
			Description: "Overwrite the saved panel configuration for one dashboard. Read the current configs with get_dashboard before editing, and use update_dashboard_meta for name or tags. Requires write access to the owning group and returns the dashboard ID on success.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Dashboard Panels",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid", "configs"},
				Properties: map[string]*jsonschema.Schema{
					"bid":     {Type: "integer", Description: "Positive dashboard ID from list_dashboards."},
					"configs": {Type: "string", Description: "Complete replacement panel configuration as a JSON-encoded string, in the format of configs from get_dashboard."},
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
			Description: "Change one dashboard's stored sharing and visibility settings. Use update_dashboard_meta for name or tags. This write returns the dashboard ID and requested public flag, and the tool supplies no business group sharing list.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Set Dashboard Public",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"bid", "public"},
				Properties: map[string]*jsonschema.Schema{
					"bid":         {Type: "integer", Description: "Positive dashboard ID from list_dashboards."},
					"public":      {Type: "integer", Description: "Use 0 for private or 1 for public."},
					"public_cate": {Type: "integer", Description: "Sharing audience. Use 0 for anonymous access (default), 1 for signed-in users or 2 for business groups. The tool has no input for the business group sharing list."},
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
			Description: "Save a copy of an existing dashboard and its panel payload in the source dashboard's business group. Use create_dashboard to start with new metadata and panels. Nightingale appends Cloned to the copy's name. Use list_dashboards to locate the copy after success.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Clone Dashboard",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "bid"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Positive business group ID from list_busi_groups. Supply the source dashboard's owning group."},
					"bid":      {Type: "integer", Description: "Positive dashboard ID from list_dashboards. Select the dashboard to copy."},
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
