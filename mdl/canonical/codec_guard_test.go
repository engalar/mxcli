// SPDX-License-Identifier: Apache-2.0

package canonical_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/mdl/canonical/entity"
)

// TestCodecComplete verifies that all registered gen TypeNames have complete
// codecs (non-nil LiftFn and HydrateFn).
//
// GREEN baseline: entity codecs are complete (both fns non-nil).
// Turns RED if: a new domain registers a codec with a nil function, or
// a Required TypeName is removed from the registry without updating this file.
//
// When adding a new domain (Phase 2+):
//   1. Add: domainpkg.RegisterCodec(r) to BuildRegistry below.
//   2. Add: "DomainModels$NewType" to Required.
//   Both must be updated together — a Required entry with no RegisterCodec
//   call will immediately fail this test.
func TestCodecComplete(t *testing.T) {
	archtest.Check(t, ".",
		archtest.CodecComplete{
			BuildRegistry: func() *canonical.DefaultRegistry {
				r := canonical.NewDefaultRegistry()
				entity.RegisterCodec(r)
				// Phase 2: association.RegisterCodec(r)
				// Phase 3: microflow.RegisterCodec(r)
				return r
			},
			Required: []string{
				"DomainModels$Entity",
				"DomainModels$EntityImpl",
				// Phase 2: "DomainModels$Association"
			},
			Hint: `Every Required gen TypeName must be registered with non-nil LiftFn and HydrateFn.
Fix:
  1. Confirm domain/codec.go calls r.Register(...) or r.RegisterGenType(...) for all listed TypeNames.
  2. LiftFn  — func(stmt any) (Persistable, error): converts AST stmt to canonical Model.
  3. HydrateFn — func(el any) (Document, []Warning, error): converts gen element to canonical Model.
     Note: current HydrateFn passes "" as moduleName — a known gap tracked in the spec.
     A future plan will upgrade HydrateFn to accept HydrateCtx{ModuleName string}.
  4. When adding a new domain: add RegisterCodec call to BuildRegistry AND
     add TypeName(s) to Required. Both must be updated together.`,
		},
	)
}
