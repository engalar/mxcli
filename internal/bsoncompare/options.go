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

var builtinIgnore = map[string]bool{
	"$ID":                      true,
	"DestinationControlVector": true,
	"OriginControlVector":      true,
	"ControlVector":            true,
	"PositionX":                true,
	"PositionY":                true,
	"RelativeMiddlePoint":      true,
}

func shouldIgnore(fieldName string, opts Options) bool {
	if builtinIgnore[fieldName] {
		return true
	}
	if opts.IgnoreLayout {
		switch fieldName {
		case "CanvasHeight", "CanvasWidth":
			return true
		}
	}
	if opts.IgnoreDocumentation && fieldName == "Documentation" {
		return true
	}
	if opts.IgnoreStableId && fieldName == "StableId" {
		return true
	}
	for _, f := range opts.IgnoreFields {
		if f == fieldName {
			return true
		}
	}
	return false
}
