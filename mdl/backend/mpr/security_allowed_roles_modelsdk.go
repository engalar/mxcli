// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMF "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genODP "github.com/mendixlabs/mxcli/modelsdk/gen/odatapublish"
	genPages "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genREST "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
)

// updateAllowedRolesViaModelsdk replaces the role list on a Microflow,
// Nanoflow, Page, or PublishedODataService unit via the gen type setters.
//
// CRITICAL: Microflow, Nanoflow, and PublishedODataService store roles under
// the BSON key "AllowedModuleRoles". Page stores them under "AllowedRoles".
// Routing through the gen type setters guarantees the correct key — earlier
// implementations that patched "AllowedRoles" unconditionally silently dropped
// the role list on Microflow/Nanoflow units.
func (b *MprBackend) updateAllowedRolesViaModelsdk(unitID model.ID, roles []string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		switch typed := elem.(type) {
		case *genMF.Microflow:
			typed.SetAllowedModuleRolesQualifiedNames(roles)
		case *genMF.Nanoflow:
			typed.SetAllowedModuleRolesQualifiedNames(roles)
		case *genPages.Page:
			typed.SetAllowedRolesQualifiedNames(roles)
		case *genODP.PublishedODataService2:
			typed.SetAllowedModuleRolesQualifiedNames(roles)
		default:
			return fmt.Errorf("updateAllowedRoles: unsupported unit type %T for unit %s", elem, unitID)
		}
		return nil
	})
}

func (b *MprBackend) removeFromAllowedRolesViaModelsdk(unitID model.ID, roleName string) (bool, error) {
	removeFromSlice := func(roles []string) ([]string, bool) {
		for i, r := range roles {
			if r == roleName {
				return append(roles[:i], roles[i+1:]...), true
			}
		}
		return roles, false
	}
	removed := false
	err := b.msdkWrite(unitID, func(elem element.Element) error {
		switch typed := elem.(type) {
		case *genMF.Microflow:
			updated, found := removeFromSlice(typed.AllowedModuleRolesQualifiedNames())
			if found {
				typed.SetAllowedModuleRolesQualifiedNames(updated)
				removed = true
			}
		case *genMF.Nanoflow:
			updated, found := removeFromSlice(typed.AllowedModuleRolesQualifiedNames())
			if found {
				typed.SetAllowedModuleRolesQualifiedNames(updated)
				removed = true
			}
		case *genPages.Page:
			updated, found := removeFromSlice(typed.AllowedRolesQualifiedNames())
			if found {
				typed.SetAllowedRolesQualifiedNames(updated)
				removed = true
			}
		case *genODP.PublishedODataService2:
			updated, found := removeFromSlice(typed.AllowedModuleRolesQualifiedNames())
			if found {
				typed.SetAllowedModuleRolesQualifiedNames(updated)
				removed = true
			}
		case *genREST.PublishedRestService:
			updated, found := removeFromSlice(typed.AllowedRolesQualifiedNames())
			if found {
				typed.SetAllowedRolesQualifiedNames(updated)
				removed = true
			}
		default:
			return fmt.Errorf("removeFromAllowedRoles: unsupported unit type %T", elem)
		}
		return nil
	})
	return removed, err
}

// updatePublishedRestServiceRolesViaModelsdk replaces the role list on a
// PublishedRestService unit. Kept here (rather than alongside the Phase 1
// service updates) so that all role-update entry points sit together.
func (b *MprBackend) updatePublishedRestServiceRolesViaModelsdk(unitID model.ID, roles []string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		typed, ok := elem.(*genREST.PublishedRestService)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *PublishedRestService)", elem)
		}
		typed.SetAllowedRolesQualifiedNames(roles)
		return nil
	})
}

