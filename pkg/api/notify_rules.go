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

// ListNotifyRulesInput represents notification rules list query parameters
type ListNotifyRulesInput struct {
	Limit int `json:"limit,omitempty"`
	Page  int `json:"p,omitempty"`
}

// GetNotifyRuleInput represents single notification rule query parameters
type GetNotifyRuleInput struct {
	RuleId int64 `json:"id"`
}

// RegisterNotifyRulesToolset registers notification rules + channels + templates toolset
func RegisterNotifyRulesToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("notify_rules", "Notification rule, channel and template management")

	ts.AddReadTools(
		listNotifyRulesTool(getClient),
		getNotifyRuleTool(getClient),
		listNotifyChannelsTool(getClient),
		getNotifyChannelTool(getClient),
		listNotifyTemplatesTool(getClient),
	)

	ts.AddWriteTools(
		createNotifyRulesTool(getClient),
		updateNotifyRuleTool(getClient),
		testNotifyRuleTool(getClient),
		createNotifyChannelTool(getClient),
		updateNotifyChannelTool(getClient),
		createNotifyTemplateTool(getClient),
		updateNotifyTemplateTool(getClient),
		updateNotifyTemplateContentTool(getClient),
	)

	group.AddToolset(ts)
}

// --- Read tools ---

func listNotifyRulesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_notify_rules",
			Description: "List all notification rules that the current user has access to",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Notification Rules",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"limit": {Type: "integer", Description: "Page size (default 20)"},
					"p":     {Type: "integer", Description: "Page number (starts from 1)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListNotifyRulesInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[[]types.NotifyRule](c, ctx, "/api/n9e/notify-rules", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			items, total := toolset.SlicePage(result, input.Page, input.Limit)
			return toolset.MarshalResult(types.PageResp[types.NotifyRule]{List: items, Total: total}), nil
		}),
	)
}

func getNotifyRuleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_notify_rule",
			Description: "Get details of a specific notification rule by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Notification Rule",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer", Description: "Notification rule ID"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetNotifyRuleInput) (*mcp.CallToolResult, error) {
			if input.RuleId <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[types.NotifyRule](c, ctx, fmt.Sprintf("/api/n9e/notify-rule/%d", input.RuleId), nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

func listNotifyChannelsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_notify_channels",
			Description: "List all notification channel configurations (full payload — admin perspective)",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Notify Channels",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{Type: "object"},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[[]map[string]any](c, ctx, "/api/n9e/notify-channel-configs", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

type getByIDInput struct {
	Id int64 `json:"id"`
}

func getNotifyChannelTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_notify_channel",
			Description: "Get a single notification channel configuration by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Notify Channel",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer", Description: "Notification channel ID"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input getByIDInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[map[string]any](c, ctx, fmt.Sprintf("/api/n9e/notify-channel-config/%d", input.Id), nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

func listNotifyTemplatesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_notify_templates",
			Description: "List all notification templates",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Notify Templates",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{Type: "object"},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoGet[[]map[string]any](c, ctx, "/api/n9e/notify-tpls", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(result), nil
		}),
	)
}

// --- Write tools ---

type createNotifyRulesInput struct {
	Rules []map[string]any `json:"rules"`
}

func createNotifyRulesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_notify_rules",
			Description: "Create one or more notification rules (n9e endpoint accepts an array)",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Notify Rules",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"rules"},
				Properties: map[string]*jsonschema.Schema{
					"rules": {
						Type:        "array",
						Description: "Notification rule bodies",
						Items:       &jsonschema.Schema{Type: "object"},
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input createNotifyRulesInput) (*mcp.CallToolResult, error) {
			if len(input.Rules) == 0 {
				return toolset.NewToolResultError("rules must be non-empty"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/notify-rules", input.Rules)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result, "created": len(input.Rules)}), nil
		}),
	)
}

type updateNotifyRuleInput struct {
	Id   int64          `json:"id"`
	Rule map[string]any `json:"rule"`
}

func updateNotifyRuleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_notify_rule",
			Description: "Update an existing notification rule by ID",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Notify Rule",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "rule"},
				Properties: map[string]*jsonschema.Schema{
					"id":   {Type: "integer", Description: "Notification rule ID"},
					"rule": {Type: "object", Description: "Full rule body"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateNotifyRuleInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/notify-rule/%d", input.Id), input.Rule)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Id, "message": "Notification rule updated"}), nil
		}),
	)
}

type testNotifyRuleInput struct {
	Body map[string]any `json:"body"`
}

func testNotifyRuleTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "test_notify_rule",
			Description: "Send a test notification using the supplied notify-rule body without persisting it.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Test Notify Rule",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body": {Type: "object", Description: "Notification rule body to test (matches n9e's /notify-rule/test payload)"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input testNotifyRuleInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/notify-rule/test", input.Body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result}), nil
		}),
	)
}

type createNotifyChannelInput struct {
	Channel map[string]any `json:"channel"`
}

func createNotifyChannelTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_notify_channel",
			Description: "Create a notification channel configuration (e.g. webhook, dingtalk, feishu).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Notify Channel",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"channel"},
				Properties: map[string]*jsonschema.Schema{
					"channel": {Type: "object", Description: "Channel configuration body"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input createNotifyChannelInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/notify-channel-configs", input.Channel)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result, "message": "Channel created"}), nil
		}),
	)
}

type updateNotifyChannelInput struct {
	Id      int64          `json:"id"`
	Channel map[string]any `json:"channel"`
}

func updateNotifyChannelTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_notify_channel",
			Description: "Update an existing notification channel configuration",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Notify Channel",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "channel"},
				Properties: map[string]*jsonschema.Schema{
					"id":      {Type: "integer"},
					"channel": {Type: "object"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateNotifyChannelInput) (*mcp.CallToolResult, error) {
			if input.Id <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, fmt.Sprintf("/api/n9e/notify-channel-config/%d", input.Id), input.Channel)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"id": input.Id, "message": "Channel updated"}), nil
		}),
	)
}

type createNotifyTemplateInput struct {
	Template map[string]any `json:"template"`
}

func createNotifyTemplateTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "create_notify_template",
			Description: "Create a notification template",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Notify Template",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"template"},
				Properties: map[string]*jsonschema.Schema{
					"template": {Type: "object"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input createNotifyTemplateInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			result, err := client.DoPost[any](c, ctx, "/api/n9e/notify-tpl", input.Template)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"result": result}), nil
		}),
	)
}

type updateNotifyTemplateInput struct {
	Template map[string]any `json:"template"`
}

func updateNotifyTemplateTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_notify_template",
			Description: "Update a notification template (full body)",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Notify Template",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"template"},
				Properties: map[string]*jsonschema.Schema{
					"template": {Type: "object", Description: "Template body including id"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateNotifyTemplateInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, "/api/n9e/notify-tpl", input.Template)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"message": "Template updated"}), nil
		}),
	)
}

type updateNotifyTemplateContentInput struct {
	Body map[string]any `json:"body"`
}

func updateNotifyTemplateContentTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "update_notify_template_content",
			Description: "Update only the rendered content of a notification template (lighter than update_notify_template).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Notify Template Content",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body": {Type: "object", Description: "Body matching n9e's /notify-tpl/content payload"},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input updateNotifyTemplateContentInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}
			_, err := client.DoPut[any](c, ctx, "/api/n9e/notify-tpl/content", input.Body)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}
			return toolset.MarshalResult(map[string]any{"message": "Template content updated"}), nil
		}),
	)
}
