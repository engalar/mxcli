// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// All Create* methods in this file produce canonical BSON via the existing
// sdk/mpr Serialize* helpers, then write the bytes through msdkWriter.InsertUnit
// (the modelsdk write path), bypassing sdk/mpr's updateTransactionID() — that
// call fails on hard-linked MPR files (SQLITE_READONLY_DBMOVED 1544).
//
// JavaAction was migrated to the gen-native javaActionRepo in Stage
// 3.3.2.D (commit c5695850); the bridge createJavaActionViaModelsdk
// was retired in Stage 3.3.2.E1.

// ── DatabaseConnection ────────────────────────────────────────────────────

func (b *MprBackend) createDatabaseConnectionViaModelsdk(conn *model.DatabaseConnection) error {
	return b.createDatabaseConnectionGen(conn)
}

// ── DataTransformer ───────────────────────────────────────────────────────

func (b *MprBackend) createDataTransformerViaModelsdk(dt *model.DataTransformer) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := sdkSerializeDataTransformer(dt)
	if err != nil {
		return fmt.Errorf("serialize data transformer: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(dt.ID),
		string(dt.ContainerID),
		"Documents",
		"DataTransformers$DataTransformer",
		contents,
	)
}

// ── ImportMapping ─────────────────────────────────────────────────────────

func (b *MprBackend) createImportMappingViaModelsdk(im *model.ImportMapping) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := sdkSerializeImportMapping(im)
	if err != nil {
		return fmt.Errorf("serialize import mapping: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(im.ID),
		string(im.ContainerID),
		"Documents",
		"ImportMappings$ImportMapping",
		contents,
	)
}

// ── ExportMapping ─────────────────────────────────────────────────────────

func (b *MprBackend) createExportMappingViaModelsdk(em *model.ExportMapping) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := sdkSerializeExportMapping(em)
	if err != nil {
		return fmt.Errorf("serialize export mapping: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(em.ID),
		string(em.ContainerID),
		"Documents",
		"ExportMappings$ExportMapping",
		contents,
	)
}

// ── JsonStructure ─────────────────────────────────────────────────────────

func (b *MprBackend) createJsonStructureViaModelsdk(js *types.JsonStructure) error {
	return b.createJsonStructureGen(js)
}

// ── BusinessEventService ──────────────────────────────────────────────────

func (b *MprBackend) createBusinessEventServiceViaModelsdk(svc *model.BusinessEventService) error {
	return b.createBusinessEventServiceGen(svc)
}

// ── OData services ────────────────────────────────────────────────────────

func (b *MprBackend) createConsumedODataServiceViaModelsdk(svc *model.ConsumedODataService) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := sdkSerializeConsumedODataService(svc)
	if err != nil {
		return fmt.Errorf("serialize consumed odata service: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(svc.ID),
		string(svc.ContainerID),
		"Documents",
		"Rest$ConsumedODataService",
		contents,
	)
}

func (b *MprBackend) createPublishedODataServiceViaModelsdk(svc *model.PublishedODataService) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := sdkSerializePublishedODataService(svc)
	if err != nil {
		return fmt.Errorf("serialize published odata service: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(svc.ID),
		string(svc.ContainerID),
		"Documents",
		"ODataPublish$PublishedODataService2",
		contents,
	)
}

// ── REST services ─────────────────────────────────────────────────────────

func (b *MprBackend) createConsumedRestServiceViaModelsdk(svc *model.ConsumedRestService) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := sdkSerializeConsumedRestService(svc)
	if err != nil {
		return fmt.Errorf("serialize consumed rest service: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(svc.ID),
		string(svc.ContainerID),
		"Documents",
		"Rest$ConsumedRestService",
		contents,
	)
}

func (b *MprBackend) createPublishedRestServiceViaModelsdk(svc *model.PublishedRestService) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := sdkSerializePublishedRestService(svc)
	if err != nil {
		return fmt.Errorf("serialize published rest service: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(svc.ID),
		string(svc.ContainerID),
		"Documents",
		"Rest$PublishedRestService",
		contents,
	)
}

// ── ImageCollection ───────────────────────────────────────────────────────

func (b *MprBackend) createImageCollectionViaModelsdk(ic *types.ImageCollection) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := sdkSerializeImageCollection(ic)
	if err != nil {
		return fmt.Errorf("serialize image collection: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(ic.ID),
		string(ic.ContainerID),
		"Documents",
		"Images$ImageCollection",
		contents,
	)
}

// ── Enumeration ───────────────────────────────────────────────────────────

func (b *MprBackend) createEnumerationViaModelsdk(enum *model.Enumeration) error {
	return b.createEnumerationGen(enum)
}

// ── Constant ──────────────────────────────────────────────────────────────

func (b *MprBackend) createConstantViaModelsdk(constant *model.Constant) error {
	return b.createConstantGen(constant)
}
