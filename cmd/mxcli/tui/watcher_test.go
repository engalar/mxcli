package tui

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// mockSender captures MprChangedMsg sends for testing.
type mockSender struct {
	count atomic.Int32
}

func (m *mockSender) Send(msg tea.Msg) {
	if _, ok := msg.(MprChangedMsg); ok {
		m.count.Add(1)
	}
}

func TestWatcherDebounce(t *testing.T) {
	dir := t.TempDir()
	unitFile := filepath.Join(dir, "test.mxunit")
	if err := os.WriteFile(unitFile, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	sender := &mockSender{}
	w, err := newWatcher("", dir, sender)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Rapidly write 5 times — should debounce into a single message
	for i := range 5 {
		_ = os.WriteFile(unitFile, []byte{byte('a' + i)}, 0644)
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for debounce to fire (500ms + margin)
	time.Sleep(700 * time.Millisecond)

	got := sender.count.Load()
	if got != 1 {
		t.Errorf("expected 1 debounced message, got %d", got)
	}
}

func TestWatcherSuppress(t *testing.T) {
	dir := t.TempDir()
	unitFile := filepath.Join(dir, "test.mxunit")
	if err := os.WriteFile(unitFile, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	sender := &mockSender{}
	w, err := newWatcher("", dir, sender)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Suppress for 2 seconds
	w.Suppress(2 * time.Second)

	// Write during suppress window
	_ = os.WriteFile(unitFile, []byte("b"), 0644)
	time.Sleep(700 * time.Millisecond)

	got := sender.count.Load()
	if got != 0 {
		t.Errorf("expected 0 messages during suppress, got %d", got)
	}
}

func TestWatcherCloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	unitFile := filepath.Join(dir, "test.mxunit")
	_ = os.WriteFile(unitFile, []byte("a"), 0644)

	sender := &mockSender{}
	w, err := newWatcher("", dir, sender)
	if err != nil {
		t.Fatal(err)
	}

	// Double close should not panic
	w.Close()
	w.Close()
}

func TestWatcherIgnoresNonMxunitFiles(t *testing.T) {
	dir := t.TempDir()
	unitFile := filepath.Join(dir, "test.mxunit")
	_ = os.WriteFile(unitFile, []byte("a"), 0644)

	sender := &mockSender{}
	w, err := newWatcher("", dir, sender)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Write a .tmp file — should be ignored
	tmpFile := filepath.Join(dir, "test.tmp")
	_ = os.WriteFile(tmpFile, []byte("b"), 0644)
	time.Sleep(700 * time.Millisecond)

	got := sender.count.Load()
	if got != 0 {
		t.Errorf("expected 0 messages for .tmp file, got %d", got)
	}
}
