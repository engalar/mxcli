// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
)

// runDaemonCommand dispatches mxcli daemon <subcommand>.
func (e *Env) runDaemonCommand(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mxcli daemon <upgrade|rollback|status>")
		fmt.Fprintln(os.Stderr, "  upgrade   Download and install the latest mxcli-daemon")
		fmt.Fprintln(os.Stderr, "  rollback  Restore the previous mxcli-daemon version")
		fmt.Fprintln(os.Stderr, "  status    Show daemon running status")
		return 1
	}

	switch args[0] {
	case "upgrade":
		if err := e.upgradeComponent(e.daemonComponentConfig()); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli daemon upgrade: %v\n", err)
			return 1
		}
		return 0

	case "rollback":
		if err := e.rollbackComponent(e.daemonComponentConfig()); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli daemon rollback: %v\n", err)
			return 1
		}
		return 0

	case "status":
		return e.runDaemonStatus()

	default:
		fmt.Fprintf(os.Stderr, "mxcli daemon: unknown subcommand %q\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: mxcli daemon <upgrade|rollback|status>")
		return 1
	}
}

// runDaemonStatus prints whether the shared daemon is running.
func (e *Env) runDaemonStatus() int {
	sock := e.daemonSocketPath()
	if isDaemonRunning(sock) {
		ver, _ := e.healthCheck(sock)
		fmt.Printf("● mxcli-daemon running  version=%s  socket=%s\n", ver, sock)
	} else {
		ver := readVersionFile(e.daemonVersionPath())
		fmt.Printf("○ mxcli-daemon stopped  version=%s\n", ver)
	}
	return 0
}
