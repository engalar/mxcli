// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D8: gen-native domainmodel write path.
// Create/Update/Move entity and CreateAssociation now mutate the
// modelsdk/gen DomainModel directly and persist via UpdateDomainModelGen.

package mprbackend

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func (b *MprBackend) CreateEntityGen(domainModelID model.ID, entity *genDm.Entity) error {
	if entity == nil {
		return fmt.Errorf("CreateEntityGen: nil entity")
	}
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return fmt.Errorf("CreateEntityGen: load domain model: %w", err)
	}
	assignEntityIDsGen(entity)
	dm.AddEntities(entity)
	return b.UpdateDomainModelGen(dm)
}

func (b *MprBackend) UpdateEntityGen(domainModelID model.ID, entity *genDm.Entity) error {
	if entity == nil {
		return fmt.Errorf("UpdateEntityGen: nil entity")
	}
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return fmt.Errorf("UpdateEntityGen: load domain model: %w", err)
	}
	assignEntityIDsGen(entity)
	for i, candidateElem := range dm.EntitiesItems() {
		candidate, ok := candidateElem.(*genDm.Entity)
		if ok && candidate != nil && candidate.ID() == entity.ID() {
			dm.RemoveEntities(i)
			dm.AddEntities(entity)
			return b.UpdateDomainModelGen(dm)
		}
	}
	return fmt.Errorf("entity not found: %s", entity.ID())
}

func (b *MprBackend) moveEntityGen(sourceDMID, targetDMID model.ID, sourceModuleName, targetModuleName string, entity *genDm.Entity) ([]string, error) {
	if entity == nil {
		return nil, fmt.Errorf("MoveEntityGen: nil entity")
	}
	sourceDM, err := b.GetDomainModelByIDGen(sourceDMID)
	if err != nil {
		return nil, fmt.Errorf("MoveEntityGen: load source domain model: %w", err)
	}
	targetDM, err := b.GetDomainModelByIDGen(targetDMID)
	if err != nil {
		return nil, fmt.Errorf("MoveEntityGen: load target domain model: %w", err)
	}

	entityID := model.ID(entity.ID())
	var moved *genDm.Entity
	for i, candidateElem := range sourceDM.EntitiesItems() {
		candidate, ok := candidateElem.(*genDm.Entity)
		if ok && candidate != nil && model.ID(candidate.ID()) == entityID {
			moved = candidate
			sourceDM.RemoveEntities(i)
			break
		}
	}
	if moved == nil {
		return nil, fmt.Errorf("MoveEntityGen: entity %s not found in source domain model %s", entity.ID(), sourceDMID)
	}

	var convertedAssocs []string
	for i := len(sourceDM.AssociationsItems()) - 1; i >= 0; i-- {
		assoc, ok := sourceDM.AssociationsItems()[i].(*genDm.Association)
		if !ok || assoc == nil {
			continue
		}
		switch {
		case model.ID(assoc.ChildRefID()) == entityID:
			sourceDM.RemoveAssociations(i)
			sourceDM.AddCrossAssociations(crossAssociationFromAssociationGen(
				assoc,
				assoc.ParentRefID(),
				targetModuleName+"."+moved.Name(),
			))
			convertedAssocs = append(convertedAssocs, assoc.Name())
		case model.ID(assoc.ParentRefID()) == entityID:
			childEntityName := findEntityNameInDomainModelGen(sourceDM, model.ID(assoc.ChildRefID()))
			sourceDM.RemoveAssociations(i)
			targetDM.AddCrossAssociations(crossAssociationFromAssociationGen(
				assoc,
				element.ID(entityID),
				sourceModuleName+"."+childEntityName,
			))
			convertedAssocs = append(convertedAssocs, assoc.Name())
		}
	}

	rewriteEntityQualifiedNamesForMoveGen(moved, sourceModuleName, targetModuleName)
	targetDM.AddEntities(moved)

	if err := b.UpdateDomainModelGen(sourceDM); err != nil {
		return nil, fmt.Errorf("update source domain model: %w", err)
	}
	if err := b.UpdateDomainModelGen(targetDM); err != nil {
		return nil, fmt.Errorf("update target domain model: %w", err)
	}
	return convertedAssocs, nil
}

func (b *MprBackend) CreateAssociationGen(domainModelID model.ID, assoc *genDm.Association) error {
	if assoc == nil {
		return fmt.Errorf("CreateAssociationGen: nil association")
	}
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return fmt.Errorf("CreateAssociationGen: load domain model: %w", err)
	}
	assignElementIDGen(assoc)
	assignElementIDGen(assoc.DeleteBehavior())
	assignElementIDGen(assoc.Source())
	assignElementIDGen(assoc.Capabilities())
	dm.AddAssociations(assoc)
	return b.UpdateDomainModelGen(dm)
}

func assignEntityIDsGen(entity *genDm.Entity) {
	if entity == nil {
		return
	}
	assignElementIDGen(entity)
	assignElementIDGen(entity.Generalization())
	assignElementIDGen(entity.Source())
	for _, attrElem := range entity.AttributesItems() {
		attr, ok := attrElem.(*genDm.Attribute)
		if !ok || attr == nil {
			continue
		}
		assignElementIDGen(attr)
		assignElementIDGen(attr.Type())
		assignElementIDGen(attr.Value())
	}
	for _, vrElem := range entity.ValidationRulesItems() {
		vr, ok := vrElem.(*genDm.ValidationRule)
		if !ok || vr == nil {
			continue
		}
		assignElementIDGen(vr)
		assignElementIDGen(vr.ErrorMessage())
		assignElementIDGen(vr.RuleInfo())
	}
	for _, ehElem := range entity.EventHandlersItems() {
		assignElementIDGen(ehElem)
	}
	for _, idxElem := range entity.IndexesItems() {
		idx, ok := idxElem.(*genDm.Index)
		if !ok || idx == nil {
			continue
		}
		assignElementIDGen(idx)
		for _, attrElem := range idx.AttributesItems() {
			assignElementIDGen(attrElem)
		}
	}
}

func assignElementIDGen(elem any) {
	idElem, ok := elem.(interface {
		ID() element.ID
		SetID(element.ID)
	})
	if !ok || idElem == nil || idElem.ID() != "" {
		return
	}
	idElem.SetID(element.ID(modelsdkmpr.GenerateID()))
}

func crossAssociationFromAssociationGen(assoc *genDm.Association, parentID element.ID, childQualifiedName string) *genDm.CrossAssociation {
	out := genDm.NewCrossAssociation()
	if assoc != nil {
		if assoc.ID() != "" {
			out.SetID(assoc.ID())
		}
		out.SetName(assoc.Name())
		out.SetDocumentation(assoc.Documentation())
		out.SetType(assoc.Type())
		out.SetOwner(assoc.Owner())
		out.SetStorageFormat(assoc.StorageFormat())
		out.SetDataStorageGuid(assoc.DataStorageGuid())
		out.SetDeleteBehavior(assoc.DeleteBehavior())
		out.SetRemoteSourceDocumentQualifiedName(assoc.RemoteSourceDocumentQualifiedName())
		out.SetSource(assoc.Source())
		out.SetCapabilities(assoc.Capabilities())
		out.SetExportLevel(assoc.ExportLevel())
	}
	if out.ID() == "" {
		assignElementIDGen(out)
	}
	out.SetParentID(parentID)
	out.SetChildQualifiedName(childQualifiedName)
	return out
}

func findEntityNameInDomainModelGen(dm *genDm.DomainModel, entityID model.ID) string {
	if dm == nil || entityID == "" {
		return ""
	}
	for _, entityElem := range dm.EntitiesItems() {
		entity, ok := entityElem.(*genDm.Entity)
		if ok && entity != nil && model.ID(entity.ID()) == entityID {
			return entity.Name()
		}
	}
	return ""
}

func rewriteEntityQualifiedNamesForMoveGen(entity *genDm.Entity, sourceModuleName, targetModuleName string) {
	if entity == nil || sourceModuleName == "" || targetModuleName == "" || sourceModuleName == targetModuleName {
		return
	}
	oldPrefix := sourceModuleName + "."
	newPrefix := targetModuleName + "."

	for _, vrElem := range entity.ValidationRulesItems() {
		vr, ok := vrElem.(*genDm.ValidationRule)
		if ok && vr != nil {
			vr.SetAttributeQualifiedName(rewriteQualifiedPrefixGen(vr.AttributeQualifiedName(), oldPrefix, newPrefix))
		}
	}
	if src, ok := entity.Source().(*genDm.OqlViewEntitySource); ok && src != nil {
		src.SetSourceDocumentQualifiedName(rewriteQualifiedPrefixGen(src.SourceDocumentQualifiedName(), oldPrefix, newPrefix))
	}
}

func rewriteQualifiedPrefixGen(qn, oldPrefix, newPrefix string) string {
	if strings.HasPrefix(qn, oldPrefix) {
		return newPrefix + qn[len(oldPrefix):]
	}
	return qn
}
