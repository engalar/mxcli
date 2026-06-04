// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestM2EEAuthHeader(t *testing.T) {
	// M2EE expects base64-encoded password WITHOUT trailing newline
	got := m2eeAuthHeader("AdminPassword1!")
	want := base64.StdEncoding.EncodeToString([]byte("AdminPassword1!"))
	if got != want {
		t.Errorf("m2eeAuthHeader: got %q, want %q", got, want)
	}
	// Verify it decodes back correctly
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if string(decoded) != "AdminPassword1!" {
		t.Errorf("decoded: got %q, want %q", string(decoded), "AdminPassword1!")
	}
	// Verify no trailing newline
	if strings.HasSuffix(got, "Cg==") {
		t.Error("auth header has trailing newline in base64 (Cg==)")
	}
}

func TestParseEnvReader(t *testing.T) {
	input := `# This is a comment
APP_PORT=8080
ADMIN_PORT=8090
M2EE_ADMIN_PASS=AdminPassword1!

# Another comment
DB_NAME=mendix
QUOTED_VAL="hello world"
SINGLE_QUOTED='foo bar'
EMPTY_VAL=
SPACES_KEY = spaces_val
`
	result, err := parseEnvReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseEnvReader: %v", err)
	}

	tests := []struct {
		key, want string
	}{
		{"APP_PORT", "8080"},
		{"ADMIN_PORT", "8090"},
		{"M2EE_ADMIN_PASS", "AdminPassword1!"},
		{"DB_NAME", "mendix"},
		{"QUOTED_VAL", "hello world"},
		{"SINGLE_QUOTED", "foo bar"},
		{"EMPTY_VAL", ""},
		{"SPACES_KEY", "spaces_val"},
	}

	for _, tt := range tests {
		got, ok := result[tt.key]
		if !ok {
			t.Errorf("missing key %q", tt.key)
			continue
		}
		if got != tt.want {
			t.Errorf("key %q: got %q, want %q", tt.key, got, tt.want)
		}
	}

	// Comments should not appear as keys
	if _, ok := result["# This is a comment"]; ok {
		t.Error("comment parsed as key")
	}
}

func TestParseEnvReaderEmpty(t *testing.T) {
	result, err := parseEnvReader(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseEnvReader: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestResolveM2EEDefaults_FlagsPriority(t *testing.T) {
	opts := M2EEOptions{
		Host:  "myhost",
		Port:  9999,
		Token: "mytoken",
	}
	if err := resolveM2EEDefaults(&opts); err != nil {
		t.Fatalf("resolveM2EEDefaults: %v", err)
	}
	if opts.Host != "myhost" {
		t.Errorf("host: got %q, want %q", opts.Host, "myhost")
	}
	if opts.Port != 9999 {
		t.Errorf("port: got %d, want %d", opts.Port, 9999)
	}
	if opts.Token != "mytoken" {
		t.Errorf("token: got %q, want %q", opts.Token, "mytoken")
	}
}

func TestResolveM2EEDefaults_Defaults(t *testing.T) {
	opts := M2EEOptions{}
	err := resolveM2EEDefaults(&opts)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "admin password required") {
		t.Errorf("unexpected error: %v", err)
	}
	if opts.Host != "localhost" {
		t.Errorf("host: got %q, want %q", opts.Host, "localhost")
	}
	if opts.Port != 8090 {
		t.Errorf("port: got %d, want %d", opts.Port, 8090)
	}
}

func TestResolveM2EEDefaults_EnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	dockerDir := filepath.Join(tmpDir, ".docker")
	if err := os.MkdirAll(dockerDir, 0755); err != nil {
		t.Fatal(err)
	}
	envContent := "ADMIN_PORT=9191\nM2EE_ADMIN_PASS=envfilepass\n"
	if err := os.WriteFile(filepath.Join(dockerDir, ".env"), []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}

	opts := M2EEOptions{
		ProjectPath: filepath.Join(tmpDir, "app.mpr"),
	}
	if err := resolveM2EEDefaults(&opts); err != nil {
		t.Fatalf("resolveM2EEDefaults: %v", err)
	}
	if opts.Port != 9191 {
		t.Errorf("port: got %d, want %d", opts.Port, 9191)
	}
	if opts.Token != "envfilepass" {
		t.Errorf("token: got %q, want %q", opts.Token, "envfilepass")
	}
}

func TestResolveM2EEDefaults_EnvVarPriority(t *testing.T) {
	tmpDir := t.TempDir()
	dockerDir := filepath.Join(tmpDir, ".docker")
	if err := os.MkdirAll(dockerDir, 0755); err != nil {
		t.Fatal(err)
	}
	envContent := "ADMIN_PORT=9191\nM2EE_ADMIN_PASS=envfilepass\n"
	if err := os.WriteFile(filepath.Join(dockerDir, ".env"), []byte(envContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ADMIN_PORT", "7777")
	t.Setenv("M2EE_ADMIN_PASS", "envvarpass")

	opts := M2EEOptions{
		ProjectPath: filepath.Join(tmpDir, "app.mpr"),
	}
	if err := resolveM2EEDefaults(&opts); err != nil {
		t.Fatalf("resolveM2EEDefaults: %v", err)
	}
	if opts.Port != 7777 {
		t.Errorf("port: got %d, want %d", opts.Port, 7777)
	}
	if opts.Token != "envvarpass" {
		t.Errorf("token: got %q, want %q", opts.Token, "envvarpass")
	}
}

func TestCallM2EE_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong content type: %s", r.Header.Get("Content-Type"))
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["action"] != "test_action" {
			t.Errorf("wrong action: %v", body["action"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":0,"feedback":{"status":"running"}}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	resp, err := CallM2EE(M2EEOptions{
		Host:   host,
		Port:   port,
		Token:  "testpass",
		Direct: true,
	}, "test_action", nil)
	if err != nil {
		t.Fatalf("CallM2EE: %v", err)
	}

	if resp.Result != 0 {
		t.Errorf("result: got %d, want 0", resp.Result)
	}
	fb := resp.Feedback()
	if fb["status"] != "running" {
		t.Errorf("feedback.status: got %v, want running", fb["status"])
	}
}

func TestCallM2EE_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":1,"cause":"model not loaded"}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	resp, err := CallM2EE(M2EEOptions{
		Host:   host,
		Port:   port,
		Token:  "testpass",
		Direct: true,
	}, "reload_model", nil)
	if err != nil {
		t.Fatalf("CallM2EE: %v", err)
	}

	if resp.Result != 1 {
		t.Errorf("result: got %d, want 1", resp.Result)
	}
	if errMsg := resp.M2EEError(); errMsg != "model not loaded" {
		t.Errorf("M2EEError: got %q, want %q", errMsg, "model not loaded")
	}
}

func TestCallM2EE_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	_, err := CallM2EE(M2EEOptions{
		Host:   host,
		Port:   port,
		Token:  "wrongpass",
		Direct: true,
	}, "test_action", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCallM2EE_ConnectionRefused(t *testing.T) {
	// Use a port that nothing is listening on
	_, err := CallM2EE(M2EEOptions{
		Host:   "127.0.0.1",
		Port:   1, // unlikely to have a service
		Token:  "testpass",
		Direct: true,
	}, "test_action", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot connect") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCallM2EE_WithParams(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":0,"feedback":{}}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	_, err := CallM2EE(M2EEOptions{
		Host:   host,
		Port:   port,
		Token:  "testpass",
		Direct: true,
	}, "preview_execute_oql", map[string]any{
		"oql":            "SELECT 1",
		"numberHandling": "asString",
	})
	if err != nil {
		t.Fatalf("CallM2EE: %v", err)
	}

	if receivedBody["action"] != "preview_execute_oql" {
		t.Errorf("action: got %v, want preview_execute_oql", receivedBody["action"])
	}
	params, ok := receivedBody["params"].(map[string]any)
	if !ok {
		t.Fatal("params not a map")
	}
	if params["oql"] != "SELECT 1" {
		t.Errorf("params.oql: got %v, want SELECT 1", params["oql"])
	}
}

func TestM2EEResponse_M2EEError(t *testing.T) {
	tests := []struct {
		name string
		resp M2EEResponse
		want string
	}{
		{
			name: "success",
			resp: M2EEResponse{Result: 0},
			want: "",
		},
		{
			name: "error with cause",
			resp: M2EEResponse{Result: 1, Cause: "model error"},
			want: "model error",
		},
		{
			name: "error with message fallback",
			resp: M2EEResponse{Result: 1, Message: "something went wrong"},
			want: "something went wrong",
		},
		{
			name: "error with no message",
			resp: M2EEResponse{Result: 1},
			want: "unknown error",
		},
		{
			name: "cause takes priority over message",
			resp: M2EEResponse{Result: 1, Cause: "primary", Message: "secondary"},
			want: "primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.resp.M2EEError()
			if got != tt.want {
				t.Errorf("M2EEError: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDirectM2EECaller_Call(t *testing.T) {
	var gotAction string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		gotAction = body["action"].(string)
		auth := r.Header.Get("X-M2EE-Authentication")
		if auth == "" {
			t.Error("missing X-M2EE-Authentication header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":0,"feedback":{}}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)
	caller := &DirectM2EECaller{Host: host, Port: port, Token: "secret"}
	resp, err := caller.Call("update_styling", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Result != 0 {
		t.Errorf("result = %d, want 0", resp.Result)
	}
	if gotAction != "update_styling" {
		t.Errorf("action = %q, want update_styling", gotAction)
	}
}

func TestDirectM2EECaller_Defaults(t *testing.T) {
	c := &DirectM2EECaller{Token: "x"}
	if c.host() != "localhost" {
		t.Errorf("host() = %q, want localhost", c.host())
	}
	if c.port() != 8090 {
		t.Errorf("port() = %d, want 8090", c.port())
	}
}

func TestNewDockerExecM2EECaller_ExplicitToken(t *testing.T) {
	caller, err := NewDockerExecM2EECaller(t.TempDir(), "mytoken")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.Token != "mytoken" {
		t.Errorf("Token = %q, want mytoken", caller.Token)
	}
}

func TestNewDockerExecM2EECaller_EnvToken(t *testing.T) {
	t.Setenv("M2EE_ADMIN_PASS", "envtoken")
	caller, err := NewDockerExecM2EECaller(t.TempDir(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.Token != "envtoken" {
		t.Errorf("Token = %q, want envtoken", caller.Token)
	}
}

func TestNewDockerExecM2EECaller_NoToken_Error(t *testing.T) {
	t.Setenv("M2EE_ADMIN_PASS", "")
	_, err := NewDockerExecM2EECaller(t.TempDir(), "")
	if err == nil {
		t.Fatal("expected error when no token available")
	}
}

func TestNewDockerExecM2EECaller_DotEnvToken(t *testing.T) {
	t.Setenv("M2EE_ADMIN_PASS", "")
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte("M2EE_ADMIN_PASS=filetoken\n"), 0600)
	caller, err := NewDockerExecM2EECaller(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if caller.Token != "filetoken" {
		t.Errorf("Token = %q, want filetoken", caller.Token)
	}
}
