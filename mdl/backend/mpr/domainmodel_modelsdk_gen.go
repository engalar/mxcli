// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D8: gen→sdk bridge for the domainmodel write path.
// CreateEntityGen / UpdateEntityGen accept gen *Entity, convert to the
// sdk Entity that the existing *ViaModelsdk write helpers consume, then
// delegate. Once Stage 4 rewrites sdk/mpr to consume gen directly,
// these bridge helpers retire.

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// CreateEntityGen creates a new entity in the given domain model from a
// gen-typed *Entity. Uses the existing createEntityViaModelsdk write
// helper after a structural conversion (genEntityToSdk).
func (b *MprBackend) CreateEntityGen(domainModelID model.ID, entity *genDm.Entity) error {
	if entity == nil {
		return fmt.Errorf("CreateEntityGen: nil entity")
	}
	sdkEntity := genEntityToSdk(entity)
	return b.createEntityViaModelsdk(domainModelID, sdkEntity)
}

// UpdateEntityGen updates an entity in the given domain model from a
// gen-typed *Entity.
func (b *MprBackend) UpdateEntityGen(domainModelID model.ID, entity *genDm.Entity) error {
	if entity == nil {
		return fmt.Errorf("UpdateEntityGen: nil entity")
	}
	sdkEntity := genEntityToSdk(entity)
	return b.updateEntityViaModelsdk(domainModelID, sdkEntity)
}

// genEntityToSdk performs a shallow gen → sdk conversion preserving the
// fields *ViaModelsdk write helpers consume:
//   - Name / Documentation / ID
//   - Persistable + system attribute flags (read from *NoGeneralization)
//   - GeneralizationRef (read from *Generalization.GeneralizationQualifiedName)
//   - Attributes (Name / Documentation / ID; Type rendered as the BSON
//     storage TypeName so mpr.Serialize can reconstruct the typed
//     attribute element).
//
// View entity sources, indexes, validation rules, and event handlers
// are NOT yet bridged; their gen→sdk converters land in follow-up D8
// sub-commits as the executors that need them migrate.
func genEntityToSdk(g *genDm.Entity) *domainmodel.Entity {
	if g == nil {
		return nil
	}
	out := &domainmodel.Entity{
		Name:          g.Name(),
		Documentation: g.Documentation(),
	}
	out.ID = model.ID(g.ID())
	out.TypeName = "DomainModels$Entity"

	switch gen := g.Generalization().(type) {
	case *genDm.NoGeneralization:
		out.Persistable = gen.Persistable()
		out.HasOwner = gen.HasOwner()
		out.HasChangedBy = gen.HasChangedBy()
		out.HasChangedDate = gen.HasChangedDate()
		out.HasCreatedDate = gen.HasCreatedDate()
	case *genDm.Generalization:
		out.GeneralizationRef = gen.GeneralizationQualifiedName()
		// Persistable is inherited from the parent — leave at zero/false
		// for the sdk struct; the write helper does not re-validate.
	}

	for _, a := range g.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		sdkAttr := &domainmodel.Attribute{
			Name:          attr.Name(),
			Documentation: attr.Documentation(),
		}
		sdkAttr.ID = model.ID(attr.ID())
		sdkAttr.TypeName = "DomainModels$Attribute"
		// Bridge the attribute Type via TypeName-only stub: createEntityViaModelsdk
		// passes the sdk attribute to mpr.SerializeAttribute, which dispatches
		// on attr.Type interface. A thin stub wired with the correct gen
		// TypeName lets serialization pick the matching sdk type. Full
		// field-level conversion lands in D8.attr.
		if t := attr.Type(); t != nil {
			sdkAttr.Type = stubAttrTypeFromGen(t.TypeName())
		}
		out.Attributes = append(out.Attributes, sdkAttr)
	}
	return out
}

// stubAttrTypeFromGen returns the matching sdk AttributeType subtype for
// a given gen storage TypeName, wired with default values. Round-trip
// from gen to sdk loses parameter values (length, enum ref, etc.); D8.attr
// extends this to read the gen element's typed accessors and copy them
// onto the sdk subtype.
func stubAttrTypeFromGen(typeName string) domainmodel.AttributeType {
	switch typeName {
	case "DomainModels$StringAttributeType":
		return &domainmodel.StringAttributeType{}
	case "DomainModels$IntegerAttributeType":
		return &domainmodel.IntegerAttributeType{}
	case "DomainModels$LongAttributeType":
		return &domainmodel.LongAttributeType{}
	case "DomainModels$DecimalAttributeType":
		return &domainmodel.DecimalAttributeType{}
	case "DomainModels$BooleanAttributeType":
		return &domainmodel.BooleanAttributeType{}
	case "DomainModels$DateTimeAttributeType":
		return &domainmodel.DateTimeAttributeType{LocalizeDate: true}
	case "DomainModels$AutoNumberAttributeType":
		return &domainmodel.AutoNumberAttributeType{}
	case "DomainModels$BinaryAttributeType":
		return &domainmodel.BinaryAttributeType{}
	case "DomainModels$EnumerationAttributeType":
		return &domainmodel.EnumerationAttributeType{}
	}
	return &domainmodel.StringAttributeType{Length: 200}
}
