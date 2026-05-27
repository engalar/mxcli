// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"net"
	"os"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

// runDaemonServer listens on sockPath and dispatches each incoming request as a
// full mxcli command execution. onReady is called once the listener is bound
// (used by tests; pass nil in production).
func runDaemonServer(sockPath string, onReady func()) {
	os.Remove(sockPath) // clean up stale socket
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli-daemon: listen %s: %v\n", sockPath, err)
		os.Exit(1)
	}
	defer ln.Close()

	if onReady != nil {
		onReady()
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		go handleConn(conn)
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
