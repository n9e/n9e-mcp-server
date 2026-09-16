package types

import (
	"encoding/json"
	"testing"
)

// TestN9eResponsePayload pins which envelope key is used for the payload.
func TestN9eResponsePayload(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"ginx dat envelope", `{"dat":{"a":1},"err":""}`, `{"a":1}`},
		{"router data envelope", `{"data":{"a":2},"error":""}`, `{"a":2}`},
		{"dat wins when both present", `{"dat":{"a":3},"data":{"a":4}}`, `{"a":3}`},
		{"falls back to data when dat is null", `{"dat":null,"data":{"a":5}}`, `{"a":5}`},
		{"array payload", `{"data":[1,2],"error":""}`, `[1,2]`},
		{"no envelope key", `{"err":""}`, ``},
		{"null payload", `{"dat":null,"err":""}`, ``},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r N9eResponse
			if err := json.Unmarshal([]byte(tc.body), &r); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got := string(r.Payload()); got != tc.want {
				t.Fatalf("Payload() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestN9eResponseBusinessError pins how the two error spellings are read. A
// business error must never read as success: the data envelope reports failure
// on an HTTP 200 response, which is what made the datasource tools silent.
func TestN9eResponseBusinessError(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"no error key", `{"dat":{"a":1}}`, ``},
		{"empty ginx err", `{"dat":{"a":1},"err":""}`, ``},
		{"empty router error", `{"data":{"a":1},"error":""}`, ``},
		{"null router error", `{"data":{"a":1},"error":null}`, ``},
		{"ginx err", `{"dat":null,"err":"boom"}`, `boom`},
		{"router message", `{"error":{"message":"name already exists"}}`, `name already exists`},
		{"router bare string", `{"error":"unauthorized"}`, `unauthorized`},
		{
			// n9e forwards Go errors unchanged, so they arrive marshalled.
			name: "marshalled go error",
			body: `{"error":{"message":{"Number":1062,"SQLState":["23000"],"Message":"Duplicate entry 'sa-prom' for key 'datasource.name'"}}}`,
			want: "Duplicate entry 'sa-prom' for key 'datasource.name'",
		},
		{
			// A gin bind failure: a failure with no message must still read as one.
			name: "error without a usable message",
			body: `{"error":{"message":{}}}`,
			want: `{"message":{}}`,
		},
		{"err wins over error", `{"err":"first","error":{"message":"second"}}`, `first`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r N9eResponse
			if err := json.Unmarshal([]byte(tc.body), &r); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got := r.BusinessError(); got != tc.want {
				t.Fatalf("BusinessError() = %q, want %q", got, tc.want)
			}
		})
	}
}
