// SPDX-License-Identifier: Apache-2.0

package main

import (
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/syntax"
	"github.com/spf13/cobra"
)

//go:embed help_topics/*.txt
var helpTopics embed.FS

// showTopicHelp prints an embedded help topic file.
func showTopicHelp(name string) {
	data, err := helpTopics.ReadFile("help_topics/" + name + ".txt")
	if err != nil {
		fmt.Printf("Unknown topic: %s\n", name)
		return
	}
	os.Stdout.Write(data)
}

// Reserved keywords that cannot be used as identifiers without quoting
var reservedKeywords = []string{
	// DDL Keywords
	"CREATE", "ALTER", "DROP", "RENAME", "MOVE", "MODIFY",
	"ENTITY", "PERSISTENT", "NON_PERSISTENT", "VIEW", "EXTERNAL",
	"ASSOCIATION", "ENUMERATION", "MODULE", "MICROFLOW", "NANOFLOW",
	"PAGE", "SNIPPET", "LAYOUT", "NOTEBOOK", "CONSTANT",
	"ATTRIBUTE", "COLUMN", "INDEX", "OWNER", "REFERENCE", "GENERALIZATION", "EXTENDS",
	"ADD", "SET", "POSITION", "DOCUMENTATION",

	// Delete Behavior
	"DELETE_BEHAVIOR", "CASCADE", "PREVENT",
	"DELETE_AND_REFERENCES", "DELETE_BUT_KEEP_REFERENCES", "DELETE_IF_NO_REFERENCES",

	// Connection Keywords
	"CONNECT", "DISCONNECT", "LOCAL", "PROJECT", "RUNTIME", "BRANCH", "TOKEN",
	"HOST", "PORT", "SHOW", "DESCRIBE", "USE", "INTROSPECT", "DEBUG",

	// Query Keywords
	"SELECT", "FROM", "WHERE", "HAVING", "OFFSET", "LIMIT", "AS", "RETURNS", "RETURNING",
	"CASE", "WHEN", "THEN", "ELSE", "END", "DISTINCT", "ALL",
	"JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "FULL", "CROSS", "ON",
	"GROUP_BY", "ORDER_BY", "SORT_BY", "ASC", "DESC",

	// Microflow Keywords
	"BEGIN", "DECLARE", "CHANGE", "RETRIEVE", "DELETE", "COMMIT", "ROLLBACK",
	"LOOP", "WHILE", "IF", "ELSIF", "ELSEIF", "CONTINUE", "BREAK", "RETURN", "THROW",
	"LOG", "CALL", "JAVA", "ACTION", "ACTIONS", "CLOSE",
	"NODE", "EVENTS",

	// List Operations
	"HEAD", "TAIL", "FIND", "SORT", "UNION", "INTERSECT", "SUBTRACT", "CONTAINS",
	"AVERAGE", "MINIMUM", "MAXIMUM", "LIST", "REMOVE", "EQUALS",

	// Log Levels
	"INFO", "WARNING", "TRACE", "CRITICAL",

	// Page/Widget Keywords (commonly confused with identifiers)
	"TITLE", "LABEL", "CAPTION", "ICON", "TOOLTIP",
	"DATASOURCE", "SOURCE", "SELECTION", "FOOTER", "HEADER", "CONTENT",
	"CLASS", "STYLE", "WIDTH", "HEIGHT", "EDITABLE", "READONLY", "VISIBLE",
	"BUTTON", "CONTAINER", "ROW", "ITEM",
	"PRIMARY", "SUCCESS", "DANGER", "DEFAULT",
	"TEMPLATE", "ONCLICK", "ONCHANGE",

	// Widget Type Keywords
	"ACTIONBUTTON", "CHECKBOX", "COMBOBOX", "CONTROLBAR", "CUSTOMWIDGET",
	"DATAGRID", "DATAVIEW", "DATEPICKER", "DROPDOWN", "DYNAMICTEXT",
	"GALLERY", "LAYOUTGRID", "LINKBUTTON", "LISTVIEW", "NAVIGATIONLIST",
	"PLACEHOLDER", "RADIOBUTTONS", "SEARCHBAR", "SNIPPETCALL",
	"STATICTEXT", "TEXTAREA", "TEXTBOX",

	// Security Keywords
	"GRANT", "REVOKE", "ROLE", "ROLES", "SECURITY", "LEVEL", "PROTOTYPE", "PRODUCTION",
	"DEMO", "USER", "MANAGE", "MATRIX", "ACCESS", "DESCRIPTION",

	// Settings / Business Events Keywords
	"SETTINGS", "CONFIGURATION", "BUSINESS", "PUBLISH", "SUBSCRIBE",

	// Fragment Keywords
	"DEFINE", "FRAGMENT", "FRAGMENTS",

	// Other MDL Keywords
	"AUTOFILL", "CHECK", "EXECUTE", "EXPOSED", "FILTER", "LINT",
	"PASSING", "REFRESH", "RENDERMODE", "SAVE_CHANGES", "CANCEL_CHANGES",
	"CLOSE_PAGE", "DELETE_ACTION", "SHOW_PAGE", "CREATE_OBJECT", "CALL_MICROFLOW",
	"TABINDEX", "TEXT", "VARIABLE", "WIDGET", "WIDGETS", "WITHOUT",

	// Data Type Keywords
	"STRING", "INTEGER", "LONG", "DECIMAL", "BOOLEAN", "DATETIME", "DATE",
	"AUTONUMBER", "BINARY", "HASHEDSTRING", "CURRENCY", "FLOAT", "ENUM",

	// Aggregate Functions
	"COUNT", "SUM", "AVG", "MIN", "MAX", "LENGTH", "TRIM", "COALESCE", "CAST",

	// Logical/Comparison
	"AND", "OR", "NOT", "NULL", "IN", "BETWEEN", "LIKE", "EXISTS",
	"UNIQUE", "TRUE", "FALSE",

	// Validation/Constraint
	"VALIDATION", "FEEDBACK", "RULE", "REQUIRED", "ERROR", "RANGE", "REGEX",
	"PATTERN", "EXPRESSION", "XPATH", "CONSTRAINT",

	// REST Client
	"REST", "SERVICE", "SERVICES", "BASE", "AUTHENTICATION", "BASIC", "OAUTH",
	"OPERATION", "METHOD", "PATH", "TIMEOUT", "BODY", "RESPONSE", "REQUEST",
	"JSON", "XML", "STATUS", "VERSION", "GET", "POST", "PUT", "PATCH",
	"API", "CLIENT", "CLIENTS", "USERNAME", "PASSWORD", "CONNECTION", "DATABASE",
	"QUERY", "MAP", "PARAMETER", "PARAMETERS",

	// Utility
	"TYPE", "VALUE", "SINGLE", "MULTIPLE", "NONE", "BOTH", "TO", "OF", "OVER", "FOR",
	"REPLACE", "MEMBERS", "FORMAT", "SQL", "WITH", "EMPTY", "OBJECT", "OBJECTS",
	"MESSAGE", "COMMENT", "CATALOG", "FORCE", "BACKGROUND", "FOLDER",
	"CALLERS", "CALLEES", "REFERENCES", "TRANSITIVE", "IMPACT", "DEPTH",
	"SEARCH", "MATCH", "STRUCTURE", "CONTEXT",
}

// Attribute types with descriptions
var attributeTypes = map[string]string{
	"String(n)":         "Variable-length text up to n characters (e.g., String(200))",
	"Integer":           "Whole number (-2,147,483,648 to 2,147,483,647)",
	"Long":              "Large whole number (-9,223,372,036,854,775,808 to ...)",
	"Decimal":           "Precise decimal number for currency/calculations",
	"Boolean":           "True or false value",
	"DateTime":          "Date and time combined (DateAndTime also accepted)",
	"Date":              "Date only (no time component)",
	"AutoNumber":        "Auto-incrementing integer",
	"AutoOwner":         "System.owner association (auto-set on create)",
	"AutoChangedBy":     "System.changedBy association (auto-set on commit)",
	"AutoCreatedDate":   "CreatedDate: DateTime (auto-set on create)",
	"AutoChangedDate":   "ChangedDate: DateTime (auto-set on commit)",
	"Binary":            "Binary data (files, images)",
	"HashedString":      "Securely hashed string (for passwords)",
	"Enumeration(Name)": "Reference to an enumeration (e.g., Enumeration(MyModule.Status))",
}

// Delete behaviors with descriptions
var deleteBehaviors = map[string]string{
	"CASCADE":                    "Delete associated objects when parent is deleted",
	"PREVENT":                    "Prevent deletion if associated objects exist",
	"DELETE_BUT_KEEP_REFERENCES": "Delete parent but keep child objects (set reference to null)",
	"DELETE_AND_REFERENCES":      "Delete both parent and referenced objects",
	"DELETE_IF_NO_REFERENCES":    "Delete only if no objects reference this",
}

var syntaxCmd = &cobra.Command{
	Use:   "syntax [topic]",
	Short: "Show MDL syntax reference",
	Long: `Show detailed help on MDL syntax topics.

Available topics:
  keywords    - List reserved keywords that cannot be used as identifiers
  types       - List valid attribute data types
  delete      - List valid DELETE_BEHAVIOR options
  entity      - Show entity creation syntax
  enumeration - Show enumeration creation syntax
  constant    - Show constant creation syntax (CREATE/SHOW/DESCRIBE/DROP)
  association - Show association creation syntax
  microflow   - Show microflow creation syntax
  page        - Show page creation syntax
  snippet     - Show snippet creation syntax
  move        - Show MOVE command syntax for relocating documents
  security    - Show security management syntax (roles, access, GRANT/REVOKE)
  odata       - Show OData client/service/external entity syntax
  rest        - Show consumed and published REST service syntax
  integration - Show all integration services, contract browsing, catalog tables
  workflow    - Show workflow commands syntax
  navigation  - Show navigation profile management syntax
  structure   - Show SHOW STRUCTURE command syntax
  search      - Show full-text search syntax
  settings    - Show project settings syntax (ALTER SETTINGS)
  fragment    - Show fragment (reusable widget group) syntax
  java-action - Show Java action syntax (CREATE/DESCRIBE/CALL, type params, EXPOSED AS)
  business-events - Show business event service syntax
  xpath       - Show XPath constraint syntax for WHERE clauses
  oql         - Show OQL query execution syntax (mxcli oql)
  sql         - Show external SQL query execution syntax (mxcli sql)
  errors      - List validation errors and how to fix them

Example:
  mxcli syntax keywords
  mxcli syntax microflow
  mxcli syntax security
`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag, _ := cmd.Flags().GetBool("json")

		// No args: show full index (JSON) or help (text)
		if len(args) == 0 {
			if jsonFlag {
				syntax.WriteJSON(os.Stdout, syntax.All())
				return
			}
			cmd.Help()
			return
		}

		// Build registry path from args: "workflow user-task" → "workflow.user-task"
		path := strings.ToLower(strings.Join(args, "."))

		// Map legacy topic aliases to registry paths
		topicAliases := map[string]string{
			"keywords":        "domain-model.keywords",
			"reserved":        "domain-model.keywords",
			"types":           "domain-model.types",
			"datatypes":       "domain-model.types",
			"data-types":      "domain-model.types",
			"delete":          "domain-model.association.delete-behavior",
			"delete_behavior": "domain-model.association.delete-behavior",
			"delete-behavior": "domain-model.association.delete-behavior",
			"entity":          "domain-model.entity",
			"entities":        "domain-model.entity",
			"enumeration":     "domain-model.enumeration",
			"enum":            "domain-model.enumeration",
			"enumerations":    "domain-model.enumeration",
			"constant":        "domain-model.constant",
			"constants":       "domain-model.constant",
			"association":     "domain-model.association",
			"associations":    "domain-model.association",
		}
		if alias, ok := topicAliases[path]; ok {
			path = alias
		}

		// Check registry first
		if syntax.HasPrefix(path) {
			features := syntax.ByPrefix(path)
			if jsonFlag {
				syntax.WriteJSON(os.Stdout, features)
			} else {
				syntax.WriteText(os.Stdout, features)
			}
			return
		}

		// Fall back to legacy topics (first arg only)
		topic := strings.ToLower(args[0])
		switch topic {
		case "microflow", "microflows":
			showTopicHelp("microflow")
		case "page", "pages":
			showTopicHelp("page")
		case "snippet", "snippets":
			showTopicHelp("snippet")
		case "fragment", "fragments":
			showTopicHelp("fragment")
		default:
			fmt.Printf("Unknown topic: %s\n\n", topic)
			cmd.Help()
		}
	},
}

func showKeywords() {
	fmt.Println("Reserved Keywords")
	fmt.Println("=================")
	fmt.Println()
	fmt.Println("Most MDL keywords can be used as module/entity names without issue.")
	fmt.Println("However, some words may cause parse errors when used as identifiers.")
	fmt.Println()
	fmt.Println("Use quoted identifiers to escape any reserved word:")
	fmt.Println()
	fmt.Println("  DESCRIBE ENTITY \"ComboBox\".\"CategoryTreeVE\";")
	fmt.Println("  SHOW ENTITIES IN \"ComboBox\";")
	fmt.Println("  SHOW MICROFLOWS IN `Order`;")
	fmt.Println()
	fmt.Println("Both double-quote (ANSI SQL) and backtick (MySQL) styles are supported.")
	fmt.Println("You can mix quoted and unquoted parts: \"ComboBox\".CategoryTreeVE")
	fmt.Println()

	// Sort keywords alphabetically
	sorted := make([]string, len(reservedKeywords))
	copy(sorted, reservedKeywords)
	sort.Strings(sorted)

	// Print in columns
	cols := 5
	colWidth := 25
	for i, kw := range sorted {
		fmt.Printf("%-*s", colWidth, kw)
		if (i+1)%cols == 0 {
			fmt.Println()
		}
	}
	if len(sorted)%cols != 0 {
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("Common keywords that conflict with Mendix module/entity names:")
	fmt.Println("  - ComboBox, DataGrid, Gallery (widget-named modules)")
	fmt.Println("  - Title, Status, Type, Value (common attribute names)")
	fmt.Println()
	fmt.Println("Workarounds:")
	fmt.Println("  1. Use quoted identifiers:  DESCRIBE ENTITY \"ComboBox\".ProductVE")
	fmt.Println("  2. Rename to avoid conflict: Title -> BookTitle, Status -> OrderStatus")
	fmt.Println()
	fmt.Printf("Total: %d reserved keywords\n", len(sorted))
}

func showTypes() {
	fmt.Println("Attribute Data Types")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("Valid data types for entity attributes:")
	fmt.Println()

	// Sort types for consistent output
	var types []string
	for t := range attributeTypes {
		types = append(types, t)
	}
	sort.Strings(types)

	for _, t := range types {
		fmt.Printf("  %-20s  %s\n", t, attributeTypes[t])
	}

	fmt.Println()
	fmt.Println("Common mistakes:")
	fmt.Println("  - DateAndTime (use DateTime instead)")
	fmt.Println("  - Text (use String(n) instead)")
	fmt.Println("  - Number (use Integer, Long, or Decimal instead)")
	fmt.Println("  - Enum (use Enumeration(Module.Name) instead)")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  CREATE PERSISTENT ENTITY MyModule.Customer (")
	fmt.Println("    Name: String(100) NOT NULL,")
	fmt.Println("    Age: Integer,")
	fmt.Println("    Balance: Decimal,")
	fmt.Println("    IsActive: Boolean DEFAULT true,")
	fmt.Println("    CreatedAt: DateTime,")
	fmt.Println("    OrderStatus: Enumeration(MyModule.Status)")
	fmt.Println("  );")
}

func showDeleteBehaviors() {
	fmt.Println("Association Delete Behaviors")
	fmt.Println("============================")
	fmt.Println()
	fmt.Println("Valid DELETE_BEHAVIOR options for associations:")
	fmt.Println()

	// Sort for consistent output
	var behaviors []string
	for b := range deleteBehaviors {
		behaviors = append(behaviors, b)
	}
	sort.Strings(behaviors)

	for _, b := range behaviors {
		fmt.Printf("  %-28s  %s\n", b, deleteBehaviors[b])
	}

	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  CREATE ASSOCIATION MyModule.Order_Customer")
	fmt.Println("  FROM MyModule.Order TO MyModule.Customer")
	fmt.Println("  TYPE Reference")
	fmt.Println("  OWNER Default")
	fmt.Println("  DELETE_BEHAVIOR PREVENT;")
}

func init() {
	rootCmd.AddCommand(syntaxCmd)
}
