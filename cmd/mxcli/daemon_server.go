// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

// runDaemonServer listens on sockPath and dispatches each incoming request as a
// full mxcli command execution. onReady is called once the listener is bound
// (used by tests; pass nil in production).
//
// idleTimeout > 0: the server closes the listener and removes the socket file
// after that duration passes without any incoming connection. This allows
// per-MPR daemon processes to self-terminate when the project is idle.
// idleTimeout <= 0 disables the idle watcher (daemon runs until killed).
//
// The socket file is always removed on exit (via defer), so callers do not need
// to clean it up separately.
func runDaemonServer(sockPath string, idleTimeout time.Duration, onReady func()) {
	os.Remove(sockPath) // clean up stale socket
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli-daemon: listen %s: %v\n", sockPath, err)
		os.Exit(1)
	}
	defer func() {
		ln.Close()
		os.Remove(sockPath) // always remove socket on exit — prevents stale files
	}()

	var (
		mu      sync.Mutex
		lastReq = time.Now()
	)

	if idleTimeout > 0 {
		go func() {
			tick := time.NewTicker(idleTimeout / 4)
			defer tick.Stop()
			for range tick.C {
				mu.Lock()
				idle := time.Since(lastReq)
				mu.Unlock()
				if idle >= idleTimeout {
					ln.Close() // triggers Accept() to return an error → server loop exits
					return
				}
			}
		}()
	}

	if onReady != nil {
		onReady()
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed (idle timeout or external signal)
		}
		go func() {
			mu.Lock()
			lastReq = time.Now()
			mu.Unlock()
			handleConn(conn)
		}()
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	var req launcherproto.Request
	if err := launcherproto.ReadMsg(conn, &req); err != nil {
		return
	}

	// Health-check shortcut
	if len(req.Argv) == 1 && req.Argv[0] == "__healthcheck__" {
		v := version
		if Version != "" {
			v = Version
		}
		_ = launcherproto.WriteMsg(conn, launcherproto.Frame{OK: true, Version: v})
		return
	}

	// Redirect cobra output to the connection
	outW := &frameWriter{conn: conn, stream: "stdout"}
	errW := &frameWriter{conn: conn, stream: "stderr"}

	// Restore working directory
	if req.Cwd != "" {
		if err := os.Chdir(req.Cwd); err != nil {
			fmt.Fprintf(errW, "chdir %s: %v\n", req.Cwd, err)
		}
	}

	exitCode := runCommand(req.Argv, outW, errW)
	_ = launcherproto.WriteMsg(conn, launcherproto.Frame{Exit: &exitCode})
}

// runCommand executes the given argv via rootCmd, writing stdout/stderr to the
// provided writers. Returns the exit code.
func runCommand(argv []string, stdout, stderr io.Writer) int {
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(argv)
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

// frameWriter wraps a net.Conn and sends each Write as a Frame.
type frameWriter struct {
	conn   net.Conn
	stream string
}

func (fw *frameWriter) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	if err := launcherproto.WriteMsg(fw.conn, launcherproto.Frame{Stream: fw.stream, Data: buf}); err != nil {
		return 0, err
	}
	return len(p), nil
}
