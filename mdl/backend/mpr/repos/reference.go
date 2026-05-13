// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// referenceService implements repos.ReferenceService. It delegates the
// BSON-walking heavy lift (ScanRenameReferences, PatchNavigationProfile,
// ScanQualifiedNameUpdates) to sdk/mpr.Writer's mature scanners, then
// persists each computed patch through the modelsdk WriteTransaction to
// avoid the SQLITE_READONLY_DBMOVED 1544 bug in the legacy update path.
//
// Future cleanup (Stage 4): port the scanners themselves to mdl/repos so
// the sdk/mpr import here disappears. Tracked separately.
type referenceService struct {
	mw   *mmpr.Writer
	sdkW *sdkmpr.Writer
}

// NewReferenceService constructs a ReferenceService that uses the modelsdk
// Writer for persistence and the sdk/mpr Writer for BSON scanning. Both
// MUST share the same underlying SQLite connection (see mprbackend.Wrap
// for the canonical pattern).
func NewReferenceService(mw *mmpr.Writer, sdkW *sdkmpr.Writer) repos.ReferenceService {
	return &referenceService{mw: mw, sdkW: sdkW}
}

var _ repos.ReferenceService = (*referenceService)(nil)

func (s *referenceService) ScanRename(oldQN, newQN string) ([]repos.RenameHit, error) {
	patches, hits, err := s.sdkW.ScanRenameReferences(oldQN, newQN)
	if err != nil {
		return nil, fmt.Errorf("scan rename references: %w", err)
	}
	for _, p := range patches {
		if err := s.writeUnit(p.ID, p.Contents); err != nil {
			// Return partial hits + error so callers can surface what was applied.
			return convertHits(hits), err
		}
	}
	return convertHits(hits), nil
}

func (s *referenceService) PatchNavigationProfile(navDocID model.ID, profileName string, spec types.NavigationProfileSpec) error {
	rawBytes, err := s.mw.Reader().GetRawUnitBytes(string(navDocID))
	if err != nil {
		return fmt.Errorf("read nav unit %s: %w", navDocID, err)
	}
	patched, err := s.sdkW.PatchNavigationProfile(rawBytes, profileName, spec)
	if err != nil {
		return fmt.Errorf("patch navigation profile: %w", err)
	}
	return s.writeUnit(string(navDocID), patched)
}

func (s *referenceService) UpdateEnumerationRefsInAllDomainModels(oldQN, newQN string) error {
	// Reuse the sdk/mpr qualified-name scanner — it walks every BSON string
	// regardless of domain, which is exactly what we want for the
	// EnumerationRef field embedded in Attribute.Type.
	patches, err := s.sdkW.ScanQualifiedNameUpdates(oldQN, newQN)
	if err != nil {
		return fmt.Errorf("scan qualified-name updates: %w", err)
	}
	for _, p := range patches {
		if err := s.writeUnit(p.ID, p.Contents); err != nil {
			return err
		}
	}
	return nil
}

// writeUnit persists one patched unit through the modelsdk
// WriteTransaction (avoiding the legacy 1544 bug path).
func (s *referenceService) writeUnit(unitID string, contents []byte) error {
	wtx, err := s.mw.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := wtx.WriteUnit(unitID, contents); err != nil {
		_ = wtx.Rollback()
		return fmt.Errorf("write unit %s: %w", unitID, err)
	}
	if err := wtx.Commit(); err != nil {
		return fmt.Errorf("commit unit %s: %w", unitID, err)
	}
	return nil
}

func convertHits(in []sdkmpr.RenameHit) []repos.RenameHit {
	out := make([]repos.RenameHit, len(in))
	for i, h := range in {
		out[i] = repos.RenameHit{
			UnitID:   h.UnitID,
			UnitType: h.UnitType,
			Name:     h.Name,
			Count:    h.Count,
		}
	}
	return out
}
