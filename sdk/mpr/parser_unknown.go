// SPDX-License-Identifier: Apache-2.0

package mpr

import "github.com/mendixlabs/mxcli/model"

// newUnknownObject creates an UnknownElement that preserves raw BSON fields
// for unrecognized $Type values, preventing silent data loss.
func newUnknownObject(typeName string, raw map[string]any) *model.UnknownElement {
	id := ""
	if raw != nil {
		id = extractBsonID(raw["$ID"])
	}
	elem := &model.UnknownElement{
		BaseElement: model.BaseElement{ID: model.ID(id), TypeName: typeName},
		RawFields:   raw,
	}
	if raw != nil {
		elem.Position = parsePoint(raw["RelativeMiddlePoint"])
		elem.Name = extractString(raw["Name"])
		elem.Caption = extractString(raw["Caption"])
	}
	return elem
}
