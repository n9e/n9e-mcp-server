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

// ListEventPipelinesInput represents event pipelines list query parameters
type ListEventPipelinesInput struct {
	Limit int `json:"limit,omitempty"`
	Page  int `json:"p,omitempty"`
}

// GetEventPipelineInput represents single event pipeline query parameters
type GetEventPipelineInput struct {
	PipelineId int64 `json:"id"`
}

// ListEventPipelineExecutionsInput represents event pipeline executions list query parameters
type ListEventPipelineExecutionsInput struct {
	PipelineId int64  `json:"pipeline_id"`
	Mode       string `json:"mode,omitempty"`
	Status     string `json:"status,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Page       int    `json:"p,omitempty"`
}

// ListAllEventPipelineExecutionsInput represents all event pipelines executions list query parameters
type ListAllEventPipelineExecutionsInput struct {
	PipelineId   int64  `json:"pipeline_id,omitempty"`
	PipelineName string `json:"pipeline_name,omitempty"`
	Mode         string `json:"mode,omitempty"`
	Status       string `json:"status,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Page         int    `json:"p,omitempty"`
}

// GetEventPipelineExecutionInput represents single execution record query parameters
type GetEventPipelineExecutionInput struct {
	ExecId string `json:"exec_id"`
}

// RegisterEventPipelinesToolset registers event pipelines toolset
func RegisterEventPipelinesToolset(group *toolset.ToolsetGroup, getClient client.GetClientFunc) {
	ts := toolset.NewToolset("event_pipelines", "Event pipeline/workflow management tools for event processing")

	ts.AddReadTools(
		listEventPipelinesTool(getClient),
		getEventPipelineTool(getClient),
		listEventPipelineExecutionsTool(getClient),
		listAllEventPipelineExecutionsTool(getClient),
		getEventPipelineExecutionTool(getClient),
	)

	group.AddToolset(ts)
}

func listEventPipelinesTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_event_pipelines",
			Description: "Discover team-scoped workflow definitions and obtain their IDs, trigger modes and enabled states. Use get_event_pipeline for a known definition or list_all_event_pipeline_executions for run history. This read-only lookup covers pipelines outside business groups and limits non-admin results to the caller's teams. It returns {list, total} in name order, paginated locally at 20 per page by default.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Event Pipelines",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
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
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListEventPipelinesInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			result, err := client.DoGet[[]types.EventPipeline](c, ctx, "/api/n9e/event-pipelines", nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			items, total := toolset.SlicePage(result, input.Page, input.Limit)
			return toolset.MarshalResult(types.PageResp[types.EventPipeline]{List: items, Total: total}), nil
		}),
	)
}

func getEventPipelineTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_event_pipeline",
			Description: "Inspect one event pipeline's workflow definition, including its trigger mode, filters, processors and configured nodes. Use list_event_pipelines to discover pipeline IDs or list_event_pipeline_executions to investigate actual runs. This read-only lookup returns one definition and enforces business group access for group-owned pipelines or team access for other pipelines.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Event Pipeline",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "Positive pipeline ID from list_event_pipelines.",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetEventPipelineInput) (*mcp.CallToolResult, error) {
			if input.PipelineId <= 0 {
				return toolset.NewToolResultError("id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/event-pipeline/%d", input.PipelineId)
			result, err := client.DoGet[types.EventPipeline](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func listEventPipelineExecutionsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_event_pipeline_executions",
			Description: "Review execution history for one known event pipeline, optionally filtered by trigger mode and run status. Use list_all_event_pipeline_executions to search across pipelines or by pipeline name, and get_event_pipeline_execution to inspect one run. This read-only lookup requires the event pipelines permission and returns {list, total} with run IDs, statuses, durations and errors when recorded, newest first. Pages default to 20 and accept up to 1000 records.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Pipeline Executions",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"pipeline_id"},
				Properties: map[string]*jsonschema.Schema{
					"pipeline_id": {
						Type:        "integer",
						Description: "Positive pipeline ID from list_event_pipelines.",
					},
					"mode": {
						Type:        "string",
						Description: "Exact trigger mode. Use event for alert-triggered runs, api for API-triggered runs or cron for scheduled runs. Omit for all modes.",
					},
					"status": {
						Type:        "string",
						Description: "Exact execution status. Use running, success or failed. Omit for all statuses.",
					},
					"limit": {
						Type:        "integer",
						Description: "Records per page, from 1 to 1000. Omitted or out-of-range values fall back to 20.",
					},
					"p": {
						Type:        "integer",
						Description: "1-based page number. Defaults to 1.",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListEventPipelineExecutionsInput) (*mcp.CallToolResult, error) {
			if input.PipelineId <= 0 {
				return toolset.NewToolResultError("pipeline_id is required and must be positive"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			params := url.Values{}
			if input.Mode != "" {
				params.Set("mode", input.Mode)
			}
			if input.Status != "" {
				params.Set("status", input.Status)
			}
			if input.Limit > 0 {
				params.Set("limit", strconv.Itoa(input.Limit))
			}
			if input.Page > 0 {
				params.Set("p", strconv.Itoa(input.Page))
			}

			path := fmt.Sprintf("/api/n9e/event-pipeline/%d/executions", input.PipelineId)
			result, err := client.DoGet[types.PageResp[types.EventPipelineExecution]](c, ctx, path, params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func listAllEventPipelineExecutionsTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "list_all_event_pipeline_executions",
			Description: "Search execution records across event pipelines, including by pipeline name when the ID is unknown. Use list_event_pipeline_executions for the run history of one known pipeline or get_event_pipeline_execution to inspect a single run. This read-only lookup requires the event pipelines permission and returns {list, total} with execution IDs, pipeline names, statuses and durations, newest first. Pages default to 20 and accept up to 1000 records.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List All Pipeline Executions",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"pipeline_id": {
						Type:        "integer",
						Description: "Positive pipeline ID from list_event_pipelines. Omit to search across pipeline IDs.",
					},
					"pipeline_name": {
						Type:        "string",
						Description: "Substring of the pipeline name recorded at execution time. Omit for every name.",
					},
					"mode": {
						Type:        "string",
						Description: "Exact trigger mode. Use event for alert-triggered runs, api for API-triggered runs or cron for scheduled runs. Omit for all modes.",
					},
					"status": {
						Type:        "string",
						Description: "Exact execution status. Use running, success or failed. Omit for all statuses.",
					},
					"limit": {
						Type:        "integer",
						Description: "Records per page, from 1 to 1000. Omitted or out-of-range values fall back to 20.",
					},
					"p": {
						Type:        "integer",
						Description: "1-based page number. Defaults to 1.",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input ListAllEventPipelineExecutionsInput) (*mcp.CallToolResult, error) {
			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			params := url.Values{}
			if input.PipelineId > 0 {
				params.Set("pipeline_id", strconv.FormatInt(input.PipelineId, 10))
			}
			if input.PipelineName != "" {
				params.Set("pipeline_name", input.PipelineName)
			}
			if input.Mode != "" {
				params.Set("mode", input.Mode)
			}
			if input.Status != "" {
				params.Set("status", input.Status)
			}
			if input.Limit > 0 {
				params.Set("limit", strconv.Itoa(input.Limit))
			}
			if input.Page > 0 {
				params.Set("p", strconv.Itoa(input.Page))
			}

			result, err := client.DoGet[types.PageResp[types.EventPipelineExecution]](c, ctx, "/api/n9e/event-pipeline-executions", params)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getEventPipelineExecutionTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_event_pipeline_execution",
			Description: "Inspect one recorded pipeline run for troubleshooting, including its status, node results, error details and duration in milliseconds. Use list_event_pipeline_executions or list_all_event_pipeline_executions to find the run ID, or get_event_pipeline for its workflow definition. This read-only lookup requires the event pipelines permission and returns one execution object, with node results as a JSON string when recorded.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Pipeline Execution",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"exec_id"},
				Properties: map[string]*jsonschema.Schema{
					"exec_id": {
						Type:        "string",
						Description: "Execution UUID from the id field returned by list_event_pipeline_executions or list_all_event_pipeline_executions.",
					},
				},
			},
		},
		toolset.MakeToolHandler(func(ctx context.Context, req *mcp.CallToolRequest, input GetEventPipelineExecutionInput) (*mcp.CallToolResult, error) {
			if input.ExecId == "" {
				return toolset.NewToolResultError("exec_id is required"), nil
			}

			c := getClient(ctx)
			if c == nil {
				return toolset.NewToolResultError("failed to get n9e client from context"), nil
			}

			path := fmt.Sprintf("/api/n9e/event-pipeline-execution/%s", input.ExecId)
			result, err := client.DoGet[types.EventPipelineExecution](c, ctx, path, nil)
			if err != nil {
				return toolset.NewToolResultError(err.Error()), nil
			}

			return toolset.MarshalResult(result), nil
		}),
	)
}
