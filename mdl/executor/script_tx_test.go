// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
)

// mockScriptTx is a stub backend.ScriptTransaction whose Commit / Rollback
// counters let tests assert lifecycle behavior.
type mockScriptTx struct {
	commits   int
	rollbacks int
	commitErr error
}

func (m *mockScriptTx) Commit() error {
	m.commits++
	return m.commitErr
}

func (m *mockScriptTx) Rollback() error {
	m.rollbacks++
	return nil
}

// writeTempScript writes body to a temp .mdl file and returns its path.
func writeTempScript(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "script.mdl")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return p
}

// scriptCtx builds an ExecContext wired to a mock backend that records
// script transaction lifecycle through tx.
func scriptCtx(t *testing.T, tx *mockScriptTx, execFn func(ast.Statement) error) (*ExecContext, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		BeginScriptTransactionFunc: func() (backend.ScriptTransaction, error) {
			return tx, nil
		},
	}
	ctx := &ExecContext{
		Context:       context.Background(),
		Backend:       mb,
		ExecIO:        ExecIO{Output: &buf, Format: FormatTable},
		ExecCallbacks: ExecCallbacks{ExecuteFn: execFn},
	}
	ctx.initRoles()
	return ctx, &buf
}

func TestExecuteScript_Success_CommitCalled(t *testing.T) {
	tx := &mockScriptTx{}
	path := writeTempScript(t, "show modules;\n")
	ctx, _ := scriptCtx(t, tx, func(stmt ast.Statement) error { return nil })

	if err := execExecuteScript(ctx, &ast.ExecuteScriptStmt{Path: path}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.commits != 1 {
		t.Errorf("expected Commit to be called once, got %d", tx.commits)
	}
	if tx.rollbacks != 0 {
		t.Errorf("expected Rollback NOT to be called, got %d", tx.rollbacks)
	}
}

func TestExecuteScript_FailureMidScript_RollbackCalled(t *testing.T) {
	tx := &mockScriptTx{}
	path := writeTempScript(t, "show modules;\nshow modules;\nshow modules;\n")

	callCount := 0
	ctx, buf := scriptCtx(t, tx, func(stmt ast.Statement) error {
		callCount++
		if callCount == 2 {
			return fmt.Errorf("simulated failure")
		}
		return nil
	})

	err := execExecuteScript(ctx, &ast.ExecuteScriptStmt{Path: path})
	if err == nil {
		t.Fatal("expected error from failing statement, got nil")
	}
	if tx.rollbacks != 1 {
		t.Errorf("expected Rollback to be called once, got %d", tx.rollbacks)
	}
	if tx.commits != 0 {
		t.Errorf("expected Commit NOT to be called, got %d", tx.commits)
	}
	// User-visible message must announce the rollback so they aren't surprised.
	if got := buf.String(); !bytes.Contains([]byte(got), []byte("rolled back")) {
		t.Errorf("expected output to mention rollback, got: %q", got)
	}
}

// EXIT inside a script is an intentional stop; pending work should commit.
func TestExecuteScript_ExitInsideScript_CommitsPriorWork(t *testing.T) {
	tx := &mockScriptTx{}
	path := writeTempScript(t, "show modules;\nshow modules;\n")
	ctx, _ := scriptCtx(t, tx, func(stmt ast.Statement) error {
		return ErrExit
	})

	if err := execExecuteScript(ctx, &ast.ExecuteScriptStmt{Path: path}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.commits != 1 {
		t.Errorf("expected Commit to be called once on EXIT, got %d", tx.commits)
	}
	if tx.rollbacks != 0 {
		t.Errorf("expected Rollback NOT to be called on EXIT, got %d", tx.rollbacks)
	}
}

// When not connected, execExecuteScript must skip BeginScriptTransaction
// entirely — read-only / unconnected sessions still run scripts.
func TestExecuteScript_NotConnected_NoTransaction(t *testing.T) {
	tx := &mockScriptTx{}
	path := writeTempScript(t, "show modules;\n")
	var buf bytes.Buffer
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return false },
		BeginScriptTransactionFunc: func() (backend.ScriptTransaction, error) {
			t.Fatal("BeginScriptTransaction must not be called when disconnected")
			return tx, nil
		},
	}
	ctx := &ExecContext{
		Context:       context.Background(),
		Backend:       mb,
		ExecIO:        ExecIO{Output: &buf, Format: FormatTable},
		ExecCallbacks: ExecCallbacks{ExecuteFn: func(stmt ast.Statement) error { return nil }},
	}
	ctx.initRoles()
	if err := execExecuteScript(ctx, &ast.ExecuteScriptStmt{Path: path}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.commits != 0 || tx.rollbacks != 0 {
		t.Errorf("expected no tx activity, got commits=%d rollbacks=%d", tx.commits, tx.rollbacks)
	}
}

// Nested EXECUTE SCRIPT must inherit the outer transaction rather than
// open a second one (which the mock backend would refuse if we wanted
// — here we assert only that BeginScriptTransaction is called exactly
// once across the nested invocation).
func TestExecuteScript_Nested_ReusesOuterTransaction(t *testing.T) {
	tx := &mockScriptTx{}
	inner := writeTempScript(t, "show modules;\n")
	outer := writeTempScript(t, fmt.Sprintf("execute script '%s';\n", inner))

	beginCalls := 0
	var buf bytes.Buffer
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		BeginScriptTransactionFunc: func() (backend.ScriptTransaction, error) {
			beginCalls++
			return tx, nil
		},
	}
	ctx := &ExecContext{
		Context: context.Background(),
		Backend: mb,
		ExecIO:  ExecIO{Output: &buf, Format: FormatTable},
	}
	ctx.initRoles()
	// ExecuteFn re-enters execExecuteScript for nested EXECUTE SCRIPT
	// statements, otherwise it's a no-op.
	ctx.ExecuteFn = func(stmt ast.Statement) error {
		if es, ok := stmt.(*ast.ExecuteScriptStmt); ok {
			return execExecuteScript(ctx, es)
		}
		return nil
	}

	if err := execExecuteScript(ctx, &ast.ExecuteScriptStmt{Path: outer}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if beginCalls != 1 {
		t.Errorf("expected BeginScriptTransaction once (root only), got %d", beginCalls)
	}
	if tx.commits != 1 {
		t.Errorf("expected Commit once, got %d", tx.commits)
	}
}

// Defensive: a Commit error from the underlying tx must propagate.
func TestExecuteScript_CommitError_Surfaces(t *testing.T) {
	tx := &mockScriptTx{commitErr: errors.New("boom")}
	path := writeTempScript(t, "show modules;\n")
	ctx, _ := scriptCtx(t, tx, func(stmt ast.Statement) error { return nil })

	err := execExecuteScript(ctx, &ast.ExecuteScriptStmt{Path: path})
	if err == nil {
		t.Fatal("expected commit error to surface, got nil")
	}
	if tx.commits != 1 {
		t.Errorf("expected Commit attempted once, got %d", tx.commits)
	}
}
