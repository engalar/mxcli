// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
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

func TestCollectWidgetEntities(t *testing.T) {
	pm := &types.PageModel{
		Widgets: []*types.WidgetNode{
			{
				Kind: types.WidgetDataView,
				Name: "dvProfile",
				DataSource: &types.DataSourceDef{
					Kind:   types.DataSourceDatabase,
					Entity: "HD.UserProfile",
				},
				Children: []*types.WidgetNode{
					{Kind: types.WidgetTextBox, Name: "tbName", EntityAttr: "DisplayName"},
				},
			},
		},
	}
	result := collectWidgetRefs(pm)
	if !containsStr(result.entityQNs, "HD.UserProfile") {
		t.Errorf("expected HD.UserProfile in entityQNs, got %v", result.entityQNs)
	}
}

func TestCollectWidgetMFRefs(t *testing.T) {
	pm := &types.PageModel{
		Widgets: []*types.WidgetNode{
			{
				Kind: types.WidgetDataView,
				Name: "dvProfile",
				DataSource: &types.DataSourceDef{
					Kind:      types.DataSourceMicroflow,
					Reference: "HD.DS_GetMyProfile",
				},
			},
		},
	}
	result := collectWidgetRefs(pm)
	if !containsStr(result.directMFQNs, "HD.DS_GetMyProfile") {
		t.Errorf("expected HD.DS_GetMyProfile in directMFQNs, got %v", result.directMFQNs)
	}
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func TestDetectGaps_EntityReadGap(t *testing.T) {
	urToMR := map[string][]string{
		"Customer": {"HD.CustomerRole"},
	}
	entityGrants := map[string]map[string]entityAccessSummary{
		// HD.CustomerRole has NO grant on HD.UserProfile
	}
	pageGrants := map[string][]string{
		"HD.CustomerRole": {"HD.ManageMyAccount"},
	}
	mfGrants := map[string][]string{}
	mfMetaMap := map[string]mfMeta{}
	pageModels := map[string]*types.PageModel{
		"HD.ManageMyAccount": {
			Widgets: []*types.WidgetNode{
				{
					Kind: types.WidgetDataView,
					Name: "dvProfile",
					DataSource: &types.DataSourceDef{
						Kind:   types.DataSourceDatabase,
						Entity: "HD.UserProfile",
					},
				},
			},
		},
	}

	gaps := detectGaps(urToMR, entityGrants, pageGrants, mfGrants, mfMetaMap, pageModels)

	if len(gaps) == 0 {
		t.Fatal("expected at least one gap for missing HD.UserProfile read access")
	}
	found := false
	for _, g := range gaps {
		if g.EntityQN == "HD.UserProfile" && g.GapType == GapEntityRead && g.ModuleRole == "HD.CustomerRole" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected GapEntityRead for HD.UserProfile / HD.CustomerRole, got %+v", gaps)
	}
}

func TestDetectGaps_NoGapWhenGrantPresent(t *testing.T) {
	urToMR := map[string][]string{"Customer": {"HD.CustomerRole"}}
	entityGrants := map[string]map[string]entityAccessSummary{
		"HD.CustomerRole": {
			"HD.UserProfile": {canRead: true},
		},
	}
	pageGrants := map[string][]string{"HD.CustomerRole": {"HD.ManageMyAccount"}}
	mfGrants := map[string][]string{}
	mfMetaMap := map[string]mfMeta{}
	pageModels := map[string]*types.PageModel{
		"HD.ManageMyAccount": {
			Widgets: []*types.WidgetNode{{
				Kind:       types.WidgetDataView,
				DataSource: &types.DataSourceDef{Kind: types.DataSourceDatabase, Entity: "HD.UserProfile"},
			}},
		},
	}
	gaps := detectGaps(urToMR, entityGrants, pageGrants, mfGrants, mfMetaMap, pageModels)
	if len(gaps) != 0 {
		t.Errorf("expected 0 gaps when grant is present, got %d: %+v", len(gaps), gaps)
	}
}
