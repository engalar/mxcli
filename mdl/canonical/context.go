// SPDX-License-Identifier: Apache-2.0

package canonical

import modelID "github.com/mendixlabs/mxcli/model"

// PersistContext carries the project-scoped identifiers and backend handle a
// canonical document needs to write itself to the project.
// Backend is typed as any to keep this package free of backend imports;
// each domain's persist.go defines a local interface and type-asserts it.
type PersistContext struct {
	DomainModelID    modelID.ID
	ExistingEntityID modelID.ID
	Backend          any
}
