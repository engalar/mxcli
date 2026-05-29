// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	mprDaemonIdleTimeout    = 30 * time.Minute
	sharedDaemonIdleTimeout = 30 * time.Minute
	mprDaemonStartTimeout   = 30 * time.Second
)

// extractMPRFromArgs scans argv for -p <path> or --project <path> or --project=<path>.
func extractMPRFromArgs(args []string) string {
	for i, a := range args {
		if (a == "-p" || a == "--project") && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, "--project=") {
			return a[len("--project="):]
		}
	}
	return ""
}

// ensureMPRDaemon ensures a per-MPR daemon is running for mprAbsPath and returns its socket path.
//
// Lifecycle:
//   - Socket path is derived from (mprPath hash, binary mtime hash): changes on recompile → stale daemons self-invalidate.
//   - Stale sockets for the same MPR (old binary) are deleted; their idle watchers trigger exit.
//   - Daemon is spawned with --idle-timeout 5m; exits and removes socket after 5 min idle.
//   - go cmd.Wait() prevents zombie processes.
func (e *Env) ensureMPRDaemon(mprAbsPath string) (string, error) {
	sockPath := e.mprDaemonSocketPath(mprAbsPath)

	if isDaemonRunning(sockPath) {
		return sockPath, nil
	}

	// Clean stale sockets before spawning to avoid bind conflicts.
	e.cleanupStaleMPRSockets(mprAbsPath, sockPath)

	cmd := exec.Command(
		e.daemonBinaryPath(),
		"--serve", sockPath,
		"--idle-timeout", mprDaemonIdleTimeout.String(),
		"--mpr-path", mprAbsPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	hideDaemonWindow(cmd)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start mpr daemon: %w", err)
	}
	go func() { _ = cmd.Wait() }() // prevent zombie

	deadline := time.Now().Add(mprDaemonStartTimeout)
	for time.Now().Before(deadline) {
		if isDaemonRunning(sockPath) {
			return sockPath, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	return "", fmt.Errorf("mpr daemon did not start within %v", mprDaemonStartTimeout)
}

// cleanupStaleMPRSockets removes:
//  1. Same-MPR old-binary sockets (mpr hash prefix matches, but binary hash differs) — deleted
//     unconditionally; removing the socket stops the old daemon from accepting new connections,
//     and its idle watcher triggers exit shortly after.
//  2. Dead sockets from any other MPR (orphans where no process is listening).
//
// currentSock is always preserved.
func (e *Env) cleanupStaleMPRSockets(mprAbsPath, currentSock string) {
	entries, err := os.ReadDir(e.daemonDir())
	if err != nil {
		return
	}
	prefix := e.mprSocketPrefix(mprAbsPath)
	currentBase := filepath.Base(currentSock)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sock") || !strings.HasPrefix(name, "mpr-") {
			continue
		}
		if name == currentBase {
			continue
		}
		sockPath := filepath.Join(e.daemonDir(), name)
		if strings.HasPrefix(name, prefix) {
			// Same MPR, old binary: delete to stop it from accepting new connections.
			_ = os.Remove(sockPath)
		} else if !isDaemonRunning(sockPath) {
			// Another MPR's dead socket: clean up the orphan file.
			_ = os.Remove(sockPath)
		}
	}
}
