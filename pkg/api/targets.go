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
			Description: "Find registered monitoring targets by business group, host metadata or time since their last heartbeat. Use list_active_alerts to investigate firing incidents or query_instant to read a metric value. This read-only lookup returns {list, total} with target identifiers, tags and heartbeat status, ordered by ident ascending with 30 per page by default. Without a group filter, non-admin callers see their business groups and unassigned targets.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Targets",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"gids": {
						Type:        "string",
						Description: "Comma-separated business group IDs from list_busi_groups. Include 0 to match unassigned targets. Nightingale checks access to each positive group ID.",
					},
					"query": {
						Type:        "string",
						Description: "Space-separated terms matched against ident, host IP, note, tags, host tags or OS. Prefix a term with - to exclude matches.",
					},
					"limit": {
						Type:        "integer",
						Description: "Targets per page. Defaults to 30 when omitted or nonpositive.",
					},
					"p": {
						Type:        "integer",
						Description: "1-based page number. Defaults to 1.",
					},
					"downtime": {
						Type:        "integer",
						Description: "Positive threshold in seconds. Finds targets whose last heartbeat is older than the threshold. Omit or use 0 for no heartbeat filter. The tool omits negative values.",
					},
					"datasource_ids": {
						Type:        "string",
						Description: "Comma-separated datasource IDs from list_datasources. The tool forwards this parameter, but the Nightingale target list endpoint does not apply it.",
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
