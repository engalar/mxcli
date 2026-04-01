# Pages

Pages define the user interface of a Mendix application. Each page consists of a widget tree arranged within a layout, with data sources that connect widgets to the domain model.

## Core Concepts

| Concept | Description |
|---------|-------------|
| **Layout** | A reusable page template that defines content regions (e.g., header, sidebar, main content) |
| **Widget tree** | A hierarchical structure of widgets that defines the page's visual content |
| **Data source** | Determines how a widget obtains its data (page parameter, database query, microflow, etc.) |
| **Widget name** | Every widget has a unique name within the page, used for ALTER PAGE operations |

## CREATE PAGE

The basic syntax for creating a page:

```sql
CREATE [OR REPLACE] PAGE <Module>.<Name>
(
  [Params: { $Param: Module.Entity [, ...] },]
  Title: '<title>',
  Layout: <Module.LayoutName>
  [, Folder: '<path>']
)
{
  <widget-tree>
}
```

### Minimal Example

```sql
CREATE PAGE MyModule.Home
(
  Title: 'Welcome',
  Layout: Atlas_Core.Atlas_Default
)
{
  CONTAINER cMain {
    DYNAMICTEXT txtWelcome (Content: 'Welcome to the application')
  }
}
```

### Page with Parameters

Pages can receive entity objects as parameters from the calling context:

```sql
CREATE PAGE MyModule.Customer_Edit
(
  Params: { $Customer: MyModule.Customer },
  Title: 'Edit Customer',
  Layout: Atlas_Core.PopupLayout
)
{
  DATAVIEW dvCustomer (DataSource: $Customer) {
    TEXTBOX txtName (Label: 'Name', Attribute: Name)
    TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
    FOOTER footer1 {
      ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)
      ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
    }
  }
}
```

## Page Properties

| Property | Description | Example |
|----------|-------------|---------|
| `Params` | Page parameters (entity objects passed in) | `Params: { $Order: Sales.Order }` |
| `Title` | Page title shown in the browser/tab | `Title: 'Edit Customer'` |
| `Layout` | Layout to use for the page | `Layout: Atlas_Core.PopupLayout` |
| `Folder` | Organizational folder within the module | `Folder: 'Pages/Customers'` |
| `Variables` | Page-level variables for conditional logic | `Variables: { $show: Boolean = 'true' }` |

## Widget Properties

### Responsive Column Widths

Layout grid columns support responsive widths for desktop, tablet, and phone:

```sql
COLUMN col1 (DesktopWidth: 8, TabletWidth: 6, PhoneWidth: 12) { ... }
```

Values are 1-12 (grid units) or `AutoFill`. TabletWidth and PhoneWidth default to auto when omitted.

### Conditional Visibility

Any widget can be conditionally visible using an XPath expression in brackets:

```sql
TEXTBOX txtName (Label: 'Name', Attribute: Name, Visible: [IsActive])
```

Static values also work: `Visible: false` hides the widget unconditionally.

### Conditional Editability

Input widgets can be conditionally editable:

```sql
TEXTBOX txtStatus (Label: 'Status', Attribute: Status, Editable: [Status != 'Closed'])
```

Static values: `Editable: Never`, `Editable: Always`.

## Layouts

Layouts are referenced by their qualified name (`Module.LayoutName`). Common Atlas layouts include:

| Layout | Usage |
|--------|-------|
| `Atlas_Core.Atlas_Default` | Full-page layout with navigation sidebar |
| `Atlas_Core.PopupLayout` | Modal popup dialog |
| `Atlas_Core.Atlas_TopBar` | Layout with top navigation bar |

## DROP PAGE

Removes a page from the project:

```sql
DROP PAGE MyModule.Customer_Edit;
```

## Inspecting Pages

Use `SHOW` and `DESCRIBE` to examine existing pages:

```sql
-- List all pages in a module
SHOW PAGES IN MyModule;

-- Show the full MDL definition of a page (round-trippable)
DESCRIBE PAGE MyModule.Customer_Edit;
```

The output of `DESCRIBE PAGE` can be used as input to `CREATE OR REPLACE PAGE` for round-trip editing.

## See Also

- [Page Structure](./page-structure.md) -- layout selection, content areas, and data sources
- [Widget Types](./widget-types.md) -- full catalog of available widgets
- [Data Binding](./data-binding.md) -- connecting widgets to entity attributes
- [Snippets](./snippets.md) -- reusable page fragments
- [ALTER PAGE](./alter-page.md) -- modifying existing pages in-place
- [Common Patterns](./page-patterns.md) -- list page, edit page, master-detail patterns
