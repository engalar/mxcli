# mxcli CLI Reference

Full command reference for `mxcli`. For installation and quick start, see the [README](../../README.md).

---

## Explore Project Structure

```bash
# list all modules
mxcli -p app.mpr -c "show modules"

# list entities in a module
mxcli -p app.mpr -c "show entities in MyModule"

# describe any element (module, entity, microflow, nanoflow, page, etc.)
mxcli describe -p app.mpr module MyModule
mxcli describe -p app.mpr entity MyModule.Customer
mxcli describe -p app.mpr microflow MyModule.ProcessOrder
mxcli describe -p app.mpr nanoflow MyModule.ValidateInput
mxcli describe -p app.mpr page MyModule.CustomerOverview
mxcli describe -p app.mpr json structure MyModule.CustomerResponse
```

## Full-Text Search

Search across validation messages, log messages, captions, labels, and MDL source:

```bash
mxcli search -p app.mpr "validation"
mxcli search -p app.mpr "error" -q --format names    # pipe-friendly: type<TAB>name per line
mxcli search -p app.mpr "Customer" -q --format json  # json for jq
```

Pipe to describe:
```bash
mxcli search -p app.mpr "validation" -q --format names | head -1 | awk '{print $2}' | \
  xargs mxcli describe -p app.mpr microflow

mxcli search -p app.mpr "error" -q --format names > results.txt
while IFS=$'\t' read -r type name; do
  mxcli describe -p app.mpr "$type" "$name"
done < results.txt
```

## Code Navigation

```bash
mxcli callers -p app.mpr MyModule.ProcessOrder
mxcli callers -p app.mpr MyModule.ProcessOrder --transitive
mxcli callees -p app.mpr MyModule.ProcessOrder
mxcli refs    -p app.mpr MyModule.Customer
mxcli impact  -p app.mpr MyModule.Customer
mxcli context -p app.mpr MyModule.ProcessOrder --depth 3
```

## Create and Modify

```bash
# execute MDL commands inline
mxcli -p app.mpr -c "create entity MyModule.Product (Name: string(200) not null, Price: decimal)"

# execute an MDL script file
mxcli -p app.mpr -c "execute script 'setup.mdl'"

# check MDL syntax before executing
mxcli check script.mdl
mxcli check script.mdl -p app.mpr --references   # also validate references

# preview changes
mxcli diff -p app.mpr changes.mdl
```

## Export and Import

Export an entire project to editable MDL files, then import into any project:

```bash
mxcli export -p app.mpr --output ./export-dir
mxcli export -p app.mpr --output ./export-dir --module MyFirstModule
mxcli export -p app.mpr --output ./export-dir --dry-run

mxcli import -p target.mpr --input ./export-dir
mxcli import -p target.mpr --input ./export-dir --skip-errors
mxcli import -p target.mpr --input ./export-dir --dry-run --skip-errors
```

The export directory mirrors the Studio Pro folder hierarchy:

```
export-dir/
  _marketplace.mdl
  _project/
    settings.mdl
    navigation.mdl
    security.mdl
  MyFirstModule/
    _module.mdl
    _module_roles.mdl
    Enumerations/
    Domain/
    _associations.mdl
    Microflows/
    Pages/
    Layouts/
```

Re-export is incremental — each file starts with `-- @cache: <hash>` and is skipped when unchanged. A typical re-export of an unchanged project takes under 10 ms.

## Catalog Queries

SQL-based querying of project metadata (requires `refresh catalog full`):

```bash
mxcli -p app.mpr -c "select Name, ActivityCount from CATALOG.MICROFLOWS where ActivityCount > 10 ORDER by ActivityCount desc"
mxcli -p app.mpr -c "refresh catalog full; select SourceName, RefKind, TargetName from CATALOG.REFS where TargetName = 'MyModule.Customer'"
mxcli -p app.mpr -c "select * from CATALOG.STRINGS where strings match 'error' limit 10"
```

Available tables: `modules`, `entities`, `microflows`, `nanoflows`, `pages`, `snippets`, `enumerations`, `workflows`, `ACTIVITIES`, `widgets`, `REFS`, `PERMISSIONS`, `STRINGS`, `source`

## Widget Scaffold and Build

```bash
mxcli widget new MySlider
mxcli widget new MySlider --property "value:attribute:Decimal" --property "label:string" --property "onChange:action"
mxcli widget new MySlider --id com.acme.widget.MySlider.MySlider --offline

mxcli widget new CrusherWidgets --package
cd CrusherWidgets
mxcli widget add-widget CrusherSlider --property "value:attribute:Decimal"
mxcli widget add-widget CrusherButton --property "label:string" --property "onClick:action"

mxcli widget build
mxcli widget build --dir ./CrusherWidgets
```

Property spec format: `key:type` or `key:type:subtype`. Supported types: `attribute`, `string`, `integer`, `boolean`, `action`, `datasource`, `expression`, `widgets`.

## Widget Discovery and Bulk Updates

> **EXPERIMENTAL**: Always use `dry run` first and backup your project before applying changes.

```bash
mxcli -p app.mpr -c "show widgets where widgettype like '%combobox%'"
mxcli -p app.mpr -c "show widgets in MyModule"
mxcli -p app.mpr -c "update widgets set 'showLabel' = false where widgettype like '%DataGrid%' dry run"
mxcli -p app.mpr -c "update widgets set 'showLabel' = false, 'labelWidth' = 4 where widgettype like '%combobox%' in MyModule"
```

## Linting

```bash
mxcli lint -p app.mpr
mxcli lint -p app.mpr --format sarif > results.sarif   # CI / GitHub integration
mxcli lint -p app.mpr --list-rules
mxcli lint -p app.mpr --exclude System --exclude Administration
```

14 built-in Go rules (MPR001-MPR007, SEC001-SEC003, CONV011-CONV014) plus 27 bundled Starlark rules covering security, architecture, quality, design, and Mendix best practice conventions. Custom `.star` rules in `.claude/lint-rules/` are loaded automatically.

## Best Practices Report

```bash
mxcli report -p app.mpr
mxcli report -p app.mpr --format html --output report.html
mxcli report -p app.mpr --format json
```

Evaluates the project across 6 categories (Security, Quality, Architecture, Performance, Naming, Design) with a 0-100 score per category and overall.

## Testing

```bash
mxcli test tests/microflows.test.mdl -p app.mpr
mxcli test tests/ --list
mxcli test tests/ -p app.mpr --junit results.xml
```

Tests use `@test` and `@expect` annotations in `.test.mdl` or `.test.md` files. See `mxcli help test` for full syntax.

---

## MDL Language Reference

MDL (Mendix Definition Language) is a SQL-like syntax for working with Mendix models.

Run `mxcli syntax` for reference, or `mxcli syntax <topic>` for specific topics:
- `mxcli syntax keywords` - Reserved keywords
- `mxcli syntax entity` - Entity creation
- `mxcli syntax microflow` - Microflow creation
- `mxcli syntax page` - Page creation
- `mxcli syntax search` - Full-text search

Full syntax tables are in [docs/01-project/MDL_QUICK_REFERENCE.md](../01-project/MDL_QUICK_REFERENCE.md).

### Common statements

```sql
-- Query
show modules;
show entities in MyModule;
describe entity MyModule.Customer;
describe microflow MyModule.ProcessOrder;
describe page MyModule.CustomerOverview;

-- Domain model
create entity MyModule.Product (
  Name: string(200) not null,
  Price: decimal(10,2),
  IsActive: boolean default true
);
create association MyModule.Order_Product
  from MyModule.Order to MyModule.Product
  type ReferenceSet;

-- Pages
create page MyModule.Product_Edit
(
  params: { $Product: MyModule.Product },
  title: 'Edit Product',
  layout: Atlas_Core.PopupLayout
)
{
  dataview dvProduct (datasource: $Product) {
    textbox txtName (label: 'Name', attribute: Name)
    textbox txtPrice (label: 'Price', attribute: Price)
    checkbox cbActive (label: 'Active', attribute: IsActive)
    footer footer1 {
      actionbutton btnSave (caption: 'Save', action: save_changes, buttonstyle: primary)
      actionbutton btnCancel (caption: 'Cancel', action: cancel_changes)
    }
  }
}

-- Security
create module role MyModule.Admin description 'Full access';
create module role MyModule.Viewer description 'Read-only';
grant execute on microflow MyModule.ProcessOrder to MyModule.Admin;
grant view on page MyModule.Product_Edit to MyModule.Admin, MyModule.Viewer;
grant MyModule.Admin on MyModule.Product (create, delete, read *, write *);
grant MyModule.Viewer on MyModule.Product (read *);
create user role AppAdmin (MyModule.Admin) manage all roles;
alter project security level production;
show security matrix in MyModule;

-- Search and navigation
search 'validation';
show callers of MyModule.ProcessOrder transitive;
show references to MyModule.Customer;
show impact of MyModule.Customer;
show context of MyModule.ProcessOrder depth 3;

-- Microflows (reset layout so Studio Pro auto-positions activities on next open)
create or modify microflow MyModule.ProcessOrder () reset layout
begin
  declare $Count integer = 0;
  return $Count;
end;
```

---

## VS Code Extension

The MDL extension for VS Code provides a rich editing experience for `.mdl` files:

- Syntax highlighting and parse diagnostics as you type
- Semantic diagnostics on save (validates references against the project)
- Code completion with context-aware keyword and snippet suggestions
- Hover over `Module.Name` references to see their MDL definition
- Go-to-definition (Ctrl+click) to open element source as a virtual document
- Document outline and folding
- Context menu commands: Run File, Run Selection, Check File

The extension is automatically installed by `mxcli init`.

**Settings:**
- `mdl.mxcliPath` - Path to the mxcli executable (default: `mxcli`)
- `mdl.mprPath` - Path to `.mpr` file (auto-discovered if empty)
