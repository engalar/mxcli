// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestReload_CSSOnly(t *testing.T) {
	t.Parallel()
	var receivedAction string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		receivedAction = body["action"].(string)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":0,"feedback":{}}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	var stdout bytes.Buffer
	opts := ReloadOptions{
		CSSOnly: true,
		Caller:  &DirectM2EECaller{Host: host, Port: port, Token: "testpass"},
		Stdout:  &stdout,
	}

	err := Reload(opts)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if receivedAction != "update_styling" {
		t.Errorf("expected update_styling action, got %q", receivedAction)
	}
	if !strings.Contains(stdout.String(), "Styling updated") {
		t.Errorf("expected 'Styling updated' in output, got: %s", stdout.String())
	}
}

func TestReload_ModelOnly(t *testing.T) {
	t.Parallel()
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		action := body["action"].(string)
		actions = append(actions, action)

		w.Header().Set("Content-Type", "application/json")
		switch action {
		case "reload_model":
			w.Write([]byte(`{"result":0,"feedback":{"startup_metrics":{"duration":98}}}`))
		case "get_ddl_commands":
			w.Write([]byte(`{"result":0,"feedback":{}}`))
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	var stdout bytes.Buffer
	opts := ReloadOptions{
		SkipBuild: true,
		Caller:    &DirectM2EECaller{Host: host, Port: port, Token: "testpass"},
		Stdout:    &stdout,
	}

	err := Reload(opts)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if len(actions) < 2 || actions[0] != "reload_model" || actions[1] != "get_ddl_commands" {
		t.Errorf("expected actions [reload_model, get_ddl_commands], got %v", actions)
	}
	if !strings.Contains(stdout.String(), "Model reloaded") {
		t.Errorf("expected 'Model reloaded' in output, got: %s", stdout.String())
	}
}

func TestReload_ParseDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		feedback map[string]any
		want     string
	}{
		{
			name:     "nil feedback",
			feedback: nil,
			want:     "",
		},
		{
			name:     "no startup_metrics",
			feedback: map[string]any{},
			want:     "",
		},
		{
			name: "duration in ms",
			feedback: map[string]any{
				"startup_metrics": map[string]any{
					"duration": float64(98),
				},
			},
			want: "98ms",
		},
		{
			name: "duration in seconds",
			feedback: map[string]any{
				"startup_metrics": map[string]any{
					"duration": float64(2500),
				},
			},
			want: "2.5s",
		},
		{
			name: "duration as string",
			feedback: map[string]any{
				"startup_metrics": map[string]any{
					"duration": "150ms",
				},
			},
			want: "150ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractReloadDuration(tt.feedback)
			if got != tt.want {
				t.Errorf("extractReloadDuration: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReload_ModelOnly_WithDuration(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		switch body["action"].(string) {
		case "reload_model":
			w.Write([]byte(`{"result":0,"feedback":{"startup_metrics":{"duration":98}}}`))
		case "get_ddl_commands":
			w.Write([]byte(`{"result":0,"feedback":{}}`))
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	var stdout bytes.Buffer
	opts := ReloadOptions{
		SkipBuild: true,
		Caller:    &DirectM2EECaller{Host: host, Port: port, Token: "testpass"},
		Stdout:    &stdout,
	}

	err := Reload(opts)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if !strings.Contains(stdout.String(), "98ms") {
		t.Errorf("expected duration in output, got: %s", stdout.String())
	}
}

func TestReload_CSSOnly_Error(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":1,"cause":"runtime not running"}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	var stdout bytes.Buffer
	opts := ReloadOptions{
		CSSOnly: true,
		Caller:  &DirectM2EECaller{Host: host, Port: port, Token: "testpass"},
		Stdout:  &stdout,
	}

	err := Reload(opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "runtime not running") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReload_ModelOnly_PendingDDL(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		switch body["action"].(string) {
		case "reload_model":
			w.Write([]byte(`{"result":0,"feedback":{}}`))
		case "get_ddl_commands":
			w.Write([]byte(`{"result":0,"feedback":{"ddl_commands":"CREATE TABLE mymodule$customer (id BIGINT NOT NULL);\nALTER TABLE mymodule$order ADD COLUMN status VARCHAR(200);"}}`))
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	var stdout bytes.Buffer
	opts := ReloadOptions{
		SkipBuild: true,
		Caller:    &DirectM2EECaller{Host: host, Port: port, Token: "testpass"},
		Stdout:    &stdout,
	}

	err := Reload(opts)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "WARNING: Database schema changes detected") {
		t.Errorf("expected DDL warning in output, got: %s", output)
	}
	if !strings.Contains(output, "CREATE TABLE") {
		t.Errorf("expected DDL commands in output, got: %s", output)
	}
	if !strings.Contains(output, "docker up --fresh") {
		t.Errorf("expected fix suggestion in output, got: %s", output)
	}
}

func TestReload_ModelOnly_ReloadError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":1,"cause":"model contains errors"}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	var stdout bytes.Buffer
	opts := ReloadOptions{
		SkipBuild: true,
		Caller:    &DirectM2EECaller{Host: host, Port: port, Token: "testpass"},
		Stdout:    &stdout,
	}

	err := Reload(opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "model contains errors") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReload_Frontend_SkipsWhenNoRollupConfig(t *testing.T) {
	t.Parallel()
	// When --frontend is set but no rollup.config.mjs exists, Reload should
	// complete normally without running frontend build.
	var actions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		action := body["action"].(string)
		actions = append(actions, action)

		w.Header().Set("Content-Type", "application/json")
		switch action {
		case "reload_model":
			w.Write([]byte(`{"result":0,"feedback":{}}`))
		case "get_ddl_commands":
			w.Write([]byte(`{"result":0,"feedback":{"ddl_commands":""}}`))
		default:
			w.Write([]byte(`{"result":0,"feedback":{}}`))
		}
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	dir := t.TempDir()
	projectPath := filepath.Join(dir, "app.mpr")

	var stdout bytes.Buffer
	opts := ReloadOptions{
		ProjectPath: projectPath,
		SkipBuild:   true,
		Frontend:    true,
		Caller:      &DirectM2EECaller{Host: host, Port: port, Token: "testpass"},
		Stdout:      &stdout,
		Stderr:      &stdout,
	}

	err := Reload(opts)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if strings.Contains(stdout.String(), "React client") {
		t.Errorf("unexpected frontend build output: %s", stdout.String())
	}

	if len(actions) == 0 || actions[0] != "reload_model" {
		t.Errorf("expected reload_model action, got %v", actions)
	}
}
