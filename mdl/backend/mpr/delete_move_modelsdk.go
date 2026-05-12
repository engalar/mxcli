// SPDX-License-Identifier: Apache-2.0
package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/microflows"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

// ── Domain Model ──────────────────────────────────────

func (b *MprBackend) deleteEntityViaModelsdk(domainModelID, entityID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	_ = domainModelID
	return b.msdkWriter.DeleteUnit(string(entityID))
}

func (b *MprBackend) deleteAttributeViaModelsdk(domainModelID, entityID, attrID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	_ = domainModelID
	_ = entityID
	return b.msdkWriter.DeleteUnit(string(attrID))
}

func (b *MprBackend) deleteAssociationViaModelsdk(domainModelID, assocID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	_ = domainModelID
	return b.msdkWriter.DeleteUnit(string(assocID))
}

func (b *MprBackend) deleteCrossAssociationViaModelsdk(domainModelID, assocID model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	_ = domainModelID
	return b.msdkWriter.DeleteUnit(string(assocID))
}

// ── Microflow / Nanoflow ───────────────────────────────

func (b *MprBackend) deleteMicroflowViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) moveMicroflowViaModelsdk(mf *microflows.Microflow) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(mf.ID), string(mf.ContainerID))
}

func (b *MprBackend) deleteNanoflowViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) moveNanoflowViaModelsdk(nf *microflows.Nanoflow) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(nf.ID), string(nf.ContainerID))
}

// ── Pages / Layouts / Snippets ────────────────────────

func (b *MprBackend) deletePageViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) movePageViaModelsdk(page *pages.Page) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(page.ID), string(page.ContainerID))
}

func (b *MprBackend) deleteLayoutViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) deleteSnippetViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) moveSnippetViaModelsdk(snippet *pages.Snippet) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(snippet.ID), string(snippet.ContainerID))
}

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
