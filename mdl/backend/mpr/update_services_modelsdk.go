// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genBE "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genDB "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatransformers"
	genEM "github.com/mendixlabs/mxcli/modelsdk/gen/exportmappings"
	genIM "github.com/mendixlabs/mxcli/modelsdk/gen/importmappings"
	genJsonStructures "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	genREST "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
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
	return b.msdkWrite(conn.ID, func(elem element.Element) error {
		typed, ok := elem.(*genDB.DatabaseConnection)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *DatabaseConnection)", elem)
		}
		typed.SetName(conn.Name)
		typed.SetDocumentation(conn.Documentation)
		typed.SetExcluded(conn.Excluded)
		typed.SetExportLevel(conn.ExportLevel)
		typed.SetDatabaseType(conn.DatabaseType)
		typed.SetConnectionStringQualifiedName(conn.ConnectionString)
		typed.SetUserNameQualifiedName(conn.UserName)
		typed.SetPasswordQualifiedName(conn.Password)
		// ConnectionInput (Part — JDBC URL value) and Queries (PartList)
		// preserved by LazyDoc; updated by dedicated mutator operations.
		return nil
	})
}

// ── DataTransformer ───────────────────────────────────────────────────────

func (b *MprBackend) updateDataTransformerViaModelsdk(dt *model.DataTransformer) error {
	return b.msdkWrite(dt.ID, func(elem element.Element) error {
		typed, ok := elem.(*genDT.DataTransformer)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *DataTransformer)", elem)
		}
		typed.SetName(dt.Name)
		typed.SetExcluded(dt.Excluded)
		if dt.SourceType != "" {
			typed.SetSourceType(dt.SourceType)
		}
		if dt.SourceJSON != "" {
			typed.SetSourceJson(dt.SourceJSON)
		}
		// Source (Part) and Elements/Steps (PartList) preserved by LazyDoc;
		// updated by dedicated mutator operations.
		return nil
	})
}

// ── ImportMapping ─────────────────────────────────────────────────────────

func (b *MprBackend) updateImportMappingViaModelsdk(im *model.ImportMapping) error {
	return b.msdkWrite(im.ID, func(elem element.Element) error {
		typed, ok := elem.(*genIM.ImportMapping)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ImportMapping)", elem)
		}
		typed.SetName(im.Name)
		typed.SetDocumentation(im.Documentation)
		typed.SetExcluded(im.Excluded)
		typed.SetExportLevel(im.ExportLevel)
		typed.SetXmlSchemaQualifiedName(im.XmlSchema)
		typed.SetJsonStructureQualifiedName(im.JsonStructure)
		typed.SetMessageDefinitionQualifiedName(im.MessageDefinition)
		// RootMappingElements (PartList) preserved by LazyDoc.
		return nil
	})
}

// ── ExportMapping ─────────────────────────────────────────────────────────

func (b *MprBackend) updateExportMappingViaModelsdk(em *model.ExportMapping) error {
	return b.msdkWrite(em.ID, func(elem element.Element) error {
		typed, ok := elem.(*genEM.ExportMapping)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ExportMapping)", elem)
		}
		typed.SetName(em.Name)
		typed.SetDocumentation(em.Documentation)
		typed.SetExcluded(em.Excluded)
		typed.SetExportLevel(em.ExportLevel)
		typed.SetXmlSchemaQualifiedName(em.XmlSchema)
		typed.SetJsonStructureQualifiedName(em.JsonStructure)
		typed.SetMessageDefinitionQualifiedName(em.MessageDefinition)
		typed.SetNullValueOption(em.NullValueOption)
		// RootMappingElements (PartList) preserved by LazyDoc.
		return nil
	})
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
	return b.msdkWrite(svc.ID, func(elem element.Element) error {
		typed, ok := elem.(*genBE.BusinessEventService)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *BusinessEventService)", elem)
		}
		typed.SetName(svc.Name)
		typed.SetDocumentation(svc.Documentation)
		typed.SetExcluded(svc.Excluded)
		typed.SetExportLevel(svc.ExportLevel)
		typed.SetDocument(svc.Document)
		// Definition (Part) and OperationImplementations (PartList) are
		// preserved by LazyDoc; updated by dedicated mutator operations.
		return nil
	})
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
	return b.msdkWrite(svc.ID, func(elem element.Element) error {
		typed, ok := elem.(*genREST.ConsumedRestService)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ConsumedRestService)", elem)
		}
		typed.SetName(svc.Name)
		typed.SetDocumentation(svc.Documentation)
		typed.SetExcluded(svc.Excluded)
		// BaseUrl, AuthenticationScheme, OpenApiFile, Operations are
		// Part/PartList children — preserved by LazyDoc.
		return nil
	})
}

func (b *MprBackend) updatePublishedRestServiceViaModelsdk(svc *model.PublishedRestService) error {
	return b.msdkWrite(svc.ID, func(elem element.Element) error {
		typed, ok := elem.(*genREST.PublishedRestService)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *PublishedRestService)", elem)
		}
		typed.SetName(svc.Name)
		typed.SetExcluded(svc.Excluded)
		typed.SetServiceName(svc.ServiceName)
		typed.SetVersion(svc.Version)
		typed.SetPath(svc.Path)
		typed.SetAllowedRolesQualifiedNames(svc.AllowedRoles)
		// Resources (PartList) preserved by LazyDoc.
		return nil
	})
}

// ── ImageCollection ───────────────────────────────────────────────────────

func (b *MprBackend) updateImageCollectionViaModelsdk(ic *types.ImageCollection) error {
	contents, err := mpr.SerializeImageCollection(unconvertImageCollection(ic))
	if err != nil {
		return fmt.Errorf("serialize image collection: %w", err)
	}
	return b.msdkWriteRaw(ic.ID, contents)
}
