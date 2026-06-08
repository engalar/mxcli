// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRollupConfigExists_Present(t *testing.T) {
	dir := t.TempDir()
	webDir := filepath.Join(dir, "web")
	os.MkdirAll(webDir, 0755)
	os.WriteFile(filepath.Join(webDir, "rollup.config.mjs"), []byte("export default {}"), 0644)

	if !RollupConfigExists(dir) {
		t.Error("expected RollupConfigExists to return true when rollup.config.mjs present")
	}
}

func TestRollupConfigExists_Absent(t *testing.T) {
	dir := t.TempDir()
	if RollupConfigExists(dir) {
		t.Error("expected RollupConfigExists to return false when rollup.config.mjs absent")
	}
}

func TestResolveNodeExeForPlatform(t *testing.T) {
	base := "/mxbuild/modeler"
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{"windows", "amd64", "tools/node/win-x64/node.exe"},
		{"linux", "amd64", "tools/node/linux-x64/node"},
		{"darwin", "amd64", "tools/node/darwin-x64/node"},
		{"darwin", "arm64", "tools/node/darwin-arm64/node"},
	}

	for _, tc := range tests {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			got := resolveNodeExeForPlatform(base, tc.goos, tc.goarch)
			gotNorm := filepath.ToSlash(got)
			wantNorm := filepath.ToSlash(filepath.Join(base, tc.want))
			if gotNorm != wantNorm {
				t.Errorf("got %q, want %q", gotNorm, wantNorm)
			}
		})
	}
}

func TestResolveNodeExeForPlatform_Unknown(t *testing.T) {
	got := resolveNodeExeForPlatform("/base", "plan9", "arm")
	if got != "" {
		t.Errorf("expected empty string for unknown platform, got %q", got)
	}
}

func TestBuildFrontend_NodeExeNotFound(t *testing.T) {
	dir := t.TempDir()
	webDir := filepath.Join(dir, "web")
	os.MkdirAll(webDir, 0755)
	os.WriteFile(filepath.Join(webDir, "rollup.config.mjs"), []byte(""), 0644)

	var out bytes.Buffer
	err := BuildFrontend(FrontendBuildOptions{
		DeployDir:  dir,
		MxBuildDir: filepath.Join(dir, "nonexistent-modeler"),
		Stdout:     &out,
	})
	if err == nil {
		t.Fatal("expected error when node exe does not exist")
	}
	if !strings.Contains(err.Error(), "bundled node not found") {
		t.Errorf("expected 'bundled node not found' in error, got: %v", err)
	}
}

func TestBuildFrontend_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fake node not supported on Windows in unit tests")
	}

	dir := t.TempDir()
	mxbuildDir := filepath.Join(dir, "modeler")

	nodeExePath := resolveNodeExeForPlatform(mxbuildDir, runtime.GOOS, runtime.GOARCH)
	if nodeExePath == "" {
		t.Skipf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	os.MkdirAll(filepath.Dir(nodeExePath), 0755)
	os.WriteFile(nodeExePath, []byte("#!/bin/sh\nexit 0\n"), 0755)

	runnerDir := filepath.Join(mxbuildDir, "tools", "node")
	os.WriteFile(filepath.Join(runnerDir, "rollup-runner.mjs"), []byte(""), 0644)

	webDir := filepath.Join(dir, "web")
	os.MkdirAll(webDir, 0755)
	os.WriteFile(filepath.Join(webDir, "rollup.config.mjs"), []byte(""), 0644)

	var out bytes.Buffer
	err := BuildFrontend(FrontendBuildOptions{
		DeployDir:  dir,
		MxBuildDir: mxbuildDir,
		Stdout:     &out,
	})
	if err != nil {
		t.Fatalf("BuildFrontend: %v\nOutput: %s", err, out.String())
	}
	if !strings.Contains(out.String(), "successfully") {
		t.Errorf("expected 'successfully' in output, got: %s", out.String())
	}
}
