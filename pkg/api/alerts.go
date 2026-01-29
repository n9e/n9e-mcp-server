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

// ListActiveAlertsInput represents active alerts query parameters
type ListActiveAlertsInput struct {
	Hours         int64  `json:"hours,omitempty"`
	Stime         int64  `json:"stime,omitempty"`
	Etime         int64  `json:"etime,omitempty"`
	Severity      string `json:"severity,omitempty"`
	Query         string `json:"query,omitempty"`
	Cate          string `json:"cate,omitempty"`
	RuleProds     string `json:"rule_prods,omitempty"`
	DatasourceIds string `json:"datasource_ids,omitempty"`
	RuleId        int64  `json:"rid,omitempty"`
	EventIds      string `json:"event_ids,omitempty"`
	BusiGroupId   int64  `json:"bgid,omitempty"`
	MyGroups      bool   `json:"my_groups,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Page          int    `json:"p,omitempty"`
}

// ListHistoryAlertsInput represents historical alerts query parameters
type ListHistoryAlertsInput struct {
	Hours         int64  `json:"hours,omitempty"`
	Stime         int64  `json:"stime,omitempty"`
	Etime         int64  `json:"etime,omitempty"`
	Severity      int    `json:"severity,omitempty"`
	IsRecovered   int    `json:"is_recovered,omitempty"`
	Query         string `json:"query,omitempty"`
	Cate          string `json:"cate,omitempty"`
	RuleProds     string `json:"rule_prods,omitempty"`
	DatasourceIds string `json:"datasource_ids,omitempty"`
	BusiGroupId   int64  `json:"bgid,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Page          int    `json:"p,omitempty"`
}

// GetAlertInput represents single alert query parameters
type GetAlertInput struct {
	EventId int64 `json:"eid"`
}

// ListAlertRulesInput represents alert rules list query parameters
type ListAlertRulesInput struct {
	GroupId int64 `json:"group_id"`
}

// GetAlertRuleInput represents single alert rule query parameters
type GetAlertRuleInput struct {
	RuleId int64 `json:"arid"`
}

// RegisterAlertsToolset registers alerts toolset
func RegisterAlertsToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("alerts", "Alert management tools for viewing and managing alerts")

	// Read-only tools
	ts.AddReadTools(
		listActiveAlertsTool(getClient),
		getActiveAlertTool(getClient),
		listHistoryAlertsTool(getClient),
		getHistoryAlertTool(getClient),
		listAlertRulesTool(getClient),
		getAlertRuleTool(getClient),
	)

	group.AddToolset(ts)
}

func listActiveAlertsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_active_alerts",
			Description: "List active alert events with optional filters. Use this to view currently firing alerts.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Active Alerts",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"hours": {
						Type:        "integer",
						Description: "Lookback hours (mutually exclusive with stime/etime)",
					},
					"stime": {
						Type:        "integer",
						Description: "Start time Unix timestamp",
					},
					"etime": {
						Type:        "integer",
						Description: "End time Unix timestamp",
					},
					"severity": {
						Type:        "string",
						Description: "Severity levels comma-separated (1=critical, 2=warning, 3=info)",
					},
					"query": {
						Type:        "string",
						Description: "Search keyword (matches rule name/tags)",
					},
					"cate": {
						Type:        "string",
						Description: "Alert category (prometheus/host/elasticsearch, default $all)",
					},
					"rule_prods": {
						Type:        "string",
						Description: "Product types comma-separated (host/metric/loki/anomaly)",
					},
					"datasource_ids": {
						Type:        "string",
						Description: "Datasource IDs comma-separated",
					},
					"rid": {
						Type:        "integer",
						Description: "Alert rule ID",
					},
					"bgid": {
						Type:        "integer",
						Description: "Business group ID",
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
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListActiveAlertsInput) (*mcp.CallToolResult, error) {
			// Parameter validation
			if err := toolset.ValidateTimeRange(input.Hours, input.Stime, input.Etime); err != nil {
				return toolset.NewToolResultError(fmt.Sprintf("invalid input: %v", err)), nil
			}
			if err := toolset.ValidateSeverity(input.Severity); err != nil {
				return toolset.NewToolResultError(fmt.Sprintf("invalid input: %v", err)), nil
			}
			if err := toolset.ValidatePagination(input.Limit, input.Page); err != nil {
				return toolset.NewToolResultError(fmt.Sprintf("invalid input: %v", err)), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			// Build query parameters
			params := url.Values{}
			if input.Hours > 0 {
				params.Set("hours", strconv.FormatInt(input.Hours, 10))
			}
			if input.Stime > 0 {
				params.Set("stime", strconv.FormatInt(input.Stime, 10))
			}
			if input.Etime > 0 {
				params.Set("etime", strconv.FormatInt(input.Etime, 10))
			}
			if input.Severity != "" {
				params.Set("severity", input.Severity)
			}
			if input.Query != "" {
				params.Set("query", input.Query)
			}
			if input.Cate != "" {
				params.Set("cate", input.Cate)
			}
			if input.RuleProds != "" {
				params.Set("rule_prods", input.RuleProds)
			}
			if input.DatasourceIds != "" {
				params.Set("datasource_ids", input.DatasourceIds)
			}
			if input.RuleId > 0 {
				params.Set("rid", strconv.FormatInt(input.RuleId, 10))
			}
			if input.BusiGroupId > 0 {
				params.Set("bgid", strconv.FormatInt(input.BusiGroupId, 10))
			}
			if input.Limit > 0 {
				params.Set("limit", strconv.Itoa(input.Limit))
			}
			if input.Page > 0 {
				params.Set("p", strconv.Itoa(input.Page))
			}

			result, err := client.DoGet[types.PageResp[types.AlertCurEvent]](c, ctx, "/api/n9e/alert-cur-events/list", params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getActiveAlertTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_active_alert",
			Description: "Get details of a specific active alert event by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Active Alert",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"eid"},
				Properties: map[string]*jsonschema.Schema{
					"eid": {
						Type:        "integer",
						Description: "Alert event ID",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetAlertInput) (*mcp.CallToolResult, error) {
			if input.EventId <= 0 {
				return toolset.NewToolResultError("eid is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/alert-cur-event/%d", input.EventId)
			result, err := client.DoGet[types.AlertCurEvent](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func listHistoryAlertsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_history_alerts",
			Description: "List historical alert events with optional filters",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List History Alerts",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"hours": {
						Type:        "integer",
						Description: "Lookback hours",
					},
					"stime": {
						Type:        "integer",
						Description: "Start time Unix timestamp",
					},
					"etime": {
						Type:        "integer",
						Description: "End time Unix timestamp",
					},
					"severity": {
						Type:        "integer",
						Description: "Severity level (-1=all, 1=critical, 2=warning, 3=info)",
					},
					"is_recovered": {
						Type:        "integer",
						Description: "Recovery status (-1=all, 0=not recovered, 1=recovered)",
					},
					"query": {
						Type:        "string",
						Description: "Search keyword",
					},
					"cate": {
						Type:        "string",
						Description: "Alert category",
					},
					"rule_prods": {
						Type:        "string",
						Description: "Product types comma-separated",
					},
					"datasource_ids": {
						Type:        "string",
						Description: "Datasource IDs comma-separated",
					},
					"bgid": {
						Type:        "integer",
						Description: "Business group ID",
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
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListHistoryAlertsInput) (*mcp.CallToolResult, error) {
			if err := toolset.ValidateTimeRange(input.Hours, input.Stime, input.Etime); err != nil {
				return toolset.NewToolResultError(fmt.Sprintf("invalid input: %v", err)), nil
			}
			if err := toolset.ValidatePagination(input.Limit, input.Page); err != nil {
				return toolset.NewToolResultError(fmt.Sprintf("invalid input: %v", err)), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			params := url.Values{}
			if input.Hours > 0 {
				params.Set("hours", strconv.FormatInt(input.Hours, 10))
			}
			if input.Stime > 0 {
				params.Set("stime", strconv.FormatInt(input.Stime, 10))
			}
			if input.Etime > 0 {
				params.Set("etime", strconv.FormatInt(input.Etime, 10))
			}
			if input.Severity != 0 {
				params.Set("severity", strconv.Itoa(input.Severity))
			}
			if input.IsRecovered != 0 {
				params.Set("is_recovered", strconv.Itoa(input.IsRecovered))
			}
			if input.Query != "" {
				params.Set("query", input.Query)
			}
			if input.Cate != "" {
				params.Set("cate", input.Cate)
			}
			if input.RuleProds != "" {
				params.Set("rule_prods", input.RuleProds)
			}
			if input.DatasourceIds != "" {
				params.Set("datasource_ids", input.DatasourceIds)
			}
			if input.BusiGroupId > 0 {
				params.Set("bgid", strconv.FormatInt(input.BusiGroupId, 10))
			}
			if input.Limit > 0 {
				params.Set("limit", strconv.Itoa(input.Limit))
			}
			if input.Page > 0 {
				params.Set("p", strconv.Itoa(input.Page))
			}

			result, err := client.DoGet[types.PageResp[types.AlertHisEvent]](c, ctx, "/api/n9e/alert-his-events/list", params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getHistoryAlertTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_history_alert",
			Description: "Get details of a specific historical alert event by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get History Alert",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"eid"},
				Properties: map[string]*jsonschema.Schema{
					"eid": {
						Type:        "integer",
						Description: "Alert event ID",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetAlertInput) (*mcp.CallToolResult, error) {
			if input.EventId <= 0 {
				return toolset.NewToolResultError("eid is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/alert-his-event/%d", input.EventId)
			result, err := client.DoGet[types.AlertHisEvent](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func listAlertRulesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_alert_rules",
			Description: "List alert rules for a business group",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Alert Rules",
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
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListAlertRulesInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-rules", input.GroupId)
			result, err := client.DoGet[[]types.AlertRule](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getAlertRuleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_alert_rule",
			Description: "Get details of a specific alert rule by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Alert Rule",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"arid"},
				Properties: map[string]*jsonschema.Schema{
					"arid": {
						Type:        "integer",
						Description: "Alert rule ID",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetAlertRuleInput) (*mcp.CallToolResult, error) {
			if input.RuleId <= 0 {
				return toolset.NewToolResultError("arid is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/alert-rule/%d", input.RuleId)
			result, err := client.DoGet[types.AlertRule](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}
