// SPDX-License-Identifier: Apache-2.0

package executor

import "fmt"

// GapType classifies an access gap found by AnalyzeAccess.
type GapType string

const (
	// GapEntityRead: role can see a page/widget that reads the entity but has no read grant.
	GapEntityRead GapType = "entity-read"
	// GapEntityWrite: role can see an editable widget on the entity but has no write grant.
	GapEntityWrite GapType = "entity-write"
	// GapMFExecute: role can reach a directly-called microflow but has no execute grant.
	GapMFExecute GapType = "mf-execute"
)

// AccessGap describes a single permission gap detected by AnalyzeAccess.
type AccessGap struct {
	UserRole   string // e.g. "Customer"
	ModuleRole string // e.g. "HD.CustomerRole"
	Path       string // human-readable diagnostic trail
	EntityQN   string // non-empty for GapEntityRead / GapEntityWrite
	MFQN       string // non-empty for GapMFExecute
	GapType    GapType
}

// SuggestedMDL returns an executable MDL grant statement that closes the gap.
func (g AccessGap) SuggestedMDL() string {
	switch g.GapType {
	case GapEntityRead:
		return fmt.Sprintf("grant %s on %s (read *);", g.ModuleRole, g.EntityQN)
	case GapEntityWrite:
		return fmt.Sprintf("grant %s on %s (create, read *, write *);", g.ModuleRole, g.EntityQN)
	case GapMFExecute:
		return fmt.Sprintf("grant execute on microflow %s to %s;", g.MFQN, g.ModuleRole)
	default:
		return ""
	}
}

// RuleID returns the ACCESS-xxx identifier for check output.
func (g AccessGap) RuleID() string {
	switch g.GapType {
	case GapEntityRead:
		return "ACCESS-001"
	case GapEntityWrite:
		return "ACCESS-002"
	case GapMFExecute:
		return "ACCESS-003"
	default:
		return "ACCESS-000"
	}
}
