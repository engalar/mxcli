// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// StmtHandler executes a single statement type.
type StmtHandler func(ctx *ExecContext, stmt ast.Statement) error

// StmtHandlerFunc is the new-style handler signature for handlers that
// have been migrated away from *ExecContext.
type StmtHandlerFunc func(ctx context.Context, stmt ast.Statement) error

// Registry maps AST statement type names to their handler functions.
type Registry struct {
	handlers    map[string]StmtHandler
	futureFuncs map[string]StmtHandlerFunc
}

// NewRegistry creates a Registry with all statement handlers registered.
func NewRegistry() *Registry {
	r := &Registry{
		handlers:    make(map[string]StmtHandler),
		futureFuncs: make(map[string]StmtHandlerFunc),
	}
	registerConnectionHandlers(r)
	registerModuleHandlers(r)
	registerEnumerationHandlers(r)
	registerConstantHandlers(r)
	registerDatabaseConnectionHandlers(r)
	registerEntityHandlers(r)
	registerAssociationHandlers(r)
	registerMicroflowAndNanoflowHandlers(r)
	registerPageHandlers(r)
	registerSecurityHandlers(r)
	registerNavigationHandlers(r)
	registerImageHandlers(r)
	registerWorkflowHandlers(r)
	registerBusinessEventHandlers(r)
	registerSettingsHandlers(r)
	registerODataHandlers(r)
	registerJSONStructureHandlers(r)
	registerMappingHandlers(r)
	registerRESTHandlers(r)
	registerDataTransformerHandlers(r)
	registerQueryHandlers(r)
	registerStylingHandlers(r)
	registerThemeCommandHandlers(r)
	registerRepositoryHandlers(r)
	registerSessionHandlers(r)
	registerLintHandlers(r)
	registerAlterPageHandlers(r)
	registerFragmentHandlers(r)
	registerSQLHandlers(r)
	registerImportHandlers(r)
	registerAgentEditorHandlers(r)
	return r
}

// Register maps a statement type to its handler using reflect.TypeOf for
// backward-compatible key generation.
func (r *Registry) Register(stmt ast.Statement, handler StmtHandler) {
	key := stmt.TypeName()
	if _, exists := r.handlers[key]; !exists {
		r.handlers[key] = handler
	}
}

// RegisterByName maps a type name to its handler.
func (r *Registry) RegisterByName(typeName string, handler StmtHandler) {
	if _, exists := r.handlers[typeName]; !exists {
		r.handlers[typeName] = handler
	}
}

// RegisterFuture maps a type name to a new-style handler (no *ExecContext).
// Used during the transition from *ExecContext to context.Context.
func (r *Registry) RegisterFuture(typeName string, handler StmtHandlerFunc) {
	if _, exists := r.futureFuncs[typeName]; !exists {
		r.futureFuncs[typeName] = handler
	}
}

// Lookup returns the handler for the given statement, or nil if none is registered.
func (r *Registry) Lookup(stmt ast.Statement) StmtHandler {
	return r.handlers[stmt.TypeName()]
}

// Dispatch finds and executes the handler for stmt.
// Prefers new-style StmtHandlerFunc (no ExecContext dependency) when available.
func (r *Registry) Dispatch(ctx *ExecContext, stmt ast.Statement) error {
	ctx.initRoles()
	// New-style handler registered via RegisterFuture — receives ctx as context.Context.
	if fn, ok := r.futureFuncs[stmt.TypeName()]; ok {
		return fn(ctx, stmt)
	}
	// Old-style handler with *ExecContext.
	h := r.Lookup(stmt)
	if h == nil {
		return mdlerrors.NewUnsupported(fmt.Sprintf("unhandled statement type %s", stmt.TypeName()))
	}
	return h(ctx, stmt)
}

// Validate checks that every known AST statement type has a registered handler.
func (r *Registry) Validate(knownTypes []ast.Statement) error {
	var missing []string
	for _, s := range knownTypes {
		if _, ok := r.handlers[s.TypeName()]; !ok {
			missing = append(missing, s.TypeName())
		}
	}
	if len(missing) > 0 {
		return mdlerrors.NewValidationf("registry: %d unregistered statement type(s): %v", len(missing), missing)
	}
	return nil
}

// HandlerCount returns the number of registered handlers.
func (r *Registry) HandlerCount() int {
	return len(r.handlers)
}

// RegisteredTypes returns all registered type names for testing.
func (r *Registry) RegisteredTypes() []string {
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// HasHandler returns true if a handler is registered for the given type.
func (r *Registry) HasHandler(stmt ast.Statement) bool {
	_, ok := r.handlers[stmt.TypeName()]
	return ok
}
