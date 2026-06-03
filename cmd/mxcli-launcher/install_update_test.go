// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/cmd/mxcli-launcher/testfixtures"
)

// newInstallEnv creates a self-contained Env + FakeGitHub for install/update tests.
// daemonContent is what gets packaged as the fake binary (e.g. []byte("v0.15.0-binary")).
// Returns both Env (for launcher calls) and FakeGitHub (for RequestLog assertions).
func newInstallEnv(t *testing.T, cfg *testfixtures.FakeGitHub, daemonContent []byte) (*Env, *testfixtures.FakeGitHub) {
	t.Helper()
	return newInstallEnvForPlatform(t, cfg, daemonContent, runtime.GOOS, runtime.GOARCH)
}

// newInstallEnvForPlatform is like newInstallEnv but targets an explicit platform,
// enabling Linux CI to exercise Windows download/extraction paths.
func newInstallEnvForPlatform(t *testing.T, cfg *testfixtures.FakeGitHub, daemonContent []byte, goos, goarch string) (*Env, *testfixtures.FakeGitHub) {
	t.Helper()
	payload, err := testfixtures.BuildDaemonPayloadForPlatform(goos, goarch, daemonContent)
	if err != nil {
		t.Fatalf("BuildDaemonPayloadForPlatform: %v", err)
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
	// LatestTag must be daemon-v* so fetchLatestTagWithPrefix("daemon-v") finds it.
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "daemon-v0.15.0"}, []byte("v015-binary"))

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
	// LatestTag must be daemon-v* so fetchLatestTagWithPrefix("daemon-v") finds it.
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "daemon-v0.15.0"}, []byte("bin"))
	writeFakeDaemon(t, e, "daemon-v0.14.0")

	e.backgroundVersionCheck()

	content, err := os.ReadFile(e.daemonUpdateAvailablePath())
	if err != nil {
		t.Fatalf("update-available not written: %v", err)
	}
	if strings.TrimSpace(string(content)) != "daemon-v0.15.0" {
		t.Errorf("update-available content = %q, want daemon-v0.15.0", content)
	}
}

func TestBackgroundVersionCheck_SkipsWhenAlreadyLatest(t *testing.T) {
	t.Parallel()
	const tag = "daemon-v0.15.0"
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: tag}, []byte("bin"))
	writeFakeDaemon(t, e, tag)

	e.backgroundVersionCheck()

	if _, err := os.Stat(e.daemonUpdateAvailablePath()); !os.IsNotExist(err) {
		t.Error("update-available should not be written when already at latest")
	}
}

// — Failure recovery tests —

func TestDownloadDaemonVersion_CorruptBinary(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{
		LatestTag:     "v0.15.0",
		CorruptBinary: true,
	}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	originalBin, _ := os.ReadFile(e.daemonBinaryPath())
	originalVer := readVersionFile(e.daemonVersionPath())

	tmpDest := e.daemonBinaryPath() + ".new"
	err := e.downloadDaemonVersion("v0.15.0", tmpDest)
	if err == nil {
		t.Fatal("expected checksum error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("error should mention checksum mismatch, got: %v", err)
	}

	// Original binary and version must be unchanged
	gotBin, _ := os.ReadFile(e.daemonBinaryPath())
	if string(gotBin) != string(originalBin) {
		t.Error("original binary must be untouched after checksum failure")
	}
	if readVersionFile(e.daemonVersionPath()) != originalVer {
		t.Error("original version must be untouched after checksum failure")
	}
	// Temp file must be cleaned up
	if _, err := os.Stat(tmpDest); !os.IsNotExist(err) {
		t.Error("temp download file should be removed after failure")
	}
}

func TestDownloadDaemonVersion_DownloadTruncated(t *testing.T) {
	t.Parallel()
	// Use larger content so the resulting archive exceeds the cut threshold.
	largeContent := make([]byte, 4096)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{
		LatestTag:   "v0.15.0",
		DownloadCut: 64,
	}, largeContent)

	tmpDest := e.daemonBinaryPath() + ".new"
	err := e.downloadDaemonVersion("v0.15.0", tmpDest)
	if err == nil {
		t.Fatal("expected error on truncated download, got nil")
	}
	// Temp files cleaned up
	if _, err2 := os.Stat(tmpDest); !os.IsNotExist(err2) {
		t.Error("temp download file should be removed after truncation")
	}
	archiveTmp := tmpDest + ".archive-dl"
	if _, err2 := os.Stat(archiveTmp); !os.IsNotExist(err2) {
		t.Error("archive temp file should be removed after truncation")
	}
}

func TestDownloadDaemonVersion_GitHub500(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{
		LatestTag:  "v0.15.0",
		StatusCode: 500,
	}, []byte("bin"))
	writeFakeDaemon(t, e, "v0.14.0")

	originalBin, _ := os.ReadFile(e.daemonBinaryPath())

	tmpDest := e.daemonBinaryPath() + ".new"
	err := e.downloadDaemonVersion("v0.15.0", tmpDest)
	if err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}

	gotBin, _ := os.ReadFile(e.daemonBinaryPath())
	if string(gotBin) != string(originalBin) {
		t.Error("original binary must be untouched after HTTP 500")
	}
}

func TestRollback_RestoresBakVersion(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))

	// Set up: current = v0.15.0 binary, bak = v0.14.0 binary
	os.WriteFile(e.daemonBinaryPath(), []byte("new-binary"), 0755)
	os.WriteFile(e.daemonVersionPath(), []byte("v0.15.0"), 0644)
	os.WriteFile(e.daemonBakPath(), []byte("old-binary"), 0755)
	os.WriteFile(e.daemonVersionBakPath(), []byte("v0.14.0"), 0644)

	e.rollback()

	gotBin, _ := os.ReadFile(e.daemonBinaryPath())
	if string(gotBin) != "old-binary" {
		t.Errorf("after rollback binary = %q, want old-binary", gotBin)
	}
	if ver := readVersionFile(e.daemonVersionPath()); ver != "v0.14.0" {
		t.Errorf("after rollback version = %q, want v0.14.0", ver)
	}
}

func TestRollback_NoBak(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("bin"))
	os.WriteFile(e.daemonBinaryPath(), []byte("current"), 0755)

	// Should not panic; logs a message but leaves current binary intact
	e.rollback()

	got, _ := os.ReadFile(e.daemonBinaryPath())
	if string(got) != "current" {
		t.Error("current binary should be untouched when no backup exists")
	}
}

func TestFetchAssetChecksum_GitHub500(t *testing.T) {
	t.Parallel()
	e, _ := newInstallEnv(t, &testfixtures.FakeGitHub{
		LatestTag:  "v0.15.0",
		StatusCode: 500,
	}, []byte("bin"))

	_, err := e.fetchAssetChecksum("v0.15.0", "mxcli-daemon-linux-amd64.tar.zst")
	if err == nil {
		t.Fatal("expected error on HTTP 500 for SHA256SUMS")
	}
}

// — Concurrent upgrade tests —

func TestRunUpgrade_ConcurrentOnlyOneWins(t *testing.T) {
	t.Parallel()

	// Build a payload that downloadDaemonVersion will accept
	payload, err := testfixtures.BuildDaemonPayload([]byte("v015-binary"))
	if err != nil {
		t.Fatalf("BuildDaemonPayload: %v", err)
	}

	// Two separate Envs sharing the same HomeDir — they race on the same lock file
	home := t.TempDir()
	makeEnv := func() *Env {
		gh := testfixtures.NewFakeGitHub(t, &testfixtures.FakeGitHub{
			LatestTag: "v0.15.0",
			Payload:   payload,
		})
		e := &Env{HomeDir: home, HTTPClient: gh.Client()}
		return e
	}

	e1 := makeEnv()
	e2 := makeEnv()
	if err := os.MkdirAll(e1.daemonDir(), 0755); err != nil {
		t.Fatal(err)
	}
	// Plant current version
	os.WriteFile(e1.daemonBinaryPath(), []byte("old"), 0755)
	os.WriteFile(e1.daemonVersionPath(), []byte("v0.14.0"), 0644)

	results := make([]error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); results[0] = e1.acquireUpgradeLock() }()
	go func() { defer wg.Done(); results[1] = e2.acquireUpgradeLock() }()
	wg.Wait()

	successes := 0
	for i, err := range results {
		if err == nil {
			successes++
			if i == 0 {
				e1.releaseUpgradeLock()
			} else {
				e2.releaseUpgradeLock()
			}
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 lock acquisition, got %d", successes)
	}
}

// — Cross-platform download path tests —
// These run on any host OS so Linux CI can exercise Windows packaging.

func TestDownloadDaemonVersion_WindowsZipOnLinuxCI(t *testing.T) {
	t.Parallel()
	// Regression test: make release was packing "mxcli-daemon-windows-amd64.exe" inside
	// the zip instead of "mxcli-daemon.exe", causing "no file named found in zip archive".
	// BuildDaemonPayloadForPlatform("windows",...) creates a zip with "mxcli-daemon.exe"
	// inside (the correct post-fix layout), so this test fails if the extraction logic
	// or the fixture naming drifts apart again.
	e, _ := newInstallEnvForPlatform(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("win-binary"), "windows", "amd64")

	dest := e.daemonBinaryPath()
	if err := e.downloadDaemonVersionForPlatform("v0.15.0", dest, "windows", "amd64"); err != nil {
		t.Fatalf("Windows download failed: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("daemon binary not written: %v", err)
	}
	if string(got) != "win-binary" {
		t.Errorf("binary content = %q, want win-binary", got)
	}
}

func TestDownloadDaemonVersion_LinuxTarZst(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tar.zst extraction not compiled on Windows; Linux CI covers this path")
	}
	t.Parallel()
	// Mirrors TestDownloadDaemon_FreshInstall but explicitly targets Linux/amd64
	// so Unix hosts exercise the tar.zst path.
	e, _ := newInstallEnvForPlatform(t, &testfixtures.FakeGitHub{LatestTag: "v0.15.0"}, []byte("linux-binary"), "linux", "amd64")

	dest := e.daemonBinaryPath()
	if err := e.downloadDaemonVersionForPlatform("v0.15.0", dest, "linux", "amd64"); err != nil {
		t.Fatalf("Linux download failed: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("daemon binary not written: %v", err)
	}
	if string(got) != "linux-binary" {
		t.Errorf("binary content = %q, want linux-binary", got)
	}
}
