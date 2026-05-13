// SPDX-License-Identifier: Apache-2.0

package repos

// ExecutorContext aggregates every interface a Stage 3 executor handler
// needs. Each field is an interface; tests inject mocks from
// mdl/repos/testing.
//
// Stage 2 only populates the wired fields (Microflows, Pages, IDs, Tx,
// Names, Cache); the remaining Repository fields will be wired in
// Stage 3 as their implementations come online. Until then they are
// declared so handler code can compile against the final shape.
//
// Spec Section 6 places this type in mdl/executor/context.go. Stage 2
// keeps it under mdl/repos to avoid forcing mdl/executor to depend on
// mdl/repos while the legacy executor.ExecContext path remains
// untouched. Stage 3 will move/rename as part of the cutover.
type ExecutorContext struct {
	Microflows MicroflowRepository
	Pages      PageRepository

	// Stage 2.6 wired domains (15 fields, one per stub repo).
	// Settings is split into ProjectSet (singleton) + ModuleSet
	// (per-module) — same shape as the underlying interface split.
	Nanoflows    NanoflowRepository
	Layouts      LayoutRepository
	Snippets     SnippetRepository
	DomainModels DomainModelRepository
	Modules      ModuleRepository
	Enumerations EnumerationRepository
	Constants    ConstantRepository
	Workflows    WorkflowRepository
	Services     ServiceRepository
	Mappings     MappingRepository
	ProjectSet   ProjectSettingsRepository
	ModuleSet    ModuleSettingsRepository
	Security     SecurityRepository
	Folders      FolderRepository
	Images       ImageRepository
	Agents       AgentRepository

	IDs   IDGenerator
	Tx    TransactionFactory
	Names QualifiedNameResolver
	Cache ReaderCache
}
