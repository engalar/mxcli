// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

// newInstallEnv creates a self-contained Env + FakeGitHub for install/update tests.
// daemonContent is what gets packaged as the fake binary (e.g. []byte("v0.15.0-binary")).
// Returns both Env (for launcher calls) and FakeGitHub (for RequestLog assertions).
func newInstallEnv(t *testing.T, cfg *testfixtures.FakeGitHub, daemonContent []byte) (*Env, *testfixtures.FakeGitHub) {
	t.Helper()
	payload, err := testfixtures.BuildDaemonPayload(daemonContent)
	if err != nil {
		t.Fatalf("BuildDaemonPayload: %v", err)
	}
	cfg.Payload = payload
	gh := testfixtures.NewFakeGitHub(t, cfg)
	e := &Env{
		HomeDir:    t.TempDir(),
		HTTPClient: gh.Client(),
	}
	if err := os.MkdirAll(e.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}
	return e, gh
}

// writeFakeDaemon plants a fake daemon binary + version file in e's daemon dir.
func writeFakeDaemon(t *testing.T, e *Env, version string) {
	t.Helper()
	if err := os.WriteFile(e.daemonBinaryPath(), []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.daemonVersionPath(), []byte(version+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

// — Core path tests —

func TestDownloadDaemon_FreshInstall(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("v015-binary"))

	if err := e.downloadDaemon(e.daemonBinaryPath()); err != nil {
		t.Fatalf("downloadDaemon: %v", err)
	}

	got, err := os.ReadFile(e.daemonBinaryPath())
	if err != nil {
		t.Fatalf("daemon binary not written: %v", err)
	}
	if string(got) != "v015-binary" {
		t.Errorf("binary content = %q, want %q", got, "v015-binary")
	}
}

func TestDownloadDaemonVersion_AlreadyLatest(t *testing.T) {
	t.Parallel()
	const tag = "v0.15.0"
	e, gh := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: tag}, []byte("binary"))
	writeFakeDaemon(t, e, tag)

	// Simulate what runUpgrade does: fetch latest, compare, skip download if equal.
	latest, err := e.fetchLatestTag()
	if err != nil {
		t.Fatalf("fetchLatestTag: %v", err)
	}
	current := readVersionFile(e.daemonVersionPath())
	if current != latest {
		t.Fatalf("test setup: current=%q latest=%q should match", current, latest)
	}

	// Assert: only the releases/latest endpoint was called — no download or SHA256SUMS.
	for _, path := range gh.RequestLog() {
		if strings.Contains(path, ".tar.zst") || strings.Contains(path, "SHA256SUMS") {
			t.Errorf("unexpected download request when already at latest: %s", path)
		}
	}
}

func TestDownloadDaemonVersion_UpgradeOldToNew(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("v015-binary"))
	writeFakeDaemon(t, e, "v0.14.0")

	// Write bak so backup rename succeeds
	if err := os.Rename(e.daemonBinaryPath(), e.daemonBakPath()); err != nil {
		t.Fatal(err)
	}
	os.Rename(e.daemonVersionPath(), e.daemonVersionBakPath())

	tmpDest := e.daemonBinaryPath() + ".new"
	if err := e.downloadDaemonVersion("v0.15.0", tmpDest); err != nil {
		t.Fatalf("downloadDaemonVersion: %v", err)
	}

	// Simulate the rename that runUpgrade does
	if err := os.Rename(tmpDest, e.daemonBinaryPath()); err != nil {
		t.Fatalf("install rename: %v", err)
	}
	if err := os.WriteFile(e.daemonVersionPath(), []byte("v0.15.0"), 0644); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(e.daemonBinaryPath())
	if string(got) != "v015-binary" {
		t.Errorf("new binary content = %q, want %q", got, "v015-binary")
	}
	bakVer := readVersionFile(e.daemonVersionBakPath())
	if bakVer != "v0.14.0" {
		t.Errorf("bak version = %q, want v0.14.0", bakVer)
	}
	newVer := readVersionFile(e.daemonVersionPath())
	if newVer != "v0.15.0" {
		t.Errorf("new version = %q, want v0.15.0", newVer)
	}
}

// — Throttle tests —

func TestBackgroundVersionCheck_ThrottledWithinHour(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	// Write a last-check timestamp 30 minutes ago
	ts := time.Now().Add(-30 * time.Minute).Unix()
	os.WriteFile(e.daemonLastCheckPath(), []byte(strconv.FormatInt(ts, 10)), 0644)

	// shouldCheckUpdate should return false
	if shouldCheckUpdate(e.daemonLastCheckPath()) {
		t.Error("should not check within 1h")
	}
}

func TestBackgroundVersionCheck_AllowedAfterHour(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	// Write a last-check timestamp 2 hours ago
	ts := time.Now().Add(-2 * time.Hour).Unix()
	os.WriteFile(e.daemonLastCheckPath(), []byte(strconv.FormatInt(ts, 10)), 0644)

	if !shouldCheckUpdate(e.daemonLastCheckPath()) {
		t.Error("should check after 1h")
	}
}

func TestBackgroundVersionCheck_NoLastCheckFile(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))

	// No last-check file — should allow check and write one
	if !shouldCheckUpdate(e.daemonLastCheckPath()) {
		t.Error("should check when last-check absent")
	}
	writeTimestamp(e.daemonLastCheckPath())
	if _, err := os.Stat(e.daemonLastCheckPath()); err != nil {
		t.Error("last-check file should exist after writeTimestamp")
	}
}

func TestBackgroundVersionCheck_WritesUpdateAvailable(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	e.backgroundVersionCheck()

	content, err := os.ReadFile(e.daemonUpdateAvailablePath())
	if err != nil {
		t.Fatalf("update-available not written: %v", err)
	}
	if strings.TrimSpace(string(content)) != "v0.15.0" {
		t.Errorf("update-available content = %q, want v0.15.0", content)
	}
}

func TestBackgroundVersionCheck_SkipsWhenAlreadyLatest(t *testing.T) {
	t.Parallel()
	const tag = "v0.15.0"
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: tag}, []byte("bin"))
	writeFakeDaemon(t, e, tag)

	e.backgroundVersionCheck()

	if _, err := os.Stat(e.daemonUpdateAvailablePath()); !os.IsNotExist(err) {
		t.Error("update-available should not be written when already at latest")
	}
}
