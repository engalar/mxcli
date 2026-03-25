# Proposal: Page Variables Support

## Summary

Add support for **page variables** (`Forms$LocalVariable`) — local variables defined at the page level that can be referenced in expressions throughout the page. This enables column visibility expressions, conditional formatting, and other dynamic behavior without requiring `$currentObject`.

## Background

### What are page variables?

Page variables are a Mendix feature (available in Mendix 10+) that allows declaring typed variables at the page level with default value expressions. They are stored in the page's `Variables` array in BSON.

### BSON Structure

```json
{
  "Variables": [
    3,
    {
      "$ID": "...",
      "$Type": "Forms$LocalVariable",
      "DefaultValue": "if ( 3 < 4 ) then true else false",
      "Name": "showStockColumn",
      "VariableType": {
        "$ID": "...",
        "$Type": "DataTypes$BooleanType"
      }
    }
  ]
}
```

### Why needed?

- **Column Visible**: DataGrid2 column `visible` property hides/shows the *entire column* for all rows. Using `$currentObject/Attr` here doesn't make sense — it needs a page-level boolean. Studio Pro shows an error if `$currentObject` is used.
- **Conditional logic**: Page variables allow toggling sections, panels, or modes based on user interaction or computed state.
- **Expression syntax**: Mendix expressions use `if (...) then ... else ...` syntax, NOT `if(..., ..., ...)` function-call style.

### Current state (broken)

The MDL example `P033b_DataGrid_ColumnProperties` had two issues:
1. `Visible: '$currentObject/IsActive'` — invalid, columns need page variables for visibility
2. `DynamicCellClass: 'if($currentObject/Stock < 10, ''text-danger'', '''')'` — wrong if-syntax

## Proposed MDL Syntax

### Page variable declaration

Add a `Variables` block to page/snippet headers:

```sql
CREATE PAGE MyModule.ProductOverview (
  Title: 'Products',
  Layout: Atlas_Core.Atlas_Default,
  Variables: {
    $showStockColumn: Boolean = 'if (3 < 4) then true else false',
    $filterMode: String = '''active'''
  }
) {
  DATAGRID dgProducts (DataSource: DATABASE MyModule.Product) {
    COLUMN colName (Attribute: Name, Caption: 'Name')
    COLUMN colStock (
      Attribute: Stock, Caption: 'Stock',
      Visible: '$showStockColumn'
    )
  }
}
```

### Syntax details

```
Variables: {
  $varName: DataType = 'defaultValueExpression',
  ...
}
```

- **DataType**: `Boolean`, `String`, `Integer`, `Decimal`, `DateTime`, or entity type
- **Default value**: Mendix expression string (required, in single quotes)
- Variable names prefixed with `$` (consistent with parameters)

### DESCRIBE output

```sql
CREATE PAGE MyModule.ProductOverview (
  Title: 'Products',
  Layout: Atlas_Core.Atlas_Default,
  Variables: { $showStockColumn: Boolean = 'true' }
) { ... }
```

## Implementation Plan

### Phase 1: DESCRIBE support (read-only)

1. **Read Variables from BSON** in `cmd_pages_describe.go`:
   - Extract `Variables` array from raw page data
   - Parse each `Forms$LocalVariable`: Name, VariableType, DefaultValue
   - Resolve `VariableType.$Type` to MDL type name (e.g., `DataTypes$BooleanType` → `Boolean`)

2. **Output in page header**: Add `Variables: { ... }` to props list in `describePage()`

3. **Same for snippets**: Snippets also support local variables

**Files**: `cmd_pages_describe.go`

### Phase 2: CREATE support (write)

1. **Grammar**: Add `variablesBlock` rule to `MDLParser.g4`:
   ```antlr
   pageProperty
       : ...
       | VARIABLES COLON LBRACE variableDecl (COMMA variableDecl)* RBRACE
       ;

   variableDecl
       : DOLLAR_IDENT COLON dataType ASSIGN STRING_LITERAL
       ;
   ```

2. **AST**: Add `Variables` field to page AST node (list of `{Name, Type, DefaultValue}`)

3. **Visitor**: Build variable list from parse tree

4. **Builder**: In page creation, serialize `Variables` array with `Forms$LocalVariable` entries:
   - Generate `$ID`
   - Set `$Type` to `Forms$LocalVariable`
   - Set `Name`, `DefaultValue`, and `VariableType` (DataType BSON)

**Files**: `MDLParser.g4`, `ast/ast_page_v3.go`, `visitor/visitor_page_v3.go`, `cmd_pages_builder.go`

### Phase 3: ALTER PAGE support

Add ability to add/modify/remove page variables via ALTER PAGE:

```sql
ALTER PAGE MyModule.ProductOverview
  ADD VARIABLE $showDetails: Boolean = 'true';

ALTER PAGE MyModule.ProductOverview
  DROP VARIABLE $showDetails;
```

**Files**: `MDLParser.g4`, `ast/ast_alter.go`, `executor/cmd_alter_page.go`

## Effort Estimate

| Phase | Scope | Complexity |
|-------|-------|------------|
| Phase 1 | DESCRIBE read | Low — parse BSON Variables array, format output |
| Phase 2 | CREATE write | Medium — grammar, AST, visitor, BSON serialization |
| Phase 3 | ALTER PAGE | Medium — add/drop variable operations |

## Risks

- **DataType mapping**: Need to map all `DataTypes$*` BSON types to MDL type names. Most are straightforward but entity types need qualified name resolution.
- **Expression validation**: Page variable default expressions should be valid Mendix expressions. We can't fully validate these but can pass them through as strings.
- **Version compatibility**: Page variables may not exist in older Mendix versions. Need to check if `Variables` array is always present or version-dependent.

## Related fixes in this changeset

- Removed `Visible: '$currentObject/...'` from DataGrid column example (columns hide entire column, not per-row)
- Fixed `DynamicCellClass` expression syntax: `if(...) then ... else ...` (not function-call style)
- Updated docs and skills with correct expression syntax
