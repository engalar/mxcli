package mxgraph

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// ── DeltaLog ──────────────────────────────────────────────

// DeltaLog is a file-backed append-only event log that implements EventSink.
// Events are written in the DeltaWriter binary format and can be replayed
// later to recover or update a Graph.
type DeltaLog struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	writer *DeltaWriter
}

// OpenDeltaLog opens or creates a delta log file at path. If the file
// already exists, new events are appended to it.
func OpenDeltaLog(path string) (*DeltaLog, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("open delta log: %w", err)
	}
	dl := &DeltaLog{
		path:   path,
		file:   f,
		writer: NewDeltaWriter(f),
	}
	return dl, nil
}

// Emit appends events to the delta log (implements EventSink).
func (dl *DeltaLog) Emit(events []Event) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	for _, ev := range events {
		if err := dl.writer.WriteEvent(ev); err != nil {
			return fmt.Errorf("delta log write: %w", err)
		}
	}
	return nil
}

// Replay reads all events from the beginning of the delta log and
// forwards them to sink. It does not reset the log afterward.
func (dl *DeltaLog) Replay(sink EventSink) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	if _, err := dl.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("delta log seek for replay: %w", err)
	}

	reader := NewDeltaReader(dl.file)
	for {
		ev, err := reader.ReadEvent()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("delta log read: %w", err)
		}
		if err := sink.Emit([]Event{ev}); err != nil {
			return err
		}
	}
	return nil
}

// Reset truncates the delta log to zero bytes, discarding all recorded events.
func (dl *DeltaLog) Reset() error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	if err := dl.file.Truncate(0); err != nil {
		return fmt.Errorf("delta log truncate: %w", err)
	}
	if _, err := dl.file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("delta log seek after reset: %w", err)
	}
	dl.writer = NewDeltaWriter(dl.file)
	return nil
}

// Size returns the current byte size of the delta log file.
func (dl *DeltaLog) Size() int64 {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	info, err := dl.file.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

// Close closes the underlying delta log file.
func (dl *DeltaLog) Close() error {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	return dl.file.Close()
}

// ── LoggingSink ───────────────────────────────────────────

// LoggingSink is an EventSink decorator that writes events to a secondary
// sink (e.g. DeltaLog) before forwarding them to the primary sink
// (e.g. IndexManager). This separates the concern of persistence from
// in-memory graph building without modifying either component.
type LoggingSink struct {
	primary   EventSink
	secondary EventSink
}

func NewLoggingSink(primary, secondary EventSink) *LoggingSink {
	return &LoggingSink{primary: primary, secondary: secondary}
}

func (s *LoggingSink) Emit(events []Event) error {
	if err := s.secondary.Emit(events); err != nil {
		return err
	}
	return s.primary.Emit(events)
}

// ── Snapshot + Delta 恢复 ─────────────────────────────────

// RestoreFromSnapshot loads a gob snapshot, replays the delta log on top,
// and returns the recovered Graph. If the snapshot file does not exist it
// returns (nil, nil) so the caller can fall through to a full build.
func RestoreFromSnapshot(snapshotPath, deltaPath string) (*Graph, error) {
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read snapshot: %w", err)
	}

	g, err := UnmarshalSnapshot(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}

	dl, err := OpenDeltaLog(deltaPath)
	if err != nil {
		return nil, fmt.Errorf("open delta log: %w", err)
	}
	defer dl.Close()

	if dl.Size() == 0 {
		return g, nil
	}

	mgr := NewIndexManagerFromGraph(g)
	if err := dl.Replay(mgr); err != nil {
		return nil, fmt.Errorf("replay delta log: %w", err)
	}

	return mgr.Query(), nil
}

// Compact merges all delta log events into a new snapshot and resets the
// delta log. It writes the new snapshot atomically via a temp file + rename
// so that a crash during compaction does not corrupt the snapshot.
func Compact(snapshotPath, deltaPath string) error {
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return fmt.Errorf("compact read snapshot: %w", err)
	}

	g, err := UnmarshalSnapshot(data)
	if err != nil {
		return fmt.Errorf("compact unmarshal snapshot: %w", err)
	}

	dl, err := OpenDeltaLog(deltaPath)
	if err != nil {
		return fmt.Errorf("compact open delta log: %w", err)
	}
	defer dl.Close()

	if dl.Size() == 0 {
		return nil
	}

	mgr := NewIndexManagerFromGraph(g)
	if err := dl.Replay(mgr); err != nil {
		return fmt.Errorf("compact replay delta: %w", err)
	}

	newData, err := MarshalSnapshot(mgr.Query())
	if err != nil {
		return fmt.Errorf("compact marshal: %w", err)
	}

	tmpPath := snapshotPath + ".tmp"
	if err := os.WriteFile(tmpPath, newData, 0600); err != nil {
		return fmt.Errorf("compact write tmp: %w", err)
	}
	if err := os.Rename(tmpPath, snapshotPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("compact rename: %w", err)
	}

	if err := dl.Reset(); err != nil {
		return fmt.Errorf("compact reset delta: %w", err)
	}

	return nil
}
