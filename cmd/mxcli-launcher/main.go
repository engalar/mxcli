// SPDX-License-Identifier: Apache-2.0

// mxcli is the launcher — a thin cross-platform client that forwards CLI
// requests to mxcli-daemon via unix socket. It handles daemon lifecycle,
// background version checks, and upgrade/rollback.
//
// Routing:
//   - TTY commands (tui, playwright, serve, oql): run directly by exec'ing the
//     daemon binary with inherited stdin/stdout/stderr so the full terminal is
//     available (mouse events, resize, raw keyboard input).
//   - Commands with -p <mpr>: forwarded to a per-MPR daemon (isolated process,
//     30-minute idle timeout, socket path derived from mpr path + binary mtime hash).
//   - Commands without -p: forwarded to the shared daemon at ~/.mxcli/daemon/mxcli.sock.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mendixlabs/mxcli/internal/launcherproto"
)

var (
	Version       = "dev"
	LauncherBuild = ""
)

func main() {
	e := DefaultEnv()
	args := os.Args[1:]

	// Clean up leftover .old binary from a previous self-upgrade on Windows.
	if selfPath, err := os.Executable(); err == nil {
		cleanupOldBinary(selfPath)
	}

	// --internal-update mode: spawned by runSelfUpgrade to replace the binary
	// after the parent process exits. Must be handled before all other cases.
	if len(args) > 0 && args[0] == "--internal-update" {
		pid, newBin, target, ok := parseInternalUpdateArgs(args[1:])
		if !ok {
			fmt.Fprintln(os.Stderr, "mxcli: invalid --internal-update args")
			os.Exit(1)
		}
		if err := runInternalUpdate(pid, newBin, target, &RealPIDWaiter{}, 30*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli: update failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(args) > 0 {
		switch args[0] {
		case "upgrade":
			os.Exit(e.runSelfUpgrade(args[1:]))
		case "rollback":
			os.Exit(e.runRollback(args[1:]))
		case "daemon":
			os.Exit(e.runDaemonCommand(args[1:]))
		case "version", "--version":
			printVersion(e)
			os.Exit(0)
		case "local":
			// Local commands are delegated directly to mxcli-local binary.
			// They bypass the daemon — no daemon needed for PAD build/run.
			os.Exit(e.runLocal(args[1:]))
		}
	}

	// Ensure daemon binary is present (download if needed).
	if err := e.ensureDaemonBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
		os.Exit(1)
	}

	// TTY commands need a real terminal (stdin, stdout, stderr attached to the
	// calling process). Run them by exec'ing the daemon binary directly instead
	// of forwarding through the Unix socket, where stdin is disconnected and
	// stdout is JSON-framed (which corrupts escape codes).
	if isTTYCommand(args) {
		launcherPath, _ := os.Executable()
		cmd := exec.Command(e.daemonBinaryPath(), args...)
		cmd.Env = ttyEnv(launcherPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "mxcli: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
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

// ttyCommands is the set of subcommands that require a real TTY.
var ttyCommands = map[string]bool{
	"tui": true, "serve": true, "oql": true, "playwright": true,
	// "new" creates a Mendix project with multi-step progress output (Steps 1-4)
	// and requires long-running file operations; socket forwarding drops streaming
	// output and causes the daemon to emit help text instead of executing.
	// "setup" downloads mxbuild with progress output and also must run directly.
	"new": true, "setup": true,
}

// flagsWithValue lists root-level flags whose next token is their value, not a
// subcommand. Knowing these lets isTTYCommand skip over -p <path>, etc.
var flagsWithValue = map[string]bool{
	"-p": true, "--project": true,
	"-c": true, "--command": true,
}

// isTTYCommand reports whether args requests a command that requires a real
// terminal (interactive TUI, browser-launched viewer, streaming output).
// These commands are exec'd directly with inherited stdin/stdout/stderr instead
// of being forwarded through the Unix socket, where the terminal is unavailable.
//
// Handles both orderings: "tui -p file.mpr" and "-p file.mpr tui".
func isTTYCommand(args []string) bool {
	skipNext := false
	for _, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if flagsWithValue[a] {
			skipNext = true // consume the flag's value token
			continue
		}
		if len(a) > 0 && a[0] == '-' {
			continue // other flags (booleans, --key=val)
		}
		// First positional token is the subcommand name.
		return ttyCommands[a]
	}
	return false
}

// ttyEnv returns the environment to pass to TTY-command subprocesses.
// It propagates the current process environment and injects MXCLI_LAUNCHER_PATH
// so that the daemon binary (exec'd for tui/serve/oql/playwright) knows the
// launcher path and can route its internal subcommand calls back through it.
// Any pre-existing MXCLI_LAUNCHER_PATH is replaced to avoid stale values.
func ttyEnv(launcherPath string) []string {
	prefix := launcherproto.EnvLauncherPath + "="
	base := os.Environ()
	result := make([]string, 0, len(base)+1)
	for _, e := range base {
		if !strings.HasPrefix(e, prefix) {
			result = append(result, e)
		}
	}
	return append(result, prefix+launcherPath)
}

func printVersion(e *Env) {
	v := Version
	if LauncherBuild != "" {
		v += " (" + LauncherBuild + ")"
	}
	fmt.Printf("mxcli launcher %s\n", v)
	if daemonBinaryExists(e.daemonBinaryPath()) {
		if out, err := exec.Command(e.daemonBinaryPath(), "--version").Output(); err == nil {
			fmt.Printf("mxcli daemon   %s\n", strings.TrimSpace(string(out)))
		}
	}
}
