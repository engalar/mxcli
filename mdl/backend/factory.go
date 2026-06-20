// SPDX-License-Identifier: Apache-2.0

package backend

// BackendFactory is the construction-time factory that knows all role
// implementations. Business logic should depend on the narrowest role
// interface, not BackendFactory.
type BackendFactory interface {
	ConnectionBackend

	ModuleLister() ModuleLister
	ModuleWriter() ModuleWriter
	DomainModelReader() DomainModelReader
	DomainModelWriter() DomainModelWriter
	MicroflowReader() MicroflowReader
	MicroflowWriter() MicroflowWriter
	WorkflowReader() WorkflowReader
	WorkflowWriter() WorkflowWriter
	PageReader() PageReader
	PageWriter() PageWriter
	JavaActionReader() JavaActionReader
	JavaActionWriter() JavaActionWriter
	JavaScriptActionReader() JavaScriptActionReader
	JavaScriptActionWriter() JavaScriptActionWriter
	EnumerationReader() EnumerationReader
	EnumerationWriter() EnumerationWriter
	ConstantReader() ConstantReader
	ConstantWriter() ConstantWriter
	SettingsReader() SettingsReader
	SettingsWriter() SettingsWriter
	MappingReader() MappingReader
	MappingWriter() MappingWriter
	UnitReader() UnitReader
	UnitWriter() UnitWriter
	NavigationReader() NavigationReader
	NavigationWriter() NavigationWriter
	ImageCollectionWriter() ImageCollectionWriter
	ServiceLister() ServiceLister
	ServiceWriter() ServiceWriter
	ScheduledEventReader() ScheduledEventReader
	MetadataReader() MetadataReader
	FolderManager() FolderManager
	ModuleSettingsReader() ModuleSettingsReader
	ModuleSettingsWriter() ModuleSettingsWriter
	RenameManager() RenameManager
	SecurityProjectManager() SecurityProjectManager
	SecurityModuleManager() SecurityModuleManager
	SecurityEntityAccessManager() SecurityEntityAccessManager
	PageModelAccess() PageModelAccess
	PageMutationOperator() PageMutationOperator
	WorkflowMutationOperator() WorkflowMutationOperator
	WidgetBuilder() WidgetBuilder
	ScriptTransactionManager() ScriptTransactionManager
	AgentEditorOperator() AgentEditorOperator
}


