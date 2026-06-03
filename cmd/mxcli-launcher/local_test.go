// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

func TestLocalDir(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	want := filepath.Join(e.HomeDir, ".mxcli", "local")
	if got := e.localDir(); got != want {
		t.Errorf("localDir: got %s, want %s", got, want)
	}
}

func TestLocalBinaryPath(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	p := e.localBinaryPath()
	base := filepath.Base(p)
	if runtime.GOOS == "windows" {
		if base != "mxcli-local.exe" {
			t.Errorf("Windows: got %s, want mxcli-local.exe", base)
		}
	} else {
		if base != "mxcli-local" {
			t.Errorf("non-Windows: got %s, want mxcli-local", base)
		}
	}
}

func newLocalInstallEnv(t *testing.T, tag string, content []byte) (*Env, *testfixtures.FakeGitHub) {
	t.Helper()
	payload, err := testfixtures.BuildLocalPayload(runtime.GOOS, runtime.GOARCH, content)
	if err != nil {
		t.Fatalf("BuildLocalPayload: %v", err)
	}
	cfg := &testfixtures.FakeGitHub{
		LatestTag: tag,
		Payload: &testfixtures.DaemonPayload{
			AssetName: payload.AssetName,
			Archive:   payload.Archive,
			Checksum:  payload.Checksum,
		},
	}
	gh := testfixtures.NewFakeGitHub(t, cfg)
	e := &Env{HomeDir: t.TempDir(), HTTPClient: gh.Client()}
	return e, gh
}

func TestEnsureLocalBinary_FreshInstall(t *testing.T) {
	e, _ := newLocalInstallEnv(t, "local-v0.1.0", []byte("local-binary-content"))

	if err := e.ensureLocalBinary(); err != nil {
		t.Fatalf("ensureLocalBinary: %v", err)
	}

	content, err := os.ReadFile(e.localBinaryPath())
	if err != nil {
		t.Fatalf("binary not installed: %v", err)
	}
	if string(content) != "local-binary-content" {
		t.Errorf("binary content mismatch: got %q", content)
	}
}

func TestEnsureLocalBinary_AlreadyInstalled(t *testing.T) {
	e, gh := newLocalInstallEnv(t, "local-v0.1.0", []byte("binary"))

	// Pre-install
	if err := e.ensureLocalBinary(); err != nil {
		t.Fatalf("first install: %v", err)
	}
	initialRequests := len(gh.RequestLog())

	// Second call should not download again
	if err := e.ensureLocalBinary(); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(gh.RequestLog()) != initialRequests {
		t.Errorf("expected no new requests, got %d new", len(gh.RequestLog())-initialRequests)
	}
}
