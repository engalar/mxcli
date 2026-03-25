# SQLite Schema

The catalog uses an in-memory SQLite database with the following table definitions.

## Core Tables

### MODULES

```sql
CREATE TABLE MODULES (
    Name        TEXT PRIMARY KEY,
    ModuleID    TEXT,
    SortIndex   INTEGER
);
```

### ENTITIES

```sql
CREATE TABLE ENTITIES (
    Name            TEXT PRIMARY KEY,   -- Qualified: Module.Entity
    ModuleName      TEXT,
    EntityName      TEXT,
    Persistent      BOOLEAN,
    AttributeCount  INTEGER,
    Documentation   TEXT
);
```

### MICROFLOWS

```sql
CREATE TABLE MICROFLOWS (
    Name            TEXT PRIMARY KEY,   -- Qualified: Module.Microflow
    ModuleName      TEXT,
    MicroflowName   TEXT,
    ActivityCount   INTEGER,
    ParameterCount  INTEGER,
    ReturnType      TEXT,
    Folder          TEXT,
    Documentation   TEXT
);
```

### NANOFLOWS

```sql
CREATE TABLE NANOFLOWS (
    Name            TEXT PRIMARY KEY,
    ModuleName      TEXT,
    NanoflowName    TEXT,
    ActivityCount   INTEGER,
    Documentation   TEXT
);
```

### PAGES

```sql
CREATE TABLE PAGES (
    Name        TEXT PRIMARY KEY,
    ModuleName  TEXT,
    PageName    TEXT,
    Layout      TEXT,
    Url         TEXT,
    Documentation TEXT
);
```

### SNIPPETS

```sql
CREATE TABLE SNIPPETS (
    Name        TEXT PRIMARY KEY,
    ModuleName  TEXT,
    SnippetName TEXT
);
```

### ENUMERATIONS

```sql
CREATE TABLE ENUMERATIONS (
    Name        TEXT PRIMARY KEY,
    ModuleName  TEXT,
    EnumName    TEXT,
    ValueCount  INTEGER
);
```

### WORKFLOWS

```sql
CREATE TABLE WORKFLOWS (
    Name            TEXT PRIMARY KEY,
    ModuleName      TEXT,
    WorkflowName    TEXT,
    ActivityCount   INTEGER
);
```

## Full Refresh Tables

These tables are only populated by `REFRESH CATALOG FULL`.

### ACTIVITIES

```sql
CREATE TABLE ACTIVITIES (
    DocumentName    TEXT,       -- Parent microflow/nanoflow
    ActivityType    TEXT,       -- e.g., "CreateObjectAction", "CallMicroflowAction"
    Caption         TEXT,       -- Activity caption/description
    SortOrder       INTEGER     -- Order within the flow
);
```

### WIDGETS

```sql
CREATE TABLE WIDGETS (
    DocumentName    TEXT,       -- Parent page/snippet
    WidgetName      TEXT,       -- Widget instance name
    WidgetType      TEXT,       -- e.g., "Forms$TextBox", "CustomWidgets$ComboBox"
    ModuleName      TEXT
);
```

### REFS

```sql
CREATE TABLE REFS (
    SourceName  TEXT,           -- Referencing document
    SourceKind  TEXT,           -- "Microflow", "Page", etc.
    TargetName  TEXT,           -- Referenced element
    TargetKind  TEXT,           -- "Entity", "Microflow", etc.
    RefKind     TEXT            -- "Call", "DataSource", "Association", etc.
);

CREATE INDEX idx_refs_source ON REFS(SourceName);
CREATE INDEX idx_refs_target ON REFS(TargetName);
```

### PERMISSIONS

```sql
CREATE TABLE PERMISSIONS (
    RoleName    TEXT,           -- Module role
    TargetName  TEXT,           -- Entity, microflow, or page
    TargetKind  TEXT,           -- "Entity", "Microflow", "Page"
    Permission  TEXT            -- "Create", "Read", "Write", "Delete", "Execute", "View"
);
```

## Full-Text Search Tables

### STRINGS (FTS5)

```sql
CREATE VIRTUAL TABLE STRINGS USING fts5(
    name,           -- Document qualified name
    kind,           -- Document type
    strings,        -- All text content concatenated
    tokenize='porter unicode61'
);
```

### SOURCE (FTS5)

```sql
CREATE VIRTUAL TABLE SOURCE USING fts5(
    name,           -- Document qualified name
    kind,           -- Document type
    source,         -- MDL source representation
    tokenize='porter unicode61'
);
```

## Querying Examples

```sql
-- Find entities with many attributes
SELECT Name, AttributeCount FROM CATALOG.ENTITIES
WHERE AttributeCount > 20 ORDER BY AttributeCount DESC;

-- Find all references to an entity
SELECT SourceName, RefKind FROM CATALOG.REFS
WHERE TargetName = 'Sales.Customer';

-- Full-text search
SELECT name, kind, snippet(STRINGS, 2, '<b>', '</b>', '...', 20)
FROM CATALOG.STRINGS WHERE strings MATCH 'validation error';
```
