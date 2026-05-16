// SPDX-License-Identifier: Apache-2.0

// bsonReader abstracts the sdk/mpr.Reader for BSON tool commands.
// Consolidates the sdk/mpr dependency to this single file so that
// cmd_bson_*.go do not import sdk/mpr directly.
package main

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// bsonReader exposes the BSON inspection methods needed by cmd_bson_*.go.
// Satisfied by *sdkmpr.Reader; future implementations can use modelsdk/mpr.
type bsonReader interface {
	GetRawUnitByName(objectType, qualifiedName string) (*types.RawUnitInfo, error)
	ListRawUnits(objectType string) ([]*types.RawUnitInfo, error)
	Close() error
}

// openBSONReader opens a project at path and returns a bsonReader.
func openBSONReader(path string) (bsonReader, error) {
	return sdkmpr.Open(path)
}

// widgetReader exposes the widget inspection methods needed by cmd_extract_templates.go.
// Satisfied by *sdkmpr.Reader.
type widgetReader interface {
	GetMendixVersion() (string, error)
	ListAllCustomWidgetTypes() ([]*types.RawCustomWidgetType, error)
	Close() error
}

// openWidgetReader opens a project at path and returns a widgetReader.
func openWidgetReader(path string) (widgetReader, error) {
	return sdkmpr.Open(path)
}
