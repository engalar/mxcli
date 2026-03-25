# Common Mistakes and Anti-Patterns

Frequent errors encountered when writing MDL scripts or using the SDK, with explanations of why they occur and how to avoid them.

## EXTENDS Must Appear Before `(`

**CRITICAL:** The `EXTENDS` clause goes BEFORE the opening parenthesis, not after.

```sql
-- CORRECT: EXTENDS before (
CREATE PERSISTENT ENTITY Module.ProductPhoto EXTENDS System.Image (
  PhotoCaption: String(200)
);

-- WRONG: EXTENDS after ) = parse error!
CREATE PERSISTENT ENTITY Module.Photo (
  PhotoCaption: String(200)
) EXTENDS System.Image;
```

## String Attributes Need Explicit Length

Always specify an explicit length for String attributes. While `String` defaults to 200, it is clearer and safer to use `String(200)` explicitly. Use `String(unlimited)` for long text.

```sql
-- GOOD: Explicit length
Name: String(200)
Description: String(unlimited)
Code: String(10)

-- AVOID: Implicit default length
Name: String
```

## Never Create Empty List Variables as Loop Sources

If processing imported data, accept the list as a microflow parameter. `DECLARE $Items List of ... = empty` followed by `LOOP $Item IN $Items` is always wrong -- the loop body will never execute.

```sql
-- WRONG: Loop over empty list does nothing
DECLARE $Items List of Module.Item = empty;
LOOP $Item IN $Items BEGIN
  -- This never executes!
END LOOP;

-- CORRECT: Accept the list as a parameter
CREATE MICROFLOW Module.ProcessItems
  PARAMETER $Items: List of Module.Item
BEGIN
  LOOP $Item IN $Items BEGIN
    -- Process each item
  END LOOP;
  RETURN true;
END;
```

## Never Use Nested LOOPs for List Matching

Loop over the primary list and use `RETRIEVE` with a filter for O(N) lookup. Nested loops are O(N^2).

```sql
-- WRONG: O(N^2) nested loops
LOOP $Item IN $Items BEGIN
  LOOP $Match IN $Targets BEGIN
    IF $Item/Key = $Match/Key THEN
      -- ...
    END IF;
  END LOOP;
END LOOP;

-- CORRECT: O(N) with RETRIEVE
LOOP $Item IN $Items BEGIN
  RETRIEVE $Match FROM $Targets WHERE Key = $Item/Key LIMIT 1;
  IF $Match != empty THEN
    -- ...
  END IF;
END LOOP;
```

## Association ParentPointer/ChildPointer Semantics Are Inverted

**CRITICAL:** Mendix BSON uses inverted naming for association pointers. This is counter-intuitive and a common source of bugs.

| BSON Field | Points To | MDL Keyword |
|------------|-----------|-------------|
| `ParentPointer` | **FROM** entity (FK owner) | `FROM Module.Child` |
| `ChildPointer` | **TO** entity (referenced) | `TO Module.Parent` |

For example, `CREATE ASSOCIATION Mod.Child_Parent FROM Mod.Child TO Mod.Parent` stores:
- `ParentPointer = Child.$ID` (the FROM entity owns the foreign key)
- `ChildPointer = Parent.$ID` (the TO entity is being referenced)

This affects **entity access rules**: MemberAccess entries for associations must only be added to the **FROM** entity (the one stored in `ParentPointer`). Adding them to the TO entity triggers CE0066 "Entity access is out of date".

The same convention applies in `domainmodel.Association`: `ParentID` = FROM entity, `ChildID` = TO entity.

## BSON Storage Names vs Qualified Names

Mendix uses different "storage names" in BSON `$Type` fields than the "qualified names" shown in the TypeScript SDK documentation. Using the wrong name causes `TypeCacheUnknownTypeException` when opening in Studio Pro.

| Qualified Name (SDK/docs) | Storage Name (BSON $Type) | Note |
|---------------------------|---------------------------|------|
| `CreateObjectAction` | `CreateChangeAction` | |
| `ChangeObjectAction` | `ChangeAction` | |
| `DeleteObjectAction` | `DeleteAction` | |
| `CommitObjectsAction` | `CommitAction` | |
| `RollbackObjectAction` | `RollbackAction` | |
| `AggregateListAction` | `AggregateAction` | |
| `ListOperationAction` | `ListOperationsAction` | |
| `ShowPageAction` | `ShowFormAction` | "Form" was original term for "Page" |
| `ClosePageAction` | `CloseFormAction` | "Form" was original term for "Page" |

When adding new types, always verify the storage name by:
1. Examining existing MPR files with the `mx` tool or SQLite browser
2. Checking the reflection data in the `reference/mendixmodellib/reflection-data/` directory
3. Looking at the parser cases in `sdk/mpr/parser_microflow.go`

## Mendix Expression String Escaping

When generating Mendix expression strings (e.g., in `expressionToString()`), single quotes within string literals must be escaped by doubling them: `'it''s here'`. Do NOT use backslash escaping (`\'`). This matches Mendix Studio Pro's expression syntax.

## Microflows: Unsupported Constructs

These constructs will cause parse errors:

| Unsupported | Use Instead |
|-------------|-------------|
| `CASE ... WHEN ... END CASE` | Nested `IF ... ELSE ... END IF` |
| `TRY ... CATCH ... END TRY` | `ON ERROR { ... }` blocks on specific activities |

## Boolean Attributes Must Have Defaults

Mendix Studio Pro requires Boolean attributes to have a `DEFAULT` value. If no `DEFAULT` is specified, MDL auto-defaults to `false`.

```sql
-- GOOD: Explicit default
IsActive: Boolean DEFAULT true

-- RISKY: Implicit default to false
Deleted: Boolean
```

## Always Validate Before Presenting to Users

Always run syntax and reference checks before delivering MDL scripts:

```bash
# Syntax + anti-pattern check
mxcli check script.mdl

# With reference validation against a project
mxcli check script.mdl -p app.mpr --references
```
