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

	ts.AddWriteTools(
		createUserTool(getClient),
		updateUserProfileTool(getClient),
		resetUserPasswordTool(getClient),
		createUserGroupTool(getClient),
		updateUserGroupTool(getClient),
		addUserGroupMembersTool(getClient),
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

// --- Write tools ---

type createUserInput struct {
	User map[string]any `json:"user"`
}

func createUserTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_user",
			Description: "Create a new user. The body should at minimum include username, password, and roles.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create User",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"user"},
				Properties: map[string]*jsonschema.Schema{
					"user": {Type: "object", Description: "User body (username, password, roles, email, phone, ...)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input createUserInput) (*mcp.CallToolResult, error) {
			if len(input.User) == 0 {
				return toolset.NewToolResultError("user body is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/users", input.User)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result, "message": "User created"}), nil
		}),
	)
}

type updateUserProfileInput struct {
	Id      int64          `json:"id"`
	Profile map[string]any `json:"profile"`
}

func updateUserProfileTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_user_profile",
			Description: "Update a user's profile. The 'roles' field on the body assigns roles to the user (n9e v8 stores roles on the user row, no dedicated assign endpoint).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update User Profile",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "profile"},
				Properties: map[string]*jsonschema.Schema{
					"id":      {Type: "integer", Description: "User ID"},
					"profile": {Type: "object", Description: "Profile body (nickname, email, phone, roles, ...)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateUserProfileInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/user/%d/profile", input.Id), input.Profile)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Id, "message": "Profile updated"}), nil
		}),
	)
}

type resetUserPasswordInput struct {
	Id       int64  `json:"id"`
	Password string `json:"password"`
}

func resetUserPasswordTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "reset_user_password",
			Description: "Reset a user's password. Requires admin privileges on the n9e side.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Reset User Password",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "password"},
				Properties: map[string]*jsonschema.Schema{
					"id":       {Type: "integer"},
					"password": {Type: "string", Description: "New password (will be hashed by n9e)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input resetUserPasswordInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 || input.Password == "" {
				return toolset.NewToolResultError("id and password are required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/user/%d/password", input.Id), map[string]any{"password": input.Password})
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Id, "message": "Password reset"}), nil
		}),
	)
}

type createUserGroupInput struct {
	Group map[string]any `json:"group"`
}

func createUserGroupTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_user_group",
			Description: "Create a new user group (team).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create User Group",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group"},
				Properties: map[string]*jsonschema.Schema{
					"group": {Type: "object", Description: "Group body (name, note, ...)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input createUserGroupInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/user-groups", input.Group)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result, "message": "User group created"}), nil
		}),
	)
}

type updateUserGroupInput struct {
	Id    int64          `json:"id"`
	Group map[string]any `json:"group"`
}

func updateUserGroupTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_user_group",
			Description: "Update a user group's metadata (name, note).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update User Group",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "group"},
				Properties: map[string]*jsonschema.Schema{
					"id":    {Type: "integer"},
					"group": {Type: "object"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateUserGroupInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 {
				return toolset.NewToolResultError("id is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/user-group/%d", input.Id), input.Group)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Id, "message": "Group updated"}), nil
		}),
	)
}

type userGroupMembersInput struct {
	Id  int64   `json:"id"`
	Ids []int64 `json:"ids"`
}

func addUserGroupMembersTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "add_user_group_members",
			Description: "Add user(s) to a user group.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Add User Group Members",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "ids"},
				Properties: map[string]*jsonschema.Schema{
					"id":  {Type: "integer", Description: "User group ID"},
					"ids": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}, Description: "User IDs to add"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input userGroupMembersInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 || len(input.Ids) == 0 {
				return toolset.NewToolResultError("id and ids are required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPost[any](c, ctx, fmt.Sprintf("/api/n9e/user-group/%d/members", input.Id), map[string]any{"ids": input.Ids})
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"added": len(input.Ids)}), nil
		}),
	)
}
