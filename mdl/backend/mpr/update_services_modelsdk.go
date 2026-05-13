// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJsonStructures "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// All Update* methods in this file produce canonical BSON via the existing
// sdk/mpr Serialize* helpers, then write the bytes through msdkWriteRaw
// (defined in security_allowed_roles_modelsdk.go), which bypasses sdk/mpr's
// updateTransactionID() — that call fails on hard-linked MPR files
// (SQLITE_READONLY_DBMOVED 1544).

// ── JavaAction ────────────────────────────────────────────────────────────

func (b *MprBackend) updateJavaActionViaModelsdk(ja *javaactions.JavaAction) error {
	contents, err := b.writer.SerializeJavaAction(ja)
	if err != nil {
		return fmt.Errorf("serialize java action: %w", err)
	}
	return b.msdkWriteRaw(ja.ID, contents)
}

// ── DatabaseConnection ────────────────────────────────────────────────────

func (b *MprBackend) updateDatabaseConnectionViaModelsdk(conn *model.DatabaseConnection) error {
	contents, err := b.writer.SerializeDatabaseConnection(conn)
	if err != nil {
		return fmt.Errorf("serialize database connection: %w", err)
	}
	return b.msdkWriteRaw(conn.ID, contents)
}

// ── DataTransformer ───────────────────────────────────────────────────────

func (b *MprBackend) updateDataTransformerViaModelsdk(dt *model.DataTransformer) error {
	contents, err := mpr.SerializeDataTransformer(dt)
	if err != nil {
		return fmt.Errorf("serialize data transformer: %w", err)
	}
	return b.msdkWriteRaw(dt.ID, contents)
}

// ── ImportMapping ─────────────────────────────────────────────────────────

func (b *MprBackend) updateImportMappingViaModelsdk(im *model.ImportMapping) error {
	contents, err := b.writer.SerializeImportMapping(im)
	if err != nil {
		return fmt.Errorf("serialize import mapping: %w", err)
	}
	return b.msdkWriteRaw(im.ID, contents)
}

// ── ExportMapping ─────────────────────────────────────────────────────────

func (b *MprBackend) updateExportMappingViaModelsdk(em *model.ExportMapping) error {
	contents, err := b.writer.SerializeExportMapping(em)
	if err != nil {
		return fmt.Errorf("serialize export mapping: %w", err)
	}
	return b.msdkWriteRaw(em.ID, contents)
}

// ── JsonStructure ─────────────────────────────────────────────────────────

func (b *MprBackend) updateJsonStructureViaModelsdk(js *types.JsonStructure) error {
	return b.msdkWrite(js.ID, func(elem element.Element) error {
		typed, ok := elem.(*genJsonStructures.JsonStructure)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *JsonStructure)", elem)
		}
		typed.SetName(js.Name)
		typed.SetDocumentation(js.Documentation)
		typed.SetExcluded(js.Excluded)
		typed.SetExportLevel(js.ExportLevel)
		typed.SetJsonSnippet(js.JsonSnippet)
		return nil
	})
}

// ── BusinessEventService ──────────────────────────────────────────────────

func (b *MprBackend) updateBusinessEventServiceViaModelsdk(svc *model.BusinessEventService) error {
	contents, err := b.writer.SerializeBusinessEventService(svc)
	if err != nil {
		return fmt.Errorf("serialize business event service: %w", err)
	}
	return b.msdkWriteRaw(svc.ID, contents)
}

// ── OData services ────────────────────────────────────────────────────────

func (b *MprBackend) updateConsumedODataServiceViaModelsdk(svc *model.ConsumedODataService) error {
	contents, err := b.writer.SerializeConsumedODataService(svc)
	if err != nil {
		return fmt.Errorf("serialize consumed odata service: %w", err)
	}
	return b.msdkWriteRaw(svc.ID, contents)
}

func (b *MprBackend) updatePublishedODataServiceViaModelsdk(svc *model.PublishedODataService) error {
	contents, err := b.writer.SerializePublishedODataService(svc)
	if err != nil {
		return fmt.Errorf("serialize published odata service: %w", err)
	}
	return b.msdkWriteRaw(svc.ID, contents)
}

// ── REST services ─────────────────────────────────────────────────────────

func (b *MprBackend) updateConsumedRestServiceViaModelsdk(svc *model.ConsumedRestService) error {
	contents, err := b.writer.SerializeConsumedRestService(svc)
	if err != nil {
		return fmt.Errorf("serialize consumed rest service: %w", err)
	}
	return b.msdkWriteRaw(svc.ID, contents)
}

func (b *MprBackend) updatePublishedRestServiceViaModelsdk(svc *model.PublishedRestService) error {
	contents, err := b.writer.SerializePublishedRestService(svc)
	if err != nil {
		return fmt.Errorf("serialize published rest service: %w", err)
	}
	return b.msdkWriteRaw(svc.ID, contents)
}

// ── ImageCollection ───────────────────────────────────────────────────────

func (b *MprBackend) updateImageCollectionViaModelsdk(ic *types.ImageCollection) error {
	contents, err := mpr.SerializeImageCollection(unconvertImageCollection(ic))
	if err != nil {
		return fmt.Errorf("serialize image collection: %w", err)
	}
	return b.msdkWriteRaw(ic.ID, contents)
}
