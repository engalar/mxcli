// SPDX-License-Identifier: Apache-2.0
package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// ── Domain Model ──────────────────────────────────────
//
// Entity、Attribute、Association 是 DomainModel BSON 内的嵌入对象，
// 不是 Unit 表的独立行。删除操作必须通过 writeDomainModel 读取-修改-
// 回写整个 DomainModel 单元，不能直接调用 msdkWriter.DeleteUnit(entityID)。

func (b *MprBackend) deleteEntityViaModelsdk(domainModelID, entityID model.ID) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		for i, e := range dm.Entities {
			if e.ID == entityID {
				dm.Entities = append(dm.Entities[:i], dm.Entities[i+1:]...)
				// 同步清理同 DM 内引用该实体的关联（防止悬空指针）
				var kept []*domainmodel.Association
				for _, a := range dm.Associations {
					if a.ParentID != entityID && a.ChildID != entityID {
						kept = append(kept, a)
					}
				}
				dm.Associations = kept
				return nil
			}
		}
		return fmt.Errorf("entity not found: %s", entityID)
	})
}

func (b *MprBackend) deleteAttributeViaModelsdk(domainModelID, entityID, attrID model.ID) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		for _, e := range dm.Entities {
			if e.ID == entityID {
				for i, a := range e.Attributes {
					if a.ID == attrID {
						e.Attributes = append(e.Attributes[:i], e.Attributes[i+1:]...)
						return nil
					}
				}
				return fmt.Errorf("attribute not found: %s", attrID)
			}
		}
		return fmt.Errorf("entity not found: %s", entityID)
	})
}

func (b *MprBackend) deleteAssociationViaModelsdk(domainModelID, assocID model.ID) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		for i, a := range dm.Associations {
			if a.ID == assocID {
				dm.Associations = append(dm.Associations[:i], dm.Associations[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("association not found: %s", assocID)
	})
}

func (b *MprBackend) deleteCrossAssociationViaModelsdk(domainModelID, assocID model.ID) error {
	return b.writeDomainModel(domainModelID, func(dm *domainmodel.DomainModel) error {
		for i, ca := range dm.CrossAssociations {
			if ca.ID == assocID {
				dm.CrossAssociations = append(dm.CrossAssociations[:i], dm.CrossAssociations[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("cross-module association not found: %s", assocID)
	})
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
