// SPDX-License-Identifier: Apache-2.0

// mxcli is the launcher — a thin cross-platform client that forwards CLI
// requests to mxcli-daemon via unix socket. It handles daemon lifecycle,
// background version checks, and upgrade/rollback.
//
// Routing:
//   - Commands with -p <mpr>: forwarded to a per-MPR daemon (isolated process,
//     5-minute idle timeout, socket path derived from mpr path + binary mtime hash).
//   - Commands without -p: forwarded to the shared daemon at ~/.mxcli/daemon/mxcli.sock.
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	Version       = "dev"
	LauncherBuild = ""
)

func main() {
	e := DefaultEnv()
	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "upgrade":
			os.Exit(e.runUpgrade(args[1:]))
		case "rollback":
			os.Exit(e.runRollback(args[1:]))
		case "version", "--version":
			printVersion(e)
			os.Exit(0)
		}
	}

	// Ensure daemon binary is present (download if needed).
	if err := e.ensureDaemonBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
		os.Exit(1)
	}

	// Route to per-MPR daemon when -p is present; otherwise use shared daemon.
	var sockPath string
	if rawMPR := extractMPRFromArgs(args); rawMPR != "" {
		absMPR, err := filepath.Abs(rawMPR)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mxcli: resolve -p path: %v\n", err)
			os.Exit(1)
		}
		absMPR = filepath.ToSlash(absMPR)
		sp, err := e.ensureMPRDaemon(absMPR)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
			os.Exit(1)
		}
		sockPath = sp
	} else {
		if !isDaemonRunning(e.daemonSocketPath()) {
			if err := e.startDaemon(); err != nil {
				fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
				os.Exit(1)
			}
		}
		sockPath = e.daemonSocketPath()
	}

	go e.backgroundVersionCheck()

	exitCode := forwardRequest(sockPath, args, os.Stdout, os.Stderr)

	e.printUpdateNotice()

	os.Exit(exitCode)
}

func printVersion(e *Env) {
	v := Version
	if LauncherBuild != "" {
		v += " (" + LauncherBuild + ")"
	}
	daemonVer := readVersionFile(e.daemonVersionPath())
	fmt.Printf("mxcli launcher %s\n", v)
	if daemonVer != "" {
		fmt.Printf("mxcli daemon   %s\n", daemonVer)
	}
}
