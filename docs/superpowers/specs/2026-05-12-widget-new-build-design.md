# Widget Scaffold & Build Pipeline Design

**Date:** 2026-05-12
**Branch:** feature/expression-checker (to be extracted to feature/widget-new-build)
**Status:** Approved

## Context

crusher-widgets (`/mnt/data_sdd/gh/posui/crusher-widgets`) is a hand-built 5-widget MPK package (CavitySelector, CrusherSlider, PredictionBadge, CrusherSimCanvas, HeatmapViz). Its build is a hand-written `build.sh` that invokes esbuild for each widget, copies assets, and zips into an MPK. Developers starting a new widget project have no mxcli-native workflow — they copy the shell script, adjust widget names, and maintain it by hand.

mxcli already knows how to **parse** `.mpk` files (`sdk/widgets/mpk/`), **derive** BSON templates from them (`sdk/widgets/generate.go`), and **use** widgets in `CREATE PAGE` MDL. This feature adds the upstream capability: scaffolding new widget projects and building them into MPKs.

## Goals

1. `mxcli widget new <name>` — scaffold a single pluggable widget project with a parameterized XML definition and JSX stub.
2. `mxcli widget new <name> --package` — scaffold a multi-widget package project.
3. `mxcli widget add-widget <name>` — add a widget to an existing package project.
4. `mxcli widget build` — validate XML, invoke esbuild via external bun/npm, package dist/ into `.mpk`, verify output. Replaces hand-written `build.sh`.

## Non-Goals

- Embedding esbuild in the mxcli binary (user's build toolchain is external).
- Auto-deploying the MPK to a Mendix project's `widgets/` directory.
- Generating test files or storybook setup.
- TypeScript support (JSX only for now).

## Command Surface

```
# Single widget scaffold
mxcli widget new MySlider
mxcli widget new MySlider \
  --id com.acme.widget.MySlider \
  --property "value:attribute:Decimal" \
  --property "label:string" \
  --property "onChange:action" \
  --offline

# Multi-widget package scaffold
mxcli widget new CrusherWidgets --package

# Add widget to existing package (run inside package dir)
mxcli widget add-widget CrusherSlider \
  --property "value:attribute:Decimal" \
  --property "label:string"

# Build (run inside widget project dir)
mxcli widget build
mxcli widget build --dir ./CrusherWidgets
```

### Flag reference

**`widget new`**

| Flag | Default | Description |
|------|---------|-------------|
| `--id` | `com.mendix.widget.custom.<Name>.<Name>` | Widget ID (`com.a.b.Name` format, 4 segments) |
| `--property key:type[:subtype]` | none | Repeatable; populates XML + JSX props |
| `--offline` | false | Sets `offlineCapable="true"` in XML |
| `--package` | false | Creates multi-widget package project |

**`widget build`**

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | `.` | Widget project root directory |

### `--property` type values

| Type string | XML type | Notes |
|-------------|----------|-------|
| `attribute:Decimal` | `attribute` | `<attributeType name="Decimal"/>` |
| `attribute:String` | `attribute` | `<attributeType name="String"/>` |
| `attribute:Integer` | `attribute` | `<attributeType name="Integer"/>` |
| `string` | `string` | Static string property |
| `integer` | `integer` | Static integer property |
| `boolean` | `boolean` | Static boolean property |
| `action` | `action` | Nanoflow/microflow action |
| `datasource` | `datasource` | XPath datasource |
| `expression` | `expression` | Expression property |
| `widgets` | `widgets` | Child widget slot |

## Scaffold Output

### Single widget (`mxcli widget new MySlider`)

```
MySlider/
├── package.json              # { devDependencies: { esbuild: "^0.20" } }
├── package.xml               # MPK manifest referencing src/MySlider
└── src/
    ├── MySlider.xml          # Widget property definition
    ├── MySlider.jsx          # React stub with typed props
    ├── MySlider.editorConfig.js
    ├── MySlider.editorPreview.js
    ├── MySlider.icon.png         # 16×16 placeholder (embedded in mxcli binary)
    ├── MySlider.icon.dark.png
    ├── MySlider.tile.png         # 256×192 placeholder
    └── MySlider.tile.dark.png
```

No `build.sh` is generated. `mxcli widget build` is the build command.

### Multi-widget package (`mxcli widget new CrusherWidgets --package`)

```
CrusherWidgets/
├── package.json
├── package.xml               # Package-level manifest (name = CrusherWidgets)
└── src/                      # Empty; add-widget appends files here
```

`add-widget` also appends a `<widgetFile path="CrusherSlider.xml"/>` entry to `package.xml` so the new widget is included in the next build.

After `mxcli widget add-widget CrusherSlider`:

```
src/
├── CrusherSlider.xml
├── CrusherSlider.jsx
├── CrusherSlider.editorConfig.js
├── CrusherSlider.editorPreview.js
├── CrusherSlider.icon.png
├── CrusherSlider.icon.dark.png
├── CrusherSlider.tile.png
└── CrusherSlider.tile.dark.png
```

All widgets are flat under `src/` — same layout as crusher-widgets.

### Generated file content

**`package.json`:**
```json
{
  "name": "my-slider",
  "version": "1.0.0",
  "private": true,
  "type": "module",
  "devDependencies": {
    "esbuild": "^0.20.0"
  }
}
```

**`package.xml`** (package-level MPK manifest):
```xml
<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.mendix.com/package/1.0/">
  <clientModule name="MySlider" version="1.0.0" xmlns="http://www.mendix.com/clientmodule/1.0/">
    <widgetFiles>
      <widgetFile path="MySlider.xml"/>
    </widgetFiles>
  </clientModule>
</package>
```

**`MySlider.xml`** (with `--property value:attribute:Decimal --property label:string --property onChange:action`):
```xml
<?xml version="1.0" encoding="utf-8"?>
<widget id="com.acme.widget.MySlider.MySlider"
        pluginWidget="true"
        offlineCapable="false"
        xmlns="http://www.mendix.com/widget/1.0/"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../../../../node_modules/mendix/custom_widget.xsd">
  <name>My Slider</name>
  <description></description>
  <properties>
    <propertyGroup caption="General">
      <property key="value" type="attribute" required="true">
        <caption>Value</caption>
        <description/>
        <attributeTypes><attributeType name="Decimal"/></attributeTypes>
      </property>
      <property key="label" type="string" required="true" defaultValue="">
        <caption>Label</caption>
        <description/>
      </property>
      <property key="onChange" type="action" required="true">
        <caption>On Change</caption>
        <description/>
      </property>
    </propertyGroup>
  </properties>
</widget>
```

**`MySlider.jsx`** (props derived from XML attributes):
```jsx
import { createElement } from 'react';

export function MySlider({ value, label, onChange }) {
    return createElement('div', { className: 'my-slider' },
        createElement('span', null, label ?? 'MySlider'),
        // TODO: implement
    );
}

export default MySlider;
```

**`MySlider.editorConfig.js`** (Studio Pro design-time preview):
```js
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getCustomCaption = function (props) {
    return props && props.label ? props.label : "MySlider";
};
exports.getPreview = function (props, isDarkMode) {
    return {
        type: "RowLayout",
        columnSize: "grow",
        children: [{
            type: "Text",
            content: "MySlider",
            fontColor: isDarkMode ? "#cba6f7" : "#89b4fa",
        }]
    };
};
```

**`MySlider.editorPreview.js`** — minimal stub (Studio Pro browser preview; rarely customized):
```js
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.preview = function () { return null; };
```

## Build Pipeline (`mxcli widget build`)

```
mxcli widget build [--dir <path>]
  │
  ├─ 1. Discover  — glob src/*.xml → parse each, extract widgetID and name
  ├─ 2. Validate  — widget ID: must be 4 dot-separated segments (com.a.b.Name)
  │                 <name>: non-empty
  │                 property types: must be known values
  │                 → fail fast with file:line and reason
  ├─ 3. Detect    — bun first, then npm; neither → exit with install hint
  ├─ 4. Install   — if node_modules/ absent: bun install / npm install
  ├─ 5. Compile   — for each widget W found in src/*.xml:
  │                   esbuild src/W.jsx --bundle --format=cjs
  │                     --external:react --external:react-dom --external:big.js
  │                     --outfile=dist/com/mendix/widget/custom/W/W.js
  │                   esbuild ... --format=esm → W.mjs
  │                 (via `bun x esbuild` or `npx esbuild`)
  ├─ 6. Assets    — cp src/*.xml dist/
  │                 cp src/*.editorConfig.js dist/
  │                 cp src/*.editorPreview.js dist/
  │                 cp src/*.icon*.png dist/
  │                 cp src/*.tile*.png dist/
  │                 cp package.xml dist/
  ├─ 7. Package   — zip dist/ → <PackageName>.mpk
  │                 PackageName from package.xml <clientModule name="...">
  └─ 8. Verify    — open MPK ZIP, confirm each widgetID has its .js file
                    → print: Built MySlider.mpk (1 widget, 12 KB)
```

### esbuild invocation

Calls `bun x esbuild` (or `npx esbuild`) — no dependency on `node_modules/.bin` path. Additional bundled dependencies (e.g. `three`) declared by the user in `package.json` are included automatically by esbuild.

### Error output

```
✗  src/MySlider.xml line 2: widget id "com.acme.MySlider" is invalid
   must be 4 dot-separated segments, e.g. com.acme.widget.MySlider

✗  bun not found, npm not found
   install bun: curl -fsSL https://bun.sh/install | bash
```

## Code Placement in mxcli

```
cmd/mxcli/
├── cmd_widget.go          # existing — add new subcommand registrations here
├── widget_scaffold.go     # new — new/add-widget logic + template rendering
├── widget_build.go        # new — build pipeline (discover/validate/compile/package/verify)
└── widget_templates/      # go:embed directory
    ├── icon.png            # 16×16 placeholder icon
    ├── icon.dark.png
    ├── tile.png            # 256×192 placeholder tile
    └── tile.dark.png
```

Template strings (XML, JSX, editorConfig, package.json, package.xml skeletons) are Go `const` strings rendered with `text/template`. No separate template files — binary stays self-contained.

## Relationship to Existing Widget Commands

| Existing command | Change |
|-----------------|--------|
| `widget extract` | Unchanged — still extracts `.def.json` from a third-party MPK for `CREATE PAGE` use |
| `widget init` | Unchanged — scans project `widgets/` and writes `.def.json` files |
| `widget list` | Unchanged — lists registered widget definitions |
| `widget docs` | Unchanged — generates markdown docs |
| `widget new` | **New** |
| `widget add-widget` | **New** |
| `widget build` | **New** |

The new commands address the **authoring** side (create + build a widget). The existing commands address the **consumption** side (import a widget into MDL). They are independent concerns with no shared code path.
