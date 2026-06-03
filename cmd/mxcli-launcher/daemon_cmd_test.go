// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"
)

func TestRunDaemonCommand_UnknownSubcommand(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	code := e.runDaemonCommand([]string{"nonexistent"})
	if code == 0 {
		t.Error("expected non-zero exit for unknown subcommand")
	}
}

func TestRunDaemonCommand_NoArgs(t *testing.T) {
	e := &Env{HomeDir: t.TempDir()}
	code := e.runDaemonCommand([]string{})
	if code == 0 {
		t.Error("expected non-zero exit for no args")
	}
}
