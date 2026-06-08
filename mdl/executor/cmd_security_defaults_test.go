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

// TestFilterAutoDocumentRoles_StripsAutoUserFromMixedList verifies that
// filterAutoDocumentRoles removes "Module.User" (the auto-provisioned role)
// while preserving explicit cross-module grants.
//
// This is the red-test for CE0148: when `grant execute on microflow FT.X to HD.ManagerRole`
// is called on a document that already has ["FT.User"] from auto-provisioning,
// the merged list must contain ONLY "HD.ManagerRole", not ["FT.User", "HD.ManagerRole"].
// Without the fix, Mendix fails to resolve the combined list and raises CE0148.
func TestFilterAutoDocumentRoles_StripsAutoUserFromMixedList(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "removes FT.User from mixed list",
			input: []string{"FT.User", "HD.ManagerRole"},
			want:  []string{"HD.ManagerRole"},
		},
		{
			name:  "removes HD.User from same-module only list",
			input: []string{"HD.User"},
			want:  []string{},
		},
		{
			name:  "preserves roles that are not named User",
			input: []string{"HD.ManagerRole", "HD.AgentRole"},
			want:  []string{"HD.ManagerRole", "HD.AgentRole"},
		},
		{
			name:  "nil input returns empty",
			input: nil,
			want:  []string{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := filterAutoDocumentRoles(c.input)
			if len(got) != len(c.want) {
				t.Errorf("filterAutoDocumentRoles(%v) = %v; want %v", c.input, got, c.want)
				return
			}
			for i, r := range got {
				if r != c.want[i] {
					t.Errorf("filterAutoDocumentRoles(%v)[%d] = %q; want %q", c.input, i, r, c.want[i])
				}
			}
		})
	}
}
