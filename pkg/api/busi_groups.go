package api

import (
	"context"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
	"github.com/n9e/n9e-mcp-server/pkg/types"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListBusiGroupsInput represents business groups list query parameters
type ListBusiGroupsInput struct {
	Limit int `json:"limit,omitempty"`
	Page  int `json:"p,omitempty"`
}

// RegisterBusiGroupsToolset registers business groups toolset
func RegisterBusiGroupsToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("busi_groups", "Business group management tools")

	ts.AddReadTools(
		listBusiGroupsTool(getClient),
	)

	group.AddToolset(ts)
}

func listBusiGroupsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_busi_groups",
			Description: "Discover business groups that own monitoring targets, alert rules, mutes, subscriptions and dashboards. Use the returned IDs with tools such as list_alert_rules or use list_user_groups for user teams. This read-only lookup returns {list, total} with IDs and names in name order, paginated locally at 20 per page by default. Nightingale supplies at most 300 groups and scopes non-admin results to their associated groups.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Business Groups",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
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
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListBusiGroupsInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			result, err := client.DoGet[[]types.BusiGroup](c, ctx, "/api/n9e/busi-groups", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			items, total := toolset.SlicePage(result, input.Page, input.Limit)
			return toolset.MarshalResult(types.PageResp[types.BusiGroup]{List: items, Total: total}), nil
		}),
	)
}
