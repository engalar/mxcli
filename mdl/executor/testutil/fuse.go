//go:build linux || darwin

// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// MPRMount presents []byte as a FUSE virtual file at mount.Path().
// Only MPR v1 (single SQLite file, Mendix < 10.18) is supported.
type MPRMount struct {
	filePath string // full path to the virtual file
	mu       sync.RWMutex
	content  []byte
	server   *fuse.Server
}

// MountMPR mounts mprBytes as a virtual file accessible at mount.Path().
// The FUSE server is unmounted automatically via t.Cleanup.
func MountMPR(t *testing.T, mprBytes []byte) *MPRMount {
	t.Helper()
	m := &MPRMount{
		content: append([]byte(nil), mprBytes...),
	}
	dir := t.TempDir()
	m.filePath = filepath.Join(dir, "app.mpr")

	root := &memRoot{m: m}
	opts := &fs.Options{}
	server, err := fs.Mount(dir, root, opts)
	if err != nil {
		t.Fatalf("MountMPR: fuse mount: %v", err)
	}
	m.server = server
	t.Cleanup(func() { _ = server.Unmount() })
	return m
}

// Path returns the absolute path to the virtual MPR file.
func (m *MPRMount) Path() string { return m.filePath }

// Bytes returns the current in-memory content (including any writes).
func (m *MPRMount) Bytes() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]byte(nil), m.content...)
}

// NewWithMPRBytes creates a TestExec backed by a FUSE-mounted in-memory MPR.
// Only MPR v1 (single SQLite file, Mendix < 10.18) is supported; v2 causes t.Skip.
// Use te.MPRBytes() after Run() to retrieve the modified bytes for AssertGoldenMPR.
func NewWithMPRBytes(t *testing.T, mprBytes []byte) *TestExec {
	t.Helper()
	if isMPRv2(mprBytes) {
		t.Skip("NewWithMPRBytes: MPR v2 (mprcontents/) not yet supported")
	}
	mount := MountMPR(t, mprBytes)
	te := NewWithProject(t, mount.Path())
	te.mount = mount
	return te
}

// MPRBytes returns the current in-memory MPR bytes from the FUSE mount.
// Panics if this TestExec was not created via NewWithMPRBytes.
func (te *TestExec) MPRBytes() []byte {
	if te.mount == nil {
		panic("MPRBytes: TestExec was not created via NewWithMPRBytes")
	}
	return te.mount.Bytes()
}

// AssertGoldenMPR validates that got bytes match the golden snapshot at
// goldenPath. Set MXCLI_UPDATE_GOLDEN=1 to update instead of fail.
func AssertGoldenMPR(t *testing.T, goldenPath string, got []byte) {
	t.Helper()
	if os.Getenv("MXCLI_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, got, 0600); err != nil {
			t.Fatalf("AssertGoldenMPR: write golden: %v", err)
		}
		t.Logf("golden updated: %s", goldenPath)
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("AssertGoldenMPR: read golden %s: %v\nRun with MXCLI_UPDATE_GOLDEN=1 to create it", goldenPath, err)
	}
	if string(got) != string(want) {
		t.Errorf("MPR bytes differ from golden %s (re-run with MXCLI_UPDATE_GOLDEN=1 to update)", goldenPath)
	}
}

// isMPRv2 returns true if b is not a SQLite file (MPR v2's .mpr is a metadata
// file, not SQLite).
func isMPRv2(b []byte) bool {
	return len(b) < 16 || string(b[:6]) != "SQLite"
}

// --- FUSE node implementation ---

// memRoot is the root directory node that exposes a single in-memory file.
type memRoot struct {
	fs.Inode
	m *MPRMount
}

var _ fs.NodeOnAdder = (*memRoot)(nil)

func (r *memRoot) OnAdd(ctx context.Context) {
	child := r.NewPersistentInode(ctx, &memFile{m: r.m}, fs.StableAttr{Mode: syscall.S_IFREG})
	r.AddChild("app.mpr", child, true)
}

// memFile is the in-memory file node.
type memFile struct {
	fs.Inode
	m *MPRMount
}

var _ fs.NodeGetattrer = (*memFile)(nil)
var _ fs.NodeOpener = (*memFile)(nil)
var _ fs.NodeReader = (*memFile)(nil)
var _ fs.NodeWriter = (*memFile)(nil)

func (f *memFile) Getattr(_ context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	f.m.mu.RLock()
	defer f.m.mu.RUnlock()
	out.Mode = 0644
	out.Size = uint64(len(f.m.content))
	return fs.OK
}

func (f *memFile) Open(_ context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, fuse.FOPEN_KEEP_CACHE, fs.OK
}

func (f *memFile) Read(_ context.Context, _ fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	f.m.mu.RLock()
	defer f.m.mu.RUnlock()
	if off >= int64(len(f.m.content)) {
		return fuse.ReadResultData(nil), fs.OK
	}
	end := off + int64(len(dest))
	if end > int64(len(f.m.content)) {
		end = int64(len(f.m.content))
	}
	return fuse.ReadResultData(f.m.content[off:end]), fs.OK
}

func (f *memFile) Write(_ context.Context, _ fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	f.m.mu.Lock()
	defer f.m.mu.Unlock()
	end := int(off) + len(data)
	if end > len(f.m.content) {
		ext := make([]byte, end-len(f.m.content))
		f.m.content = append(f.m.content, ext...)
	}
	copy(f.m.content[off:], data)
	return uint32(len(data)), fs.OK
}
