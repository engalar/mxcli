// SPDX-License-Identifier: Apache-2.0

package testutil_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/executor/testutil"
	"github.com/mendixlabs/mxcli/model"
)

func TestNew_MockBackendAccessible(t *testing.T) {
	te := testutil.New(t)
	if te.Mock == nil {
		t.Fatal("Mock should be non-nil after New(t)")
	}
}

func TestNew_RunReturnsOutput(t *testing.T) {
	te := testutil.New(t)
	te.Mock.ListModulesFunc = func() ([]*model.Module, error) {
		return []*model.Module{
			{BaseElement: model.BaseElement{ID: "mod-1"}, Name: "SalesModule"},
		}, nil
	}

	out, err := te.Run("show modules")
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if !strings.Contains(out, "SalesModule") {
		t.Errorf("output should contain SalesModule, got:\n%s", out)
	}
}

func TestNew_RunError_WhenNotConnected(t *testing.T) {
	te := testutil.New(t)
	te.Mock.IsConnectedFunc = func() bool { return false }

	_, err := te.RunError("show modules")
	if err == nil {
		t.Fatal("expected error when not connected, got nil")
	}
}
