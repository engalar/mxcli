# Debug BSON Serialization Issues

This skill provides a systematic workflow for debugging BSON serialization errors when programmatically creating Mendix pages and widgets.

## When to Use This Skill

Use when encountering:
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

Use `mxcli bson dump` to compare working vs broken objects:

```bash
# Dump as JSON (default)
mxcli bson dump -p app.mpr --type page --object "Mod.BrokenPage" > broken.json

# Dump as NDSL (human-readable, $ID-normalized, sorted fields)
mxcli bson dump -p app.mpr --type page --object "Mod.Page" --format ndsl

# Dump raw BSON bytes (for baseline extraction)
mxcli bson dump -p app.mpr --type page --object "Mod.Page" --format bson > baseline.mxunit

# Compare two objects side-by-side
mxcli bson dump -p app.mpr --type page --compare "Mod.Broken,Mod.Fixed"
mxcli bson dump -p app.mpr --type page --compare "Mod.Broken,Mod.Fixed" --format ndsl
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

## Automated Roundtrip Testing

Instead of manual Studio Pro verification, use the golden file roundtrip test framework to catch parse↔serialize regressions automatically.

### Extracting Baselines

Extract known-good BSON from a Studio Pro-verified project:

```bash
# Extract baselines by type
mxcli bson dump -p app.mpr --type page --object "Mod.Page" --format bson > sdk/mpr/testdata/pages/Page.mxunit
mxcli bson dump -p app.mpr --type microflow --object "Mod.Flow" --format bson > sdk/mpr/testdata/microflows/Flow.mxunit
mxcli bson dump -p app.mpr --type enumeration --object "Mod.Enum" --format bson > sdk/mpr/testdata/enumerations/Enum.mxunit
mxcli bson dump -p app.mpr --type snippet --object "Mod.Snip" --format bson > sdk/mpr/testdata/snippets/Snip.mxunit
```

### Running Roundtrip Tests

Tests in `sdk/mpr/roundtrip_test.go` automatically load all `.mxunit` files from testdata subdirectories, parse to Go structs, serialize back to BSON, and compare via NDSL rendering (skips `$ID`, sorts fields deterministically).

```bash
# Run all roundtrip tests
go test -run TestRoundtrip ./sdk/mpr/ -v

# Run specific type
go test -run TestRoundtrip_Pages ./sdk/mpr/ -v
```

Failures show a line-by-line NDSL diff identifying exactly which fields are lost or changed during roundtrip.

## Related Documentation

- [PAGE_BSON_SERIALIZATION.md](../../docs/03-development/PAGE_BSON_SERIALIZATION.md) — Full BSON format reference and CE0463 analysis
- [sdk/widgets/templates/README.md](../../sdk/widgets/templates/README.md) — Template extraction requirements
- [implement-mdl-feature.md](./implement-mdl-feature.md) — Full feature implementation workflow
