// SPDX-License-Identifier: Apache-2.0

// Package model defines the canonical, in-memory representation of Mendix
// documents used by the MDL pipeline. It is decoupled from the parser AST and
// from the generated BSON-serialisable gen types: Lift converts an AST
// statement into a model; Hydrate builds a model from a gen-typed element;
// ToMDL serialises a model back into MDL text.
package model

// Document is any in-memory canonical representation of a Mendix document
// (entity, microflow, page, ...). Implementations must be able to serialise
// themselves back to MDL text.
type Document interface {
	ToMDL() string
}

// Persistable is a Document that can be written back to a project via a
// backend. POC-phase implementations may not yet implement Persist.
type Persistable interface {
	Document
	Persist(ctx PersistContext) error
}

// Warning is a non-fatal issue surfaced during Hydrate (e.g., an unexpected
// child type, a value that could not be round-tripped).
type Warning struct {
	Field   string
	Message string
}
