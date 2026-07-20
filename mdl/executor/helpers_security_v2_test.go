// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"io"
	"testing"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// newSecurityTestContext builds an ExecContext with Security wired from a
// fixture MPR, following the same pattern as newGenDescribeContext.
func newSecurityTestContext(t *testing.T) *ExecContext {
	t.Helper()
	w := openMprWriterForTest(t)
	repoCtx := mprbackend.NewExecutorContext(w)

	path := w.ConcreteReader().Path()
	be, err := mprbackend.NewFromPath(path)
	if err != nil {
		t.Fatalf("mprbackend.NewFromPath(%s): %v", path, err)
	}
	t.Cleanup(func() { _ = be.Disconnect() })

	ctx := &ExecContext{
		Backend:   be,
		Security: repoCtx.Security, DomainModels: repoCtx.DomainModels, Pages: repoCtx.Pages,
		Output: io.Discard,
	}
	ctx.initRoles()
	ctx.ensureCache()
	return ctx
}

func TestListModuleSecurityWithContainerGen_CachesAcrossCalls(t *testing.T) {
	ctx := newSecurityTestContext(t)
	list1, err := listModuleSecurityWithContainerGen(ctx)
	if err != nil {
		t.Fatalf("listModuleSecurityWithContainerGen: %v", err)
	}
	list2, _ := listModuleSecurityWithContainerGen(ctx)
	if len(list1) != len(list2) {
		t.Fatalf("cache produced different lengths: %d vs %d", len(list1), len(list2))
	}
	if len(list1) > 0 && &list1[0] != &list2[0] {
		t.Fatalf("cache must return same slice header on second call")
	}
}

func TestGetProjectSecurityGen_CachesAcrossCalls(t *testing.T) {
	ctx := newSecurityTestContext(t)
	ps1, err := getProjectSecurityGen(ctx)
	if err != nil {
		t.Fatalf("getProjectSecurityGen: %v", err)
	}
	ps2, _ := getProjectSecurityGen(ctx)
	if ps1 != ps2 {
		t.Fatalf("cache must return same pointer on second call")
	}
}
