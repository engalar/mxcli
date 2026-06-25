package tui

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// shortSockPath returns a socket path short enough for Unix domain sockets
// (max 104-108 chars depending on OS). t.TempDir() on macOS produces paths
// that exceed this limit.
func shortSockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "sock")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, fmt.Sprintf("%d.sock", os.Getpid()))
}

func TestAgentListenerAcceptsConnection(t *testing.T) {
	sockPath := shortSockPath(t)

	var collector msgCollector
	collector.received = make(chan struct{}, 1)

	listener, err := NewAgentListener(sockPath, collector.add, false)
	if err != nil {
		t.Fatalf("NewAgentListener: %v", err)
	}
	defer listener.Close()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := AgentRequest{ID: 1, Action: "state"}
	reqBytes, _ := json.Marshal(req)
	reqBytes = append(reqBytes, '\n')
	if _, err := conn.Write(reqBytes); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Wait for message to arrive
	collector.waitForMsg(t, 2*time.Second)

	msgs := collector.snapshot()
	if len(msgs) == 0 {
		t.Fatal("expected at least one message")
	}
	msg := msgs[0]

	stateMsg, ok := msg.(AgentStateMsg)
	if !ok {
		t.Fatalf("expected AgentStateMsg, got %T", msg)
	}

	// Send response so handleConnection unblocks
	stateMsg.ResponseCh <- AgentResponse{ID: 1, OK: true, Result: "test-state"}

	// Read response from socket
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	var resp AgentResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID != 1 || !resp.OK || resp.Result != "test-state" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestAgentListenerCleansUpSocket(t *testing.T) {
	sockPath := shortSockPath(t)
	listener, err := NewAgentListener(sockPath, func(tea.Msg) {}, false)
	if err != nil {
		t.Fatalf("NewAgentListener: %v", err)
	}
	// Socket should exist while listener is open
	if _, err := os.Stat(sockPath); err != nil {
		t.Errorf("socket should exist while listener is open: %v", err)
	}
	listener.Close()
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Error("socket file should be removed after Close")
	}
}

func TestAgentListenerInvalidJSON(t *testing.T) {
	sockPath := shortSockPath(t)
	listener, err := NewAgentListener(sockPath, func(tea.Msg) {}, false)
	if err != nil {
		t.Fatalf("NewAgentListener: %v", err)
	}
	defer listener.Close()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send invalid JSON
	conn.Write([]byte("not json\n"))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var resp AgentResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.OK {
		t.Error("expected ok=false for invalid json")
	}
	if resp.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestAgentListenerValidationError(t *testing.T) {
	sockPath := shortSockPath(t)
	listener, err := NewAgentListener(sockPath, func(tea.Msg) {}, false)
	if err != nil {
		t.Fatalf("NewAgentListener: %v", err)
	}
	defer listener.Close()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send valid JSON but missing required fields (no ID)
	req := AgentRequest{Action: "exec"}
	reqBytes, _ := json.Marshal(req)
	reqBytes = append(reqBytes, '\n')
	conn.Write(reqBytes)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var resp AgentResponse
	json.Unmarshal(buf[:n], &resp)
	if resp.OK {
		t.Error("expected ok=false for validation error")
	}
}
