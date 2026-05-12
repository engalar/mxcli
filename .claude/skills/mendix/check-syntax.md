# MDL Syntax Validation Skill

This skill ensures MDL scripts are validated before presenting them to users or executing them.

## When to Use This Skill

**ALWAYS** use this skill before:
- Presenting MDL code to users
- Executing MDL scripts via `mxcli exec`
- Committing MDL files to version control

## Pre-Flight Validation Checklist

Before writing any MDL, verify these requirements:

### 1. Check Supported Syntax

**Supported in Microflows:**
- `declare $Var type = value;` (primitives)
- `declare $entity Module.Entity;` (entities - no AS keyword, no = empty)
- `declare $list list of Module.Entity = empty;` (lists)
- `set $Var = expression;`
- `$Var = create Module.Entity (attr = value);`
- `change $entity (attr = value);`
- `commit $entity [with events] [refresh];`
- `delete $entity;`
- `retrieve $Var from Module.Entity [where condition];`
- `$Result = call microflow Module.Name (Param = $value);` (NOT `set $Result = ...`)
- `$Result = call nanoflow Module.Name (Param = $value);`
- `show page Module.PageName ($Param = $value);`
- `close page;`
- `validation feedback $entity/attribute message 'message';`
- `log info|warning|error [node 'name'] 'message';`
- `if condition then ... [else ...] end if;`
- `loop $item in $list begin ... end loop;`
- `return $value;`
- `on error continue|rollback|{ handler };`

**Now Supported (previously not):**
- `rollback $entity [refresh];` - Reverts uncommitted changes
- `retrieve ... limit n` - Returns single entity when `limit 1`
- `boolean` without `default` - Auto-defaults to `false`
- `buttonstyle: warning` and `buttonstyle: info` - Now parse correctly
- Keywords as attribute names - `caption`, `label`, `title`, `text`, `content`, `format`, `range`, `source`, `check`, etc. all work unquoted

**NOT Supported (will cause errors):**
- `set $var = call microflow ...` - Use `$var = call microflow ...` (no SET)
- `while ... end while` - Use `loop` with lists
- `case ... when ... end case` - Use nested `if`
- `TRY ... CATCH` - Use `on error` blocks
- `break` / `continue` - Not implemented
- `commit message 'text'` - Not in current grammar (session command only)

### 2. Quote All Identifiers

**Best practice: Always quote all identifiers** (entity names, attribute names, parameter names) with double quotes. This eliminates all reserved keyword conflicts and is always safe — quotes are stripped automatically by the parser.

```sql
create persistent entity Module."Customer" (
  "Name": string(200),
  "status": string(50),
  "create": datetime
);
```

Both `"Name"` and `` `Name` `` syntax are supported. Prefer double quotes for consistency.

Run `mxcli syntax keywords` for the full list of 320+ reserved keywords.

### 3. Validate with mxcli

**Always run these checks:**

```bash
# Step 1: Syntax check (no project needed)
./bin/mxcli check script.mdl

# Step 2: reference validation (needs project)
# Validates microflow bodies, entity/enum references, and widget tree references
# (datasource microflow/nanoflow/entity, action page/microflow, snippet refs)
./bin/mxcli check script.mdl -p app.mpr --references
```

### 4. Common Error Patterns

| Error Message | Likely Cause | Fix |
|---------------|--------------|-----|
| `mismatched input 'set'` after `call microflow` | SET not valid with CALL | Use `$var = call microflow ...` |
| `mismatched input 'create'` | Structural keyword as identifier | Use `"create"` (quoted) or rename |
| `no viable alternative at input` | Unsupported syntax | Check supported statements list |
| `microflow not found` | Referenced before created | Move microflow definition earlier or check spelling |
| `page not found` | Page doesn't exist | Check qualified name with `--references` |
| `entity not found` | Typo or wrong module | Use fully qualified name |

## Validation Workflow

### Before Writing MDL

1. **Read the skill files:**
   ```bash
   cat .claude/skills/write-microflows.md
   cat .claude/skills/overview-pages.md
   ```

2. **Check help for specific syntax:**
   ```bash
   ./bin/mxcli syntax microflow
   ./bin/mxcli syntax page
   ./bin/mxcli syntax entity
   ```

### After Writing MDL

1. **Save to a file:**
   ```bash
   cat > script.mdl << 'EOF'
   -- Your MDL here
   EOF
   ```

2. **Run syntax check:**
   ```bash
   ./bin/mxcli check script.mdl
   ```

3. **If errors, check specific syntax:**
   ```bash
   ./bin/mxcli syntax keywords    # Reserved words
   ./bin/mxcli syntax microflow   # microflow syntax
   ```

4. **Run reference check (with project):**
   ```bash
   ./bin/mxcli check script.mdl -p app.mpr --references
   ```

5. **Execute only after all checks pass:**
   ```bash
   ./bin/mxcli exec script.mdl -p app.mpr
   ```

### Expression Validation (for microflow expressions)

Every expression inside a microflow statement must be separately validated with the expression checker. `mxcli check` catches structural MDL errors; the expression checker catches semantic errors (wrong types, enum mismatches, arity mistakes).

**Before writing an expression** — look up what type the slot expects and see real examples:
```bash
./bin/mxcli show expr-slot IfStmt.Condition
./bin/mxcli show expr-slot ChangeItem.Value
./bin/mxcli show expr-slot CallArgument.Value
# → prints ExpectedKind (Boolean / Integer / …) and mined sample expressions
```

**SlotPath reference:**

| MDL position | SlotPath |
|---|---|
| `IF <expr> THEN` | `IfStmt.Condition` |
| `WHILE <expr> BEGIN` | `WhileStmt.Condition` |
| `RETURN <expr>` | `ReturnStmt.Value` |
| `DECLARE $x T = <expr>` | `DeclareStmt.InitialValue` |
| `SET $x = <expr>` | `MfSetStmt.Value` |
| `CHANGE $obj (Attr = <expr>)` | `ChangeItem.Value` |
| `CREATE Mod.Entity (Attr = <expr>)` | `CreateItem.Value` |
| `CALL Mf(Param = <expr>)` | `CallArgument.Value` |
| `LOG INFO <expr>` | `LogStmt.Message` |

**After writing an expression** — validate it before using it:
```bash
./bin/mxcli explain expression '$Alert/Status = empty' --in IfStmt.Condition
# → "no hints — expression is well-formed for this slot" means safe to use

./bin/mxcli explain expression "'NewAlert'" --in ChangeItem.Value
# → HINT [E001 enum-string-mismatch] error — must fix before using
```

**If a hint code appears**, look it up:
```bash
./bin/mxcli hint E001   # enum-string-mismatch
./bin/mxcli hint E002   # bool-string-mismatch
./bin/mxcli hint E003   # null-to-empty (use empty, not null)
./bin/mxcli hint E004   # concat-type (toString() needed)
./bin/mxcli hint E006   # func-arg-arity (wrong number of args)
./bin/mxcli hint E007   # unknown-token (parse error in expression)
```

**Rule:** An expression is only safe when `mxcli explain expression` returns `no hints`.

## Script Execution Behavior

**IMPORTANT: Script execution is atomic per statement, NOT per script.**

When a script fails on statement N, statements 1 through N-1 have already been committed:

```
Statement 1: create module ✓ (committed)
Statement 2: create entity ✓ (committed)
Statement 3: create association ✓ (committed)
Statement 4: create view entity ✗ (failed - execution stops here)
Statement 5: create page (never executed)
```

**Recommendations:**
1. Split scripts into phases when experimenting with uncertain syntax
2. Use `create or replace` to make scripts idempotent
3. Test new syntax patterns with minimal scripts first
4. Keep a backup of your project before running large scripts

## Script Organization

Organize scripts in dependency order:

```mdl
-- ============================================
-- PHASE 1: Enumerations (no dependencies)
-- ============================================
create enumeration Module.Status (
  Active = 'Active',
  Inactive = 'Inactive'
);
/

-- ============================================
-- PHASE 2: Entities (depend on enumerations)
-- ============================================
create persistent entity Module.Customer (
  Name: string(200),
  status: Module.Status
);
/

-- ============================================
-- PHASE 3: Associations (depend on entities)
-- ============================================
create association Module.Order_Customer (
  Module.Order [*] -> Module.Customer [1]
);
/

-- ============================================
-- PHASE 4: Microflows (depend on entities)
-- ============================================
create microflow Module.ACT_Save ($Customer: Module.Customer)
returns boolean as $success
begin
  declare $success boolean = false;
  commit $Customer;
  set $success = true;
  return $success;
end;
/

-- ============================================
-- PHASE 5: Pages (depend on microflows)
-- ============================================
create page Module.Customer_Edit
layout Atlas_Default
title 'Edit Customer'
parameter $Customer: Module.Customer
widgets (
  -- Can reference microflows created in Phase 4
  button 'Save' call microflow Module.ACT_Save (Customer = $Customer)
);
/
```

## Troubleshooting Parse Errors

### Error: "snippet not found" / "page not found"

A reference to a document that hasn't been created yet in the script:

```
Error: snippet not found: MyModule.NavMenu
Error: page not found: MyModule.Customer_NewEdit
```

Script execution is sequential — each `CREATE` commits immediately. Forward references
fail because the target doesn't exist in the database at the moment the referencing
document is created.

**Fix options:**
1. **Reorder** — move the target document's `CREATE` earlier in the script (simplest fix)
2. **Placeholder pattern** — for circular dependencies (e.g. a snippet that shows pages
   that embed the snippet), create a minimal placeholder first, then create the referencing
   documents, then fill in the placeholder with `CREATE OR MODIFY` — which preserves the
   original UUID so all existing bindings remain valid
   (see [Resolve Forward References](./resolve-forward-references.md))

Declaration order that avoids most forward references:
```
enumerations → entities → snippets (placeholder) → pages → snippets (fill-in) → microflows → navigation
```

> **Never use `CREATE OR REPLACE` for the placeholder fill-in step.** OR REPLACE deletes
> the placeholder and creates a new document with a different UUID, silently breaking
> every page or snippet that references it.

### Error: "mismatched input 'X'"

The word `X` is either:
1. A reserved word - rename the identifier
2. Unsupported syntax - check the supported statements list
3. A typo - check spelling

### Error: "no viable alternative at input"

The parser expected something different:
1. Check for missing semicolons
2. Check for missing `end if`, `end loop`, etc.
3. Verify statement syntax against the reference

### Error: "extraneous input"

Extra tokens found:
1. Check for stray characters
2. Check for duplicate semicolons
3. Verify string quotes are balanced

## Related Skills

- [/write-microflows](./write-microflows.md) - Detailed microflow syntax
- [/overview-pages](./overview-pages.md) - Page building syntax
- [/resolve-forward-references](./resolve-forward-references.md) - Placeholder pattern and declaration ordering
- [/migrate-oracle-forms](./migrate-oracle-forms.md) - Migration-specific guidance

## Expression Hint Reference

All 10 hint codes (E001–E010) with full explanations and fix examples:
`docs/06-mdl-reference/expr-hints.md`

Quick lookup: `./bin/mxcli hint <code>`
