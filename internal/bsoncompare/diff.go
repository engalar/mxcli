// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import "github.com/mendixlabs/mxcli/modelsdk/element"

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
	ActualDoc     element.Element
}
