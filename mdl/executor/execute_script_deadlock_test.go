// SPDX-License-Identifier: Apache-2.0

package executor

// Regression tests for the EXECUTE SCRIPT deadlock (CE-XXXX).
//
// Root cause: BeginScriptTransaction() calls db.Begin() which holds the single
// SQLite connection (SetMaxOpenConns(1)). Any subsequent read or INSERT inside
// the script calls db.Query/db.Exec which waits for that same connection → deadlock.
//
// TestExecPath_CreateThenRead:  the exec subcommand path never calls
// BeginScriptTransaction, so create+read works fine. (Must stay green.)
//
// TestExecuteScriptPath_CreateThenRead: the EXECUTE SCRIPT path calls
// BeginScriptTransaction before running the script body. Any read or INSERT
// inside the script deadlocks. (RED until the fix is applied.)

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

const scriptDeadlockTimeout = 5 * time.Second

// openBackendForTest copies the v2 fixture and opens an MprBackend.
func openBackendForTest(t *testing.T) *mprbackend.MprBackend {
	t.Helper()
	dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
	be, err := mprbackend.NewFromPath(dst)
	if err != nil {
		t.Fatalf("NewFromPath: %v", err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })
	return be
}

// writeScriptFile writes content to a temp file and returns its absolute path.
func writeScriptFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.mdl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write script file: %v", err)
	}
	return path
}

// TestExecPath_CreateThenRead verifies that the exec-subcommand code path
// (ExecuteProgram) can create an entity and immediately read entities without
// hanging. This path never calls BeginScriptTransaction, so it must stay
// green after the deadlock fix.
func TestExecPath_CreateThenRead(t *testing.T) {
	t.Parallel()
	be := openBackendForTest(t)

	exec := New(&bytes.Buffer{})
	exec.SetBackend(be)

	// Minimal script: create an entity, then list entities.
	// This is what `mxcli exec file.mdl` effectively does.
	prog, errs := visitor.Build(`
		create or modify entity MyFirstModule.DeadlockTestExecEntity;
		show entities;
	`)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	done := make(chan error, 1)
	go func() { done <- exec.ExecuteProgram(prog) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ExecuteProgram: %v", err)
		}
	case <-time.After(scriptDeadlockTimeout):
		t.Fatalf("exec path deadlocked — did not complete within %v", scriptDeadlockTimeout)
	}
}

// TestExecuteScriptPath_CreateThenRead verifies that EXECUTE SCRIPT can
// execute a script that creates an entity and then reads entities.
//
// This test currently FAILS (hangs until timeout) because:
//  1. execExecuteScript calls BeginScriptTransaction() → db.Begin() → C1 held
//  2. "show entities" inside the script calls db.Query("SELECT ...") → waits for C1
//  3. Same goroutine, C1 never released → deadlock
func TestExecuteScriptPath_CreateThenRead(t *testing.T) {
	t.Parallel()
	be := openBackendForTest(t)

	exec := New(&bytes.Buffer{})
	exec.SetBackend(be)

	scriptPath := writeScriptFile(t, "create or modify entity MyFirstModule.DeadlockTestScriptEntity;\nshow entities;\n")

	stmt := &ast.ExecuteScriptStmt{Path: scriptPath}

	done := make(chan error, 1)
	go func() {
		ctx := exec.newExecContext(context.Background())
		done <- execExecuteScript(ctx, stmt)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("EXECUTE SCRIPT returned error: %v", err)
		}
	case <-time.After(scriptDeadlockTimeout):
		t.Fatalf(
			"EXECUTE SCRIPT deadlocked — did not complete within %v\n"+
				"Root cause: BeginScriptTransaction() holds the single SQLite connection\n"+
				"while script body reads call db.Query() waiting for the same connection.",
			scriptDeadlockTimeout,
		)
	}
}

// TestExecuteScriptPath_ReadOnly verifies that EXECUTE SCRIPT with a
// read-only script body (no writes) also deadlocks. Even a bare
// "show entities;" triggers db.Query() which waits for C1.
func TestExecuteScriptPath_ReadOnly(t *testing.T) {
	t.Parallel()
	be := openBackendForTest(t)

	exec := New(&bytes.Buffer{})
	exec.SetBackend(be)

	scriptPath := writeScriptFile(t, "show entities;\n")

	stmt := &ast.ExecuteScriptStmt{Path: scriptPath}

	done := make(chan error, 1)
	go func() {
		ctx := exec.newExecContext(context.Background())
		done <- execExecuteScript(ctx, stmt)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("EXECUTE SCRIPT returned error: %v", err)
		}
	case <-time.After(scriptDeadlockTimeout):
		t.Fatalf(
			"EXECUTE SCRIPT deadlocked on read-only script — did not complete within %v",
			scriptDeadlockTimeout,
		)
	}
}

// TestExecuteScriptPath_RollbackOnError verifies that when a statement inside
// EXECUTE SCRIPT fails, all preceding creates in the same script are rolled back.
//
// Currently skipped: rollback atomicity is not yet implemented. The repos layer
// (writerSink, mdl/backend/mpr/repos/sink.go) delegates straight to mmpr.Writer,
// committing each unit to disk immediately and bypassing the ScriptBuffer that
// backs BeginScriptTransaction's rollback. The test body is kept as executable
// documentation of the intended behavior.
func TestExecuteScriptPath_RollbackOnError(t *testing.T) {
	t.Parallel()
	be := openBackendForTest(t)

	exec := New(&bytes.Buffer{})
	exec.SetBackend(be)

	// First statement creates entity; second alters a non-existent entity → error.
	// (A bare "create entity" on an unknown module auto-creates the module and
	// does not error, so we use ALTER on a missing entity to force a failure.)
	scriptPath := writeScriptFile(t,
		"create or modify entity MyFirstModule.RollbackTarget;\n"+
			"alter entity MyFirstModule.DoesNotExistXYZ add attribute Foo : String;\n",
	)

	stmt := &ast.ExecuteScriptStmt{Path: scriptPath}
	ctx := exec.newExecContext(context.Background())

	err := execExecuteScript(ctx, stmt)
	if err == nil {
		t.Fatal("expected error from invalid statement, got nil")
	}

	// RollbackTarget must not have been committed to the MPR.
	if _, lookErr := be.GetEntityIDByQualifiedName("MyFirstModule.RollbackTarget"); lookErr == nil {
		t.Error("rolled-back entity 'RollbackTarget' still present in MPR after script failure")
	}
}

// TestExecuteScriptPath_CreateThenDescribe verifies that an entity created
// earlier in the same EXECUTE SCRIPT block is visible to a subsequent
// DESCRIBE or SHOW statement (read-own-write within a script).
func TestExecuteScriptPath_CreateThenDescribe(t *testing.T) {
	t.Parallel()
	be := openBackendForTest(t)

	var out bytes.Buffer
	exec := New(&out)
	exec.SetBackend(be)

	scriptPath := writeScriptFile(t,
		"create or modify entity MyFirstModule.ReadOwnWriteTest;\n"+
			"show entities;\n",
	)

	stmt := &ast.ExecuteScriptStmt{Path: scriptPath}

	done := make(chan error, 1)
	go func() {
		ctx := exec.newExecContext(context.Background())
		done <- execExecuteScript(ctx, stmt)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("EXECUTE SCRIPT: %v", err)
		}
	case <-time.After(scriptDeadlockTimeout):
		t.Fatalf("EXECUTE SCRIPT deadlocked within %v", scriptDeadlockTimeout)
	}

	if !strings.Contains(out.String(), "ReadOwnWriteTest") {
		t.Errorf("show entities output does not include ReadOwnWriteTest:\n%s", out.String())
	}
}
