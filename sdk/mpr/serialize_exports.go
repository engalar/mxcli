// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/agenteditor"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
	"github.com/mendixlabs/mxcli/sdk/microflows"
	"github.com/mendixlabs/mxcli/sdk/pages"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// Public Serialize* wrappers expose the existing private serialize* functions
// so that callers (e.g. the mprbackend modelsdk-write helpers) can produce
// canonical BSON bytes without going through Writer.updateUnit, which calls
// updateTransactionID() and fails on hard-linked MPR files
// (SQLITE_READONLY_DBMOVED 1544).
//
// These wrappers are intentionally thin — they exist only to widen the
// visibility of serialization functions that are otherwise correct.

// SerializeDatabaseConnection returns BSON bytes for a database connection unit.
func (w *Writer) SerializeDatabaseConnection(conn *model.DatabaseConnection) ([]byte, error) {
	return w.serializeDatabaseConnection(conn)
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
func (w *Writer) SerializeImportMapping(im *model.ImportMapping) ([]byte, error) {
	return w.serializeImportMapping(im)
}

// SerializeExportMapping returns BSON bytes for an export mapping unit.
func (w *Writer) SerializeExportMapping(em *model.ExportMapping) ([]byte, error) {
	return w.serializeExportMapping(em)
}

// SerializeBusinessEventService returns BSON bytes for a business event service unit.
func (w *Writer) SerializeBusinessEventService(svc *model.BusinessEventService) ([]byte, error) {
	return w.serializeBusinessEventService(svc)
}

// SerializeConsumedODataService returns BSON bytes for a consumed OData service unit.
func (w *Writer) SerializeConsumedODataService(svc *model.ConsumedODataService) ([]byte, error) {
	return w.serializeConsumedODataService(svc)
}

// SerializePublishedODataService returns BSON bytes for a published OData service unit.
func (w *Writer) SerializePublishedODataService(svc *model.PublishedODataService) ([]byte, error) {
	return w.serializePublishedODataService(svc)
}

// SerializeConsumedRestService returns BSON bytes for a consumed REST service unit.
func (w *Writer) SerializeConsumedRestService(svc *model.ConsumedRestService) ([]byte, error) {
	return w.serializeConsumedRestService(svc)
}

// SerializePublishedRestService returns BSON bytes for a published REST service unit.
func (w *Writer) SerializePublishedRestService(svc *model.PublishedRestService) ([]byte, error) {
	return w.serializePublishedRestService(svc)
}

// SerializeJavaAction returns BSON bytes for a Java action unit.
func (w *Writer) SerializeJavaAction(ja *javaactions.JavaAction) ([]byte, error) {
	return w.serializeJavaAction(ja)
}

// SerializeEnumeration returns BSON bytes for an enumeration unit.
func (w *Writer) SerializeEnumeration(enum *model.Enumeration) ([]byte, error) {
	return w.serializeEnumeration(enum)
}

// SerializeConstant returns BSON bytes for a constant unit.
func (w *Writer) SerializeConstant(constant *model.Constant) ([]byte, error) {
	return w.serializeConstant(constant)
}

// SerializeMicroflow returns BSON bytes for a microflow unit.
func (w *Writer) SerializeMicroflow(mf *microflows.Microflow) ([]byte, error) {
	return w.serializeMicroflow(mf)
}

// SerializeNanoflow returns BSON bytes for a nanoflow unit.
func (w *Writer) SerializeNanoflow(nf *microflows.Nanoflow) ([]byte, error) {
	return w.serializeNanoflow(nf)
}

// SerializePage returns BSON bytes for a page unit.
func (w *Writer) SerializePage(page *pages.Page) ([]byte, error) {
	return w.serializePage(page)
}

// SerializeLayout returns BSON bytes for a layout unit.
func (w *Writer) SerializeLayout(layout *pages.Layout) ([]byte, error) {
	return w.serializeLayout(layout)
}

// SerializeSnippet returns BSON bytes for a snippet unit.
func (w *Writer) SerializeSnippet(snippet *pages.Snippet) ([]byte, error) {
	return w.serializeSnippet(snippet)
}

// SerializeWorkflow returns BSON bytes for a workflow unit.
func (w *Writer) SerializeWorkflow(wf *workflows.Workflow) ([]byte, error) {
	return w.serializeWorkflow(wf)
}

// SerializeDomainModel returns BSON bytes for a domain model unit.
//
// Deprecated: prefer the package-level SerializeDomainModel(dm, moduleName, pv)
// which does not require a *Writer and decouples encoding from reader state.
// This receiver shim is kept for backwards compatibility and will be removed
// once all callers migrate (see retire-serialize-writer-receiver plan).
func (w *Writer) SerializeDomainModel(dm *domainmodel.DomainModel) ([]byte, error) {
	return w.serializeDomainModel(dm)
}

// SerializeProjectSettings returns BSON bytes for the project settings unit.
func (w *Writer) SerializeProjectSettings(ps *model.ProjectSettings) ([]byte, error) {
	return w.serializeProjectSettings(ps)
}

// SerializeModule returns BSON bytes for a module unit.
func (w *Writer) SerializeModule(module *model.Module) ([]byte, error) {
	return w.serializeModule(module)
}

// SerializeFolder returns BSON bytes for a folder unit.
func (w *Writer) SerializeFolder(folder *model.Folder) ([]byte, error) {
	return w.serializeFolder(folder)
}

// SerializeModuleSecurity returns BSON bytes for an empty module security unit.
func (w *Writer) SerializeModuleSecurity(id string) ([]byte, error) {
	return w.serializeModuleSecurity(id)
}

// SerializeModuleSettings returns BSON bytes for a default module settings unit.
func (w *Writer) SerializeModuleSettings(id string) ([]byte, error) {
	return w.serializeModuleSettings(id)
}

// SerializeAgentEditorModel returns canonical CustomBlobDocument BSON bytes for
// an agent-editor Model unit. The Provider defaults to "MxCloudGenAI" when
// unset, mirroring the validation in CreateAgentEditorModel.
func SerializeAgentEditorModel(m *agenteditor.Model) ([]byte, error) {
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
		CustomDocumentType: agenteditor.CustomTypeModel,
		ReadableTypeName:   agenteditor.ReadableModel,
		ContentsJSON:       contentsJSON,
	})
}

// SerializeAgentEditorKnowledgeBase returns canonical CustomBlobDocument BSON
// bytes for an agent-editor KnowledgeBase unit.
func SerializeAgentEditorKnowledgeBase(k *agenteditor.KnowledgeBase) ([]byte, error) {
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
		CustomDocumentType: agenteditor.CustomTypeKnowledgeBase,
		ReadableTypeName:   agenteditor.ReadableKnowledgeBase,
		ContentsJSON:       contentsJSON,
	})
}

// SerializeAgentEditorConsumedMCPService returns canonical CustomBlobDocument
// BSON bytes for an agent-editor Consumed MCP Service unit.
func SerializeAgentEditorConsumedMCPService(c *agenteditor.ConsumedMCPService) ([]byte, error) {
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
		CustomDocumentType: agenteditor.CustomTypeConsumedMCPService,
		ReadableTypeName:   agenteditor.ReadableConsumedMCPService,
		ContentsJSON:       contentsJSON,
	})
}

// SerializeAgentEditorAgent returns canonical CustomBlobDocument BSON bytes
// for an agent-editor Agent unit. Tool / KB-tool entries without IDs are
// assigned fresh stable IDs, mirroring CreateAgentEditorAgent.
func SerializeAgentEditorAgent(a *agenteditor.Agent) ([]byte, error) {
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
		CustomDocumentType: agenteditor.CustomTypeAgent,
		ReadableTypeName:   agenteditor.ReadableAgent,
		ContentsJSON:       contentsJSON,
	})
}

// GenerateJavaSource generates Java source code for a Java action.
// Exported so that callers outside sdk/mpr can produce Java source
// without going through the Writer (e.g., path-based file operations).
func GenerateJavaSource(moduleName, actionName string, userCode string, params []*javaactions.JavaActionParameter, returnType javaactions.CodeActionReturnType, extraImports []string, extraCode string) string {
	return generateJavaSource(moduleName, actionName, userCode, params, returnType, extraImports, extraCode)
}
