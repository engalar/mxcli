// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDeployHOCON_DTAPModeOff(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	err := writeDeployHOCON(path, deployConfig{
		Configuration: map[string]string{
			"DatabaseType": "HSQLDB",
			"DatabaseName": "default",
		},
		Constants: map[string]string{},
	}, "", "Admin123!", 8080, 8090, "D")
	if err != nil {
		t.Fatalf("writeDeployHOCON: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "DTAPMode = D") {
		t.Errorf("expected DTAPMode = D for security=off, got:\n%s", content)
	}
	if !strings.Contains(content, "DTAPMode = ${?RUNTIME_PARAMS_DTAPMODE}") {
		t.Error("should include env var override line")
	}
}

func TestWriteDeployHOCON_DTAPModeDemo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	err := writeDeployHOCON(path, deployConfig{
		Configuration: map[string]string{
			"DatabaseType": "HSQLDB",
			"DatabaseName": "default",
		},
		Constants: map[string]string{},
	}, "", "Admin123!", 8080, 8090, "T")
	if err != nil {
		t.Fatalf("writeDeployHOCON: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "DTAPMode = T") {
		t.Errorf("expected DTAPMode = T for security=demo, got:\n%s", content)
	}
}

func TestWriteDeployHOCON_DTAPModeProduction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.conf")

	err := writeDeployHOCON(path, deployConfig{
		Configuration: map[string]string{
			"DatabaseType": "HSQLDB",
			"DatabaseName": "default",
		},
		Constants: map[string]string{},
	}, "", "Admin123!", 8080, 8090, "P")
	if err != nil {
		t.Fatalf("writeDeployHOCON: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "DTAPMode = P") {
		t.Errorf("expected DTAPMode = P for security=production, got:\n%s", content)
	}
}

func TestSecurityDTAPValue_Mapping(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{"off", "D"},
		{"", "D"},
		{"demo", "T"},
		{"production", "P"},
		{"PRODUCTION", "P"},
		{"DEMO", "T"},
	}
	for _, tc := range tests {
		opts := LocalRunOptions{SecurityMode: tc.mode}
		got := opts.securityDTAPValue()
		if got != tc.want {
			t.Errorf("securityDTAPValue(%q) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}
