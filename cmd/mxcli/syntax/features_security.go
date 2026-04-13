// SPDX-License-Identifier: Apache-2.0

package syntax

func init() {
	Register(SyntaxFeature{
		Path:    "security",
		Summary: "Application security: roles, access control, demo users",
		Keywords: []string{
			"security", "access control", "roles", "permissions",
			"grant", "revoke", "authentication", "authorization",
		},
		Syntax: "SHOW PROJECT SECURITY;\nSHOW MODULE ROLES [IN <module>];\nSHOW USER ROLES;\nSHOW SECURITY MATRIX [IN <module>];",
		Example: "SHOW PROJECT SECURITY;\nSHOW SECURITY MATRIX IN Shop;",
		SeeAlso: []string{"security.module-role", "security.entity-access", "security.user-role"},
	})

	Register(SyntaxFeature{
		Path:    "security.module-role",
		Summary: "Create and manage module-level security roles",
		Keywords: []string{
			"module role", "create role", "drop role",
		},
		Syntax: "CREATE MODULE ROLE <module>.<role> [DESCRIPTION '<text>'];\nDROP MODULE ROLE <module>.<role>;",
		Example: "CREATE MODULE ROLE Shop.Admin DESCRIPTION 'Full access';\nCREATE MODULE ROLE Shop.User DESCRIPTION 'Read-only access';",
		SeeAlso: []string{"security.user-role", "security.entity-access"},
	})

	Register(SyntaxFeature{
		Path:    "security.entity-access",
		Summary: "Grant or revoke entity-level access (CRUD, attribute-level, XPath rules)",
		Keywords: []string{
			"entity access", "grant", "revoke", "read", "write",
			"create", "delete", "xpath", "row-level security",
		},
		Syntax: "GRANT <role> ON <module>.<entity> (<rights>) [WHERE '<xpath>'];\nREVOKE <role> ON <module>.<entity>;\nREVOKE <role> ON <module>.<entity> (<rights>);\n\nRights: CREATE, DELETE, READ *, READ (<attr>,...), WRITE *, WRITE (<attr>,...)",
		Example: "GRANT Shop.Admin ON Shop.Customer (CREATE, DELETE, READ *, WRITE *);\nGRANT Shop.User ON Shop.Customer (READ *) WHERE '[Active = true()]';",
		SeeAlso: []string{"security.module-role", "security.microflow-access"},
	})

	Register(SyntaxFeature{
		Path:    "security.microflow-access",
		Summary: "Grant or revoke execution rights on microflows",
		Keywords: []string{
			"microflow access", "execute", "grant microflow",
			"revoke microflow",
		},
		Syntax: "GRANT EXECUTE ON MICROFLOW <module>.<name> TO <role> [, <role>...];\nREVOKE EXECUTE ON MICROFLOW <module>.<name> FROM <role> [, <role>...];",
		Example: "GRANT EXECUTE ON MICROFLOW Shop.ProcessOrder TO Shop.Admin, Shop.User;\nREVOKE EXECUTE ON MICROFLOW Shop.ProcessOrder FROM Shop.User;",
		SeeAlso: []string{"security.page-access", "security.entity-access"},
	})

	Register(SyntaxFeature{
		Path:    "security.page-access",
		Summary: "Grant or revoke view rights on pages",
		Keywords: []string{
			"page access", "view", "grant page", "revoke page",
		},
		Syntax: "GRANT VIEW ON PAGE <module>.<name> TO <role> [, <role>...];\nREVOKE VIEW ON PAGE <module>.<name> FROM <role> [, <role>...];",
		Example: "GRANT VIEW ON PAGE Shop.OrderOverview TO Shop.Admin, Shop.User;",
		SeeAlso: []string{"security.microflow-access", "security.entity-access"},
	})

	Register(SyntaxFeature{
		Path:    "security.user-role",
		Summary: "Create and manage application-level user roles that bundle module roles",
		Keywords: []string{
			"user role", "application role", "manage roles",
			"add module roles", "remove module roles",
		},
		Syntax: "CREATE USER ROLE <name> (<role> [, ...]) [MANAGE ALL ROLES];\nALTER USER ROLE <name> ADD MODULE ROLES (<role> [, ...]);\nALTER USER ROLE <name> REMOVE MODULE ROLES (<role> [, ...]);\nDROP USER ROLE <name>;",
		Example: "CREATE USER ROLE AppAdmin (Shop.Admin, HR.Admin) MANAGE ALL ROLES;\nALTER USER ROLE AppAdmin ADD MODULE ROLES (Reporting.Viewer);",
		SeeAlso: []string{"security.module-role", "security.demo-user"},
	})

	Register(SyntaxFeature{
		Path:    "security.project-security",
		Summary: "Set project security level and demo user toggle",
		Keywords: []string{
			"project security", "security level", "prototype",
			"production", "off",
		},
		Syntax: "ALTER PROJECT SECURITY LEVEL OFF|PROTOTYPE|PRODUCTION;\nALTER PROJECT SECURITY DEMO USERS ON|OFF;",
		Example: "ALTER PROJECT SECURITY LEVEL PRODUCTION;\nALTER PROJECT SECURITY DEMO USERS OFF;",
		SeeAlso: []string{"security.demo-user"},
	})

	Register(SyntaxFeature{
		Path:    "security.demo-user",
		Summary: "Create and manage demo users for testing",
		Keywords: []string{
			"demo user", "test user", "demo account",
			"password", "login",
		},
		Syntax: "CREATE DEMO USER '<name>' PASSWORD '<pass>' [ENTITY Module.Entity] (<userrole> [, ...]);\nDROP DEMO USER '<name>';",
		Example: "CREATE DEMO USER 'admin' PASSWORD 'Admin1!' (AppAdmin);\nCREATE DEMO USER 'user' PASSWORD 'User1!' (AppUser);",
		SeeAlso: []string{"security.user-role", "security.project-security"},
	})
}
