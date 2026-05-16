// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// Public Serialize* package-level functions produce canonical BSON bytes for
// each unit type without going through Writer.updateUnit (which calls
// updateTransactionID() and fails on hard-linked MPR files —
// SQLITE_READONLY_DBMOVED 1544).
//
// SerializeMicroflow / SerializeNanoflow / SerializeDomainModel live in
// writer_microflow.go and writer_domainmodel.go because they take a
// ProjectVersion parameter.

// SerializeDatabaseConnection returns BSON bytes for a database connection unit.
func SerializeDatabaseConnection(conn *model.DatabaseConnection) ([]byte, error) {
	return serializeDatabaseConnection(conn)
}

// SerializeDataTransformer returns BSON bytes for a data transformer unit.
func SerializeDataTransformer(dt *model.DataTransformer) ([]byte, error) {
	return serializeDataTransformer(dt)
}

// SerializeImageCollection returns BSON bytes for an image collection unit.
func SerializeImageCollection(ic *ImageCollection) ([]byte, error) {
	return serializeImageCollection(ic)
}

// SerializeJsonStructure returns BSON bytes for a JSON structure unit.
func SerializeJsonStructure(js *JsonStructure) ([]byte, error) {
	return serializeJsonStructure(js)
}

// SerializeImportMapping returns BSON bytes for an import mapping unit.
func SerializeImportMapping(im *model.ImportMapping) ([]byte, error) {
	return serializeImportMapping(im)
}

// SerializeExportMapping returns BSON bytes for an export mapping unit.
func SerializeExportMapping(em *model.ExportMapping) ([]byte, error) {
	return serializeExportMapping(em)
}

// SerializeBusinessEventService returns BSON bytes for a business event service unit.
func SerializeBusinessEventService(svc *model.BusinessEventService) ([]byte, error) {
	return serializeBusinessEventService(svc)
}

// SerializeConsumedODataService returns BSON bytes for a consumed OData service unit.
func SerializeConsumedODataService(svc *model.ConsumedODataService) ([]byte, error) {
	return serializeConsumedODataService(svc)
}

// SerializePublishedODataService returns BSON bytes for a published OData service unit.
func SerializePublishedODataService(svc *model.PublishedODataService) ([]byte, error) {
	return serializePublishedODataService(svc)
}

// SerializeConsumedRestService returns BSON bytes for a consumed REST service unit.
func SerializeConsumedRestService(svc *model.ConsumedRestService) ([]byte, error) {
	return serializeConsumedRestService(svc)
}

// SerializePublishedRestService returns BSON bytes for a published REST service unit.
func SerializePublishedRestService(svc *model.PublishedRestService) ([]byte, error) {
	return serializePublishedRestService(svc)
}

// SerializeEnumeration returns BSON bytes for an enumeration unit.
func SerializeEnumeration(enum *model.Enumeration) ([]byte, error) {
	return serializeEnumeration(enum)
}

// SerializeConstant returns BSON bytes for a constant unit.
func SerializeConstant(constant *model.Constant) ([]byte, error) {
	return serializeConstant(constant)
}

// SerializeProjectSettings returns BSON bytes for the project settings unit.
func SerializeProjectSettings(ps *model.ProjectSettings) ([]byte, error) {
	return serializeProjectSettings(ps)
}

// SerializeModule returns BSON bytes for a module unit.
func SerializeModule(module *model.Module) ([]byte, error) {
	return serializeModule(module)
}

// SerializeFolder returns BSON bytes for a folder unit.
func SerializeFolder(folder *model.Folder) ([]byte, error) {
	return serializeFolder(folder)
}

// SerializeModuleSecurity returns BSON bytes for an empty module security unit.
func SerializeModuleSecurity(id string) ([]byte, error) {
	return serializeModuleSecurity(id)
}

// SerializeModuleSettings returns BSON bytes for a default module settings unit.
func SerializeModuleSettings(id string) ([]byte, error) {
	return serializeModuleSettings(id)
}

// SerializeAgentEditorModel returns canonical CustomBlobDocument BSON bytes for
// an agent-editor Model unit. The Provider defaults to "MxCloudGenAI" when
// unset, mirroring the validation in CreateAgentEditorModel.
func SerializeAgentEditorModel(m *types.Model) ([]byte, error) {
	if m.Provider == "" {
		m.Provider = "MxCloudGenAI"
	}
	contentsJSON, err := encodeAgentEditorModelContents(m)
	if err != nil {
		return nil, err
	}
	return serializeCustomBlobDocument(&customBlobInput{
		UnitID:             string(m.ID),
		ContainerID:        string(m.ContainerID),
		Name:               m.Name,
		Documentation:      m.Documentation,
		Excluded:           m.Excluded,
		ExportLevel:        m.ExportLevel,
		CustomDocumentType: types.CustomTypeModel,
		ReadableTypeName:   types.ReadableModel,
		ContentsJSON:       contentsJSON,
	})
}

// SerializeAgentEditorKnowledgeBase returns canonical CustomBlobDocument BSON
// bytes for an agent-editor KnowledgeBase unit.
func SerializeAgentEditorKnowledgeBase(k *types.KnowledgeBase) ([]byte, error) {
	if k.Provider == "" {
		k.Provider = "MxCloudGenAI"
	}
	contentsJSON, err := encodeKnowledgeBaseContents(k)
	if err != nil {
		return nil, err
	}
	return serializeCustomBlobDocument(&customBlobInput{
		UnitID:             string(k.ID),
		ContainerID:        string(k.ContainerID),
		Name:               k.Name,
		Documentation:      k.Documentation,
		Excluded:           k.Excluded,
		ExportLevel:        k.ExportLevel,
		CustomDocumentType: types.CustomTypeKnowledgeBase,
		ReadableTypeName:   types.ReadableKnowledgeBase,
		ContentsJSON:       contentsJSON,
	})
}

// SerializeAgentEditorConsumedMCPService returns canonical CustomBlobDocument
// BSON bytes for an agent-editor Consumed MCP Service unit.
func SerializeAgentEditorConsumedMCPService(c *types.ConsumedMCPService) ([]byte, error) {
	contentsJSON, err := encodeConsumedMCPServiceContents(c)
	if err != nil {
		return nil, err
	}
	return serializeCustomBlobDocument(&customBlobInput{
		UnitID:             string(c.ID),
		ContainerID:        string(c.ContainerID),
		Name:               c.Name,
		Documentation:      c.Documentation,
		Excluded:           c.Excluded,
		ExportLevel:        c.ExportLevel,
		CustomDocumentType: types.CustomTypeConsumedMCPService,
		ReadableTypeName:   types.ReadableConsumedMCPService,
		ContentsJSON:       contentsJSON,
	})
}

// SerializeAgentEditorAgent returns canonical CustomBlobDocument BSON bytes
// for an agent-editor Agent unit. Tool / KB-tool entries without IDs are
// assigned fresh stable IDs, mirroring CreateAgentEditorAgent.
func SerializeAgentEditorAgent(a *types.Agent) ([]byte, error) {
	for i := range a.Tools {
		if a.Tools[i].ID == "" {
			a.Tools[i].ID = generateUUID()
		}
	}
	for i := range a.KBTools {
		if a.KBTools[i].ID == "" {
			a.KBTools[i].ID = generateUUID()
		}
	}
	contentsJSON, err := encodeAgentContents(a)
	if err != nil {
		return nil, err
	}
	return serializeCustomBlobDocument(&customBlobInput{
		UnitID:             string(a.ID),
		ContainerID:        string(a.ContainerID),
		Name:               a.Name,
		Documentation:      a.Documentation,
		Excluded:           a.Excluded,
		ExportLevel:        a.ExportLevel,
		CustomDocumentType: types.CustomTypeAgent,
		ReadableTypeName:   types.ReadableAgent,
		ContentsJSON:       contentsJSON,
	})
}

