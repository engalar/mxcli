// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

// mkGenProjectSecurity builds a gen-typed ProjectSecurity for use with
// withSecurityRepo. Returns both the object and the repo so tests can
// customise further before passing it to withSecurityRepo.
func mkGenProjectSecurity() *genSec.ProjectSecurity {
	return genSec.NewProjectSecurity()
}

func mkGenModuleSecurity() *genSec.ModuleSecurity {
	return genSec.NewModuleSecurity()
}

func mkGenModuleRole(name, description string) *genSec.ModuleRole {
	mr := genSec.NewModuleRole()
	mr.SetName(name)
	mr.SetDescription(description)
	return mr
}

func mkGenUserRole(name string, moduleRoles []string) *genSec.UserRole {
	ur := genSec.NewUserRole()
	ur.SetID(element.ID(nextID("ur")))
	ur.SetName(name)
	ur.SetModuleRolesQualifiedNames(moduleRoles)
	return ur
}

func mkGenDemoUser(userName string, userRoles []string) *genSec.DemoUser {
	du := genSec.NewDemoUser()
	du.SetUserName(userName)
	du.SetUserRolesQualifiedNames(userRoles)
	return du
}

func TestShowProjectSecurity_Mock(t *testing.T) {
	ps := mkGenProjectSecurity()
	ps.SetSecurityLevel("CheckEverything")
	ps.SetEnableDemoUsers(true)
	ps.SetAdminUserName("MxAdmin")
	ur1 := mkGenUserRole("Admin", nil)
	ur2 := mkGenUserRole("User", nil)
	ps.AddUserRoles(ur1)
	ps.AddUserRoles(ur2)
	du := mkGenDemoUser("demo_admin", nil)
	ps.AddDemoUsers(du)

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertNoError(t, listProjectSecurityGen(ctx))

	out := buf.String()
	assertContainsStr(t, out, "Security Level:")
	assertContainsStr(t, out, "MxAdmin")
	assertContainsStr(t, out, "Demo Users Enabled:")
}

func TestShowModuleRoles_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	ms := mkGenModuleSecurity()
	ms.AddModuleRoles(mkGenModuleRole("Admin", ""))
	ms.AddModuleRoles(mkGenModuleRole("User", ""))

	sec := &repostesting.RecordingSecurityRepository{
		GetModuleSecFunc: func(id model.ID) (*genSec.ModuleSecurity, error) {
			if id == mod.ID {
				return ms, nil
			}
			return nil, nil
		},
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec), withHierarchy(h))
	assertNoError(t, listModuleRolesGen(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "Qualified Name")
	assertContainsStr(t, out, "Role")
	assertContainsStr(t, out, "Admin")
	assertContainsStr(t, out, "User")
}

func TestShowUserRoles_Mock(t *testing.T) {
	ps := mkGenProjectSecurity()
	ps.AddUserRoles(mkGenUserRole("Administrator", []string{"MyModule.Admin"}))
	ps.AddUserRoles(mkGenUserRole("NormalUser", []string{"MyModule.User"}))

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertNoError(t, listUserRolesGen(ctx))

	out := buf.String()
	assertContainsStr(t, out, "Name")
	assertContainsStr(t, out, "Module Roles")
	assertContainsStr(t, out, "Administrator")
	assertContainsStr(t, out, "NormalUser")
}

func TestShowDemoUsers_Mock(t *testing.T) {
	ps := mkGenProjectSecurity()
	ps.SetEnableDemoUsers(true)
	ps.AddDemoUsers(mkGenDemoUser("demo_admin", []string{"Administrator"}))

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertNoError(t, listDemoUsersGen(ctx))

	out := buf.String()
	assertContainsStr(t, out, "User Name")
	assertContainsStr(t, out, "demo_admin")
}

func TestShowDemoUsers_Disabled_Mock(t *testing.T) {
	ps := mkGenProjectSecurity()
	ps.SetEnableDemoUsers(false)

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertNoError(t, listDemoUsersGen(ctx))
	assertContainsStr(t, buf.String(), "Demo users are disabled.")
}

func TestDescribeModuleRole_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	ms := mkGenModuleSecurity()
	ms.AddModuleRoles(mkGenModuleRole("Admin", "Full access"))

	psRoles := mkGenProjectSecurity()
	psRoles.AddUserRoles(mkGenUserRole("Administrator", []string{"MyModule.Admin"}))

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return psRoles, nil },
		GetModuleSecFunc: func(id model.ID) (*genSec.ModuleSecurity, error) {
			if id == mod.ID {
				return ms, nil
			}
			return nil, nil
		},
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec), withHierarchy(h))
	assertNoError(t, describeModuleRoleGen(ctx, ast.QualifiedName{Module: "MyModule", Name: "Admin"}))
	assertContainsStr(t, buf.String(), "create module role")
}

func TestDescribeUserRole_Mock(t *testing.T) {
	ps := mkGenProjectSecurity()
	ps.AddUserRoles(mkGenUserRole("Administrator", []string{"MyModule.Admin"}))

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertNoError(t, describeUserRoleGen(ctx, ast.QualifiedName{Name: "Administrator"}))
	assertContainsStr(t, buf.String(), "create user role")
}

func TestDescribeDemoUser_Mock(t *testing.T) {
	ps := mkGenProjectSecurity()
	ps.SetEnableDemoUsers(true)
	ps.AddDemoUsers(mkGenDemoUser("demo_admin", []string{"Administrator"}))

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertNoError(t, describeDemoUserGen(ctx, "demo_admin"))
	assertContainsStr(t, buf.String(), "create demo user")
}

func TestShowModuleRoles_Mock_FilterByModule(t *testing.T) {
	mod1 := mkModule("Sales")
	mod2 := mkModule("HR")
	h := mkHierarchy(mod1, mod2)

	ms1 := mkGenModuleSecurity()
	ms1.AddModuleRoles(mkGenModuleRole("Manager", ""))

	ms2 := mkGenModuleSecurity()
	ms2.AddModuleRoles(mkGenModuleRole("Employee", ""))

	sec := &repostesting.RecordingSecurityRepository{
		GetModuleSecFunc: func(id model.ID) (*genSec.ModuleSecurity, error) {
			switch id {
			case mod1.ID:
				return ms1, nil
			case mod2.ID:
				return ms2, nil
			}
			return nil, nil
		},
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod1, mod2}, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withSecurityRepo(sec), withHierarchy(h))
	assertNoError(t, listModuleRolesGen(ctx, "HR"))

	out := buf.String()
	assertNotContainsStr(t, out, "Sales")
	assertContainsStr(t, out, "HR")
	assertContainsStr(t, out, "Employee")
}

func TestDescribeModuleRole_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	ms := mkGenModuleSecurity()
	ms.AddModuleRoles(mkGenModuleRole("Admin", ""))

	sec := &repostesting.RecordingSecurityRepository{
		GetModuleSecFunc: func(id model.ID) (*genSec.ModuleSecurity, error) {
			if id == mod.ID {
				return ms, nil
			}
			return nil, nil
		},
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec), withHierarchy(h))
	assertError(t, describeModuleRoleGen(ctx, ast.QualifiedName{Module: "MyModule", Name: "NonExistent"}))
}

func TestDescribeUserRole_Mock_NotFound(t *testing.T) {
	ps := mkGenProjectSecurity()
	ps.AddUserRoles(mkGenUserRole("Admin", nil))

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, describeUserRoleGen(ctx, ast.QualifiedName{Name: "NonExistent"}))
}

func TestDescribeDemoUser_Mock_NotFound(t *testing.T) {
	ps := mkGenProjectSecurity()
	ps.SetEnableDemoUsers(true)
	ps.AddDemoUsers(mkGenDemoUser("demo_admin", nil))

	sec := &repostesting.RecordingSecurityRepository{
		GetFunc: func() (*genSec.ProjectSecurity, error) { return ps, nil },
	}
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withSecurityRepo(sec))
	assertError(t, describeDemoUserGen(ctx, "nonexistent"))
}

func TestShowAccessOnEntity_Mock_NilName(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listAccessOnEntityGen(ctx, nil))
}

// Stage 3.2.6.5: TestShowAccessOnMicroflow_Mock_NotFound removed —
// `listAccessOnMicroflow` (legacy sdk-typed) is gone; the dispatch
// in executor_query.go now calls `listAccessOnMicroflowGen` which
// reads from ctx.Microflows. Equivalent gen coverage is exercised
// via cmd_security_gen.go's NotFound paths.

func TestShowAccessOnPage_Mock_NotFound(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPagesFunc:   func() ([]*pages.Page, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertError(t, listAccessOnPageGen(ctx, &ast.QualifiedName{Module: "MyModule", Name: "NonExistent"}))
}

func TestShowAccessOnWorkflow_Mock_Unsupported(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, listAccessOnWorkflow(ctx, &ast.QualifiedName{Module: "MyModule", Name: "SomeWorkflow"}))
}

// TestGrantEntityAccess_XPathConstraint_PreservesRights verifies that granting
// entity access with an XPath WHERE clause shows the correct rights immediately
// after the GRANT (issue #431: output showed "(no access)" instead of "read *, write *").
func TestGrantEntityAccess_XPathConstraint_PreservesRights(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	statusAttr := &domainmodel.Attribute{
		BaseElement: model.BaseElement{ID: nextID("attr")},
		Name:        "Status",
	}
	entityBefore := &domainmodel.Entity{
		BaseElement: model.BaseElement{ID: nextID("ent")},
		ContainerID: mod.ID,
		Name:        "Order",
		Persistable: true,
		Attributes:  []*domainmodel.Attribute{statusAttr},
		AccessRules: nil, // no rules yet
	}
	dmBefore := &domainmodel.DomainModel{
		BaseElement: model.BaseElement{ID: nextID("dm")},
		ContainerID: mod.ID,
		Entities:    []*domainmodel.Entity{entityBefore},
	}

	// After AddEntityAccessRule, the second GetDomainModel call returns the entity
	// with the rule already applied (simulating what a real MPR backend would do).
	entityAfter := &domainmodel.Entity{
		BaseElement: model.BaseElement{ID: entityBefore.ID},
		ContainerID: mod.ID,
		Name:        "Order",
		Persistable: true,
		Attributes:  []*domainmodel.Attribute{statusAttr},
		AccessRules: []*domainmodel.AccessRule{
			{
				ModuleRoleNames:           []string{"MyModule.User"},
				AllowCreate:               false,
				AllowDelete:               false,
				DefaultMemberAccessRights: domainmodel.MemberAccessRightsReadWrite,
				XPathConstraint:           "[Status = 'Open']",
				MemberAccesses: []*domainmodel.MemberAccess{
					{
						AttributeName: "MyModule.Order.Status",
						AccessRights:  domainmodel.MemberAccessRightsReadWrite,
					},
				},
			},
		},
	}
	dmAfter := &domainmodel.DomainModel{
		BaseElement: model.BaseElement{ID: dmBefore.ID},
		ContainerID: mod.ID,
		Entities:    []*domainmodel.Entity{entityAfter},
	}

	callCount := 0

	// gen-typed ModuleSecurity with "User" role for role validation
	ms := mkGenModuleSecurity()
	ms.AddModuleRoles(mkGenModuleRole("User", ""))

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return ms, nil
		},
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) {
			callCount++
			if callCount == 1 {
				return dmBefore, nil // first call: before grant
			}
			return dmAfter, nil // second call (formatAccessRuleResult): after grant
		},
		AddEntityAccessRuleFunc: func(params backend.EntityAccessRuleParams) error {
			if params.XPathConstraint != "[Status = 'Open']" {
				t.Errorf("XPathConstraint not passed: got %q, want %q", params.XPathConstraint, "[Status = 'Open']")
			}
			if params.DefaultMemberAccess != "ReadWrite" {
				t.Errorf("DefaultMemberAccess not passed: got %q, want ReadWrite", params.DefaultMemberAccess)
			}
			return nil
		},
		ReconcileMemberAccessesFunc: func(unitID model.ID, moduleName string) (int, error) {
			return 0, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	stmt := &ast.GrantEntityAccessStmt{
		Entity: ast.QualifiedName{Module: "MyModule", Name: "Order"},
		Roles:  []ast.QualifiedName{{Module: "MyModule", Name: "User"}},
		Rights: []ast.EntityAccessRight{
			{Type: ast.EntityAccessReadAll},
			{Type: ast.EntityAccessWriteAll},
		},
		XPathConstraint: "[Status = 'Open']",
	}
	assertNoError(t, execGrantEntityAccessGen(ctx, stmt))

	out := buf.String()
	assertContainsStr(t, out, "Granted access")
	assertNotContainsStr(t, out, "(no access)")
	assertContainsStr(t, out, "read *")
}

// TestOutputEntityAccessGrants_XPathConstraint_EscapedQuotes verifies that
// outputEntityAccessGrants escapes single quotes inside the XPath constraint
// so the DESCRIBE ENTITY output is valid re-parseable MDL (issue #431).
func TestOutputEntityAccessGrants_XPathConstraint_EscapedQuotes(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	entity := &domainmodel.Entity{
		BaseElement: model.BaseElement{ID: nextID("ent")},
		ContainerID: mod.ID,
		Name:        "Order",
		Persistable: true,
		Attributes: []*domainmodel.Attribute{
			{BaseElement: model.BaseElement{ID: nextID("attr")}, Name: "Status"},
		},
		AccessRules: []*domainmodel.AccessRule{
			{
				ModuleRoleNames:           []string{"MyModule.User"},
				DefaultMemberAccessRights: domainmodel.MemberAccessRightsReadWrite,
				XPathConstraint:           "[Status = 'Open']",
				MemberAccesses: []*domainmodel.MemberAccess{
					{
						AttributeName: "MyModule.Order.Status",
						AccessRights:  domainmodel.MemberAccessRightsReadWrite,
					},
				},
			},
		},
	}

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))

	outputEntityAccessGrants(ctx, entity, "MyModule", "Order")

	out := buf.String()
	// Single quotes inside the XPath must be doubled for valid MDL
	assertContainsStr(t, out, "''Open''")
	// Should NOT contain unescaped version
	assertNotContainsStr(t, out, "= 'Open'")
	// The outer where clause delimiters must still be single quotes
	assertContainsStr(t, out, "where '")
}

// TestGrantEntityAccess_FakeRole_Issue399 verifies that GRANT ON ENTITY rejects
// a non-existent module role instead of silently creating a phantom access rule.
func TestGrantEntityAccess_FakeRole_Issue399(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	entity := &domainmodel.Entity{
		BaseElement: model.BaseElement{ID: nextID("ent")},
		Name:        "Order",
		Persistable: true,
	}
	dm := &domainmodel.DomainModel{
		BaseElement: model.BaseElement{ID: nextID("dm")},
		ContainerID: mod.ID,
		Entities:    []*domainmodel.Entity{entity},
	}

	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
		// No roles defined in module security
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return genSec.NewModuleSecurity(), nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execGrantEntityAccessGen(ctx, &ast.GrantEntityAccessStmt{
		Entity: ast.QualifiedName{Module: "MyModule", Name: "Order"},
		Roles:  []ast.QualifiedName{{Module: "MyModule", Name: "FakeRole"}},
		Rights: []ast.EntityAccessRight{{Type: ast.EntityAccessReadAll}},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "module role")
	assertContainsStr(t, err.Error(), "FakeRole")
}

// TestRevokeEntityAccess_FakeRole_Issue399 verifies that REVOKE ON ENTITY also
// rejects non-existent module roles.
func TestRevokeEntityAccess_FakeRole_Issue399(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	entity := &domainmodel.Entity{
		BaseElement: model.BaseElement{ID: nextID("ent")},
		Name:        "Customer",
		Persistable: true,
	}
	dm := &domainmodel.DomainModel{
		BaseElement: model.BaseElement{ID: nextID("dm")},
		ContainerID: mod.ID,
		Entities:    []*domainmodel.Entity{entity},
	}

	mb := &mock.MockBackend{
		IsConnectedFunc:    func() bool { return true },
		ListModulesFunc:    func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) { return dm, nil },
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return genSec.NewModuleSecurity(), nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execRevokeEntityAccessGen(ctx, &ast.RevokeEntityAccessStmt{
		Entity: ast.QualifiedName{Module: "MyModule", Name: "Customer"},
		Roles:  []ast.QualifiedName{{Module: "MyModule", Name: "GhostRole"}},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "module role")
	assertContainsStr(t, err.Error(), "GhostRole")
}
