// SPDX-License-Identifier: Apache-2.0

package launcherproto

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// EnvLauncherPath is the environment variable the launcher injects when exec'ing
// the daemon binary for TTY commands (tui, serve, oql, playwright). The TUI reads
// this to route its internal subcommand calls back through the launcher so they
// benefit from per-MPR daemon routing rather than opening SQLite directly.
const EnvLauncherPath = "MXCLI_LAUNCHER_PATH"

// Request is sent from launcher to daemon over the unix socket.
type Request struct {
	Argv []string          `json:"argv"`
	Cwd  string            `json:"cwd"`
	Env  map[string]string `json:"env"`
}

// Frame is streamed from daemon to launcher.
// Exactly one of (Stream+Data), (Exit), or (OK+Version) is set per frame.
type Frame struct {
	// stdout/stderr stream frame
	Stream string `json:"stream,omitempty"` // "stdout" or "stderr"
	Data   []byte `json:"data,omitempty"`   // raw bytes (JSON encodes as base64)

	// Terminal frame: daemon finished
	Exit *int `json:"exit,omitempty"`

	// Health-check response
	OK      bool   `json:"ok,omitempty"`
	Version string `json:"version,omitempty"`
}

// WriteMsg serialises v as JSON and writes it preceded by a 4-byte big-endian length.
func WriteMsg(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("launcherproto marshal: %w", err)
	}
	if len(b) > 1<<24 {
		return fmt.Errorf("launcherproto: message too large (%d bytes)", len(b))
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// ReadMsg reads a length-prefixed JSON message from r and unmarshals into v.
func ReadMsg(r io.Reader, v any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > 1<<24 {
		return fmt.Errorf("launcherproto: incoming message too large (%d bytes)", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	return json.Unmarshal(buf, v)
}
