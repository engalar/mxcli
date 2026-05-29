// SPDX-License-Identifier: Apache-2.0

//go:build windows

package main

import (
	"fmt"
	"io"
)

// extractTarZst is not used on Windows (daemon is distributed as .exe.zip).
// This stub satisfies the compiler on Windows without importing zstd/tar.
func extractTarZst(_ io.Reader, _, expectedName string) error {
	return fmt.Errorf("tar.zst extraction not supported on Windows (expected %q)", expectedName)
}
