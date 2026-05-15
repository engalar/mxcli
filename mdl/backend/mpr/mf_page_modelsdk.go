// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// updateDomainModelViaModelsdk produces canonical BSON via sdk/mpr's
// SerializeDomainModel, then commits the bytes through the modelsdk
// write path. This bypasses sdk/mpr's updateTransactionID(), which fails
// on hard-linked MPR files (SQLITE_READONLY_DBMOVED 1544).
//
// Microflow / Nanoflow Create+Update helpers were retired in
// Followup E6 — production routes through the modelsdk-native repos
// (mdl/backend/mpr/repos/microflow_writer.go) directly.

// Page / Layout / Snippet create/update helpers retired in Stage 3.3.5.E1;
// the gen-typed path (mprrepos.NewPageRepository(...).Create / .Update,
// etc.) is the only production write surface. The V3 page builder still
// emits sdk-typed structs which are converted through the
// SDKPageToGen / SDKSnippetToGen bridges (page_bridge.go) before hitting
// the gen writer.
//
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
