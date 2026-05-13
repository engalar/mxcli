// SPDX-License-Identifier: Apache-2.0

// NOTE: Lives in package mprbackend (same package as MprBackend) so it
// can be reached from existing call sites without a new import. It
// imports the new mprrepos sub-package by its full path.
package mprbackend

import (
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/repos"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// NewExecutorContext wires a single *mmpr.Writer through every Stage 2
// repository. The returned context is owned by the caller; closing the
// underlying Writer is the caller's responsibility.
//
// Stage 3 will replace this with a path-taking constructor that opens
// the Writer internally, mirroring spec Section 7.
func NewExecutorContext(w *mmpr.Writer) *repos.ExecutorContext {
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
