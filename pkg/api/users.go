package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
	"github.com/n9e/n9e-mcp-server/pkg/types"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListUsersInput represents users list query parameters
type ListUsersInput struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
	Page  int    `json:"p,omitempty"`
}

// GetUserInput represents single user query parameters
type GetUserInput struct {
	UserId int64 `json:"id"`
}

// ListUserGroupsInput represents user groups list query parameters
type ListUserGroupsInput struct {
	Query string `json:"query,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

// GetUserGroupInput represents single user group query parameters
type GetUserGroupInput struct {
	GroupId int64 `json:"id"`
}

// RegisterUsersToolset registers users and user groups toolset
func RegisterUsersToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("users", "User and user group management tools")

	ts.AddReadTools(
		listUsersTool(getClient),
		getUserTool(getClient),
		listUserGroupsTool(getClient),
		getUserGroupTool(getClient),
	)

	group.AddToolset(ts)
}

func listUsersTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_users",
			Description: "List users with optional filters",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Users",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"query": {
						Type:        "string",
						Description: "Search keyword (matches username/nickname/email/phone)",
					},
					"limit": {
						Type:        "integer",
						Description: "Page size (default 20)",
					},
					"p": {
						Type:        "integer",
						Description: "Page number (starts from 1)",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListUsersInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			params := url.Values{}
			if input.Query != "" {
				params.Set("query", input.Query)
			}
			if input.Limit > 0 {
				params.Set("limit", strconv.Itoa(input.Limit))
			}
			if input.Page > 0 {
				params.Set("p", strconv.Itoa(input.Page))
			}

			result, err := client.DoGet[types.PageResp[types.User]](c, ctx, "/api/n9e/users", params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getUserTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_user",
			Description: "Get details of a specific user by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get User",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "User ID",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetUserInput) (*mcp.CallToolResult, error) {
			if input.UserId <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/user/%d/profile", input.UserId)
			result, err := client.DoGet[types.User](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func listUserGroupsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_user_groups",
			Description: "List user groups/teams that the current user has access to",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List User Groups",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"query": {
						Type:        "string",
						Description: "Search keyword for group name",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of groups to return (default 1500)",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListUserGroupsInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			params := url.Values{}
			if input.Query != "" {
				params.Set("query", input.Query)
			}
			if input.Limit > 0 {
				params.Set("limit", strconv.Itoa(input.Limit))
			}

			result, err := client.DoGet[[]types.UserGroup](c, ctx, "/api/n9e/user-groups", params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getUserGroupTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_user_group",
			Description: "Get details of a specific user group including its members",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get User Group",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "User group ID",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetUserGroupInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/user-group/%d", input.GroupId)
			result, err := client.DoGet[types.UserGroupDetail](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}
