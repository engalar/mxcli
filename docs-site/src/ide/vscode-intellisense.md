# Completion, Hover, Go-to-Definition

The VS Code extension provides IntelliSense features powered by the MDL language server (`mxcli lsp --stdio`).

## Code Completion

Press Ctrl+Space or start typing to trigger context-aware completions:

### Keyword Completion

At the start of a statement, the extension suggests top-level MDL keywords:

```
CREATE  SHOW  DESCRIBE  ALTER  DROP  GRANT  REVOKE  SEARCH  REFRESH  ...
```

After a keyword, context-appropriate follow-up keywords appear:

```sql
CREATE |
       ├── ENTITY
       ├── ASSOCIATION
       ├── ENUMERATION
       ├── MICROFLOW
       ├── PAGE
       ├── SNIPPET
       ├── MODULE
       ├── MODULE ROLE
       ├── USER ROLE
       └── WORKFLOW
```

### Snippet Completion

Common patterns are offered as multi-line snippets. For example, typing `entity` and selecting the snippet inserts:

```sql
CREATE PERSISTENT ENTITY ${1:Module}.${2:EntityName} (
    ${3:Name}: ${4:String(200)}
);
```

### Reference Completion

When the language server has a loaded project, qualified name references offer completions from the actual model:

- **Entity names** after `FROM`, `TO`, and in data source expressions
- **Module names** after `IN` and as the first part of qualified names
- **Attribute names** inside entity contexts
- **Microflow names** after `CALL MICROFLOW`

## Hover

Hover over any `Module.Name` qualified reference to see its MDL definition in a tooltip. For example, hovering over `Sales.Customer` in:

```sql
RETRIEVE $c FROM Sales.Customer WHERE ...
```

Displays the entity definition:

```sql
CREATE PERSISTENT ENTITY Sales.Customer (
    Name: String(200) NOT NULL,
    Email: String(200),
    IsActive: Boolean DEFAULT true
);
```

Hover works for entities, microflows, pages, enumerations, associations, and snippets.

## Go-to-Definition

**Ctrl+Click** (or F12) on a qualified reference opens the element's MDL source as a virtual read-only document. The definition is generated on-the-fly by describing the element from the project.

For example, Ctrl+clicking `Sales.ProcessOrder` opens a virtual document showing the full microflow definition with all its activities, conditions, and flows.

This also works from the terminal: qualified names like `MyModule.EntityName` in mxcli output are rendered as clickable links that open the definition.

## Document Symbols and Outline

The Outline panel (View > Outline) shows all MDL statements in the current file as a navigable tree:

```
├── CREATE ENTITY Sales.Customer
├── CREATE ENTITY Sales.Order
├── CREATE ASSOCIATION Sales.Order_Customer
├── CREATE MICROFLOW Sales.ProcessOrder
└── CREATE PAGE Sales.CustomerOverview
```

Click any entry to jump to that statement in the editor.

## Folding

Statement blocks can be collapsed:

- `CREATE MICROFLOW ... BEGIN ... END;` -- fold the microflow body
- `CREATE PAGE ... { ... }` -- fold the widget tree
- `IF ... THEN ... END IF;` -- fold conditional blocks
- `LOOP ... BEGIN ... END LOOP;` -- fold loop bodies
- `/** ... */` -- fold documentation comments
