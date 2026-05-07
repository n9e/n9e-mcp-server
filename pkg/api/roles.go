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
			Description: "List all roles defined in the system.",
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
			Description: "List every operation (permission) the system understands.",
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
			Description: "List operations bound to a specific role.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Role Operations",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer", Description: "Role ID"},
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
			Description: "Create a new role.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Role",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"role"},
				Properties: map[string]*jsonschema.Schema{
					"role": {Type: "object", Description: "Role body (name, note)"},
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
			Description: "Update a role's metadata (PUT /roles).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Role",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"role"},
				Properties: map[string]*jsonschema.Schema{
					"role": {Type: "object", Description: "Role body including id"},
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
			Description: "Replace the operations bound to a role (PUT /role/{id}/ops). Send the full ops list, not a delta.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Bind Role Operations",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "ops"},
				Properties: map[string]*jsonschema.Schema{
					"id":  {Type: "integer", Description: "Role ID"},
					"ops": {Type: "array", Items: &jsonschema.Schema{Type: "string"}, Description: "Operation names"},
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

