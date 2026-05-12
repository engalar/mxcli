// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// All Create* methods in this file produce canonical BSON via the existing
// sdk/mpr Serialize* helpers, then write the bytes through msdkWriter.InsertUnit
// (the modelsdk write path), bypassing sdk/mpr's updateTransactionID() — that
// call fails on hard-linked MPR files (SQLITE_READONLY_DBMOVED 1544).

// ── JavaAction ────────────────────────────────────────────────────────────

func (b *MprBackend) createJavaActionViaModelsdk(ja *javaactions.JavaAction) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeJavaAction(ja)
	if err != nil {
		return fmt.Errorf("serialize java action: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(ja.ID),
		string(ja.ContainerID),
		"Documents",
		"JavaActions$JavaAction",
		contents,
	)
}

// ── DatabaseConnection ────────────────────────────────────────────────────

func (b *MprBackend) createDatabaseConnectionViaModelsdk(conn *model.DatabaseConnection) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeDatabaseConnection(conn)
	if err != nil {
		return fmt.Errorf("serialize database connection: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(conn.ID),
		string(conn.ContainerID),
		"Documents",
		"DatabaseConnector$DatabaseConnection",
		contents,
	)
}

// ── DataTransformer ───────────────────────────────────────────────────────

func (b *MprBackend) createDataTransformerViaModelsdk(dt *model.DataTransformer) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := mpr.SerializeDataTransformer(dt)
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
	contents, err := b.writer.SerializeImportMapping(im)
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
	contents, err := b.writer.SerializeExportMapping(em)
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
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := mpr.SerializeJsonStructure(unconvertJsonStructure(js))
	if err != nil {
		return fmt.Errorf("serialize json structure: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(js.ID),
		string(js.ContainerID),
		"Documents",
		"JsonStructures$JsonStructure",
		contents,
	)
}

// ── BusinessEventService ──────────────────────────────────────────────────

func (b *MprBackend) createBusinessEventServiceViaModelsdk(svc *model.BusinessEventService) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeBusinessEventService(svc)
	if err != nil {
		return fmt.Errorf("serialize business event service: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(svc.ID),
		string(svc.ContainerID),
		"Documents",
		"BusinessEvents$BusinessEventService",
		contents,
	)
}

// ── OData services ────────────────────────────────────────────────────────

func (b *MprBackend) createConsumedODataServiceViaModelsdk(svc *model.ConsumedODataService) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeConsumedODataService(svc)
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
	contents, err := b.writer.SerializePublishedODataService(svc)
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
	contents, err := b.writer.SerializeConsumedRestService(svc)
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
	contents, err := b.writer.SerializePublishedRestService(svc)
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
	contents, err := mpr.SerializeImageCollection(unconvertImageCollection(ic))
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
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeEnumeration(enum)
	if err != nil {
		return fmt.Errorf("serialize enumeration: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(enum.ID),
		string(enum.ContainerID),
		"Documents",
		"Enumerations$Enumeration",
		contents,
	)
}

// ── Constant ──────────────────────────────────────────────────────────────

func (b *MprBackend) createConstantViaModelsdk(constant *model.Constant) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeConstant(constant)
	if err != nil {
		return fmt.Errorf("serialize constant: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(constant.ID),
		string(constant.ContainerID),
		"Documents",
		"Constants$Constant",
		contents,
	)
}
