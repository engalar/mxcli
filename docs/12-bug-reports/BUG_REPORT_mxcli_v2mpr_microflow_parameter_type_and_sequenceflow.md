# Bug Report: mxcli CREATE MICROFLOW in MPR v2 — entity parameter type lost + CE0108 SequenceFlow error

## Summary

When `mxcli exec` creates a new microflow in a Mendix 11.6.6 MPR v2 project, two related bugs occur:

1. **Entity parameter type is stored as `Object`** instead of the declared entity type (e.g., `PayerRegistration.OrgPayerChoice_Dto` becomes `Object`).
2. **CE0108 "Variable not in scope"** is reported for uses of that parameter, indicating the SequenceFlow connections in the generated BSON do not properly connect activities to the Start Event.

Both bugs make newly created microflows non-functional in Studio Pro and fail `mx check`.

## Environment

- **mxcli version**: current (as of 2026-05-19)
- **Mendix version**: 11.6.6
- **MPR format**: v2 (mxunit-based)
- **Platform**: Linux (Ubuntu 22.04)

## Steps to Reproduce

1. Open a Mendix 11.6.6 MPR v2 project.
2. Create a microflow with a typed entity parameter:
   ```sql
   create or modify microflow MyModule.ACT_Example (
     $Dto: MyModule.MyEntity
   )
   returns Nothing
   begin
     if $Dto = empty then
       return;
     end if;
     change $Dto (SomeAttr = 'value');
     return;
   end;
   /
   ```
3. After fixing the Point/Size format bug (see separate bug report), run:
   ```bash
   ./mxcli docker check -p project.mpr
   ```
4. Describe the created microflow:
   ```bash
   ./mxcli -p project.mpr -c "DESCRIBE MICROFLOW MyModule.ACT_Example"
   ```

## Actual Results

### Bug 1 — Parameter type lost

`DESCRIBE MICROFLOW` returns:
```sql
create or modify microflow MyModule.ACT_Example (
  $Dto: Object        -- ← should be MyModule.MyEntity
)
```
The entity type is replaced with `Object`.

### Bug 2 — CE0108 SequenceFlow error

`mx check` reports:
```
[CE0108] "Variable 'Dto' is defined but not in scope at this location."
    at Decision '$Dto = empty'
[CE0108] "Variable 'Dto' is defined but not in scope at this location."
    at Change object activity 'Change ...'
```

`$Dto` is a microflow parameter (always in scope), but CE0108 indicates the activity nodes are not connected to the Start Event via SequenceFlow edges in the BSON.

## Expected Results

1. Parameter type should be preserved: `$Dto: MyModule.MyEntity`
2. All activities should be reachable from Start Event → no CE0108 errors

## Root Cause Analysis

### Bug 1 — Entity type serialization

In MPR v2, the mxunit BSON for a microflow parameter stores the type reference as a `$ref` pointer to the entity's unit ID. When mxcli creates a new microflow in v2 format, it appears to write `Object` (the base type) instead of resolving the entity reference and writing the correct `$ref` BSON pointer.

This is likely because the v2 mxunit format requires resolving entity UUIDs across the unit index, which mxcli's new-object creation path may not handle for entity-typed parameters.

### Bug 2 — SequenceFlow connectivity

In the Mendix microflow BSON, each activity node has:
- `SequenceFlowsIn`: list of incoming flow connections
- `SequenceFlowsOut`: list of outgoing flow connections

When mxcli generates a new microflow, the SequenceFlow edges between activities appear to be malformed — possibly using incorrect ID references or missing required fields — causing the Mendix engine to not recognize the activities as connected to the Start Event. This makes any variable defined before an activity (including parameters) appear "not in scope."

## Evidence

Comparing DESCRIBE output before and after exec:

**MDL input:**
```sql
create or modify microflow PayerRegistration.ACT_POC_NewPayer (
  $OrgDto: PayerRegistration.OrgPayerChoice_Dto
)
returns Nothing
begin
  if $OrgDto = empty then
    return;
  end if;
  -- ...
end;
```

**DESCRIBE output after exec:**
```sql
create or modify microflow PayerRegistration.ACT_POC_NewPayer (
  $OrgDto: Object    -- ← type lost
)
begin
  if $OrgDto = empty then
    return;
  else
    -- ...
    show page UnknownPage;  -- ← page reference also lost
    return;
  end if;
end;
```

CE0108 errors for all uses of `$OrgDto`.

## Workaround

Create the microflow in Studio Pro manually. The mxcli DESCRIBE syntax can be used to document the required signature; the implementation must be done in Studio Pro for v2 MPR projects until this bug is fixed.

## Relationship to Other Bugs

This bug likely shares a root cause with the Point/Size format bug (separate report): both occur because the mxcli BSON serialization path for v2 mxunit files uses a different (incomplete) serialization format compared to what Studio Pro 11 writes.

## Affected Operations

- `CREATE [OR MODIFY] MICROFLOW` with entity-typed parameters
- Possibly also entity-typed return types and entity-typed local variables

## Priority

**High** — makes it impossible to create any new typed microflow via `mxcli exec` in MPR v2 format. Typed parameters are required for all non-trivial microflows.

## Suggested Fix

1. In the mxunit BSON generation for microflow parameters, look up the target entity's unit UUID and write a proper `$ref` BSON pointer instead of writing `Object` type.
2. Fix the SequenceFlow generation to use correct UUID references and required fields so that activity nodes are properly connected in the flow graph.
3. Consider adding a validation step: after generating a new mxunit file, verify it round-trips through DESCRIBE without type loss.
