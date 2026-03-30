# Debug BSON Serialization Issues

This skill provides a systematic workflow for debugging BSON serialization errors when programmatically creating Mendix pages and widgets.

## When to Use This Skill

Use when encountering:
- **Studio Pro crash** `System.InvalidOperationException: Sequence contains no matching element` at `MprProperty..ctor`
- **CE1613** "The selected attribute/enumeration no longer exists"
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
# Create a page via MDL
./bin/mxcli exec script.mdl -p /path/to/app.mpr

# Run mx check to get the error
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

# Find the document containing the widget
cursor.execute("SELECT UnitData FROM Unit$ WHERE ContainmentName = 'Document' AND Name = ?", ('PageName',))
row = cursor.fetchone()
doc = bson.decode(row[0])

# Pretty-print to find the widget
print(json.dumps(doc, indent=2, default=str))
```

### Step 4: Check the Widget Package (.mpk)

Extract the widget's mpk to understand its schema and mode-dependent rules:

```bash
# Find the mpk in the project's widgets folder
ls /path/to/project/widgets/*.mpk

# Extract (mpk is a ZIP archive)
mkdir /tmp/mpk-widget
cd /tmp/mpk-widget && unzip /path/to/project/widgets/com.mendix.widget.web.Datagrid.mpk
```

Key files inside the mpk:
- **`{Widget}.xml`** — Property schema: types, defaults, enumerations, nested objects
- **`{Widget}.editorConfig.js`** — Mode-dependent visibility rules (which properties hide/show based on other values)
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

# Read the working widget BSON
with open('working-widget.bson', 'rb') as f:
    doc = bson.decode(f.read())

# Change only one property value
# ... modify the specific property ...

# Re-encode and write back
with open('test-widget.bson', 'wb') as f:
    f.write(bson.encode(doc))

# Then insert back into the MPR and run mx check
```

### Step 7: Extract Fresh Templates

If the widget template is outdated, extract a fresh one:

```bash
# First update the test project's widgets
reference/mxbuild/modeler/mx convert -p /path/to/app.mpr
reference/mxbuild/modeler/mx update-widgets /path/to/app.mpr

# Then extract using mxcli
./bin/mxcli extract-templates -p /path/to/app.mpr -widget "com.mendix.widget.web.datagrid.DataGrid2" -o /tmp/template.json
```

Templates must include both `type` (PropertyTypes schema) AND `object` (default WidgetObject).

## Common Error Patterns

### Studio Pro Crash: InvalidOperationException in MprProperty..ctor

**Symptom**: Studio Pro crashes when opening a project with `System.InvalidOperationException: Sequence contains no matching element` at `Mendix.Modeler.Storage.Mpr.MprProperty..ctor`.

**Root cause**: A BSON document contains a property (field name) that does not exist in the Mendix type definition for its `$Type`. Studio Pro's `MprProperty` constructor uses `First()` to look up each BSON field in the type cache, and crashes on unrecognized fields.

**Diagnosis workflow**:

1. **Collect all (type, property) pairs from the crash project** (requires `pip install pymongo`):
```python
import bson, os
from collections import defaultdict

type_props = defaultdict(set)

def walk_bson(obj, tp):
    if isinstance(obj, dict):
        t = obj.get("$Type", "")
        if t:
            for k in obj.keys():
                if k not in ("$Type", "$ID"):
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

**Example**: `DomainModels$CrossAssociation` had `ParentConnection` and `ChildConnection` copied from `DomainModels$Association`, but these fields don't exist on `CrossAssociation`. Removing them fixed the crash.

**Key principle**: When copying serialization code between similar types (e.g., Association → CrossAssociation), always verify which fields belong to each type by checking a baseline project's BSON.

### CE1613: Selected Attribute/Enumeration No Longer Exists

**Symptom**: `mx check` reports `[CE1613] "The selected attribute 'Module.Entity.AssocName' no longer exists."` or `"The selected enumeration 'Module.Entity' no longer exists."`

**Root cause**: Two variants:

1. **Association stored as Attribute**: In `ChangeActionItem` BSON, an association name was written to the `Attribute` field instead of the `Association` field. Check the executor code that builds `MemberChange` — it must query the domain model to distinguish associations from attributes.

2. **Entity treated as Enumeration**: In `CreateVariableAction` BSON, an entity qualified name was used as `DataTypes$EnumerationType` instead of `DataTypes$ObjectType`. Check `buildDataType()` in the visitor — bare qualified names default to `TypeEnumeration` and need catalog-based disambiguation.

### CE0463: Widget Definition Changed

**Root cause**: Object property values inconsistent with mode-dependent visibility rules.

**Fix**: Adjust properties based on the widget's current mode. See [PAGE_BSON_SERIALIZATION.md](../../docs/03-development/PAGE_BSON_SERIALIZATION.md#ce0463-widget-definition-changed--root-cause-analysis) for the full analysis.

**Quick workaround**: Run `mx update-widgets` after creating pages.

### CE0642: Property X Is Required

**Root cause**: A property that should be visible (per editorConfig.js rules) has been cleared or is missing a required value.

**Fix**: Check the property state matrix — visible properties need their default values, hidden properties can be cleared.

### Type Section Mismatch

**Symptoms**: New properties missing, old properties present, wrong property count.

**Fix**: Extract a fresh template from a project with `mx update-widgets` applied. The Type section must match the installed widget version exactly.

## Key Principles

1. **Template cloning > building from scratch**: Clone properties from a known-good template Object, then modify only specific values. Building from scratch produces subtly different structures.

2. **Mode-dependent properties must be consistent**: When changing a mode-switching property (e.g., `showContentAs`), all dependent properties must be updated to match.

3. **`mx update-widgets` is the safety net**: Running this post-processing step normalizes all widget Objects to match mpk definitions. Use it as a fallback.

4. **The mpk is the source of truth**: The XML schema defines property types/defaults, the editorConfig.js defines visibility rules. Together they specify the complete expected Object structure.

## Related Documentation

- [PAGE_BSON_SERIALIZATION.md](../../docs/03-development/PAGE_BSON_SERIALIZATION.md) — Full BSON format reference and CE0463 analysis
- [sdk/widgets/templates/README.md](../../sdk/widgets/templates/README.md) — Template extraction requirements
- [implement-mdl-feature.md](./implement-mdl-feature.md) — Full feature implementation workflow
