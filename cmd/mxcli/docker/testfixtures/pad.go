// cmd/mxcli/docker/testfixtures/pad.go
// SPDX-License-Identifier: Apache-2.0

package testfixtures

import (
	"os"
	"path/filepath"
	"testing"
)

// FakePAD creates a minimal valid PAD directory structure for StartLocal tests.
// It satisfies hasExtractedPADLayout() exactly: bin/start (executable),
// lib/runtime/launcher/runtimelauncher.jar, app/, bin/, etc/, lib/ dirs.
type FakePAD struct {
	Dir string
}

// NewFakePAD creates the PAD structure in a temp dir and registers cleanup.
func NewFakePAD(t *testing.T) *FakePAD {
	t.Helper()
	dir := t.TempDir()
	p := &FakePAD{Dir: dir}

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("FakePAD setup: %v", err)
		}
	}
	mkdirAll := func(rel string) {
		must(os.MkdirAll(filepath.Join(dir, rel), 0755))
	}
	writeFile := func(rel, content string, mode os.FileMode) {
		path := filepath.Join(dir, rel)
		must(os.MkdirAll(filepath.Dir(path), 0755))
		must(os.WriteFile(path, []byte(content), mode))
	}

	// Required dirs
	mkdirAll("app/model/lib/userlib")
	mkdirAll("bin")
	mkdirAll("etc/configurations")
	mkdirAll("etc/constants")
	mkdirAll("lib/runtime/lib/x64")
	mkdirAll("lib/runtime/launcher")

	// bin/start — executable POSIX script
	writeFile("bin/start", `#!/bin/sh
exec java -jar "$ROOT_PATH/lib/runtime/launcher/runtimelauncher.jar" "$ROOT_PATH/app/." "$ROOT_PATH/etc/Default"
`, 0755)

	// bin/start.bat — Windows batch script
	writeFile("bin/start.bat", `@echo off
java -jar "%ROOT_PATH%\lib\runtime\launcher\runtimelauncher.jar" "%ROOT_PATH%\app\." "%ROOT_PATH%\etc\Default"
`, 0644)

	// runtimelauncher.jar placeholder (content irrelevant for path tests)
	writeFile("lib/runtime/launcher/runtimelauncher.jar", "fake-jar", 0644)

	// Minimal HOCON config chain
	writeFile("etc/Default", `
include file("etc/configurations/Default.conf")
include file("etc/variables.conf")
`, 0644)
	writeFile("etc/configurations/Default.conf", `
runtime.params {
  DatabaseType = HSQLDB
  DatabaseName = default
}
admin { port = 8090 }
runtime.http { port = 8080 }
`, 0644)
	writeFile("etc/variables.conf", `
admin.adminPassword = ${?ADMIN_ADMINPASSWORD}
runtime.adminUser.password = ${?RUNTIME_ADMINUSER_PASSWORD}
runtime.params {
  "DatabaseType" = ${?RUNTIME_PARAMS_DATABASETYPE}
  "DatabaseJdbcUrl" = ${?RUNTIME_PARAMS_DATABASEJDBCURL}
  "DatabaseUserName" = ${?RUNTIME_PARAMS_DATABASEUSERNAME}
  "DatabasePassword" = ${?RUNTIME_PARAMS_DATABASEPASSWORD}
}
`, 0644)

	return p
}

// SetJVMHeap adds a jvm.heap entry to a dedicated jvm.conf include file.
// Call before passing Dir to StartLocal.
func (p *FakePAD) SetJVMHeap(t *testing.T, heap string) *FakePAD {
	t.Helper()
	path := filepath.Join(p.Dir, "etc", "jvm.conf")
	if err := os.WriteFile(path, []byte("jvm.heap = "+heap+"\n"), 0644); err != nil {
		t.Fatalf("FakePAD.SetJVMHeap: %v", err)
	}
	return p
}
