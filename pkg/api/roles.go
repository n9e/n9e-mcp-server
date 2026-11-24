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

// RegisterRolesToolset registers the roles & operations (RBAC) toolset.
//
// Note: in n9e v8 there is no dedicated "assign role to user" endpoint —
// the user's roles live on the users.roles column and are updated via
// update_user_profile (see users toolset).
func RegisterRolesToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("roles", "Role and permission management (n9e RBAC)")

	ts.AddReadTools(
		listRolesTool(getClient),
		listOperationsTool(getClient),
		listRoleOperationsTool(getClient),
	)

	ts.AddWriteTools(
		createRoleTool(getClient),
		updateRoleTool(getClient),
		bindRoleOperationsTool(getClient),
	)

	group.AddToolset(ts)
}

func listRolesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_roles",
			Description: "Discover Nightingale RBAC roles and obtain names for user profile assignments or IDs for permission lookup. Use list_role_operations to inspect one role's grants or list_operations to browse available permissions. This read-only lookup returns an unpaginated array of role IDs, names and notes for signed-in callers.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Roles",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{Type: "object"},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[[]types.Role](c, ctx, "/api/n9e/roles", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

func listOperationsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_operations",
			Description: "Browse the configured permission operations available for role assignment. Use list_role_operations to see the permission names assigned to one role or list_roles to discover role IDs. This read-only lookup returns an unpaginated array of groups with nested ops entries containing operation names and display labels.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Operations",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{Type: "object"},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[any](c, ctx, "/api/n9e/operation", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

type roleByIDInput struct {
	Id int64 `json:"id"`
}

func listRoleOperationsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_role_operations",
			Description: "Inspect the permission operation names assigned to one role. Use list_operations for the grouped catalog of available permissions or list_roles to discover role IDs. This read-only lookup requires the roles permission and returns a flat list of operation names, with the full configured operation set for the built-in Admin role.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Role Operations",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer", Description: "Positive role ID from list_roles."},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input roleByIDInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 {
				return toolset.NewToolResultError("id is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[any](c, ctx, fmt.Sprintf("/api/n9e/role/%d/ops", input.Id), nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

type createRoleInput struct {
	Role map[string]any `json:"role"`
}

func createRoleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_role",
			Description: "Add a named RBAC role to Nightingale. Use update_role to rename an existing role or bind_role_operations to change its permissions. Nightingale rejects a duplicate role name. Use list_roles to look up the new role ID.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Role",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"role"},
				Properties: map[string]*jsonschema.Schema{
					"role": {Type: "object", Description: "Role object with a unique name and optional note."},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input createRoleInput) (*mcp.CallToolResult, error) {
			if len(input.Role) == 0 {
				return toolset.NewToolResultError("role body is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/roles", input.Role)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result, "message": "Role created"}), nil
		}),
	)
}

type updateRoleInput struct {
	Role map[string]any `json:"role"`
}

func updateRoleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_role",
			Description: "Replace the stored name and note for a role. Use bind_role_operations to change operation grants or create_role to add a new role. Nightingale rejects changes to the built-in Admin role and duplicate names.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Role",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"role"},
				Properties: map[string]*jsonschema.Schema{
					"role": {Type: "object", Description: "Role object with id from list_roles, the desired name and the note to store."},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateRoleInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, "/api/n9e/roles", input.Role)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"message": "Role updated"}), nil
		}),
	)
}

type bindRoleOperationsInput struct {
	Id  int64    `json:"id"`
	Ops []string `json:"ops"`
}

func bindRoleOperationsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "bind_role_operations",
			Description: "Submit a replacement set of permission grants for a role. Use list_role_operations to read current grants and list_operations to find valid grant names. This tool sends an object containing ops, which Nightingale versions that require a top-level array will reject.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Bind Role Operations",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "ops"},
				Properties: map[string]*jsonschema.Schema{
					"id":  {Type: "integer", Description: "Positive role ID from list_roles."},
					"ops": {Type: "array", Items: &jsonschema.Schema{Type: "string"}, Description: "Complete replacement array of operation names from list_operations."},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input bindRoleOperationsInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 {
				return toolset.NewToolResultError("id is required"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/role/%d/ops", input.Id), map[string]any{"ops": input.Ops})
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Id, "ops_count": len(input.Ops)}), nil
		}),
	)
}
