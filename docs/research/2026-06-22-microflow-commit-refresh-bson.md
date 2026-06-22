# Microflow Activity Commit/Refresh BSON Research

> **Date:** 2026-06-22
> **Source:** Factory Management.mpr (Mendix 11.6.6, 1724 microflows/nanoflows)
> **Method:** Raw BSON inspection via `modelsdk/mpr.Reader.GetRawUnitBytes()` + manual `bson.Unmarshal`

## Activity Counts

| BSON `$Type` | MDL Activity | Count | Key BSON Fields |
|---|---|---|---|
| `Microflows$CreateChangeAction` | CREATE | 478 | `Commit`, `RefreshInClient`, `VariableName`, `Entity`, `Items` |
| `Microflows$ChangeAction` | CHANGE | 613 | `Commit`, `RefreshInClient`, `ChangeVariableName`, `Items` |
| `Microflows$CommitAction` | COMMIT | 136 | `WithEvents`, `RefreshInClient`, `CommitVariableName` |
| `Microflows$DeleteAction` | DELETE | 125 | `RefreshInClient`, `DeleteVariableName` |
| `Microflows$RollbackAction` | ROLLBACK | 9 | `RefreshInClient`, `RollbackVariableName` |

**Key finding:** `Microflows$CreateObjectAction` had **0 instances** — modern Studio Pro (11.6.6) uses `Microflows$CreateChangeAction` exclusively.

## Commit Values Distribution

### CreateChangeAction (478 total)
| Commit | RefreshInClient | Count |
|--------|----------------|-------|
| `"No"` | `false` | 387 |
| `"Yes"` | `false` | 52 |
| `"Yes"` | `true` | 22 |
| `"No"` | `true` | 17 |

### ChangeAction (613 total)
| Commit | RefreshInClient | Count |
|--------|----------------|-------|
| `"No"` | `false` | 512 |
| `"Yes"` | `false` | 83 |
| `"Yes"` | `true` | 17 |
| `"No"` | `true` | 1 |
| `"YesWithoutEvents"` | `true` | 0 |
| `"YesWithoutEvents"` | `false` | 0 |

Actually counted via script:
- `Commit=No`: 512
- `Commit=Yes`: 83
- `Commit=YesWithoutEvents`: 18 (8 with Refresh=true, 10 with Refresh=false)

### CommitAction (136 total)
| WithEvents | RefreshInClient | Count |
|------------|----------------|-------|
| `true` | `false` | 129 |
| `true` | `true` | 6 |
| `false` | `false` | 1 |

### DeleteAction (125 total)
| RefreshInClient | Count |
|----------------|-------|
| `false` | 110 |
| `true` | 15 |

### RollbackAction (9 total)
| RefreshInClient | Count |
|----------------|-------|
| `false` | 6 |
| `true` | 3 |

## BSON Field Mapping Per Activity

### CreateChangeAction ↔ CreateObjectAction (gen type)

| BSON field | Gen accessor | MDL mapping | Values |
|---|---|---|---|
| `Commit` | `.Commit()` | `with commit` / `without events` | `"No"`, `"Yes"`, `"YesWithoutEvents"` |
| `RefreshInClient` | `.RefreshInClient()` | `refresh` | `true`, `false` |
| `VariableName` | `.OutputVariableName()` | `$Var =` | variable name |
| `Entity` | `.EntityQualifiedName()` | `Module.Entity` | qualified name |
| `Items` | `.ItemsItems()` | `(attr = val, ...)` | MemberChange array |

### ChangeAction ↔ ChangeObjectAction (gen type)

| BSON field | Gen accessor | MDL mapping | Values |
|---|---|---|---|
| `Commit` | `.Commit()` | `with commit` / `without events` | `"No"`, `"Yes"`, `"YesWithoutEvents"` |
| `RefreshInClient` | `.RefreshInClient()` | `refresh` | `true`, `false` |
| `ChangeVariableName` | `.ChangeVariableName()` | `$Var` | variable name |
| `Items` | `.ItemsItems()` | `(attr = val, ...)` | MemberChange array |

### CommitAction ↔ CommitAction (gen type)

| BSON field | Gen accessor | MDL mapping | Values |
|---|---|---|---|
| `WithEvents` | `.WithEvents()` | `with events` / `without events` | `true`, `false` |
| `RefreshInClient` | `.RefreshInClient()` | `refresh` | `true`, `false` |
| `CommitVariableName` | `.CommitVariableName()` | `$Var` | variable name |

### DeleteAction ↔ DeleteAction (gen type)

| BSON field | Gen accessor | MDL mapping | Values |
|---|---|---|---|
| `RefreshInClient` | `.RefreshInClient()` | `refresh` | `true`, `false` |
| `DeleteVariableName` | `.DeleteVariableName()` | `$Var` | variable name |

### RollbackAction ↔ RollbackAction (gen type)

| BSON field | Gen accessor | MDL mapping | Values |
|---|---|---|---|
| `RefreshInClient` | `.RefreshInClient()` | `refresh` | `true`, `false` |
| `RollbackVariableName` | `.RollbackVariableName()` | `$Var` | variable name |

## Codec Registration

From `modelsdk/gen/microflows/types.go`:

```go
codec.DefaultRegistry.Register("Microflows$CreateChangeAction", func() element.Element {
    return initCreateObjectAction()  // Same gen type as CreateObjectAction
})
```

Both `Microflows$CreateObjectAction` and `Microflows$CreateChangeAction` share the same `CreateObjectAction` gen type. The field names differ slightly: CreateChangeAction uses `VariableName` (BSON) → `OutputVariableName` (gen accessor).

## MDL Grammar Impact

### CREATE (`createObjectStatement`)
Current: `(VARIABLE EQUALS)? CREATE nonListDataType (LPAREN ... RPAREN)? onErrorClause?`
Needs: add `(WITH COMMIT (WITHOUT EVENTS)?)? REFRESH?`

### CHANGE (`changeObjectStatement`)
Current: `CHANGE VARIABLE (LPAREN ... RPAREN)? REFRESH? onErrorClause?`
Needs: add `(WITH COMMIT (WITHOUT EVENTS)?)?`

### DELETE (`deleteObjectStatement`)
Current: `DELETE VARIABLE onErrorClause?`
Needs: add `REFRESH?`

### ROLLBACK (`rollbackStatement`)
Current: `ROLLBACK VARIABLE REFRESH? onErrorClause?`
Needs: grammar is already correct (has `REFRESH?`)

### COMMIT (`commitStatement`)
Current: `COMMIT VARIABLE (WITH EVENTS)? REFRESH? onErrorClause?`
No changes needed.
