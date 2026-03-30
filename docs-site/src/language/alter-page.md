# ALTER PAGE / ALTER SNIPPET

The `ALTER PAGE` and `ALTER SNIPPET` statements modify an existing page or snippet's widget tree in-place, without requiring a full `CREATE OR REPLACE`. This is especially useful for incremental changes: adding a field, changing a button caption, or removing an unused widget.

ALTER operates directly on the raw widget tree, preserving any widget types that MDL does not natively support (pluggable widgets, custom widgets, etc.).

## Syntax

```sql
ALTER PAGE <Module>.<Name> {
  <operations>
};

ALTER SNIPPET <Module>.<Name> {
  <operations>
};
```

## Operations

### SET -- Modify Widget Properties

Change one or more properties on a widget identified by name:

```sql
-- Single property
ALTER PAGE Module.EditPage {
  SET Caption = 'Save & Close' ON btnSave
};

-- Multiple properties at once
ALTER PAGE Module.EditPage {
  SET (Caption = 'Save & Close', ButtonStyle = Success) ON btnSave
};
```

**Supported SET properties:**

| Property | Description | Example |
|----------|-------------|---------|
| `Caption` | Button/link caption | `SET Caption = 'Submit' ON btnSave` |
| `Label` | Input field label | `SET Label = 'Full Name' ON txtName` |
| `ButtonStyle` | Button visual style | `SET ButtonStyle = Danger ON btnDelete` |
| `Class` | CSS class names | `SET Class = 'card p-3' ON cMain` |
| `Style` | Inline CSS | `SET Style = 'margin: 8px;' ON cBox` |
| `Editable` | Editability mode | `SET Editable = ReadOnly ON txtEmail` |
| `Visible` | Visibility expression | `SET Visible = '$showField' ON txtPhone` |
| `Name` | Widget name | `SET Name = 'txtFullName' ON txtName` |

### SET -- Page-Level Properties

Omit the `ON` clause to set page-level properties:

```sql
ALTER PAGE Module.EditPage {
  SET Title = 'Customer Details'
};
```

### SET -- Pluggable Widget Properties

Use quoted property names to set properties on pluggable widgets (ComboBox, DataGrid2, etc.):

```sql
ALTER PAGE Module.EditPage {
  SET 'showLabel' = false ON cbStatus
};
```

### SET Layout -- Change Page Layout

Switch a page's layout without rebuilding the widget tree. All widget content is preserved -- only the layout reference and placeholder mappings are updated.

```sql
-- Auto-map placeholders by name (common case)
ALTER PAGE Module.EditPage {
  SET Layout = Atlas_Core.Atlas_Default
};

-- Explicit mapping when placeholder names differ
ALTER PAGE Module.EditPage {
  SET Layout = Atlas_Core.Atlas_SideBar MAP (Main AS Content, Extra AS Sidebar)
};
```

When both the old and new layouts share the same placeholder names (e.g., both have `Main`), no `MAP` clause is needed -- placeholders are matched automatically. Use `MAP` when the new layout has different placeholder names.

Not supported for snippets (snippets don't have layouts).

### INSERT -- Add Widgets

Insert new widgets before or after an existing widget:

```sql
-- Insert after a widget
ALTER PAGE Module.EditPage {
  INSERT AFTER txtEmail {
    TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
    TEXTBOX txtFax (Label: 'Fax', Attribute: Fax)
  }
};

-- Insert before a widget
ALTER PAGE Module.EditPage {
  INSERT BEFORE btnSave {
    ACTIONBUTTON btnPreview (Caption: 'Preview', Action: MICROFLOW Module.ACT_Preview)
  }
};
```

The inserted widgets use the same syntax as in `CREATE PAGE`. Multiple widgets can be inserted in a single block.

### DROP WIDGET -- Remove Widgets

Remove one or more widgets by name:

```sql
ALTER PAGE Module.EditPage {
  DROP WIDGET txtUnused
};

-- Multiple widgets
ALTER PAGE Module.EditPage {
  DROP WIDGET txtFax, txtPager, btnObsolete
};
```

Dropping a container widget also removes all of its children.

### REPLACE -- Replace a Widget

Replace a widget (and its subtree) with new widgets:

```sql
ALTER PAGE Module.EditPage {
  REPLACE txtOldField WITH {
    TEXTAREA txtNotes (Label: 'Notes', Attribute: Notes)
  }
};
```

### Page Variables

Add or remove page-level variables:

```sql
-- Add a variable
ALTER PAGE Module.EditPage {
  ADD Variables $showAdvanced: Boolean = 'false'
};

-- Remove a variable
ALTER PAGE Module.EditPage {
  DROP Variables $showAdvanced
};
```

## Combining Operations

Multiple operations can be combined in a single ALTER statement. They are applied in order:

```sql
ALTER PAGE Module.Customer_Edit {
  -- Change button appearance
  SET (Caption = 'Save & Close', ButtonStyle = Success) ON btnSave;

  -- Remove unused fields
  DROP WIDGET txtFax;

  -- Add new fields after email
  INSERT AFTER txtEmail {
    TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
    TEXTBOX txtMobile (Label: 'Mobile', Attribute: Mobile)
  };

  -- Replace old status dropdown with combobox
  REPLACE ddStatus WITH {
    COMBOBOX cbStatus (Label: 'Status', Attribute: Status)
  }
};
```

## Workflow Tips

1. **Discover widget names first** -- Run `DESCRIBE PAGE Module.PageName` to see the current widget tree with all widget names.

2. **Use ALTER for small changes** -- For adding a field or changing a caption, ALTER is faster and safer than `CREATE OR REPLACE`, because it preserves widgets that MDL cannot round-trip (pluggable widgets with complex configurations).

3. **Use CREATE OR REPLACE for major rewrites** -- When restructuring the entire page layout, a full replacement is cleaner.

## Examples

### Add a Field to an Edit Page

```sql
-- First, check what's on the page
DESCRIBE PAGE MyModule.Customer_Edit;

-- Add a phone field after email
ALTER PAGE MyModule.Customer_Edit {
  INSERT AFTER txtEmail {
    TEXTBOX txtPhone (Label: 'Phone Number', Attribute: Phone)
  }
};
```

### Change Button Behavior

```sql
ALTER PAGE MyModule.Order_Edit {
  SET (Caption = 'Submit Order', ButtonStyle = Success) ON btnSave;
  SET Caption = 'Discard' ON btnCancel
};
```

### Add a DataGrid Column

Since DataGrid columns are part of the widget tree, use INSERT to add columns:

```sql
ALTER PAGE MyModule.Customer_Overview {
  INSERT AFTER colEmail {
    COLUMN colPhone (Attribute: Phone, Caption: 'Phone')
  }
};
```

## See Also

- [Pages](./pages.md) -- page overview and CREATE PAGE basics
- [Widget Types](./widget-types.md) -- widgets available for INSERT and REPLACE
- [Snippets](./snippets.md) -- ALTER SNIPPET uses the same operations
- [Common Patterns](./page-patterns.md) -- page layout patterns
