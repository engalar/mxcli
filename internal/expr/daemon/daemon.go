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

// IndexBuilder constructs a meta.Index from an MPR file path.
// The implementation is responsible for opening and closing any backend
// resources; it must not retain them after Build returns.
type IndexBuilder interface {
	BuildIndex(mprPath string) (*meta.Index, error)
}

// MprIndexBuilder is the production IndexBuilder. It opens the MPR backend,
// builds the index, then closes the backend before returning.
type MprIndexBuilder struct{}

func (b *MprIndexBuilder) BuildIndex(mprPath string) (*meta.Index, error) {
	be, err := mprbackend.NewFromPath(mprPath)
	if err != nil {
		return nil, fmt.Errorf("open mpr: %w", err)
	}
	idx, buildErr := meta.BuildFromBackend(be)
	_ = be.Disconnect() // always close, regardless of build outcome
	if buildErr != nil {
		return nil, fmt.Errorf("build index: %w", buildErr)
	}
	return idx, nil
}

// compile-time check.
var _ IndexBuilder = (*MprIndexBuilder)(nil)

// Daemon is a long-running process bound to one MPR file.
type Daemon struct {
	mprPath     string
	sockPath    string
	idleTimeout time.Duration
	builder     IndexBuilder // never nil

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
	return NewWithBuilder(mprPath, "", idleTimeout, &MprIndexBuilder{})
}

// NewWithSocket constructs a Daemon for mprPath bound to the given socket path.
// An empty socketPath falls back to SocketPath(mprPath). Intended for the
// `mxcli expr daemon start --socket <path>` CLI flag so DaemonClient and the
// spawned daemon agree on a non-default socket location.
func NewWithSocket(mprPath, socketPath string, idleTimeout time.Duration) (*Daemon, error) {
	return NewWithBuilder(mprPath, socketPath, idleTimeout, &MprIndexBuilder{})
}

// NewWithBuilder is the primary constructor. Pass a custom IndexBuilder to
// replace the production MPR backend — useful for unit tests that want to
// inject a pre-built meta.Index without touching the file system.
func NewWithBuilder(mprPath, socketPath string, idleTimeout time.Duration, builder IndexBuilder) (*Daemon, error) {
	if mprPath == "" {
		return nil, errors.New("daemon: mprPath is required")
	}
	if builder == nil {
		return nil, errors.New("daemon: builder is required")
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
		builder:     builder,
		lastReq:     time.Now(),
		stopCh:      make(chan struct{}),
	}
	return d, nil
}

// MprPath returns the absolute MPR path bound to this daemon.
func (d *Daemon) MprPath() string { return d.mprPath }

// SocketPath returns the Unix socket path this daemon serves on.
func (d *Daemon) SocketPath() string { return d.sockPath }

// rebuildIndex delegates to the injected IndexBuilder, replacing d.index.
func (d *Daemon) rebuildIndex() error {
	idx, err := d.builder.BuildIndex(d.mprPath)
	if err != nil {
		return fmt.Errorf("daemon: %w", err)
	}
	d.index = idx
	d.builtAt = time.Now()
	return nil
}

// Serve binds the Unix socket and accepts connections until Stop is called
// or the idle watcher triggers shutdown. Blocking call; run on a goroutine.
func (d *Daemon) Serve() error {
	// Bind first so IsAlive() becomes true immediately. The index build that
	// follows takes several seconds; connections arriving during that window are
	// queued by the OS and served once we enter the Accept loop.
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

	// Build index after binding so New() is instant. On failure, tear down
	// the socket so callers don't see a dead listener.
	if err := d.rebuildIndex(); err != nil {
		d.mu.Lock()
		d.listener = nil
		d.mu.Unlock()
		l.Close()
		os.Remove(d.sockPath) //nolint:errcheck
		return fmt.Errorf("daemon: build index: %w", err)
	}

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

// Stop closes the listener and removes the socket file.
func (d *Daemon) Stop() error {
	d.mu.Lock()
	if d.stopped {
		d.mu.Unlock()
		return nil
	}
	d.stopped = true
	close(d.stopCh)
	l := d.listener
	d.listener = nil
	d.mu.Unlock()

	if l != nil {
		_ = l.Close()
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
// and writes either a PingResponse (Type==ReqPing) or a ValidateResponse.
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

	// ReqPing (or legacy empty-MprPath) => status probe.
	if req.Type == ReqPing || (req.Type == "" && req.MprPath == "") {
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
	var records []scan.ExprRecord
	var err error
	if fi, statErr := os.Stat(mprcontents); statErr == nil && fi.IsDir() {
		records, err = scan.ScanMprcontents(mprcontents, scan.Options{FilterType: req.Filter})
	} else {
		records, err = scan.ScanMPR(d.mprPath, scan.Options{FilterType: req.Filter})
	}
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
