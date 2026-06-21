// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genBE "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genDBC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDTrans "github.com/mendixlabs/mxcli/modelsdk/gen/datatransformers"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

// ---------------------------------------------------------------------------
// NavigationBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListNavigationDocuments() ([]*types.NavigationDocument, error) {
	return b.listNavigationDocumentsFromRaw()
}
func (b *MprBackend) GetNavigation() (*types.NavigationDocument, error) {
	return b.getNavigationFromRaw()
}
func (b *MprBackend) UpdateNavigationProfile(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error {
	return b.updateNavigationProfileViaModelsdk(navDocID, profileName, spec)
}

// ---------------------------------------------------------------------------
// ServiceBackend (OData + REST + BusinessEvent + DatabaseConnection + DataTransformer)
// ---------------------------------------------------------------------------

func (b *MprBackend) ListConsumedODataServices() ([]*model.ConsumedODataService, error) {
	// TODO(phase4): migrate to mprread. ConsumedODataService has 28 scalar fields plus
	// HttpConfiguration + HttpHeaderEntry nested parts; full converter ~80 lines.
	// gen package: modelsdk/gen/rest (Rest$ConsumedODataService).
	return b.listConsumedODataServicesFromRaw()
}
func (b *MprBackend) ListPublishedODataServices() ([]*model.PublishedODataService, error) {
	// TODO(phase4): migrate to mprread. PublishedODataService has nested EntityTypes
	// (with Members) + EntitySets + AuthenticationTypes + AllowedModuleRoles.
	// gen package: modelsdk/gen/odatapublish (ODataPublish$PublishedODataService2).
	return b.listPublishedODataServicesFromRaw()
}
func (b *MprBackend) CreateConsumedODataService(svc *model.ConsumedODataService) error {
	return b.createConsumedODataServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdateConsumedODataService(svc *model.ConsumedODataService) error {
	return b.updateConsumedODataServiceViaModelsdk(svc)
}
func (b *MprBackend) DeleteConsumedODataService(id model.ID) error {
	return b.deleteConsumedODataServiceViaModelsdk(id)
}
func (b *MprBackend) CreatePublishedODataService(svc *model.PublishedODataService) error {
	return b.createPublishedODataServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdatePublishedODataService(svc *model.PublishedODataService) error {
	return b.updatePublishedODataServiceViaModelsdk(svc)
}
func (b *MprBackend) DeletePublishedODataService(id model.ID) error {
	return b.deletePublishedODataServiceViaModelsdk(id)
}

func (b *MprBackend) ListConsumedRestServices() ([]*model.ConsumedRestService, error) {
	// TODO(phase4): migrate to mprread. Same shape class as ConsumedODataService —
	// scalar transport config + Operations nested list.
	// gen package: modelsdk/gen/rest (Rest$ConsumedRestService).
	return b.listConsumedRestServicesFromRaw()
}
func (b *MprBackend) ListPublishedRestServices() ([]*model.PublishedRestService, error) {
	// TODO(phase4): migrate to mprread. Has Resources → Operations → PathParams tree.
	// gen package: modelsdk/gen/rest (Rest$PublishedRestService).
	return b.listPublishedRestServicesFromRaw()
}
func (b *MprBackend) CreateConsumedRestService(svc *model.ConsumedRestService) error {
	return b.createConsumedRestServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdateConsumedRestService(svc *model.ConsumedRestService) error {
	return b.updateConsumedRestServiceViaModelsdk(svc)
}
func (b *MprBackend) DeleteConsumedRestService(id model.ID) error {
	return b.deleteConsumedRestServiceViaModelsdk(id)
}
func (b *MprBackend) CreatePublishedRestService(svc *model.PublishedRestService) error {
	return b.createPublishedRestServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdatePublishedRestService(svc *model.PublishedRestService) error {
	return b.updatePublishedRestServiceViaModelsdk(svc)
}
func (b *MprBackend) DeletePublishedRestService(id model.ID) error {
	return b.deletePublishedRestServiceViaModelsdk(id)
}

func (b *MprBackend) ListBusinessEventServices() ([]*model.BusinessEventService, error) {
	units, err := mprread.ListUnitsWithContainer[*genBE.BusinessEventService](b.msdkReader)
	if err != nil {
		return nil, err
	}
	return businessEventServiceUnitsToModel(units), nil
}
func (b *MprBackend) CreateBusinessEventService(svc *model.BusinessEventService) error {
	return b.createBusinessEventServiceViaModelsdk(svc)
}
func (b *MprBackend) UpdateBusinessEventService(svc *model.BusinessEventService) error {
	return b.updateBusinessEventServiceViaModelsdk(svc)
}
func (b *MprBackend) DeleteBusinessEventService(id model.ID) error {
	return b.deleteBusinessEventServiceViaModelsdk(id)
}

func (b *MprBackend) ListDatabaseConnections() ([]*model.DatabaseConnection, error) {
	units, err := mprread.ListUnitsWithContainer[*genDBC.DatabaseConnection](b.msdkReader)
	if err != nil {
		return nil, err
	}
	return databaseConnectionUnitsToModel(units), nil
}
func (b *MprBackend) CreateDatabaseConnection(conn *model.DatabaseConnection) error {
	return b.createDatabaseConnectionViaModelsdk(conn)
}
func (b *MprBackend) UpdateDatabaseConnection(conn *model.DatabaseConnection) error {
	return b.updateDatabaseConnectionViaModelsdk(conn)
}
func (b *MprBackend) MoveDatabaseConnection(conn *model.DatabaseConnection) error {
	return b.moveDatabaseConnectionViaModelsdk(conn)
}
func (b *MprBackend) DeleteDatabaseConnection(id model.ID) error {
	return b.deleteDatabaseConnectionViaModelsdk(id)
}

func (b *MprBackend) ListDataTransformers() ([]*model.DataTransformer, error) {
	units, err := mprread.ListUnitsWithContainer[*genDTrans.DataTransformer](b.msdkReader)
	if err != nil {
		return nil, err
	}
	return dataTransformerUnitsToModel(units), nil
}
func (b *MprBackend) CreateDataTransformer(dt *model.DataTransformer) error {
	return b.createDataTransformerViaModelsdk(dt)
}
func (b *MprBackend) UpdateDataTransformer(dt *model.DataTransformer) error {
	return b.updateDataTransformerViaModelsdk(dt)
}
func (b *MprBackend) DeleteDataTransformer(id model.ID) error {
	return b.deleteDataTransformerViaModelsdk(id)
}

// ---------------------------------------------------------------------------
// MappingBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListImportMappings() ([]*model.ImportMapping, error) {
	b.initSubBackends()
	return b.mappings.ListImportMappings()
}
func (b *MprBackend) GetImportMappingByQualifiedName(moduleName, name string) (*model.ImportMapping, error) {
	b.initSubBackends()
	return b.mappings.GetImportMappingByQualifiedName(moduleName, name)
}
func (b *MprBackend) CreateImportMapping(im *model.ImportMapping) error {
	return b.createImportMappingViaModelsdk(im)
}
func (b *MprBackend) UpdateImportMapping(im *model.ImportMapping) error {
	return b.updateImportMappingViaModelsdk(im)
}
func (b *MprBackend) DeleteImportMapping(id model.ID) error {
	return b.deleteImportMappingViaModelsdk(id)
}
func (b *MprBackend) MoveImportMapping(im *model.ImportMapping) error {
	return b.moveImportMappingViaModelsdk(im)
}

func (b *MprBackend) ListExportMappings() ([]*model.ExportMapping, error) {
	b.initSubBackends()
	return b.mappings.ListExportMappings()
}
func (b *MprBackend) GetExportMappingByQualifiedName(moduleName, name string) (*model.ExportMapping, error) {
	b.initSubBackends()
	return b.mappings.GetExportMappingByQualifiedName(moduleName, name)
}
func (b *MprBackend) CreateExportMapping(em *model.ExportMapping) error {
	return b.createExportMappingViaModelsdk(em)
}
func (b *MprBackend) UpdateExportMapping(em *model.ExportMapping) error {
	return b.updateExportMappingViaModelsdk(em)
}
func (b *MprBackend) DeleteExportMapping(id model.ID) error {
	return b.deleteExportMappingViaModelsdk(id)
}
func (b *MprBackend) MoveExportMapping(em *model.ExportMapping) error {
	return b.moveExportMappingViaModelsdk(em)
}

func (b *MprBackend) ListJsonStructures() ([]*types.JsonStructure, error) {
	b.initSubBackends()
	return b.mappings.ListJsonStructures()
}
func (b *MprBackend) GetJsonStructureByQualifiedName(moduleName, name string) (*types.JsonStructure, error) {
	all, err := b.ListJsonStructures()
	if err != nil {
		return nil, err
	}
	// js.ContainerID may be a folder (MPR v2) rather than the module itself,
	// so walk the container hierarchy via the sdk/mpr helper to resolve the
	// enclosing module name. TODO(phase4): port buildContainerModuleNameMap
	// to use mprread once Folder lister exists there.
	containerToModule, err := b.buildContainerModuleNameMapViaSdk()
	if err != nil {
		return nil, err
	}
	for _, js := range all {
		if js.Name == name && containerToModule[js.ContainerID] == moduleName {
			return js, nil
		}
	}
	return nil, fmt.Errorf("json structure %s.%s not found", moduleName, name)
}

// buildContainerModuleNameMapViaSdk delegates to the legacy sdk/mpr reader's
// hierarchy walker. Used by GetXxxByQualifiedName methods until the helper
// is ported to mprread.
func (b *MprBackend) buildContainerModuleNameMapViaSdk() (map[model.ID]string, error) {
	modules, err := b.ListModules()
	if err != nil {
		return nil, err
	}
	moduleNames := make(map[model.ID]string, len(modules))
	for _, m := range modules {
		moduleNames[m.ID] = m.Name
	}
	units, err := b.ListUnits()
	if err != nil {
		return nil, err
	}
	parentOf := make(map[model.ID]model.ID, len(units))
	for _, u := range units {
		parentOf[u.ID] = u.ContainerID
	}
	result := make(map[model.ID]string)
	var find func(id model.ID) string
	find = func(id model.ID) string {
		if cached, ok := result[id]; ok {
			return cached
		}
		if name, ok := moduleNames[id]; ok {
			result[id] = name
			return name
		}
		parent, ok := parentOf[id]
		if !ok || parent == id || parent == "" {
			result[id] = ""
			return ""
		}
		name := find(parent)
		result[id] = name
		return name
	}
	for id := range parentOf {
		find(id)
	}
	return result, nil
}
func (b *MprBackend) CreateJsonStructure(js *types.JsonStructure) error {
	return b.createJsonStructureViaModelsdk(js)
}
func (b *MprBackend) UpdateJsonStructure(js *types.JsonStructure) error {
	return b.updateJsonStructureViaModelsdk(js)
}
func (b *MprBackend) DeleteJsonStructure(id string) error {
	return b.deleteJsonStructureViaModelsdk(id)
}

// ---------------------------------------------------------------------------
// SettingsBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) GetProjectSettings() (*model.ProjectSettings, error) {
	// TODO(phase4): migrate to mprread. Settings$ProjectSettings holds a polymorphic
	// Settings array (10+ Part subtypes: WebUI, Integration, Configuration, Model,
	// Convention, Language, Certificate, Workflows, JarDeployment, Distribution).
	// model.ProjectSettings.RawParts []map[string]any is critical for round-trip
	// fidelity of unrecognized part types — gen-typed read drops this.
	// gen package: modelsdk/gen/settings (Settings$ProjectSettings).
	return b.getProjectSettingsFromRaw()
}
func (b *MprBackend) UpdateProjectSettings(ps *model.ProjectSettings) error {
	return b.updateProjectSettingsViaModelsdk(ps)
}

// ---------------------------------------------------------------------------
// ImageBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) ListImageCollections() ([]*types.ImageCollection, error) {
	return b.listImageCollectionsViaModelsdk()
}
func (b *MprBackend) CreateImageCollection(ic *types.ImageCollection) error {
	return b.createImageCollectionViaModelsdk(ic)
}
func (b *MprBackend) UpdateImageCollection(ic *types.ImageCollection) error {
	return b.updateImageCollectionViaModelsdk(ic)
}
func (b *MprBackend) DeleteImageCollection(id string) error {
	return b.deleteImageCollectionViaModelsdk(id)
}
func (b *MprBackend) MoveImageCollection(ic *types.ImageCollection) error {
	return b.moveImageCollectionViaModelsdk(ic)
}
