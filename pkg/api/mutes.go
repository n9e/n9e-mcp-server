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
	Limit   int   `json:"limit,omitempty"`
	Page    int   `json:"p,omitempty"`
}

// GetMuteInput represents get single mute rule parameters
type GetMuteInput struct {
	GroupId int64 `json:"group_id"`
	MuteId  int64 `json:"mute_id"`
}

// CreateMuteInput represents create mute rule parameters
type CreateMuteInput struct {
	GroupId       int64                `json:"group_id"`
	Note          string               `json:"note"`
	Cate          string               `json:"cate,omitempty"`
	Prod          string               `json:"prod,omitempty"`
	DatasourceIds []int64              `json:"datasource_ids,omitempty"`
	Cluster       string               `json:"cluster,omitempty"`
	Tags          []types.TagFilter    `json:"tags,omitempty"`
	Cause         string               `json:"cause"`
	Btime         int64                `json:"btime"`
	Etime         int64                `json:"etime"`
	Severities    []int                `json:"severities,omitempty"`
	Disabled      int                  `json:"disabled,omitempty"`
	MuteTimeType  int                  `json:"mute_time_type,omitempty"`
	PeriodicMutes []types.PeriodicMute `json:"periodic_mutes,omitempty"`
}

// UpdateMuteInput represents update mute rule parameters
type UpdateMuteInput struct {
	GroupId       int64                `json:"group_id"`
	MuteId        int64                `json:"mute_id"`
	Note          string               `json:"note"`
	Cate          string               `json:"cate,omitempty"`
	Prod          string               `json:"prod,omitempty"`
	DatasourceIds []int64              `json:"datasource_ids,omitempty"`
	Cluster       string               `json:"cluster,omitempty"`
	Tags          []types.TagFilter    `json:"tags,omitempty"`
	Cause         string               `json:"cause"`
	Btime         int64                `json:"btime"`
	Etime         int64                `json:"etime"`
	Severities    []int                `json:"severities,omitempty"`
	Disabled      int                  `json:"disabled,omitempty"`
	MuteTimeType  int                  `json:"mute_time_type,omitempty"`
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
			Description: "Find alert silences configured in one readable business group, including disabled or expired entries. Use get_mute to inspect a known mute or list_active_alerts to see firing events. This read-only lookup returns {list, total} with mute IDs, match criteria and schedules in descending ID order, paginated locally at 20 per page by default.",
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
						Description: "Positive business group ID from list_busi_groups.",
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

			items, total := toolset.SlicePage(result, input.Page, input.Limit)
			return toolset.MarshalResult(types.PageResp[types.AlertMute]{List: items, Total: total}), nil
		}),
	)
}

func getMuteTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_mute",
			Description: "Inspect the match criteria and schedule of one alert mute, including its tags, severities, reason and time windows. Use list_mutes to discover mute IDs within a business group. This read-only lookup requires the alert mutes permission and returns one mute object with its owning group and enabled state.",
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
						Description: "Positive owning business group ID from list_busi_groups or the mute's group_id in list_mutes.",
					},
					"mute_id": {
						Type:        "integer",
						Description: "Positive mute ID from list_mutes.",
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
			Description: "Save a silence for matching alerts in one business group. Use a fixed time range for one-off maintenance or periodic_mutes for a repeating schedule. Supply btime and etime in increasing order for server validation. Use update_mute for an existing silence. Returns the new mute ID.",
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
						Description: "Positive business group ID from list_busi_groups.",
					},
					"note": {
						Type:        "string",
						Description: "Optional short label for the mute.",
					},
					"cate": {
						Type:        "string",
						Description: "Category stored on the mute, such as prometheus or elasticsearch.",
					},
					"prod": {
						Type:        "string",
						Description: "Product stored on the mute, such as metric, host or loki.",
					},
					"datasource_ids": {
						Type:        "array",
						Description: "Array of datasource IDs from list_datasources. An empty array or an array containing 0 matches all datasources.",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"cluster": {
						Type:        "string",
						Description: "Cluster name stored on the mute configuration.",
					},
					"tags": {
						Type:        "array",
						Description: "Alert tag match conditions. Supply objects with key, func and value. Omit or pass an empty array for no tag conditions.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"key":   {Type: "string", Description: "Nonempty tag key from an alert's tags_map."},
								"func":  {Type: "string", Description: "Match operator. Use ==, !=, in, not in, =~ or !~."},
								"value": {Type: "string", Description: "Comparison value. Use a regular expression for =~ or !~ and space-separated values for in or not in."},
							},
						},
					},
					"cause": {
						Type:        "string",
						Description: "Required reason for the silence.",
					},
					"btime": {
						Type:        "integer",
						Description: "Start of the mute time range in Unix seconds. Set it before etime. Time range mode requires a positive value.",
					},
					"etime": {
						Type:        "integer",
						Description: "End of the mute time range in Unix seconds. Set it after btime. Time range mode requires a positive value.",
					},
					"severities": {
						Type:        "array",
						Description: "Severity levels to match. Use 1 for critical, 2 for warning, 3 for info or 0 for all. An empty array also matches all severities.",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"disabled": {
						Type:        "integer",
						Description: "Use 0 for enabled (default) or 1 for disabled.",
					},
					"mute_time_type": {
						Type:        "integer",
						Description: "Use 0 for a fixed time range (default) or 1 for a recurring schedule.",
					},
					"periodic_mutes": {
						Type:        "array",
						Description: "Recurring windows evaluated when mute_time_type is 1. Each window repeats weekly on chosen weekdays or monthly on chosen days of the month, between local clock times.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"cycle":                {Type: "string", Enum: []any{"week", "month"}, Description: "Use week (default) to repeat on enable_days_of_week, or month to repeat on enable_days_of_month."},
								"enable_stime":         {Type: "string", Description: "Start of the recurring window, in 24-hour HH:MM format in the Nightingale server's local time."},
								"enable_etime":         {Type: "string", Description: "End of the recurring window, in 24-hour HH:MM format in the Nightingale server's local time. An earlier end crosses midnight for weekly windows; monthly windows must not cross midnight, so split them into two windows instead."},
								"enable_days_of_week":  {Type: "string", Description: "Weekly windows only. Space-separated weekday numbers from 0 (Sunday) through 6 (Saturday)."},
								"enable_days_of_month": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}, Description: "Monthly windows only. Days of the month from 1 through 31, or -1 for the last day of every month. A day missing from a month, such as 31 in April, is skipped that month."},
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
			Description: "Replace an existing silence's matching criteria and schedule. Read its current configuration with get_mute before submitting the replacement, and use create_mute to add a separate silence. Nightingale checks write access to the owning business group. Returns the updated mute ID.",
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
						Description: "Positive business group ID from list_busi_groups.",
					},
					"mute_id": {
						Type:        "integer",
						Description: "Positive mute ID from list_mutes for the owning business group.",
					},
					"note": {
						Type:        "string",
						Description: "Optional short label for the mute.",
					},
					"cate": {
						Type:        "string",
						Description: "Category stored on the mute, such as prometheus or elasticsearch.",
					},
					"prod": {
						Type:        "string",
						Description: "Product stored on the mute, such as metric, host or loki.",
					},
					"datasource_ids": {
						Type:        "array",
						Description: "Array of datasource IDs from list_datasources. An empty array or an array containing 0 matches all datasources.",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"cluster": {
						Type:        "string",
						Description: "Cluster name stored on the mute configuration.",
					},
					"tags": {
						Type:        "array",
						Description: "Alert tag match conditions. Supply objects with key, func and value. Omit or pass an empty array for no tag conditions.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"key":   {Type: "string", Description: "Nonempty tag key from an alert's tags_map."},
								"func":  {Type: "string", Description: "Match operator. Use ==, !=, in, not in, =~ or !~."},
								"value": {Type: "string", Description: "Comparison value. Use a regular expression for =~ or !~ and space-separated values for in or not in."},
							},
						},
					},
					"cause": {
						Type:        "string",
						Description: "Required reason for the silence.",
					},
					"btime": {
						Type:        "integer",
						Description: "Start of the mute time range in Unix seconds. Set it before etime. Time range mode requires a positive value.",
					},
					"etime": {
						Type:        "integer",
						Description: "End of the mute time range in Unix seconds. Set it after btime. Time range mode requires a positive value.",
					},
					"severities": {
						Type:        "array",
						Description: "Severity levels to match. Use 1 for critical, 2 for warning, 3 for info or 0 for all. An empty array also matches all severities.",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"disabled": {
						Type:        "integer",
						Description: "Use 0 for enabled (default) or 1 for disabled.",
					},
					"mute_time_type": {
						Type:        "integer",
						Description: "Use 0 for a fixed time range (default) or 1 for a recurring schedule.",
					},
					"periodic_mutes": {
						Type:        "array",
						Description: "Recurring windows evaluated when mute_time_type is 1. Each window repeats weekly on chosen weekdays or monthly on chosen days of the month, between local clock times.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"cycle":                {Type: "string", Enum: []any{"week", "month"}, Description: "Use week (default) to repeat on enable_days_of_week, or month to repeat on enable_days_of_month."},
								"enable_stime":         {Type: "string", Description: "Start of the recurring window, in 24-hour HH:MM format in the Nightingale server's local time."},
								"enable_etime":         {Type: "string", Description: "End of the recurring window, in 24-hour HH:MM format in the Nightingale server's local time. An earlier end crosses midnight for weekly windows; monthly windows must not cross midnight, so split them into two windows instead."},
								"enable_days_of_week":  {Type: "string", Description: "Weekly windows only. Space-separated weekday numbers from 0 (Sunday) through 6 (Saturday)."},
								"enable_days_of_month": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}, Description: "Monthly windows only. Days of the month from 1 through 31, or -1 for the last day of every month. A day missing from a month, such as 31 in April, is skipped that month."},
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
