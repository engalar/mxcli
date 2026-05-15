// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/model"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// All Create*/Update* methods in this file produce canonical BSON via the
// existing sdk/mpr Serialize* helpers, then write the bytes through the
// modelsdk write path:
//   - Create*: msdkWriter.InsertUnit
//   - Update*: writeUnitContents (defined in write_helpers.go)
//
// This bypasses sdk/mpr's updateTransactionID(), which fails on hard-linked
// MPR files (SQLITE_READONLY_DBMOVED 1544).

// Microflow / Nanoflow Create+Update helpers were retired in
// Followup E6 — production routes through the modelsdk-native repos
// (mdl/backend/mpr/repos/microflow_writer.go) directly. Pages,
// layouts, snippets, and workflows remain on this sdk-typed path.

// ── Page ──────────────────────────────────────────────────────────────────

func (b *MprBackend) createPageViaModelsdk(page *backend.Page) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if page.ID == "" {
		page.ID = model.ID(modelsdkmpr.GenerateID())
	}
	page.TypeName = "Forms$Page"
	contents, err := mpr.SerializePage(page)
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

func (b *MprBackend) updatePageViaModelsdk(page *backend.Page) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := mpr.SerializePage(page)
	if err != nil {
		return fmt.Errorf("serialize page: %w", err)
	}
	return b.writeUnitContents(page.ID, contents)
}

// ── Layout ────────────────────────────────────────────────────────────────

func (b *MprBackend) createLayoutViaModelsdk(layout *backend.Layout) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if layout.ID == "" {
		layout.ID = model.ID(modelsdkmpr.GenerateID())
	}
	layout.TypeName = "Forms$Layout"
	contents, err := mpr.SerializeLayout(layout)
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

func (b *MprBackend) updateLayoutViaModelsdk(layout *backend.Layout) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := mpr.SerializeLayout(layout)
	if err != nil {
		return fmt.Errorf("serialize layout: %w", err)
	}
	return b.writeUnitContents(layout.ID, contents)
}

// ── Snippet ───────────────────────────────────────────────────────────────

func (b *MprBackend) createSnippetViaModelsdk(snippet *backend.Snippet) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	if snippet.ID == "" {
		snippet.ID = model.ID(modelsdkmpr.GenerateID())
	}
	snippet.TypeName = "Forms$Snippet"
	contents, err := mpr.SerializeSnippet(snippet)
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

func (b *MprBackend) updateSnippetViaModelsdk(snippet *backend.Snippet) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	contents, err := mpr.SerializeSnippet(snippet)
	if err != nil {
		return fmt.Errorf("serialize snippet: %w", err)
	}
	return b.writeUnitContents(snippet.ID, contents)
}

// Workflow create/update helpers retired in Stage 3.3.3.E1; the gen-typed
// path (mprrepos.NewWorkflowRepository(...).Create / .Update) is the only
// production write surface.

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
	// Module name is best-effort: used for qualified names in validation rules.
	// Mirrors the soft-lookup behavior of the legacy (*Writer).serializeDomainModel.
	moduleName := ""
	if dm.ContainerID != "" {
		if module, mErr := b.reader.GetModule(dm.ContainerID); mErr == nil && module != nil {
			moduleName = module.Name
		}
	}
	contents, err := mpr.SerializeDomainModel(dm, moduleName, b.reader.ProjectVersion())
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
