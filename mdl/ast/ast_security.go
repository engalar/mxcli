// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Security Statements
// ============================================================================

// CreateModuleRoleStmt represents: CREATE MODULE ROLE Module.RoleName [DESCRIPTION '...']
type CreateModuleRoleStmt struct {
	Name        QualifiedName
	Description string
}

func (s *CreateModuleRoleStmt) isStatement() {}
func (s *CreateModuleRoleStmt) TypeName() string { return "CreateModuleRole" }

// DropModuleRoleStmt represents: DROP MODULE ROLE Module.RoleName
type DropModuleRoleStmt struct {
	Name QualifiedName
}

func (s *DropModuleRoleStmt) isStatement() {}
func (s *DropModuleRoleStmt) TypeName() string { return "DropModuleRole" }

// CreateUserRoleStmt represents: CREATE [OR MODIFY] USER ROLE Name (ModuleRole, ...) [MANAGE ALL ROLES]
type CreateUserRoleStmt struct {
	Name           string
	ModuleRoles    []QualifiedName
	ManageAllRoles bool
	CreateOrModify bool // If true, adds module roles to existing role instead of failing
}

func (s *CreateUserRoleStmt) isStatement() {}
func (s *CreateUserRoleStmt) TypeName() string { return "CreateUserRole" }

// AlterUserRoleStmt represents: ALTER USER ROLE Name ADD/REMOVE MODULE ROLES (...)
type AlterUserRoleStmt struct {
	Name        string
	Add         bool // true = ADD, false = REMOVE
	ModuleRoles []QualifiedName
}

func (s *AlterUserRoleStmt) isStatement() {}
func (s *AlterUserRoleStmt) TypeName() string { return "AlterUserRole" }

// DropUserRoleStmt represents: DROP USER ROLE Name
type DropUserRoleStmt struct {
	Name string
}

func (s *DropUserRoleStmt) isStatement() {}
func (s *DropUserRoleStmt) TypeName() string { return "DropUserRole" }

// EntityAccessRight represents a single access right in a GRANT statement.
type EntityAccessRight struct {
	Type    EntityAccessRightType
	Members []string // For READ/WRITE with specific members
}

// EntityAccessRightType represents the type of entity access right.
type EntityAccessRightType int

const (
	EntityAccessCreate EntityAccessRightType = iota
	EntityAccessDelete
	EntityAccessReadAll      // READ *
	EntityAccessReadMembers  // READ (member1, member2)
	EntityAccessWriteAll     // WRITE *
	EntityAccessWriteMembers // WRITE (member1, member2)
)

// GrantEntityAccessStmt represents: GRANT role1, role2 ON Module.Entity (CREATE, DELETE, READ *, WRITE *) [WHERE '...']
type GrantEntityAccessStmt struct {
	Roles           []QualifiedName
	Entity          QualifiedName
	Rights          []EntityAccessRight
	XPathConstraint string // Optional WHERE clause
}

func (s *GrantEntityAccessStmt) isStatement() {}
func (s *GrantEntityAccessStmt) TypeName() string { return "GrantEntityAccess" }

// RevokeEntityAccessStmt represents: REVOKE role1, role2 ON Module.Entity [(rights...)]
// When Rights is nil, the entire access rule is removed. When non-nil, only the
// specified rights are revoked (partial revoke).
type RevokeEntityAccessStmt struct {
	Roles  []QualifiedName
	Entity QualifiedName
	Rights []EntityAccessRight // nil = full revoke, non-nil = partial
}

func (s *RevokeEntityAccessStmt) isStatement() {}
func (s *RevokeEntityAccessStmt) TypeName() string { return "RevokeEntityAccess" }

// GrantMicroflowAccessStmt represents: GRANT EXECUTE ON MICROFLOW Module.MF TO role1, role2
type GrantMicroflowAccessStmt struct {
	Microflow QualifiedName
	Roles     []QualifiedName
}

func (s *GrantMicroflowAccessStmt) isStatement() {}
func (s *GrantMicroflowAccessStmt) TypeName() string { return "GrantMicroflowAccess" }

// RevokeMicroflowAccessStmt represents: REVOKE EXECUTE ON MICROFLOW Module.MF FROM role1, role2
type RevokeMicroflowAccessStmt struct {
	Microflow QualifiedName
	Roles     []QualifiedName
}

func (s *RevokeMicroflowAccessStmt) isStatement() {}
func (s *RevokeMicroflowAccessStmt) TypeName() string { return "RevokeMicroflowAccess" }

// GrantNanoflowAccessStmt represents: GRANT EXECUTE ON NANOFLOW Module.NF TO role1, role2
type GrantNanoflowAccessStmt struct {
	Nanoflow QualifiedName
	Roles    []QualifiedName
}

func (s *GrantNanoflowAccessStmt) isStatement() {}
func (s *GrantNanoflowAccessStmt) TypeName() string { return "GrantNanoflowAccess" }

// RevokeNanoflowAccessStmt represents: REVOKE EXECUTE ON NANOFLOW Module.NF FROM role1, role2
type RevokeNanoflowAccessStmt struct {
	Nanoflow QualifiedName
	Roles    []QualifiedName
}

func (s *RevokeNanoflowAccessStmt) isStatement() {}
func (s *RevokeNanoflowAccessStmt) TypeName() string { return "RevokeNanoflowAccess" }

// GrantPageAccessStmt represents: GRANT VIEW ON PAGE Module.Page TO role1, role2
type GrantPageAccessStmt struct {
	Page  QualifiedName
	Roles []QualifiedName
}

func (s *GrantPageAccessStmt) isStatement() {}
func (s *GrantPageAccessStmt) TypeName() string { return "GrantPageAccess" }

// RevokePageAccessStmt represents: REVOKE VIEW ON PAGE Module.Page FROM role1, role2
type RevokePageAccessStmt struct {
	Page  QualifiedName
	Roles []QualifiedName
}

func (s *RevokePageAccessStmt) isStatement() {}
func (s *RevokePageAccessStmt) TypeName() string { return "RevokePageAccess" }

// GrantWorkflowAccessStmt represents: GRANT EXECUTE ON WORKFLOW Module.WF TO role1, role2
type GrantWorkflowAccessStmt struct {
	Workflow QualifiedName
	Roles    []QualifiedName
}

func (s *GrantWorkflowAccessStmt) isStatement() {}
func (s *GrantWorkflowAccessStmt) TypeName() string { return "GrantWorkflowAccess" }

// RevokeWorkflowAccessStmt represents: REVOKE EXECUTE ON WORKFLOW Module.WF FROM role1, role2
type RevokeWorkflowAccessStmt struct {
	Workflow QualifiedName
	Roles    []QualifiedName
}

func (s *RevokeWorkflowAccessStmt) isStatement() {}
func (s *RevokeWorkflowAccessStmt) TypeName() string { return "RevokeWorkflowAccess" }

// GrantODataServiceAccessStmt represents: GRANT ACCESS ON ODATA SERVICE Module.Svc TO role1, role2
type GrantODataServiceAccessStmt struct {
	Service QualifiedName
	Roles   []QualifiedName
}

func (s *GrantODataServiceAccessStmt) isStatement() {}
func (s *GrantODataServiceAccessStmt) TypeName() string { return "GrantODataServiceAccess" }

// RevokeODataServiceAccessStmt represents: REVOKE ACCESS ON ODATA SERVICE Module.Svc FROM role1, role2
type RevokeODataServiceAccessStmt struct {
	Service QualifiedName
	Roles   []QualifiedName
}

func (s *RevokeODataServiceAccessStmt) isStatement() {}
func (s *RevokeODataServiceAccessStmt) TypeName() string { return "RevokeODataServiceAccess" }

// GrantPublishedRestServiceAccessStmt represents: GRANT ACCESS ON PUBLISHED REST SERVICE Module.Svc TO role1, role2
type GrantPublishedRestServiceAccessStmt struct {
	Service QualifiedName
	Roles   []QualifiedName
}

func (s *GrantPublishedRestServiceAccessStmt) isStatement() {}
func (s *GrantPublishedRestServiceAccessStmt) TypeName() string { return "GrantPublishedRestServiceAccess" }

// RevokePublishedRestServiceAccessStmt represents: REVOKE ACCESS ON PUBLISHED REST SERVICE Module.Svc FROM role1, role2
type RevokePublishedRestServiceAccessStmt struct {
	Service QualifiedName
	Roles   []QualifiedName
}

func (s *RevokePublishedRestServiceAccessStmt) isStatement() {}
func (s *RevokePublishedRestServiceAccessStmt) TypeName() string { return "RevokePublishedRestServiceAccess" }

// AlterProjectSecurityStmt represents ALTER PROJECT SECURITY commands.
type AlterProjectSecurityStmt struct {
	// SecurityLevel is set for ALTER PROJECT SECURITY LEVEL (PRODUCTION|PROTOTYPE|OFF)
	SecurityLevel string
	// DemoUsersEnabled is set for ALTER PROJECT SECURITY DEMO USERS ON/OFF
	DemoUsersEnabled *bool
	// PasswordPolicy is set for ALTER PROJECT SECURITY PASSWORD POLICY (...)
	PasswordPolicy *AlterPasswordPolicyOptions
}

func (s *AlterProjectSecurityStmt) isStatement() {}
func (s *AlterProjectSecurityStmt) TypeName() string { return "AlterProjectSecurity" }

// AlterPasswordPolicyOptions holds the parsed options from
// ALTER PROJECT SECURITY PASSWORD POLICY (min_length: N, ...).
// Nil pointer fields mean "not specified, keep existing value".
type AlterPasswordPolicyOptions struct {
	MinLength        *int32
	RequireDigit     *bool
	RequireMixedCase *bool
	RequireSymbol    *bool
}

// CreateDemoUserStmt represents: CREATE [OR MODIFY] DEMO USER 'name' PASSWORD 'pw' [ENTITY Module.Entity] (Role1, Role2)
type CreateDemoUserStmt struct {
	UserName       string
	Password       string
	Entity         string // qualified name of user entity, e.g. "Administration.Account"
	UserRoles      []string
	CreateOrModify bool // If true, updates existing user's roles additively
}

func (s *CreateDemoUserStmt) isStatement() {}
func (s *CreateDemoUserStmt) TypeName() string { return "CreateDemoUser" }

// DropDemoUserStmt represents: DROP DEMO USER 'name'
type DropDemoUserStmt struct {
	UserName string
}

func (s *DropDemoUserStmt) isStatement() {}
func (s *DropDemoUserStmt) TypeName() string { return "DropDemoUser" }

// UpdateSecurityStmt represents: UPDATE SECURITY [IN Module]
type UpdateSecurityStmt struct {
	Module string // optional, empty = all modules
}

func (s *UpdateSecurityStmt) isStatement() {}
func (s *UpdateSecurityStmt) TypeName() string { return "UpdateSecurity" }
