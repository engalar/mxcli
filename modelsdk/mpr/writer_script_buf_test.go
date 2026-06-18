package mpr

import (
	"errors"
	"testing"
)

// TestWriterScriptBuf verifies that Writer.SetScriptBuf diverts insertUnit
// and updateUnit through the provided callbacks instead of file I/O + SQL.
// This is the regression guard against losing the ~9× script-buffer batching
// that EXECUTE SCRIPT depends on.
func TestWriterScriptBuf(t *testing.T) {
	w := &Writer{}
	var insertedID, insertedContainer string
	var insertedContents []byte
	var updatedID string
	var updatedContents []byte

	w.SetScriptBuf(
		func(unitID, containerID, _, _ string, contents []byte) error {
			insertedID = unitID
			insertedContainer = containerID
			insertedContents = contents
			return nil
		},
		func(unitID string, contents []byte) error {
			updatedID = unitID
			updatedContents = contents
			return nil
		},
	)
	t.Cleanup(w.ClearScriptBuf)

	// InsertUnit should route through the insert callback (the callback
	// runs BEFORE any database access, so no real MPR is needed).
	insertID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	containerID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	if err := w.InsertUnit(insertID, containerID, "Documents", "Forms$Page", []byte("page content")); err != nil {
		t.Fatalf("InsertUnit: %v", err)
	}
	if insertedID != insertID {
		t.Errorf("insert callback unitID = %q, want %q", insertedID, insertID)
	}
	if insertedContainer != containerID {
		t.Errorf("insert callback containerID = %q, want %q", insertedContainer, containerID)
	}
	if string(insertedContents) != "page content" {
		t.Errorf("insert callback contents = %q, want %q", string(insertedContents), "page content")
	}

	// UpdateRawUnit should route through the update callback.
	updateID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	if err := w.UpdateRawUnit(updateID, []byte("updated content")); err != nil {
		t.Fatalf("UpdateRawUnit: %v", err)
	}
	if updatedID != updateID {
		t.Errorf("update callback unitID = %q, want %q", updatedID, updateID)
	}
	if string(updatedContents) != "updated content" {
		t.Errorf("update callback contents = %q, want %q", string(updatedContents), "updated content")
	}
}

// TestWriterScriptBuf_ClearRestoresDirectMode verifies that ClearScriptBuf
// removes the interceptors. After clearing, the callback is NOT called.
// The subsequent write attempt would fail at database level (nil reader),
// proving the callback was bypassed.
func TestWriterScriptBuf_ClearRestoresDirectMode(t *testing.T) {
	w := &Writer{}
	cbCalled := false
	w.SetScriptBuf(
		func(_, _, _, _ string, _ []byte) error {
			cbCalled = true
			return nil
		},
		func(_ string, _ []byte) error {
			cbCalled = true
			return nil
		},
	)
	w.ClearScriptBuf()

	// Without SetScriptBuf, InsertUnit goes to the db path which panics
	// with nil reader — proving the callback was NOT called.
	func() {
		defer func() { recover() }()
		_ = w.InsertUnit("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Documents", "Forms$Page", []byte("content"))
	}()
	if cbCalled {
		t.Fatal("scriptBuf callback was called after ClearScriptBuf")
	}
}

// TestWriterScriptBuf_ErrorPropagation verifies that errors from the script
// buffer callbacks are propagated correctly.
func TestWriterScriptBuf_ErrorPropagation(t *testing.T) {
	w := &Writer{}
	wantErr := errors.New("script buffer insert error")
	w.SetScriptBuf(
		func(_, _, _, _ string, _ []byte) error { return wantErr },
		func(_ string, _ []byte) error { return errors.New("script buffer update error") },
	)
	t.Cleanup(w.ClearScriptBuf)

	err := w.InsertUnit("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "Documents", "Forms$Page", []byte("content"))
	if !errors.Is(err, wantErr) {
		t.Errorf("InsertUnit error = %v, want %v", err, wantErr)
	}
}


