// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

func TestCreateModuleRole_OverwritesAutoProvisioned(t *testing.T) {
	removeCalled := false
	addCount := 0

	ms := genSec.NewModuleSecurity()
	autoRole := genSec.NewModuleRole()
	autoRole.SetName("User")
	autoRole.SetDescription(autoDocumentRoleDescription)
	ms.AddModuleRoles(autoRole)

	const moduleID = model.ID("mod-taskmgr")

	mb := &mock.MockBackend{}
	mb.IsConnectedFunc = func() bool { return true }
	mb.ListModulesFunc = func() ([]*model.Module, error) {
		return []*model.Module{{BaseElement: model.BaseElement{ID: moduleID}, Name: "TaskMgr"}}, nil
	}
	mb.GetModuleSecurityGenFunc = func(id model.ID) (*genSec.ModuleSecurity, error) {
		return ms, nil
	}
	mb.RemoveModuleRoleFunc = func(unitID model.ID, roleName string) error {
		removeCalled = true
		for i, mr := range ms.ModuleRolesItems() {
			if typed, ok := mr.(*genSec.ModuleRole); ok && typed.Name() == roleName {
				ms.RemoveModuleRoles(i)
				break
			}
		}
		return nil
	}
	mb.AddModuleRoleFunc = func(unitID model.ID, roleName, description string) error {
		addCount++
		newRole := genSec.NewModuleRole()
		newRole.SetName(roleName)
		newRole.SetDescription(description)
		ms.AddModuleRoles(newRole)
		return nil
	}
	mb.UpdateQualifiedNameInAllUnitsFunc = func(old, new string) (int, error) { return 0, nil }

	ctx := &ExecContext{
		Backend: mb,
		ExecIO:  ExecIO{Output: &strings.Builder{}, Quiet: true},
	}

	stmt := &ast.CreateModuleRoleStmt{
		Name:        ast.QualifiedName{Module: "TaskMgr", Name: "User"},
		Description: "Task manager user role",
	}

	if err := execCreateModuleRoleGen(ctx, stmt); err != nil {
		t.Fatalf("execCreateModuleRoleGen: %v", err)
	}

	if !removeCalled {
		t.Error("RemoveModuleRole was not called — auto-provisioned role not removed before re-adding")
	}

	roles := ms.ModuleRolesItems()
	if len(roles) != 1 {
		names := make([]string, 0, len(roles))
		for _, r := range roles {
			if typed, ok := r.(*genSec.ModuleRole); ok {
				names = append(names, typed.Name())
			}
		}
		t.Errorf("expected 1 module role, got %d: %v (would cause Mendix CE1613)", len(roles), names)
	}

	if len(roles) == 1 {
		if typed, ok := roles[0].(*genSec.ModuleRole); ok {
			if typed.Description() == autoDocumentRoleDescription {
				t.Error("role still has auto-provisioned description — overwrite failed")
			}
		}
	}
}
