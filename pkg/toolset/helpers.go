package toolset

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BoolPtr returns a pointer to bool
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to int
func IntPtr(i int) *int {
	return &i
}

// StringPtr returns a pointer to string
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// MarshalResult serializes result to JSON and returns MCP tool result
func MarshalResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return NewToolResultError("failed to marshal result: " + err.Error())
	}
	return NewToolResultText(string(data))
}

// NewToolResultText creates a text result
func NewToolResultText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// NewToolResultError creates an error result
func NewToolResultError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}
