package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"
)

// TestMuteToolsKeepMonthlyWindows pins that create_mute and update_mute pass
// monthly windows through to n9e. update_mute replaces the whole mute, so a
// dropped field would silently turn a monthly window into an empty weekly one.
func TestMuteToolsKeepMonthlyWindows(t *testing.T) {
	const periodic = `[{"cycle":"month","enable_days_of_month":[1,-1],"enable_stime":"00:00","enable_etime":"06:00"},{"enable_days_of_week":"0 6","enable_stime":"00:00","enable_etime":"00:00"}]`
	var want []any
	if err := json.Unmarshal([]byte(`[{"cycle":"month","enable_days_of_month":[1,-1],"enable_stime":"00:00","enable_etime":"06:00","enable_days_of_week":""},{"enable_days_of_week":"0 6","enable_stime":"00:00","enable_etime":"00:00"}]`), &want); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		tool   func(client.GetClientFunc) toolset.ServerTool
		method string
		path   string
		args   string
	}{
		{"create", createMuteTool, http.MethodPost, "/api/n9e/busi-group/1/alert-mutes", `{"group_id":1,"cause":"billing","mute_time_type":1,"periodic_mutes":` + periodic + `}`},
		{"update", updateMuteTool, http.MethodPut, "/api/n9e/busi-group/1/alert-mute/7", `{"group_id":1,"mute_id":7,"cause":"billing","mute_time_type":1,"periodic_mutes":` + periodic + `}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			c := newBatchClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method || r.URL.Path != tt.path {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &gotBody)
				_, _ = w.Write([]byte(`{"dat":7,"err":""}`))
			})
			getClient := func(context.Context) *client.Client { return c }

			res := callTool(t, tt.tool(getClient), tt.args)
			if res.IsError {
				t.Fatalf("tool errored: %s", resultText(res))
			}
			if !reflect.DeepEqual(gotBody["periodic_mutes"], want) {
				t.Fatalf("periodic_mutes = %v, want %v", gotBody["periodic_mutes"], want)
			}
		})
	}
}
