// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
)

// ── BackendFactory accessors ─────────────────────────────────────────────

func (b *MprBackend) ModuleLister() backend.ModuleLister                               { return b.modules }
func (b *MprBackend) ModuleWriter() backend.ModuleWriter                               { return b }
func (b *MprBackend) DomainModelReader() backend.DomainModelReader                     { return b }
func (b *MprBackend) DomainModelWriter() backend.DomainModelWriter                     { return b }
func (b *MprBackend) MicroflowWriter() backend.MicroflowWriter                         { return b }
func (b *MprBackend) WorkflowWriter() backend.WorkflowWriter                           { return b }
func (b *MprBackend) PageReader() backend.PageReader                                   { return b.pages }
func (b *MprBackend) PageWriter() backend.PageWriter                                   { return b }
func (b *MprBackend) JavaActionReader() backend.JavaActionReader                       { return b }
func (b *MprBackend) JavaActionWriter() backend.JavaActionWriter                       { return b }
func (b *MprBackend) JavaScriptActionReader() backend.JavaScriptActionReader           { return b.java }
func (b *MprBackend) JavaScriptActionWriter() backend.JavaScriptActionWriter           { return b }
func (b *MprBackend) EnumerationReader() backend.EnumerationReader                     { return b.enumerations }
func (b *MprBackend) EnumerationWriter() backend.EnumerationWriter                     { return b }
func (b *MprBackend) ConstantReader() backend.ConstantReader                           { return b.constants }
func (b *MprBackend) ConstantWriter() backend.ConstantWriter                           { return b }
func (b *MprBackend) SettingsReader() backend.SettingsReader                           { return b.settings }
func (b *MprBackend) SettingsWriter() backend.SettingsWriter                           { return b }
func (b *MprBackend) MappingReader() backend.MappingReader                             { return b }
func (b *MprBackend) MappingWriter() backend.MappingWriter                             { return b }
func (b *MprBackend) UnitReader() backend.UnitReader                                   { return b.rawUnits }
func (b *MprBackend) UnitWriter() backend.UnitWriter                                   { return b.rawUnits }
func (b *MprBackend) NavigationReader() backend.NavigationReader                       { return b.navigation }
func (b *MprBackend) NavigationWriter() backend.NavigationWriter                       { return b }
func (b *MprBackend) ImageCollectionWriter() backend.ImageCollectionWriter             { return b }
func (b *MprBackend) ServiceLister() backend.ServiceLister                             { return b }
func (b *MprBackend) ServiceWriter() backend.ServiceWriter                             { return b }
func (b *MprBackend) ScheduledEventReader() backend.ScheduledEventReader               { return b.scheduledEvents }
func (b *MprBackend) MetadataReader() backend.MetadataReader                           { return b.metadata }
func (b *MprBackend) FolderManager() backend.FolderManager                             { return b }
func (b *MprBackend) ModuleSettingsReader() backend.ModuleSettingsReader               { return b }
func (b *MprBackend) ModuleSettingsWriter() backend.ModuleSettingsWriter               { return b }
func (b *MprBackend) RenameManager() backend.RenameManager                             { return b }
func (b *MprBackend) SecurityProjectManager() backend.SecurityProjectManager           { return b }
func (b *MprBackend) SecurityModuleManager() backend.SecurityModuleManager             { return b }
func (b *MprBackend) SecurityEntityAccessManager() backend.SecurityEntityAccessManager { return b }
func (b *MprBackend) PageModelAccess() backend.PageModelAccess                         { return b }
func (b *MprBackend) PageMutationOperator() backend.PageMutationOperator               { return b }
func (b *MprBackend) WorkflowMutationOperator() backend.WorkflowMutationOperator       { return b }
func (b *MprBackend) WidgetBuilder() backend.WidgetBuilder                             { return b }
func (b *MprBackend) ScriptTransactionManager() backend.ScriptTransactionManager       { return b }
func (b *MprBackend) AgentEditorOperator() backend.AgentEditorOperator                 { return b }
