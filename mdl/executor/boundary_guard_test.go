// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

// TestExecutorBoundary verifies that cmd_*.go files do not import
// mdl/canonical/entity (or any future mdl/canonical/{domain} subpackage) directly.
// All canonical model operations must go through ctx.ModelCodecs dispatch.
//
// RED state: cmd_entities_gen.go and cmd_diff_mdl.go import mdl/canonical/entity
// and call entity.HydrateWithModule() directly, bypassing the codec registry.
//
// GREEN when: describe and create paths route through ctx.ModelCodecs.HydrateFrom()
// and ctx.ModelCodecs.LiftFrom() respectively. executor.go (RegisterCodec site)
// remains in the Allowlist.
func TestExecutorBoundary(t *testing.T) {
	archtest.Check(t, ".",
		archtest.NoImport{
			Forbidden: []string{
				"github.com/mendixlabs/mxcli/mdl/canonical/entity",
				// Phase 2: "github.com/mendixlabs/mxcli/mdl/canonical/association"
				// Phase 3: "github.com/mendixlabs/mxcli/mdl/canonical/microflow"
			},
			Allowlist: map[string]bool{
				// executor.go is the sole file permitted to import domain subpackages.
				// It uses them only to call RegisterCodec(). All other files must
				// access canonical models through ctx.ModelCodecs.
				"executor.go": true,
			},
			Hint: `cmd_*.go files must not import mdl/canonical/{domain}/ subpackages directly.
Fix for cmd_entities_gen.go (describe path):
  Replace:
    m, warns, err := entityModel.HydrateWithModule(modName, entity)
  With:
    doc, warns, err := ctx.ModelCodecs.HydrateFrom(entity)
    m := doc.(*entitypkg.EntityModel)  // type-assert only if domain fields needed
    _ = modName                        // modName must be overlaid on m.Name.Module

Fix for cmd_diff_mdl.go:
  Replace direct entityModel.Lift(s) / entityModel.HydrateWithModule calls
  with ctx.ModelCodecs.LiftFrom(s) and ctx.ModelCodecs.HydrateFrom(el).

The codec registry (executor.go) is the ONLY file that may import domain subpackages.`,
		},
	)
}
