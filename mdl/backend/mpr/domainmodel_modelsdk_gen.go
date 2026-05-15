// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D8: gen→sdk bridge for the domainmodel write path.
// CreateEntityGen / UpdateEntityGen accept gen *Entity, convert to the
// sdk Entity that the existing *ViaModelsdk write helpers consume, then
// delegate. Once Stage 4 rewrites sdk/mpr to consume gen directly,
// these bridge helpers retire.

package mprbackend

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genRest "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
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

// moveEntityGen bridges a gen-typed entity move onto the existing
// sdk-backed moveEntityViaModelsdk implementation. To avoid a lossy
// gen→sdk structural conversion for the full entity graph, it reloads
// the sdk entity from the source DomainModel by UUID and delegates the
// actual move to the legacy writer.
func (b *MprBackend) moveEntityGen(sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string, entity *genDm.Entity) ([]string, error) {
	if entity == nil {
		return nil, fmt.Errorf("MoveEntityGen: nil entity")
	}
	sourceDM, err := b.reader.GetDomainModelByID(sourceDMID)
	if err != nil {
		return nil, fmt.Errorf("MoveEntityGen: load source domain model: %w", err)
	}
	var sdkEntity *domainmodel.Entity
	for _, candidate := range sourceDM.Entities {
		if candidate != nil && candidate.ID == model.ID(entity.ID()) {
			sdkEntity = candidate
			break
		}
	}
	if sdkEntity == nil {
		return nil, fmt.Errorf("MoveEntityGen: entity %s not found in source domain model %s", entity.ID(), sourceDMID)
	}
	return b.moveEntityViaModelsdk(sdkEntity, sourceDMID, targetDMID, sourceModuleName, targetModuleName)
}

// genEntityToSdk performs a gen → sdk conversion preserving the entity
// structure the existing sdk-backed write helpers consume:
//   - base entity fields / flags / location / source metadata
//   - attributes including typed metadata and value kinds
//   - validation rules, indexes, and event handlers
//
// This keeps the current write path compatible while executors migrate
// off sdk/domainmodel one operation at a time.
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
	if x, y, ok := parseLocationXY(g.Location()); ok {
		out.Location = model.Point{X: x, Y: y}
	}
	if src := g.Source(); src != nil {
		out.SourceObjectID = model.ID(src.ID())
	}

	attrNameToID := make(map[string]model.ID)
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
		if t := attr.Type(); t != nil {
			sdkAttr.Type = sdkAttrTypeFromGen(t)
		}
		if v := attr.Value(); v != nil {
			sdkAttr.Value = sdkAttrValueFromGen(v)
		}
		switch v := attr.Value().(type) {
		case *genRest.ODataMappedValue:
			sdkAttr.RemoteName = v.RemoteName()
			sdkAttr.RemoteType = v.RemoteType()
			sdkAttr.Filterable = v.Filterable()
			sdkAttr.Sortable = v.Sortable()
			sdkAttr.Creatable = v.Creatable()
			sdkAttr.Updatable = v.Updatable()
		case *genRest.ODataMappedPrimitiveCollectionValue:
			sdkAttr.RemoteName = v.RemoteName()
			sdkAttr.RemoteType = v.RemoteType()
			sdkAttr.IsPrimitiveCollection = true
		}
		out.Attributes = append(out.Attributes, sdkAttr)
		attrNameToID[attr.Name()] = sdkAttr.ID
	}
	for _, vrElem := range g.ValidationRulesItems() {
		vr, ok := vrElem.(*genDm.ValidationRule)
		if !ok {
			continue
		}
		sdkRule := sdkValidationRuleFromGen(vr, attrNameToID)
		if sdkRule != nil {
			out.ValidationRules = append(out.ValidationRules, sdkRule)
		}
	}
	for _, idxElem := range g.IndexesItems() {
		idx, ok := idxElem.(*genDm.Index)
		if !ok {
			continue
		}
		sdkIndex := sdkIndexFromGen(idx)
		if sdkIndex != nil {
			out.Indexes = append(out.Indexes, sdkIndex)
		}
	}
	for _, ehElem := range g.EventHandlersItems() {
		eh, ok := ehElem.(*genDm.EventHandler)
		if !ok {
			continue
		}
		sdkEH := sdkEventHandlerFromGen(eh)
		if sdkEH != nil {
			out.EventHandlers = append(out.EventHandlers, sdkEH)
		}
	}
	switch src := g.Source().(type) {
	case *genRest.ODataRemoteEntitySource:
		out.Source = src.TypeName()
		out.RemoteServiceName = src.SourceDocumentQualifiedName()
		out.RemoteEntitySet = src.EntitySet()
		out.RemoteEntityName = src.RemoteName()
		out.Countable = src.Countable()
		out.Creatable = src.Creatable()
		out.Deletable = src.Deletable()
		out.SkipSupported = src.SkipSupported()
		out.TopSupported = src.TopSupported()
		out.CreateChangeLocally = src.CreateChangeLocally()
		if key, ok := src.Key().(*genRest.ODataKey); ok && key != nil {
			for _, partElem := range key.PartsItems() {
				part, ok := partElem.(*genRest.ODataKeyPart)
				if !ok {
					continue
				}
				out.RemoteKeyParts = append(out.RemoteKeyParts, &domainmodel.RemoteKeyPart{
					Name:       part.EntityKeyPartName(),
					RemoteName: part.Name(),
					RemoteType: part.RemoteType(),
					Type:       stubAttrTypeFromGen(typeNameForKeyPart(part.Type())),
				})
			}
		}
	case *genRest.ODataEntityTypeSource:
		out.Source = src.TypeName()
		out.RemoteServiceName = src.SourceDocumentQualifiedName()
		out.RemoteEntityName = src.EntityTypeName()
		out.IsOpen = src.IsOpen()
		if key, ok := src.Key().(*genRest.ODataKey); ok && key != nil {
			for _, partElem := range key.PartsItems() {
				part, ok := partElem.(*genRest.ODataKeyPart)
				if !ok {
					continue
				}
				out.RemoteKeyParts = append(out.RemoteKeyParts, &domainmodel.RemoteKeyPart{
					Name:       part.EntityKeyPartName(),
					RemoteName: part.Name(),
					RemoteType: part.RemoteType(),
					Type:       stubAttrTypeFromGen(typeNameForKeyPart(part.Type())),
				})
			}
		}
	case *genRest.ODataPrimitiveCollectionEntitySource:
		out.Source = src.TypeName()
		out.RemoteServiceName = src.SourceDocumentQualifiedName()
	case *genDm.OqlViewEntitySource:
		out.Source = src.TypeName()
		out.SourceDocumentRef = src.SourceDocumentQualifiedName()
		out.OqlQuery = src.Oql()
	}
	return out
}

func sdkAttrTypeFromGen(t interface{ TypeName() string }) domainmodel.AttributeType {
	if t == nil {
		return &domainmodel.StringAttributeType{Length: 200}
	}
	switch typed := t.(type) {
	case *genDm.StringAttributeType:
		return &domainmodel.StringAttributeType{Length: int(typed.Length())}
	case *genDm.IntegerAttributeType:
		return &domainmodel.IntegerAttributeType{}
	case *genDm.LongAttributeType:
		return &domainmodel.LongAttributeType{}
	case *genDm.DecimalAttributeType:
		return &domainmodel.DecimalAttributeType{}
	case *genDm.BooleanAttributeType:
		return &domainmodel.BooleanAttributeType{}
	case *genDm.DateTimeAttributeType:
		return &domainmodel.DateTimeAttributeType{LocalizeDate: typed.LocalizeDate()}
	case *genDm.AutoNumberAttributeType:
		return &domainmodel.AutoNumberAttributeType{}
	case *genDm.BinaryAttributeType:
		return &domainmodel.BinaryAttributeType{}
	case *genDm.EnumerationAttributeType:
		return &domainmodel.EnumerationAttributeType{
			EnumerationRef: typed.EnumerationQualifiedName(),
		}
	default:
		return stubAttrTypeFromGen(t.TypeName())
	}
}

func sdkAttrValueFromGen(v any) *domainmodel.AttributeValue {
	switch typed := v.(type) {
	case *genDm.StoredValue:
		out := &domainmodel.AttributeValue{
			DefaultValue: typed.DefaultValue(),
		}
		out.ID = model.ID(typed.ID())
		return out
	case *genDm.CalculatedValue:
		out := &domainmodel.AttributeValue{
			Type:          "CalculatedValue",
			MicroflowName: typed.MicroflowQualifiedName(),
		}
		out.ID = model.ID(typed.ID())
		return out
	case *genDm.OqlViewValue:
		out := &domainmodel.AttributeValue{
			ViewReference: typed.Reference(),
		}
		out.ID = model.ID(typed.ID())
		return out
	case *genRest.ODataMappedValue:
		out := &domainmodel.AttributeValue{
			DefaultValue: typed.DefaultValueDesignTime(),
		}
		out.ID = model.ID(typed.ID())
		return out
	case *genRest.ODataMappedPrimitiveCollectionValue:
		out := &domainmodel.AttributeValue{
			DefaultValue: typed.DefaultValueDesignTime(),
		}
		out.ID = model.ID(typed.ID())
		return out
	default:
		return nil
	}
}

func sdkValidationRuleFromGen(vr *genDm.ValidationRule, attrNameToID map[string]model.ID) *domainmodel.ValidationRule {
	if vr == nil {
		return nil
	}
	attrName := lastQualifiedSegment(vr.AttributeQualifiedName())
	attrID := attrNameToID[attrName]
	if attrID == "" {
		return nil
	}
	out := &domainmodel.ValidationRule{
		AttributeID: attrID,
	}
	out.ID = model.ID(vr.ID())
	switch vr.RuleInfo().(type) {
	case *genDm.RequiredRuleInfo:
		out.Type = "Required"
		out.Rule = &domainmodel.RequiredValidationRuleInfo{}
	case *genDm.UniqueRuleInfo:
		out.Type = "Unique"
		out.Rule = &domainmodel.UniqueValidationRuleInfo{}
	default:
		return nil
	}
	return out
}

func sdkIndexFromGen(idx *genDm.Index) *domainmodel.Index {
	if idx == nil {
		return nil
	}
	out := &domainmodel.Index{}
	out.ID = model.ID(idx.ID())
	for _, attrElem := range idx.AttributesItems() {
		ia, ok := attrElem.(*genDm.IndexedAttribute)
		if !ok {
			continue
		}
		sdkIA := &domainmodel.IndexAttribute{
			AttributeID: model.ID(ia.AttributeRefID()),
			Ascending:   ia.Ascending(),
		}
		sdkIA.ID = model.ID(ia.ID())
		out.Attributes = append(out.Attributes, sdkIA)
	}
	if len(out.Attributes) == 0 {
		return nil
	}
	return out
}

func sdkEventHandlerFromGen(eh *genDm.EventHandler) *domainmodel.EventHandler {
	if eh == nil {
		return nil
	}
	out := &domainmodel.EventHandler{
		Moment:            domainmodel.EventMoment(eh.Moment()),
		Event:             domainmodel.EventType(eh.Event()),
		MicroflowName:     eh.MicroflowQualifiedName(),
		RaiseErrorOnFalse: eh.RaiseErrorOnFalse(),
		PassEventObject:   eh.PassEventObject(),
	}
	out.ID = model.ID(eh.ID())
	return out
}

func lastQualifiedSegment(qn string) string {
	if qn == "" {
		return ""
	}
	if i := strings.LastIndex(qn, "."); i >= 0 && i < len(qn)-1 {
		return qn[i+1:]
	}
	return qn
}

func typeNameForKeyPart(elem any) string {
	typeNamer, ok := elem.(interface{ TypeName() string })
	if !ok || typeNamer == nil {
		return ""
	}
	typeName := typeNamer.TypeName()
	if strings.HasPrefix(typeName, "Rest$") {
		typeName = strings.TrimPrefix(typeName, "Rest$")
	}
	return typeName
}

func parseLocationXY(loc string) (int, int, bool) {
	if loc == "" {
		return 0, 0, false
	}
	parts := strings.Split(loc, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	x, errX := strconv.Atoi(strings.TrimSpace(parts[0]))
	y, errY := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errX != nil || errY != nil {
		return 0, 0, false
	}
	return x, y, true
}

// CreateAssociationGen creates a new association in the given domain
// model from a gen-typed *Association. Bridges to the existing
// createAssociationViaModelsdk write helper.
func (b *MprBackend) CreateAssociationGen(domainModelID model.ID, assoc *genDm.Association) error {
	if assoc == nil {
		return fmt.Errorf("CreateAssociationGen: nil association")
	}
	sdkAssoc := genAssociationToSdk(assoc)
	return b.createAssociationViaModelsdk(domainModelID, sdkAssoc)
}

// genAssociationToSdk performs a shallow gen → sdk conversion preserving
// the fields the *ViaModelsdk write helper consumes:
//   - Name / ID / Documentation
//   - ParentID (FROM entity, FK owner per CLAUDE.md "Association Parent/Child Pointer Semantics")
//   - ChildID (TO entity)
//   - Type / Owner / StorageFormat strings → sdk enums via the small
//     converter functions inline below.
func genAssociationToSdk(g *genDm.Association) *domainmodel.Association {
	if g == nil {
		return nil
	}
	out := &domainmodel.Association{
		Name:          g.Name(),
		Documentation: g.Documentation(),
	}
	out.ID = model.ID(g.ID())
	out.TypeName = "DomainModels$Association"
	out.ParentID = model.ID(g.ParentRefID())
	out.ChildID = model.ID(g.ChildRefID())

	switch g.Type() {
	case "ReferenceSet":
		out.Type = domainmodel.AssociationTypeReferenceSet
	default:
		out.Type = domainmodel.AssociationTypeReference
	}
	switch g.Owner() {
	case "Both":
		out.Owner = domainmodel.AssociationOwnerBoth
	default:
		out.Owner = domainmodel.AssociationOwnerDefault
	}
	switch g.StorageFormat() {
	case "Column":
		out.StorageFormat = domainmodel.StorageFormatColumn
	default:
		out.StorageFormat = domainmodel.StorageFormatTable
	}
	switch src := g.Source().(type) {
	case *genRest.ODataRemoteAssociationSource:
		out.Source = src.TypeName()
		out.RemoteParentNavigationProperty = src.RemoteParentNavigationProperty()
		out.RemoteChildNavigationProperty = src.RemoteChildNavigationProperty()
		out.CreatableFromParent = src.CreatableFromParent()
		out.CreatableFromChild = src.CreatableFromChild()
		out.UpdatableFromParent = src.UpdatableFromParent()
		out.UpdatableFromChild = src.UpdatableFromChild()
		out.Navigability2 = src.Navigability2()
	case *genRest.ODataPrimitiveCollectionAssociationSource:
		out.Source = src.TypeName()
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
