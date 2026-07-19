// SPDX-License-Identifier: Apache-2.0

package golden

import (
	_ "embed"
)

//go:embed testdata/MyFirstModule.Nanoflow.mxunit
var nanoflowBSON []byte

// Registry returns all registered golden entries.
func Registry() []GoldenEntry {
	return []GoldenEntry{
		{
			Name:      "Nanoflow",
			Source:    "Studio Pro 11.12.1 — MyFirstModule.Nanoflow",
			BSON:      nanoflowBSON,
			Builder:   BuildNanoflow,
			SkipFields: nil, // $ID is auto-skipped
		},
	}
}
