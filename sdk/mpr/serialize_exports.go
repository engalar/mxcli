// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"github.com/mendixlabs/mxcli/model"
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
func (w *Writer) SerializeDomainModel(dm *domainmodel.DomainModel) ([]byte, error) {
	return w.serializeDomainModel(dm)
}
