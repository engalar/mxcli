// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

import (
	"context"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// overlayNode represents a file or directory in the overlay.
// relPath is the slash-separated path relative to baseDir (empty string = root).
type overlayNode struct {
	fs.Inode
	baseDir string
	relPath string
	layer   *dirtyLayer
}

// pathIno generates a stable inode number from a relative path.
func pathIno(relPath string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(relPath))
	return h.Sum64()
}

// absBase returns the absolute path of this node in baseDir.
func (n *overlayNode) absBase() string {
	if n.relPath == "" {
		return n.baseDir
	}
	return filepath.Join(n.baseDir, filepath.FromSlash(n.relPath))
}

// childRel returns the relative path of a named child of this node.
func (n *overlayNode) childRel(name string) string {
	if n.relPath == "" {
		return name
	}
	return n.relPath + "/" + name
}

// --- NodeLookuper ---

func (n *overlayNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	rel := n.childRel(name)

	if n.layer.isDeleted(rel) {
		return nil, syscall.ENOENT
	}

	// Determine mode: dirty layer takes precedence over base.
	// mode is in syscall form (S_IFREG/S_IFDIR | perm bits), NOT os.FileMode form.
	var mode uint32
	if content := n.layer.read(rel); content != nil {
		mode = syscall.S_IFREG | 0644
		out.Size = uint64(len(content))
	} else {
		info, err := os.Lstat(filepath.Join(n.baseDir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, syscall.ENOENT
		}
		mode = sysMode(info)
		out.Size = uint64(info.Size())
	}

	out.Mode = mode
	child := &overlayNode{baseDir: n.baseDir, relPath: rel, layer: n.layer}
	// Inode StableAttr only needs the type bits (S_IFDIR/S_IFREG/...) not permissions.
	stable := fs.StableAttr{Mode: mode & syscall.S_IFMT, Ino: pathIno(rel)}
	return n.NewInode(ctx, child, stable), 0
}

// sysMode returns the syscall (Linux) mode from a Go FileInfo, including
// S_IFDIR/S_IFREG/... type bits and permission bits. We can't just cast
// os.FileMode to uint32 because Go uses its own encoding (os.ModeDir = 1<<31).
func sysMode(info os.FileInfo) uint32 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Mode
	}
	// Fallback: synthesize from os.FileMode.
	perm := uint32(info.Mode().Perm())
	if info.IsDir() {
		return syscall.S_IFDIR | perm
	}
	return syscall.S_IFREG | perm
}

// --- NodeGetattrer ---

func (n *overlayNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if n.relPath == "" {
		// Root directory
		info, err := os.Stat(n.baseDir)
		if err != nil {
			return syscall.EIO
		}
		out.Mode = sysMode(info)
		out.Ino = 1
		return 0
	}

	if n.layer.isDeleted(n.relPath) {
		return syscall.ENOENT
	}

	if content := n.layer.read(n.relPath); content != nil {
		out.Mode = syscall.S_IFREG | 0644
		out.Size = uint64(len(content))
		out.Ino = pathIno(n.relPath)
		return 0
	}

	info, err := os.Lstat(n.absBase())
	if err != nil {
		return syscall.ENOENT
	}
	out.Mode = sysMode(info)
	out.Size = uint64(info.Size())
	out.Ino = pathIno(n.relPath)
	return 0
}

// --- NodeReaddirer ---

func (n *overlayNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	seen := map[string]bool{}
	var entries []fuse.DirEntry

	// Base dir entries
	baseEntries, err := os.ReadDir(n.absBase())
	if err != nil && !os.IsNotExist(err) {
		return nil, syscall.EIO
	}
	prefix := n.relPath
	if prefix != "" {
		prefix += "/"
	}
	for _, e := range baseEntries {
		rel := prefix + e.Name()
		if n.layer.isDeleted(rel) {
			continue
		}
		seen[e.Name()] = true
		// fuse.DirEntry.Mode wants syscall type bits (S_IFDIR/S_IFREG/...),
		// not os.FileMode (which encodes ModeDir as 1<<31).
		mode := uint32(fuse.S_IFREG)
		if e.IsDir() {
			mode = fuse.S_IFDIR
		} else if e.Type()&os.ModeSymlink != 0 {
			mode = fuse.S_IFLNK
		}
		entries = append(entries, fuse.DirEntry{Name: e.Name(), Mode: mode, Ino: pathIno(rel)})
	}

	// Dirty-layer additions in this directory
	n.layer.mu.RLock()
	defer n.layer.mu.RUnlock()
	for rel, content := range n.layer.files {
		if content == nil {
			continue // tombstone
		}
		if !isDirectChild(rel, n.relPath) {
			continue
		}
		name := filepath.Base(rel)
		if seen[name] {
			continue
		}
		entries = append(entries, fuse.DirEntry{Name: name, Mode: fuse.S_IFREG, Ino: pathIno(rel)})
	}

	return fs.NewListDirStream(entries), 0
}

// isDirectChild returns true if rel is a direct child of parentRel.
func isDirectChild(rel, parentRel string) bool {
	if parentRel == "" {
		return !strings.Contains(rel, "/")
	}
	if !strings.HasPrefix(rel, parentRel+"/") {
		return false
	}
	rest := rel[len(parentRel)+1:]
	return !strings.Contains(rest, "/")
}

// --- NodeOpener + file handle (read) ---

type overlayReadHandle struct {
	content []byte
}

func (h *overlayReadHandle) Read(_ context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if off >= int64(len(h.content)) {
		return fuse.ReadResultData(nil), 0
	}
	end := off + int64(len(dest))
	if end > int64(len(h.content)) {
		end = int64(len(h.content))
	}
	return fuse.ReadResultData(h.content[off:end]), 0
}

func (n *overlayNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// Write-capable open handled by Create / separate write handle.
	content := n.layer.read(n.relPath)
	if content == nil {
		var err error
		content, err = os.ReadFile(n.absBase())
		if err != nil {
			return nil, 0, syscall.EIO
		}
	}
	return &overlayReadHandle{content: content}, fuse.FOPEN_DIRECT_IO, 0
}
