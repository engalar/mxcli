// SPDX-License-Identifier: Apache-2.0
package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
)

// ── OData ─────────────────────────────────────────────

func (b *MprBackend) deleteConsumedODataServiceViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) deletePublishedODataServiceViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

// ── REST ──────────────────────────────────────────────

func (b *MprBackend) deleteConsumedRestServiceViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) deletePublishedRestServiceViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

// ── BusinessEvent ─────────────────────────────────────

func (b *MprBackend) deleteBusinessEventServiceViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

// ── DatabaseConnection ────────────────────────────────

func (b *MprBackend) deleteDatabaseConnectionViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) moveDatabaseConnectionViaModelsdk(conn *model.DatabaseConnection) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(conn.ID), string(conn.ContainerID))
}

// ── DataTransformer ───────────────────────────────────

func (b *MprBackend) deleteDataTransformerViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

// ── Mappings ──────────────────────────────────────────

func (b *MprBackend) deleteImportMappingViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) moveImportMappingViaModelsdk(im *model.ImportMapping) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(im.ID), string(im.ContainerID))
}

func (b *MprBackend) deleteExportMappingViaModelsdk(id model.ID) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(string(id))
}

func (b *MprBackend) moveExportMappingViaModelsdk(em *model.ExportMapping) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(em.ID), string(em.ContainerID))
}

// ── JsonStructure (id is string) ──────────────────────

func (b *MprBackend) deleteJsonStructureViaModelsdk(id string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.DeleteUnit(id)
}
