package api

import (
	"context"
	"net/url"
	"strconv"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
	"github.com/n9e/n9e-mcp-server/pkg/types"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListTargetsInput represents monitored objects list query parameters
type ListTargetsInput struct {
	GroupIds      string `json:"gids,omitempty"`
	Query         string `json:"query,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Page          int    `json:"p,omitempty"`
	Downtime      int64  `json:"downtime,omitempty"`
	DatasourceIds string `json:"datasource_ids,omitempty"`
}

// RegisterTargetsToolset registers targets toolset
func RegisterTargetsToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("targets", "Target/Host management tools for viewing monitored objects")

	ts.AddReadTools(
		listTargetsTool(getClient),
	)

	group.AddToolset(ts)
}

func listTargetsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_targets",
			Description: "List monitored targets/hosts with optional filters",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Targets",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"gids": {
						Type:        "string",
						Description: "Business group IDs comma-separated",
					},
					"query": {
						Type:        "string",
						Description: "Search keyword (matches ident/tags)",
					},
					"limit": {
						Type:        "integer",
						Description: "Page size (default 20)",
					},
					"p": {
						Type:        "integer",
						Description: "Page number (starts from 1)",
					},
					"downtime": {
						Type:        "integer",
						Description: "Filter by downtime in seconds (targets not reporting for this duration)",
					},
					"datasource_ids": {
						Type:        "string",
						Description: "Datasource IDs comma-separated",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListTargetsInput) (*mcp.CallToolResult, error) {
			if err := toolset.ValidatePagination(input.Limit, input.Page); err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			params := url.Values{}
			if input.GroupIds != "" {
				params.Set("gids", input.GroupIds)
			}
			if input.Query != "" {
				params.Set("query", input.Query)
			}
			if input.Limit > 0 {
				params.Set("limit", strconv.Itoa(input.Limit))
			}
			if input.Page > 0 {
				params.Set("p", strconv.Itoa(input.Page))
			}
			if input.Downtime > 0 {
				params.Set("downtime", strconv.FormatInt(input.Downtime, 10))
			}
			if input.DatasourceIds != "" {
				params.Set("datasource_ids", input.DatasourceIds)
			}

			result, err := client.DoGet[types.PageResp[types.Target]](c, ctx, "/api/n9e/targets", params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}
