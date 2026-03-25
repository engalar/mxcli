# Lexical Structure

This page describes the tokens that make up the MDL language: keywords, literals, and identifiers.

## Keywords

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

Most keywords work **unquoted** as identifiers (entity names, attribute names). Only structural keywords like `CREATE`, `DELETE`, `BEGIN`, `END`, `RETURN`, `ENTITY`, and `MODULE` require quoting when used as identifiers.

## Literals

### String Literals

String literals use single quotes:

```sql
'single quoted string'
'it''s here'            -- doubled single quote to escape
```

The only escape sequence is `''` (two single quotes) to represent a literal single quote. Backslash escaping is **not** supported.

### Numeric Literals

```sql
42          -- Integer
3.14        -- Decimal
-100        -- Negative integer
1.5e10      -- Scientific notation
```

### Boolean Literals

```sql
TRUE
FALSE
```

## Quoted Identifiers

When an identifier collides with a reserved keyword, use double quotes (ANSI SQL style) or backticks (MySQL style):

```sql
"ComboBox"."CategoryTreeVE"
`Order`.`Status`
"ComboBox".CategoryTreeVE    -- mixed quoting is allowed
```

See [Qualified Names](./qualified-names.md) for more on identifier syntax and the `Module.Name` notation.
