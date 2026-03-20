// SPDX-License-Identifier: Apache-2.0

// Package executor - Miscellaneous commands (UPDATE, REFRESH, SET, HELP, EXIT, EXECUTE SCRIPT)
package executor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// ErrExit is a sentinel error indicating clean script/session termination.
// Use errors.Is(err, ErrExit) to detect exit requests.
var ErrExit = errors.New("exit")

// execUpdate handles UPDATE statements (refresh from disk).
func (e *Executor) execUpdate() error {
	if e.mprPath == "" {
		return fmt.Errorf("not connected to a project")
	}

	// Reconnect to refresh
	path := e.mprPath
	e.execDisconnect()
	return e.execConnect(&ast.ConnectStmt{Path: path})
}

// execRefresh handles REFRESH statements (alias for UPDATE).
func (e *Executor) execRefresh() error {
	return e.execUpdate()
}

// execSet handles SET statements.
func (e *Executor) execSet(s *ast.SetStmt) error {
	e.settings[s.Key] = s.Value
	fmt.Fprintf(e.output, "Set %s = %v\n", s.Key, s.Value)
	return nil
}

// execHelp handles HELP statements.
func (e *Executor) execHelp() error {
	help := `MDL Commands:

Connection:
  CONNECT LOCAL '<path>'      Connect to local .mpr file
  DISCONNECT                  Disconnect from project
  STATUS                      Show connection status

Domain Model - Enumerations:
  /** Documentation */
  CREATE ENUMERATION Module.Name (
    VALUE1 'Caption1',
    VALUE2 'Caption2'
  );

  DROP ENUMERATION Module.Name;
  SHOW ENUMERATIONS [IN Module];
  DESCRIBE ENUMERATION Module.Name;

Domain Model - Entities:
  /** Entity documentation */
  @Position(x, y)
  CREATE [OR MODIFY] PERSISTENT|NON-PERSISTENT ENTITY Module.Name (
    /** Attribute documentation */
    AttrName: Type [NOT NULL [ERROR 'msg']] [UNIQUE [ERROR 'msg']] [DEFAULT value]
  )
  [INDEX (col1, col2 DESC)];
  /

  CREATE VIEW ENTITY Module.Name (
    AttrName: Type
  ) AS
    SELECT ... FROM ... WHERE ...;
  /

  DROP ENTITY Module.Name;
  SHOW ENTITIES [IN Module];
  DESCRIBE ENTITY Module.Name;

Domain Model - Associations:
  CREATE ASSOCIATION Module.Name
    FROM Module.Parent
    TO Module.Child
    TYPE Reference|ReferenceSet
    [OWNER Default|Both|Parent|Child]
    [DELETE_BEHAVIOR DELETE_BUT_KEEP_REFERENCES|DELETE_CASCADE];
  /

  DROP ASSOCIATION Module.Name;
  SHOW ASSOCIATIONS [IN Module];
  DESCRIBE ASSOCIATION Module.Name;

Microflows:
  /** Documentation */
  CREATE MICROFLOW Module.Name (
    $Param1: Type,
    $Param2: Module.Entity
  )
  RETURNS ReturnType AS $ReturnVar
  [FOLDER 'folder/path']
  BEGIN
    DECLARE $Var Type = value;           -- Declare primitive variable
    DECLARE $Entity AS Module.Entity;    -- Declare entity variable
    SET $Var = expression;               -- Change variable (must be declared)
    IF condition THEN ... END IF;        -- Conditional
    LOOP $Item IN $List BEGIN ... END LOOP;
    $Result = CREATE Module.Entity (attr = value);
    CHANGE $Object (attr = value);
    COMMIT $Object [WITH EVENTS] [REFRESH];
    RETRIEVE $List FROM Module.Entity WHERE condition;
    $Var = CALL MICROFLOW Module.Name($param = value);
    $Var = CALL JAVA ACTION Module.Name($param = value);
    VALIDATION FEEDBACK $Var/Attr MESSAGE 'Error';  -- Show validation error
    CLOSE PAGE [n];                      -- Close page(s)
    LOG INFO|WARNING|ERROR [NODE 'name'] 'message';
    @annotation 'text'                   -- Visual annotation on next activity
    @caption 'text'                      -- Custom caption for activity
    @color Green                         -- Background color for activity
    @position(100, 200)                  -- Canvas position for activity
    RETURN $ReturnVar;
  END;
  /

  DROP MICROFLOW Module.Name;
  SHOW MICROFLOWS [IN Module];
  SHOW NANOFLOWS [IN Module];
  DESCRIBE MICROFLOW Module.Name;

Pages, Snippets, Layouts, Java Actions:
  CREATE [OR REPLACE] PAGE Module.Name (...) { ... };
  DROP PAGE Module.Name;
  CREATE [OR REPLACE] SNIPPET Module.Name (...) { ... };
  DROP SNIPPET Module.Name;
  SHOW PAGES [IN Module];
  SHOW SNIPPETS [IN Module];
  SHOW LAYOUTS [IN Module];
  SHOW JAVA ACTIONS [IN Module];
  DESCRIBE PAGE Module.Name;
  DESCRIBE SNIPPET Module.Name;

Widget Discovery and Bulk Updates (requires REFRESH CATALOG FULL):
  *** EXPERIMENTAL: Untested proof-of-concept. Use DRY RUN first! ***

  SHOW WIDGETS [WHERE condition] [IN Module];
    WHERE conditions: WidgetType LIKE '%pattern%', Name = 'value'

  UPDATE WIDGETS
    SET 'property' = value [, 'property' = value]
    WHERE condition [AND condition]
    [IN Module]
    [DRY RUN];

  Examples:
    SHOW WIDGETS WHERE WidgetType LIKE '%combobox%';
    UPDATE WIDGETS SET 'showLabel' = false WHERE WidgetType LIKE '%DataGrid%' DRY RUN;

  Always backup your project before applying changes without DRY RUN.

Catalog Queries:
  SHOW CATALOG TABLES;
  SHOW CATALOG STATUS;             Show cache information
  DESCRIBE CATALOG.tablename;      Show table columns and required mode
  REFRESH CATALOG;                 Rebuild catalog (uses cache if valid)
  REFRESH CATALOG FULL;            Full mode with activities/widgets/refs
  REFRESH CATALOG FULL SOURCE;     Full + MDL source for full-text search
  REFRESH CATALOG [FULL] FORCE;    Force rebuild (ignore cache)
  REFRESH CATALOG [FULL] BACKGROUND; Build in background
  SELECT columns FROM CATALOG.tablename
    [WHERE condition]
    [GROUP BY column [HAVING condition]]
    [ORDER BY column [ASC|DESC]]
    [LIMIT n] [OFFSET n];

  Tables: MODULES, ENTITIES, ATTRIBUTES, MICROFLOWS, NANOFLOWS, PAGES,
          SNIPPETS, LAYOUTS, ENUMERATIONS, JAVA_ACTIONS, ACTIVITIES*,
          WIDGETS*, XPATH_EXPRESSIONS, REFS*, PROJECTS, SNAPSHOTS,
          OBJECTS, ODATA_CLIENTS, ODATA_SERVICES,
          BUSINESS_EVENT_SERVICES, STRINGS*, SOURCE**
  (* only populated with REFRESH CATALOG FULL)
  (** only populated with REFRESH CATALOG FULL SOURCE)

  Cache is stored in .mxcli/catalog.db next to the .mpr file.

Code Search (requires REFRESH CATALOG FULL):
  SHOW CALLERS OF Module.Microflow [TRANSITIVE];
  SHOW CALLEES OF Module.Microflow [TRANSITIVE];
  SHOW REFERENCES TO Module.Element;
  SHOW IMPACT OF Module.Element;
  SHOW CONTEXT OF Module.Element [DEPTH n];  -- Assemble context for LLM

Security - Roles:
  CREATE MODULE ROLE Module.Role [DESCRIPTION 'text'];
  DROP MODULE ROLE Module.Role;
  CREATE USER ROLE Name (Module.Role [, ...]) [MANAGE ALL ROLES];
  ALTER USER ROLE Name ADD MODULE ROLES (Module.Role [, ...]);
  ALTER USER ROLE Name REMOVE MODULE ROLES (Module.Role [, ...]);
  DROP USER ROLE Name;

Security - Access Control:
  GRANT EXECUTE ON MICROFLOW Module.Name TO Role [, Role...];
  REVOKE EXECUTE ON MICROFLOW Module.Name FROM Role [, Role...];
  GRANT VIEW ON PAGE Module.Name TO Role [, Role...];
  REVOKE VIEW ON PAGE Module.Name FROM Role [, Role...];
  GRANT EXECUTE ON WORKFLOW Module.Name TO Role [, Role...];
  REVOKE EXECUTE ON WORKFLOW Module.Name FROM Role [, Role...];
  GRANT Role ON Module.Entity (CREATE, DELETE, READ *, WRITE *) [WHERE 'xpath'];
  REVOKE Role ON Module.Entity;

Security - Project Settings:
  ALTER PROJECT SECURITY LEVEL OFF|PROTOTYPE|PRODUCTION;
  ALTER PROJECT SECURITY DEMO USERS ON|OFF;
  CREATE DEMO USER 'name' PASSWORD 'pass' (UserRole [, ...]);
  DROP DEMO USER 'name';

Security - Queries:
  SHOW PROJECT SECURITY;
  SHOW MODULE ROLES [IN Module];
  SHOW USER ROLES;
  SHOW DEMO USERS;
  SHOW ACCESS ON MICROFLOW Module.Name;
  SHOW ACCESS ON PAGE Module.Name;
  SHOW ACCESS ON WORKFLOW Module.Name;
  SHOW ACCESS ON Module.Entity;
  SHOW SECURITY MATRIX [IN Module];
  DESCRIBE MODULE ROLE Module.Role;
  DESCRIBE USER ROLE Name;
  DESCRIBE DEMO USER 'name';

Navigation:
  SHOW NAVIGATION;
  SHOW NAVIGATION MENU [Profile];
  SHOW NAVIGATION HOMES;
  DESCRIBE NAVIGATION Profile;
  CREATE OR REPLACE NAVIGATION Profile
    HOME PAGE Module.Page
    [HOME PAGE Module.Page FOR Module.Role]
    [LOGIN PAGE Module.Page]
    [NOT FOUND PAGE Module.Page]
    [MENU (
      MENU ITEM 'Caption' PAGE Module.Page;
      MENU 'SubMenu' (
        MENU ITEM 'Child' MICROFLOW Module.Flow;
      );
    )];

Data Types:
  String[(length)]  Integer  Long  Decimal[(p,s)]
  Boolean  DateTime  Date  AutoNumber  Binary
  Enumeration(Module.EnumName)

Scripts:
  EXECUTE SCRIPT 'path/to/script.mdl';

Modules:
  CREATE MODULE Name;
  DROP MODULE Name;                -- Cascade-deletes all contents
  SHOW MODULES;

External SQL:
  SQL CONNECT <driver> '<dsn>' AS <alias>;
  SQL DISCONNECT <alias>;
  SQL CONNECTIONS;
  SQL <alias> SHOW TABLES;
  SQL <alias> SHOW VIEWS;
  SQL <alias> SHOW FUNCTIONS;
  SQL <alias> DESCRIBE <table>;
  SQL <alias> SELECT * FROM users LIMIT 10;
  SQL <alias> GENERATE CONNECTOR INTO <module> [TABLES (...)] [VIEWS (...)] [EXEC];

  Supported drivers: postgres (pg), oracle (ora), sqlserver (mssql)
  DSN examples:
    'postgres://user:pass@localhost:5432/dbname'
    'oracle://user:pass@localhost:1521/service'
    'sqlserver://sa:pass@localhost:1433?database=mydb&encrypt=disable'

Import from External DB into Mendix App DB:
  IMPORT FROM <alias> QUERY '<sql>'
    INTO Module.Entity
    MAP (source_col AS TargetAttr [, ...])
    [LINK (source_col TO AssocName ON ChildAttr [, ...])]
    [BATCH n]
    [LIMIT n];

  LINK maps source columns to associations (Reference type only):
    ON ChildAttr  — lookup child entity by attribute value
    (no ON)       — source value is a raw Mendix object ID

  Reads rows from an external database and inserts them into the
  Mendix app's PostgreSQL database. Auto-connects using project settings.
  Handles both Column and Table association storage automatically.
  Default batch size: 1000.

  Env var overrides (for devcontainers/Docker):
    MXCLI_DB_TYPE, MXCLI_DB_HOST, MXCLI_DB_PORT,
    MXCLI_DB_NAME, MXCLI_DB_USER, MXCLI_DB_PASSWORD

Other:
  COMMIT [MESSAGE 'message'];
  SET key = value;
  HELP or ?
  EXIT or QUIT

Statement Terminator:
  Use ; or / to end statements
`
	fmt.Fprint(e.output, help)
	return nil
}

// showVersion displays Mendix project version information.
func (e *Executor) showVersion() error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}

	pv := e.reader.ProjectVersion()
	fmt.Fprintf(e.output, "Mendix Version: %s\n", pv.ProductVersion)
	fmt.Fprintf(e.output, "Build Version:  %s\n", pv.BuildVersion)
	fmt.Fprintf(e.output, "MPR Format:     v%d\n", pv.FormatVersion)
	if pv.SchemaHash != "" {
		fmt.Fprintf(e.output, "Schema Hash:    %s\n", pv.SchemaHash)
	}
	return nil
}

// execExit handles EXIT statements.
// Note: This just signals exit intent via ErrExit. The actual cleanup
// is done by the caller (CLI/REPL) when they handle ErrExit at the top level.
// This allows exit within nested scripts to stop just that script without
// closing the database connection.
func (e *Executor) execExit() error {
	return ErrExit
}

// execExecuteScript handles EXECUTE SCRIPT statements.
func (e *Executor) execExecuteScript(s *ast.ExecuteScriptStmt) error {
	// Resolve path relative to current working directory
	scriptPath := s.Path
	if !filepath.IsAbs(scriptPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		scriptPath = filepath.Join(cwd, scriptPath)
	}

	// Read the script file
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script file '%s': %w", s.Path, err)
	}

	// Pre-process: remove "/" statement separators (SQL*Plus style)
	// The "/" allows scripts to have multi-statement blocks when statements contain ";" internally
	processedContent := stripSlashSeparators(string(content))

	// Parse the script
	prog, errs := visitor.Build(processedContent)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(e.output, "Parse error in %s: %v\n", s.Path, err)
		}
		return fmt.Errorf("script '%s' has parse errors", s.Path)
	}

	// Execute all statements in the script
	fmt.Fprintf(e.output, "Executing script: %s\n", s.Path)
	for _, stmt := range prog.Statements {
		if err := e.Execute(stmt); err != nil {
			// Exit within a script just stops the script, doesn't exit mxcli
			if errors.Is(err, ErrExit) {
				fmt.Fprintf(e.output, "Script exited: %s\n", s.Path)
				return nil
			}
			return fmt.Errorf("error in script '%s': %w", s.Path, err)
		}
	}
	fmt.Fprintf(e.output, "Script completed: %s\n", s.Path)

	return nil
}

// stripSlashSeparators removes lines that contain only "/" from the script content.
// This allows "/" to be used as a statement separator (SQL*Plus style) without
// requiring the grammar to support it.
func stripSlashSeparators(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		// Skip lines that are just "/" (with optional whitespace)
		if strings.TrimSpace(line) == "/" {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}
