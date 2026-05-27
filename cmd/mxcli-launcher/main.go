// SPDX-License-Identifier: Apache-2.0

// mxcli is the launcher — a thin cross-platform client that forwards CLI
// requests to mxcli-daemon via unix socket. It handles daemon lifecycle,
// background version checks, and upgrade/rollback.
package main

import (
	"fmt"
	"os"
)

// Version and LauncherBuild are set by ldflags at build time.
var (
	Version       = "dev"
	LauncherBuild = ""
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "upgrade":
			os.Exit(runUpgrade(args[1:]))
		case "rollback":
			os.Exit(runRollback(args[1:]))
		case "version", "--version":
			printVersion()
			os.Exit(0)
		}
	}

	if err := ensureDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
		os.Exit(1)
	}

	go backgroundVersionCheck()

	exitCode := forwardRequest(daemonSocketPath(), args, os.Stdout, os.Stderr)

	printUpdateNotice()

	os.Exit(exitCode)
}

func printVersion() {
	v := Version
	if LauncherBuild != "" {
		v += " (" + LauncherBuild + ")"
	}
	daemonVer := readVersionFile(daemonVersionPath())
	fmt.Printf("mxcli launcher %s\n", v)
	if daemonVer != "" {
		fmt.Printf("mxcli daemon   %s\n", daemonVer)
	}
}
