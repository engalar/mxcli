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
	done := make(chan struct{})
	go func() {
		defer close(done)
		runDaemonServer(sockPath, 500*time.Millisecond, func() { close(ready) })
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("daemon server did not start in time")
	}
	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	})

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

func TestDaemonServer_IdleTimeout(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "idle.sock")
	ready := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		runDaemonServer(sockPath, 100*time.Millisecond, func() { close(ready) })
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}

	// Wait for idle timeout + a margin.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not exit after idle timeout")
	}

	// Socket file must be removed after idle exit.
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Error("socket file should be removed after idle timeout")
	}
}

func TestDaemonServer_IdleResetOnRequest(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "idle-reset.sock")
	ready := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		runDaemonServer(sockPath, 200*time.Millisecond, func() { close(ready) })
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}

	// Connect at ~100ms (before idle would fire) to reset the timer.
	time.Sleep(100 * time.Millisecond)
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.Close()

	// Server should still be alive for another ~200ms after the last request.
	select {
	case <-done:
		t.Error("server exited too early (idle timer was not reset)")
	case <-time.After(50 * time.Millisecond):
		// Good: server is still running 50ms after the request.
	}
}

func TestDaemonServer_ExitCode(t *testing.T) {
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	sockPath := filepath.Join(t.TempDir(), "test2.sock")
	ready := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		runDaemonServer(sockPath, 500*time.Millisecond, func() { close(ready) })
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

	// Daemon's handleConn calls os.Chdir(req.Cwd) which mutates the process CWD.
	// Cleanup order (LIFO): done-wait → os.Chdir-restore → TempDir-delete.
	// This ordering avoids "file in use" errors on Windows when the temp dir is
	// deleted while it is still the process CWD.
	reqCwd := t.TempDir()
	// Register cleanups AFTER the CWD TempDir so they run BEFORE its deletion.
	t.Cleanup(func() { _ = os.Chdir(origCwd) })
	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	})

	req := launcherproto.Request{
		Argv: []string{"--version"},
		Cwd:  reqCwd,
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
