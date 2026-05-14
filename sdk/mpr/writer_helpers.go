// SPDX-License-Identifier: Apache-2.0

// Package mpr — writer-side cross-domain helpers.
//
// Counterpart to parser.go (the read-side helpers home). Helpers here
// are shared across multiple writer_*.go files; per-domain helpers stay
// in their own writer_<domain>.go file.

package mpr

import (
	"go.mongodb.org/mongo-driver/bson"
)

// stringOrDefault returns value if non-empty, otherwise defaultValue.
//
// Cross-domain helper preserved during Stage 4 S5 (microflow writer
// retire). Originally lived in writer_microflow.go; consumed by
// writer_javaactions.go.
func stringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// emptyTextTemplate returns an empty Microflows$TextTemplate embedded
// BSON document.
//
// Cross-domain Mendix BSON schema quirk: TitleOverride on
// Forms$FormSettings (and similar fields on widget action templates) is
// typed as Microflows$TextTemplate even though the consumer is not a
// microflow. The field must be written as an empty object rather than
// nil — same pattern as emptyPageVariable() for Forms$PageVariable
// (see PR #338 / issue #295).
//
// Cross-domain helper preserved during Stage 4 S5 (microflow writer
// retire). Originally lived in writer_microflow_actions.go; consumed by
// writer_navigation.go and writer_widgets_action.go.
func emptyTextTemplate() bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "Microflows$TextTemplate"},
		{Key: "Parameters", Value: bson.A{int32(2)}},
		{Key: "Text", Value: bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: bson.A{int32(2)}},
		}},
	}
}
