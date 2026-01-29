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
}

// ListAlertSubscribesByGidsInput represents query alert subscriptions by business group IDs
type ListAlertSubscribesByGidsInput struct {
	Gids string `json:"gids,omitempty"`
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
			Description: "List alert subscriptions for a business group",
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
						Description: "Business group ID",
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

			return toolset.MarshalResult(result), nil
		}),
	)
}

func listAlertSubscribesByGidsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_alert_subscribes_by_gids",
			Description: "List alert subscriptions across multiple business groups",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Alert Subscriptions By Group IDs",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"gids": {
						Type:        "string",
						Description: "Business group IDs comma-separated (empty for all accessible groups)",
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

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getAlertSubscribeTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_alert_subscribe",
			Description: "Get details of a specific alert subscription by ID",
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
						Description: "Alert subscription ID",
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
