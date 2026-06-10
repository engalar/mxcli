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
	genODP "github.com/mendixlabs/mxcli/modelsdk/gen/odatapublish"
	genREST "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// All Update* methods in this file decode the existing unit BSON, mutate
// scalar/enum/ref properties on the gen type, and write back via msdkWrite.
// PartList/Part children (operations, resources, queries, etc.) are preserved
// by LazyDoc and are mutated via dedicated mutator operations elsewhere.
//
// updateImageCollectionViaModelsdk still uses the legacy
// Serialize+writeUnitContents path because the modelsdk gen type is not yet
// wired up.
//
// JavaAction was migrated to the gen-native javaActionRepo in Stage
// 3.3.2.D (commit c5695850); the bridge updateJavaActionViaModelsdk
// was retired in Stage 3.3.2.E1.

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
		// Rebuild the element tree so it matches the new snippet. Without this
		// the structure exposes no source and mappings fail mx check (CE0271).
		for i := len(typed.ElementsItems()) - 1; i >= 0; i-- {
			typed.RemoveElements(i)
		}
		for _, e := range js.Elements {
			typed.AddElements(jsonElementToGen(e))
		}
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
	return b.msdkWrite(svc.ID, func(elem element.Element) error {
		typed, ok := elem.(*genREST.ConsumedODataService)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ConsumedODataService)", elem)
		}
		typed.SetName(svc.Name)
		typed.SetDocumentation(svc.Documentation)
		typed.SetExcluded(svc.Excluded)
		typed.SetServiceName(svc.ServiceName)
		typed.SetVersion(svc.Version)
		typed.SetODataVersion(svc.ODataVersion)
		typed.SetMetadataUrl(svc.MetadataUrl)
		typed.SetTimeoutExpression(svc.TimeoutExpression)
		typed.SetProxyType(svc.ProxyType)
		typed.SetDescription(svc.Description)
		typed.SetValidated(svc.Validated)
		typed.SetMetadata(svc.Metadata)
		typed.SetMetadataHash(svc.MetadataHash)
		typed.SetApplicationId(svc.ApplicationId)
		typed.SetEndpointId(svc.EndpointId)
		typed.SetCatalogUrl(svc.CatalogUrl)
		typed.SetEnvironmentType(svc.EnvironmentType)
		typed.SetConfigurationMicroflowQualifiedName(svc.ConfigurationMicroflow)
		typed.SetErrorHandlingMicroflowQualifiedName(svc.ErrorHandlingMicroflow)
		typed.SetProxyHostQualifiedName(svc.ProxyHost)
		typed.SetProxyPortQualifiedName(svc.ProxyPort)
		typed.SetProxyUsernameQualifiedName(svc.ProxyUsername)
		typed.SetProxyPasswordQualifiedName(svc.ProxyPassword)
		// HttpConfiguration (Part) preserved by LazyDoc.
		return nil
	})
}

func (b *MprBackend) updatePublishedODataServiceViaModelsdk(svc *model.PublishedODataService) error {
	return b.msdkWrite(svc.ID, func(elem element.Element) error {
		typed, ok := elem.(*genODP.PublishedODataService2)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *PublishedODataService2)", elem)
		}
		typed.SetName(svc.Name)
		typed.SetDocumentation(svc.Documentation)
		typed.SetExcluded(svc.Excluded)
		typed.SetPath(svc.Path)
		typed.SetNamespace(svc.Namespace)
		typed.SetServiceName(svc.ServiceName)
		typed.SetVersion(svc.Version)
		typed.SetODataVersion(svc.ODataVersion)
		typed.SetSummary(svc.Summary)
		typed.SetDescription(svc.Description)
		typed.SetPublishAssociations(svc.PublishAssociations)
		typed.SetUseGeneralization(svc.UseGeneralization)
		typed.SetAuthenticationMicroflowQualifiedName(svc.AuthMicroflow)
		typed.SetAllowedModuleRolesQualifiedNames(svc.AllowedModuleRoles)
		// EntityTypes, EntitySets (PartList) preserved by LazyDoc.
		return nil
	})
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
	contents, err := modelsdkmpr.SerializePublishedRestService(svc)
	if err != nil {
		return fmt.Errorf("serialize published rest service: %w", err)
	}
	return b.writeUnitContents(svc.ID, contents)
}

// ── ImageCollection ───────────────────────────────────────────────────────

func (b *MprBackend) updateImageCollectionViaModelsdk(ic *types.ImageCollection) error {
	contents, err := modelsdkmpr.SerializeImageCollection(ic)
	if err != nil {
		return fmt.Errorf("serialize image collection: %w", err)
	}
	return b.writeUnitContents(ic.ID, contents)
}

func (b *MprBackend) moveImageCollectionViaModelsdk(ic *types.ImageCollection) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	return b.msdkWriter.UpdateUnitContainer(string(ic.ID), string(ic.ContainerID))
}
