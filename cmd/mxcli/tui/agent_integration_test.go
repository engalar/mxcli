package tui

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// msgCollector is a thread-safe collector for tea.Msg values dispatched by the
// agent listener goroutine. It replaces the bare []tea.Msg slice that caused
// data races when the sender callback and the test goroutine accessed it
// concurrently.
type msgCollector struct {
	mu       sync.Mutex
	msgs     []tea.Msg
	received chan struct{}
}

func (c *msgCollector) add(msg tea.Msg) {
	c.mu.Lock()
	c.msgs = append(c.msgs, msg)
	c.mu.Unlock()
	select {
	case c.received <- struct{}{}:
	default:
	}
}

func (c *msgCollector) snapshot() []tea.Msg {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]tea.Msg, len(c.msgs))
	copy(out, c.msgs)
	return out
}

func (c *msgCollector) reset() {
	c.mu.Lock()
	c.msgs = nil
	c.mu.Unlock()
}

func (c *msgCollector) waitForMsg(t *testing.T, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		c.mu.Lock()
		hasMsgs := len(c.msgs) > 0
		c.mu.Unlock()
		if hasMsgs {
			return
		}
		select {
		case <-c.received:
		case <-deadline:
			c.mu.Lock()
			count := len(c.msgs)
			c.mu.Unlock()
			t.Fatalf("timed out waiting for message after %v (have %d msgs)", timeout, count)
		}
	}
}

func TestAgentExecEndToEnd(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")

	var collector msgCollector
	collector.received = make(chan struct{}, 1)
	listener, err := NewAgentListener(sockPath, collector.add, true)
	if err != nil {
		t.Fatalf("NewAgentListener: %v", err)
	}
	defer listener.Close()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Send exec request
	req := AgentRequest{ID: 42, Action: "exec", MDL: "SHOW ENTITIES"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Wait for message dispatch
	collector.waitForMsg(t, 2*time.Second)
	messages := collector.snapshot()
	if len(messages) == 0 {
		t.Fatal("no messages received")
	}

	execMsg, ok := messages[0].(AgentExecMsg)
	if !ok {
		t.Fatalf("expected AgentExecMsg, got %T", messages[0])
	}
	if execMsg.MDL != "SHOW ENTITIES" {
		t.Errorf("mdl = %q, want SHOW ENTITIES", execMsg.MDL)
	}
	if execMsg.RequestID != 42 {
		t.Errorf("requestID = %d, want 42", execMsg.RequestID)
	}

	// Simulate TUI response
	execMsg.ResponseCh <- AgentResponse{ID: 42, OK: true, Result: "done"}

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
	if resp.ID != 42 || !resp.OK {
		t.Errorf("resp = %+v", resp)
	}
}

func TestAgentInvalidRequests(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")
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

	// Invalid JSON
	conn.Write([]byte("not json\n"))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var resp AgentResponse
	json.Unmarshal(buf[:n], &resp)
	if resp.OK {
		t.Error("expected ok=false for invalid json")
	}

	// Missing required fields
	req := AgentRequest{Action: "exec"} // missing ID and MDL
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	conn.Write(data)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	json.Unmarshal(buf[:n], &resp)
	if resp.OK {
		t.Error("expected ok=false for missing fields")
	}
}

func TestAgentSocketCleanup(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")
	listener, err := NewAgentListener(sockPath, func(tea.Msg) {}, false)
	if err != nil {
		t.Fatalf("NewAgentListener: %v", err)
	}
	if _, err := os.Stat(sockPath); err != nil {
		t.Errorf("socket should exist: %v", err)
	}
	listener.Close()
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Error("socket should be removed after close")
	}
}

func TestAgentMultipleActions(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")

	var collector msgCollector
	collector.received = make(chan struct{}, 1)
	listener, err := NewAgentListener(sockPath, collector.add, true)
	if err != nil {
		t.Fatalf("NewAgentListener: %v", err)
	}
	defer listener.Close()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Test state action
	req := AgentRequest{ID: 1, Action: "state"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	conn.Write(data)
	collector.waitForMsg(t, 2*time.Second)
	messages := collector.snapshot()
	if len(messages) == 0 {
		t.Fatal("no state message received")
	}
	stateMsg, ok := messages[0].(AgentStateMsg)
	if !ok {
		t.Fatalf("expected AgentStateMsg, got %T", messages[0])
	}
	stateMsg.ResponseCh <- AgentResponse{ID: 1, OK: true, Result: `{"mode":"Browse"}`}

	// Read state response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 4096)
	n, _ := conn.Read(buf)
	var resp AgentResponse
	json.Unmarshal(buf[:n], &resp)
	if !resp.OK || resp.ID != 1 {
		t.Errorf("state resp = %+v", resp)
	}

	// Test navigate action
	collector.reset()
	collector.received = make(chan struct{}, 1)
	req = AgentRequest{ID: 2, Action: "navigate", Target: "entity:MyModule.Customer"}
	data, _ = json.Marshal(req)
	data = append(data, '\n')
	conn.Write(data)
	collector.waitForMsg(t, 2*time.Second)
	messages = collector.snapshot()
	if len(messages) == 0 {
		t.Fatal("no navigate message received")
	}
	navMsg, ok := messages[0].(AgentNavigateMsg)
	if !ok {
		t.Fatalf("expected AgentNavigateMsg, got %T", messages[0])
	}
	if navMsg.Target != "entity:MyModule.Customer" {
		t.Errorf("target = %q", navMsg.Target)
	}
	navMsg.ResponseCh <- AgentResponse{ID: 2, OK: true}

	// Read navigate response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ = conn.Read(buf)
	json.Unmarshal(buf[:n], &resp)
	if !resp.OK || resp.ID != 2 {
		t.Errorf("navigate resp = %+v", resp)
	}
}

func TestAgentDeleteMsg(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")

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

	req := AgentRequest{ID: 10, Action: "delete", Target: "entity:MyModule.Customer"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	collector.waitForMsg(t, 2*time.Second)
	messages := collector.snapshot()
	if len(messages) == 0 {
		t.Fatal("no messages received")
	}

	delMsg, ok := messages[0].(AgentDeleteMsg)
	if !ok {
		t.Fatalf("expected AgentDeleteMsg, got %T", messages[0])
	}
	if delMsg.RequestID != 10 {
		t.Errorf("requestID = %d, want 10", delMsg.RequestID)
	}
	if delMsg.Target != "entity:MyModule.Customer" {
		t.Errorf("target = %q, want entity:MyModule.Customer", delMsg.Target)
	}

	delMsg.ResponseCh <- AgentResponse{ID: 10, OK: true, Result: "deleted"}

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
	if resp.ID != 10 || !resp.OK {
		t.Errorf("resp = %+v", resp)
	}
}

func TestAgentCreateModuleMsg(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")

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

	req := AgentRequest{ID: 11, Action: "create_module", Name: "NewModule"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	collector.waitForMsg(t, 2*time.Second)
	messages := collector.snapshot()
	if len(messages) == 0 {
		t.Fatal("no messages received")
	}

	createMsg, ok := messages[0].(AgentCreateModuleMsg)
	if !ok {
		t.Fatalf("expected AgentCreateModuleMsg, got %T", messages[0])
	}
	if createMsg.RequestID != 11 {
		t.Errorf("requestID = %d, want 11", createMsg.RequestID)
	}
	if createMsg.Name != "NewModule" {
		t.Errorf("name = %q, want NewModule", createMsg.Name)
	}

	createMsg.ResponseCh <- AgentResponse{ID: 11, OK: true, Result: "module created"}

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
	if resp.ID != 11 || !resp.OK {
		t.Errorf("resp = %+v", resp)
	}
}

func TestAgentFormatSync(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")

	var sendCount atomic.Int32
	sender := func(msg tea.Msg) { sendCount.Add(1) }

	listener, err := NewAgentListener(sockPath, sender, false)
	if err != nil {
		t.Fatalf("NewAgentListener: %v", err)
	}
	defer listener.Close()

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := AgentRequest{ID: 12, Action: "format", MDL: "CREATE ENTITY    Mod.E"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

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
	if resp.ID != 12 {
		t.Errorf("resp.ID = %d, want 12", resp.ID)
	}
	if !resp.OK {
		t.Errorf("resp.OK = false, want true; error = %q", resp.Error)
	}
	if resp.Result == "" {
		t.Error("expected non-empty formatted result")
	}
	if sendCount.Load() != 0 {
		t.Errorf("sendMsg called %d times, want 0 (sync action)", sendCount.Load())
	}
}

func TestAgentDescribeAsync(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")

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

	req := AgentRequest{ID: 13, Action: "describe", Target: "entity:Mod.E"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	collector.waitForMsg(t, 2*time.Second)
	messages := collector.snapshot()
	if len(messages) == 0 {
		t.Fatal("no messages received")
	}

	execMsg, ok := messages[0].(AgentExecMsg)
	if !ok {
		t.Fatalf("expected AgentExecMsg, got %T", messages[0])
	}
	if execMsg.MDL != "DESCRIBE ENTITY Mod.E" {
		t.Errorf("mdl = %q, want DESCRIBE ENTITY Mod.E", execMsg.MDL)
	}
	if execMsg.RequestID != 13 {
		t.Errorf("requestID = %d, want 13", execMsg.RequestID)
	}

	// Simulate TUI response
	execMsg.ResponseCh <- AgentResponse{ID: 13, OK: true, Result: "entity details"}

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
	if resp.ID != 13 || !resp.OK {
		t.Errorf("resp = %+v", resp)
	}
}

func TestAgentListAsync(t *testing.T) {
	sockPath := filepath.Join(t.TempDir(), "agent.sock")

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

	req := AgentRequest{ID: 14, Action: "list", Target: "entities"}
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write: %v", err)
	}

	collector.waitForMsg(t, 2*time.Second)
	messages := collector.snapshot()
	if len(messages) == 0 {
		t.Fatal("no messages received")
	}

	execMsg, ok := messages[0].(AgentExecMsg)
	if !ok {
		t.Fatalf("expected AgentExecMsg, got %T", messages[0])
	}
	if execMsg.MDL != "SHOW ENTITIES" {
		t.Errorf("mdl = %q, want SHOW ENTITIES", execMsg.MDL)
	}
	if execMsg.RequestID != 14 {
		t.Errorf("requestID = %d, want 14", execMsg.RequestID)
	}

	// Simulate TUI response
	execMsg.ResponseCh <- AgentResponse{ID: 14, OK: true, Result: "entity list"}

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
	if resp.ID != 14 || !resp.OK {
		t.Errorf("resp = %+v", resp)
	}
}
