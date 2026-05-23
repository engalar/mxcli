// SPDX-License-Identifier: Apache-2.0

// Package mpr — built-in System module domain model.
//
// Mendix's System module (entities, associations, enumerations) is baked into
// the Mendix runtime and not stored in the MPR file. Studio Pro knows about it
// through its own schema.
//
// We reconstruct a virtual genDm.DomainModel from the meta.SystemEntities /
// meta.SystemAssociations / meta.SystemEnumerations definitions, using
// deterministic IDs so that all backend lookups (resolveEntity, lookupEnumRef,
// association resolution) can find System elements without caller-side hacks.
//
// System enumerations are also registered in ListEnumerations so that
// expression validators can resolve System.WorkflowState etc.

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/modelsdk/meta"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// systemDomainModelCache is computed once and reused (it never changes).
var systemDomainModelCache *genDm.DomainModel

// systemEntityIDMap maps entity name → deterministic BSON ID (used for
// association parent/child pointer resolution).
var systemEntityIDMap map[string]element.ID

// builtinSystemDomainModel returns the virtual System module domain model
// built from meta.SystemEntities and meta.SystemAssociations.
// The result is cached after the first call.
func builtinSystemDomainModel() *genDm.DomainModel {
	if systemDomainModelCache != nil {
		return systemDomainModelCache
	}
	dm := genDm.NewDomainModel()
	dm.SetID(element.ID(meta.SystemDomainModelID))

	entityIDMap := make(map[string]element.ID, len(meta.SystemEntities))

	// --- Entities ---
	for i, def := range meta.SystemEntities {
		e := genDm.NewEntity()
		// Deterministic ID based on index (stable across runs).
		eid := element.ID(fmt.Sprintf("00000000-0000-0001-%04x-000000000000", i+1))
		e.SetID(eid)
		e.SetName(def.Name)
		entityIDMap[def.Name] = eid

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

	// --- Associations ---
	for i, def := range meta.SystemAssociations {
		assoc := genDm.NewAssociation()
		assoc.SetID(element.ID(fmt.Sprintf("00000000-0000-0003-%04x-000000000000", i+1)))
		assoc.SetName(def.Name)

		parentID := entityIDMap[def.Parent]
		childID := entityIDMap[def.Child]
		if parentID != "" {
			assoc.SetParentID(parentID)
		}
		if childID != "" {
			assoc.SetChildID(childID)
		}
		switch def.Type {
		case "ReferenceSet":
			assoc.SetType(string(genDm.AssociationTypeReferenceSet))
		default:
			assoc.SetType(string(genDm.AssociationTypeReference))
		}
		switch def.Owner {
		case "Both":
			assoc.SetOwner(string(genDm.AssociationOwnerBoth))
		default:
			assoc.SetOwner(string(genDm.AssociationOwnerDefault))
		}
		dm.AddAssociations(assoc)
	}

	systemDomainModelCache = dm
	systemEntityIDMap = entityIDMap
	return dm
}

// builtinSystemEnumerations returns model.Enumeration objects for all System
// enumerations defined in meta.SystemEnumerations. Used by ListEnumerations
// so expression validators can resolve System.WorkflowState etc.
func builtinSystemEnumerations() []*model.Enumeration {
	enums := make([]*model.Enumeration, 0, len(meta.SystemEnumerations))
	for i, def := range meta.SystemEnumerations {
		enum := &model.Enumeration{
			BaseElement: model.BaseElement{
				ID:       model.ID(fmt.Sprintf("00000000-0000-0004-%04x-000000000000", i+1)),
				TypeName: "Enumerations$Enumeration",
			},
			ContainerID: model.ID(meta.SystemModuleID),
			Name:        def.Name[len("System."):], // strip "System." prefix → bare name
		}
		for _, v := range def.Values {
			enum.Values = append(enum.Values, model.EnumerationValue{Name: v})
		}
		enums = append(enums, enum)
	}
	return enums
}
