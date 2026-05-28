// SPDX-License-Identifier: Apache-2.0

// mxcli is the launcher — a thin cross-platform client that forwards CLI
// requests to mxcli-daemon via unix socket. It handles daemon lifecycle,
// background version checks, and upgrade/rollback.
package main

import (
	"fmt"
	"os"
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

	if err := e.ensureDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
		os.Exit(1)
	}

	go e.backgroundVersionCheck()

	exitCode := forwardRequest(e.daemonSocketPath(), args, os.Stdout, os.Stderr)

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
