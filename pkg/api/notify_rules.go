package api

import (
	"context"
	"fmt"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
	"github.com/n9e/n9e-mcp-server/pkg/types"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListNotifyRulesInput represents notification rules list query parameters
type ListNotifyRulesInput struct{}

// GetNotifyRuleInput represents single notification rule query parameters
type GetNotifyRuleInput struct {
	RuleId int64 `json:"id"`
}

// RegisterNotifyRulesToolset registers notification rules toolset
func RegisterNotifyRulesToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("notify_rules", "Notification rule management tools")

	ts.AddReadTools(
		listNotifyRulesTool(getClient),
		getNotifyRuleTool(getClient),
	)

	group.AddToolset(ts)
}

func listNotifyRulesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_notify_rules",
			Description: "List all notification rules that the current user has access to",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Notification Rules",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListNotifyRulesInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			result, err := client.DoGet[[]types.NotifyRule](c, ctx, "/api/n9e/notify-rules", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getNotifyRuleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_notify_rule",
			Description: "Get details of a specific notification rule by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Notification Rule",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "Notification rule ID",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetNotifyRuleInput) (*mcp.CallToolResult, error) {
			if input.RuleId <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/notify-rule/%d", input.RuleId)
			result, err := client.DoGet[types.NotifyRule](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}
