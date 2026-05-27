// SPDX-License-Identifier: Apache-2.0

package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

func TestDaemonServer_HealthCheck(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "test.sock")

	ready := make(chan struct{})
	go func() {
		runDaemonServer(sockPath, func() { close(ready) })
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("daemon server did not start in time")
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := launcherproto.Request{Argv: []string{"__healthcheck__"}, Cwd: "/", Env: map[string]string{}}
	if err := launcherproto.WriteMsg(conn, req); err != nil {
		t.Fatalf("write request: %v", err)
	}

	var frame launcherproto.Frame
	if err := launcherproto.ReadMsg(conn, &frame); err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if !frame.OK {
		t.Errorf("health check: expected ok=true, got %+v", frame)
	}
	if frame.Version == "" {
		t.Error("health check: expected non-empty version")
	}
}

func TestDaemonServer_ExitCode(t *testing.T) {
	// Daemon's handleConn calls os.Chdir(req.Cwd), which mutates process-global
	// state. Restore it on test exit so we don't break later tests that rely on
	// relative paths (e.g. project_tree_test referencing ../../testdata/...).
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	sockPath := filepath.Join(t.TempDir(), "test2.sock")
	ready := make(chan struct{})
	go func() {
		runDaemonServer(sockPath, func() { close(ready) })
	}()
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := launcherproto.Request{
		Argv: []string{"--version"},
		Cwd:  t.TempDir(),
		Env:  map[string]string{},
	}
	if err := launcherproto.WriteMsg(conn, req); err != nil {
		t.Fatalf("write: %v", err)
	}

	var lastExit *int
	for {
		var frame launcherproto.Frame
		if err := launcherproto.ReadMsg(conn, &frame); err != nil {
			break
		}
		if frame.Exit != nil {
			lastExit = frame.Exit
			break
		}
	}
	if lastExit == nil {
		t.Fatal("expected exit frame, got none")
	}
	if *lastExit != 0 {
		t.Errorf("expected exit 0, got %d", *lastExit)
	}
}
