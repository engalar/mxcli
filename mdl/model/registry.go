// SPDX-License-Identifier: Apache-2.0

package model

import (
	"fmt"
	"reflect"
)

// Codec wires Lift (AST -> canonical model) and Hydrate (gen-typed element ->
// canonical model) for a single document kind.
type Codec struct {
	LiftFn    func(stmt any) (Persistable, error)
	HydrateFn func(el any) (Document, []Warning, error)
}

// DefaultRegistry is the in-memory codec lookup keyed by AST statement type
// (for Lift) and by gen TypeName (for Hydrate).
type DefaultRegistry struct {
	byStmtType map[reflect.Type]Codec
	byGenType  map[string]Codec
}

// NewDefaultRegistry returns an empty registry.
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		byStmtType: make(map[reflect.Type]Codec),
		byGenType:  make(map[string]Codec),
	}
}

// Register binds a codec to both an AST statement type (matched via reflect)
// and a gen TypeName (the string returned by gen types' TypeName() method,
// e.g. "DomainModels$Entity").
func (r *DefaultRegistry) Register(stmtExample any, genTypeName string, c Codec) {
	r.byStmtType[reflect.TypeOf(stmtExample)] = c
	r.byGenType[genTypeName] = c
}

// RegisterGenType binds an additional gen TypeName to an existing Codec.
// Use when multiple TypeName stamps on disk (e.g. "DomainModels$Entity" and
// "DomainModels$EntityImpl") should dispatch to the same Hydrate function.
func (r *DefaultRegistry) RegisterGenType(genTypeName string, c Codec) {
	r.byGenType[genTypeName] = c
}

type genTyper interface{ TypeName() string }

// LiftFrom dispatches the AST statement to the registered Lift codec.
func (r *DefaultRegistry) LiftFrom(stmt any) (Persistable, error) {
	c, ok := r.byStmtType[reflect.TypeOf(stmt)]
	if !ok {
		return nil, fmt.Errorf("model: no codec for %T", stmt)
	}
	return c.LiftFn(stmt)
}

// HydrateFrom dispatches a gen-typed element to the registered Hydrate codec.
func (r *DefaultRegistry) HydrateFrom(el any) (Document, []Warning, error) {
	gt, ok := el.(genTyper)
	if !ok {
		return nil, nil, fmt.Errorf("model: %T does not implement TypeName()", el)
	}
	c, ok := r.byGenType[gt.TypeName()]
	if !ok {
		return nil, nil, fmt.Errorf("model: no codec for gen type %q", gt.TypeName())
	}
	return c.HydrateFn(el)
}
