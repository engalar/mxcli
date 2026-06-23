// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// StmtHandler executes a single statement type.
type StmtHandler func(ctx context.Context, stmt ast.Statement) error

// Registry maps AST statement type names to their handler functions.
type Registry struct {
	handlers map[string]StmtHandler
}

// NewRegistry creates a Registry with all statement handlers registered.
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]StmtHandler),
	}
	return r
}

// RegisterFuture maps a type name to a handler. Overwrites any existing
// registration so that SetBackend() can replace minimal handlers with
// fully-configured ones.
func (r *Registry) RegisterFuture(typeName string, handler StmtHandler) {
	r.handlers[typeName] = handler
}

// Dispatch executes the handler for stmt.
func (r *Registry) Dispatch(ctx context.Context, stmt ast.Statement) error {
	h, ok := r.handlers[stmt.TypeName()]
	if !ok {
		return mdlerrors.NewUnsupported(fmt.Sprintf("unhandled statement type %s", stmt.TypeName()))
	}
	return h(ctx, stmt)
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
