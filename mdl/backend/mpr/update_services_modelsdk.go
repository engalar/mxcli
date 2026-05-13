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
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJsonStructures "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	genODP "github.com/mendixlabs/mxcli/modelsdk/gen/odatapublish"
	genREST "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// All Update* methods in this file decode the existing unit BSON, mutate
// scalar/enum/ref properties on the gen type, and write back via msdkWrite.
// PartList/Part children (operations, resources, queries, etc.) are preserved
// by LazyDoc and are mutated via dedicated mutator operations elsewhere.
//
// updateImageCollectionViaModelsdk still uses the legacy
// Serialize+writeUnitContents path because the modelsdk gen type is not yet
// wired up.

// ── JavaAction ────────────────────────────────────────────────────────────

func (b *MprBackend) updateJavaActionViaModelsdk(ja *javaactions.JavaAction) error {
	return b.msdkWrite(ja.ID, func(elem element.Element) error {
		typed, ok := elem.(*genJA.JavaAction)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *JavaAction)", elem)
		}
		typed.SetName(ja.Name)
		typed.SetDocumentation(ja.Documentation)
		typed.SetExcluded(ja.Excluded)
		typed.SetExportLevel(ja.ExportLevel)
		typed.SetActionDefaultReturnName(ja.ActionDefaultReturnName)
		// ActionReturnType (Part — polymorphic VoidType/StringType/etc.),
		// ActionParameters, ActionTypeParameters (PartList) and Java return
		// type (Part) preserved by LazyDoc; updated by dedicated mutator
		// operations.
		return nil
	})
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
	return b.writeUnitContents(ic.ID, contents)
}
