// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// ConnectionBackend manages the lifecycle of a backend connection.
type ConnectionBackend interface {
	// Connect opens a connection to the project at path.
	Connect(path string) error
	// Disconnect closes the connection, finalizing any pending work.
	Disconnect() error
	// Commit flushes any pending writes. Implementations that auto-commit
	// (e.g. MprBackend) may treat this as a no-op.
	Commit() error
	// IsConnected reports whether the backend has an active connection.
	IsConnected() bool
	// Path returns the path of the connected project, or "" if not connected.
	Path() string
	// Version returns the MPR format version.
	Version() types.MPRVersion
	// ProjectVersion returns the Mendix project version.
	ProjectVersion() *types.ProjectVersion
	// GetMendixVersion returns the Mendix version string.
	// NOTE: uses Get prefix unlike Version()/ProjectVersion() for historical SDK compatibility.
	GetMendixVersion() (string, error)
}

// ModuleBackend provides module-level operations.
type ModuleBackend interface {
	// ListModules returns all modules in the project.
	ListModules() ([]*model.Module, error)
	// GetModule returns a module by ID.
	GetModule(id model.ID) (*model.Module, error)
	// GetModuleByName returns a module by name.
	GetModuleByName(name string) (*model.Module, error)
	// CreateModule adds a new module to the project.
	CreateModule(module *model.Module) error
	// UpdateModule persists changes to an existing module.
	UpdateModule(module *model.Module) error
	// DeleteModule removes a module by ID.
	DeleteModule(id model.ID) error
	// DeleteModuleWithCleanup removes a module and cleans up associated documents.
	DeleteModuleWithCleanup(id model.ID, moduleName string) error
}

// ModuleSettingsBackend provides access to Projects$ModuleSettings (Maven/JAR dependencies).
type ModuleSettingsBackend interface {
	// ListModuleSettings returns all module settings documents in the project.
	ListModuleSettings() ([]*types.ModuleSettings, error)
	// GetModuleSettings returns the settings for the given module.
	GetModuleSettings(moduleID model.ID) (*types.ModuleSettings, error)
	// UpdateModuleSettings persists changes to the module settings (incl. JarDependencies).
	UpdateModuleSettings(ms *types.ModuleSettings) error
}

// FolderBackend provides folder operations.
type FolderBackend interface {
	// ListFolders returns all folders in the project.
	ListFolders() ([]*types.FolderInfo, error)
	// CreateFolder adds a new folder.
	CreateFolder(folder *model.Folder) error
	// DeleteFolder removes a folder by ID.
	DeleteFolder(id model.ID) error
	// MoveFolder moves a folder to a new container.
	MoveFolder(id model.ID, newContainerID model.ID) error
}

// ScriptTransaction represents an open write transaction held for the
// duration of an EXECUTE SCRIPT block. The executor begins one at the
// root script invocation, commits it after the last statement succeeds,
// and rolls it back on any error so a failed script never leaves the
// project in a half-written state.
type ScriptTransaction interface {
	Commit() error
	Rollback() error
}

// ScriptTransactionBackend is implemented by backends that support
// atomic multi-statement script execution. Backends that cannot honour
// script-level atomicity (e.g. read-only or mock backends) may return a
// no-op ScriptTransaction.
type ScriptTransactionBackend interface {
	BeginScriptTransaction() (ScriptTransaction, error)
}

// ImportBuffer is a write-buffer handle returned by ImportBufferBackend.BeginImportBuffer.
// Flush commits all buffered units in a single transaction; Discard drops them.
type ImportBuffer interface {
	Flush() error
	Discard()
}

// ImportBufferBackend is optionally implemented by backends that support
// buffered import sessions for bulk-write performance.
// Executor code uses a type assertion: if bufBE, ok := ctx.Backend.(backend.ImportBufferBackend); ok { ... }
type ImportBufferBackend interface {
	BeginImportBuffer() ImportBuffer
	DisableImportBuffer()
}
