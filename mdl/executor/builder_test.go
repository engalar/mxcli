// SPDX-License-Identifier: Apache-2.0

package executor_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/executor"
)

func TestBuilder_WithBackend_CreatesConnectedExecutor(t *testing.T) {
	m := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	var buf strings.Builder
	exec := executor.Build().Out(&buf).WithBackend(m).Quiet().Create()
	defer exec.Close()

	if !exec.IsConnected() {
		t.Fatal("executor should be connected after WithBackend")
	}
}

func TestBuilder_WithFactory_DoesNotCallFactoryAtBuildTime(t *testing.T) {
	called := false
	var buf strings.Builder
	exec := executor.Build().
		Out(&buf).
		WithFactory(func() executor.BackendIface {
			called = true
			return &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
		}).
		Quiet().
		Create()
	defer exec.Close()

	if called {
		t.Fatal("factory should not be called at Build time, only on CONNECT")
	}
}

func TestBuilder_Quiet_AndProgressOut(t *testing.T) {
	m := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	var stdout, progress strings.Builder

	exec := executor.Build().
		Out(&stdout).
		ProgressOut(&progress).
		WithBackend(m).
		Quiet().
		Create()
	defer exec.Close()

	// Just verify it compiles and constructs without panic.
	_ = exec
}
