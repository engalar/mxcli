// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"testing"
)

func TestScriptBuffer_AddUpdate_VisibleInOverlay(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	b := New()
	if err := b.Connect(dst); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	buf := newScriptBuffer(b.reader)
	unitID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	contents := []byte{0x01, 0x02, 0x03}

	if err := buf.AddUpdate(unitID, contents); err != nil {
		t.Fatalf("AddUpdate: %v", err)
	}

	overlay := b.reader.ScriptOverlay()
	if overlay == nil {
		t.Fatal("scriptOverlay is nil after AddUpdate")
	}
	got, ok := overlay[unitID]
	if !ok || string(got) != string(contents) {
		t.Errorf("overlay mismatch: got %v ok=%v", got, ok)
	}
}

func TestScriptBuffer_AddInsert_VisibleInInsertList(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	b := New()
	if err := b.Connect(dst); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	buf := newScriptBuffer(b.reader)
	if err := buf.AddInsert(
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		"00000000-0000-0000-0000-000000000000",
		"Documents", "Microflows$Microflow", []byte{0x05},
	); err != nil {
		t.Fatalf("AddInsert: %v", err)
	}

	inserts := b.reader.ScriptInserts()
	if len(inserts) == 0 {
		t.Fatal("scriptInserts empty after AddInsert")
	}
	if inserts[0].ID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Errorf("unexpected ID: %s", inserts[0].ID)
	}
}

func TestBeginScriptTransaction_NoDBBegin(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	b := New()
	if err := b.Connect(dst); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	tx, err := b.BeginScriptTransaction()
	if err != nil {
		t.Fatalf("BeginScriptTransaction: %v", err)
	}
	if b.scriptBuf == nil {
		t.Error("scriptBuf is nil after BeginScriptTransaction")
	}
	_ = tx.Rollback()
	if b.scriptBuf != nil {
		t.Error("scriptBuf not nil after Rollback")
	}
}

func TestScriptBuffer_Rollback_ClearsOverlay(t *testing.T) {
	dst := copyFixture(t, fixturePath, t.TempDir())
	b := New()
	if err := b.Connect(dst); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer b.Disconnect()

	buf := newScriptBuffer(b.reader)
	_ = buf.AddUpdate("cccccccc-cccc-cccc-cccc-cccccccccccc", []byte{0x99})

	if b.reader.ScriptOverlay() == nil {
		t.Fatal("expected overlay before rollback")
	}
	buf.Rollback()

	if b.reader.ScriptOverlay() != nil {
		t.Error("scriptOverlay not nil after Rollback")
	}
	if b.reader.ScriptInserts() != nil {
		t.Error("scriptInserts not nil after Rollback")
	}
}
