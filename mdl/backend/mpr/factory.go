// SPDX-License-Identifier: Apache-2.0

// NOTE: Lives in package mprbackend (same package as MprBackend) so it
// can be reached from existing call sites without a new import. It
// imports the new mprrepos sub-package by its full path.
package mprbackend

import (
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/repos"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// NewExecutorContext wires a single *mmpr.Writer through every Stage 2/2.6
// repository. The returned context's Cascade service is always populated;
// References is left nil — callers needing rename/navigation/enum-ref
// updates should use NewExecutorContextWithReferences instead.
//
// Stage 3 will replace this with a path-taking constructor that opens
// the Writer internally, mirroring spec Section 7.
func NewExecutorContext(w *mmpr.Writer) *repos.ExecutorContext {
	ctx := newExecutorContextCommon(w)
	ctx.Cascade = mprrepos.NewCascadeService(w)
	return ctx
}

// NewExecutorContextWithReferences wires Cascade + References. The
// References service requires a *sdk/mpr.Writer for the BSON scanners
// (ScanRenameReferences, PatchNavigationProfile, ScanQualifiedNameUpdates);
// the modelsdk Writer handles all persistence. Both writers MUST share
// the same SQLite *sql.DB connection — see mprbackend.Wrap for the
// canonical pattern.
//
// Stage 4 cleanup will port the scanners themselves into mprrepos and
// drop the sdk/mpr dependency, collapsing this constructor back into
// NewExecutorContext.
func NewExecutorContextWithReferences(mw *mmpr.Writer, sdkW *sdkmpr.Writer) *repos.ExecutorContext {
	ctx := newExecutorContextCommon(mw)
	ctx.Cascade = mprrepos.NewCascadeService(mw)
	ctx.References = mprrepos.NewReferenceService(mw, sdkW)
	return ctx
}

func newExecutorContextCommon(w *mmpr.Writer) *repos.ExecutorContext {
	return &repos.ExecutorContext{
		Microflows: mprrepos.NewMicroflowRepository(w),
		Pages:      mprrepos.NewPageRepository(w),

		// Stage 2.6 — 15 domains landed via Plan T1-T15. ProjectSet
		// and ModuleSet share Settings$ProjectSettings and
		// Projects$ModuleSettings storage respectively.
		Nanoflows:    mprrepos.NewNanoflowRepository(w),
		Layouts:      mprrepos.NewLayoutRepository(w),
		Snippets:     mprrepos.NewSnippetRepository(w),
		DomainModels: mprrepos.NewDomainModelRepository(w),
		Modules:      mprrepos.NewModuleRepository(w),
		Enumerations: mprrepos.NewEnumerationRepository(w),
		Constants:    mprrepos.NewConstantRepository(w),
		Workflows:    mprrepos.NewWorkflowRepository(w),
		Services:     mprrepos.NewServiceRepository(w),
		Mappings:     mprrepos.NewMappingRepository(w),
		ProjectSet:   mprrepos.NewProjectSettingsRepository(w),
		ModuleSet:    mprrepos.NewModuleSettingsRepository(w),
		Security:     mprrepos.NewSecurityRepository(w),
		Folders:      mprrepos.NewFolderRepository(w),
		Images:       mprrepos.NewImageRepository(w),
		Agents:       mprrepos.NewAgentRepository(w),

		IDs:   mprrepos.NewIDGenerator(),
		Tx:    mprrepos.NewTransactionFactory(w),
		Names: mprrepos.NewQualifiedNameResolver(w),
		Cache: mprrepos.NewReaderCache(w),
	}
}
