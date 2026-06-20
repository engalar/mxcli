// SPDX-License-Identifier: Apache-2.0

package backend

// FullBackend composes every domain backend into a single interface.
//
// Deprecated: Use BackendFactory for construction and narrow role
// interfaces (ModuleLister, MicroflowReader, etc.) for business logic.
// FullBackend and its sub-interfaces will be removed once all callers
// are migrated (Tasks 5, 7-9 of the SOLID refactoring).
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
