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

// ListAlertSubscribesInput represents alert subscriptions list query parameters
type ListAlertSubscribesInput struct {
	GroupId int64 `json:"group_id"`
	Limit   int   `json:"limit,omitempty"`
	Page    int   `json:"p,omitempty"`
}

// ListAlertSubscribesByGidsInput represents query alert subscriptions by business group IDs
type ListAlertSubscribesByGidsInput struct {
	Gids  string `json:"gids,omitempty"`
	Limit int    `json:"limit,omitempty"`
	Page  int    `json:"p,omitempty"`
}

// GetAlertSubscribeInput represents single alert subscription query parameters
type GetAlertSubscribeInput struct {
	SubscribeId int64 `json:"sid"`
}

// RegisterAlertSubscribesToolset registers alert subscriptions toolset
func RegisterAlertSubscribesToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("alert_subscribes", "Alert subscription management tools for event handling")

	ts.AddReadTools(
		listAlertSubscribesTool(getClient),
		listAlertSubscribesByGidsTool(getClient),
		getAlertSubscribeTool(getClient),
	)

	group.AddToolset(ts)
}

func listAlertSubscribesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_alert_subscribes",
			Description: "Find alert subscriptions configured in one readable business group, including which alert rules or tags they match and how they change notification routing. Use list_alert_subscribes_by_gids for a cross-group search or get_alert_subscribe to inspect a known subscription ID. This read-only lookup returns {list, total} with subscription IDs and matching criteria in descending ID order, paginated locally at 20 per page by default.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Alert Subscriptions",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {
						Type:        "integer",
						Description: "Positive business group ID from list_busi_groups.",
					},
					"limit": {
						Type:        "integer",
						Description: "Number of records per page. Defaults to 20.",
					},
					"p": {
						Type:        "integer",
						Description: "1-based page number. Defaults to 1.",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListAlertSubscribesInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-subscribes", input.GroupId)
			result, err := client.DoGet[[]types.AlertSubscribe](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			items, total := toolset.SlicePage(result, input.Page, input.Limit)
			return toolset.MarshalResult(types.PageResp[types.AlertSubscribe]{List: items, Total: total}), nil
		}),
	)
}

func listAlertSubscribesByGidsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_alert_subscribes_by_gids",
			Description: "Compare alert subscriptions across multiple business groups or search the caller's default group scope without choosing IDs. Use list_alert_subscribes for one known group or get_alert_subscribe for one known subscription. Nightingale checks access to every requested group and defaults to all groups for admins or the caller's business groups otherwise. This read-only lookup returns {list, total} with subscription IDs, matching criteria and notification settings, paginated locally in descending ID order at 20 per page by default.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Alert Subscriptions By Group IDs",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"gids": {
						Type:        "string",
						Description: "Comma-separated business group IDs from list_busi_groups. Omit for all groups as an admin or your own business groups otherwise.",
					},
					"limit": {
						Type:        "integer",
						Description: "Number of records per page. Defaults to 20.",
					},
					"p": {
						Type:        "integer",
						Description: "1-based page number. Defaults to 1.",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListAlertSubscribesByGidsInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			params := url.Values{}
			if input.Gids != "" {
				params.Set("gids", input.Gids)
			}

			result, err := client.DoGet[[]types.AlertSubscribe](c, ctx, "/api/n9e/busi-groups/alert-subscribes", params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			items, total := toolset.SlicePage(result, input.Page, input.Limit)
			return toolset.MarshalResult(types.PageResp[types.AlertSubscribe]{List: items, Total: total}), nil
		}),
	)
}

func getAlertSubscribeTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_alert_subscribe",
			Description: "Inspect how one alert subscription matches events and routes or overrides their notifications. Use list_alert_subscribes or list_alert_subscribes_by_gids to find its ID. This read-only lookup requires the alert subscriptions permission and returns one object with its match criteria, user groups and notification settings, or a not-found error.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Alert Subscription",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"sid"},
				Properties: map[string]*jsonschema.Schema{
					"sid": {
						Type:        "integer",
						Description: "Positive subscription ID from list_alert_subscribes or list_alert_subscribes_by_gids.",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetAlertSubscribeInput) (*mcp.CallToolResult, error) {
			if input.SubscribeId <= 0 {
				return toolset.NewToolResultError("sid is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/alert-subscribe/%d", input.SubscribeId)
			result, err := client.DoGet[types.AlertSubscribe](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}
