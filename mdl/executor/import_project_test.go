// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"slices"
	"testing"
)

func TestImportOrder_ModuleRolesBeforeEntities(t *testing.T) {
	paths := []string{
		"MyModule/Domain/MyModule.Item.mdl",
		"MyModule/_module_roles.mdl",
		"MyModule/Enumerations/MyModule.Status.mdl",
		"MyModule/_associations.mdl",
		"MyModule/_module.mdl",
	}
	sorted := sortMDLFiles(paths)

	idxModule := slices.Index(sorted, "MyModule/_module.mdl")
	idxEnum := slices.Index(sorted, "MyModule/Enumerations/MyModule.Status.mdl")
	idxRoles := slices.Index(sorted, "MyModule/_module_roles.mdl")
	idxDomain := slices.Index(sorted, "MyModule/Domain/MyModule.Item.mdl")
	idxAssoc := slices.Index(sorted, "MyModule/_associations.mdl")

	if idxModule >= idxEnum {
		t.Errorf("_module.mdl (%d) must precede Enumerations (%d)", idxModule, idxEnum)
	}
	if idxEnum >= idxRoles {
		t.Errorf("Enumerations (%d) must precede _module_roles.mdl (%d)", idxEnum, idxRoles)
	}
	if idxRoles >= idxDomain {
		t.Errorf("_module_roles.mdl (%d) must precede Domain/ (%d)", idxRoles, idxDomain)
	}
	if idxDomain >= idxAssoc {
		t.Errorf("Domain/ (%d) must precede _associations.mdl (%d)", idxDomain, idxAssoc)
	}
}
