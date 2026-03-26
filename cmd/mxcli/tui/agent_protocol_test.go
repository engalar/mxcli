package tui

import (
	"encoding/json"
	"testing"
)

func TestAgentRequestParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    AgentRequest
		wantErr bool
	}{
		{
			name:  "exec action",
			input: `{"id":1,"action":"exec","mdl":"SHOW ENTITIES"}`,
			want:  AgentRequest{ID: 1, Action: "exec", MDL: "SHOW ENTITIES"},
		},
		{
			name:  "check action",
			input: `{"id":2,"action":"check"}`,
			want:  AgentRequest{ID: 2, Action: "check"},
		},
		{
			name:  "state action",
			input: `{"id":3,"action":"state"}`,
			want:  AgentRequest{ID: 3, Action: "state"},
		},
		{
			name:  "navigate action",
			input: `{"id":4,"action":"navigate","target":"MyModule.Customer"}`,
			want:  AgentRequest{ID: 4, Action: "navigate", Target: "MyModule.Customer"},
		},
		{
			name:    "invalid json",
			input:   `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got AgentRequest
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAgentResponseSerialization(t *testing.T) {
	tests := []struct {
		name string
		resp AgentResponse
		want string
	}{
		{
			name: "success response",
			resp: AgentResponse{ID: 1, OK: true, Result: "3 entities found"},
			want: `{"id":1,"ok":true,"result":"3 entities found"}`,
		},
		{
			name: "error response",
			resp: AgentResponse{ID: 2, OK: false, Error: "syntax error at line 1"},
			want: `{"id":2,"ok":false,"error":"syntax error at line 1"}`,
		},
		{
			name: "response with mode",
			resp: AgentResponse{ID: 3, OK: true, Mode: "browser"},
			want: `{"id":3,"ok":true,"mode":"browser"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestAgentRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     AgentRequest
		wantErr string
	}{
		{
			name:    "missing id",
			req:     AgentRequest{Action: "check"},
			wantErr: "missing request id",
		},
		{
			name: "valid check",
			req:  AgentRequest{ID: 1, Action: "check"},
		},
		{
			name: "valid state",
			req:  AgentRequest{ID: 2, Action: "state"},
		},
		{
			name:    "exec without mdl",
			req:     AgentRequest{ID: 3, Action: "exec"},
			wantErr: "exec action requires mdl field",
		},
		{
			name: "valid exec",
			req:  AgentRequest{ID: 4, Action: "exec", MDL: "SHOW ENTITIES"},
		},
		{
			name:    "navigate without target",
			req:     AgentRequest{ID: 5, Action: "navigate"},
			wantErr: "navigate action requires target field",
		},
		{
			name: "valid navigate",
			req:  AgentRequest{ID: 6, Action: "navigate", Target: "MyModule.Entity"},
		},
		{
			name:    "unknown action",
			req:     AgentRequest{ID: 7, Action: "destroy"},
			wantErr: `unknown action: "destroy"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("got error %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
