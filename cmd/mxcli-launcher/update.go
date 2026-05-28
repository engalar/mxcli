// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const updateCheckInterval = time.Hour

// backgroundVersionCheck checks GitHub for a newer daemon version (at most
// once per hour) and writes update-available if found. Runs in a goroutine.
func backgroundVersionCheck() {
	defer func() { recover() }()

	if !shouldCheckUpdate(daemonLastCheckPath()) {
		return
	}
	writeTimestamp(daemonLastCheckPath())

	latest, err := fetchLatestTag()
	if err != nil {
		return
	}
	current := readVersionFile(daemonVersionPath())
	if current != "" && latest != "" && latest != current {
		os.WriteFile(daemonUpdateAvailablePath(), []byte(latest), 0644)
	}
}

// shouldCheckUpdate returns true if last-check is older than 1h or missing.
func shouldCheckUpdate(lastCheckPath string) bool {
	b, err := os.ReadFile(lastCheckPath)
	if err != nil {
		return true
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return true
	}
	return time.Since(time.Unix(ts, 0)) > updateCheckInterval
}

// writeTimestamp writes the current Unix timestamp to path.
func writeTimestamp(path string) {
	os.WriteFile(path, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)
}

// fprintUpdateNotice prints the update notice to w and then removes the marker
// file at p. Printing before deletion ensures the notice survives a crash between
// the two operations (file stays → shown again next run).
func fprintUpdateNotice(w io.Writer, p string) {
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "\n🆕 mxcli-daemon %s available → run: mxcli upgrade\n", strings.TrimSpace(string(b)))
	os.Remove(p)
}

// printUpdateNotice checks update-available and prints a notice if present,
// then removes the marker file.
func printUpdateNotice() {
	fprintUpdateNotice(os.Stderr, daemonUpdateAvailablePath())
}

// runUpgrade downloads the latest daemon, backs up the current one (N-1),
// health-checks the new daemon, and rolls back on failure.
// The backup is always retained — only overwritten by the next upgrade.
func runUpgrade(_ []string) int {
	fmt.Println("Checking for updates...")
	latest, err := fetchLatestTag()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: fetch latest tag: %v\n", err)
		return 1
	}
	current := readVersionFile(daemonVersionPath())
	if current == latest {
		fmt.Printf("mxcli daemon is already at %s — nothing to do.\n", current)
		return 0
	}
	fmt.Printf("Upgrading daemon %s → %s\n", current, latest)

	tmpDest := daemonBinaryPath() + ".new"
	if err := downloadDaemonVersion(latest, tmpDest); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: download: %v\n", err)
		return 1
	}

	if daemonBinaryExists(daemonBinaryPath()) {
		os.Rename(daemonVersionPath(), daemonVersionBakPath())
		if err := os.Rename(daemonBinaryPath(), daemonBakPath()); err != nil {
			fmt.Fprintf(os.Stderr, "mxcli upgrade: backup current: %v\n", err)
			os.Remove(tmpDest)
			return 1
		}
	}

	if err := os.Rename(tmpDest, daemonBinaryPath()); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli upgrade: install: %v\n", err)
		rollback()
		return 1
	}
	os.WriteFile(daemonVersionPath(), []byte(latest), 0644)

	fmt.Print("Verifying new daemon...")
	sock := daemonSocketPath()
	os.Remove(sock)
	if err := startDaemon(); err != nil {
		fmt.Printf(" FAILED: %v\n", err)
		fmt.Println("Rolling back to previous version...")
		rollback()
		return 1
	}
	if _, err := healthCheck(sock); err != nil {
		fmt.Printf(" FAILED: %v\n", err)
		fmt.Println("Rolling back to previous version...")
		rollback()
		return 1
	}
	fmt.Println(" OK")
	fmt.Printf("✅ Upgraded to %s (previous version kept as backup)\n", latest)
	os.Remove(daemonUpdateAvailablePath())
	return 0
}

// runRollback implements `mxcli rollback [--list]`.
func runRollback(args []string) int {
	if len(args) > 0 && args[0] == "--list" {
		current := readVersionFile(daemonVersionPath())
		bak := readVersionFile(daemonVersionBakPath())
		fmt.Printf("current: %s\n", current)
		if bak != "" {
			fmt.Printf("backup:  %s  (run 'mxcli rollback' to restore)\n", bak)
		} else {
			fmt.Println("backup:  (none)")
		}
		return 0
	}

	if !daemonBinaryExists(daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli rollback: no backup available")
		return 1
	}

	bakVer := readVersionFile(daemonVersionBakPath())
	curVer := readVersionFile(daemonVersionPath())
	fmt.Printf("Rolling back daemon %s → %s\n", curVer, bakVer)

	killRunningDaemon()

	tmpBin := daemonBinaryPath() + ".rb-tmp"
	tmpVer := daemonVersionPath() + ".rb-tmp"
	os.Rename(daemonBinaryPath(), tmpBin)
	os.Rename(daemonVersionPath(), tmpVer)
	os.Rename(daemonBakPath(), daemonBinaryPath())
	os.Rename(daemonVersionBakPath(), daemonVersionPath())
	os.Rename(tmpBin, daemonBakPath())
	os.Rename(tmpVer, daemonVersionBakPath())
	if err := startDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli rollback: restart daemon: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Rolled back to %s\n", bakVer)
	return 0
}
