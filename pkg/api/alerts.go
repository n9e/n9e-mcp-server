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
	Limit   int   `json:"limit,omitempty"`
	Page    int   `json:"p,omitempty"`
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

	// Write tools (create/update/import/clone/toggle)
	ts.AddWriteTools(
		createAlertRuleTool(getClient),
		updateAlertRuleTool(getClient),
		importAlertRulesTool(getClient),
		importPromRulesTool(getClient),
		cloneAlertRulesToBgsTool(getClient),
		toggleAlertRulesTool(getClient),
	)

	group.AddToolset(ts)
}

func listActiveAlertsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_active_alerts",
			Description: "Find currently firing alert events, optionally filtering by rule, severity or trigger time. Use list_history_alerts to search stored events including recoveries, or get_active_alert for one active event. This read-only lookup returns {list, total} with event IDs, rule names, severity and tags, newest trigger first, with 20 events per page by default. Nightingale can restrict results to the caller's business groups.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Active Alerts",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"hours": {
						Type:        "integer",
						Description: "Positive lookback in hours. Nightingale sets the start to now minus this value and the end to now plus 24 hours. Provide either hours or absolute times.",
					},
					"stime": {
						Type:        "integer",
						Description: "Inclusive start of the trigger-time window in Unix seconds. Omit hours, stime and etime to include all trigger times.",
					},
					"etime": {
						Type:        "integer",
						Description: "End of the window in Unix seconds, inclusive. Defaults to now plus 24 hours when stime is supplied. If set with stime, it must be later. The active alert list needs a start time to apply this boundary.",
					},
					"severity": {
						Type:        "string",
						Description: "Comma-separated severity levels. Use 1 for critical, 2 for warning and 3 for info. Omit for every severity.",
					},
					"query": {
						Type:        "string",
						Description: "Space-separated terms. Each term must match part of the rule name or tags. Omit for no text filter.",
					},
					"cate": {
						Type:        "string",
						Description: "Comma-separated event categories, such as prometheus, host, elasticsearch or loki. Omit or use $all for every category.",
					},
					"rule_prods": {
						Type:        "string",
						Description: "Comma-separated rule products, such as metric, host, loki or anomaly. Omit for every product.",
					},
					"datasource_ids": {
						Type:        "string",
						Description: "Comma-separated datasource IDs from list_datasources. Omit for every datasource.",
					},
					"rid": {
						Type:        "integer",
						Description: "Positive alert rule ID from list_alert_rules. Omit for events from any rule.",
					},
					"bgid": {
						Type:        "integer",
						Description: "Positive business group ID from list_busi_groups. Omit to use the server's default business group scope.",
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
			Description: "Inspect one currently firing event for triage, including its rule, severity, tags, trigger value and claimant when present. Use list_active_alerts to find its ID or get_history_alert if the event has recovered. This read-only lookup returns one current event and checks access to its business group unless Nightingale allows anonymous or shared detail access.",
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
						Description: "Positive current event ID from list_active_alerts. Use the event's id field.",
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
			Description: "Search stored alert events, including recovered events, by last evaluation time. Provide hours or an absolute time window, or use list_active_alerts to investigate currently firing alerts. This read-only lookup returns {list, total} with event IDs, severity and recovery fields, ordered by last evaluation time then ID descending. Pages default to 20, and Nightingale can restrict the list to the caller's business groups.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List History Alerts",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"hours": {
						Type:        "integer",
						Description: "Positive lookback in hours. Nightingale sets the start to now minus this value and the end to now plus 24 hours. Provide either hours or absolute times.",
					},
					"stime": {
						Type:        "integer",
						Description: "Start of the last-evaluation-time window in Unix seconds, inclusive. Defaults to 0 when no lookback is given. Supply stime or hours to search a recent window.",
					},
					"etime": {
						Type:        "integer",
						Description: "End of the window in Unix seconds, inclusive. Defaults to now plus 24 hours when stime is supplied. If set with stime, it must be later.",
					},
					"severity": {
						Type:        "integer",
						Description: "One severity level. Use 1 for critical, 2 for warning, 3 for info or -1 for all. Omission and 0 also select all.",
					},
					"is_recovered": {
						Type:        "integer",
						Description: "Use 1 for recovered events or -1 for all events. The tool omits 0 when calling Nightingale, so 0 also selects all. For unrecovered records, filter the returned is_recovered field for 0.",
					},
					"query": {
						Type:        "string",
						Description: "Space-separated terms. Each term must match part of the rule name or tags. Omit for no text filter.",
					},
					"cate": {
						Type:        "string",
						Description: "Comma-separated event categories, such as prometheus, host, elasticsearch or loki. Omit or use $all for every category.",
					},
					"rule_prods": {
						Type:        "string",
						Description: "Comma-separated rule products, such as metric, host, loki or anomaly. Omit for every product.",
					},
					"datasource_ids": {
						Type:        "string",
						Description: "Comma-separated datasource IDs from list_datasources. Omit for every datasource.",
					},
					"bgid": {
						Type:        "integer",
						Description: "Positive business group ID from list_busi_groups. Omit to use the server's default business group scope.",
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
			Description: "Inspect one stored alert event by history ID, including recorded trigger values, tags and recovery status or timestamp. Use list_history_alerts to find the ID or get_active_alert to check a currently firing event. This read-only lookup returns one event or a not-found error and checks access to the owning business group unless Nightingale allows anonymous or shared detail access.",
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
						Description: "Positive history event ID from list_history_alerts. Use the event's id field.",
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
			Description: "Find the rules that evaluate conditions and generate alerts in one business group. Use get_alert_rule to inspect one known rule or list_active_alerts to see events that are firing. This read-only lookup requires the alert rules permission and returns {list, total} with rule IDs, names and configuration, sorted by name and paginated locally at 20 per page by default.",
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

			items, total := toolset.SlicePage(result, input.Page, input.Limit)
			return toolset.MarshalResult(types.PageResp[types.AlertRule]{List: items, Total: total}), nil
		}),
	)
}

func getAlertRuleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_alert_rule",
			Description: "Inspect the configured query, evaluation settings and notification routing for a single alert rule. Use list_alert_rules to discover rules in a group, or get_active_alert to inspect a firing event produced by a rule. This read-only lookup requires the alert rules permission and returns one rule object, or an error if the ID is unknown.",
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
						Description: "Positive alert rule ID from list_alert_rules.",
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

// --- Write tools ---

// CreateAlertRuleInput accepts a full rule object (matches the shape returned by get_alert_rule).
// Using map[string]any keeps the surface flexible across n9e versions and rule categories
// (metric/log/host/anomaly each carry different `rule_config`/`severities` shapes).
type CreateAlertRuleInput struct {
	GroupId int64          `json:"group_id"`
	Rule    map[string]any `json:"rule"`
}

func createAlertRuleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_alert_rule",
			Description: "Save a new alert evaluation rule in one business group. Use update_alert_rule to change an existing rule or import_prom_rules for a Prometheus rule file. Requires write access to the group. Inspect result for rule-level validation or save errors.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Alert Rule",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "rule"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Positive business group ID from list_busi_groups."},
					"rule": {
						Type:        "object",
						Description: "Nightingale rule configuration. Include name and rule_config with a query and evaluation settings for the rule kind. Nightingale defaults prod to metric, cate to prometheus and prom_eval_interval to 15 seconds. Use get_alert_rule for a working example and omit its ID and audit fields.",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input CreateAlertRuleInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}
			if len(input.Rule) == 0 {
				return toolset.NewToolResultError("rule body is required"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-rules", input.GroupId)
			result, err := client.DoPost[any](c, ctx, path, []map[string]any{input.Rule})
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{
				"result":  result,
				"message": "Alert rule created",
			}), nil
		}),
	)
}

// UpdateAlertRuleInput updates a single rule by id within its business group.
type UpdateAlertRuleInput struct {
	GroupId int64          `json:"group_id"`
	RuleId  int64          `json:"arid"`
	Rule    map[string]any `json:"rule"`
}

func updateAlertRuleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_alert_rule",
			Description: "Replace the saved configuration of one alert rule. Start from get_alert_rule and submit the edited configuration. Use toggle_alert_rules for an enable or disable change across rules. Nightingale checks write access to the rule's owning group.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Alert Rule",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "arid", "rule"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Positive business group ID from list_busi_groups. Use the group that owns the rule."},
					"arid":     {Type: "integer", Description: "Positive alert rule ID from list_alert_rules."},
					"rule":     {Type: "object", Description: "Full edited rule configuration based on get_alert_rule. Include the query, evaluation settings and notification configuration you want to keep."},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input UpdateAlertRuleInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 || input.RuleId <= 0 {
				return toolset.NewToolResultError("group_id and arid are required and must be positive"), nil
			}
			if len(input.Rule) == 0 {
				return toolset.NewToolResultError("rule body is required"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-rule/%d", input.GroupId, input.RuleId)
			_, err := client.DoPut[any](c, ctx, path, input.Rule)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{
				"id":      input.RuleId,
				"message": "Alert rule updated",
			}), nil
		}),
	)
}

// ImportAlertRulesInput bulk-imports an array of rule bodies.
type ImportAlertRulesInput struct {
	GroupId int64            `json:"group_id"`
	Rules   []map[string]any `json:"rules"`
}

func importAlertRulesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "import_alert_rules",
			Description: "Bulk-create rules from Nightingale alert-rule objects in one business group. Use import_prom_rules for a Prometheus rule file or create_alert_rule for a single newly authored rule. Nightingale converts imported rules to the new notification configuration and returns rule-level errors in result.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Import Alert Rules",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "rules"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Positive business group ID from list_busi_groups."},
					"rules": {
						Type:        "array",
						Description: "Nonempty array of Nightingale rule objects. Use list_alert_rules or get_alert_rule for the field layout. Rules without datasource_queries default to all datasources of the relevant category.",
						Items:       &jsonschema.Schema{Type: "object"},
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ImportAlertRulesInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}
			if len(input.Rules) == 0 {
				return toolset.NewToolResultError("rules must be a non-empty array"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-rules/import", input.GroupId)
			result, err := client.DoPost[any](c, ctx, path, input.Rules)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{
				"result":   result,
				"imported": len(input.Rules),
			}), nil
		}),
	)
}

// ImportPromRulesInput imports Prometheus YAML/JSON rule definitions.
type ImportPromRulesInput struct {
	GroupId           int64  `json:"group_id"`
	Payload           string `json:"payload"`
	DatasourceQueries any    `json:"datasource_queries,omitempty"`
	Disabled          int    `json:"disabled,omitempty"`
}

func importPromRulesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "import_prom_rules",
			Description: "Convert Prometheus alerting rules and save them as Nightingale rules in one business group. Use import_alert_rules for Nightingale-format rule objects. Check result for rule-level errors after the import.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Import Prometheus Rules",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "payload"},
				Properties: map[string]*jsonschema.Schema{
					"group_id":           {Type: "integer", Description: "Positive business group ID from list_busi_groups."},
					"payload":            {Type: "string", Description: "Raw Prometheus rule file as a YAML or JSON string with a top-level groups array."},
					"datasource_queries": {Description: "Datasource selectors required by Nightingale for the imported rules. For exact IDs from list_datasources, use an array of objects with match_type 0, op in and numeric IDs in values."},
					"disabled":           {Type: "integer", Description: "Initial rule state. Use 0 for enabled (default) or 1 for disabled."},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ImportPromRulesInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}
			if input.Payload == "" {
				return toolset.NewToolResultError("payload is required"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			body := map[string]any{
				"Payload":           input.Payload,
				"DatasourceQueries": input.DatasourceQueries,
				"Disabled":          input.Disabled,
			}
			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-rules/import-prom-rule", input.GroupId)
			result, err := client.DoPost[any](c, ctx, path, body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result}), nil
		}),
	)
}

// CloneAlertRulesToBgsInput clones rules to one or more target business groups.
type CloneAlertRulesToBgsInput struct {
	RuleIds []int64 `json:"rule_ids"`
	Bgids   []int64 `json:"bgids"`
}

func cloneAlertRulesToBgsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "clone_alert_rules_to_bgs",
			Description: "Create copies of existing alert rules in one or more destination business groups. Use import_alert_rules when the source is a set of Nightingale rule objects. Nightingale checks destination write access and returns per-copy errors in result, keyed by source rule and destination group IDs.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Clone Alert Rules to Business Groups",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"rule_ids", "bgids"},
				Properties: map[string]*jsonschema.Schema{
					"rule_ids": {Type: "array", Description: "Nonempty array of source alert rule IDs from list_alert_rules.", Items: &jsonschema.Schema{Type: "integer"}},
					"bgids":    {Type: "array", Description: "Nonempty array of destination business group IDs from list_busi_groups.", Items: &jsonschema.Schema{Type: "integer"}},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input CloneAlertRulesToBgsInput) (*mcp.CallToolResult, error) {
			if len(input.RuleIds) == 0 || len(input.Bgids) == 0 {
				return toolset.NewToolResultError("rule_ids and bgids are required and must be non-empty"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			body := map[string]any{
				"RuleIds": input.RuleIds,
				"Bgids":   input.Bgids,
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/busi-groups/alert-rules/clones", body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result}), nil
		}),
	)
}

// ToggleAlertRulesInput enables or disables a batch of rules within a business group.
type ToggleAlertRulesInput struct {
	GroupId  int64   `json:"group_id"`
	Ids      []int64 `json:"ids"`
	Disabled int     `json:"disabled"`
}

func toggleAlertRulesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "toggle_alert_rules",
			Description: "Enable or disable evaluation for selected alert rules in one business group. Use update_alert_rule to change the query or notification settings. This writes the disabled flag and returns the requested state and number of submitted IDs.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Toggle Alert Rules",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "ids", "disabled"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Positive business group ID from list_busi_groups. Use the group that owns the rules."},
					"ids":      {Type: "array", Description: "Nonempty array of IDs from list_alert_rules for that group.", Items: &jsonschema.Schema{Type: "integer"}},
					"disabled": {Type: "integer", Description: "Use 0 to enable rule evaluation or 1 to disable it."},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ToggleAlertRulesInput) (*mcp.CallToolResult, error) {
			if input.GroupId <= 0 {
				return toolset.NewToolResultError("group_id is required and must be positive"), nil
			}
			if len(input.Ids) == 0 {
				return toolset.NewToolResultError("ids must be non-empty"), nil
			}
			if input.Disabled != 0 && input.Disabled != 1 {
				return toolset.NewToolResultError("disabled must be 0 or 1"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			body := map[string]any{
				"ids":    input.Ids,
				"fields": map[string]any{"disabled": input.Disabled},
				"action": "update_fields",
			}
			path := fmt.Sprintf("/api/n9e/busi-group/%d/alert-rules/fields", input.GroupId)
			_, err := client.DoPut[any](c, ctx, path, body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{
				"updated":  len(input.Ids),
				"disabled": input.Disabled,
			}), nil
		}),
	)
}
