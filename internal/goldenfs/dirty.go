// SPDX-License-Identifier: Apache-2.0

//go:build linux

package goldenfs

// dirtyLayer is the in-memory copy-on-write layer.
// Real implementation lands in Task 2 (TDD).
type dirtyLayer struct{}

func newDirtyLayer() *dirtyLayer { return &dirtyLayer{} }

func (l *dirtyLayer) commit(baseDir string) error {
	_ = baseDir
	return nil
}

func (l *dirtyLayer) rollback() {}
