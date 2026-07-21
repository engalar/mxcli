# Debug BSON Serialization Issues

This skill provides a systematic workflow for debugging BSON serialization errors when programmatically creating Mendix pages and widgets.

## When to Use This Skill

Use when encountering:
- **Studio Pro crash** `System.InvalidOperationException: Sequence contains no matching element` at `MprObject..ctor` or `MprProperty..ctor`
- **Studio Pro crash on open** with `RevStatusCache.CreateDeleteStatusItem` in stack trace
- **`mx diff` crash** with "Sequence contains no matching element"
- **CE1613** "The selected attribute/enumeration no longer exists"
- **CE0115** "The arguments that are passed to page X do not match the expected parameters"
- **CE0463** "The definition of this widget has changed"
- **CE0642** "Property X is required"
- **CE0091** validation errors on widget properties
- Any `mx check` error related to widget structure after creating pages via MDL

## Prerequisites

- A Mendix test project (`.mpr` file)
- The `mx` tool at `reference/mxbuild/modeler/mx`
- Python 3 with `pymongo` (for BSON inspection): `pip install pymongo`

## Workflow

### Step 1: Reproduce the Error

```bash
# create a page via MDL
./bin/mxcli exec script.mdl -p /path/to/app.mpr

# run mx check to get the error
reference/mxbuild/modeler/mx check /path/to/app.mpr
```

Note the exact error code (CE0463, CE0642, etc.) and which widget triggers it.

### Step 2: Get a Known-Good Reference

Create a working example in Studio Pro and update it:

```bash
# Convert project to latest format and update widget definitions
reference/mxbuild/modeler/mx convert -p /path/to/app.mpr
reference/mxbuild/modeler/mx update-widgets /path/to/app.mpr
```

Then extract the widget's BSON to compare against your generated output.

### Step 3: Extract and Compare BSON

Use the debug dump tool or Python to compare working vs broken widgets:

```python
import bson
import sqlite3
import json

conn = sqlite3.connect('/path/to/app.mpr')
cursor = conn.cursor()

# find the document containing the widget
cursor.execute("select UnitData from Unit$ where ContainmentName = 'Document' and Name = ?", ('PageName',))
row = cursor.fetchone()
doc = bson.decode(row[0])

# Pretty-print to find the widget
print(json.dumps(doc, indent=2, default=str))
```

### Step 4: Check the Widget Package (.mpk)

Extract the widget's mpk to understand its schema and mode-dependent rules:

```bash
# find the mpk in the project's widgets folder
ls /path/to/project/widgets/*.mpk

# Extract (mpk is a ZIP archive)
mkdir /tmp/mpk-widget
cd /tmp/mpk-widget && unzip /path/to/project/widgets/com.mendix.widget.web.Datagrid.mpk
```

Key files inside the mpk:
- **`{widget}.xml`** — Property schema: types, defaults, enumerations, nested objects
- **`{widget}.editorConfig.js`** — Mode-dependent visibility rules (which properties hide/show based on other values)
- **`package.xml`** — Package version metadata

### Step 5: Read editorConfig.js for Mode Rules

The `editorConfig.js` defines which properties are hidden based on other property values. Look for patterns like:

```javascript
// hidePropertyIn(props, values, "listName", index, "propName")
// hideNestedPropertiesIn(props, values, "listName", index, ["prop1", "prop2"])
```

These rules define the **property state matrix** — when a mode-switching property (like `showContentAs`) changes, certain other properties must be in the correct hidden/visible state.

### Step 6: Isolation Testing

Use binary search to find the exact property causing the error:

1. **Clone all properties from template** (no modifications) → should PASS
2. **Change one property at a time** → find which change causes FAIL
3. **Check mode-dependent properties** → verify hidden properties have appropriate values

```python
# Mutation test: change a single property on a known-good widget
import bson

# read the working widget BSON
with open('working-widget.bson', 'rb') as f:
    doc = bson.decode(f.read())

# change only one property value
# ... modify the specific property ...

# Re-encode and write back
with open('test-widget.bson', 'wb') as f:
    f.write(bson.encode(doc))

# then insert back into the MPR and run mx check
```

### Step 7: Extract Fresh Templates

If the widget template is outdated, extract a fresh one:

```bash
# First update the test project's widgets
reference/mxbuild/modeler/mx convert -p /path/to/app.mpr
reference/mxbuild/modeler/mx update-widgets /path/to/app.mpr

# then extract using mxcli
./bin/mxcli extract-templates -p /path/to/app.mpr -widget "com.mendix.widget.web.datagrid.DataGrid2" -o /tmp/template.json
```

Templates must include both `type` (PropertyTypes schema) AND `object` (default WidgetObject).

## Critical: mx check vs mx diff vs Studio Pro Tolerance Levels

Three Mendix tools parse the same BSON but with **different strictness levels**:

| Tool | Extra properties | Missing properties | When used |
|------|-----------------|-------------------|-----------|
| `mx check` | **Tolerant** — silently skips | **Tolerant** — uses defaults | Validation |
| `mx dump-mpr` | **Tolerant** — silently skips | **Tolerant** — uses defaults | Export |
| `mx diff` | **STRICT** — crashes | **STRICT** — crashes | Version control diff |
| Studio Pro `RevStatusCache` | **STRICT** — crashes | **STRICT** — crashes | Opening project with uncommitted changes |

**Key insight**: `mx check` passing does NOT mean Studio Pro can open the project. Always verify with `mx diff` when testing BSON writers.

**Why it matters**: Studio Pro calls `mx diff` internally during `RevStatusCache.DoRefresh()` to compare the working copy against git HEAD. Any BSON property mismatch → crash on open.

## Diagnostic: mx diff as Crash Reproducer

### Step 0: Reproduce Outside Studio Pro

```bash
# Self-diff: is the project's BSON clean?
mx diff project.mpr project.mpr output.mpr
# success = all BSON matches schema. Crash = some mxunit has bad properties.

# cross-diff: compare baseline vs modified
# 1. Extract baseline from git
mkdir /tmp/baseline && cd project-dir && git archive head -- *.mpr mprcontents/ | tar -x -C /tmp/baseline/
# 2. run diff (file names must match!)
mx diff /tmp/baseline/App.mpr ./App.mpr /tmp/diff-output.mpr
```

### Interpreting mx diff Output

**Detailed error** (when both sides have same-ID objects):
```
objects with ID b6fc893f-... of type settings$ServerConfiguration do not have the same properties.
baseNames = ApplicationRootUrl, ConstantValues, CustomSettings, ..., Tracing;
newNames = ApplicationRootUrl, ConstantValues, ...
```
→ Compare the two lists. Properties in `baseNames` but not `newNames` = missing in our BSON. Properties in `newNames` but not `baseNames` = extra in our BSON.

**Generic error** (when objects don't have matching IDs — e.g., deleted units):
```
Sequence contains no matching element
```
→ No detail given. Use the property comparison tools below.

### Finding the Offending mxunit File

Write a Go tool (or use the pattern below) to compare property keys of mxcli-written files against Studio Pro-native files of the same `$type`:

```go
// Walk mprcontents/, group files by $type, compare key sets
// Files with EXTRA keys (vs native files of same type) = the crash cause
// Files with MISSING keys = also crash cause for mx diff
```

The principle: for each `$type`, ALL instances must have the **exact same set of non-$ property names**. Any deviation → crash.

### Version-Specific Properties

Some properties only exist in certain Mendix versions. Before adding a property to a BSON writer:
1. Check `reference/mendixmodellib/reflection-data/` for the property definition
2. Check `min_version` if present
3. Test with `mx diff` self-diff on the target version

**Example**: `IsReusableComponent` exists in `Projects$ModuleImpl` in newer Mendix but NOT in 11.6.4. Writing it → crash.

## Common Error Patterns

### Studio Pro Crash: InvalidOperationException in MprObject..ctor (RevStatusCache)

**Symptom**: Studio Pro crashes on open with stack trace through `RevStatusCache.DoRefresh` → `CreateDeleteStatusItem` → `MprUnit.get_Contents` → `MprObject..ctor` → `b__2(JProperty p)`.

**Root cause**: `MprObject` constructor iterates each non-`$` BSON property and does a LINQ `First()` lookup against the Mendix type schema. Any property name not in the schema → `First()` fails → crash.

**This is triggered by**:
- Any uncommitted change to `.mpr` or `mprcontents/` files
- Studio Pro uses `mx diff` internally to diff working copy vs git HEAD
- Even `git update-index --assume-unchanged` does NOT help — Studio Pro reads files directly, bypassing git index

**Diagnosis**:
1. Run `mx diff baseline.mpr working.mpr output.mpr` to reproduce
2. If self-diff (`mx diff a.mpr a.mpr out.mpr`) crashes, the project itself has bad BSON
3. Use property comparison tool to find extra/missing keys

**Fix pattern**: Remove extra properties from writer, add missing properties with correct defaults (empty array `bson.A{int32(2)}` for lists, `nil` for nullable objects, `""` for strings).

### Studio Pro Crash: InvalidOperationException in MprProperty..ctor

**Symptom**: Studio Pro crashes when opening a project with `System.InvalidOperationException: Sequence contains no matching element` at `Mendix.Modeler.Storage.Mpr.MprProperty..ctor`.

**Root cause**: A BSON document contains a property (field name) that does not exist in the Mendix type definition for its `$type`. Studio Pro's `MprProperty` constructor uses `First()` to look up each BSON field in the type cache, and crashes on unrecognized fields.

**Diagnosis workflow**:

1. **Collect all (type, property) pairs from the crash project** (requires `pip install pymongo`):
```python
import bson, os
from collections import defaultdict

type_props = defaultdict(set)

def walk_bson(obj, tp):
    if isinstance(obj, dict):
        t = obj.get("$type", "")
        if t:
            for k in obj.keys():
                if k not in ("$type", "$ID"):
                    tp[t].add(k)
        for v in obj.values():
            walk_bson(v, tp)
    elif isinstance(obj, list):
        for item in obj:
            walk_bson(item, tp)

for root, dirs, files in os.walk("mprcontents"):
    for f in files:
        if f.endswith(".mxunit"):
            with open(os.path.join(root, f), "rb") as fh:
                walk_bson(bson.decode(fh.read()), type_props)
```

2. **Compare against a known-good baseline project** (e.g., GenAIDemo):
```python
# Collect baseline_props the same way, then:
for t, props in crash_props.items():
    if t in baseline_props:
        extra = props - baseline_props[t]
        if extra:
            print(f"{t}: EXTRA props = {sorted(extra)}")
```

3. **Extra properties = the crash cause**. The fix is to remove those fields from the writer function.

**Example**: `DomainModels$CrossAssociation` had `ParentConnection` and `ChildConnection` copied from `DomainModels$association`, but these fields don't exist on `CrossAssociation`. Removing them fixed the crash.

**Key principle**: When copying serialization code between similar types (e.g., Association → CrossAssociation), always verify which fields belong to each type by checking a baseline project's BSON.

### CE1613: Selected Attribute/Enumeration No Longer Exists

**Symptom**: `mx check` reports `[CE1613] "The selected attribute 'Module.Entity.AssocName' no longer exists."` or `"The selected enumeration 'Module.Entity' no longer exists."`

**Root cause**: Two variants:

1. **Association stored as Attribute**: In `ChangeActionItem` BSON, an association name was written to the `attribute` field instead of the `association` field. Check the executor code that builds `MemberChange` — it must query the domain model to distinguish associations from attributes.

2. **Entity treated as Enumeration**: In `CreateVariableAction` BSON, an entity qualified name was used as `DataTypes$EnumerationType` instead of `DataTypes$ObjectType`. Check `buildDataType()` in the visitor — bare qualified names default to `TypeEnumeration` and need catalog-based disambiguation.

### CE0463: Widget Definition Changed

**Root cause**: Object property values inconsistent with the widget's editorConfig
mode-dependent **visibility** rules. For `TextTemplate` (Forms$ClientTemplate)
specifically: every WidgetValue carries an empty ClientTemplate when its property
is VISIBLE in the current mode, and `null` when HIDDEN. Property TYPE is
irrelevant — visibility decides. (A `ValueType.Translations` heuristic does NOT
predict this: `loadMoreButtonCaption` ships with a translation yet is hidden by
default; the accessibility labels ship without one yet are visible.)

**Where mxcli decides it**:
- Top-level TextTemplate states: `modelsdk/widgets/texttemplate.go`
  (`normalizeTextTemplateStates` + `defaultHiddenTextTemplates`), the single
  point run from `GetTemplateFullBSON` for every pluggable widget.
- Per-column / per-instance / filter-mode states: the concrete builders in
  `mdl/backend/mpr/datagrid_builder.go` (these are mode-dependent per instance).

**Key-based BSON oracle (fastest way to converge)**: build the widget with mxcli,
run `mx update-widgets` on a v1 copy to get the golden, then diff **by
PropertyKey** (join `Object.Properties[].TypePointer` →
`Type.ObjectType.PropertyTypes[].$ID` → `PropertyKey`), NOT by numeric index.
A reusable implementation lives at `scripts/ce0463-tt-oracle.py` (uses the standalone
`bson` Python package; `bson.loads`/`dumps` round-trips Mendix v1 BSON faithfully).
Run `ce0463-tt-oracle.py built.mpr golden.mpr` for per-key diffs and
`ce0463-tt-oracle.py --hidden-set golden.mpr` to enumerate a widget's default-hidden set.

**Confirm causation by mutation (isolate before fixing)**: start from the golden
(0 errors), flip exactly ONE property in the SQLite `Unit.Contents` BSON, re-run
`mx check`. If the CE0463 appears, that property is the driver. This disproved
two plausible-but-wrong hypotheses (Translations count; the dropdown's
`filterOptions`) — the dropdown driver was actually `attrChoice`/`attr` mode.

**Quick workaround (v1 only)**: Run `mx update-widgets` after creating pages.
NOTE: for MPR v2 projects this irreversibly converts v2→v1, so mxcli must emit
correct BSON natively (see `cmd/mxcli/docker/check.go`).

### CE0642: Property X Is Required

**Root cause**: A property that should be visible (per editorConfig.js rules) has been cleared or is missing a required value.

**Fix**: Check the property state matrix — visible properties need their default values, hidden properties can be cleared.

### Type Section Mismatch

**Symptoms**: New properties missing, old properties present, wrong property count.

**Fix**: Extract a fresh template from a project with `mx update-widgets` applied. The Type section must match the installed widget version exactly.

### CE0115: Arguments Passed to Page Do Not Match Expected Parameters

**Symptom**: `mx check` reports `[CE0115] "The arguments that are passed to page 'X' do not match the expected parameters."` on an action button that uses `show_page TargetPage (Param: $currentObject)`.

**Root cause**: The BSON array type indicator rule. Every Mendix BSON array must begin with an `int32` **type indicator** of `2` or `3`. This indicator is skipped by `extractBsonArray` and by Studio Pro's reader. Writing `int32(len(items))` as the first element instead produces an invalid indicator when `len ≠ 2` and `len ≠ 3` — Studio Pro cannot recognise the array and reads 0 parameter mappings.

**Type indicator values**:
| First element | Meaning |
|---------------|---------|
| `int32(3)` | Initialized array — used for `Items`, `DesignProperties`, and most non-empty lists |
| `int32(2)` | Initialized empty array — used for `ParameterMappings`, `PagesForSpecializations`, `Parameters` |
| Any other value | **Invalid** — Studio Pro ignores the entire array |

**How Studio Pro stores page parameter mappings**: Studio Pro does **not** store explicit `Forms$PageParameterMapping` objects in BSON. It always writes `ParameterMappings: [2]` (type indicator only, no inline objects) and infers `$currentObject` at runtime from the enclosing widget context — DataGrid row, DataView datasource, etc. No matter what the MDL source specifies for `(Param: $currentObject)`, the correct serialization is always the empty `[2]` array.

**Correct writer pattern for `Forms$FormAction`**:
```go
formSettings := bson.D{
    {Key: "$ID",               Value: idToBsonBinary(generateUUID())},
    {Key: "$Type",             Value: "Forms$FormSettings"},
    {Key: "Form",              Value: pageName},          // BY_NAME_REFERENCE
    {Key: "ParameterMappings", Value: bson.A{int32(2)}}, // always empty; runtime infers mapping
    {Key: "TitleOverride",     Value: nil},
}
return bson.D{
    {Key: "$ID",                    Value: idToBsonBinary(id)},
    {Key: "$Type",                  Value: "Forms$FormAction"},
    {Key: "DisabledDuringExecution", Value: true},
    {Key: "FormSettings",           Value: formSettings},
    {Key: "NumberOfPagesToClose2",  Value: ""},
    {Key: "PagesForSpecializations", Value: bson.A{int32(2)}},
}
```

**What NOT to do**:
```go
// WRONG: produces [1, {mapping}] — invalid type indicator 1
paramMappings := bson.A{int32(len(a.ParameterMappings))}
for _, pm := range a.ParameterMappings {
    paramMappings = append(paramMappings, bson.D{...Forms$PageParameterMapping...})
}
```

**Diagnostic check**: Inspect the raw `ParameterMappings` array in a Python BSON dump. If it shows `[1, {...}]` instead of `[2]`, the type indicator is wrong. Studio Pro-generated pages always show `[2]`.

## Key Principles

1. **Template cloning > building from scratch**: Clone properties from a known-good template Object, then modify only specific values. Building from scratch produces subtly different structures.

2. **Mode-dependent properties must be consistent**: When changing a mode-switching property (e.g., `showContentAs`), all dependent properties must be updated to match.

3. **`mx update-widgets` is the safety net**: Running this post-processing step normalizes all widget Objects to match mpk definitions. Use it as a fallback.

4. **The mpk is the source of truth**: The XML schema defines property types/defaults, the editorConfig.js defines visibility rules. Together they specify the complete expected Object structure.

## Related Documentation

- [BSON Tooling Guide](../../docs/03-development/BSON_TOOLING_GUIDE.md) — Which BSON tool to use when (dump, compare, discover, TUI, Python)
- [PAGE_BSON_SERIALIZATION.md](../../docs/03-development/PAGE_BSON_SERIALIZATION.md) — Full BSON format reference and CE0463 analysis
- [sdk/widgets/templates/README.md](../../sdk/widgets/templates/README.md) — Template extraction requirements
- [implement-mdl-feature.md](./implement-mdl-feature.md) — Full feature implementation workflow
