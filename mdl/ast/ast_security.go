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

// DropModuleRoleStmt represents: DROP MODULE ROLE Module.RoleName
type DropModuleRoleStmt struct {
	Name QualifiedName
}

func (s *DropModuleRoleStmt) isStatement() {}

// CreateUserRoleStmt represents: CREATE USER ROLE Name (ModuleRole, ...) [MANAGE ALL ROLES]
type CreateUserRoleStmt struct {
	Name           string
	ModuleRoles    []QualifiedName
	ManageAllRoles bool
}

func (s *CreateUserRoleStmt) isStatement() {}

// AlterUserRoleStmt represents: ALTER USER ROLE Name ADD/REMOVE MODULE ROLES (...)
type AlterUserRoleStmt struct {
	Name        string
	Add         bool // true = ADD, false = REMOVE
	ModuleRoles []QualifiedName
}

func (s *AlterUserRoleStmt) isStatement() {}

// DropUserRoleStmt represents: DROP USER ROLE Name
type DropUserRoleStmt struct {
	Name string
}

func (s *DropUserRoleStmt) isStatement() {}

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

// RevokeEntityAccessStmt represents: REVOKE role1, role2 ON Module.Entity
type RevokeEntityAccessStmt struct {
	Roles  []QualifiedName
	Entity QualifiedName
}

func (s *RevokeEntityAccessStmt) isStatement() {}

// GrantMicroflowAccessStmt represents: GRANT EXECUTE ON MICROFLOW Module.MF TO role1, role2
type GrantMicroflowAccessStmt struct {
	Microflow QualifiedName
	Roles     []QualifiedName
}

func (s *GrantMicroflowAccessStmt) isStatement() {}

// RevokeMicroflowAccessStmt represents: REVOKE EXECUTE ON MICROFLOW Module.MF FROM role1, role2
type RevokeMicroflowAccessStmt struct {
	Microflow QualifiedName
	Roles     []QualifiedName
}

func (s *RevokeMicroflowAccessStmt) isStatement() {}

// GrantPageAccessStmt represents: GRANT VIEW ON PAGE Module.Page TO role1, role2
type GrantPageAccessStmt struct {
	Page  QualifiedName
	Roles []QualifiedName
}

func (s *GrantPageAccessStmt) isStatement() {}

// RevokePageAccessStmt represents: REVOKE VIEW ON PAGE Module.Page FROM role1, role2
type RevokePageAccessStmt struct {
	Page  QualifiedName
	Roles []QualifiedName
}

func (s *RevokePageAccessStmt) isStatement() {}

// GrantWorkflowAccessStmt represents: GRANT EXECUTE ON WORKFLOW Module.WF TO role1, role2
type GrantWorkflowAccessStmt struct {
	Workflow QualifiedName
	Roles    []QualifiedName
}

func (s *GrantWorkflowAccessStmt) isStatement() {}

// RevokeWorkflowAccessStmt represents: REVOKE EXECUTE ON WORKFLOW Module.WF FROM role1, role2
type RevokeWorkflowAccessStmt struct {
	Workflow QualifiedName
	Roles    []QualifiedName
}

func (s *RevokeWorkflowAccessStmt) isStatement() {}

// GrantODataServiceAccessStmt represents: GRANT ACCESS ON ODATA SERVICE Module.Svc TO role1, role2
type GrantODataServiceAccessStmt struct {
	Service QualifiedName
	Roles   []QualifiedName
}

func (s *GrantODataServiceAccessStmt) isStatement() {}

// RevokeODataServiceAccessStmt represents: REVOKE ACCESS ON ODATA SERVICE Module.Svc FROM role1, role2
type RevokeODataServiceAccessStmt struct {
	Service QualifiedName
	Roles   []QualifiedName
}

func (s *RevokeODataServiceAccessStmt) isStatement() {}

// AlterProjectSecurityStmt represents ALTER PROJECT SECURITY commands.
type AlterProjectSecurityStmt struct {
	// SecurityLevel is set for ALTER PROJECT SECURITY LEVEL (PRODUCTION|PROTOTYPE|OFF)
	SecurityLevel string
	// DemoUsersEnabled is set for ALTER PROJECT SECURITY DEMO USERS ON/OFF
	DemoUsersEnabled *bool
}

func (s *AlterProjectSecurityStmt) isStatement() {}

// CreateDemoUserStmt represents: CREATE DEMO USER 'name' PASSWORD 'pw' [ENTITY Module.Entity] (Role1, Role2)
type CreateDemoUserStmt struct {
	UserName  string
	Password  string
	Entity    string // qualified name of user entity, e.g. "Administration.Account"
	UserRoles []string
}

func (s *CreateDemoUserStmt) isStatement() {}

// DropDemoUserStmt represents: DROP DEMO USER 'name'
type DropDemoUserStmt struct {
	UserName string
}

func (s *DropDemoUserStmt) isStatement() {}

// UpdateSecurityStmt represents: UPDATE SECURITY [IN Module]
type UpdateSecurityStmt struct {
	Module string // optional, empty = all modules
}

func (s *UpdateSecurityStmt) isStatement() {}
