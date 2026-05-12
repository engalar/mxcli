// SPDX-License-Identifier: Apache-2.0

package mpr

import "database/sql"

// UnitReader is the minimal read interface used by modelsdk-based backend writes.
type UnitReader interface {
	GetRawUnitBytes(unitID string) ([]byte, error)
	Version() MPRVersion
	ContentsDir() string
	DB() *sql.DB
	InvalidateCache()
}

// UnitWriter is the minimal write interface for modelsdk-based backend writes.
type UnitWriter interface {
	Reader() UnitReader
	BeginWriteTransaction() (*WriteTransaction, error)
	UpdateRawUnit(unitID string, contents []byte) error
	InsertUnit(unitID, containerID, containmentName, unitType string, contents []byte) error
	DeleteUnit(unitID string) error
	DeleteChildUnits(parentID string) error
	UpdateUnitContainer(unitID, newContainerID string) error
	Close() error
}

// Compile-time interface satisfaction checks.
var _ UnitReader = (*Reader)(nil)
var _ UnitWriter = (*Writer)(nil)
