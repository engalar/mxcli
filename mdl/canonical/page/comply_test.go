// SPDX-License-Identifier: Apache-2.0

package page_test

import (
	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/mdl/canonical/page"
)

// Compile-time interface assertion. The Persistable assertion is added in the
// Persist task (CM-4) once the stub method exists.
var _ canonical.Document = (*page.PageDocument)(nil)
