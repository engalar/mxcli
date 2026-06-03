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

func (e *Env) backgroundVersionCheck() {
	defer func() { recover() }()

	if !shouldCheckUpdate(e.daemonLastCheckPath()) {
		return
	}
	writeTimestamp(e.daemonLastCheckPath())

	latest, err := e.fetchLatestTag()
	if err != nil {
		return
	}
	current := readVersionFile(e.daemonVersionPath())
	if current != "" && latest != "" && latest != current {
		os.WriteFile(e.daemonUpdateAvailablePath(), []byte(latest), 0644)
	}
}

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

func writeTimestamp(path string) {
	os.WriteFile(path, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)
}

func fprintUpdateNotice(w io.Writer, p string) {
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "\n🆕 mxcli-daemon %s available → run: mxcli upgrade\n", strings.TrimSpace(string(b)))
	os.Remove(p)
}

func (e *Env) printUpdateNotice() {
	fprintUpdateNotice(os.Stderr, e.daemonUpdateAvailablePath())
}

func (e *Env) runUpgrade(_ []string) int {
	// runUpgrade now upgrades the daemon component.
	// Launcher self-upgrade is handled by runSelfUpgrade (--internal-update flow).
	if err := e.acquireUpgradeLock(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli daemon upgrade: %v\n", err)
		return 1
	}
	defer e.releaseUpgradeLock()

	if err := e.upgradeComponent(e.daemonComponentConfig()); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli daemon upgrade: %v\n", err)
		return 1
	}
	return 0
}

func (e *Env) runRollback(args []string) int {
	if len(args) > 0 && args[0] == "--list" {
		current := readVersionFile(e.daemonVersionPath())
		bak := readVersionFile(e.daemonVersionBakPath())
		fmt.Printf("current: %s\n", current)
		if bak != "" {
			fmt.Printf("backup:  %s  (run 'mxcli rollback' to restore)\n", bak)
		} else {
			fmt.Println("backup:  (none)")
		}
		return 0
	}

	if !daemonBinaryExists(e.daemonBakPath()) {
		fmt.Fprintln(os.Stderr, "mxcli rollback: no backup available")
		return 1
	}

	bakVer := readVersionFile(e.daemonVersionBakPath())
	curVer := readVersionFile(e.daemonVersionPath())
	fmt.Printf("Rolling back daemon %s → %s\n", curVer, bakVer)

	e.killRunningDaemon()

	tmpBin := e.daemonBinaryPath() + ".rb-tmp"
	tmpVer := e.daemonVersionPath() + ".rb-tmp"
	os.Rename(e.daemonBinaryPath(), tmpBin)
	os.Rename(e.daemonVersionPath(), tmpVer)
	os.Rename(e.daemonBakPath(), e.daemonBinaryPath())
	os.Rename(e.daemonVersionBakPath(), e.daemonVersionPath())
	os.Rename(tmpBin, e.daemonBakPath())
	os.Rename(tmpVer, e.daemonVersionBakPath())
	if err := e.startDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "mxcli rollback: restart daemon: %v\n", err)
		return 1
	}
	fmt.Printf("✅ Rolled back to %s\n", bakVer)
	return 0
}
