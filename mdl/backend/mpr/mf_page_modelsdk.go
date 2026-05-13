// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/microflows"
	"github.com/mendixlabs/mxcli/sdk/pages"
	"github.com/mendixlabs/mxcli/sdk/workflows"
)

// All Create*/Update* methods in this file produce canonical BSON via the
// existing sdk/mpr Serialize* helpers, then write the bytes through the
// modelsdk write path:
//   - Create*: msdkWriter.InsertUnit
//   - Update*: writeUnitContents (defined in write_helpers.go)
//
// This bypasses sdk/mpr's updateTransactionID(), which fails on hard-linked
// MPR files (SQLITE_READONLY_DBMOVED 1544).

// ── Microflow ─────────────────────────────────────────────────────────────

func (b *MprBackend) createMicroflowViaModelsdk(mf *microflows.Microflow) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if mf.ID == "" {
		mf.ID = model.ID(modelsdkmpr.GenerateID())
	}
	mf.TypeName = "Microflows$Microflow"
	contents, err := b.writer.SerializeMicroflow(mf)
	if err != nil {
		return fmt.Errorf("serialize microflow: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(mf.ID),
		string(mf.ContainerID),
		"Documents",
		"Microflows$Microflow",
		contents,
	)
}

func (b *MprBackend) updateMicroflowViaModelsdk(mf *microflows.Microflow) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeMicroflow(mf)
	if err != nil {
		return fmt.Errorf("serialize microflow: %w", err)
	}
	return b.writeUnitContents(mf.ID, contents)
}

// ── Nanoflow ──────────────────────────────────────────────────────────────

func (b *MprBackend) createNanoflowViaModelsdk(nf *microflows.Nanoflow) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if nf.ID == "" {
		nf.ID = model.ID(modelsdkmpr.GenerateID())
	}
	nf.TypeName = "Microflows$Nanoflow"
	contents, err := b.writer.SerializeNanoflow(nf)
	if err != nil {
		return fmt.Errorf("serialize nanoflow: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(nf.ID),
		string(nf.ContainerID),
		"Documents",
		"Microflows$Nanoflow",
		contents,
	)
}

func (b *MprBackend) updateNanoflowViaModelsdk(nf *microflows.Nanoflow) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeNanoflow(nf)
	if err != nil {
		return fmt.Errorf("serialize nanoflow: %w", err)
	}
	return b.writeUnitContents(nf.ID, contents)
}

// ── Page ──────────────────────────────────────────────────────────────────

func (b *MprBackend) createPageViaModelsdk(page *pages.Page) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if page.ID == "" {
		page.ID = model.ID(modelsdkmpr.GenerateID())
	}
	page.TypeName = "Forms$Page"
	contents, err := b.writer.SerializePage(page)
	if err != nil {
		return fmt.Errorf("serialize page: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(page.ID),
		string(page.ContainerID),
		"Documents",
		"Forms$Page",
		contents,
	)
}

func (b *MprBackend) updatePageViaModelsdk(page *pages.Page) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializePage(page)
	if err != nil {
		return fmt.Errorf("serialize page: %w", err)
	}
	return b.writeUnitContents(page.ID, contents)
}

// ── Layout ────────────────────────────────────────────────────────────────

func (b *MprBackend) createLayoutViaModelsdk(layout *pages.Layout) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if layout.ID == "" {
		layout.ID = model.ID(modelsdkmpr.GenerateID())
	}
	layout.TypeName = "Forms$Layout"
	contents, err := b.writer.SerializeLayout(layout)
	if err != nil {
		return fmt.Errorf("serialize layout: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(layout.ID),
		string(layout.ContainerID),
		"Documents",
		"Forms$Layout",
		contents,
	)
}

func (b *MprBackend) updateLayoutViaModelsdk(layout *pages.Layout) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeLayout(layout)
	if err != nil {
		return fmt.Errorf("serialize layout: %w", err)
	}
	return b.writeUnitContents(layout.ID, contents)
}

// ── Snippet ───────────────────────────────────────────────────────────────

func (b *MprBackend) createSnippetViaModelsdk(snippet *pages.Snippet) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if snippet.ID == "" {
		snippet.ID = model.ID(modelsdkmpr.GenerateID())
	}
	snippet.TypeName = "Forms$Snippet"
	contents, err := b.writer.SerializeSnippet(snippet)
	if err != nil {
		return fmt.Errorf("serialize snippet: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(snippet.ID),
		string(snippet.ContainerID),
		"Documents",
		"Forms$Snippet",
		contents,
	)
}

func (b *MprBackend) updateSnippetViaModelsdk(snippet *pages.Snippet) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeSnippet(snippet)
	if err != nil {
		return fmt.Errorf("serialize snippet: %w", err)
	}
	return b.writeUnitContents(snippet.ID, contents)
}

// ── Workflow ──────────────────────────────────────────────────────────────

func (b *MprBackend) createWorkflowViaModelsdk(wf *workflows.Workflow) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if wf.ID == "" {
		wf.ID = model.ID(modelsdkmpr.GenerateID())
	}
	wf.TypeName = "Workflows$Workflow"
	contents, err := b.writer.SerializeWorkflow(wf)
	if err != nil {
		return fmt.Errorf("serialize workflow: %w", err)
	}
	return b.msdkWriter.InsertUnit(
		string(wf.ID),
		string(wf.ContainerID),
		"Documents",
		"Workflows$Workflow",
		contents,
	)
}

func (b *MprBackend) updateWorkflowViaModelsdk(wf *workflows.Workflow) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeWorkflow(wf)
	if err != nil {
		return fmt.Errorf("serialize workflow: %w", err)
	}
	return b.writeUnitContents(wf.ID, contents)
}

// ── DomainModel ───────────────────────────────────────────────────────────

// updateDomainModelViaModelsdk re-serializes a mutated DomainModel via sdk/mpr
// (full polymorphic fidelity for AttributeType, Generalization, etc.) and
// commits through modelsdk WriteTransaction. Avoids sdk/mpr's
// updateTransactionID() which fails with SQLITE_READONLY_DBMOVED 1544 on
// hard-linked MPR files. The in-memory representation stays as
// *domainmodel.DomainModel — gen-type migration is a separate plan.
func (b *MprBackend) updateDomainModelViaModelsdk(dm *domainmodel.DomainModel) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := b.writer.SerializeDomainModel(dm)
	if err != nil {
		return fmt.Errorf("serialize domain model: %w", err)
	}
	wtx, err := b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := wtx.WriteUnit(string(dm.ID), contents); err != nil {
		_ = wtx.Rollback()
		return fmt.Errorf("write unit: %w", err)
	}
	if err := wtx.Commit(); err != nil {
		return err
	}
	b.reader.InvalidateCache()
	return nil
}
