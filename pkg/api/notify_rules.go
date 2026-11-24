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
			Description: "Find notification routing rules that connect alerts to event pipelines and delivery configurations. Use get_notify_rule for one rule, list_notify_channels for transport settings or list_notify_templates for message format. This read-only lookup returns {list, total} in name order with IDs, enabled flags and routing settings, paginated locally at 20 per page by default. Admins see all rules, and other callers see rules shared with at least one of their user groups.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Notification Rules",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"limit": {Type: "integer", Description: "Number of records per page. Defaults to 20."},
					"p":     {Type: "integer", Description: "1-based page number. Defaults to 1."},
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
			Description: "Inspect one rule's event pipeline and notification delivery settings, including channel and template IDs and delivery conditions. Use list_notify_rules to discover rules or get_notify_channel to inspect the transport. This read-only lookup returns one rule and requires the notification rules permission plus membership in a linked user group for non-admin callers.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Notification Rule",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer", Description: "Positive notification rule ID from list_notify_rules."},
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
			Description: "Find configured notification transports and their IDs, including request and parameter settings. Use get_notify_channel to inspect one known channel, list_notify_rules for routing or list_notify_templates for message source. This read-only lookup requires the notification channels permission and returns an unpaginated array ordered by ascending weight. Configurations can contain credentials.",
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
			Description: "Inspect the transport and parameter configuration for one notification channel, such as its endpoint or request settings. Use list_notify_channels to discover IDs, list_notify_rules to find where a channel is used, or list_notify_templates to inspect message formatting. This read-only lookup requires the notification channels permission and returns one configuration object, which can contain credentials, or a not-found error.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Notify Channel",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer", Description: "Positive notification channel ID from list_notify_channels."},
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
			Description: "Retrieve the message templates that define how Nightingale formats alert notifications. Use this to inspect template source or find a template ID for a notification rule. Use list_notify_channels for delivery configuration and list_notify_rules for routing. This read-only lookup returns an unpaginated array for signed-in callers with IDs, names, channel keys, template source and a built_in flag.",
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
			Description: "Save one or more new notification routing rules. Use update_notify_rule for an existing rule or test_notify_rule to exercise delivery with sample events. Non-admin callers must belong to at least one linked user group for each rule. Returns the saved rules and the submitted count.",
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
						Description: "Nonempty array of rule objects with name, enable, user_group_ids, and the relevant pipeline_configs and notify_configs. Get IDs from list_user_groups, list_event_pipelines, list_notify_channels and list_notify_templates.",
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
			Description: "Replace the saved routing configuration of one notification rule, including delivery conditions and destinations. Start with get_notify_rule and submit the edited rule. Nightingale checks access to the existing rule's user groups for non-admin callers and returns the updated ID on success.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Notify Rule",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "rule"},
				Properties: map[string]*jsonschema.Schema{
					"id":   {Type: "integer", Description: "Positive notification rule ID from list_notify_rules."},
					"rule": {Type: "object", Description: "Full edited rule from get_notify_rule. Include the name, user_group_ids, pipeline_configs and notify_configs to retain. Resolve any new channel or template IDs through list_notify_channels or list_notify_templates."},
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
			Description: "Send a real test notification through a configured channel using stored alert events or a synthetic event. Use get_notify_rule to inspect routing or update_notify_rule to persist an edit. The test can trigger downstream actions and returns the delivery response or an error.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Test Notify Rule",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body": {Type: "object", Description: "Test object with notify_config, including channel_id from list_notify_channels and the channel parameters. Supply event_ids from list_history_alerts or set use_mock_event to true. Optional with_recovery also sends a recovery notification."},
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
			Description: "Submit a new notification transport configuration for saving. Use update_notify_channel to replace a saved channel. This tool sends one object, which Nightingale versions that require a top-level array will reject.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Notify Channel",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"channel"},
				Properties: map[string]*jsonschema.Schema{
					"channel": {Type: "object", Description: "Channel configuration with name, ident, request_type, and the corresponding request_config and param_config. See get_notify_channel for the shape of an existing transport."},
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
			Description: "Replace the saved transport and parameter settings for one notification channel. Use get_notify_channel to read the current configuration before editing it, or update_notify_rule to change how the channel is used in routing. This write returns the channel ID and a confirmation.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Notify Channel",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "channel"},
				Properties: map[string]*jsonschema.Schema{
					"id":      {Type: "integer", Description: "Positive notification channel ID from list_notify_channels."},
					"channel": {Type: "object", Description: "Channel configuration with name, ident, request_type, and the corresponding request_config and param_config. See get_notify_channel for the shape of an existing transport. Include the existing id for the server's duplicate-name check."},
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
			Description: "Save a new message template for notification rendering. Use update_notify_template to rename an existing template or update_notify_template_content to edit its source. Nightingale validates the template syntax and rejects duplicate names or channel keys.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Notify Template",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"template"},
				Properties: map[string]*jsonschema.Schema{
					"template": {Type: "object", Description: "Template object with a unique name and channel key plus content as Go template source. Use list_notify_templates to inspect examples and existing keys."},
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
			Description: "Rename a saved notification template. Use update_notify_template_content to change the source used for rendering. Only the creator or an admin can perform this write, and Nightingale rejects a conflicting name.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Notify Template",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"template"},
				Properties: map[string]*jsonschema.Schema{
					"template": {Type: "object", Description: "Object with the template id from list_notify_templates and the new name. Keep its existing channel key when building the body."},
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
			Description: "Replace the Go template source used to render notifications. Use update_notify_template to rename the template. Nightingale validates the source and allows this write for the template creator or an admin.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Notify Template Content",
				ReadOnlyHint:    false,
				DestructiveHint: toolset.BoolPtr(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"body"},
				Properties: map[string]*jsonschema.Schema{
					"body": {Type: "object", Description: "Object with id from list_notify_templates and content as the full replacement Go template source."},
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
