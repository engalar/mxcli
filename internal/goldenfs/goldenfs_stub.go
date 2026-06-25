// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package goldenfs

// Snapshot is a no-op placeholder on non-Linux platforms.
type Snapshot struct{}

func Open(_ string) (*Snapshot, error) { return nil, ErrNotSupported }
func (s *Snapshot) MountDir() string   { return "" }
func (s *Snapshot) Commit() error      { return ErrNotSupported }
func (s *Snapshot) Rollback()          {}
func (s *Snapshot) Close() error       { return nil }

func (s *Snapshot) DirtyPaths() []string          { return nil }
func (s *Snapshot) ReadDirtyFile(_ string) []byte { return nil }
