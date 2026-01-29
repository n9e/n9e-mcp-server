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
type ListEventPipelinesInput struct{}

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
			Description: "List all event pipelines/workflows that the current user has access to",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Event Pipelines",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
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

			return toolset.MarshalResult(result), nil
		}),
	)
}

func getEventPipelineTool(getClient client.GetClientFunc) toolset.ServerTool {
	return toolset.NewServerTool(
		mcp.Tool{
			Name:        "get_event_pipeline",
			Description: "Get details of a specific event pipeline/workflow by ID",
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
						Description: "Event pipeline ID",
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
			Description: "List execution records for a specific event pipeline",
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
						Description: "Event pipeline ID",
					},
					"mode": {
						Type:        "string",
						Description: "Trigger mode filter (event/api/cron)",
					},
					"status": {
						Type:        "string",
						Description: "Status filter (running/success/failed)",
					},
					"limit": {
						Type:        "integer",
						Description: "Page size (default 20, max 1000)",
					},
					"p": {
						Type:        "integer",
						Description: "Page number (starts from 1)",
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
			Description: "List all event pipeline execution records across all pipelines",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List All Pipeline Executions",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"pipeline_id": {
						Type:        "integer",
						Description: "Filter by pipeline ID",
					},
					"pipeline_name": {
						Type:        "string",
						Description: "Filter by pipeline name",
					},
					"mode": {
						Type:        "string",
						Description: "Trigger mode filter (event/api/cron)",
					},
					"status": {
						Type:        "string",
						Description: "Status filter (running/success/failed)",
					},
					"limit": {
						Type:        "integer",
						Description: "Page size (default 20, max 1000)",
					},
					"p": {
						Type:        "integer",
						Description: "Page number (starts from 1)",
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
			Description: "Get details of a specific pipeline execution by execution ID",
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
						Description: "Execution ID (UUID)",
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
