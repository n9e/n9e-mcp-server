package api

import (
	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
)

// DefaultToolsetGroup creates the default toolset group
func DefaultToolsetGroup(getClient client.GetClientFunc, readOnly bool) *toolset.ToolsetGroup {
	group := toolset.NewToolsetGroup(readOnly)

	// Register all toolsets
	RegisterAlertsToolset(group, getClient)
	RegisterTargetsToolset(group, getClient)
	RegisterDatasourceToolset(group, getClient)
	RegisterMutesToolset(group, getClient)
	RegisterBusiGroupsToolset(group, getClient)
	RegisterNotifyRulesToolset(group, getClient)
	RegisterAlertSubscribesToolset(group, getClient)
	RegisterEventPipelinesToolset(group, getClient)
	RegisterUsersToolset(group, getClient)

	return group
}
