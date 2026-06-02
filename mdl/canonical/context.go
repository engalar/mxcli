// SPDX-License-Identifier: Apache-2.0

package canonical

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	modelID "github.com/mendixlabs/mxcli/model"
)

// PersistContext carries the project-scoped identifiers and backend handles a
// canonical document needs in order to write itself back to the project.
type PersistContext struct {
	DomainModelID    modelID.ID
	ExistingEntityID modelID.ID
	Backend          backend.DomainModelBackend
}
