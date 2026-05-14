// SPDX-License-Identifier: Apache-2.0

// Package mpr — parser-side cross-domain helpers.
//
// Counterpart to writer_helpers.go (the write-side helpers home). Helpers
// here are shared across multiple parser_*.go files; per-domain helpers
// stay in their own parser_<domain>.go file.

package mpr

import (
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/model"
)

// parsePoint extracts a model.Point from raw BSON data.
//
// Cross-domain Mendix BSON shape quirk: positions are stored either as
// {X: int, Y: int} maps (MPR v1) or as "X;Y" formatted strings (MPR v2).
// Both forms decode to the same model.Point.
//
// Cross-domain helper preserved during Stage 4 S6 (microflow parser
// retire). Originally lived in parser_microflow.go; consumed by
// parser_unknown.go for any element with a RelativeMiddlePoint field.
func parsePoint(raw any) model.Point {
	switch v := raw.(type) {
	case map[string]any:
		return model.Point{
			X: extractInt(v["X"]),
			Y: extractInt(v["Y"]),
		}
	case string:
		parts := strings.SplitN(v, ";", 2)
		if len(parts) == 2 {
			x, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			y, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			return model.Point{X: x, Y: y}
		}
	}
	return model.Point{}
}
