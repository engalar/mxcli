// SPDX-License-Identifier: Apache-2.0

package backend

// FullBackend composes every domain backend into a single interface.
//
// Deprecated: Use BackendFactory for construction and narrow role
// interfaces (ModuleLister, MicroflowReader, etc.) for business logic.
//
// Migration status (2026-06):
//   - ExecContext initRoles() prefers backendFactory; FullBackend is
//     the fallback only (mock backends without BackendFactory).
//   - Executor.Backend() is deprecated — use Executor.LintReader()
//     for lint/report contexts.
//   - flowbuilder_v2 and pageBuilder carry their own deprecated
//     backend.FullBackend fields for backward compat.
//   - ~100 callers remain across executor, expr/meta, cmd binaries.
//     Phase removal is tracked in REFACTOR_PLAN.md.
type FullBackend interface {
	ConnectionBackend
	ModuleBackend
	ModuleSettingsBackend
	FolderBackend
	DomainModelBackend
	MicroflowBackend
	PageBackend
	PageModelBackend
	EnumerationBackend
	ConstantBackend
	SecurityBackend
	NavigationBackend
	ServiceBackend
	MappingBackend
	JavaBackend
	JavaScriptBackend
	WorkflowBackend
	SettingsBackend
	ImageBackend
	ScheduledEventBackend
	RenameBackend
	RawUnitBackend
	MetadataBackend
	WidgetBackend
	AgentEditorBackend
	PageMutationBackend
	WorkflowMutationBackend
	WidgetSerializationBackend
	WidgetBuilderBackend
	ScriptTransactionBackend

	// Role interfaces (segregated by responsibility)
	ModuleLister
	ModuleWriter
	DomainModelReader
	DomainModelWriter
	MicroflowReader
	MicroflowWriter
	WorkflowReader
	WorkflowWriter
	JavaActionReader
	JavaActionWriter
	JavaScriptActionReader
	JavaScriptActionWriter
	EnumerationReader
	EnumerationWriter
	ConstantReader
	ConstantWriter
	SettingsReader
	SettingsWriter
	MappingReader
	MappingWriter
	UnitReader
	UnitWriter
	NavigationReader
	NavigationWriter
	ImageCollectionWriter
	ServiceLister
	ServiceWriter
	ScheduledEventReader
	MetadataReader
	WidgetInspector
	ConnectionManager
	FolderManager
	ModuleSettingsReader
	ModuleSettingsWriter
	RenameManager
	SecurityProjectManager
	SecurityModuleManager
	SecurityEntityAccessManager
	PageModelAccess
	PageMutationOperator
	WorkflowMutationOperator
	WidgetBuilder
	ScriptTransactionManager
	AgentEditorOperator
}
