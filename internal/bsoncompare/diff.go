// SPDX-License-Identifier: Apache-2.0
package bsoncompare

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
}
