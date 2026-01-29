package api

import (
	"context"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
	"github.com/n9e/n9e-mcp-server/pkg/types"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
			Description: "List all business groups that the current user has access to",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Business Groups",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			result, err := client.DoGet[[]types.BusiGroup](c, ctx, "/api/n9e/busi-groups", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}
