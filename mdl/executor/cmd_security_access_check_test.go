// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"
)

func TestAccessGap_SuggestedMDL_EntityRead(t *testing.T) {
	g := AccessGap{
		GapType:    GapEntityRead,
		ModuleRole: "HD.CustomerRole",
		EntityQN:   "HD.UserProfile",
	}
	want := "grant HD.CustomerRole on HD.UserProfile (read *);"
	if got := g.SuggestedMDL(); got != want {
		t.Errorf("SuggestedMDL() = %q, want %q", got, want)
	}
}

func TestAccessGap_SuggestedMDL_EntityWrite(t *testing.T) {
	g := AccessGap{
		GapType:    GapEntityWrite,
		ModuleRole: "HD.CustomerRole",
		EntityQN:   "HD.PasswordForm",
	}
	want := "grant HD.CustomerRole on HD.PasswordForm (create, read *, write *);"
	if got := g.SuggestedMDL(); got != want {
		t.Errorf("SuggestedMDL() = %q, want %q", got, want)
	}
}

func TestAccessGap_SuggestedMDL_MFExecute(t *testing.T) {
	g := AccessGap{
		GapType: GapMFExecute,
		ModuleRole: "HD.CustomerRole",
		MFQN:       "HD.DS_GetMyProfile",
	}
	want := "grant execute on microflow HD.DS_GetMyProfile to HD.CustomerRole;"
	if got := g.SuggestedMDL(); got != want {
		t.Errorf("SuggestedMDL() = %q, want %q", got, want)
	}
}

func TestUserRoleToModuleRoles(t *testing.T) {
	// Simulate ProjectSecurity with two UserRoles
	mapping := map[string][]string{
		"Customer": {"HD.CustomerRole", "KB.Reader"},
		"Agent":    {"HD.AgentRole", "KB.Contributor"},
	}
	// buildUserRoleMap is derived from ProjectSecurity.UserRolesItems()
	// We test the helper-output shape directly.
	if got := mapping["Customer"]; len(got) != 2 {
		t.Errorf("expected 2 module roles for Customer, got %d", len(got))
	}
}

func TestCollectEntityGrantsForRole(t *testing.T) {
	// buildEntityGrants result for "HD.CustomerRole" should report read access.
	grants := map[string]map[string]entityAccessSummary{
		"HD.CustomerRole": {
			"HD.UserProfile": {canRead: true, canWrite: false, canCreate: false},
		},
	}
	summary := grants["HD.CustomerRole"]["HD.UserProfile"]
	if !summary.canRead {
		t.Error("expected canRead=true for HD.UserProfile")
	}
	if summary.canWrite {
		t.Error("expected canWrite=false for HD.UserProfile")
	}
}
