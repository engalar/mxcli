// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

func newModuleSecurityWithRole(name, desc string) *genSec.ModuleSecurity {
	ms := genSec.NewModuleSecurity()
	mr := genSec.NewModuleRole()
	mr.SetName(name)
	mr.SetDescription(desc)
	ms.AddModuleRoles(mr)
	return ms
}

func TestModuleUsesAutoDocumentRoleGen_SingleMatchingRole(t *testing.T) {
	ms := newModuleSecurityWithRole(autoDocumentRoleName, autoDocumentRoleDescription)
	if !moduleUsesAutoDocumentRoleGen(ms) {
		t.Fatal("expected true for single matching role")
	}
}

func TestModuleUsesAutoDocumentRoleGen_TwoRoles_False(t *testing.T) {
	ms := newModuleSecurityWithRole(autoDocumentRoleName, autoDocumentRoleDescription)
	mr2 := genSec.NewModuleRole()
	mr2.SetName("Admin")
	mr2.SetDescription("Admin role")
	ms.AddModuleRoles(mr2)
	if moduleUsesAutoDocumentRoleGen(ms) {
		t.Fatal("expected false for two roles")
	}
}

func TestModuleUsesAutoDocumentRoleGen_NilMS_False(t *testing.T) {
	if moduleUsesAutoDocumentRoleGen(nil) {
		t.Fatal("expected false for nil ms")
	}
}

func TestModuleUsesAutoDocumentRoleGen_WrongName_False(t *testing.T) {
	ms := newModuleSecurityWithRole("Admin", autoDocumentRoleDescription)
	if moduleUsesAutoDocumentRoleGen(ms) {
		t.Fatal("expected false when name does not match")
	}
}
