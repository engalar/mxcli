# Model SDK BSON Investigation Methodology

> **Purpose:** How to use the mxcli `modelsdk/mpr` package to inspect raw BSON data in real Mendix projects.
> **Author:** claude_dev
> **Date:** 2026-06-22
> **Test project:** `/mnt/data_sdd/gh/Mx2026AIDay/Factory Management.mpr` (Mendix 11.6.6, 1724 microflows)

## Quick Start Template

Create a Go file in `/tmp/` and run with `go run`:

```go
package main

import (
    "fmt"
    "os"
    "strings"

    "go.mongodb.org/mongo-driver/v2/bson"
    "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func main() {
    path := "/path/to/YourProject.mpr"
    r, err := mpr.Open(path)
    if err != nil { panic(err) }
    defer r.Close()

    units, _ := r.ListUnits()
    for _, unit := range units {
        // Filter by type
        if !strings.HasSuffix(unit.Type, "$Microflow") {
            continue
        }
        raw, _ := r.GetRawUnitBytes(unit.ID)
        doc := bsonToMap(raw)
        name, _ := doc["Name"].(string)
        fmt.Println(name)
    }
}
```

## Step-by-step Methodology

### 1. Open the MPR File

```go
import "github.com/mendixlabs/mxcli/modelsdk/mpr"

r, err := mpr.Open(path)  // Returns *mpr.Reader (read-only)
defer r.Close()
```

**Gotcha:** `mpr.Open()` returns `*mpr.Reader`. There is also `modelsdk.Open()` which returns `*modelsdk.Model` (a different type with gen-typed accessors). For raw BSON inspection, use `mpr.Open()`.

### 2. List All Document Units

```go
units, err := r.ListUnits()  // Returns []*UnitInfo
```

`UnitInfo` has fields:
- `ID` — UUID string
- `ContainerID` — parent module UUID
- `Type` — Mendix type name like `"Microflows$Microflow"`, `"Pages$Page"`, etc.
- `ContainmentName` — always `"Documents"`

**Filtering pattern:**
```go
for _, unit := range units {
    if strings.HasSuffix(unit.Type, "$Microflow") {
        // This is a microflow
    }
    if strings.HasSuffix(unit.Type, "$Nanoflow") {
        // This is a nanoflow
    }
}
```

### 3. Read Raw BSON Bytes

```go
raw, err := r.GetRawUnitBytes(unit.ID)  // Returns []byte, error
```

**Gotcha:** `GetRawUnitBytes` is on `*mpr.Reader` (not on `*modelsdk.Model`).

### 4. Decode BSON to `map[string]any`

The mongo-driver v2 BSON library is used. The key challenge is that `bson.D` doesn't have a `.Map()` method in v2, so a recursive conversion is needed:

```go
import "go.mongodb.org/mongo-driver/v2/bson"

func bsonToMap(raw []byte) map[string]any {
    var doc bson.D
    if err := bson.Unmarshal(raw, &doc); err != nil {
        return nil
    }
    return d2m(doc)  // recursive conversion
}

func d2m(d bson.D) map[string]any {
    m := make(map[string]any, len(d))
    for _, e := range d {
        switch v := e.Value.(type) {
        case bson.D:
            m[e.Key] = d2m(v)
        case bson.A:
            a := make([]any, len(v))
            for i, x := range v {
                if d2, ok := x.(bson.D); ok {
                    a[i] = d2m(d2)
                } else {
                    a[i] = x
                }
            }
            m[e.Key] = a
        default:
            m[e.Key] = e.Value
        }
    }
    return m
}

// Convenience accessors
func getMap(m map[string]any, key string) map[string]any {
    v, _ := m[key].(map[string]any)
    return v
}

func getMapArray(m map[string]any, key string) []map[string]any {
    v, ok := m[key]
    if !ok { return nil }
    arr, _ := v.([]any)
    var result []map[string]any
    for _, item := range arr {
        if m2, ok := item.(map[string]any); ok {
            result = append(result, m2)
        }
    }
    return result
}
```

### 5. Navigate Microflow Structure

The BSON tree structure of a microflow:

```
Microflows$Microflow
├── Name: string
├── ObjectCollection (map)
│   ├── $ID: bson.Binary
│   ├── $Type: "Microflows$ObjectCollection"
│   └── Objects: []any  ← array of MicroflowObject
│       ├── [0] StartEvent (no Action)
│       │   ├── $Type: "Microflows$StartEvent"
│       │   └── RelativeMiddlePoint: string
│       ├── [1] EndEvent (no Action)
│       │   ├── $Type: "Microflows$EndEvent"
│       │   └── ReturnValue: string
│       ├── [2] VariableDeclaration (no Action)
│       │   ├── $Type: "Microflows$VariableDeclaration"
│       │   ├── Name: string
│       │   └── VariableType: ...
│       ├── [3] ActionActivity ← has Action!
│       │   ├── $Type: "Microflows$ActionActivity"
│       │   ├── Action: map ← the actual activity
│       │   │   ├── $Type: "Microflows$CreateChangeAction"
│       │   │   ├── Commit: "No" | "Yes" | "YesWithoutEvents"
│       │   │   ├── RefreshInClient: true | false
│       │   │   └── ... (activity-specific fields)
│       │   ├── Caption: string
│       │   └── RelativeMiddlePoint: string
│       └── ...
├── Flows: []any  ← SequenceFlow connections
└── MicroflowReturnType: ...
```

**Navigation code:**
```go
mf := bsonToMap(raw)
name, _ := mf["Name"].(string)

objColl := getMap(mf, "ObjectCollection")
objects := getMapArray(objColl, "Objects")

for _, obj := range objects {
    objType, _ := obj["$Type"].(string)
    if objType != "Microflows$ActionActivity" {
        continue  // Skip start/end/variable objects
    }
    act := getMap(obj, "Action")  // The actual activity definition
}
```

**Gotcha:** Activity is NOT a direct child of Objects[]. It's nested inside `Objects[].Action` (not `Objects[].Activity`). The outer object is always `$Type: "Microflows$ActionActivity"`.

### 6. Find Specific Activities

Filter by `act["$Type"]`:

```go
actType, _ := act["$Type"].(string)
switch actType {
case "Microflows$CreateChangeAction":  // CREATE object (modern)
case "Microflows$CreateObjectAction":  // CREATE object (legacy, rare)
case "Microflows$ChangeAction":        // CHANGE object
case "Microflows$CommitAction":        // COMMIT
case "Microflows$DeleteAction":        // DELETE
case "Microflows$RollbackAction":      // ROLLBACK
case "Microflows$MicroflowCallAction": // CALL MICROFLOW
case "Microflows$ShowMessageAction":   // SHOW MESSAGE
case "Microflows$CloseFormAction":     // CLOSE PAGE
case "Microflows$RetrieveAction":      // RETRIEVE
}
```

### 7. Extract Activity Fields

Common fields across activities:
- `$ID`: bson.Binary (GUID)
- `$Type`: string
- `ErrorHandlingType`: "Rollback" | "Continue" | "CustomWithRollback" | "CustomWithoutRollback"

Per-activity fields:

**CreateChangeAction / CreateObjectAction:**
- `Commit`: "No" | "Yes" | "YesWithoutEvents"
- `RefreshInClient`: true | false
- `VariableName`: string  ← **not** OutputVariableName!
- `Entity`: string (qualified name like "Module.Entity")
- `Items`: []MemberChange

**ChangeAction:**
- `Commit`: "No" | "Yes" | "YesWithoutEvents"
- `RefreshInClient`: true | false
- `ChangeVariableName`: string
- `Items`: []MemberChange

**CommitAction:**
- `WithEvents`: true | false
- `RefreshInClient`: true | false
- `CommitVariableName`: string

**DeleteAction:**
- `RefreshInClient`: true | false
- `DeleteVariableName`: string

**RollbackAction:**
- `RefreshInClient`: true | false
- `RollbackVariableName`: string

### 8. Counting and Deduplication

To count unique field combinations across activities:

```go
type key struct{ commit, refresh, events string }
counts := make(map[key]int)
// For each activity:
k := key{
    commit:  fmt.Sprintf("%v", act["Commit"]),
    refresh: fmt.Sprintf("%v", act["RefreshInClient"]),
    events:  fmt.Sprintf("%v", act["WithEvents"]),
}
counts[k]++
```

The template script `investigate_specific.go` shows the full pattern.

## Key Gotchas (From Experience)

1. **CreateChangeAction vs CreateObjectAction:** Studio Pro 11.6.6 writes `Microflows$CreateChangeAction`, NOT `Microflows$CreateObjectAction` (0 instances found). The gen type registration maps both to the same `CreateObjectAction` struct.

2. **Naming discrepancy:** `CreateChangeAction` stores the variable name in BSON field `VariableName` but the gen accessor is `OutputVariableName()`. The executor code already handles this mapping.

3. **Activity nesting:** The activity definition is in `Objects[].Action`, not `Objects[].Activity`. The wrapper is always `$Type: "Microflows$ActionActivity"`.

4. **MemberChange mapping:** When decoding BSON `memberChanges`, each item has:
   - `Type`: "Set"
   - `Value`: the expression string
   - Association or Attribute qualified name (not both)

5. **V2 BSON vs V1 BSON:** V2 format (`>= Mendix 10.18`) stores units as individual `.mxunit` files in `mprcontents/`. V1 stores BSON inline in SQLite. The `mpr.Reader` handles both transparently.

6. **bson.D vs map[string]any:** The mongo-driver v2 uses `bson.D` (ordered document) instead of `map[string]any`. Always convert with `d2m()`.

7. **bson.Binary vs string GUIDs:** `$ID` fields are `bson.Binary` (subtype 3 = UUID), not strings. To convert: `uuid := hex.EncodeToString(id.Data)`.

8. **Unit filtering:** Always check `$Type` suffix with `strings.HasSuffix`, not `==`, because unit types can have module-specific prefixes.

## Reusable Script Templates

The following template scripts were used in this investigation (located in `/tmp/` during the session):

| Script | Purpose |
|--------|---------|
| `investigate_mf.go` | Extract all Create/Change/Commit/Delete/Rollback activities with their fields |
| `investigate_specific.go` | Count unique Commit+RefreshInClient+WithEvents combinations per activity type |
| `dump_mf.go` | Dump full BSON structure of one microflow for structural analysis |
| `count_mf.go` | Count total units by type |
| `debug_mf.go` | Debug BSON decoding and explore key structure |

## References

- Gen types: `modelsdk/gen/microflows/types.go` (17,705 lines, all microflow action types)
- BSON descriptors: `modelsdk/gen/microflows/descriptors.go` (BSON key → Go field mapping)
- Codec registry: `modelsdk/gen/microflows/types.go` (line ~17000+, type registration)
- Executor action builders: `mdl/executor/flowbuilder_actions_v2.go`
- Executor action formatters: `mdl/executor/microflows_format_action_v2.go`
