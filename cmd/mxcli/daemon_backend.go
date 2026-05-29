// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// persistentDaemonBackend holds the pre-connected backend for the daemon process.
// Non-nil only in --serve + --mpr-path mode (per-MPR daemon).
// Kept as concrete type so duck-type checks (microflowsRepoProvider etc.) work.
// Never modified after main() sets it.
var persistentDaemonBackend *mprbackend.MprBackend

// daemonRequestMu serializes command execution on the persistent backend.
// SQLite is single-writer; os.Chdir is process-global — both require serialization.
var daemonRequestMu sync.Mutex

// noOpConnectBackend wraps *mprbackend.MprBackend (not the interface) so that
// duck-type checks like b.(microflowsRepoProvider) still succeed — the concrete
// type's Microflows() / Nanoflows() / etc. methods are promoted and visible.
// Only Connect / Disconnect / IsConnected are overridden to be no-ops so the
// persistent SQLite connection is never closed between daemon requests.
type noOpConnectBackend struct{ *mprbackend.MprBackend }

func (n *noOpConnectBackend) Connect(string) error { return nil }
func (n *noOpConnectBackend) Disconnect() error    { return nil }
func (n *noOpConnectBackend) IsConnected() bool    { return true }

// openPersistentBackend opens an MprBackend and returns it as FullBackend.
// Called once at daemon startup; the connection lives for the daemon's lifetime.
// EnableContentCache is called so mxunit file reads are cached in memory:
// the first request populates the cache; subsequent requests skip all file I/O.
func openPersistentBackend(mprPath string) (*mprbackend.MprBackend, error) {
	b := mprbackend.New()
	if err := b.Connect(mprPath); err != nil {
		return nil, fmt.Errorf("pre-connect %s: %w", mprPath, err)
	}
	b.EnableContentCache()
	return b, nil
}

// extractMPRPath scans args for "--mpr-path <path>" or "--mpr-path=<path>".
func extractMPRPath(args []string) string {
	const flag = "--mpr-path"
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(a, flag+"=") {
			return a[len(flag)+1:]
		}
	}
	return ""
}

// closePersistentBackend is called (deferred) when the daemon exits.
func closePersistentBackend() {
	if persistentDaemonBackend != nil {
		_ = persistentDaemonBackend.Disconnect()
		// Also log to stderr so it's visible in diagnostic output.
		fmt.Fprintln(os.Stderr, "mxcli-daemon: persistent backend closed")
	}
}
