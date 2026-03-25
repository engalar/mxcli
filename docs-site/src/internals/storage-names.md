# Storage Names vs Qualified Names

Mendix uses different "storage names" in BSON `$Type` fields than the "qualified names" shown in the TypeScript SDK documentation. Using the wrong name causes a `TypeCacheUnknownTypeException` when opening the project in Studio Pro.

## The Problem

When serializing model elements to BSON, the `$Type` field must use the **storage name**, not the qualified name from the SDK documentation. These names differ in ways that are not always predictable.

## Domain Prefix Mapping

The most systematic difference is the domain prefix used for page-related types:

| API Prefix | Storage Prefix | Domain |
|------------|----------------|--------|
| `Pages$` | `Forms$` | Page widgets |
| `Microflows$` | `Microflows$` | Microflow elements (same) |
| `DomainModels$` | `DomainModels$` | Domain model elements (same) |
| `Texts$` | `Texts$` | Text/translation elements (same) |
| `DataTypes$` | `DataTypes$` | Data type definitions (same) |
| `CustomWidgets$` | `CustomWidgets$` | Pluggable widgets (same) |

The `Pages$` to `Forms$` renaming reflects Mendix's historical terminology -- "Form" was the original term for "Page".

## Microflow Action Name Mapping

Several microflow actions have storage names that differ significantly from their qualified names:

| Qualified Name (SDK/docs) | Storage Name (BSON `$Type`) | Note |
|---------------------------|---------------------------|------|
| `CreateObjectAction` | `CreateChangeAction` | |
| `ChangeObjectAction` | `ChangeAction` | |
| `DeleteObjectAction` | `DeleteAction` | |
| `CommitObjectsAction` | `CommitAction` | |
| `RollbackObjectAction` | `RollbackAction` | |
| `AggregateListAction` | `AggregateAction` | |
| `ListOperationAction` | `ListOperationsAction` | Note the plural |
| `ShowPageAction` | `ShowFormAction` | "Form" was original for "Page" |
| `ClosePageAction` | `CloseFormAction` | "Form" was original for "Page" |

## Client Action Name Mapping

Page client actions also have storage name differences:

| Incorrect (will fail) | Correct Storage Name |
|-----------------------|---------------------|
| `Forms$NoClientAction` | `Forms$NoAction` |
| `Forms$PageClientAction` | `Forms$FormAction` |
| `Forms$MicroflowClientAction` | `Forms$MicroflowAction` |
| `Pages$DivContainer` | `Forms$DivContainer` |
| `Pages$ActionButton` | `Forms$ActionButton` |

## How to Verify Storage Names

When adding new types, always verify the storage name by:

1. **Examining existing MPR files** with the `mx` tool or an SQLite browser
2. **Checking the reflection data** in `reference/mendixmodellib/reflection-data/`
3. **Looking at the parser cases** in `sdk/mpr/parser_microflow.go`

### Querying Reflection Data

Use this Python snippet to check widget default settings:

```python
import json

with open('reference/mendixmodellib/reflection-data/11.0.0-structures.json') as f:
    data = json.load(f)

# Find widget by API name
widget = data.get('Pages$DivContainer', {})
print('Storage name:', widget.get('storageName'))
print('Defaults:', json.dumps(widget.get('defaultSettings', {}), indent=2))

# Search by storage name
for key, val in data.items():
    if val.get('storageName') == 'Forms$NoAction':
        print(f'{key}: {val.get("defaultSettings")}')
```

## Association Parent/Child Pointer Semantics

Mendix BSON uses **inverted naming** for association pointers, which is counter-intuitive:

| BSON Field | Points To | MDL Keyword |
|------------|-----------|-------------|
| `ParentPointer` | **FROM** entity (FK owner) | `FROM Module.Child` |
| `ChildPointer` | **TO** entity (referenced) | `TO Module.Parent` |

For example, `CREATE ASSOCIATION Mod.Child_Parent FROM Mod.Child TO Mod.Parent` stores:
- `ParentPointer = Child.$ID` (the FROM entity owns the foreign key)
- `ChildPointer = Parent.$ID` (the TO entity is being referenced)

This affects **entity access rules**: MemberAccess entries for associations must only be added to the **FROM** entity (the one stored in `ParentPointer`). Adding them to the TO entity triggers CE0066 "Entity access is out of date".

## Key Takeaway

When unsure about the correct BSON structure for a new feature, create a working example in Mendix Studio Pro and compare the generated BSON against a known-good reference. The reflection data at `reference/mendixmodellib/reflection-data/` is the definitive source for storage names and default values.
