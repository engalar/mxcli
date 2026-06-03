// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"runtime"
	"testing"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

func newUpgradeEnv(t *testing.T, tag string, component string, content []byte) (*Env, *testfixtures.FakeGitHub) {
	t.Helper()
	payload, err := testfixtures.BuildComponentPayload(component, runtime.GOOS, runtime.GOARCH, content)
	if err != nil {
		t.Fatalf("BuildComponentPayload: %v", err)
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

func TestUpgradeComponent_Daemon_FreshInstall(t *testing.T) {
	e, _ := newUpgradeEnv(t, "daemon-v1.5.0", "mxcli-daemon", []byte("daemon-v150-bin"))

	cfg := e.daemonComponentConfig()
	if err := e.upgradeComponent(cfg); err != nil {
		t.Fatalf("upgradeComponent: %v", err)
	}

	got, err := os.ReadFile(cfg.BinPath)
	if err != nil {
		t.Fatalf("binary not installed: %v", err)
	}
	if string(got) != "daemon-v150-bin" {
		t.Errorf("content: got %q", got)
	}
	ver, _ := os.ReadFile(cfg.VersionPath)
	if string(ver) != "daemon-v1.5.0" {
		t.Errorf("version: got %q", ver)
	}
}

func TestUpgradeComponent_Local_FreshInstall(t *testing.T) {
	e, _ := newUpgradeEnv(t, "local-v0.2.0", "mxcli-local", []byte("local-v020-bin"))

	cfg := e.localComponentConfig()
	if err := e.upgradeComponent(cfg); err != nil {
		t.Fatalf("upgradeComponent: %v", err)
	}

	got, _ := os.ReadFile(cfg.BinPath)
	if string(got) != "local-v020-bin" {
		t.Errorf("content: got %q", got)
	}
}

func TestRollbackComponent_Daemon(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	cfg := e.daemonComponentConfig()

	// Pre-install v1 and v2 (v2 is current, v1 is backup)
	if err := os.MkdirAll(e.daemonDir(), 0700); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(cfg.BinPath, []byte("v2-bin"), 0755)
	os.WriteFile(cfg.BakPath, []byte("v1-bin"), 0755)
	os.WriteFile(cfg.VersionPath, []byte("daemon-v2.0.0"), 0644)
	os.WriteFile(e.daemonVersionBakPath(), []byte("daemon-v1.0.0"), 0644)

	if err := e.rollbackComponent(cfg); err != nil {
		t.Fatalf("rollbackComponent: %v", err)
	}

	got, _ := os.ReadFile(cfg.BinPath)
	if string(got) != "v1-bin" {
		t.Errorf("after rollback binary: got %q", got)
	}
	ver, _ := os.ReadFile(cfg.VersionPath)
	if string(ver) != "daemon-v1.0.0" {
		t.Errorf("after rollback version: got %q", ver)
	}
}
