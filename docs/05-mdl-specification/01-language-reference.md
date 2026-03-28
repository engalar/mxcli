# MDL Language Reference

This document provides a complete reference for MDL (Mendix Definition Language) syntax and semantics.

## Table of Contents

1. [Lexical Structure](#lexical-structure)
2. [Identifiers and Names](#identifiers-and-names)
3. [Connection Statements](#connection-statements)
4. [Query Statements](#query-statements)
5. [Module Statements](#module-statements)
6. [Enumeration Statements](#enumeration-statements)
7. [Entity Statements](#entity-statements)
8. [ALTER ENTITY Statements](#alter-entity-statements)
9. [Association Statements](#association-statements)
10. [Microflow Statements](#microflow-statements)
11. [Page Statements](#page-statements)
12. [ALTER PAGE / ALTER SNIPPET](#alter-page--alter-snippet)
13. [MOVE Statements](#move-statements)
14. [Security Statements](#security-statements)
15. [Navigation Statements](#navigation-statements)
16. [Settings Statements](#settings-statements)
17. [External SQL Statements](#external-sql-statements)
18. [Import Statements](#import-statements)
19. [Catalog and Search Statements](#catalog-and-search-statements)
20. [Business Event Statements](#business-event-statements)
21. [Java Action Statements](#java-action-statements)
22. [Session Statements](#session-statements)

---

## Lexical Structure

### Keywords

MDL keywords are **case-insensitive**. The following are reserved keywords:

```
ACCESS, ACTIONS, ADD, AFTER, ALL, ALTER, AND, ANNOTATION, AS, ASC,
ASCENDING, ASSOCIATION, AUTONUMBER, BATCH, BEFORE, BEGIN, BINARY,
BOOLEAN, BOTH, BUSINESS, BY, CALL, CANCEL, CAPTION, CASCADE,
CATALOG, CHANGE, CHILD, CLOSE, COLUMN, COMBOBOX, COMMIT, CONNECT,
CONFIGURATION, CONNECTOR, CONSTANT, CONSTRAINT, CONTAINER, CREATE,
CRUD, DATAGRID, DATAVIEW, DATE, DATETIME, DECLARE, DEFAULT, DELETE,
DELETE_BEHAVIOR, DELETE_BUT_KEEP_REFERENCES, DELETE_CASCADE, DEMO,
DEPTH, DESC, DESCENDING, DESCRIBE, DIFF, DISCONNECT, DROP, ELSE,
EMPTY, END, ENTITY, ENUMERATION, ERROR, EVENT, EVENTS, EXECUTE,
EXEC, EXIT, EXPORT, EXTENDS, EXTERNAL, FALSE, FOLDER, FOOTER,
FOR, FORMAT, FROM, FULL, GALLERY, GENERATE, GRANT, HEADER, HELP,
HOME, IF, IMPORT, IN, INDEX, INFO, INSERT, INTEGER, INTO, JAVA,
KEEP_REFERENCES, LABEL, LANGUAGE, LAYOUT, LAYOUTGRID, LEVEL, LIMIT,
LINK, LIST, LISTVIEW, LOCAL, LOG, LOGIN, LONG, LOOP, MANAGE, MAP,
MATRIX, MENU, MESSAGE, MICROFLOW, MICROFLOWS, MODEL, MODIFY, MODULE,
MODULES, MOVE, NANOFLOW, NANOFLOWS, NAVIGATION, NODE, NON_PERSISTENT,
NOT, NULL, OF, ON, OR, ORACLE, OVERVIEW, OWNER, PAGE, PAGES, PARENT,
PASSWORD, PERSISTENT, POSITION, POSTGRES, PRODUCTION, PROJECT,
PROTOTYPE, QUERY, QUIT, REFERENCE, REFERENCESET, REFRESH, REMOVE,
REPLACE, REPORT, RESPONSIVE, RETRIEVE, RETURN, REVOKE, ROLE, ROLES,
ROLLBACK, ROW, SAVE, SCRIPT, SEARCH, SECURITY, SELECTION, SET, SHOW,
SNIPPET, SNIPPETS, SQL, SQLSERVER, STATUS, STRING, STRUCTURE,
TABLES, TEXTBOX, TEXTAREA, THEN, TO, TRUE, TYPE, UNIQUE, UPDATE,
USER, VALIDATION, VALUE, VIEW, VIEWS, VISIBLE, WARNING, WHERE, WIDGET,
WIDGETS, WITH, WORKFLOWS, WRITE
```

Most MDL keywords work **unquoted** as identifiers (entity names, attribute names, etc.).
Only structural keywords like `CREATE`, `DELETE`, `BEGIN`, `END`, `RETURN`, `ENTITY`, `MODULE`
require quoting when used as identifiers. See [MDL Quick Reference](../MDL_QUICK_REFERENCE.md#reserved-words-and-quoted-identifiers) for details.

### Literals

#### String Literals
```sql
'single quoted string'
```

Escape sequences:
- `''` - Single quote (doubled)

Note: Mendix expression strings use doubled single quotes for escaping (`'it''s here'`), not backslash escaping.

#### Quoted Identifiers

Use double-quotes (ANSI SQL) or backticks (MySQL) to escape reserved words in identifiers:
```sql
"ComboBox"."CategoryTreeVE"
`Order`.`Status`
"ComboBox".CategoryTreeVE    -- mixed is fine
```

#### Numeric Literals
```sql
42          -- Integer
3.14        -- Decimal
-100        -- Negative integer
1.5e10      -- Scientific notation
```

#### Boolean Literals
```sql
TRUE
FALSE
```

### Comments

```sql
-- Single line comment
// Also single line comment

/* Multi-line
   comment */

/** Documentation comment
 *  Used for entity/attribute documentation
 */
```

### Statement Terminators

Statements can be terminated with:
- `;` - Standard SQL terminator
- `/` - Oracle-style terminator (useful for multi-line statements)

Simple commands (HELP, EXIT, STATUS, SHOW, DESCRIBE) don't require terminators.

---

## Identifiers and Names

### Simple Identifiers

Valid identifier characters:
- Letters: `A-Z`, `a-z`
- Digits: `0-9` (not as first character)
- Underscore: `_`

```sql
MyEntity
my_attribute
Attribute123
```

### Qualified Names

Format: `Module.Element` or `Module.Entity.Attribute`

```sql
MyModule.Customer           -- Entity in module
MyModule.OrderStatus        -- Enumeration in module
MyModule.Customer.Name      -- Attribute in entity
```

---

## Connection Statements

### CONNECT

Establishes a connection to a Mendix project file.

**Syntax:**
```sql
CONNECT LOCAL '<path>'
```

**Parameters:**
- `<path>` - Path to `.mpr` file (absolute or relative)

**Examples:**
```sql
CONNECT LOCAL '/Users/dev/projects/MyApp/MyApp.mpr';
CONNECT LOCAL './mx-test-projects/test1-go-app/test1-go.mpr';
```

### DISCONNECT

Closes the current project connection.

**Syntax:**
```sql
DISCONNECT
```

### STATUS

Shows current connection status and project information.

**Syntax:**
```sql
STATUS
```

**Output:**
```
Status: Connected
Project: /path/to/project.mpr
Modules: 5
```

---

## Query Statements

### SHOW

Lists elements of a specific type.

**Syntax:**
```sql
SHOW MODULES
SHOW ENTITIES [IN <module>]
SHOW ENTITY <qualified-name>
SHOW ENUMERATIONS [IN <module>]
SHOW ASSOCIATIONS [IN <module>]
SHOW ASSOCIATION <qualified-name>
SHOW MICROFLOWS [IN <module>]
SHOW NANOFLOWS [IN <module>]
SHOW PAGES [IN <module>]
SHOW SNIPPETS [IN <module>]
SHOW JAVA ACTIONS [IN <module>]
SHOW WIDGETS
SHOW STRUCTURE [DEPTH 1|2|3] [IN <module>] [ALL]
SHOW BUSINESS EVENTS [IN <module>]
```

**Examples:**
```sql
SHOW MODULES
SHOW ENTITIES IN MyFirstModule
SHOW ENTITY MyModule.Customer
SHOW MICROFLOWS IN Administration
SHOW STRUCTURE DEPTH 1          -- Module counts
SHOW STRUCTURE IN MyModule      -- Single module at depth 2
SHOW STRUCTURE DEPTH 3 ALL      -- Full detail, all modules
```

### DESCRIBE

Shows detailed definition of an element in MDL syntax.

**Syntax:**
```sql
DESCRIBE ENTITY <qualified-name>
DESCRIBE ENUMERATION <qualified-name>
DESCRIBE ASSOCIATION <qualified-name>
DESCRIBE MICROFLOW <qualified-name>
DESCRIBE NANOFLOW <qualified-name>
DESCRIBE PAGE <qualified-name>
DESCRIBE SNIPPET <qualified-name>
DESCRIBE JAVA ACTION <qualified-name>
DESCRIBE BUSINESS EVENT SERVICE <qualified-name>
DESCRIBE NAVIGATION [<profile>]
DESCRIBE SETTINGS
```

**Examples:**
```sql
DESCRIBE ENTITY MyModule.Customer
DESCRIBE ENUMERATION MyModule.OrderStatus
DESCRIBE MICROFLOW MyModule.CreateOrder
DESCRIBE PAGE MyModule.Customer_Edit
DESCRIBE NAVIGATION Responsive
```

**Output Example:**
```sql
/**
 * Customer entity stores customer information.
 */
@Position(100, 200)
CREATE PERSISTENT ENTITY MyModule.Customer (
  /** Customer identifier */
  CustomerId: AutoNumber NOT NULL UNIQUE DEFAULT 1,
  Name: String(200) NOT NULL,
  Email: String(200),
  Status: Enumeration(MyModule.CustomerStatus) DEFAULT 'Active'
)
INDEX (Name);
/
```

---

## Module Statements

### CREATE MODULE

Creates a new module in the project.

**Syntax:**
```sql
CREATE MODULE <name>
```

**Example:**
```sql
CREATE MODULE OrderManagement;
```

### DROP MODULE

Removes a module from the project.

**Syntax:**
```sql
DROP MODULE <name>
```

**Example:**
```sql
DROP MODULE OrderManagement;
```

---

## Enumeration Statements

### CREATE ENUMERATION

Creates a new enumeration type.

**Syntax:**
```sql
[/** <documentation> */]
CREATE ENUMERATION <qualified-name> (
  <value-name> '<caption>' [, ...]
)
```

**Parameters:**
- `<qualified-name>` - Module.EnumerationName
- `<value-name>` - Identifier for the enum value
- `<caption>` - Display caption for the value

**Example:**
```sql
/** Order status enumeration */
CREATE ENUMERATION OrderModule.OrderStatus (
  Draft 'Draft',
  Pending 'Pending Approval',
  Approved 'Approved',
  Shipped 'Shipped',
  Delivered 'Delivered',
  Cancelled 'Cancelled'
);
```

### ALTER ENUMERATION

Modifies an existing enumeration.

**Syntax:**
```sql
ALTER ENUMERATION <qualified-name>
  ADD VALUE <value-name> '<caption>'

ALTER ENUMERATION <qualified-name>
  REMOVE VALUE <value-name>
```

**Examples:**
```sql
ALTER ENUMERATION OrderModule.OrderStatus
  ADD VALUE OnHold 'On Hold';

ALTER ENUMERATION OrderModule.OrderStatus
  REMOVE VALUE Draft;
```

### DROP ENUMERATION

Removes an enumeration.

**Syntax:**
```sql
DROP ENUMERATION <qualified-name>
```

---

## Entity Statements

### CREATE ENTITY

Creates a new entity in the domain model.

**Syntax:**
```sql
[/** <documentation> */]
[@Position(<x>, <y>)]
CREATE [OR MODIFY] <entity-type> ENTITY <qualified-name> (
  [<attribute-definition> [, ...]]
)
[INDEX (<column-list>)]
[INDEX (<column-list>)]
```

**Entity Types:**
- `PERSISTENT` - Stored in database
- `NON-PERSISTENT` - In-memory only
- `VIEW` - Based on OQL query (see CREATE VIEW ENTITY)
- `EXTERNAL` - From external data source

**Attribute Definition:**
```sql
[/** <documentation> */]
<name>: <type> [NOT NULL [ERROR '<message>']] [UNIQUE [ERROR '<message>']] [DEFAULT <value>] [CALCULATED]
```

**Examples:**

```sql
/** Customer entity */
@Position(100, 200)
CREATE PERSISTENT ENTITY Sales.Customer (
  /** Unique customer ID */
  CustomerId: AutoNumber NOT NULL UNIQUE,

  /** Customer full name */
  Name: String(200) NOT NULL ERROR 'Name is required',

  Email: String(200) UNIQUE ERROR 'Email must be unique',

  Balance: Decimal DEFAULT 0,

  IsActive: Boolean DEFAULT TRUE,

  CreatedDate: DateTime,

  Status: Enumeration(Sales.CustomerStatus) DEFAULT 'Active'
)
INDEX (Name)
INDEX (Email);
/
```

```sql
CREATE NON-PERSISTENT ENTITY Sales.CustomerFilter (
  SearchName: String(200),
  MinBalance: Decimal,
  MaxBalance: Decimal
);
```

### CREATE OR MODIFY ENTITY

Creates an entity if it doesn't exist, or modifies it if it does.

```sql
CREATE OR MODIFY PERSISTENT ENTITY Sales.Customer (
  CustomerId: AutoNumber NOT NULL UNIQUE,
  Name: String(200) NOT NULL,
  Email: String(200),
  Phone: String(50)  -- New attribute added
);
```

### CREATE VIEW ENTITY

Creates a view entity based on an OQL query.

**Syntax:**
```sql
CREATE VIEW ENTITY <qualified-name> (
  <attribute-definition> [, ...]
) AS
  <oql-query>
```

**Example:**
```sql
CREATE VIEW ENTITY Reports.CustomerSummary (
  CustomerName: String,
  TotalOrders: Integer,
  TotalAmount: Decimal
) AS
  SELECT
    c.Name AS CustomerName,
    COUNT(o.OrderId) AS TotalOrders,
    SUM(o.Amount) AS TotalAmount
  FROM Sales.Customer c
  LEFT JOIN Sales.Order o ON o.Customer = c
  GROUP BY c.Name;
/
```

### DROP ENTITY

Removes an entity from the domain model.

**Syntax:**
```sql
DROP ENTITY <qualified-name>
```

---

## ALTER ENTITY Statements

### ALTER ENTITY

Modifies an existing entity's attributes, indexes, or documentation without recreating it.

**Syntax:**
```sql
ALTER ENTITY <qualified-name>
  ADD (<attribute-definition> [, ...])

ALTER ENTITY <qualified-name>
  DROP (<attribute-name> [, ...])

ALTER ENTITY <qualified-name>
  MODIFY (<attribute-definition> [, ...])

ALTER ENTITY <qualified-name>
  RENAME <old-name> TO <new-name>

ALTER ENTITY <qualified-name>
  ADD INDEX (<column-list>)

ALTER ENTITY <qualified-name>
  DROP INDEX (<column-list>)

ALTER ENTITY <qualified-name>
  SET DOCUMENTATION '<text>'
```

**Examples:**
```sql
-- Add new attributes
ALTER ENTITY Sales.Customer
  ADD (Phone: String(50), Notes: String(unlimited));

-- Drop attributes
ALTER ENTITY Sales.Customer
  DROP (Notes);

-- Rename attribute
ALTER ENTITY Sales.Customer
  RENAME Phone TO PhoneNumber;

-- Add index
ALTER ENTITY Sales.Customer
  ADD INDEX (Email);

-- Set documentation
ALTER ENTITY Sales.Customer
  SET DOCUMENTATION 'Customer master data';
```

---

## Association Statements

### CREATE ASSOCIATION

Creates an association between two entities.

**Syntax:**
```sql
[/** <documentation> */]
CREATE ASSOCIATION <qualified-name>
  FROM <parent-entity>
  TO <child-entity>
  TYPE <association-type>
  [OWNER <owner>]
  [DELETE_BEHAVIOR <behavior>]
```

**Association Types:**
- `Reference` - One-to-many (child has reference to parent)
- `ReferenceSet` - Many-to-many

**Owner Options:**
- `Default` - Child owns the association
- `Both` - Both ends can modify
- `Parent` - Parent owns the association
- `Child` - Child owns the association

**Delete Behavior:**
- `DELETE_BUT_KEEP_REFERENCES` - Delete object but keep references
- `DELETE_CASCADE` - Delete associated objects

**Example:**
```sql
/** Links orders to customers */
CREATE ASSOCIATION Sales.Order_Customer
  FROM Sales.Customer
  TO Sales.Order
  TYPE Reference
  OWNER Default
  DELETE_BEHAVIOR DELETE_BUT_KEEP_REFERENCES;
/
```

```sql
CREATE ASSOCIATION Sales.Order_Product
  FROM Sales.Order
  TO Sales.Product
  TYPE ReferenceSet
  OWNER Both;
/
```

### DROP ASSOCIATION

Removes an association.

**Syntax:**
```sql
DROP ASSOCIATION <qualified-name>
```

---

## Microflow Statements

### CREATE MICROFLOW

Creates a microflow with activities, parameters, return type, and control flow.

**Syntax:**
```sql
CREATE [OR REPLACE] MICROFLOW <qualified-name>
  [FOLDER '<path>']
BEGIN
  [<statements>]
END
```

**Statement Types:**
```sql
-- Variable declaration
DECLARE $Var Type = value;
DECLARE $Entity Module.Entity;
DECLARE $List List of Module.Entity = empty;

-- Object operations
$Var = CREATE Module.Entity (Attr = value);
CHANGE $Entity (Attr = value);
COMMIT $Entity [WITH EVENTS] [REFRESH];
DELETE $Entity;
ROLLBACK $Entity [REFRESH];

-- Retrieval
RETRIEVE $Var FROM Module.Entity [WHERE condition] [LIMIT n];

-- Calls
$Result = CALL MICROFLOW Module.Name (Param = $value);
$Result = CALL NANOFLOW Module.Name (Param = $value);
$Result = CALL JAVA ACTION Module.Name (Param = value);

-- UI actions
SHOW PAGE Module.PageName ($Param = $value);
CLOSE PAGE;

-- Validation and logging
VALIDATION FEEDBACK $Entity/Attribute MESSAGE 'message';
LOG INFO|WARNING|ERROR [NODE 'name'] 'message';

-- Control flow
IF condition THEN ... [ELSE ...] END IF;
LOOP $Item IN $List BEGIN ... END LOOP;
RETURN $value;

-- Error handling (suffix on any activity)
... ON ERROR CONTINUE|ROLLBACK|{ handler };

-- Annotations (before any activity)
@position(x, y)
@caption 'text'
@color Green
@annotation 'text'
```

**Example:**
```sql
CREATE MICROFLOW Sales.ACT_CreateOrder
FOLDER 'Orders'
BEGIN
  DECLARE $Order Sales.Order;
  $Order = CREATE Sales.Order (
    OrderDate = [%CurrentDateTime%],
    Status = 'Draft'
  );
  COMMIT $Order;
  SHOW PAGE Sales.Order_Edit ($Order = $Order);
  RETURN $Order;
END;
```

### DESCRIBE MICROFLOW

Shows the full MDL definition of an existing microflow (round-trippable output).

### DROP MICROFLOW

```sql
DROP MICROFLOW <qualified-name>
```

---

## Page Statements

### CREATE PAGE

Creates a page with a widget tree.

**Syntax:**
```sql
CREATE [OR REPLACE] PAGE <qualified-name>
(
  [Params: { $Param: Module.Entity [, ...] },]
  Title: '<title>',
  Layout: <Module.LayoutName>
  [, Folder: '<path>']
)
{
  <widget-tree>
}
```

**Widget Syntax:**
```sql
WIDGET_TYPE widgetName (Property: value, ...) [{ children }]
```

**Supported Widget Types:**
- Layout: `LAYOUTGRID`, `ROW`, `COLUMN`, `CONTAINER`, `CUSTOMCONTAINER`
- Input: `TEXTBOX`, `TEXTAREA`, `CHECKBOX`, `RADIOBUTTONS`, `DATEPICKER`, `COMBOBOX`
- Display: `DYNAMICTEXT`, `DATAGRID`, `GALLERY`, `LISTVIEW`, `IMAGE`, `STATICIMAGE`, `DYNAMICIMAGE`
- Actions: `ACTIONBUTTON`, `LINKBUTTON`, `NAVIGATIONLIST`
- Structure: `DATAVIEW`, `HEADER`, `FOOTER`, `CONTROLBAR`, `SNIPPETCALL`

**Example:**
```sql
CREATE PAGE MyModule.Customer_Edit
(
  Params: { $Customer: MyModule.Customer },
  Title: 'Edit Customer',
  Layout: Atlas_Core.PopupLayout
)
{
  DATAVIEW dvCustomer (DataSource: $Customer) {
    TEXTBOX txtName (Label: 'Name', Attribute: Name)
    TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
    FOOTER footer1 {
      ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)
      ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
    }
  }
}
```

### DROP PAGE

```sql
DROP PAGE <qualified-name>
```

---

## ALTER PAGE / ALTER SNIPPET

Modifies an existing page or snippet's widget tree in-place without full `CREATE OR REPLACE`.

**Syntax:**
```sql
ALTER PAGE <qualified-name> {
  <operations>
}

ALTER SNIPPET <qualified-name> {
  <operations>
}
```

**Operations:**
```sql
-- Set property on widget
SET Caption = 'New' ON widgetName;
SET (Caption = 'Save', ButtonStyle = Success) ON btn;

-- Set page-level property
SET Title = 'New Title';

-- Insert widgets
INSERT AFTER widgetName { <widgets> };
INSERT BEFORE widgetName { <widgets> };

-- Remove widgets
DROP WIDGET name1, name2;

-- Replace widget
REPLACE widgetName WITH { <widgets> };

-- Pluggable widget properties (quoted)
SET 'showLabel' = false ON cbStatus;
```

**Example:**
```sql
ALTER PAGE Module.EditPage {
  SET (Caption = 'Save & Close', ButtonStyle = Success) ON btnSave;
  DROP WIDGET txtUnused;
  INSERT AFTER txtEmail {
    TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
  }
};
```

---

## MOVE Statements

### MOVE

Moves documents (pages, microflows, snippets, nanoflows, entities, enumerations) between folders and modules.

**Syntax:**
```sql
-- Move to folder within same module
MOVE PAGE <qualified-name> TO FOLDER '<folder-path>';
MOVE MICROFLOW <qualified-name> TO FOLDER '<folder-path>';
MOVE SNIPPET <qualified-name> TO FOLDER '<folder-path>';
MOVE NANOFLOW <qualified-name> TO FOLDER '<folder-path>';
MOVE ENUMERATION <qualified-name> TO FOLDER '<folder-path>';

-- Move to module root
MOVE PAGE <qualified-name> TO <module-name>;

-- Move entity to different module (no folder support)
MOVE ENTITY <qualified-name> TO <module-name>;

-- Move to folder in different module
MOVE PAGE <qualified-name> TO FOLDER '<folder-path>' IN <module-name>;
```

**Parameters:**
- `<qualified-name>` - The current Module.Name of the document
- `<folder-path>` - Target folder path (nested: 'Parent/Child')
- `<module-name>` - Target module name (for cross-module moves)

**Examples:**
```sql
-- Move page to folder
MOVE PAGE MyModule.CustomerEdit TO FOLDER 'Customers';

-- Move microflow to nested folder
MOVE MICROFLOW MyModule.ACT_ProcessOrder TO FOLDER 'Orders/Processing';

-- Move snippet to different module
MOVE SNIPPET OldModule.NavigationMenu TO Common;

-- Move entity to different module
MOVE ENTITY OldModule.Customer TO NewModule;

-- Move enumeration to different module
MOVE ENUMERATION OldModule.OrderStatus TO NewModule;

-- Move page to folder in different module
MOVE PAGE OldModule.CustomerPage TO FOLDER 'Screens' IN NewModule;
```

**Warning:** Cross-module moves change the qualified name and may break by-name references. Use `SHOW IMPACT OF <name>` to check before moving.

**Note:** `MOVE ENTITY` only supports moving to a module (not to a folder), since entities are embedded in domain model documents.

---

## Security Statements

### SHOW PROJECT SECURITY

Displays project-wide security settings.

**Syntax:**
```sql
SHOW PROJECT SECURITY
```

### SHOW MODULE ROLES

Lists module roles, optionally filtered by module.

**Syntax:**
```sql
SHOW MODULE ROLES
SHOW MODULE ROLES IN <module>
```

### SHOW USER ROLES

Lists project-level user roles.

**Syntax:**
```sql
SHOW USER ROLES
```

### SHOW DEMO USERS

Lists configured demo users.

**Syntax:**
```sql
SHOW DEMO USERS
```

### SHOW ACCESS ON

Shows which roles have access to a specific element.

**Syntax:**
```sql
SHOW ACCESS ON MICROFLOW <module>.<name>
SHOW ACCESS ON PAGE <module>.<name>
SHOW ACCESS ON <module>.<entity>
```

### SHOW SECURITY MATRIX

Displays a comprehensive access matrix for all or one module.

**Syntax:**
```sql
SHOW SECURITY MATRIX
SHOW SECURITY MATRIX IN <module>
```

### CREATE MODULE ROLE

Creates a new module role within a module.

**Syntax:**
```sql
CREATE MODULE ROLE <module>.<role> [DESCRIPTION '<text>']
```

**Example:**
```sql
CREATE MODULE ROLE Shop.Admin DESCRIPTION 'Full administrative access';
CREATE MODULE ROLE Shop.Viewer;
```

### DROP MODULE ROLE

Removes a module role.

**Syntax:**
```sql
DROP MODULE ROLE <module>.<role>
```

### GRANT EXECUTE ON MICROFLOW

Grants execute access on a microflow to one or more module roles.

**Syntax:**
```sql
GRANT EXECUTE ON MICROFLOW <module>.<name> TO <module>.<role> [, ...]
```

**Example:**
```sql
GRANT EXECUTE ON MICROFLOW Shop.ACT_Order_Process TO Shop.User, Shop.Admin;
```

### REVOKE EXECUTE ON MICROFLOW

Removes execute access on a microflow from one or more module roles.

**Syntax:**
```sql
REVOKE EXECUTE ON MICROFLOW <module>.<name> FROM <module>.<role> [, ...]
```

### GRANT VIEW ON PAGE

Grants view access on a page to one or more module roles.

**Syntax:**
```sql
GRANT VIEW ON PAGE <module>.<name> TO <module>.<role> [, ...]
```

### REVOKE VIEW ON PAGE

Removes view access on a page from one or more module roles.

**Syntax:**
```sql
REVOKE VIEW ON PAGE <module>.<name> FROM <module>.<role> [, ...]
```

### GRANT (Entity Access)

Creates an access rule on an entity for one or more module roles with CRUD permissions.

**Syntax:**
```sql
GRANT <module>.<role> ON <module>.<entity> (<rights>) [WHERE '<xpath>']
```

Where `<rights>` is a comma-separated list of:
- `CREATE` — allow creating instances
- `DELETE` — allow deleting instances
- `READ *` — read all members, or `READ (<attr>, ...)` for specific attributes
- `WRITE *` — write all members, or `WRITE (<attr>, ...)` for specific attributes

**Examples:**
```sql
-- Full access
GRANT Shop.Admin ON Shop.Customer (CREATE, DELETE, READ *, WRITE *);

-- Read-only
GRANT Shop.Viewer ON Shop.Customer (READ *);

-- Selective member access
GRANT Shop.User ON Shop.Customer (READ (Name, Email), WRITE (Email));

-- With XPath constraint
GRANT Shop.User ON Shop.Order (READ *, WRITE *) WHERE '[Status = ''Open'']';
```

### REVOKE (Entity Access)

Removes an entity access rule for specified roles.

**Syntax:**
```sql
REVOKE <module>.<role> ON <module>.<entity>
```

### CREATE USER ROLE

Creates a project-level user role that aggregates module roles.

**Syntax:**
```sql
CREATE USER ROLE <name> (<module>.<role> [, ...]) [MANAGE ALL ROLES]
```

**Example:**
```sql
CREATE USER ROLE AppAdmin (Shop.Admin, System.Administrator) MANAGE ALL ROLES;
CREATE USER ROLE AppUser (Shop.User);
```

### ALTER USER ROLE

Adds or removes module roles from a user role.

**Syntax:**
```sql
ALTER USER ROLE <name> ADD MODULE ROLES (<module>.<role> [, ...])
ALTER USER ROLE <name> REMOVE MODULE ROLES (<module>.<role> [, ...])
```

### DROP USER ROLE

Removes a project-level user role.

**Syntax:**
```sql
DROP USER ROLE <name>
```

### ALTER PROJECT SECURITY

Changes project-wide security settings.

**Syntax:**
```sql
ALTER PROJECT SECURITY LEVEL OFF | PROTOTYPE | PRODUCTION
ALTER PROJECT SECURITY DEMO USERS ON | OFF
```

### CREATE DEMO USER

Creates a demo user for development/testing.

**Syntax:**
```sql
CREATE DEMO USER '<username>' PASSWORD '<password>' [ENTITY <Module.Entity>] (<userrole> [, ...])
```

The optional `ENTITY` clause specifies the entity that generalizes `System.User` (e.g., `Administration.Account`). If omitted, the system auto-detects the unique `System.User` subtype.

**Example:**
```sql
CREATE DEMO USER 'demo_admin' PASSWORD 'Admin123!' (AppAdmin);
CREATE DEMO USER 'demo_admin' PASSWORD 'Admin123!' ENTITY Administration.Account (AppAdmin);
```

### DROP DEMO USER

Removes a demo user.

**Syntax:**
```sql
DROP DEMO USER '<username>'
```

---

## Navigation Statements

### SHOW NAVIGATION

```sql
SHOW NAVIGATION                    -- Summary of all profiles
SHOW NAVIGATION MENU [<profile>]   -- Menu tree
SHOW NAVIGATION HOMES              -- Home page assignments
```

### DESCRIBE NAVIGATION

```sql
DESCRIBE NAVIGATION [<profile>]    -- Full MDL output (round-trippable)
```

### CREATE OR REPLACE NAVIGATION

```sql
CREATE OR REPLACE NAVIGATION <profile>
  HOME PAGE Module.HomePage
  [HOME PAGE Module.AdminHome FOR Module.AdminRole]
  [LOGIN PAGE Module.LoginPage]
  [NOT FOUND PAGE Module.Custom404]
  [MENU (
    MENU ITEM 'Label' PAGE Module.Page;
    MENU 'Submenu' (
      MENU ITEM 'Label' PAGE Module.Page;
    );
  )]
```

**Example:**
```sql
CREATE OR REPLACE NAVIGATION Responsive
  HOME PAGE MyModule.Home_Web
  HOME PAGE MyModule.AdminHome FOR MyModule.Administrator
  LOGIN PAGE Administration.Login
  MENU (
    MENU ITEM 'Home' PAGE MyModule.Home_Web;
    MENU 'Admin' (
      MENU ITEM 'Users' PAGE Administration.Account_Overview;
    );
  );
```

---

## Settings Statements

### SHOW / DESCRIBE SETTINGS

```sql
SHOW SETTINGS              -- Overview of all settings
DESCRIBE SETTINGS           -- Full MDL output (round-trippable)
```

### SHOW CONSTANT VALUES

```sql
SHOW CONSTANT VALUES [IN Module]   -- Compare constant values across configurations
```

Displays one row per constant per configuration. Shows the default value followed by any per-configuration overrides.

### ALTER SETTINGS

```sql
ALTER SETTINGS MODEL Key = Value;
ALTER SETTINGS CONFIGURATION 'Name' Key = Value;
ALTER SETTINGS CONSTANT 'Name' VALUE 'val' IN CONFIGURATION 'cfg';
ALTER SETTINGS DROP CONSTANT 'Name' IN CONFIGURATION 'cfg';
ALTER SETTINGS LANGUAGE Key = Value;
ALTER SETTINGS WORKFLOWS Key = Value;
```

### CREATE / DROP CONFIGURATION

```sql
CREATE CONFIGURATION 'Name' [Key = Value, ...];
DROP CONFIGURATION 'Name';
```

**Example:**
```sql
ALTER SETTINGS MODEL AfterStartupMicroflow = 'MyModule.ACT_Startup';
ALTER SETTINGS CONFIGURATION 'default' DatabaseType = 'POSTGRESQL';
ALTER SETTINGS LANGUAGE DefaultLanguageCode = 'en_US';

-- View constant values across all configurations
SHOW CONSTANT VALUES;

-- Create a new configuration
CREATE CONFIGURATION 'Staging' DatabaseType = 'POSTGRESQL', DatabaseUrl = 'staging-db:5432';

-- Remove a constant override
ALTER SETTINGS DROP CONSTANT 'MyModule.ApiKey' IN CONFIGURATION 'Default';

-- Drop a configuration
DROP CONFIGURATION 'Staging';
```

---

## External SQL Statements

Direct SQL query execution against external databases (PostgreSQL, Oracle, SQL Server). Credentials are isolated from session output.

### SQL CONNECT

```sql
SQL CONNECT <driver> '<dsn>' AS <alias>
```

Drivers: `postgres` (pg, postgresql), `oracle` (ora), `sqlserver` (mssql).

### SQL Commands

```sql
SQL CONNECTIONS                     -- List active connections (alias + driver only)
SQL DISCONNECT <alias>              -- Close connection
SQL <alias> SHOW TABLES             -- List user tables
SQL <alias> SHOW VIEWS              -- List user views
SQL <alias> SHOW FUNCTIONS          -- List functions/procedures
SQL <alias> DESCRIBE <table>        -- Column details
SQL <alias> <any-sql>               -- Raw SQL passthrough
SQL <alias> GENERATE CONNECTOR INTO <module> [TABLES (...)] [VIEWS (...)] [EXEC]
```

**Example:**
```sql
SQL CONNECT postgres 'postgres://user:pass@localhost:5432/mydb' AS source;
SQL source SHOW TABLES;
SQL source SELECT * FROM users WHERE active = true LIMIT 10;
SQL source GENERATE CONNECTOR INTO HRModule TABLES (employees, departments) EXEC;
SQL DISCONNECT source;
```

---

## Import Statements

### IMPORT

Imports data from an external database into a Mendix application database.

**Syntax:**
```sql
IMPORT FROM <alias> QUERY '<sql>'
  INTO <Module.Entity>
  MAP (<source-col> AS <AttrName> [, ...])
  [LINK (<source-col> TO <AssocName> ON <MatchAttr>) [, ...]]
  [BATCH <size>]
  [LIMIT <count>]
```

**Example:**
```sql
IMPORT FROM source QUERY 'SELECT name, email, dept_name FROM employees'
  INTO HR.Employee
  MAP (name AS Name, email AS Email)
  LINK (dept_name TO Employee_Department ON Name)
  BATCH 500
  LIMIT 1000;
```

---

## Catalog and Search Statements

The catalog provides SQLite-based cross-reference queries over project metadata.

### REFRESH CATALOG

```sql
REFRESH CATALOG             -- Rebuild basic catalog
REFRESH CATALOG FULL        -- Rebuild including cross-references and source
```

### Catalog Queries

```sql
SHOW CATALOG TABLES                           -- List available catalog tables
SELECT ... FROM CATALOG.<table> [WHERE ...]   -- SQL query against catalog
```

### Cross-Reference Navigation

Requires `REFRESH CATALOG FULL` to populate reference data.

```sql
SHOW CALLERS OF <qualified-name>       -- What calls this element
SHOW CALLEES OF <qualified-name>       -- What this element calls
SHOW REFERENCES OF <qualified-name>    -- All references to/from
SHOW IMPACT OF <qualified-name>        -- Impact analysis
SHOW CONTEXT OF <qualified-name>       -- Surrounding context
```

### Full-Text Search

```sql
SEARCH '<keyword>'                     -- Search across all strings and source
```

---

## Business Event Statements

```sql
SHOW BUSINESS EVENTS [IN <module>]
DESCRIBE BUSINESS EVENT SERVICE <qualified-name>
CREATE BUSINESS EVENT SERVICE <qualified-name> (...) { MESSAGE ... }
DROP BUSINESS EVENT SERVICE <qualified-name>
```

---

## Java Action Statements

```sql
-- List and inspect
SHOW JAVA ACTIONS [IN <module>]
DESCRIBE JAVA ACTION <qualified-name>

-- Create with inline Java code
CREATE JAVA ACTION <qualified-name>(<params>) RETURNS <type>
  [EXPOSED AS '<caption>' IN '<category>']
  AS $$ <java-code> $$

-- Type parameters for generic entity handling
CREATE JAVA ACTION Module.Validate(
  EntityType: ENTITY <pEntity> NOT NULL,
  InputObject: pEntity NOT NULL
) RETURNS Boolean AS $$ return InputObject != null; $$

-- Drop
DROP JAVA ACTION <qualified-name>
```

Parameter types: `String`, `Integer`, `Long`, `Decimal`, `Boolean`, `DateTime`, `Module.Entity`, `List of Module.Entity`, `StringTemplate(Sql)`, `StringTemplate(Oql)`, `ENTITY <typeParam>`.

---

## Session Statements

### SET

Sets a session variable.

**Syntax:**
```sql
SET <key> = <value>
```

**Example:**
```sql
SET output_format = 'json'
SET verbose = TRUE
```

### REFRESH / UPDATE

Reloads the project from disk.

**Syntax:**
```sql
REFRESH
UPDATE
```

### EXECUTE SCRIPT

Executes an MDL script file.

**Syntax:**
```sql
EXECUTE SCRIPT '<path>'
```

**Example:**
```sql
EXECUTE SCRIPT './scripts/setup_domain_model.mdl';
```

### HELP

Shows available commands.

**Syntax:**
```sql
HELP
?
```

### EXIT / QUIT

Exits the REPL.

**Syntax:**
```sql
EXIT
QUIT
```

---

## Annotations

### @Position

Specifies the visual position in the domain model diagram.

**Syntax:**
```sql
@Position(<x>, <y>)
```

**Example:**
```sql
@Position(100, 200)
CREATE PERSISTENT ENTITY MyModule.Customer (
  ...
);
```

---

## Grammar (EBNF Summary)

This is a simplified overview. The complete formal grammar is defined in the ANTLR4 files:
- `mdl/grammar/MDLLexer.g4` - Lexer tokens
- `mdl/grammar/MDLParser.g4` - Parser rules

```ebnf
program         = { statement } ;

statement       = connect_stmt | disconnect_stmt | status_stmt
                | show_stmt | describe_stmt
                | create_module_stmt | drop_module_stmt
                | create_enum_stmt | alter_enum_stmt | drop_enum_stmt
                | create_entity_stmt | alter_entity_stmt | drop_entity_stmt
                | create_assoc_stmt | drop_assoc_stmt
                | create_microflow_stmt | drop_microflow_stmt
                | create_page_stmt | alter_page_stmt | drop_page_stmt
                | move_stmt
                | security_stmt | grant_stmt | revoke_stmt
                | navigation_stmt | settings_stmt
                | sql_stmt | import_stmt
                | catalog_stmt | search_stmt
                | business_event_stmt | java_action_stmt
                | set_stmt | refresh_stmt | execute_script_stmt
                | help_stmt | exit_stmt ;

qualified_name  = IDENTIFIER [ '.' IDENTIFIER [ '.' IDENTIFIER ] ] ;

data_type       = 'String' [ '(' INTEGER ')' ]
                | 'Integer' | 'Long' | 'Decimal' | 'Boolean'
                | 'DateTime' | 'Date' | 'AutoNumber' | 'Binary'
                | 'HashedString'
                | 'Enumeration' '(' qualified_name ')'
                | 'List' 'of' qualified_name
                | qualified_name ;

entity_type     = 'PERSISTENT' | 'NON-PERSISTENT' | 'VIEW' | 'EXTERNAL' ;

literal         = STRING | INTEGER | DECIMAL | 'TRUE' | 'FALSE' | 'NULL' ;
```
