// SPDX-License-Identifier: Apache-2.0

package association_test

import (
	"github.com/mendixlabs/mxcli/mdl/canonical"
	"github.com/mendixlabs/mxcli/mdl/canonical/association"
)

// Compile-time interface assertions.
var _ canonical.Document = (*association.AssociationModel)(nil)
var _ canonical.Persistable = (*association.AssociationModel)(nil)
