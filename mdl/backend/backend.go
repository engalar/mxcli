// SPDX-License-Identifier: Apache-2.0

package backend

// FullBackend composes every domain backend into a single interface.
//
// Deprecated: Use BackendFactory for construction and narrow role
// interfaces (ModuleLister, MicroflowReader, etc.) for business logic.
//
// Migration status (2026-07):
//   - HandlerDeps no longer carries this interface (Task 5, 2026-07-20).
//   - ExecContext.Backend still exists for executor_connect.go runtime
//     and the initRoles mock fallback (Task 6 deferred — more callers
//     than estimated).
//   - Executor.Backend() is deprecated — use Executor.LintReader()
//     for lint/report contexts.
//   - flowbuilder_v2 and pageBuilder carry their own deprecated
//     backend.FullBackend fields for backward compat.
//   - ~80 callers remain across executor, expr/meta, cmd binaries.
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
	MicroflowWriter
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
