package toolset

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultToolsets is the default enabled toolsets
var DefaultToolsets = []string{"alerts", "targets", "datasource", "mutes", "busi_groups", "notify_rules", "alert_subscribes", "event_pipelines", "users"}

// ServerTool wraps MCP tool and its handler function
type ServerTool struct {
	Tool    mcp.Tool
	Handler mcp.ToolHandler
}

// NewServerTool creates a ServerTool
func NewServerTool(tool mcp.Tool, handler mcp.ToolHandler) ServerTool {
	return ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// Toolset represents a toolset
type Toolset struct {
	Name        string
	Description string
	ReadTools   []ServerTool
	WriteTools  []ServerTool
}

// NewToolset creates a toolset
func NewToolset(name, description string) *Toolset {
	return &Toolset{
		Name:        name,
		Description: description,
		ReadTools:   make([]ServerTool, 0),
		WriteTools:  make([]ServerTool, 0),
	}
}

// AddReadTools adds read-only tools
func (t *Toolset) AddReadTools(tools ...ServerTool) *Toolset {
	t.ReadTools = append(t.ReadTools, tools...)
	return t
}

// AddWriteTools adds write tools
func (t *Toolset) AddWriteTools(tools ...ServerTool) *Toolset {
	t.WriteTools = append(t.WriteTools, tools...)
	return t
}

// ToolsetGroup represents a toolset group
type ToolsetGroup struct {
	toolsets map[string]*Toolset
	enabled  map[string]bool
	readOnly bool
}

// NewToolsetGroup creates a toolset group
func NewToolsetGroup(readOnly bool) *ToolsetGroup {
	return &ToolsetGroup{
		toolsets: make(map[string]*Toolset),
		enabled:  make(map[string]bool),
		readOnly: readOnly,
	}
}

// AddToolset adds a toolset
func (g *ToolsetGroup) AddToolset(t *Toolset) {
	g.toolsets[t.Name] = t
}

// EnableToolsets enables the specified toolsets
func (g *ToolsetGroup) EnableToolsets(names []string) error {
	// Handle "all" special value
	for _, name := range names {
		if name == "all" {
			for toolsetName := range g.toolsets {
				g.enabled[toolsetName] = true
			}
			return nil
		}
	}

	// Handle specific toolset names
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := g.toolsets[name]; !ok {
			return fmt.Errorf("unknown toolset: %s", name)
		}
		g.enabled[name] = true
	}

	return nil
}

// RegisterAll registers all enabled tools to MCP Server
func (g *ToolsetGroup) RegisterAll(s *mcp.Server) {
	for name, toolset := range g.toolsets {
		if !g.enabled[name] {
			continue
		}

		// Register read-only tools
		for _, st := range toolset.ReadTools {
			tool := st.Tool
			s.AddTool(&tool, st.Handler)
		}

		// If not read-only mode, register write tools
		if !g.readOnly {
			for _, st := range toolset.WriteTools {
				tool := st.Tool
				s.AddTool(&tool, st.Handler)
			}
		}
	}
}

// GetAvailableToolsets gets all available toolset names
func (g *ToolsetGroup) GetAvailableToolsets() []string {
	names := make([]string, 0, len(g.toolsets))
	for name := range g.toolsets {
		names = append(names, name)
	}
	return names
}

// GetEnabledToolsets gets enabled toolset names
func (g *ToolsetGroup) GetEnabledToolsets() []string {
	names := make([]string, 0, len(g.enabled))
	for name := range g.enabled {
		names = append(names, name)
	}
	return names
}

// MakeToolHandler creates a tool handler helper function
func MakeToolHandler[T any](handler func(ctx context.Context, req *mcp.CallToolRequest, input T) (*mcp.CallToolResult, error)) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input T
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
				return NewToolResultError(fmt.Sprintf("failed to parse input: %v", err)), nil
			}
		}
		return handler(ctx, req, input)
	}
}
