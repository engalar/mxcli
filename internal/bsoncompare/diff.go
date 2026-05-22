// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import "go.mongodb.org/mongo-driver/bson"

type DiffKind string

const (
	DiffChanged DiffKind = "changed"
	DiffAdded   DiffKind = "added"
	DiffRemoved DiffKind = "removed"
	DiffWarning DiffKind = "warning"
)

type FieldDiff struct {
	Path   string
	Golden string
	Actual string
	Kind   DiffKind
}

type UnitDiff struct {
	QualifiedName string
	UnitType      string
	Kind          DiffKind
	Fields        []FieldDiff
	// ActualDoc is the raw BSON document from the bPath (post-mutation) side.
	// Populated for DiffChanged and DiffAdded units; nil for DiffRemoved
	// (the unit no longer exists on the actual side).
	ActualDoc bson.D
}
