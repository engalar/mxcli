// SPDX-License-Identifier: Apache-2.0

package canonical_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
)

// TestImportDirection verifies that the mdl/canonical root package — the stable
// canonical center — does not import volatile layers (AST, gen types, backend).
//
// RED state: context.go imports "github.com/mendixlabs/mxcli/mdl/backend".
// GREEN when: PersistContext.Backend is replaced by a Writer interface defined
// in this package (see Hint below).
func TestImportDirection(t *testing.T) {
	archtest.Check(t, ".",
		archtest.NoImport{
			Forbidden: []string{
				"github.com/mendixlabs/mxcli/mdl/ast",
				"github.com/mendixlabs/mxcli/modelsdk/gen/",
				"github.com/mendixlabs/mxcli/mdl/backend",
			},
			Hint: `mdl/canonical/ is the stable center.
It must not depend on any volatile layer (AST parser, gen types, backend).
Fix for context.go:
  1. Define a Writer interface in this package:
       type Writer interface {
           CreateDoc(domainModelID canonical.ID, doc Persistable) error
           UpdateDoc(existingID canonical.ID, domainModelID canonical.ID, doc Persistable) error
       }
  2. Change PersistContext.Backend field type from backend.DomainModelBackend to Writer.
  3. backend/mpr's MprBackend implements Writer; executor passes it in.
  4. Remove the "mdl/backend" import from context.go.`,
		},
	)
}
