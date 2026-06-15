// SPDX-License-Identifier: Apache-2.0

package backend

// FullBackend composes every domain backend into a single interface.
// Implementations must satisfy all sub-interfaces.
//
// Role interfaces (e.g. ModuleLister, DomainModelReader) are embedded
// alongside the original domain sub-interfaces. Callers should prefer
// the narrowest role interface for dependency injection.
//
// FullBackend exists primarily as a construction-time constraint on
// backend implementations.
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
