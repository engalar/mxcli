// SPDX-License-Identifier: Apache-2.0

package page

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/canonical"
)

// Persist is a stub — page creation is handled by execCreatePageV3 in the
// executor, which uses the gen builder for rich BSON generation. This stub
// satisfies the canonical.Persistable interface so PageDocument participates
// in the codec registry. Full implementation is a future plan.
func (d *PageDocument) Persist(ctx canonical.PersistContext) error {
	return fmt.Errorf("page.Persist: not yet implemented — use execCreatePageV3 directly")
}
