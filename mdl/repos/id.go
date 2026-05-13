// SPDX-License-Identifier: Apache-2.0

package repos

import "github.com/mendixlabs/mxcli/model"

// IDGenerator mints fresh model.IDs. Implementations must produce IDs
// in the same UUID-string shape mmpr.GenerateID produces (addendum
// Blocker 2: element.ID(mmpr.GenerateID()) is the canonical cast).
type IDGenerator interface {
	NewID() model.ID
}
