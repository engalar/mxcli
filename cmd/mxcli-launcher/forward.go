// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

// forwardRequest connects to the daemon socket, sends the request, streams
// stdout/stderr to out/err, and returns the daemon's exit code.
// Returns 1 if the socket cannot be reached.
func forwardRequest(sockPath string, argv []string, out, err io.Writer) int {
	conn, dialErr := net.DialTimeout("unix", sockPath, 3*time.Second)
	if dialErr != nil {
		fmt.Fprintf(err, "mxcli: cannot connect to daemon (%v)\n", dialErr)
		fmt.Fprintf(err, "       Try: mxcli upgrade\n")
		return 1
	}
	defer conn.Close()

	cwd, _ := os.Getwd()
	env := captureEnv()

	req := launcherproto.Request{Argv: argv, Cwd: cwd, Env: env}
	if writeErr := launcherproto.WriteMsg(conn, req); writeErr != nil {
		fmt.Fprintf(err, "mxcli: send request: %v\n", writeErr)
		return 1
	}

	for {
		var frame launcherproto.Frame
		if readErr := launcherproto.ReadMsg(conn, &frame); readErr != nil {
			if readErr != io.EOF {
				fmt.Fprintf(err, "mxcli: read frame: %v\n", readErr)
			}
			return 1
		}
		switch {
		case frame.Exit != nil:
			return *frame.Exit
		case frame.Stream == "stdout":
			out.Write(frame.Data)
		case frame.Stream == "stderr":
			err.Write(frame.Data)
		case frame.Stream == "progress":
			fmt.Fprintf(err, "▶ %s\n", bytes.TrimRight(frame.Data, "\n"))
		}
	}
}

// captureEnv captures the current process environment as a map.
func captureEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				env[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return env
}
