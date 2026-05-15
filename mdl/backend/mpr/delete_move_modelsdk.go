// SPDX-License-Identifier: Apache-2.0
package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// ── Domain Model ──────────────────────────────────────
//
// Entity、Attribute、Association 是 DomainModel BSON 内的嵌入对象，
// 不是 Unit 表的独立行。删除操作必须通过读取-修改-回写整个
// DomainModel 单元，不能直接调用 msdkWriter.DeleteUnit(entityID)。

func (b *MprBackend) deleteEntityViaModelsdk(domainModelID, entityID model.ID) error {
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return fmt.Errorf("read domain model: %w", err)
	}
	for i, e := range dm.EntitiesItems() {
		entity, ok := e.(*genDm.Entity)
		if ok && model.ID(entity.ID()) == entityID {
			dm.RemoveEntities(i)
			// Keep same semantics as the legacy path: clean same-DM
			// associations that reference the deleted entity.
			for j := len(dm.AssociationsItems()) - 1; j >= 0; j-- {
				assoc, ok := dm.AssociationsItems()[j].(*genDm.Association)
				if ok && (model.ID(assoc.ParentRefID()) == entityID || model.ID(assoc.ChildRefID()) == entityID) {
					dm.RemoveAssociations(j)
				}
			}
			if err := b.UpdateDomainModelGen(dm); err != nil {
				return fmt.Errorf("update domain model: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("entity not found: %s", entityID)
}

func (b *MprBackend) deleteAttributeViaModelsdk(domainModelID, entityID, attrID model.ID) error {
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return fmt.Errorf("read domain model: %w", err)
	}
	for _, e := range dm.EntitiesItems() {
		entity, ok := e.(*genDm.Entity)
		if !ok || model.ID(entity.ID()) != entityID {
			continue
		}
		for i, a := range entity.AttributesItems() {
			attr, ok := a.(*genDm.Attribute)
			if ok && model.ID(attr.ID()) == attrID {
				entity.RemoveAttributes(i)
				if err := b.UpdateDomainModelGen(dm); err != nil {
					return fmt.Errorf("update domain model: %w", err)
				}
				return nil
			}
		}
		return fmt.Errorf("attribute not found: %s", attrID)
	}
	return fmt.Errorf("entity not found: %s", entityID)
}

func (b *MprBackend) deleteAssociationViaModelsdk(domainModelID, assocID model.ID) error {
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return fmt.Errorf("read domain model: %w", err)
	}
	for i, a := range dm.AssociationsItems() {
		assoc, ok := a.(*genDm.Association)
		if ok && model.ID(assoc.ID()) == assocID {
			dm.RemoveAssociations(i)
			if err := b.UpdateDomainModelGen(dm); err != nil {
				return fmt.Errorf("update domain model: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("association not found: %s", assocID)
}

func (b *MprBackend) deleteCrossAssociationViaModelsdk(domainModelID, assocID model.ID) error {
	dm, err := b.GetDomainModelByIDGen(domainModelID)
	if err != nil {
		return fmt.Errorf("read domain model: %w", err)
	}
	for i, ca := range dm.CrossAssociationsItems() {
		crossAssoc, ok := ca.(*genDm.CrossAssociation)
		if ok && model.ID(crossAssoc.ID()) == assocID {
			dm.RemoveCrossAssociations(i)
			if err := b.UpdateDomainModelGen(dm); err != nil {
				return fmt.Errorf("update domain model: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("cross-module association not found: %s", assocID)
}

// ── Microflow / Nanoflow ───────────────────────────────
//
// Move helpers were retired in Followup E6 — production routes
// through ctx.Microflows.Move / ctx.Nanoflows.Move (modelsdk repos).
// Delete helpers stay because deleteMicroflowViaRepoOrBackend /
// deleteNanoflowViaRepoOrBackend in mdl/executor still falls back to
// ctx.Backend.DeleteMicroflow / DeleteNanoflow when the repos are
// not wired in mock-only test contexts.

func (b *MprBackend) deleteMicroflowViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) deleteNanoflowViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

// ── Pages / Layouts / Snippets ────────────────────────
//
// Stage 3.3.5.E1 retired the deletePageViaModelsdk / deleteLayoutViaModelsdk /
// deleteSnippetViaModelsdk helpers along with the legacy DeletePage /
// DeleteLayout / DeleteSnippet PageBackend methods. The Gen-typed
// DeletePageGen / DeleteLayoutGen / DeleteSnippetGen methods in
// backend.go call msdkWriter.DeleteUnit directly.

// ── Workflows ────────────────────────────────────────

func (b *MprBackend) deleteWorkflowViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

// ── Image Collections (id is string) ──────────────────

func (b *MprBackend) deleteImageCollectionViaModelsdk(id string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(id)
}

// ── Agent Editor (ids are strings) ────────────────────

func (b *MprBackend) deleteAgentEditorModelViaModelsdk(id string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(id)
}

func (b *MprBackend) deleteAgentEditorKnowledgeBaseViaModelsdk(id string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(id)
}

func (b *MprBackend) deleteAgentEditorConsumedMCPServiceViaModelsdk(id string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(id)
}

func (b *MprBackend) deleteAgentEditorAgentViaModelsdk(id string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(id)
}

// ── Java Actions ──────────────────────────────────────

func (b *MprBackend) deleteJavaActionViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}
