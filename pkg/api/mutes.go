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

// ListMutesInput represents alert mutes list query parameters
type ListMutesInput struct {
	GroupId int64 `json:"group_id"`
}

// GetMuteInput represents get single mute rule parameters
type GetMuteInput struct {
	GroupId int64 `json:"group_id"`
	MuteId  int64 `json:"mute_id"`
}

// CreateMuteInput represents create mute rule parameters
type CreateMuteInput struct {
	GroupId       int64               `json:"group_id"`
	Note          string              `json:"note"`
	Cate          string              `json:"cate,omitempty"`
	Prod          string              `json:"prod,omitempty"`
	DatasourceIds []int64             `json:"datasource_ids,omitempty"`
	Cluster       string              `json:"cluster,omitempty"`
	Tags          []types.TagFilter   `json:"tags,omitempty"`
	Cause         string              `json:"cause"`
	Btime         int64               `json:"btime"`
	Etime         int64               `json:"etime"`
	Severities    []int               `json:"severities,omitempty"`
	Disabled      int                 `json:"disabled,omitempty"`
	MuteTimeType  int                 `json:"mute_time_type,omitempty"`
	PeriodicMutes []types.PeriodicMute `json:"periodic_mutes,omitempty"`
}

// UpdateMuteInput represents update mute rule parameters
type UpdateMuteInput struct {
	GroupId       int64               `json:"group_id"`
	MuteId        int64               `json:"mute_id"`
	Note          string              `json:"note"`
	Cate          string              `json:"cate,omitempty"`
	Prod          string              `json:"prod,omitempty"`
	DatasourceIds []int64             `json:"datasource_ids,omitempty"`
	Cluster       string              `json:"cluster,omitempty"`
	Tags          []types.TagFilter   `json:"tags,omitempty"`
	Cause         string              `json:"cause"`
	Btime         int64               `json:"btime"`
	Etime         int64               `json:"etime"`
	Severities    []int               `json:"severities,omitempty"`
	Disabled      int                 `json:"disabled,omitempty"`
	MuteTimeType  int                 `json:"mute_time_type,omitempty"`
	PeriodicMutes []types.PeriodicMute `json:"periodic_mutes,omitempty"`
}

// RegisterMutesToolset registers alert mutes toolset
func RegisterMutesToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("mutes", "Alert mute/silence management tools")

	ts.AddReadTools(
		listMutesTool(getClient),
		getMuteTool(getClient),
	)

	ts.AddWriteTools(
		createMuteTool(getClient),
		updateMuteTool(getClient),
	)

	group.AddToolset(ts)
}

func listMutesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_mutes",
			Description: "List alert mutes/silences for a business group",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Alert Mutes",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {
						Type:        "integer",
						Description: "Business group ID",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListMutesInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-mutes", input.GroupId)
			result, err := client.DoGet[[]types.AlertMute](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getMuteTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_mute",
			Description: "Get details of a specific alert mute by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Alert Mute",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "mute_id"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {
						Type:        "integer",
						Description: "Business group ID",
					},
					"mute_id": {
						Type:        "integer",
						Description: "Alert mute ID",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetMuteInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}
			if input.MuteId <= 0 {
				return toolset.NewToolResultError("mute_id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-mute/%d", input.GroupId, input.MuteId)
			result, err := client.DoGet[types.AlertMute](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func createMuteTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_mute",
			Description: "Create a new alert mute/silence rule. Use mute_time_type=0 for time range mode (btime/etime), or mute_time_type=1 for periodic mode (periodic_mutes).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Alert Mute",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "cause", "btime", "etime"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {
						Type:        "integer",
						Description: "Business group ID",
					},
					"note": {
						Type:        "string",
						Description: "Note/title for the mute rule",
					},
					"cate": {
						Type:        "string",
						Description: "Category (e.g., prometheus, host, elasticsearch)",
					},
					"prod": {
						Type:        "string",
						Description: "Product type (e.g., metric, host, loki)",
					},
					"datasource_ids": {
						Type:        "array",
						Description: "Datasource IDs to match (empty means all)",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"cluster": {
						Type:        "string",
						Description: "Cluster name filter",
					},
					"tags": {
						Type:        "array",
						Description: "Tag filters. Each filter has key, func (==, !=, in, not in, =~, !~), and value",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"key":   {Type: "string", Description: "Tag key"},
								"func":  {Type: "string", Description: "Operator: ==, !=, in, not in, =~, !~"},
								"value": {Type: "string", Description: "Tag value (for 'in'/'not in', space-separated values)"},
							},
						},
					},
					"cause": {
						Type:        "string",
						Description: "Reason/description for the mute",
					},
					"btime": {
						Type:        "integer",
						Description: "Start time Unix timestamp",
					},
					"etime": {
						Type:        "integer",
						Description: "End time Unix timestamp",
					},
					"severities": {
						Type:        "array",
						Description: "Severity levels to match (1=critical, 2=warning, 3=info). Empty means all.",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"disabled": {
						Type:        "integer",
						Description: "Disabled status (0=enabled, 1=disabled)",
					},
					"mute_time_type": {
						Type:        "integer",
						Description: "Mute time type (0=time range, 1=periodic)",
					},
					"periodic_mutes": {
						Type:        "array",
						Description: "Periodic mute rules (when mute_time_type=1)",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"enable_stime":        {Type: "string", Description: "Start time in HH:MM format"},
								"enable_etime":        {Type: "string", Description: "End time in HH:MM format"},
								"enable_days_of_week": {Type: "string", Description: "Days of week (0-6, space-separated, 0=Sunday)"},
							},
						},
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input CreateMuteInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}
			if input.Cause == "" {
				return toolset.NewToolResultError("cause is required"), nil
			}
			if input.MuteTimeType == 0 {
				if input.Btime <= 0 || input.Etime <= 0 {
					return toolset.NewToolResultError("btime and etime are required for time range mode"), nil
				}
				if input.Btime >= input.Etime {
					return toolset.NewToolResultError("btime must be less than etime"), nil
				}
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			// Construct request body
			body := map[string]any{
				"note":           input.Note,
				"cate":           input.Cate,
				"prod":           input.Prod,
				"datasource_ids": input.DatasourceIds,
				"cluster":        input.Cluster,
				"tags":           input.Tags,
				"cause":          input.Cause,
				"btime":          input.Btime,
				"etime":          input.Etime,
				"severities":     input.Severities,
				"disabled":       input.Disabled,
				"mute_time_type": input.MuteTimeType,
				"periodic_mutes": input.PeriodicMutes,
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-mutes", input.GroupId)
			result, err := client.DoPost[int64](c, ctx, path, body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(map[string]any{
				"id":      result,
				"message": "Alert mute created successfully",
			}), nil
		}),
	)
}

func updateMuteTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_mute",
			Description: "Update an existing alert mute/silence rule",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Alert Mute",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "mute_id", "cause", "btime", "etime"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {
						Type:        "integer",
						Description: "Business group ID",
					},
					"mute_id": {
						Type:        "integer",
						Description: "Alert mute ID to update",
					},
					"note": {
						Type:        "string",
						Description: "Note/title for the mute rule",
					},
					"cate": {
						Type:        "string",
						Description: "Category (e.g., prometheus, host, elasticsearch)",
					},
					"prod": {
						Type:        "string",
						Description: "Product type (e.g., metric, host, loki)",
					},
					"datasource_ids": {
						Type:        "array",
						Description: "Datasource IDs to match (empty means all)",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"cluster": {
						Type:        "string",
						Description: "Cluster name filter",
					},
					"tags": {
						Type:        "array",
						Description: "Tag filters. Each filter has key, func (==, !=, in, not in, =~, !~), and value",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"key":   {Type: "string", Description: "Tag key"},
								"func":  {Type: "string", Description: "Operator: ==, !=, in, not in, =~, !~"},
								"value": {Type: "string", Description: "Tag value (for 'in'/'not in', space-separated values)"},
							},
						},
					},
					"cause": {
						Type:        "string",
						Description: "Reason/description for the mute",
					},
					"btime": {
						Type:        "integer",
						Description: "Start time Unix timestamp",
					},
					"etime": {
						Type:        "integer",
						Description: "End time Unix timestamp",
					},
					"severities": {
						Type:        "array",
						Description: "Severity levels to match (1=critical, 2=warning, 3=info). Empty means all.",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"disabled": {
						Type:        "integer",
						Description: "Disabled status (0=enabled, 1=disabled)",
					},
					"mute_time_type": {
						Type:        "integer",
						Description: "Mute time type (0=time range, 1=periodic)",
					},
					"periodic_mutes": {
						Type:        "array",
						Description: "Periodic mute rules (when mute_time_type=1)",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"enable_stime":        {Type: "string", Description: "Start time in HH:MM format"},
								"enable_etime":        {Type: "string", Description: "End time in HH:MM format"},
								"enable_days_of_week": {Type: "string", Description: "Days of week (0-6, space-separated, 0=Sunday)"},
							},
						},
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input UpdateMuteInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}
			if input.MuteId <= 0 {
				return toolset.NewToolResultError("mute_id is required and must be positive"), nil
			}
			if input.Cause == "" {
				return toolset.NewToolResultError("cause is required"), nil
			}
			if input.MuteTimeType == 0 {
				if input.Btime <= 0 || input.Etime <= 0 {
					return toolset.NewToolResultError("btime and etime are required for time range mode"), nil
				}
				if input.Btime >= input.Etime {
					return toolset.NewToolResultError("btime must be less than etime"), nil
				}
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			// Construct request body
			body := map[string]any{
				"note":           input.Note,
				"cate":           input.Cate,
				"prod":           input.Prod,
				"datasource_ids": input.DatasourceIds,
				"cluster":        input.Cluster,
				"tags":           input.Tags,
				"cause":          input.Cause,
				"btime":          input.Btime,
				"etime":          input.Etime,
				"severities":     input.Severities,
				"disabled":       input.Disabled,
				"mute_time_type": input.MuteTimeType,
				"periodic_mutes": input.PeriodicMutes,
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-mute/%d", input.GroupId, input.MuteId)
			_, err := client.DoPut[any](c, ctx, path, body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(map[string]any{
				"id":      input.MuteId,
				"message": "Alert mute updated successfully",
			}), nil
		}),
	)
}
