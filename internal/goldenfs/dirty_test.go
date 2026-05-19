// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirtyLayer_ReadMiss_ReturnsNil(t *testing.T) {
	l := newDirtyLayer()
	if got := l.read("foo.txt"); got != nil {
		t.Fatalf("expected nil for miss, got %d bytes", len(got))
	}
}

func TestDirtyLayer_WriteAndRead(t *testing.T) {
	l := newDirtyLayer()
	l.write("foo.txt", 0, []byte("hello"))
	got := l.read("foo.txt")
	if string(got) != "hello" {
		t.Fatalf("want %q got %q", "hello", got)
	}
}

func TestDirtyLayer_PartialWrite_CopyUp(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "f.bin"), []byte("AAAAAA"), 0644); err != nil {
		t.Fatal(err)
	}
	l := newDirtyLayer()
	// Write "BB" at offset 2; triggers copy-up from base.
	l.writeWithBase("f.bin", base, 2, []byte("BB"))
	got := l.read("f.bin")
	if string(got) != "AABBAA" {
		t.Fatalf("want AABBAA got %q", got)
	}
}

func TestDirtyLayer_Tombstone(t *testing.T) {
	l := newDirtyLayer()
	l.write("f.txt", 0, []byte("x"))
	l.delete("f.txt")
	if !l.isDeleted("f.txt") {
		t.Fatal("expected tombstone")
	}
}

func TestDirtyLayer_Rollback_ClearsAll(t *testing.T) {
	l := newDirtyLayer()
	l.write("a.txt", 0, []byte("data"))
	l.delete("b.txt")
	l.rollback()
	if l.read("a.txt") != nil {
		t.Fatal("dirty file should be gone after rollback")
	}
	if l.isDeleted("b.txt") {
		t.Fatal("tombstone should be gone after rollback")
	}
}

func TestDirtyLayer_Commit_WritesBase(t *testing.T) {
	base := t.TempDir()
	l := newDirtyLayer()
	l.write("out.txt", 0, []byte("committed"))
	if err := l.commit(base); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(base, "out.txt"))
	if err != nil || string(got) != "committed" {
		t.Fatalf("commit did not write file: %v", err)
	}
}

func TestDirtyLayer_Commit_SkipsWAL(t *testing.T) {
	base := t.TempDir()
	l := newDirtyLayer()
	l.write("minimal.mpr-wal", 0, []byte("wal data"))
	l.write("minimal.mpr-shm", 0, []byte("shm data"))
	if err := l.commit(base); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "minimal.mpr-wal")); !os.IsNotExist(err) {
		t.Fatal("WAL file must not be committed to base")
	}
}
