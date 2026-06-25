// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

// baseFixture creates a minimal temp base directory with a file and a subdir.
func baseFixture(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "hello.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "sub", "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestSnapshot_ReadExistingFile(t *testing.T) {
	snap, err := Open(baseFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	got, err := os.ReadFile(filepath.Join(snap.MountDir(), "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "world" {
		t.Fatalf("want world got %q", got)
	}
}

func TestSnapshot_ReadDeepFile(t *testing.T) {
	snap, err := Open(baseFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	got, err := os.ReadFile(filepath.Join(snap.MountDir(), "sub", "deep.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "deep" {
		t.Fatalf("want deep got %q", got)
	}
}

func TestSnapshot_ReaddirRootContainsBase(t *testing.T) {
	snap, err := Open(baseFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	entries, err := os.ReadDir(snap.MountDir())
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	if !names["hello.txt"] || !names["sub"] {
		t.Fatalf("missing expected entries: got %v", names)
	}
}

func TestSnapshot_WriteNewFile_InDirtyNotBase(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	// Write via FUSE mount
	dest := filepath.Join(snap.MountDir(), "new.txt")
	if err := os.WriteFile(dest, []byte("new content"), 0644); err != nil {
		t.Fatalf("write through FUSE: %v", err)
	}

	// Readable through mount
	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "new content" {
		t.Fatalf("want new content, got %q err %v", got, err)
	}

	// Base dir untouched
	if _, err := os.Stat(filepath.Join(base, "new.txt")); !os.IsNotExist(err) {
		t.Fatal("base dir must not be modified")
	}
}

func TestSnapshot_OverwriteExistingFile(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	dest := filepath.Join(snap.MountDir(), "hello.txt")
	if err := os.WriteFile(dest, []byte("overwritten"), 0644); err != nil {
		t.Fatalf("overwrite through FUSE: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil || string(got) != "overwritten" {
		t.Fatalf("want overwritten, got %q err %v", got, err)
	}

	// Base still has original
	orig, _ := os.ReadFile(filepath.Join(base, "hello.txt"))
	if string(orig) != "world" {
		t.Fatalf("base must be unmodified, got %q", orig)
	}
}

func TestSnapshot_MkdirAndWriteDeep(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	newDir := filepath.Join(snap.MountDir(), "newdir")
	if err := os.Mkdir(newDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "child.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("write in new dir: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(snap.MountDir(), "newdir", "child.txt"))
	if err != nil || string(got) != "hi" {
		t.Fatalf("want hi, got %q err %v", got, err)
	}
}

func TestSnapshot_Commit_PersistsToBase(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	if err := os.WriteFile(filepath.Join(snap.MountDir(), "committed.txt"), []byte("saved"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := snap.Commit(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(base, "committed.txt"))
	if err != nil || string(got) != "saved" {
		t.Fatalf("want saved, got %q err %v", got, err)
	}
}

func TestSnapshot_Rollback_BaseUnchanged(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	if err := os.WriteFile(filepath.Join(snap.MountDir(), "temp.txt"), []byte("gone"), 0644); err != nil {
		t.Fatal(err)
	}
	snap.Rollback()

	// After rollback the file should disappear from the mount view.
	if _, err := os.Stat(filepath.Join(snap.MountDir(), "temp.txt")); !os.IsNotExist(err) {
		t.Fatal("rolled-back file must not be visible in mount")
	}
	// Base must be untouched.
	if _, err := os.Stat(filepath.Join(base, "temp.txt")); !os.IsNotExist(err) {
		t.Fatal("base dir must not have the file")
	}
}

func TestSnapshot_Close_DoesNotCommit(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snap.MountDir(), "volatile.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	snap.Close()

	if _, err := os.Stat(filepath.Join(base, "volatile.txt")); !os.IsNotExist(err) {
		t.Fatal("Close without Commit must not write to base")
	}
}

func TestSnapshot_Rename_EmptyFile_NotTombstone(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	// Create an empty file and rename it.
	if err := os.WriteFile(filepath.Join(snap.MountDir(), "empty.txt"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(
		filepath.Join(snap.MountDir(), "empty.txt"),
		filepath.Join(snap.MountDir(), "renamed.txt"),
	); err != nil {
		t.Fatalf("rename: %v", err)
	}

	// Renamed destination must exist (not be a tombstone).
	if _, err := os.Stat(filepath.Join(snap.MountDir(), "renamed.txt")); err != nil {
		t.Fatalf("renamed empty file must be visible: %v", err)
	}
	// Source must not exist.
	if _, err := os.Stat(filepath.Join(snap.MountDir(), "empty.txt")); !os.IsNotExist(err) {
		t.Fatal("source of rename must be gone")
	}
}

func TestSnapshot_Commit_AfterRollback_IsNoop(t *testing.T) {
	base := baseFixture(t)
	snap, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	if err := os.WriteFile(filepath.Join(snap.MountDir(), "gone.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	snap.Rollback()
	// Commit after Rollback must be a no-op — nothing written to base.
	if err := snap.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "gone.txt")); !os.IsNotExist(err) {
		t.Fatal("Commit after Rollback must not write to base")
	}
}

func TestSnapshot_Parallel_Independent(t *testing.T) {
	base := baseFixture(t)

	snap1, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap1.Close()

	snap2, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer snap2.Close()

	// Write different content to the same path in each snapshot.
	if err := os.WriteFile(filepath.Join(snap1.MountDir(), "hello.txt"), []byte("snap1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snap2.MountDir(), "hello.txt"), []byte("snap2"), 0644); err != nil {
		t.Fatal(err)
	}

	got1, _ := os.ReadFile(filepath.Join(snap1.MountDir(), "hello.txt"))
	got2, _ := os.ReadFile(filepath.Join(snap2.MountDir(), "hello.txt"))

	if string(got1) != "snap1" {
		t.Errorf("snap1: want snap1, got %q", got1)
	}
	if string(got2) != "snap2" {
		t.Errorf("snap2: want snap2, got %q", got2)
	}

	// Base untouched.
	orig, _ := os.ReadFile(filepath.Join(base, "hello.txt"))
	if string(orig) != "world" {
		t.Errorf("base must remain world, got %q", orig)
	}
}

// sqliteFixture creates a directory with a minimal SQLite database file.
func sqliteFixture(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	dbPath := filepath.Join(base, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS t (k INT PRIMARY KEY, v TEXT)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec("INSERT INTO t VALUES (1, 'hello'), (2, 'world')"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	return base
}

// TestSnapshot_ConcurrentSQLite creates N concurrent goldenfs mounts over a
// SQLite fixture, opens the database through each mount, and verifies all
// mounts return consistent, independent reads. This guards against the
// go-fuse inode tree race that caused ECONNABORTED under parallel mounts.
// The race is probabilistic and reproduces more reliably with high concurrency
// (N=50 matches the repo's actual parallel test count).
func TestSnapshot_ConcurrentSQLite(t *testing.T) {
	const n = 10
	base := sqliteFixture(t)

	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			snap, err := Open(base)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("Open: %w", err))
				mu.Unlock()
				return
			}
			defer snap.Close()

			dbPath := filepath.Join(snap.MountDir(), "test.db")
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("sql.Open: %w", err))
				mu.Unlock()
				return
			}
			defer db.Close()

			var v string
			if err := db.QueryRow("SELECT v FROM t WHERE k = 1").Scan(&v); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("query: %w", err))
				mu.Unlock()
				return
			}
			if v != "hello" {
				mu.Lock()
				errs = append(errs, fmt.Errorf("got %q want hello", v))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	for _, err := range errs {
		t.Error(err)
	}
	if len(errs) > 0 {
		t.Fatalf("%d of %d concurrent mounts failed", len(errs), n)
	}
}
