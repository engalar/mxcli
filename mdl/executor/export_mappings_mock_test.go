// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

func TestShowExportMappings_Mock(t *testing.T) {
	mod := mkModule("Integration")
	em := &model.ExportMapping{
		BaseElement: model.BaseElement{ID: nextID("em")},
		ContainerID: mod.ID,
		Name:        "ExportOrders",
	}

	h := mkHierarchy(mod)
	withContainer(h, em.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		ListExportMappingsFunc: func() ([]*model.ExportMapping, error) { return []*model.ExportMapping{em}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listExportMappings(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "Export Mapping")
	assertContainsStr(t, out, "Integration.ExportOrders")
}

func TestShowExportMappings_FilterByModule(t *testing.T) {
	mod1 := mkModule("Integration")
	mod2 := mkModule("Other")
	em1 := &model.ExportMapping{
		BaseElement: model.BaseElement{ID: nextID("em")},
		ContainerID: mod1.ID,
		Name:        "ExportOrders",
	}
	em2 := &model.ExportMapping{
		BaseElement: model.BaseElement{ID: nextID("em")},
		ContainerID: mod2.ID,
		Name:        "ExportOther",
	}

	h := mkHierarchy(mod1, mod2)
	withContainer(h, em1.ContainerID, mod1.ID)
	withContainer(h, em2.ContainerID, mod2.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		ListExportMappingsFunc: func() ([]*model.ExportMapping, error) { return []*model.ExportMapping{em1, em2}, nil },
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listExportMappings(ctx, "Integration"))

	out := buf.String()
	assertContainsStr(t, out, "Integration.ExportOrders")
	assertNotContainsStr(t, out, "Other.ExportOther")
}

func TestDescribeExportMapping_Mock(t *testing.T) {
	mod := mkModule("Integration")
	em := &model.ExportMapping{
		BaseElement: model.BaseElement{ID: nextID("em")},
		ContainerID: mod.ID,
		Name:        "ExportOrders",
	}

	h := mkHierarchy(mod)
	withContainer(h, em.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetExportMappingByQualifiedNameFunc: func(moduleName, name string) (*model.ExportMapping, error) {
			return em, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeExportMapping(ctx, ast.QualifiedName{Module: "Integration", Name: "ExportOrders"}))
	assertContainsStr(t, buf.String(), "create export mapping")
}

func TestDescribeExportMapping_NotFound(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetExportMappingByQualifiedNameFunc: func(moduleName, name string) (*model.ExportMapping, error) {
			return nil, fmt.Errorf("export mapping not found: %s.%s", moduleName, name)
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeExportMapping(ctx, ast.QualifiedName{Module: "Integration", Name: "NoSuch"}))
}

func TestDescribeExportMapping_NotFound_NilNil(t *testing.T) {
	// Real MprBackend returns nil, nil when not found — ensure no nil dereference.
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetExportMappingByQualifiedNameFunc: func(moduleName, name string) (*model.ExportMapping, error) {
			return nil, nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeExportMapping(ctx, ast.QualifiedName{Module: "Integration", Name: "NoSuch"}))
}

func TestDropExportMapping_NotFound_NilNil(t *testing.T) {
	// Real MprBackend returns nil, nil when not found — ensure no nil dereference.
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetExportMappingByQualifiedNameFunc: func(moduleName, name string) (*model.ExportMapping, error) {
			return nil, nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.DropExportMappingStmt{Name: ast.QualifiedName{Module: "Integration", Name: "NoSuch"}}
	assertError(t, execDropExportMappingFn(ctx, stmt, ctx.Deps))
}

func TestCreateExportMapping_OrModify_PreservesID(t *testing.T) {
	mod := mkModule("Integration")
	existingID := nextID("em")
	existing := &model.ExportMapping{
		BaseElement: model.BaseElement{ID: existingID},
		ContainerID: mod.ID,
		Name:        "ExportOrders",
	}
	h := mkHierarchy(mod)
	withContainer(h, existing.ContainerID, mod.ID)

	var updatedID model.ID
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetExportMappingByQualifiedNameFunc: func(moduleName, name string) (*model.ExportMapping, error) {
			if moduleName == "Integration" && name == "ExportOrders" {
				return existing, nil
			}
			return nil, nil
		},
		UpdateExportMappingFunc: func(em *model.ExportMapping) error {
			updatedID = em.ID
			return nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execCreateExportMappingFn(ctx, &ast.CreateExportMappingStmt{
		Name:           ast.QualifiedName{Module: "Integration", Name: "ExportOrders"},
		SchemaKind:     "JSON_STRUCTURE",
		SchemaRef:      ast.QualifiedName{Module: "Integration", Name: "PetSchema"},
		CreateOrModify: true,
	}, ctx.Deps)
	assertNoError(t, err)
	assertContainsStr(t, buf.String(), "Modified export mapping")
	if updatedID != existingID {
		t.Errorf("UpdateExportMapping called with ID %q, want %q", updatedID, existingID)
	}
}

func TestCreateExportMapping_AlreadyExists_NoOrModify(t *testing.T) {
	mod := mkModule("Integration")
	existing := &model.ExportMapping{
		BaseElement: model.BaseElement{ID: nextID("em")},
		ContainerID: mod.ID,
		Name:        "ExportOrders",
	}

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetExportMappingByQualifiedNameFunc: func(moduleName, name string) (*model.ExportMapping, error) {
			return existing, nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execCreateExportMappingFn(ctx, &ast.CreateExportMappingStmt{
		Name: ast.QualifiedName{Module: "Integration", Name: "ExportOrders"},
	}, ctx.Deps)
	assertError(t, err)
}
