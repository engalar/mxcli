// SPDX-License-Identifier: Apache-2.0

package syntax

func init() {
	// ── Connection ──────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "connect",
		Summary: "Connect to a Mendix project (.mpr file) for the current session",
		Keywords: []string{
			"connect", "connect local", "connect project",
			"open project", "mpr", "connection",
		},
		Syntax: `CONNECT LOCAL '<path/to/app.mpr>';
CONNECT LOCAL '<path>' BRANCH '<branch>';

-- CLI flags (equivalent)
mxcli -p <path/to/app.mpr> -c "<statement>"`,
		Example: `CONNECT LOCAL '/projects/MyApp/MyApp.mpr';

-- Read-write session
CONNECT LOCAL '/projects/MyApp/MyApp.mpr';
CREATE ENTITY MyModule.Product ( Name: String(200) );
DISCONNECT;`,
		SeeAlso: []string{"disconnect", "status"},
	})

	Register(SyntaxFeature{
		Path:    "disconnect",
		Summary: "Close the current project connection",
		Keywords: []string{
			"disconnect", "close", "close connection", "close project",
		},
		Syntax:  "DISCONNECT;",
		Example: "DISCONNECT;",
		SeeAlso: []string{"connect", "status"},
	})

	Register(SyntaxFeature{
		Path:    "status",
		Summary: "Show the current connection status: project path, Mendix version, and module count",
		Keywords: []string{
			"status", "show status", "connection status",
			"project info", "version", "connected",
		},
		Syntax:  "STATUS;\nSHOW STATUS;",
		Example: "STATUS;\n-- Output: Connected to /projects/MyApp/MyApp.mpr (Mendix 10.24.0, 5 modules)",
		SeeAlso: []string{"connect", "disconnect"},
	})

	// ── Navigation ──────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "navigation",
		Summary: "Navigation profiles — home pages, menus, login pages per device type",
		Keywords: []string{
			"navigation", "nav", "profile", "responsive", "phone", "tablet",
			"home page", "menu", "login page",
		},
		Syntax:  "SHOW NAVIGATION;\nDESCRIBE NAVIGATION [profile];\nCREATE OR REPLACE NAVIGATION <profile> ...;",
		Example: "SHOW NAVIGATION;\nDESCRIBE NAVIGATION Responsive;",
		SeeAlso: []string{"navigation.show", "navigation.create", "navigation.alter"},
	})

	Register(SyntaxFeature{
		Path:    "navigation.show",
		Summary: "List navigation profiles, menus, and home page assignments",
		Keywords: []string{
			"show navigation", "describe navigation", "navigation menu",
			"navigation homes", "list profiles",
		},
		Syntax:  "SHOW NAVIGATION;\nSHOW NAVIGATION MENU;\nSHOW NAVIGATION MENU <profile>;\nSHOW NAVIGATION HOMES;\nDESCRIBE NAVIGATION;\nDESCRIBE NAVIGATION <profile>;",
		Example: "SHOW NAVIGATION;\nSHOW NAVIGATION MENU Responsive;\nDESCRIBE NAVIGATION Responsive;",
	})

	Register(SyntaxFeature{
		Path:    "navigation.create",
		Summary: "Create or replace a navigation profile with home pages, menus, and login page",
		Keywords: []string{
			"create navigation", "replace navigation", "home page",
			"login page", "not found page", "menu item",
		},
		Syntax: `CREATE OR REPLACE NAVIGATION <profile>
  HOME PAGE Module.Page
  [HOME PAGE Module.Page FOR Module.UserRole]
  [LOGIN PAGE Module.LoginPage]
  [NOT FOUND PAGE Module.Custom404]
  [MENU (
    MENU ITEM 'Label' PAGE Module.Page;
    MENU 'Group' ( ... );
  )];`,
		Example: `CREATE OR REPLACE NAVIGATION Responsive
  HOME PAGE MyModule.Home_Web
  HOME PAGE MyModule.AdminDashboard FOR Administration.Administrator
  LOGIN PAGE Administration.Login
  MENU (
    MENU ITEM 'Home' PAGE MyModule.Home_Web;
    MENU 'Orders' (
      MENU ITEM 'All Orders' PAGE Orders.Order_Overview;
      MENU ITEM 'New Order' PAGE Orders.Order_New;
    );
  );`,
		SeeAlso: []string{"navigation.show"},
	})

	Register(SyntaxFeature{
		Path:    "navigation.alter",
		Summary: "Modify navigation via round-trip: DESCRIBE, edit, CREATE OR REPLACE",
		Keywords: []string{
			"alter navigation", "modify navigation", "update navigation",
			"round-trip", "edit menu",
		},
		Syntax:  "-- Round-trip workflow:\n-- 1. DESCRIBE NAVIGATION <profile>;\n-- 2. Copy output, modify\n-- 3. Paste as CREATE OR REPLACE NAVIGATION ...",
		Example: "-- Inspect current state\nDESCRIBE NAVIGATION Responsive;\n-- Copy output, modify, paste back as CREATE OR REPLACE",
		SeeAlso: []string{"navigation.create", "navigation.show"},
	})

	// ── Settings ────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "settings",
		Summary: "Project settings — model, configuration, constants, language, workflows",
		Keywords: []string{
			"settings", "project settings", "configuration",
			"startup", "shutdown", "hash algorithm", "java version",
		},
		Syntax:  "SHOW SETTINGS;\nDESCRIBE SETTINGS;\nALTER SETTINGS MODEL <key> = <value>;\nALTER SETTINGS CONFIGURATION '<name>' <key> = <value>;",
		Example: "SHOW SETTINGS;\nALTER SETTINGS MODEL AfterStartupMicroflow = 'Module.MF_Startup';",
		SeeAlso: []string{"settings.show", "settings.alter"},
	})

	Register(SyntaxFeature{
		Path:    "settings.show",
		Summary: "Show and describe project settings",
		Keywords: []string{
			"show settings", "describe settings", "list settings",
		},
		Syntax:  "SHOW SETTINGS;\nDESCRIBE SETTINGS;",
		Example: "SHOW SETTINGS;\nDESCRIBE SETTINGS;",
	})

	Register(SyntaxFeature{
		Path:    "settings.alter",
		Summary: "Modify project settings — model, configuration, constants, language, workflows",
		Keywords: []string{
			"alter settings", "modify settings", "change settings",
			"after startup", "before shutdown", "hash algorithm",
			"database type", "constant override", "language",
		},
		Syntax: `ALTER SETTINGS MODEL <key> = <value>;
ALTER SETTINGS CONFIGURATION '<name>' <key> = <value>, ...;
ALTER SETTINGS CONSTANT '<qualifiedName>' VALUE '<value>' IN CONFIGURATION '<name>';
ALTER SETTINGS DROP CONSTANT '<qualifiedName>' IN CONFIGURATION '<name>';
ALTER SETTINGS LANGUAGE DefaultLanguageCode = '<code>';
ALTER SETTINGS WORKFLOWS UserEntity = '<qualifiedName>';
CREATE CONFIGURATION '<name>' [<key> = <value>, ...];
DROP CONFIGURATION '<name>';`,
		Example: `ALTER SETTINGS MODEL AfterStartupMicroflow = 'Module.MF_Startup';
ALTER SETTINGS MODEL HashAlgorithm = 'BCrypt';
ALTER SETTINGS CONFIGURATION 'Default'
  DatabaseType = 'PostgreSql',
  DatabaseUrl = 'localhost:5432',
  DatabaseName = 'mydb';
ALTER SETTINGS CONSTANT 'BusinessEvents.ServerUrl' VALUE 'kafka:9092'
  IN CONFIGURATION 'Default';
CREATE CONFIGURATION 'Production'
  DatabaseType = 'POSTGRESQL',
  HttpPortNumber = 8080;`,
		SeeAlso: []string{"settings.show"},
	})

	// ── Structure ───────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "structure",
		Summary: "SHOW STRUCTURE — compact project overview at configurable depth",
		Keywords: []string{
			"structure", "show structure", "project overview",
			"repo map", "module summary", "depth",
		},
		Syntax: "SHOW STRUCTURE [DEPTH 1|2|3] [IN <module>] [ALL];",
		Example: `-- Module counts only
SHOW STRUCTURE DEPTH 1;

-- Elements with signatures (default)
SHOW STRUCTURE;

-- Full types and parameter names
SHOW STRUCTURE DEPTH 3;

-- Focus on one module
SHOW STRUCTURE IN MyModule;

-- Include system modules
SHOW STRUCTURE DEPTH 1 ALL;`,
	})

	// ── Move ────────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "move",
		Summary: "MOVE command — relocate documents between folders and modules",
		Keywords: []string{
			"move", "relocate", "folder", "cross-module move",
			"move page", "move microflow", "move entity",
			"move folder", "drop folder",
		},
		Syntax: `MOVE <doctype> Module.Name TO FOLDER 'Path';
MOVE <doctype> Module.Name TO TargetModule;
MOVE <doctype> OldModule.Name TO FOLDER 'Path' IN NewModule;
MOVE FOLDER Module.FolderName TO FOLDER 'Path';
DROP FOLDER 'Path' IN Module;`,
		Example: `-- Move page to a folder
MOVE PAGE MyModule.CustomerEdit TO FOLDER 'Customers';

-- Move microflow to nested folder
MOVE MICROFLOW MyModule.ACT_ProcessOrder TO FOLDER 'Orders/Processing';

-- Move entity to different module
MOVE ENTITY OldModule.Customer TO NewModule;

-- Check impact before cross-module move
SHOW IMPACT OF OldModule.CustomerPage;
MOVE PAGE OldModule.CustomerPage TO NewModule;

-- Drop empty folder
DROP FOLDER 'OldFolder' IN Module;`,
	})

	// ── Search ──────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "search",
		Summary: "Full-text search across project strings and source definitions",
		Keywords: []string{
			"search", "full-text search", "find", "grep",
			"fts", "catalog strings", "catalog source",
		},
		Syntax: `SEARCH '<query>';

-- CLI
mxcli search -p app.mpr "<query>" [--format table|names|json] [-q]

-- Raw FTS queries
SELECT * FROM CATALOG.STRINGS WHERE strings MATCH '<query>';
SELECT * FROM CATALOG.SOURCE WHERE source MATCH '<query>';`,
		Example: `SEARCH 'validation';
SEARCH 'Customer';

-- CLI with piping
mxcli search -p app.mpr "validation" -q --format names

-- FTS5 operators
SEARCH 'word1 OR word2';
SEARCH '"exact phrase"';
SEARCH 'word*';`,
	})

	// ── Testing ────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "test",
		Summary: "Microflow testing — run .test.mdl or .test.md files against a Mendix project in Docker",
		Keywords: []string{
			"test", "testing", "microflow test", "nanoflow test",
			"test.mdl", "test.md", "junit", "docker",
			"@test", "@expect", "@throws", "@cleanup",
		},
		Syntax: `mxcli test <file|dir> -p app.mpr [flags]

Flags:
  -l, --list          List tests without executing
  -j, --junit FILE    Write JUnit XML results
  -s, --skip-build    Skip Docker build (reuse existing)
  -v, --verbose       Show runtime log lines
  -t, --timeout DUR   Runtime startup timeout (default: 5m)

Annotations:
  @test <name>              Test name (required)
  @expect $var = value      Assert variable equals value
  @expect $obj/Attr = val   Assert entity attribute
  @throws 'message'         Expect error
  @cleanup rollback|none    Cleanup strategy (default: rollback)`,
		Example: `-- .test.mdl file format
/**
 * @test String concatenation
 * @expect $result = 'John Doe'
 */
$result = CALL MICROFLOW MyModule.ConcatNames(
  FirstName = 'John', LastName = 'Doe'
);
/

-- Run tests
mxcli test tests/ -p app.mpr
mxcli test tests/ -p app.mpr --junit results.xml`,
	})

	// ── Errors ──────────────────────────────────────────────────────────

	Register(SyntaxFeature{
		Path:    "errors",
		Summary: "Common validation errors and how to fix them",
		Keywords: []string{
			"errors", "validation", "syntax error", "reference error",
			"reserved keyword", "module not found", "entity not found",
			"check", "troubleshooting",
		},
		Syntax: `mxcli check script.mdl                    -- Syntax + anti-pattern check
mxcli check script.mdl -p app.mpr --references  -- With reference validation`,
		Example: `-- Reserved keyword as identifier
-- Error:  mismatched input 'Title' expecting IDENTIFIER
-- Fix:    Use quoted identifiers: "Title"

-- Module not found
-- Error:  module not found: ModuleName
-- Fix:    CREATE MODULE ModuleName;

-- Missing module prefix on enumeration
-- Error:  enumeration reference 'X' is missing module prefix
-- Fix:    Use Enumeration(MyModule.Status)

-- Invalid association path in OQL (dot instead of slash)
-- Wrong:  WHERE l.Library.Loan_Member = m.ID
-- Right:  WHERE l/Library.Loan_Member = m.ID`,
		SeeAlso: []string{"errors.syntax", "errors.reference", "errors.execution"},
	})

	Register(SyntaxFeature{
		Path:    "errors.syntax",
		Summary: "Syntax errors — reserved keywords, invalid types, malformed enumerations",
		Keywords: []string{
			"syntax error", "reserved keyword", "invalid type",
			"malformed enumeration", "parse error", "mismatched input",
		},
		Syntax: "mxcli check script.mdl",
		Example: `-- Reserved keyword used as identifier
-- Error:  mismatched input 'Title' expecting IDENTIFIER
-- Fix:    Use quoted identifiers: "Title", "ComboBox"."Entity"
-- Alt:    Rename to avoid keyword: BookTitle, OrderStatus

-- Invalid data type
-- Error:  Unknown type parsed as enumeration reference
-- Fix:    Use correct type: DateTime (not DateAndTime)

-- Malformed enumeration
-- Error:  Invalid enumeration value: each value must have a name
-- Fix:    Use syntax: ValueName 'Caption'`,
	})

	Register(SyntaxFeature{
		Path:    "errors.reference",
		Summary: "Reference errors — missing modules, entities, enumerations",
		Keywords: []string{
			"reference error", "module not found", "entity not found",
			"enumeration not found", "missing module prefix",
		},
		Syntax: "mxcli check script.mdl -p app.mpr --references",
		Example: `-- Module not found
-- Error:  module not found: ModuleName
-- Fix:    CREATE MODULE ModuleName;

-- Enumeration not found
-- Error:  attribute 'X': enumeration not found: Module.EnumName
-- Fix:    Create the enumeration first, or check spelling

-- Missing module prefix on enumeration
-- Error:  enumeration reference 'X' is missing module prefix
-- Fix:    Use fully qualified name: Enumeration(MyModule.Status)`,
	})

	Register(SyntaxFeature{
		Path:    "module.jar-dependencies",
		Summary: "Manage Maven/JAR dependencies in a module's settings",
		Keywords: []string{
			"jar dependency", "maven", "jar dep", "module settings",
			"group", "artifact", "classpath", "exclusion",
		},
		Syntax: `LIST JAR DEPENDENCIES [IN <module>];
DESCRIBE JAR DEPENDENCY <module> '<group:artifact>';
ALTER MODULE <name>
  ADD JAR DEPENDENCY (
    group    = '<group>',
    artifact = '<artifact>',
    version  = '<version>',
    included = true|false,
  );
ALTER MODULE <name> SET JAR DEPENDENCY '<group:artifact>' VERSION '<version>';
ALTER MODULE <name> SET JAR DEPENDENCY '<group:artifact>' INCLUDED true|false;
ALTER MODULE <name> SET JAR DEPENDENCY '<group:artifact>' ADD EXCLUSION '<group:artifact>';
ALTER MODULE <name> SET JAR DEPENDENCY '<group:artifact>' DROP EXCLUSION '<group:artifact>';
ALTER MODULE <name> DROP JAR DEPENDENCY '<group:artifact>';`,
		Example: `-- Add a new JAR dependency to a module
ALTER MODULE MyModule
  ADD JAR DEPENDENCY (
    group    = 'org.duckdb',
    artifact = 'duckdb_jdbc',
    version  = '1.1.3',
    included = true,
  );

-- Update the version
ALTER MODULE MyModule SET JAR DEPENDENCY 'org.duckdb:duckdb_jdbc' VERSION '1.2.0';

-- Exclude a transitive dependency
ALTER MODULE MyModule SET JAR DEPENDENCY 'org.duckdb:duckdb_jdbc' ADD EXCLUSION 'com.example:unwanted';

-- List all jar dependencies
LIST JAR DEPENDENCIES;
LIST JAR DEPENDENCIES IN MyModule;

-- Describe (outputs roundtrippable MDL)
DESCRIBE JAR DEPENDENCY MyModule 'org.duckdb:duckdb_jdbc';

-- Remove a dependency
ALTER MODULE MyModule DROP JAR DEPENDENCY 'org.duckdb:duckdb_jdbc';`,
	})

	Register(SyntaxFeature{
		Path:    "errors.execution",
		Summary: "Execution errors — entity exists, type mismatches, validation failures",
		Keywords: []string{
			"execution error", "entity already exists", "type mismatch",
			"boolean default", "view entity", "microflow validation",
		},
		Syntax: "mxcli check script.mdl -p app.mpr --references",
		Example: `-- Entity already exists
-- Error:  entity already exists: Module.Entity
-- Fix:    Use CREATE OR MODIFY ENTITY to update existing entities

-- Boolean without default
-- Note:   Boolean attributes auto-default to false

-- OQL invalid association path (dot vs slash)
-- Wrong:  WHERE l.Library.Loan_Member = m.ID
-- Right:  WHERE l/Library.Loan_Member = m.ID`,
	})
}
