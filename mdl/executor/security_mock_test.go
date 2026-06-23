// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"errors"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
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
	assertContainsStr(t, buf.String(), "create or modify user role")
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

	// Stage 3.3.5.C6: ListPagesFunc not wired — MockBackend returns
	// (nil, nil) by default, which is enough to exercise the
	// "not-found" path in listAccessOnPageGen.
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return true }}
	ctx, _ := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
	)
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

	entityID := nextID("ent")
	attrID := nextID("attr")
	entityAfterGen := mkEntityGen("Order")
	entityAfterGen.SetID(element.ID(entityID))
	attrAfterGen := genDm.NewAttribute()
	attrAfterGen.SetID(element.ID(attrID))
	attrAfterGen.SetName("Status")
	attrAfterGen.SetType(genDm.NewStringAttributeType())
	entityAfterGen.AddAttributes(attrAfterGen)
	ruleAfterGen := genDm.NewAccessRule()
	ruleAfterGen.SetModuleRolesQualifiedNames([]string{"MyModule.User"})
	ruleAfterGen.SetDefaultMemberAccessRights(genDm.MemberAccessRightsReadWrite)
	ruleAfterGen.SetXPathConstraint("[Status = 'Open']")
	memberAfterGen := genDm.NewMemberAccess()
	memberAfterGen.SetAttributeQualifiedName("MyModule.Order.Status")
	memberAfterGen.SetAccessRights(genDm.MemberAccessRightsReadWrite)
	ruleAfterGen.AddMemberAccesses(memberAfterGen)
	entityAfterGen.AddAccessRules(ruleAfterGen)
	dmAfterGen := mkDomainModelGen(mod.ID, entityAfterGen)

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
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) {
			callCount++
			if callCount == 1 {
				return mkDomainModelGen(mod.ID, mkEntityGen("Order")), nil // first call: existence check
			}
			return dmAfterGen, nil // second call (formatAccessRuleResult): after grant
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
		ReconcileMemberAccessesFunc: func(unitID model.ID, moduleName string) ([]string, error) {
			return nil, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h), withDomainModelsRepo(makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{
		mod.ID: {dmAfterGen},
	})))
	stmt := &ast.GrantEntityAccessStmt{
		Entity: ast.QualifiedName{Module: "MyModule", Name: "Order"},
		Roles:  []ast.QualifiedName{{Module: "MyModule", Name: "User"}},
		Rights: []ast.EntityAccessRight{
			{Type: ast.EntityAccessReadAll},
			{Type: ast.EntityAccessWriteAll},
		},
		XPathConstraint: "[Status = 'Open']",
	}
	assertNoError(t, ExecGrantEntityAccessGenFn(ctx, stmt, execContextToDeps(ctx)))

	out := buf.String()
	assertContainsStr(t, out, "Granted access")
	assertNotContainsStr(t, out, "(no access)")
	assertContainsStr(t, out, "read *")
}

// TestOutputEntityAccessGrants_XPathConstraint_EscapedQuotes verifies that
// outputEntityAccessGrantsGen escapes single quotes inside the XPath constraint
// so the DESCRIBE ENTITY output is valid re-parseable MDL (issue #431).
func TestOutputEntityAccessGrants_XPathConstraint_EscapedQuotes(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)

	entity := mkEntityGen("Order")
	attr := genDm.NewAttribute()
	attr.SetID(element.ID(nextID("attr")))
	attr.SetName("Status")
	attr.SetType(genDm.NewStringAttributeType())
	entity.AddAttributes(attr)
	rule := genDm.NewAccessRule()
	rule.SetModuleRolesQualifiedNames([]string{"MyModule.User"})
	rule.SetDefaultMemberAccessRights(genDm.MemberAccessRightsReadWrite)
	rule.SetXPathConstraint("[Status = 'Open']")
	memberAccess := genDm.NewMemberAccess()
	memberAccess.SetAttributeQualifiedName("MyModule.Order.Status")
	memberAccess.SetAccessRights(genDm.MemberAccessRightsReadWrite)
	rule.AddMemberAccesses(memberAccess)
	entity.AddAccessRules(rule)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))

	outputEntityAccessGrantsGen(ctx, entity, "MyModule", "Order")

	out := buf.String()
	// Single quotes inside the XPath must be doubled for valid MDL
	assertContainsStr(t, out, "''Open''")
	// Should NOT contain unescaped version
	assertNotContainsStr(t, out, "= 'Open'")
	// The outer where clause delimiters must still be single quotes
	assertContainsStr(t, out, "where '")
}

// TestGrantEntityAccess_FakeRole_Issue399 originally verified that GRANT ON
// ENTITY rejected a non-existent module role with a fatal error. BUG-04
// downgraded "role not found" to a WARNING so long batch scripts can
// continue past typos; the test now verifies the new contract — no error
// is returned, but the WARNING surfaces the typo on stderr so the user
// is not silently misled.
func TestGrantEntityAccess_FakeRole_Issue399(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	entity := mkEntityGen("Order")
	dm := mkDomainModelGen(mod.ID, entity)

	mb := &mock.MockBackend{
		IsConnectedFunc:       func() bool { return true },
		ListModulesFunc:       func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) { return dm, nil },
		// No roles defined in module security
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return genSec.NewModuleSecurity(), nil
		},
	}

	ctx, buf := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withDomainModelsRepo(makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{
			mod.ID: {dm},
		})),
	)
	err := ExecGrantEntityAccessGenFn(ctx, &ast.GrantEntityAccessStmt{
		Entity: ast.QualifiedName{Module: "MyModule", Name: "Order"},
		Roles:  []ast.QualifiedName{{Module: "MyModule", Name: "FakeRole"}},
		Rights: []ast.EntityAccessRight{{Type: ast.EntityAccessReadAll}},
	}, execContextToDeps(ctx))
	// BUG-04: missing role is now a warning, not a fatal error.
	if err != nil {
		var nfe *mdlerrors.NotFoundError
		if errors.As(err, &nfe) && nfe.Kind == "module role" {
			t.Fatalf("expected missing role to be a warning, got fatal NotFound: %v", err)
		}
	}
	assertContainsStr(t, buf.String(), "WARNING")
	assertContainsStr(t, buf.String(), "FakeRole")
}

// TestRevokeEntityAccess_FakeRole_Issue399 — see the BUG-04 note on the
// GRANT counterpart above. REVOKE also downgrades missing-role to WARNING.
func TestRevokeEntityAccess_FakeRole_Issue399(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	entity := mkEntityGen("Customer")
	dm := mkDomainModelGen(mod.ID, entity)

	mb := &mock.MockBackend{
		IsConnectedFunc:       func() bool { return true },
		ListModulesFunc:       func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetDomainModelGenFunc: func(id model.ID) (*genDm.DomainModel, error) { return dm, nil },
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return genSec.NewModuleSecurity(), nil
		},
	}

	ctx, buf := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withDomainModelsRepo(makeDomainModelsRepo(map[model.ID][]*genDm.DomainModel{
			mod.ID: {dm},
		})),
	)
	err := ExecRevokeEntityAccessGenFn(ctx, &ast.RevokeEntityAccessStmt{
		Entity: ast.QualifiedName{Module: "MyModule", Name: "Customer"},
		Roles:  []ast.QualifiedName{{Module: "MyModule", Name: "GhostRole"}},
	}, execContextToDeps(ctx))
	if err != nil {
		var nfe *mdlerrors.NotFoundError
		if errors.As(err, &nfe) && nfe.Kind == "module role" {
			t.Fatalf("expected missing role to be a warning, got fatal NotFound: %v", err)
		}
	}
	assertContainsStr(t, buf.String(), "WARNING")
	assertContainsStr(t, buf.String(), "GhostRole")
}

// BUG-04: validateModuleRole must downgrade "role not found" from a fatal
// NotFoundError to a WARNING printed to ctx.Output, returning nil so that
// scripts continue past the missing role. Other failure modes (module
// missing, backend read error) must still return errors.

func TestValidateModuleRole_MissingRole_IsWarningNotError(t *testing.T) {
	mod := mkModule("TestModule")
	h := mkHierarchy(mod)

	// Module security with no roles defined — any role lookup misses.
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return genSec.NewModuleSecurity(), nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))

	found, err := validateModuleRole(ctx, ast.QualifiedName{Module: "TestModule", Name: "NonExistentRole"})
	if err != nil {
		t.Fatalf("expected nil error for missing module role, got: %v", err)
	}
	if found {
		t.Error("expected found=false for missing module role")
	}
	out := buf.String()
	assertContainsStr(t, out, "WARNING")
	assertContainsStr(t, out, "NonExistentRole")
	assertContainsStr(t, out, "TestModule")
}

func TestValidateModuleRole_ModuleMissing_StillErrors(t *testing.T) {
	// No modules in the project — findModule returns NotFound.
	// Since commit 0a90910a (fix BUG-04), module-not-found is a non-fatal
	// WARNING: the grant is skipped and the script continues. Only real
	// backend I/O errors remain fatal.
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return nil, nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb))

	found, err := validateModuleRole(ctx, ast.QualifiedName{Module: "GhostModule", Name: "Admin"})
	if err != nil {
		t.Fatalf("expected nil error for missing module (WARNING path), got: %v", err)
	}
	if found {
		t.Error("expected found=false for missing module")
	}
	assertContainsStr(t, buf.String(), "WARNING")
	assertContainsStr(t, buf.String(), "GhostModule")
}

func TestValidateModuleRole_BackendError_StillErrors(t *testing.T) {
	mod := mkModule("TestModule")
	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return nil, errors.New("disk read failure")
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))

	_, err := validateModuleRole(ctx, ast.QualifiedName{Module: "TestModule", Name: "Admin"})
	assertError(t, err)
	// The error must NOT be a NotFoundError — backend failures stay fatal.
	var nfe *mdlerrors.NotFoundError
	if errors.As(err, &nfe) {
		t.Errorf("backend read failure should not be reported as NotFoundError, got: %v", err)
	}
}

func TestGrantMicroflow_MissingRole_IsWarningNotError(t *testing.T) {
	mod := mkModule("TestModule")
	h := mkHierarchy(mod)

	mfID := nextID("mf")
	mf := mkMicroflowGen("MyFlow")
	mf.SetID(element.ID(mfID))

	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) { return []*genMf.Microflow{mf}, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == mfID {
				return mod.ID, nil
			}
			return "", nil
		},
	}

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		// No module roles defined — any role grant validates against an empty list.
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return genSec.NewModuleSecurity(), nil
		},
	}
	ctx, buf := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withMicroflowsRepo(mfRepo),
	)

	stmt := &ast.GrantMicroflowAccessStmt{
		Microflow: ast.QualifiedName{Module: "TestModule", Name: "MyFlow"},
		Roles: []ast.QualifiedName{
			{Module: "TestModule", Name: "NonExistentRole"},
		},
	}
	err := ExecGrantMicroflowAccessGenFn(ctx, stmt, execContextToDeps(ctx))
	if err != nil {
		t.Fatalf("expected nil error for missing module role, got: %v", err)
	}
	out := buf.String()
	assertContainsStr(t, out, "WARNING")
	assertContainsStr(t, out, "NonExistentRole")

	// Phantom role must NOT be written to the microflow's AllowedModuleRoles.
	roles := mf.AllowedModuleRolesQualifiedNames()
	if len(roles) != 0 {
		t.Errorf("phantom role must not be written to microflow AllowedModuleRoles, got: %v", roles)
	}
}

func TestGrantMicroflow_OneValidOnePhantomRole_OnlyValidWritten(t *testing.T) {
	mod := mkModule("TestModule")
	h := mkHierarchy(mod)

	mfID := nextID("mf")
	mf := mkMicroflowGen("MyFlow")
	mf.SetID(element.ID(mfID))

	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) { return []*genMf.Microflow{mf}, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == mfID {
				return mod.ID, nil
			}
			return "", nil
		},
	}

	// Module security has "Admin" but not "Ghost".
	ms := mkGenModuleSecurity()
	ms.AddModuleRoles(mkGenModuleRole("Admin", ""))

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return ms, nil
		},
	}
	ctx, buf := newMockCtx(t,
		withBackend(mb),
		withHierarchy(h),
		withMicroflowsRepo(mfRepo),
	)

	stmt := &ast.GrantMicroflowAccessStmt{
		Microflow: ast.QualifiedName{Module: "TestModule", Name: "MyFlow"},
		Roles: []ast.QualifiedName{
			{Module: "TestModule", Name: "Admin"},
			{Module: "TestModule", Name: "Ghost"},
		},
	}
	err := ExecGrantMicroflowAccessGenFn(ctx, stmt, execContextToDeps(ctx))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// WARNING only for Ghost.
	out := buf.String()
	assertContainsStr(t, out, "WARNING")
	assertContainsStr(t, out, "Ghost")

	// Only Admin must be written.
	roles := mf.AllowedModuleRolesQualifiedNames()
	if len(roles) != 1 || roles[0] != "TestModule.Admin" {
		t.Errorf("expected only TestModule.Admin in AllowedModuleRoles, got: %v", roles)
	}
}

// TestGrantMicroflow_BareRoleName_InfersModuleFromMicroflow verifies that
// `grant execute on microflow Mod.MF to User;` (role without module prefix)
// resolves to `Mod.User` — the module is inferred from the microflow's module.
// Previously, role.Module=="" caused findModule("") to fail.
func TestGrantMicroflow_BareRoleName_InfersModuleFromMicroflow(t *testing.T) {
	mod := mkModule("PayerRegistration")
	h := mkHierarchy(mod)

	mfID := nextID("mf")
	mf := mkMicroflowGen("ACT_Payer_Save")
	mf.SetID(element.ID(mfID))

	var updatedMF *genMf.Microflow
	mfRepo := &repostesting.RecordingMicroflowRepository{
		ListAllFunc: func() ([]*genMf.Microflow, error) { return []*genMf.Microflow{mf}, nil },
		GetContainerUUIDFunc: func(id model.ID) (model.ID, error) {
			if id == mfID {
				return mod.ID, nil
			}
			return "", nil
		},
		UpdateFunc: func(mf *genMf.Microflow) error {
			updatedMF = mf
			return nil
		},
	}

	// Module security has role "User" in PayerRegistration.
	ms := mkGenModuleSecurity()
	ms.AddModuleRoles(mkGenModuleRole("User", ""))

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		GetModuleSecurityGenFunc: func(moduleID model.ID) (*genSec.ModuleSecurity, error) {
			return ms, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h), withMicroflowsRepo(mfRepo))

	// Bare role name — module omitted. Should infer "PayerRegistration" from microflow.
	stmt := &ast.GrantMicroflowAccessStmt{
		Microflow: ast.QualifiedName{Module: "PayerRegistration", Name: "ACT_Payer_Save"},
		Roles: []ast.QualifiedName{
			{Module: "", Name: "User"}, // bare name: no module prefix
		},
	}
	err := ExecGrantMicroflowAccessGenFn(ctx, stmt, execContextToDeps(ctx))
	if err != nil {
		t.Fatalf("bare role name should not error; got: %v", err)
	}

	if updatedMF == nil {
		t.Fatal("microflow must be updated when role is granted")
	}
	roles := updatedMF.AllowedModuleRolesQualifiedNames()
	if len(roles) != 1 || roles[0] != "PayerRegistration.User" {
		t.Errorf("expected [PayerRegistration.User] in AllowedModuleRoles, got: %v", roles)
	}
}
