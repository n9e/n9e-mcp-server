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
			Description: "Search the Nightingale user directory to find account IDs, contact details and assigned roles. Use get_user for a known account ID or get_user_group for a team's member roster. This read-only lookup requires the users permission and returns {list, total}, ordered by username ascending, with 20 accounts per page by default.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Users",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"query": {
						Type:        "string",
						Description: "Optional substring search across username, nickname, email or phone. An empty string lists all accounts.",
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
			Description: "Look up one Nightingale account by numeric ID to inspect its profile, contact details and role names. Use list_users to search by name or contact details, or get_user_group to inspect team membership. This read-only lookup returns one profile for a signed-in caller.",
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
						Description: "Positive user ID from list_users.",
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
			Description: "Find Nightingale teams by name and obtain group IDs for membership or notification settings. Use get_user_group for the member roster or list_users to search individual accounts. This read-only lookup returns a list of group IDs, names and notes for all admin-visible matches, or groups the caller belongs to or created, with a default name-search limit of 1500.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List User Groups",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"query": {
						Type:        "string",
						Description: "Optional group-name substring. For admins, if no name matches, Nightingale also tries an exact username and returns that user's groups.",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of group-name matches to return, default 1500. The admin username fallback uses a separate lookup.",
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
			Description: "Inspect a team's membership by ID, including user IDs and profiles. Use list_user_groups to find the team or get_user to inspect a single account. This read-only lookup returns {user_group, users} for admins, the group creator or group members.",
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
						Description: "Positive user group ID from list_user_groups.",
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
			Description: "Create a Nightingale account with a password and explicit role assignments. Use update_user_profile to edit a saved account. Nightingale hashes the password before storage and requires a nonempty role list. Use list_users to find the new account ID.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create User",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"user"},
				Properties: map[string]*jsonschema.Schema{
					"user": {Type: "object", Description: "Account object with username, password and a nonempty roles array of names from list_roles. Optional fields include nickname, email, phone, portrait and contacts. Supply an RSA-encrypted password if the server requires it."},
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
			Description: "Replace an existing account's editable profile fields and role assignments. Read get_user first to carry forward the values you want to keep. Use reset_user_password to change credentials. This write returns the account ID and a confirmation.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update User Profile",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "profile"},
				Properties: map[string]*jsonschema.Schema{
					"id":      {Type: "integer", Description: "Positive user ID from list_users."},
					"profile": {Type: "object", Description: "Object with nickname, email, phone, contacts and a nonempty roles array of names from list_roles. Omitted profile fields take their zero values in the update."},
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
			Description: "Replace the password stored for an existing Nightingale account. Use update_user_profile for profile details or role assignments. This write requires the users edit permission and returns the account ID and a confirmation.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Reset User Password",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "password"},
				Properties: map[string]*jsonschema.Schema{
					"id":       {Type: "integer", Description: "Positive user ID from list_users."},
					"password": {Type: "string", Description: "Nonempty replacement password. Nightingale hashes it before storage. Supply it in the server's RSA-encrypted format when HTTP RSA is enabled."},
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
			Description: "Save a new user group for team membership and notification routing. Nightingale attempts to add the caller as a member and returns the new group ID in result. Use add_user_group_members to add other users or update_user_group to edit an existing group's name or note.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create User Group",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group"},
				Properties: map[string]*jsonschema.Schema{
					"group": {Type: "object", Description: "Team object with a name and optional note. Set is_sync_to_flashduty to true to request FlashDuty team synchronization."},
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
			Description: "Replace a team's saved name and note. Use add_user_group_members for a membership change or get_user_group to inspect the roster. Nightingale allows this write for admins, the group creator or group members and rejects a conflicting name.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update User Group",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "group"},
				Properties: map[string]*jsonschema.Schema{
					"id":    {Type: "integer", Description: "Positive user group ID from list_user_groups."},
					"group": {Type: "object", Description: "Object with the desired name and note. An omitted note is stored as empty. Set is_sync_to_flashduty to true to request FlashDuty team synchronization."},
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
			Description: "Add existing Nightingale accounts to a team. Use create_user for accounts that do not exist and get_user_group to verify membership. This write returns the submitted ID count as added.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Add User Group Members",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "ids"},
				Properties: map[string]*jsonschema.Schema{
					"id":  {Type: "integer", Description: "Positive user group ID from list_user_groups."},
					"ids": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}, Description: "Nonempty array of user IDs from list_users."},
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
