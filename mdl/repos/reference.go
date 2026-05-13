// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// RenameHit reports a single document that contained references to a
// renamed element. Returned by ReferenceService.ScanRename in dry-run-style
// flows so callers can preview impact before applying changes.
type RenameHit struct {
	UnitID   string // Document UUID
	UnitType string // e.g., "Microflows$Microflow"
	Name     string // Document name
	Count    int    // Number of string occurrences replaced in this document
}

// ReferenceService coordinates cross-domain reference updates that
// follow a rename or schema change. Per-domain Writers cannot do this
// because a single rename (e.g., entity name change) can ripple through
// microflows, pages, security rules, navigation, and external mappings —
// the service walks all units and patches references uniformly.
//
// Stage 2.7 implementations delegate the BSON-walking heavy lift to
// sdk/mpr scanners (mature code, well-tested), while writes go through
// the modelsdk path to avoid the legacy SQLITE_READONLY_DBMOVED 1544 bug.
type ReferenceService interface {
	// ScanRename rewrites every occurrence of oldQN to newQN across all
	// units in the project. Returns the per-document hit list so callers
	// can log impact. Always applies; pass the same name to dry-run.
	ScanRename(oldQN, newQN string) ([]RenameHit, error)

	// PatchNavigationProfile applies a navigation-profile spec patch to a
	// navigation document. Used after page renames or new home-page wiring.
	PatchNavigationProfile(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error

	// UpdateEnumerationRefsInAllDomainModels walks every domain model and
	// updates Attribute.EnumerationAttributeType.EnumerationRef where it
	// matches oldQN, replacing with newQN. Used by ALTER ENUMERATION RENAME.
	UpdateEnumerationRefsInAllDomainModels(oldQN, newQN string) error
}
