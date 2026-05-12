// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	msdksecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func (b *MprBackend) addModuleRoleViaModelsdk(unitID model.ID, roleName, description string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ms, ok := elem.(*msdksecurity.ModuleSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ModuleSecurity)", elem)
		}
		mr := msdksecurity.NewModuleRole()
		mr.SetID(element.ID(modelsdkmpr.GenerateID()))
		mr.SetName(roleName)
		mr.SetDescription(description)
		ms.AddModuleRoles(mr)
		return nil
	})
}

func (b *MprBackend) removeModuleRoleViaModelsdk(unitID model.ID, roleName string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ms, ok := elem.(*msdksecurity.ModuleSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ModuleSecurity)", elem)
		}
		for i, mr := range ms.ModuleRolesItems() {
			typed, ok := mr.(*msdksecurity.ModuleRole)
			if ok && typed.Name() == roleName {
				ms.RemoveModuleRoles(i)
				return nil
			}
		}
		return fmt.Errorf("module role %q not found", roleName)
	})
}
