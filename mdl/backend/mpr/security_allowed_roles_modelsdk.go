// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMF "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genODP "github.com/mendixlabs/mxcli/modelsdk/gen/odatapublish"
	genPages "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genREST "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
	"go.mongodb.org/mongo-driver/bson"
)

// updateAllowedRolesViaModelsdk replaces the role list on a Microflow,
// Nanoflow, Page, or PublishedODataService unit via the gen type setters.
//
// CRITICAL: Microflow, Nanoflow, and PublishedODataService store roles under
// the BSON key "AllowedModuleRoles". Page stores roles under "AllowedRoles"
// (newer format) AND "AllowedModuleRoles" (older format).
//
// Mendix Studio Pro uses "AllowedModuleRoles" (BSON version int32(1)) to
// determine page access for CE0557 checks (navigation/button/URL usage).
// The gen type setter writes "AllowedRoles" (version 3) — a newer field
// used by Mendix for visibility rules but NOT by the CE0557 validator.
// To avoid false CE0557 errors, we write BOTH fields for pages.
func (b *MprBackend) updateAllowedRolesViaModelsdk(unitID model.ID, roles []string) error {
	return b.msdkWritePage(unitID, roles, func(elem element.Element) error {
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

// msdkWritePage is like msdkWrite but for page units: after encoding the gen
// type (which writes "AllowedRoles"), it also injects "AllowedModuleRoles"
// (version int32(1)) into the raw BSON. This dual-write matches what Studio Pro
// stores and is required for Mendix's CE0557 validator to recognize page access.
func (b *MprBackend) msdkWritePage(unitID model.ID, roles []string, mutateFn func(elem element.Element) error) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return fmt.Errorf("read unit: %w", err)
	}
	elem, err := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(rawBytes))
	if err != nil {
		return fmt.Errorf("decode unit: %w", err)
	}

	// Only apply dual-write for page units.
	_, isPage := elem.(*genPages.Page)
	if err := mutateFn(elem); err != nil {
		return err
	}
	newBytes, err := (&codec.Encoder{}).Encode(elem)
	if err != nil {
		return fmt.Errorf("encode unit: %w", err)
	}

	// For pages: also inject AllowedModuleRoles (BSON version int32(1)) to satisfy
	// Mendix's CE0557 page-access validator. Studio Pro writes this field alongside
	// AllowedRoles; without it, navigation/button-referenced pages get CE0557.
	if isPage && len(roles) > 0 {
		rolesArr := bson.A{int32(1)}
		for _, r := range roles {
			rolesArr = append(rolesArr, r)
		}
		var doc bson.D
		if err := bson.Unmarshal(newBytes, &doc); err != nil {
			return fmt.Errorf("unmarshal for AllowedModuleRoles patch: %w", err)
		}
		found := false
		for i, e := range doc {
			if e.Key == "AllowedModuleRoles" {
				doc[i].Value = rolesArr
				found = true
				break
			}
		}
		if !found {
			doc = append(doc, bson.E{Key: "AllowedModuleRoles", Value: rolesArr})
		}
		newBytes, err = bson.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal AllowedModuleRoles patch: %w", err)
		}
	}

	if err := b.msdkWriter.UpdateRawUnit(string(unitID), newBytes); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}
	return nil
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

