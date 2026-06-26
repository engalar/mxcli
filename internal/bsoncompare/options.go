// SPDX-License-Identifier: Apache-2.0
package bsoncompare

type Options struct {
	IgnoreFields        []string
	IgnoreDocumentation bool
	IgnoreLayout        bool
	IgnoreStableId      bool
}

func DefaultOptions() Options {
	return Options{
		IgnoreDocumentation: true,
		IgnoreLayout:        true,
		IgnoreStableId:      true,
	}
}

// shouldIgnore checks if a BSON field should be excluded from comparison.
//
// Hot path — kept as a single switch + O(n) scan to maximise inlining
// potential for the common cases.
func shouldIgnore(fieldName string, opts Options) bool {
	switch fieldName {
	case "$ID", "DestinationControlVector", "OriginControlVector",
		"ControlVector", "PositionX", "PositionY", "RelativeMiddlePoint":
		return true
	case "CanvasHeight", "CanvasWidth":
		return opts.IgnoreLayout
	case "Documentation":
		return opts.IgnoreDocumentation
	case "StableId":
		return opts.IgnoreStableId
	}
	for _, f := range opts.IgnoreFields {
		if f == fieldName {
			return true
		}
	}
	return false
}
