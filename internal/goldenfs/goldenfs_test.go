// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"os"
	"path/filepath"
	"testing"
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
