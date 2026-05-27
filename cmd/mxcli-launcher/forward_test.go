// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

// startEchoServer is a minimal server that echoes argv[0] back as stdout and exits 0.
func startEchoServer(t *testing.T) string {
	t.Helper()
	sockPath := filepath.Join(t.TempDir(), "echo.sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				var req launcherproto.Request
				launcherproto.ReadMsg(c, &req)
				if len(req.Argv) > 0 {
					launcherproto.WriteMsg(c, launcherproto.Frame{
						Stream: "stdout",
						Data:   []byte(req.Argv[0] + "\n"),
					})
				}
				code := 0
				launcherproto.WriteMsg(c, launcherproto.Frame{Exit: &code})
			}(conn)
		}
	}()
	time.Sleep(10 * time.Millisecond)
	return sockPath
}

func TestForwardRequest_CapturesStdout(t *testing.T) {
	sockPath := startEchoServer(t)

	var stdout, stderr bytes.Buffer
	exitCode := forwardRequest(sockPath, []string{"hello"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}
	if stdout.String() != "hello\n" {
		t.Errorf("expected stdout 'hello\\n', got %q", stdout.String())
	}
}

func TestForwardRequest_NoServer(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := forwardRequest(filepath.Join(t.TempDir(), "nosuch.sock"), []string{"x"}, &stdout, &stderr)
	if code == 0 {
		t.Error("expected non-zero exit code when daemon not running")
	}
}
