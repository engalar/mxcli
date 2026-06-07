// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

// TestExecutorBoundary verifies that no executor file imports the deleted
// mdl/canonical/entity or mdl/canonical/association lifecycle sub-packages.
//
// These sub-packages were removed; their entity/association persist + render
// logic was ported into the executor itself (entity_from_ast.go,
// entity_mdl_render.go, assoc_mdl_render.go). This guard is a ratchet — if it
// fails after a refactor, the canonical lifecycle layer is being re-introduced.
// Do NOT add an allowlist entry; no file may import these packages.
func TestExecutorBoundary(t *testing.T) {
	archtest.Check(t, ".",
		archtest.NoImport{
			Forbidden: []string{
				"github.com/mendixlabs/mxcli/mdl/canonical/entity",
				"github.com/mendixlabs/mxcli/mdl/canonical/association",
			},
			Hint: `executor files must not import mdl/canonical/entity or
mdl/canonical/association — these lifecycle sub-packages were deleted.
The entity/association persist + render logic now lives in the executor:
  entity_from_ast.go    — buildEntityFromAST (was canonical/entity/persist.go)
  entity_mdl_render.go  — entity DESCRIBE rendering
  assoc_mdl_render.go   — association DESCRIBE rendering`,
		},
	)
}
