---
title: CE0066 Root Cause — gen-native $ID encoding bug + CE0066 association reconcile
date: 2026-05-13
status: approved
---

# CE0066 Root Cause Investigation

## Problem Statement

Running `mx check macnica/mendix-app/MacnicaApp.mpr` fails after mxcli GRANT operations.
The user expected CE0066 ("Entity access is out of date"), but the actual first error is a crash:

```
System.InvalidCastException: Unable to cast object of type 'System.String' to type 'System.Byte[]'
   at Mendix.Modeler.Storage.ContentsUtil.GetGuidFromBson(JToken token)
```

This happens because a BSON `$ID` field is written as `""` (empty string) instead of Binary.

---

## Layer 1 — Critical Bug: `$ID: ""` (regression from 65e38cb9)

### Root Cause

`addEntityAccessRuleViaModelsdk` (commit 65e38cb9) creates new elements via gen-native:

```go
rule := genDM.NewAccessRule()      // id == "" (unassigned)
genMA := genDM.NewMemberAccess()   // id == "" (unassigned)
```

The encoder (`modelsdk/codec/encoder.go:buildDoc`) handles new elements (no raw bytes) with:

```go
doc = bson.D{
    {Key: "$ID", Value: idToBinarySubtype0(elem.ID())},
    ...
}
```

`idToBinarySubtype0("")` short-circuits and returns `""`:

```go
func idToBinarySubtype0(id element.ID) any {
    if id == "" {
        return id   // BUG: writes empty string to BSON
    }
    return mpr.IDToBsonBinary(string(id))
}
```

Result: `{$ID: ""}` written to the `.mxunit` file. When Studio Pro / mx check loads the
unit, `GetGuidFromBson` tries to cast the string to `byte[]` → `InvalidCastException`.

Confirmed by Python BSON scan of macnica after GRANT: found `$ID: ''` on
`AccessRules[2]` and all its `MemberAccesses` in the ContractorRegistration domain model.

### Fix Options

**Option A — Fix at `New*` call site in `addEntityAccessRuleViaModelsdk`** (preferred):

```go
rule := genDM.NewAccessRule()
rule.SetID(element.ID(mpr.GenerateID()))
// and for each MA:
genMA := genDM.NewMemberAccess()
genMA.SetID(element.ID(mpr.GenerateID()))
```

**Option B — Fix `idToBinarySubtype0` to auto-generate** (broader protection):

```go
func idToBinarySubtype0(id element.ID) any {
    if id == "" {
        return mpr.IDToBsonBinary(mpr.GenerateID())
    }
    return mpr.IDToBsonBinary(string(id))
}
```

**Recommendation: Option B** — fixes the encoder unconditionally, protects all future
`New*` call sites. Option A is a local patch that will recur whenever someone adds a
new `New*` call site without remembering to set an ID.

### Regression Test

After fix, a BSON scan of the written unit must confirm all `$ID` fields are 16-byte
Binary values, not strings. The existing `security_entity_access_gen_test.go` should
assert that the encoded `$ID` field is Binary.

---

## Layer 2 — Pre-existing Issue: CE0066 with associations

### Status

The Python BSON checker finds **0 structural issues** in macnica's current BSON. This
means the CE0066 described in `16-xpath-examples.mdl` TODO is **not currently present**
in the macnica BSON on disk. It appears only after certain GRANT sequences.

The TODO comment:
> "GRANT with associations triggers CE0066 even when all entities have access rules.
>  The association MemberAccess entries are added by ReconcileMemberAccesses but
>  MxBuild still reports the domain model as out of date."

### Hypothesis

Once Layer 1 is fixed, mx check may pass cleanly for GRANT operations that don't
involve cross-module associations. The CE0066 TODO may represent an older observation
that is no longer reproducible, or requires a specific association topology.

### Investigation Plan (if CE0066 persists after Layer 1 fix)

1. Run GRANT on an entity with cross-module associations (e.g. ContractorRegistration
   which has 5 CrossAssociations to BusinessApp_Common, Customer_Common,
   EndCustomerRegistration).
2. Run mx check. If CE0066 appears, capture the exact entity name.
3. Dump the entity's MemberAccesses BSON and compare against a Studio Pro-created
   access rule for the same entity.
4. Known suspects:
   - Association qualified name format (`moduleName + "." + assocName` vs what Studio Pro writes)
   - System associations (owner/changedBy) not detected correctly
   - Reconcile reading stale bytes after msdkWrite (unlikely for v2 — reads from disk)

---

## Implementation Plan

### Task 1: Fix `idToBinarySubtype0` (Option B)

**File:** `modelsdk/codec/encoder.go`

Change `idToBinarySubtype0` to generate a UUID when id is empty:
```go
func idToBinarySubtype0(id element.ID) any {
    if id == "" {
        return mpr.IDToBsonBinary(mpr.GenerateID())
    }
    return mpr.IDToBsonBinary(string(id))
}
```

### Task 2: Add regression test for $ID as Binary

**File:** `mdl/backend/mpr/security_entity_access_gen_test.go`

Extend existing regression test to scan all `$ID` fields in the encoded BSON and
assert none are empty strings. The test file already checks BSON shape post-encode.

### Task 3: Verify with mx check on macnica

Run GRANT on ContractorRegistration.ContractorApplication on a fresh macnica copy
then run mx check. Expected: no InvalidCastException, no CE0066.

If CE0066 appears, open Layer 2 investigation.

---

## Scope

This spec covers Layer 1 only. Layer 2 (CE0066 association reconcile) is deferred to
a follow-up investigation once Layer 1 is confirmed fixed.

## Files Changed

- `modelsdk/codec/encoder.go` — fix `idToBinarySubtype0`
- `mdl/backend/mpr/security_entity_access_gen_test.go` — regression test for Binary $ID
