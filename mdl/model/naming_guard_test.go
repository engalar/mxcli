// SPDX-License-Identifier: Apache-2.0

package model_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

// TestPackageStructure verifies that mdl/model/ (future: mdl/canonical/) uses
// a flat one-level subpackage structure where each subdirectory name matches
// its declared package name.
//
// GREEN baseline: entity/ and layout/ both satisfy the constraints.
// Turns RED if:
//   - A new domain creates nested subdirectories (e.g. entity/conversion/)
//   - A subdirectory declares a package name that does not match its dirname
//
// Rule: add files within the same directory to split large files; do not
// create sub-subdirectories.
func TestPackageStructure(t *testing.T) {
	archtest.Check(t, ".",
		archtest.PackageName{
			MaxDepth:       1,
			NameMatchesDir: true,
			Hint: `mdl/model/ allows exactly one level of subpackages (entity/, association/, etc.).
Do not create nested subdirectories inside domain packages (e.g. entity/conversion/).
Reason: nesting makes import paths verbose and cross-domain references complex.
To split a large file: add files in the same directory with the same package name.
Example: entity/lift.go, entity/hydrate.go, entity/persist.go — all "package entity".`,
		},
	)
}
