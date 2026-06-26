// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"
)

func TestResolveMxcliPath_ReturnsExecutable(t *testing.T) {
	got := resolveMxcliPath()
	exe, _ := os.Executable()
	if got != exe {
		t.Errorf("got %q, want os.Executable() = %q", got, exe)
	}
}
