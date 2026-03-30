# CREATE PAGE - MDL Syntax Guide

## Overview
Guide for writing CREATE PAGE statements in Mendix Definition Language (MDL).

## Syntax

```sql
CREATE [OR REPLACE] PAGE Module.PageName
(
  [Params: { $ParamName: Module.EntityType, ... },]
  [Variables: { $varName: DataType = 'defaultExpression', ... },]
  Title: 'Page Title',
  Layout: Module.LayoutName,
  [Url: 'page-url',]
  [Folder: 'FolderPath']
)
{
  -- Widget definitions using explicit properties
}
```

**Page Variables**: Local variables at the page level for use in expressions (e.g., column visibility).
- DataType: `Boolean`, `String`, `Integer`, `Decimal`, `DateTime`
- Default value: Mendix expression in single quotes
- Referenced in expressions as `$varName`
- Use for DataGrid2 column `Visible:` (which hides/shows entire column, NOT per-row)

### Key Syntax Elements

| Element | Syntax | Example |
|---------|--------|---------|
| Properties | `(Key: value, ...)` | `(Title: 'Edit', Layout: Atlas_Core.Atlas_Default)` |
| Widget name | Required after type | `TEXTBOX txtName (...)` |
| Attribute binding | `Attribute: AttrName` | `TEXTBOX txt (Label: 'Name', Attribute: Name)` |
| Variable binding | `DataSource: $Var` | `DATAVIEW dv (DataSource: $Product) { ... }` |
| Action binding | `Action: TYPE` | `ACTIONBUTTON btn (Caption: 'Save', Action: SAVE_CHANGES)` |
| Database source | `DataSource: DATABASE Entity` | `DATAGRID dg (DataSource: DATABASE Module.Entity)` |
| Selection binding | `DataSource: SELECTION widget` | `DATAVIEW dv (DataSource: SELECTION galleryList)` |
| CSS class | `Class: 'classes'` | `CONTAINER c (Class: 'card mx-spacing-top-large')` |
| Inline style | `Style: 'css'` | `CONTAINER c (Style: 'padding: 16px;')` |
| Design properties | `DesignProperties: [...]` | `CONTAINER c (DesignProperties: ['Spacing top': 'Large', 'Full width': ON])` |

### Reserved Words as Attribute Names

When an attribute name matches an MDL reserved keyword (like `Status`, `Date`, `Value`, `Type`), use double quotes to escape it:

| Syntax | Purpose | Example |
|--------|---------|---------|
| `"Status"` | Escape reserved keyword | `COMBOBOX cb (Label: 'Status', Attribute: "Status")` |
| `"Date"` | Escape reserved keyword | `DATEPICKER dp (Label: 'Date', Attribute: "Date")` |
| `"Type"` | Escape reserved keyword | `TEXTBOX tb (Label: 'Type', Attribute: "Type")` |

This also works in DATAGRID columns, DataSource WHERE clauses, and anywhere an attribute name is expected.

Common reserved keywords that need quoting: `Status`, `Date`, `Value`, `Type`, `Name`, `Order`, `User`, `Role`, `Action`, `Event`, `Module`, `Entity`, `All`, `None`, `Default`, `Error`, `Range`, `Level`, `Description`, `Version`.

### FOLDER Option

Place pages in folders for better organization:

```sql
CREATE PAGE MyModule.CustomerEdit
(
  Title: 'Edit Customer',
  Layout: Atlas_Core.PopupLayout,
  Folder: 'Customers'
)
{
  -- widgets
}

-- Nested folders (created automatically if they don't exist)
CREATE PAGE MyModule.OrderDetail
(
  Title: 'Order Details',
  Layout: Atlas_Core.Atlas_Default,
  Folder: 'Orders/Details'
)
{
  -- widgets
}
```

### Styling: Class, Style, and DesignProperties

Three styling mechanisms can be applied to any widget:

**CSS Class** — Atlas UI utility classes or custom CSS classes:
```sql
CONTAINER c (Class: 'card mx-spacing-top-large') { ... }
ACTIONBUTTON btn (Caption: 'Save', Class: 'btn-lg')
```

**Inline Style** — One-off CSS styles (use sparingly):
```sql
CONTAINER c (Style: 'background-color: #f8f9fa; padding: 16px;') { ... }
```

> **Warning:** Do NOT use `Style` directly on DYNAMICTEXT widgets — it crashes MxBuild with a NullReferenceException. Wrap the DYNAMICTEXT in a styled CONTAINER instead.

**Design Properties** — Atlas UI structured properties (spacing, colors, toggles):
```sql
-- Option property: 'Key': 'Value'
CONTAINER c (DesignProperties: ['Spacing top': 'Large', 'Background color': 'Brand Primary']) { ... }

-- Toggle property: 'Key': ON (enabled) or OFF (disabled/omitted)
CONTAINER c (DesignProperties: ['Full width': ON]) { ... }

-- Multiple types combined
ACTIONBUTTON btn (Caption: 'Save', DesignProperties: ['Size': 'Large', 'Full width': ON])
```

**All three can be combined on a single widget:**
```sql
CONTAINER ctnHero (
  Class: 'card',
  Style: 'border-left: 4px solid #264AE5;',
  DesignProperties: ['Spacing top': 'Large', 'Full width': ON]
) {
  DYNAMICTEXT txtTitle (Content: 'Styled Container', RenderMode: H3)
}
```

## Basic Examples

### Simple Page with Title

```sql
CREATE PAGE MyModule.HomePage
(
  Title: 'Home Page',
  Layout: Atlas_Core.Atlas_Default
)
{
  DYNAMICTEXT welcomeText (Content: 'Welcome to My App', RenderMode: H1)
}
```

### Page with Multiple Widgets

```sql
CREATE PAGE MyModule.CustomerPage
(
  Title: 'Customer Details',
  Layout: Atlas_Core.Atlas_Default
)
{
  LAYOUTGRID mainGrid {
    ROW row1 {
      COLUMN col1 (DesktopWidth: 12) {
        DYNAMICTEXT heading (Content: 'Customer Information', RenderMode: H2)
      }
    }
    ROW row2 {
      COLUMN col2a (DesktopWidth: 6) {
        ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)
      }
      COLUMN col2b (DesktopWidth: 6) {
        ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
      }
    }
  }
}
```

## Supported Widgets

### DYNAMICTEXT Widget

Display dynamic or static text:

```sql
-- Simple text
DYNAMICTEXT heading (Content: 'Heading Text', RenderMode: H2)

-- Text bound to page parameter attribute (use $ParamName.Attribute)
-- This preserves the parameter reference for pages with multiple parameters of the same type
DYNAMICTEXT productName (Content: '$Product.Name', RenderMode: H3)

-- Explicit template with page parameter binding
DYNAMICTEXT greeting (Content: 'Welcome, {1}!', ContentParams: [{1} = $Customer.Name])

-- Template with attribute from current DataView context (simple attribute name)
DYNAMICTEXT email (Content: 'Email: {1}', ContentParams: [{1} = Email])
```

**ContentParams Reference Types:**
| Syntax | Context | Example |
|--------|---------|---------|
| `$ParamName.Attr` | Page parameter attribute | `$Product.Name` |
| `AttrName` | Current DataView/Gallery entity | `Name`, `Email` |
| `'literal'` | String literal expression | `'Hello'` |

### ACTIONBUTTON Widget

Create a button with action binding:

```sql
ACTIONBUTTON widgetName (Caption: 'Caption', Action: ACTION_TYPE [, ButtonStyle: style])
```

**Action Bindings:**
- `Action: SAVE_CHANGES` - Save changes to object
- `Action: SAVE_CHANGES CLOSE_PAGE` - Save and close page
- `Action: CANCEL_CHANGES` - Cancel changes
- `Action: CLOSE_PAGE` - Close the page
- `Action: DELETE` - Delete object
- `Action: MICROFLOW Module.MicroflowName` - Call microflow
- `Action: MICROFLOW Module.MicroflowName(Param: $value)` - Call microflow with parameters
- `Action: SHOW_PAGE Module.PageName` - Navigate to page
- `Action: SHOW_PAGE Module.PageName(Param: $value)` - Navigate with parameters
- `Action: SHOW_PAGE Module.PageName($Param = $value)` - Also accepted (microflow-style)
- `Action: CREATE_OBJECT Module.Entity THEN SHOW_PAGE Module.PageName` - Create and navigate

**Button Styles:**
- `Default`, `Primary`, `Success`, `Info`, `Warning`, `Danger`, `Inverse`

**Examples:**
```sql
-- Save with style
ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)

-- Navigate with parameter (inside DATAVIEW)
ACTIONBUTTON btnEdit (Caption: 'Edit', Action: SHOW_PAGE Module.EditPage(Product: $Product))

-- Navigate with $currentObject (inside DATAGRID column)
ACTIONBUTTON btnEdit (Caption: 'Edit', Action: SHOW_PAGE Module.EditPage(Product: $currentObject))

-- Call microflow with page/dataview parameter
ACTIONBUTTON btnProcess (Caption: 'Process', Action: MICROFLOW Module.ACT_Process(Order: $Order), ButtonStyle: Success)

-- Call microflow with $currentObject (inside DATAGRID/LISTVIEW column)
ACTIONBUTTON btnDelete (Caption: 'Delete', Action: MICROFLOW Module.ACT_Delete(Target: $currentObject), ButtonStyle: Danger)

-- Create object and show page
ACTIONBUTTON btnNew (Caption: 'New', Action: CREATE_OBJECT Module.Product THEN SHOW_PAGE Module.Product_Edit, ButtonStyle: Primary)
```

**Using `$currentObject`:**
Use `$currentObject` inside DATAGRID, LISTVIEW, or GALLERY columns to reference the current row's object. This is typically used in columns with `ShowContentAs: customContent` for action buttons.

### LINKBUTTON Widget

Similar to ActionButton but rendered as link:

```sql
LINKBUTTON linkName (Caption: 'Caption', Action: ACTION_TYPE)
```

### LAYOUTGRID Widget

Create responsive grid layout:

```sql
LAYOUTGRID gridName {
  ROW rowName {
    COLUMN colName (DesktopWidth: 8) {
      -- Nested widgets
    }
    COLUMN col2 (DesktopWidth: 4) {
      -- Nested widgets
    }
  }
}
```

**Width Values:**
- Numeric: `1` through `12`
- Auto: `AutoFill`, `AutoFit`

Example:
```sql
LAYOUTGRID mainGrid {
  ROW row1 {
    COLUMN colMain (DesktopWidth: 8) {
      DYNAMICTEXT heading (Content: 'Main Content', RenderMode: H3)
    }
    COLUMN colSide (DesktopWidth: 4) {
      DYNAMICTEXT sideHeading (Content: 'Sidebar', RenderMode: H3)
    }
  }
}
```

### DATAGRID Widget

Display list of objects using DataGrid widget:

```sql
DATAGRID gridName (
  DataSource: DATABASE FROM Module.Entity WHERE [condition] SORT BY AttributeName ASC|DESC,
  Selection: Single|Multiple|None
) {
  COLUMN colName (Attribute: AttributeName, Caption: 'Label')
}
```

**Column Properties:**

| Property | Values | Default | Description |
|----------|--------|---------|-------------|
| `Attribute` | attribute name | (required) | Attribute binding |
| `Caption` | string | attribute name | Column header text |
| `Alignment` | `left`, `center`, `right` | `left` | Text alignment |
| `WrapText` | `true`, `false` | `false` | Wrap text in cell |
| `Sortable` | `true`, `false` | `true` (if attribute), `false` (if not) | Can sort column |
| `Resizable` | `true`, `false` | `true` | Can resize column |
| `Draggable` | `true`, `false` | `true` | Can reorder column |
| `Hidable` | `yes`, `hidden`, `no` | `yes` | Can hide column |
| `ColumnWidth` | `autoFill`, `autoFit`, `manual` | `autoFill` | Column width mode |
| `Size` | integer (px) | `1` | Width in pixels (when `ColumnWidth: manual`) |
| `Visible` | expression string | `true` | Conditional visibility (use page variables, NOT `$currentObject`) |
| `DynamicCellClass` | expression string | (empty) | Dynamic CSS class per cell |
| `Tooltip` | text string | (empty) | Cell tooltip text |

Only non-default column properties appear in `DESCRIBE PAGE` output.

```sql
COLUMN colPrice (
  Attribute: Price, Caption: 'Unit Price',
  Alignment: right, WrapText: true,
  Sortable: false, Resizable: false,
  Hidable: hidden,
  ColumnWidth: manual, Size: 150,
  DynamicCellClass: 'if($currentObject/Price > 100) then ''highlight'' else '''' ',
  Tooltip: 'Price in USD'
)
```

**Custom Content Columns (EXPERIMENTAL):**

Columns can contain nested widgets instead of attribute bindings. This feature is experimental and may show CE0463 "widget definition changed" warnings in Studio Pro:

```sql
COLUMN colActions (Caption: 'Actions') {
  ACTIONBUTTON btnView (Caption: 'View', Action: CLOSE_PAGE)
}
```

> **Note:** Custom content columns work at the syntax level but may require manual widget update in Studio Pro due to complex BSON structure requirements.

**Supported Datasource Types:**

| Syntax | Description |
|--------|-------------|
| `DataSource: DATABASE FROM Module.Entity` | Direct database query |
| `DataSource: $Variable` | Variable bound (requires DATAVIEW parent with entity) |
| `DataSource: MICROFLOW Module.GetData()` | Microflow datasource |
| `DataSource: SELECTION widgetName` | Listen to selection from another widget |

**With WHERE and SORT BY (inline in DataSource):**
```sql
DATAGRID dgActive (
  DataSource: DATABASE FROM Module.Product WHERE [IsActive = true] SORT BY Name ASC
) {
  COLUMN colName (Attribute: Name, Caption: 'Name')
  COLUMN colPrice (Attribute: Price, Caption: 'Price')
}
```

**Complex WHERE conditions:**
```sql
DATAGRID dgFiltered (
  DataSource: DATABASE FROM Module.Product
    WHERE [IsActive = true AND contains(Code, 'a') AND Price > 10] OR [Stock < 2]
    SORT BY Name ASC, Price DESC
) {
  COLUMN colName (Attribute: Name, Caption: 'Name')
}
```

**Paging Properties:**

| Property | Values | Default | Description |
|----------|--------|---------|-------------|
| `PageSize` | Any positive integer | 20 | Number of rows per page |
| `Pagination` | `buttons`, `virtualScrolling`, `loadMore` | `buttons` | Paging mode |
| `PagingPosition` | `bottom`, `top`, `both` | `bottom` | Position of paging controls |
| `ShowPagingButtons` | `always`, `auto` | `always` | When to show paging buttons |

```sql
-- Paging buttons above and below, 25 rows per page
DATAGRID dgProducts (
  DataSource: DATABASE Module.Product,
  PageSize: 25,
  PagingPosition: both
) {
  COLUMN colName (Attribute: Name, Caption: 'Name')
}

-- Virtual scrolling (no paging buttons)
DATAGRID dgLargeList (
  DataSource: DATABASE Module.Product,
  PageSize: 50,
  Pagination: virtualScrolling
) {
  COLUMN colName (Attribute: Name, Caption: 'Name')
}
```

Only non-default paging properties appear in `DESCRIBE PAGE` output.

### DATAVIEW Widget

Display single object with nested input widgets:

```sql
DATAVIEW dvName (DataSource: $VariableName) {
  -- Nested input widgets
  TEXTBOX txtName (Label: 'Name', Attribute: Name)
  TEXTAREA txtDescription (Label: 'Description', Attribute: Description)

  FOOTER footer1 {
    ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)
    ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
  }
}
```

### Input Widgets

Input widgets must be inside a DATAVIEW context. Use `Attribute:` to bind to attributes:

**TEXTBOX** - Single-line text input:
```sql
TEXTBOX txtName (Label: 'Label', Attribute: AttributeName)
```

**TEXTAREA** - Multi-line text input:
```sql
TEXTAREA txtDescription (Label: 'Description', Attribute: Description)
```

**CHECKBOX** - Boolean checkbox:
```sql
CHECKBOX cbActive (Label: 'Active', Attribute: IsActive)
```

**RADIOBUTTONS** - Boolean or enum selection:
```sql
RADIOBUTTONS rbStatus (Label: 'Status', Attribute: Status)
```

**DATEPICKER** - Date/time selection:
```sql
DATEPICKER dpCreated (Label: 'Created Date', Attribute: CreatedDate)
```

**COMBOBOX** - Combo box (pluggable widget):
```sql
-- Enumeration mode (attribute is an enum type):
COMBOBOX cbCountry (Label: 'Country', Attribute: Country)

-- Association mode (Attribute = association, DataSource = target entity, CaptionAttribute = display attr):
COMBOBOX cmbCustomer (Label: 'Customer', Attribute: Order_Customer, DataSource: DATABASE MyModule.Customer, CaptionAttribute: Name)
```

### DataView with Form Layout

```sql
DATAVIEW dataView1 (DataSource: $Customer) {
  TEXTBOX txtName (Label: 'Name', Attribute: Name)
  TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
  TEXTAREA txtAddress (Label: 'Address', Attribute: Address)
  COMBOBOX cbStatus (Label: 'Status', Attribute: Status)
  CHECKBOX cbActive (Label: 'Active', Attribute: IsActive)
  DATEPICKER dpCreated (Label: 'Created', Attribute: CreateDate)

  FOOTER footer1 {
    ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)
    ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
  }
}
```

### GALLERY Widget

Display items in card layout with selection and responsive columns:

```sql
GALLERY galleryName (
  DataSource: DATABASE FROM Module.Entity SORT BY Name ASC,
  Selection: Single|Multiple|None,
  DesktopColumns: 3,
  TabletColumns: 2,
  PhoneColumns: 1
) {
  TEMPLATE template1 {
    DYNAMICTEXT name (Content: '{1}', ContentParams: [{1} = Name], RenderMode: H4)
    DYNAMICTEXT email (Content: '{1}', ContentParams: [{1} = Email])
  }
}
```

**With Filter:**
```sql
GALLERY productGallery (DataSource: DATABASE Module.Product, Selection: Single) {
  FILTER filter1 {
    TEXTFILTER searchName (Attribute: Name)
  }
  TEMPLATE template1 {
    DYNAMICTEXT prodName (Content: '{1}', ContentParams: [{1} = Name], RenderMode: H4)
    DYNAMICTEXT prodCode (Content: 'SKU: {1}', ContentParams: [{1} = Code])
  }
}
```

### Filter Widgets

Filter widgets are used inside GALLERY FILTER containers to enable search/filtering:

**TEXTFILTER** - Text search filter:
```sql
-- Simple binding to single attribute
TEXTFILTER searchName (Attribute: Name)

-- Multiple attributes with explicit list
TEXTFILTER textFilter1 (Attributes: [Module.Entity.Name, Module.Entity.Code, Module.Entity.Description])

-- With filter type
TEXTFILTER textFilter1 (Attributes: [Module.Entity.Description], FilterType: startsWith)
```

**FilterType Options:**
- `contains` (default) - Matches if attribute contains text
- `startsWith` - Matches if attribute starts with text
- `endsWith` - Matches if attribute ends with text
- `equal` - Exact match

**NUMBERFILTER** - Numeric range filter:
```sql
NUMBERFILTER priceFilter (Attributes: [Module.Entity.Price])
```

**DATEFILTER** - Date range filter:
```sql
DATEFILTER dateFilter (Attributes: [Module.Entity.CreateDate])
```

**DROPDOWNFILTER** - Dropdown selection filter:
```sql
DROPDOWNFILTER statusFilter (Attributes: [Module.Entity.Status])
```

### NAVIGATIONLIST Widget

Create a menu with action items:

```sql
NAVIGATIONLIST navName {
  ITEM itemEdit (Caption: 'Edit', Action: SHOW_PAGE Module.EditPage(Entity: $EntityParameter))
  ITEM itemDelete (Caption: 'Delete', Action: DELETE)
  ITEM itemBack (Caption: 'Back', Action: CLOSE_PAGE)
}
```

### SNIPPETCALL Widget

Embed a reusable snippet:

```sql
-- Simple snippet call
SNIPPETCALL snippetName (Snippet: Module.SnippetName)

-- With parameters
SNIPPETCALL actions (Snippet: Module.EntityActions, Params: {Entity: $currentObject})
```

### IMAGE / STATICIMAGE / DYNAMICIMAGE Widgets

Display images on pages:

```sql
-- Static image (from image gallery) - IMAGE and STATICIMAGE are equivalent
IMAGE imgLogo (Width: 200, Height: 100)
STATICIMAGE imgBanner (Width: 400, Height: 120)

-- Dynamic image (from entity data source, e.g. inside a DataView)
DYNAMICIMAGE imgProduct (Width: 300, Height: 200)

-- Image without explicit dimensions (responsive by default)
IMAGE imgIcon
```

**Properties:** `Width: integer`, `Height: integer`, `Class: 'css'`, `Style: 'css'`

### CONTAINER / CUSTOMCONTAINER Widgets

Generic container for grouping widgets. `CUSTOMCONTAINER` is an alias for `CONTAINER` (both map to `Forms$DivContainer`):

```sql
-- Basic container with CSS class
CONTAINER card1 (Class: 'card', Style: 'padding: 16px;') {
  DYNAMICTEXT title (Content: 'Card Title', RenderMode: H4)
  DYNAMICTEXT body (Content: 'Card body content')
}

-- Container with design properties
CONTAINER spaced1 (DesignProperties: ['Spacing top': 'Large', 'Full width': ON]) {
  DYNAMICTEXT text1 (Content: 'Spaced full-width content')
}

-- Nested containers with combined styling
CUSTOMCONTAINER outer1 (Class: 'section') {
  CONTAINER inner1 (Class: 'card', DesignProperties: ['Spacing top': 'Medium']) {
    DYNAMICTEXT text1 (Content: 'Nested content')
  }
}
```

### FOOTER Widget

Container for form action buttons:

```sql
FOOTER footerName {
  ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)
  ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
}
```

### HEADER Widget

Container for header content:

```sql
HEADER headerName {
  DYNAMICTEXT title (Content: 'Form Title', RenderMode: H3)
}
```

### CONTROLBAR Widget

Control bar for data widgets:

```sql
CONTROLBAR controlBar1 {
  ACTIONBUTTON btnNew (Caption: 'New', Action: CREATE_OBJECT Module.Entity THEN SHOW_PAGE Module.EditPage, ButtonStyle: Primary)
}
```

## Complete Examples

### Customer Edit Page

```sql
CREATE OR REPLACE PAGE CRM.CustomerEdit
(
  Params: { $Customer: CRM.Customer },
  Title: 'Edit Customer',
  Layout: Atlas_Core.PopupLayout
)
{
  DATAVIEW dvCustomer (DataSource: $Customer) {
    TEXTBOX txtName (Label: 'Name', Attribute: Name)
    TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
    TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
    CHECKBOX cbActive (Label: 'Active', Attribute: IsActive)

    FOOTER footer1 {
      ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)
      ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
    }
  }
}
```

### Order Overview Page

```sql
CREATE PAGE Orders.OrderOverview
(
  Title: 'Orders',
  Layout: Atlas_Core.Atlas_Default
)
{
  LAYOUTGRID mainGrid {
    ROW row1 {
      COLUMN col1 (DesktopWidth: 12) {
        DYNAMICTEXT heading (Content: 'Order Overview', RenderMode: H2)
      }
    }
    ROW row2 {
      COLUMN col2 (DesktopWidth: 12) {
        DATAGRID dgOrders (DataSource: DATABASE FROM Orders.Order SORT BY OrderDate DESC) {
          COLUMN colNumber (Attribute: OrderNumber, Caption: 'Order #')
          COLUMN colDate (Attribute: OrderDate, Caption: 'Date')
          COLUMN colTotal (Attribute: TotalAmount, Caption: 'Total')
        }
      }
    }
  }
}
```

### Master-Detail Page

```sql
CREATE PAGE CRM.Customer_MasterDetail
(
  Title: 'Customer Management',
  Layout: Atlas_Core.Atlas_Default
)
{
  LAYOUTGRID mainGrid {
    ROW row1 {
      -- Master list (left column)
      COLUMN colMaster (DesktopWidth: 4) {
        DYNAMICTEXT heading (Content: 'Customers', RenderMode: H3)
        GALLERY customerList (DataSource: DATABASE FROM CRM.Customer SORT BY Name ASC, Selection: Single) {
          TEMPLATE template1 {
            DYNAMICTEXT name (Content: '{1}', ContentParams: [{1} = Name], RenderMode: H4)
            DYNAMICTEXT email (Content: '{1}', ContentParams: [{1} = Email])
          }
        }
      }

      -- Detail form (right column)
      COLUMN colDetail (DesktopWidth: 8) {
        DATAVIEW customerDetail (DataSource: SELECTION customerList) {
          DYNAMICTEXT detailHeading (Content: 'Customer Details', RenderMode: H3)
          TEXTBOX txtName (Label: 'Name', Attribute: Name)
          TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
          TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)

          FOOTER footer1 {
            ACTIONBUTTON btnSave (Caption: 'Save', Action: SAVE_CHANGES, ButtonStyle: Primary)
            ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
          }
        }
      }
    }
  }
}
```

## Modifying Existing Pages

To make targeted changes to an existing page (change a label, add a field, remove a widget), use `ALTER PAGE` instead of `CREATE OR REPLACE PAGE`. ALTER PAGE modifies the widget tree in-place, preserving properties that MDL doesn't model.

```sql
-- Change a button caption and add a field
ALTER PAGE Module.Customer_Edit {
  SET Caption = 'Save & Close' ON btnSave;
  INSERT AFTER txtEmail {
    TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
  }
};
```

See the dedicated skill file: [ALTER PAGE/SNIPPET](./alter-page.md)

## Known Limitations

The following features are NOT implemented in mxcli and require manual configuration in Studio Pro:

| Feature | Workaround |
|---------|------------|
| `DataSource: ASSOCIATION` | Use `DATABASE` with WHERE constraint, or microflow datasource |
| Nested dataviews filtering by parent | Use microflow datasource or configure in Studio Pro |
| Complex conditional visibility | Configure visibility rules in Studio Pro |
| Widget-level security | Configure access rules in Studio Pro |

### Runtime Pitfalls

> **Empty CONTAINER crashes at runtime.** A CONTAINER with no child widgets compiles and builds successfully but crashes when the page loads with "Did not expect an argument to be undefined". Always include at least one child widget:
> ```sql
> -- Wrong: crashes at runtime
> CONTAINER spacer1 (Style: 'height: 6px;')
>
> -- Correct: include a child (even a space)
> CONTAINER spacer1 (Style: 'height: 6px;') {
>   DYNAMICTEXT spacerText (Content: ' ', RenderMode: Paragraph)
> }
> ```

> **`Content: ''` (empty string) fails MxBuild.** An empty Content on DYNAMICTEXT causes a misleading error: "Place holder index 1 is greater than 0, the number of parameter(s)." Use a single space instead:
> ```sql
> -- Wrong: MxBuild error
> DYNAMICTEXT spacer (Content: '')
>
> -- Correct: use a space
> DYNAMICTEXT spacer (Content: ' ')
> ```

**Script Execution Note:** Script execution stops on the first error. If a page fails to create (e.g., invalid widget syntax), earlier statements in the script will have already been committed. Plan scripts with uncertain syntax in phases.

## Tips

1. **OR REPLACE**: Use to recreate existing pages
2. **Widget Names**: Required - use descriptive camelCase names
3. **Layout Requirement**: Layout must exist in the project
4. **Nesting**: Use `{ }` blocks for all widget children
5. **Properties**: Use `(Key: value)` syntax for all widget properties
6. **Bindings**: Use `Attribute:` for attributes, `DataSource:` for data, `Action:` for buttons

## Related Commands

- `ALTER PAGE Module.PageName { ... }` - Modify page widgets in-place (SET, INSERT, DROP, REPLACE)
- `ALTER SNIPPET Module.SnippetName { ... }` - Modify snippet widgets in-place
- `DESCRIBE PAGE Module.PageName` - View page source in MDL format (shows Class, Style, DesignProperties)
- `DESCRIBE SNIPPET Module.SnippetName` - View snippet source in MDL format
- `SHOW PAGES [IN Module]` - List all pages
- `SHOW WIDGETS [WHERE ...] [IN Module]` - Discover widgets across pages/snippets
- `UPDATE WIDGETS SET ... WHERE ... [DRY RUN]` - Bulk update widget properties (see below)
- `DROP PAGE Module.PageName` - Delete a page

### Bulk Widget Updates

Use `UPDATE WIDGETS` to change properties across many widgets at once:

```sql
-- Preview changes first (always use DRY RUN)
UPDATE WIDGETS SET 'Class' = 'card' WHERE WidgetType LIKE '%Container%' IN MyModule DRY RUN;

-- Apply changes
UPDATE WIDGETS SET 'showLabel' = false WHERE WidgetType LIKE '%combobox%';

-- Multiple properties
UPDATE WIDGETS SET 'Class' = 'btn-lg', 'Style' = 'margin-top: 8px;' WHERE WidgetType LIKE '%ActionButton%';
```

### PLUGGABLEWIDGET — Any Pluggable Widget with Explicit Properties

Create any pluggable widget and set its properties explicitly by propertyKey:

```sql
PLUGGABLEWIDGET 'com.mendix.widget.custom.switch.Switch' switch1 (booleanAttribute: IsActive)
```

The widget is created from its embedded template. Properties are set by their widget XML property key names. Attribute names are resolved using the current entity context (DataView parent).

**Syntax:**
```sql
PLUGGABLEWIDGET 'widgetID' widgetName [(key1: value1, key2: value2, ...)]
```

**Property value types:**
| Value | Interpreted as | Example |
|-------|---------------|---------|
| Short name (no dots) | Attribute ref (resolved via entity context) | `booleanAttribute: IsActive` |
| Qualified path (2+ dots) | Attribute ref (fully qualified) | `booleanAttribute: Module.Entity.IsActive` |
| Other string | Primitive value | `animationType: bar` |

**Finding property keys:**
- Run `mxcli widget docs -p app.mpr` to generate widget documentation
- Check `.claude/skills/widgets/` for per-widget property reference
- Property keys match the `key` attribute in the widget's XML definition

**Examples:**
```sql
-- Boolean toggle (inside DATAVIEW with entity context)
PLUGGABLEWIDGET 'com.mendix.widget.custom.switch.Switch' sw1 (booleanAttribute: IsActive)

-- Star rating bound to attribute
PLUGGABLEWIDGET 'com.mendix.widget.custom.starrating.StarRating' rating1 (rateAttribute: Score)

-- Widget with default properties only
PLUGGABLEWIDGET 'com.mendix.widget.custom.badge.Badge' badge1

-- Display widgets (no entity context needed)
PLUGGABLEWIDGET 'com.mendix.widget.custom.progressbar.ProgressBar' progBar1
PLUGGABLEWIDGET 'com.mendix.widget.web.accordion.Accordion' accordion1
```

**DESCRIBE round-trip:** `DESCRIBE PAGE` outputs generic pluggable widgets as `PLUGGABLEWIDGET 'widgetID' name (key: value, ...)`, which can be re-executed as valid MDL.

**Legacy syntax:** `CUSTOMWIDGET name (WidgetType: 'widgetID')` is still supported but does not support explicit properties.

## See Also

- [Overview Pages](./overview-pages.md) - CRUD page patterns
- [Master-Detail Pages](./master-detail-pages.md) - Selection binding pattern
