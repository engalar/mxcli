# bsoncompare modelsdk API Migration

**Date:** 2026-06-26
**Status:** Draft
**Author:** Brainstorming session

## 1. Motivation

bsoncompare currently operates directly on `go.mongodb.org/mongo-driver/v2/bson` types
(`bson.Raw`, `bson.D`, `bson.A`, `bson.RawValue`, `RawElement`, `bson.Type`).
Every BSON decode/iterate/compare logic is hand-rolled at the raw byte level.

The goal: bsoncompare should only use modelsdk's public API
(`element.Element`, `property.Property`, `codec.Decoder`, `mpr.Reader`)
and never call the `bson` package directly.

## 2. Design

### 2.1. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  bsoncompare                                                │
│                                                             │
│  ReadAllUnits → stores []byte (not bson.Raw) + ContentHash  │
│       ↓                                                     │
│  IDMap: mpr.Reader.ListUnitIdentities() → []UnitIdentity    │
│       ↓                                                     │
│  Compare:                                                   │
│    aPath==bPath → empty   (early return, unchanged)         │
│    ContentHash==match → skip  (unchanged)                   │
│    ContentHash!=match →                                     │
│      codec.Decoder.DecodeBytes(raw) → element.Element       │
│      compareProperties(Element.Properties()) → []FieldDiff   │
└─────────────────────────────────────────────────────────────┘
         │                 │
         ▼                 ▼
  modelsdk/codec     modelsdk/mpr
  (Decoder, Store)   (Reader)
         │                 │
         ▼                 ▼
  go.mongodb.org/mongo-driver/v2/bson
  (only inside modelsdk — bsoncompare never imports this)
```

### 2.2. Reading Units

**Current (`mprreader.go`):** Stores `bson.Raw` + `bson.D` (lazy-decoded).

**New:** Stores `Raw []byte` (plain Go type, no bson).

```go
type UnitDoc struct {
    QualifiedName string
    UnitType      string
    Raw           []byte       // raw BSON bytes (plain Go, not bson.Raw)
    ContentHash   uint64       // FNV-1a of raw BSON (unchanged)
}
```

`ReadAllUnits` passes `info.Contents` directly to FNV-1a and stores as `[]byte`.
The `Doc bson.D` field and `Decode()` method are removed — `codec.Decoder` replaces them.

### 2.3. ID Resolution

**Current (`idmap.go`):** `collectIDsRaw()` iterates `bson.Raw.Elements()` across all units.

**New:** modelsdk adds `Reader.ListUnitIdentities()`:

```go
// In modelsdk/mpr/reader.go
type UnitIdentity struct {
    ID   string  // hex-encoded UUID from $ID
    Name string  // Name field
    Type string  // $Type field
}

func (r *Reader) ListUnitIdentities() ([]UnitIdentity, error)
```

This method does the minimum BSON parse per unit (extract 3 top-level fields),
does NOT create Element/Property objects. Comparable to current `collectIDsRaw`.

bsoncompare then builds `IDMap` from `[]UnitIdentity`:

```go
func BuildIDMap(idents []UnitIdentity) IDMap {
    m := make(IDMap, len(idents)*2)
    for _, id := range idents {
        m[id.ID] = makeLabel(id.Type, id.Name, "")
    }
    return m
}
```

### 2.4. Comparison Algorithm

When two units have mismatched `ContentHash`, bsoncompare decodes both via modelsdk
and compares their Properties() trees:

```go
func compareElements(path string, golden, actual element.Element,
    idMap IDMap, opts Options) []FieldDiff {
    // Return empty if both are nil
    if golden == nil && actual == nil { return nil }
    // Handle add/remove
    if golden == nil { return []FieldDiff{{Path: path, Kind: DiffAdded, Actual: formatElem(actual)}} }
    if actual == nil { return []FieldDiff{{Path: path, Kind: DiffRemoved, Golden: formatElem(golden)}} }
    // Unknown type (bare *Base) — limited comparison
    if isBareBase(golden) || isBareBase(actual) {
        return compareBareBase(path, golden, actual)
    }
    // Type name mismatch
    if golden.TypeName() != actual.TypeName() {
        return []FieldDiff{{Path: path + ".$Type",
            Golden: golden.TypeName(), Actual: actual.TypeName(), Kind: DiffChanged}}
    }
    // Nested property comparison
    return compareProperties(path, golden.Properties(), actual.Properties(), idMap, opts)
}
```

#### Property comparison by type

```go
func compareProperty(path string, gp, ap element.Property,
    idMap IDMap, opts Options) []FieldDiff {

    if shouldIgnore(gp.Name(), opts) { return nil }

    // ChildProperty (Part)
    if gc, ok := gp.(element.ChildProperty); ok {
        ac := ap.(element.ChildProperty)
        return compareElements(path, gc.ChildElement(), ac.ChildElement(), idMap, opts)
    }

    // ChildListProperty (PartList)
    if gl, ok := gp.(element.ChildListProperty); ok {
        al := ap.(element.ChildListProperty)
        return compareChildList(path, gl.ChildElements(), al.ChildElements(), idMap, opts)
    }

    // WritableProperty — typed comparison
    gw := gp.(element.WritableProperty)
    aw := ap.(element.WritableProperty)
    return compareBSONValue(path, gw.BSONValue(), aw.BSONValue(), idMap, opts)
}
```

#### BSONValue type dispatch

```go
func compareBSONValue(path string, g, a any, idMap IDMap, opts Options) []FieldDiff {
    switch gv := g.(type) {
    case string:
        av, _ := a.(string)
        if gv != av { return diffChanged(path, gv, av) }
    case int32:
        av, _ := a.(int32)
        if gv != av { return diffChanged(path, gv, av) }
    case bool:
        av, _ := a.(bool)
        if gv != av { return diffChanged(path, gv, av) }
    case float64:
        av, _ := a.(float64)
        if gv != av { return diffChanged(path, gv, av) }
    case bson.Binary:
        return compareBinary(path, gv, a, idMap)
    case element.ID:
        av, _ := a.(element.ID)
        if gv != av { return diffChanged(path, string(gv), string(av)) }
    case []string:             // EnumList
        ga := sortedCopy(gv)
        aa := sortedCopy(a.([]string))
        return compareStringSlice(path, ga, aa)
    case []any:                // ByNameRefList (versioned)
        gItems := stripVersion(gv)
        aItems := stripVersion(a.([]any))
        return compareStringSlice(path, toStrings(gItems), toStrings(aItems))
    default:
        return nil  // unknown type, skip
    }
}
```

#### PartList alignment

```go
func compareChildList(path string, golden, actual []element.Element,
    idMap IDMap, opts Options) []FieldDiff {
    switch {
    case allHaveName(golden) || allHaveName(actual):
        return compareByName(path, golden, actual, idMap, opts)
    case allAreRefs(golden):
        return compareSetRefs(path, golden, actual, idMap)
    default:
        return compareByPosition(path, golden, actual, idMap, opts)
    }
}
```

The `allAreRefs` check uses `DescRegistry` to determine if a child element
looks like a reference (has only ID/Name/Type metadata, no typed properties).

### 2.5. Unknown Types (bare `*element.Base`)

Detection:

```go
func isBareBase(elem element.Element) bool {
    _, ok := elem.(*element.Base)
    return ok
}
```

Comparison — only three fields:

```go
func compareBareBase(path string, golden, actual element.Element) []FieldDiff {
    var diffs []FieldDiff
    if golden.TypeName() != actual.TypeName() {
        diffs = append(diffs, FieldDiff{Path: path + ".$Type",
            Golden: golden.TypeName(), Actual: actual.TypeName(), Kind: DiffChanged})
    }
    // NameValue() reads from raw bytes via modelsdk's Base.NameValue()
    gn := golden.NameValue()
    an := actual.NameValue()
    if gn != an {
        diffs = append(diffs, FieldDiff{Path: path + ".Name",
            Golden: gn, Actual: an, Kind: DiffChanged})
    }
    return diffs
}
```

All other fields (unknown structure) are skipped per user requirement.

### 2.6. Ignored Fields

Same `shouldIgnore()` logic as current, but operates on `property.Name()`
instead of raw BSON key strings. The property names correspond to BSON field names
via `PropDesc.BSONKey`.

`$ID` is automatically absent from `Properties()` (ID is Element-level, not a property),
so the `builtinIgnore` entry for `$ID` becomes a no-op (kept for safety).

### 2.7. BSON 类型泄漏处理

`Property.BSONValue()` 返回 `any`，其具体类型可能是：
- `bson.Binary`（来自 `BinaryPrimitive`、`BinaryUUIDPrimitive`）
- `bson.A`（来自 `StringListPrimitive`）

要在 bsoncompare 里做类型断言 `case bson.Binary`，需要 import `go.mongodb.org/mongo-driver/v2/bson`。

**结论：bsoncompare 可以 import bson 包用于 TYPE REFERENCE（类型断言），
但绝不可以调用 bson 的任何 decode/encode/iterate 函数。**

理由：
1. `bson.Binary` 和 `bson.A` 是 modelsdk 公开 API 的返回值类型
2. 不对它们做类型断言，就无法比较值
3. 这是 "receive from modelsdk" 而非 "operate on bson"

```go
// 允许：类型断言 bson 返回值
case bson.Binary:
    // 只读 .Subtype 和 .Data 字段
    // 不调任何 bson 函数

// 允许：类型断言 bson.A（本质是 []any）
case bson.A:
    // 当 []any 用

// 不允许：
// bson.Unmarshal(raw, &doc)
// raw.Elements()
// bson.Raw(bytes)
```

### 2.8. `$ID` 与 Properties() 的关系

`$ID` 不出现在 `Properties()` 中——ID 是 `Element.ID()` 级别的属性。
因此 `shouldIgnore` 中 `case "$ID"` 分支在新路径中不会命中，但仍保留作为
安全网（兼容未来可能的 codegen 变更）。

## 3. Modelsdk Extensions

| Extension | Package | Signature | Reason |
|-----------|---------|-----------|--------|
| `ListUnitIdentities` | `mpr/reader.go` | `func (r *Reader) ListUnitIdentities() ([]UnitIdentity, error)` | IDMap build, avoids full Element creation |
| `UnitIdentity` struct | `mpr/types.go` | `type UnitIdentity struct { ID, Name, Type string }` | Return type for above |
| `DecodeBytes` | `codec/decoder.go` | `func (d *Decoder) DecodeBytes(raw []byte) (element.Element, error)` | Wraps `bson.Raw` conversion inside modelsdk so bsoncompare never needs it |
| `ReadAllElements` | `codec/store.go` | `func (s *Store) ReadAllElements() ([]element.Element, error)` | Bulk typed read for future use; optional for bsoncompare |

## 4. Performance Analysis

### 4.1. ContentHash fast path (unchanged)

- `aPath == bPath` → early return (same)
- `au.ContentHash == bu.ContentHash` → skip (same)
- For self-compare and mutation tests where most units unchanged: **zero regression**

### 4.2. IDMap build (new overhead)

- Current: `collectIDsRaw` does `Raw.Elements()` per unit
- New: `ListUnitIdentities()` does minimal BSON parse per unit (read $ID, Name, $Type)
- Both are O(n) per unit with similar per-unit cost
- But old path iterates all elements recursively (into sub-docs), new path only reads top-level fields
- **Net: slightly less work**

### 4.3. Changed-unit comparison (new path)

- Current: `bson.Unmarshal` → `bson.D` → `Normalize` (map[string]any) → `diffMaps`
- New: `codec.Decoder.DecodeBytes` → `Element` → `compareProperties`
- The Decoder internally does a similar BSON walk to `Unmarshal`, but:
  1. Properties are lazy (`Init` stores raw pointer, doesn't decode values)
  2. `BSONValue()` triggers single-field decode on demand
  3. No intermediate `map[string]any` allocation
- **Net: less alloc, same or less CPU for changed units**

### 4.4. Allocation summary

| Phase | Current alloc | New alloc | Delta |
|-------|--------------|-----------|-------|
| ReadAllUnits | bson.Raw (alias) | `[]byte` | same |
| IDMap build (`collectIDsRaw`) | `[]RawElement` per doc | `[]UnitIdentity` per doc | ~same |
| Changed unit comparison | `bson.D` (44% mem) + `map[string]any` (13%) | `Element` struct + property holders | -44% mem |
| Total test suite (corpus-b) | 5.21GB | ~3GB (est) | -40%+ |

## 5. Edge Cases

### 5.1. Versioned arrays (PartList)

The Encoder's `ChildListProperty` path handles version markers. `ChildElements()`
returns children without version prefix (stored separately in `PartList.versionMarker`).
No special handling needed in compare — the version is transparent.

### 5.2. ByNameRefList (e.g., AllowedModuleRoles)

`BSONValue()` returns `[]any{version, qn1, qn2, ...}` (versioned array).
Compare strips the first `int32` element (version) via `stripVersion()`,
then compares remaining elements as a set of strings.

### 5.3. Binary UUID comparison

`BinaryUUIDPrimitive.BSONValue()` returns `bson.Binary` with MS GUID byte swap.
The 16-byte data is resolved via IDMap (same as current `normalizeVal` for 16-byte binary).
Other binary lengths compared by length only.

### 5.4. Decode failure

If `codec.Decoder.DecodeBytes(raw)` fails for a unit (malformed BSON),
the unit is skipped in comparison (same behavior as current `continue` on unmarshal error).

### 5.5. Part version mismatch (V2 vs V3)

`PartList` has `versionMarker` stored internally. When comparing two PartLists,
if the golden and actual have different version markers (e.g., 2 vs 3),
the children are still compared — the version is metadata, not payload.
If the version difference matters, it would show up as an extra field diff.
Current behavior: version prefix is stripped by `normalizeArray` before comparison.
Same semantic effect.

## 6. Future Considerations

- The `DescRegistry` is available for field-level metadata queries but currently
  not required by the comparison path. If needed later (e.g., for extra-field
  safety nets), it's already wired.
- `Store.ReadAllElements()` is optional — bsoncompare can use `mpr.Reader`
  directly + `codec.Decoder.DecodeBytes` for now.
- When codegen coverage reaches 100%, the `isBareBase` fallback path can be removed.
