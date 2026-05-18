// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

// ClientOptions configures a DaemonClient.
type ClientOptions struct {
	MprPath      string
	SocketPath   string
	StartTimeout time.Duration
}

// DaemonClient connects to a per-MPR daemon over its Unix socket and can
// implicitly start a fresh daemon process if none is running.
type DaemonClient struct {
	mprPath      string
	socketPath   string
	startTimeout time.Duration
}

// NewClient builds a DaemonClient. SocketPath defaults to SocketPath(MprPath);
// StartTimeout defaults to 30 seconds.
func NewClient(opts ClientOptions) *DaemonClient {
	sp := opts.SocketPath
	if sp == "" {
		sp = SocketPath(opts.MprPath)
	}
	timeout := opts.StartTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &DaemonClient{
		mprPath:      opts.MprPath,
		socketPath:   sp,
		startTimeout: timeout,
	}
}

// SocketPath exposes the resolved socket path (useful for diagnostics and tests).
func (c *DaemonClient) SocketPath() string { return c.socketPath }

// MprPath exposes the bound MPR path.
func (c *DaemonClient) MprPath() string { return c.mprPath }

// StartIfNeeded returns nil immediately when a daemon is already alive on the
// socket. Otherwise it forks `mxcli expr daemon start -p <mprPath>` and waits
// up to startTimeout for the socket to come online.
func (c *DaemonClient) StartIfNeeded() error {
	if IsAlive(c.socketPath) {
		return nil
	}
	if c.mprPath == "" {
		return errors.New("daemon client: cannot start daemon without MprPath")
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("daemon client: find executable: %w", err)
	}
	args := []string{"expr", "daemon", "start", "-p", c.mprPath}
	if c.socketPath != SocketPath(c.mprPath) {
		args = append(args, "--socket", c.socketPath)
	}
	cmd := exec.Command(exe, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("daemon client: start daemon: %w", err)
	}
	// Detach: don't wait synchronously. Reap the child in background to avoid
	// zombies if the daemon exits early.
	go func() { _ = cmd.Wait() }()

	deadline := time.Now().Add(c.startTimeout)
	for time.Now().Before(deadline) {
		if IsAlive(c.socketPath) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("daemon client: daemon did not start within %s", c.startTimeout)
}

// Ping sends an empty ValidateRequest, which the server interprets as a
// status probe, and decodes a PingResponse.
func (c *DaemonClient) Ping() (*PingResponse, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(ValidateRequest{}); err != nil {
		return nil, err
	}
	var resp PingResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Validate sends req to the daemon and decodes a ValidateResponse.
func (c *DaemonClient) Validate(req ValidateRequest) (*ValidateResponse, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, err
	}
	var resp ValidateResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopDaemon removes the socket file so subsequent IsAlive checks fail. The
// server-side daemon will exit at its next Accept(). Intended for tests; in
// production use a dedicated `mxcli expr daemon stop` CLI verb.
func (c *DaemonClient) StopDaemon() {
	_ = os.Remove(c.socketPath)
}
