// SPDX-License-Identifier: Apache-2.0

// Package mpr — built-in System module domain model.
//
// Mendix's System module entities (System.WorkflowUserTask, System.User, etc.)
// are not stored in the MPR file. Instead they are baked into the Mendix
// runtime and Studio Pro knows about them through its own schema.
//
// We reconstruct a virtual genDm.DomainModel from the meta.SystemEntities /
// meta.SystemAssociations definitions, using deterministic IDs derived from
// entity names. This allows all backend lookups (resolveEntity, lookupEnum…)
// to find System entities without special-casing the caller.

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/modelsdk/meta"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// systemDomainModelCache is computed once and reused (it never changes).
var systemDomainModelCache *genDm.DomainModel

// builtinSystemDomainModel returns the virtual System module domain model
// built from meta.SystemEntities and meta.SystemAssociations.
// The result is cached after the first call.
func builtinSystemDomainModel() *genDm.DomainModel {
	if systemDomainModelCache != nil {
		return systemDomainModelCache
	}
	dm := genDm.NewDomainModel()
	dm.SetID(element.ID(meta.SystemDomainModelID))

	for i, def := range meta.SystemEntities {
		e := genDm.NewEntity()
		// Deterministic ID: 00000000-0000-0001-<entity_index_hex>
		e.SetID(element.ID(fmt.Sprintf("00000000-0000-0001-%04x-000000000000", i+1)))
		e.SetName(def.Name)

		for j, attrDef := range def.Attributes {
			a := genDm.NewAttribute()
			a.SetID(element.ID(fmt.Sprintf("00000000-0000-0002-%04x-%012x", i+1, j+1)))
			a.SetName(attrDef.Name)
			switch attrDef.Type {
			case "String", "HashedString":
				t := genDm.NewStringAttributeType()
				t.SetID(element.ID(mmpr.GenerateID()))
				a.SetType(t)
			case "Integer":
				t := genDm.NewIntegerAttributeType()
				t.SetID(element.ID(mmpr.GenerateID()))
				a.SetType(t)
			case "Long", "AutoNumber":
				t := genDm.NewLongAttributeType()
				t.SetID(element.ID(mmpr.GenerateID()))
				a.SetType(t)
			case "Decimal":
				t := genDm.NewDecimalAttributeType()
				t.SetID(element.ID(mmpr.GenerateID()))
				a.SetType(t)
			case "Boolean":
				t := genDm.NewBooleanAttributeType()
				t.SetID(element.ID(mmpr.GenerateID()))
				a.SetType(t)
			case "DateTime":
				t := genDm.NewDateTimeAttributeType()
				t.SetID(element.ID(mmpr.GenerateID()))
				a.SetType(t)
			case "Binary":
				t := genDm.NewBinaryAttributeType()
				t.SetID(element.ID(mmpr.GenerateID()))
				a.SetType(t)
			case "Enumeration":
				t := genDm.NewEnumerationAttributeType()
				t.SetID(element.ID(mmpr.GenerateID()))
				t.SetEnumerationQualifiedName(attrDef.EnumQN)
				a.SetType(t)
			}
			e.AddAttributes(a)
		}
		dm.AddEntities(e)
	}
	systemDomainModelCache = dm
	return dm
}
