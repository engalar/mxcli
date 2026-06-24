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
// relPath is the slash-separated path relative to the overlay baseDir
// (empty string = root). The over field provides access to both the
// base directory path and the dirty layer.
type overlayNode struct {
	fs.Inode
	over    *fuseOverlay
	relPath string
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
		return n.over.baseDir
	}
	return filepath.Join(n.over.baseDir, filepath.FromSlash(n.relPath))
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

	if n.over.layer.isDeleted(rel) {
		return nil, syscall.ENOENT
	}

	// Determine mode: dirty layer takes precedence over base.
	// mode is in syscall form (S_IFREG/S_IFDIR | perm bits), NOT os.FileMode form.
	var mode uint32
	if content := n.over.layer.read(rel); content != nil {
		mode = syscall.S_IFREG | 0644
		out.Size = uint64(len(content))
	} else if n.over.layer.hasDirtyDir(rel) {
		// Directory created in the dirty layer (e.g. by Mkdir) — not on disk yet.
		mode = syscall.S_IFDIR | 0755
		out.Size = 0
	} else {
		info, err := os.Lstat(filepath.Join(n.over.baseDir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, syscall.ENOENT
		}
		mode = sysMode(info)
		out.Size = uint64(info.Size())
	}

	out.Mode = mode
	child := &overlayNode{over: n.over, relPath: rel}
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
		info, err := os.Stat(n.over.baseDir)
		if err != nil {
			return syscall.EIO
		}
		out.Mode = sysMode(info)
		out.Ino = 1
		return 0
	}

	if n.over.layer.isDeleted(n.relPath) {
		return syscall.ENOENT
	}

	if content := n.over.layer.read(n.relPath); content != nil {
		out.Mode = syscall.S_IFREG | 0644
		out.Size = uint64(len(content))
		out.Ino = pathIno(n.relPath)
		return 0
	}

	if n.over.layer.hasDirtyDir(n.relPath) {
		out.Mode = syscall.S_IFDIR | 0755
		out.Size = 0
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

// --- NodeSetattrer ---
//
// Required so that O_TRUNC on open() — which the kernel turns into a SETATTR
// with FATTR_SIZE — does not fail with EOPNOTSUPP. We only honour the size
// field; mode/owner/timestamps are accepted but ignored (the dirty layer has
// no permission model and rolls back on Close).

func (n *overlayNode) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if size, ok := in.GetSize(); ok {
		n.over.layer.truncate(n.relPath, n.over.baseDir, int64(size))
	}
	// Getattr re-populates out (Size, Mode, Ino) for the kernel's attr cache.
	return n.Getattr(ctx, fh, out)
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
		if n.over.layer.isDeleted(rel) {
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

	// Dirty-layer additions in this directory. We want both direct child files
	// (key == prefix+name) and synthesised direct child directories (key has
	// the form prefix+name+"/..."). prefix is "" at the root.
	//
	// Direct mu access is safe: overlay and dirtyLayer are in the same package,
	// and we need atomic iteration over files without a per-entry method call.
	n.over.layer.mu.RLock()
	defer n.over.layer.mu.RUnlock()
	for rel, content := range n.over.layer.files {
		if content == nil {
			continue // tombstone
		}
		if prefix != "" && !strings.HasPrefix(rel, prefix) {
			continue
		}
		rest := rel[len(prefix):]
		if rest == "" {
			continue
		}
		name := rest
		mode := uint32(fuse.S_IFREG)
		if idx := strings.IndexByte(rest, '/'); idx >= 0 {
			name = rest[:idx]
			mode = fuse.S_IFDIR
		}
		if name == ".keep" {
			continue // dirty-layer internal sentinel; don't surface to userspace
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		entries = append(entries, fuse.DirEntry{Name: name, Mode: mode, Ino: pathIno(prefix + name)})
	}

	return fs.NewListDirStream(entries), 0
}

// --- NodeOpener + file handle (read) ---

type overlayReadHandle struct {
	content []byte
}

func (h *overlayReadHandle) Read(_ context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if off < 0 {
		return fuse.ReadResultData(nil), syscall.EINVAL
	}
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
	if flags&(syscall.O_WRONLY|syscall.O_RDWR) != 0 {
		// Seed dirty map via copy-up, then return write handle.
		if !n.over.layer.has(n.relPath) {
			if b, err := os.ReadFile(n.absBase()); err == nil {
				n.over.layer.write(n.relPath, 0, b)
			}
		}
		return &overlayWriteHandle{node: n, baseDir: n.over.baseDir}, fuse.FOPEN_DIRECT_IO, 0
	}
	content := n.over.layer.read(n.relPath)
	if content == nil {
		var err error
		content, err = os.ReadFile(n.absBase())
		if err != nil {
			return nil, 0, syscall.EIO
		}
	}
	return &overlayReadHandle{content: content}, fuse.FOPEN_DIRECT_IO, 0
}

// --- Write file handle ---

type overlayWriteHandle struct {
	node    *overlayNode
	baseDir string
}

func (h *overlayWriteHandle) Write(_ context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	h.node.over.layer.writeWithBase(h.node.relPath, h.baseDir, off, data)
	return uint32(len(data)), 0
}

// Read services reads through a write-capable handle. SQLite opens O_RDWR
// and then immediately reads the file header before any write; without this
// method the kernel returns ENOSYS and SQLite reports "disk I/O error (10)".
// Returns current dirty-layer content, falling back to the base file.
func (h *overlayWriteHandle) Read(_ context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if off < 0 {
		return fuse.ReadResultData(nil), syscall.EINVAL
	}
	content := h.node.over.layer.read(h.node.relPath)
	if content == nil {
		var err error
		content, err = os.ReadFile(h.node.absBase())
		if err != nil {
			return nil, syscall.EIO
		}
	}
	if off >= int64(len(content)) {
		return fuse.ReadResultData(nil), 0
	}
	end := off + int64(len(dest))
	if end > int64(len(content)) {
		end = int64(len(content))
	}
	return fuse.ReadResultData(content[off:end]), 0
}

func (h *overlayWriteHandle) Flush(_ context.Context, flags uint32) syscall.Errno   { return 0 }
func (h *overlayWriteHandle) Release(_ context.Context, flags uint32) syscall.Errno { return 0 }

// Fsync is a no-op: the dirty layer is in-memory. Without this handler
// go-fuse returns EOPNOTSUPP, which SQLite maps to a fatal I/O error.
func (h *overlayWriteHandle) Fsync(_ context.Context, flags uint32) syscall.Errno { return 0 }

// --- NodeCreater ---

func (n *overlayNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	rel := n.childRel(name)
	// Seed the dirty map with an empty file (non-nil empty byte slice).
	n.over.layer.truncate(rel, n.over.baseDir, 0)

	child := &overlayNode{over: n.over, relPath: rel}
	stable := fs.StableAttr{Mode: syscall.S_IFREG, Ino: pathIno(rel)}
	inode := n.NewInode(ctx, child, stable)

	out.Mode = syscall.S_IFREG | 0644
	out.Size = 0
	out.Ino = pathIno(rel)
	fh := &overlayWriteHandle{node: child, baseDir: n.over.baseDir}
	return inode, fh, fuse.FOPEN_DIRECT_IO, 0
}

// --- NodeMkdirer ---

func (n *overlayNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	rel := n.childRel(name)
	// Store a non-nil sentinel under the new directory so Readdir/Lookup recognise it
	// as a dirty-layer dir even before any real files are written into it.
	// (Must be non-nil — a nil entry would be interpreted as a tombstone.)
	n.over.layer.truncate(rel+"/.keep", n.over.baseDir, 0)

	child := &overlayNode{over: n.over, relPath: rel}
	stable := fs.StableAttr{Mode: syscall.S_IFDIR, Ino: pathIno(rel)}
	out.Mode = syscall.S_IFDIR | 0755
	out.Size = 0
	out.Ino = pathIno(rel)
	return n.NewInode(ctx, child, stable), 0
}

// --- NodeUnlinker ---

func (n *overlayNode) Unlink(ctx context.Context, name string) syscall.Errno {
	rel := n.childRel(name)
	n.over.layer.delete(rel)
	return 0
}

// --- NodeRmdirer ---

func (n *overlayNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	rel := n.childRel(name)
	// Remove the Mkdir sentinel; if the dir was Mkdir-only, hasDirtyDir() now returns false.
	n.over.layer.delete(rel + "/.keep")
	return 0
}

// --- NodeRenamer ---

func (n *overlayNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	srcRel := n.childRel(name)

	newParentNode, ok := newParent.(*overlayNode)
	if !ok {
		return syscall.EINVAL
	}
	dstRel := newParentNode.childRel(newName)

	// Copy content from dirty or base.
	content := n.over.layer.read(srcRel)
	if content == nil {
		var err error
		content, err = os.ReadFile(filepath.Join(n.over.baseDir, filepath.FromSlash(srcRel)))
		if err != nil {
			return syscall.ENOENT
		}
	}
	// Replace the destination atomically: truncate to the new length (this also
	// stores a non-nil byte slice — never a tombstone, even for empty content),
	// then overwrite with the new bytes. write() short-circuits on len(data)==0,
	// so the truncate() pass is what guarantees a 0-byte file lands as a real
	// (empty) file rather than a tombstone left over from `delete()`.
	n.over.layer.truncate(dstRel, n.over.baseDir, int64(len(content)))
	if len(content) > 0 {
		n.over.layer.write(dstRel, 0, content)
	}
	n.over.layer.delete(srcRel)
	return 0
}
