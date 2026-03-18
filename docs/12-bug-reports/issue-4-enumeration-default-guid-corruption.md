# Issue #4: MPR Data Corruption After Attribute Drop/Add with Enumeration Default

**Source**: https://github.com/mendixlabs/mxcli/issues/4
**Severity**: Critical (data loss)
**Date**: 2026-03-18

## Symptoms

After dropping and re-adding an entity attribute with an enumeration default value, the MPR becomes corrupted. MxBuild/Studio Pro fails with:

```
System.Collections.Generic.KeyNotFoundException: The given key '3622ee3a-8d34-4495-9788-6e6462f0ab3c' was not present in the dictionary
```

## Reproduction

```mdl
ALTER ENTITY MaisonElegance.FormSubmission DROP ATTRIBUTE SubmissionStatus;
ALTER ENTITY MaisonElegance.FormSubmission ADD ATTRIBUTE SubmissionStatus:
  Enumeration(MaisonElegance.FormSubmissionStatus) DEFAULT
  MaisonElegance.FormSubmissionStatus.StatusNew;
```

## Root Cause

A two-part synchronization failure between the BSON parser and writer for the `$ID` field inside `StoredValue` (and `CalculatedValue`, `OqlViewValue`) objects.

### Part 1: Parser drops the `$ID`

**File**: `sdk/mpr/parser_domainmodel.go`, function `parseAttributeValue()` (lines 224-249)

The parser extracts `$Type` and `DefaultValue` from BSON but never reads `$ID`. The `AttributeValue` struct embeds `BaseElement` (which has an `ID` field via `model.BaseElement`), but the parser never populates it:

```go
case "DomainModels$StoredValue":
    return &domainmodel.AttributeValue{
        Type:         "StoredValue",
        DefaultValue: defaultValue,
        // $ID is never extracted from raw — lost here
    }
```

### Part 2: Writer always generates a new GUID

**File**: `sdk/mpr/writer_domainmodel.go`, function `serializeAttribute()` (lines 823-827)

The writer always calls `generateUUID()` for the StoredValue's `$ID`, never checking if an existing ID should be preserved:

```go
valueDoc = bson.D{
    {Key: "$ID", Value: idToBsonBinary(generateUUID())},  // always new
    {Key: "$Type", Value: "DomainModels$StoredValue"},
    {Key: "DefaultValue", Value: defaultValue},
}
```

### The Corruption Flow

1. **Read MPR**: StoredValue has `$ID: "3622ee3a-..."` in BSON
2. **Parser drops `$ID`**: `AttributeValue.ID` is empty string
3. **DROP ATTRIBUTE**: Removes the attribute from the entity's attribute list, but the old GUID remains referenced elsewhere in the MPR (GUID registry, cross-references)
4. **ADD ATTRIBUTE**: Writer generates a brand new UUID for the StoredValue
5. **Old GUID becomes orphaned**: Studio Pro resolves all GUID references, finds `"3622ee3a-..."` pointing to nothing → `KeyNotFoundException`

## Impact Scope

- Affects **all** attribute value types, not just enumerations (StoredValue, CalculatedValue, OqlViewValue)
- Any DROP + ADD attribute cycle on an attribute with a default value will produce orphaned GUIDs
- Multiple drop/add cycles compound the problem with more orphaned GUIDs
- Corrupted MPR cannot be opened in Studio Pro or built with MxBuild
- No workaround exists beyond reverting to backups

## Fix Required

1. **`sdk/mpr/parser_domainmodel.go`** — Extract `$ID` from the BSON `raw` map and set it on `AttributeValue.BaseElement.ID` for all value types (StoredValue, CalculatedValue, OqlViewValue)
2. **`sdk/mpr/writer_domainmodel.go`** — If `a.Value.ID` is non-empty, preserve it instead of generating a new UUID. Only generate a new UUID for truly new attributes (where `ID` is empty).
