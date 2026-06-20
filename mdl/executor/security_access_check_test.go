// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDM "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
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

// TODO(ACC-005): TestAccessGap_SuggestedMDL_EntityWrite — restore when GapEntityWrite
// detection is implemented in detectGaps.

func TestAccessGap_SuggestedMDL_MFExecute(t *testing.T) {
	g := AccessGap{
		GapType:    GapMFExecute,
		ModuleRole: "HD.CustomerRole",
		MFQN:       "HD.DS_GetMyProfile",
	}
	want := "grant execute on microflow HD.DS_GetMyProfile to HD.CustomerRole;"
	if got := g.SuggestedMDL(); got != want {
		t.Errorf("SuggestedMDL() = %q, want %q", got, want)
	}
}

// TestUserRoleToModuleRoles exercises buildUserRoleMap via a MockBackend whose
// GetProjectSecurityGen returns synthetic UserRole items. The old test only
// asserted a hand-built map; this version calls the real builder function.
func TestUserRoleToModuleRoles(t *testing.T) {
	cases := []struct {
		name      string
		userRoles []struct {
			roleName    string
			moduleRoles []string
		}
		checkRole       string
		wantModuleRoles []string
	}{
		{
			name: "two user roles with distinct module roles",
			userRoles: []struct {
				roleName    string
				moduleRoles []string
			}{
				{"Customer", []string{"HD.CustomerRole", "KB.Reader"}},
				{"Agent", []string{"HD.AgentRole", "KB.Contributor"}},
			},
			checkRole:       "Customer",
			wantModuleRoles: []string{"HD.CustomerRole", "KB.Reader"},
		},
		{
			name: "user role with no module roles",
			userRoles: []struct {
				roleName    string
				moduleRoles []string
			}{
				{"Guest", nil},
			},
			checkRole:       "Guest",
			wantModuleRoles: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ps := genSec.NewProjectSecurity()
			for _, ur := range tc.userRoles {
				genUR := genSec.NewUserRole()
				genUR.SetID(element.ID(nextID("ur")))
				genUR.SetName(ur.roleName)
				genUR.SetModuleRolesQualifiedNames(ur.moduleRoles)
				ps.AddUserRoles(genUR)
			}

			mb := &mock.MockBackend{
				IsConnectedFunc: func() bool { return true },
				GetProjectSecurityGenFunc: func() (*genSec.ProjectSecurity, error) {
					return ps, nil
				},
			}
			ctx, _ := newMockCtx(t, withBackend(mb))

			got, err := buildUserRoleMap(ctx)
			if err != nil {
				t.Fatalf("buildUserRoleMap returned error: %v", err)
			}
			gotRoles := got[tc.checkRole]
			if len(gotRoles) != len(tc.wantModuleRoles) {
				t.Errorf("got %d module roles for %q, want %d: %v", len(gotRoles), tc.checkRole, len(tc.wantModuleRoles), gotRoles)
				return
			}
			for i, want := range tc.wantModuleRoles {
				if gotRoles[i] != want {
					t.Errorf("module role[%d] = %q, want %q", i, gotRoles[i], want)
				}
			}
		})
	}
}

// TestCollectEntityGrantsForRole exercises buildEntityGrants via a MockBackend
// whose ListDomainModelsGen returns a synthetic DomainModel with one Entity and
// one AccessRule. The old test only asserted a hand-built map; this version
// calls the real builder function.
func TestCollectEntityGrantsForRole(t *testing.T) {
	mod := mkModule("HD")

	ent := genDM.NewEntity()
	ent.SetID(element.ID(nextID("ent")))
	ent.SetName("UserProfile")
	ar := genDM.NewAccessRule()
	ar.SetModuleRolesQualifiedNames([]string{"HD.CustomerRole"})
	ar.SetDefaultMemberAccessRights("ReadOnly")
	ar.SetAllowCreate(false)
	ar.SetAllowDelete(false)
	ent.AddAccessRules(ar)

	dm := genDM.NewDomainModel()
	dmID := model.ID(nextID("dm"))
	dm.SetID(element.ID(dmID))
	dm.AddEntities(ent)

	// Register dm.ID → mod.ID so FindModuleID resolves the module name "HD".
	h := mkHierarchy(mod)
	withContainer(h, dmID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListDomainModelsGenFunc: func() ([]*genDM.DomainModel, error) {
			return []*genDM.DomainModel{dm}, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))

	grants, err := buildEntityGrants(ctx)
	if err != nil {
		t.Fatalf("buildEntityGrants returned error: %v", err)
	}

	summary, ok := grants["HD.CustomerRole"]["HD.UserProfile"]
	if !ok {
		t.Fatalf("expected grant for HD.CustomerRole on HD.UserProfile, got map: %v", grants)
	}
	if !summary.canRead {
		t.Error("expected canRead=true for HD.UserProfile")
	}
	if summary.canWrite {
		t.Error("expected canWrite=false for HD.UserProfile")
	}
	if summary.canCreate {
		t.Error("expected canCreate=false for HD.UserProfile")
	}
}

// TestCollectMFEntityRefs is a placeholder tracking the TODO in
// collectMFEntityRefs. Update this test when ObjectsItems() traversal is
// implemented (Task 5).
func TestCollectMFEntityRefs(t *testing.T) {
	t.Skip("TODO(Task 5): collectMFEntityRefs not yet implemented — update when ObjectsItems() traversal is complete")
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
	result := collectWidgetRefs(pm, map[string]mfMeta{})
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
	result := collectWidgetRefs(pm, map[string]mfMeta{})
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
