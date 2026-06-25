// SPDX-License-Identifier: Apache-2.0

// Package goldenfs provides FUSE-based copy-on-write overlays for MPR
// integration tests. The interfaces below live in this package (not on the
// consumer side) because multiple consumers share them — same reason
// io.Reader lives in "io", not in every package that reads bytes.
//
// Import what you need:
//
//   var o goldenfs.Overlay     // MountDir + Close + Rollback
//   var d goldenfs.DirtyReader // + DirtyPaths + ReadDirtyFile
//   var c goldenfs.Committer   // + Commit
package goldenfs

import "errors"

// ErrNotSupported is returned on non-Linux platforms where FUSE is unavailable.
var ErrNotSupported = errors.New("goldenfs: FUSE overlay requires Linux")

// Overlay is a copy-on-write view of a base directory.
// All writes go to an in-memory layer that can be discarded.
type Overlay interface {
	MountDir() string
	Close() error
	Rollback()
}

// DirtyReader extends Overlay with write-path introspection
// for test assertions and audit.
type DirtyReader interface {
	Overlay
	DirtyPaths() []string
	ReadDirtyFile(relPath string) []byte
}

// Committer extends DirtyReader with the ability to flush
// dirty writes to the base directory. Used by golden-update
// pipelines.
type Committer interface {
	DirtyReader
	Commit() error
}

// Option configures overlay behaviour.
type Option func(*overlayConfig)

type overlayConfig struct {
	skipOrphanCheck bool
}

// WithSkipOrphanCheck disables the cleanup of stale FUSE mounts
// from crashed processes. Useful when nesting overlays or when
// the caller manages lifecycle externally.
func WithSkipOrphanCheck() Option {
	return func(c *overlayConfig) { c.skipOrphanCheck = true }
}
