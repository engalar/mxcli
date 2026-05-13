// SPDX-License-Identifier: Apache-2.0

package repos

import "github.com/mendixlabs/mxcli/model"

// CascadeService coordinates deletes that span multiple domains. The standard
// per-domain Writers can only delete units they own; cascading deletes
// (drop module → all its microflows / pages / domain models / etc.; drop
// folder → all child documents) require a service that walks containment
// relationships and removes child units before the parent.
//
// Implementations rely on mmpr.Writer.DeleteChildUnits + DeleteUnit; no
// per-domain knowledge is required because containment lives in the
// MPR Unit table.
type CascadeService interface {
	// DeleteModule removes a module and every unit contained under it
	// (microflows, pages, domain models, security, settings, etc.) atomically
	// from the caller's perspective. Returns nil if the module did not exist.
	DeleteModule(moduleID model.ID) error

	// DeleteFolder removes an empty folder. Refuses (returns error) if the
	// folder still contains child units — folder deletion is not cascading
	// because folders are user-organisational and accidental nuking is bad.
	DeleteFolder(folderID model.ID) error
}
