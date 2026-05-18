// SPDX-License-Identifier: Apache-2.0

// Package daemon implements a per-MPR validation daemon. One daemon process
// owns one MPR file, holds a hot meta.Index in memory, and serves validation
// requests over a Unix socket. Idle connections trigger an auto-stop.
package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/typecheck"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// Daemon is a long-running process bound to one MPR file.
type Daemon struct {
	mprPath     string
	sockPath    string
	idleTimeout time.Duration

	backend *mprbackend.MprBackend
	index   *meta.Index
	builtAt time.Time

	mu       sync.Mutex
	listener net.Listener
	lastReq  time.Time
	stopped  bool
	stopCh   chan struct{}
}

// New constructs a Daemon for mprPath using the default SocketPath(mprPath).
// The socket file is not yet bound — call Serve() to bind and accept connections.
func New(mprPath string, idleTimeout time.Duration) (*Daemon, error) {
	return NewWithSocket(mprPath, "", idleTimeout)
}

// NewWithSocket constructs a Daemon for mprPath bound to the given socket path.
// An empty socketPath falls back to SocketPath(mprPath). Intended for the
// `mxcli expr daemon start --socket <path>` CLI flag so DaemonClient and the
// spawned daemon agree on a non-default socket location.
func NewWithSocket(mprPath, socketPath string, idleTimeout time.Duration) (*Daemon, error) {
	if mprPath == "" {
		return nil, errors.New("daemon: mprPath is required")
	}
	abs, err := filepath.Abs(mprPath)
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve mprPath: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("daemon: stat mprPath: %w", err)
	}
	if err := EnsureDaemonDir(); err != nil {
		return nil, fmt.Errorf("daemon: create socket dir: %w", err)
	}
	sock := socketPath
	if sock == "" {
		sock = SocketPath(abs)
	}
	d := &Daemon{
		mprPath:     abs,
		sockPath:    sock,
		idleTimeout: idleTimeout,
		lastReq:     time.Now(),
		stopCh:      make(chan struct{}),
	}
	if err := d.rebuildIndex(); err != nil {
		return nil, err
	}
	return d, nil
}

// MprPath returns the absolute MPR path bound to this daemon.
func (d *Daemon) MprPath() string { return d.mprPath }

// SocketPath returns the Unix socket path this daemon serves on.
func (d *Daemon) SocketPath() string { return d.sockPath }

// rebuildIndex opens (or reopens) the MPR backend and builds a fresh Index.
func (d *Daemon) rebuildIndex() error {
	if d.backend != nil {
		_ = d.backend.Disconnect()
		d.backend = nil
	}
	b, err := mprbackend.NewFromPath(d.mprPath)
	if err != nil {
		return fmt.Errorf("daemon: open mpr: %w", err)
	}
	idx, err := meta.BuildFromBackend(b)
	if err != nil {
		_ = b.Disconnect()
		return fmt.Errorf("daemon: build index: %w", err)
	}
	d.backend = b
	d.index = idx
	d.builtAt = time.Now()
	return nil
}

// Serve binds the Unix socket and accepts connections until Stop is called
// or the idle watcher triggers shutdown. Blocking call; run on a goroutine.
func (d *Daemon) Serve() error {
	// Remove any stale socket file from a prior crash before binding.
	_ = os.Remove(d.sockPath)
	l, err := net.Listen("unix", d.sockPath)
	if err != nil {
		return fmt.Errorf("daemon: listen: %w", err)
	}
	// 0600 is required for security (skill: secure socket permissions).
	_ = os.Chmod(d.sockPath, 0o600)

	d.mu.Lock()
	d.listener = l
	d.stopped = false
	d.mu.Unlock()

	go d.idleWatcher()

	for {
		conn, acceptErr := l.Accept()
		if acceptErr != nil {
			d.mu.Lock()
			stopped := d.stopped
			d.mu.Unlock()
			if stopped {
				return nil
			}
			return acceptErr
		}
		go d.handleConn(conn)
	}
}

// Stop closes the listener, the backend, and removes the socket file.
func (d *Daemon) Stop() error {
	d.mu.Lock()
	if d.stopped {
		d.mu.Unlock()
		return nil
	}
	d.stopped = true
	close(d.stopCh)
	l := d.listener
	b := d.backend
	d.listener = nil
	d.backend = nil
	d.mu.Unlock()

	if l != nil {
		_ = l.Close()
	}
	if b != nil {
		_ = b.Disconnect()
	}
	_ = os.Remove(d.sockPath)
	return nil
}

// idleWatcher stops the daemon after idleTimeout passes without a request.
// A zero or negative timeout disables the watcher.
func (d *Daemon) idleWatcher() {
	if d.idleTimeout <= 0 {
		return
	}
	tick := time.NewTicker(d.idleTimeout / 4)
	defer tick.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-tick.C:
			d.mu.Lock()
			idle := time.Since(d.lastReq)
			d.mu.Unlock()
			if idle >= d.idleTimeout {
				_ = d.Stop()
				return
			}
		}
	}
}

// handleConn services one client connection: it decodes a ValidateRequest
// and writes either a PingResponse (empty MprPath) or a ValidateResponse.
func (d *Daemon) handleConn(conn net.Conn) {
	defer conn.Close()

	d.mu.Lock()
	d.lastReq = time.Now()
	d.mu.Unlock()

	var req ValidateRequest
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(ValidateResponse{Error: "bad request: " + err.Error()})
		return
	}

	// Empty MprPath => ping/status request.
	if req.MprPath == "" {
		_ = json.NewEncoder(conn).Encode(d.pingResponse())
		return
	}

	// Reject requests aimed at a different MPR.
	if req.MprPath != d.mprPath {
		_ = json.NewEncoder(conn).Encode(ValidateResponse{
			Error: fmt.Sprintf("mpr mismatch: daemon bound to %s, request asked for %s",
				d.mprPath, req.MprPath),
		})
		return
	}

	results, err := d.validate(req)
	if err != nil {
		_ = json.NewEncoder(conn).Encode(ValidateResponse{Error: err.Error()})
		return
	}
	_ = json.NewEncoder(conn).Encode(ValidateResponse{
		IndexAge: time.Since(d.builtAt).Truncate(time.Second).String(),
		Results:  results,
	})
}

func (d *Daemon) pingResponse() PingResponse {
	return PingResponse{
		OK:          true,
		MprPath:     d.mprPath,
		IndexAge:    time.Since(d.builtAt).Truncate(time.Second).String(),
		EntityCount: d.index.EntityCount(),
		EnumCount:   d.index.EnumCount(),
	}
}

// validate runs scan → parse-with-catalog → semantic validation, returning
// items filtered by req.Filter / req.Severity.
func (d *Daemon) validate(req ValidateRequest) ([]ValidationItem, error) {
	mprcontents := scan.MprContentsPath(d.mprPath)
	records, err := scan.ScanMprcontents(mprcontents, scan.Options{FilterType: req.Filter})
	if err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	parseResults := parse.BatchParseWithCatalog(records, d.index)

	wantSev := strings.ToUpper(strings.TrimSpace(req.Severity))
	var items []ValidationItem
	emit := func(vrs []validate.ValidationResult) {
		for _, vr := range vrs {
			if wantSev != "" && vr.Severity != wantSev {
				continue
			}
			items = append(items, ValidationItem{
				UnitID:   vr.UnitID,
				UnitType: vr.UnitType,
				UnitPath: vr.UnitPath,
				Location: d.index.UnitQN(vr.UnitPath),
				Field:    vr.Field,
				Raw:      vr.Raw,
				RuleID:   vr.RuleID,
				Severity: vr.Severity,
				Message:  vr.Message,
				Fix:      vr.Fix,
			})
		}
	}
	checker := typecheck.NewChecker(d.index)
	for _, pr := range parseResults {
		emit(validate.ValidateSyntax(pr))
		emit(validate.ValidateSemantic(pr, d.index))
		emit(checker.Check(pr))
		emit(checker.CheckStructural(pr))
	}
	if items == nil {
		items = []ValidationItem{}
	}
	return items, nil
}
