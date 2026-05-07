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
			Description: "Create a new alert rule in a business group. Pass the full rule body in 'rule' (mirror the structure returned by get_alert_rule, omit id/create_at/update_at).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Alert Rule",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "rule"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Business group ID"},
					"rule": {
						Type:        "object",
						Description: "Full alert rule body. Must include name, prod, cate, severity, and rule_config / prom_ql appropriate for the rule kind.",
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
			Description: "Update an existing alert rule. Pass the full rule body (typically: get_alert_rule, modify fields, then submit).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Alert Rule",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "arid", "rule"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Business group ID owning the rule"},
					"arid":     {Type: "integer", Description: "Alert rule ID"},
					"rule":     {Type: "object", Description: "Full rule body"},
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
			Description: "Bulk import alert rules into a business group from a JSON array (matches the n9e import format).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Import Alert Rules",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "rules"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Business group ID"},
					"rules": {
						Type:        "array",
						Description: "Array of rule bodies",
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
			Description: "Import Prometheus alerting rules (YAML or JSON) into a business group. Wraps n9e's /alert-rules/import-prom-rule endpoint.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Import Prometheus Rules",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "payload"},
				Properties: map[string]*jsonschema.Schema{
					"group_id":           {Type: "integer", Description: "Business group ID"},
					"payload":            {Type: "string", Description: "Raw Prometheus rules YAML/JSON"},
					"datasource_queries": {Description: "Datasource selector (matches the FE wizard payload)"},
					"disabled":           {Type: "integer", Description: "Create rules disabled (0 = enabled, 1 = disabled)"},
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
			Description: "Clone the given alert rules into one or more target business groups.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Clone Alert Rules to Business Groups",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"rule_ids", "bgids"},
				Properties: map[string]*jsonschema.Schema{
					"rule_ids": {Type: "array", Description: "Source rule IDs", Items: &jsonschema.Schema{Type: "integer"}},
					"bgids":    {Type: "array", Description: "Target business group IDs", Items: &jsonschema.Schema{Type: "integer"}},
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
			Description: "Enable or disable a batch of alert rules. Uses n9e's PUT .../alert-rules/fields with {disabled:0|1}.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Toggle Alert Rules",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"group_id", "ids", "disabled"},
				Properties: map[string]*jsonschema.Schema{
					"group_id": {Type: "integer", Description: "Business group ID"},
					"ids":      {Type: "array", Description: "Alert rule IDs", Items: &jsonschema.Schema{Type: "integer"}},
					"disabled": {Type: "integer", Description: "0 = enable, 1 = disable"},
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
