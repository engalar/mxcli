# ALTER PAGE / ALTER SNIPPET

## Synopsis

```sql
ALTER PAGE module.Name {
    operations
}

ALTER SNIPPET module.Name {
    operations
}
```

Where each operation is one of:

```sql
-- Set a property on a widget
SET property = value ON widgetName;
SET ( property1 = value1, property2 = value2 ) ON widgetName;

-- Set a page-level property (no ON clause)
SET Title = 'New Title';

-- Set a pluggable widget property (quoted name)
SET 'propertyName' = value ON widgetName;

-- Insert widgets before or after a target
INSERT BEFORE widgetName { widget_definitions };
INSERT AFTER widgetName { widget_definitions };

-- Remove widgets
DROP WIDGET widgetName1, widgetName2;

-- Replace a widget with new widgets
REPLACE widgetName WITH { widget_definitions };

-- Add a page variable
ADD Variables $name : type = 'expression';

-- Drop a page variable
DROP Variables $name;

-- Change page layout (auto-map placeholders by name)
SET Layout = module.LayoutName;

-- Change layout with explicit placeholder mapping
SET Layout = module.LayoutName MAP (OldPlaceholder AS NewPlaceholder);
```

## Description

Modifies an existing page or snippet in-place without requiring a full `CREATE OR REPLACE`. This is useful when you need to make targeted changes to a page while preserving the rest of its widget tree, including any unsupported or third-party widget types that cannot be round-tripped through MDL.

ALTER PAGE works directly on the raw BSON widget tree, so it preserves widgets and properties that MDL does not explicitly support.

### SET

Sets one or more properties on a named widget. Widget names are assigned during `CREATE PAGE` and can be discovered with `DESCRIBE PAGE`.

Supported standard properties: `Caption`, `Label`, `ButtonStyle`, `Class`, `Style`, `Editable`, `Visible`, `Name`.

For page-level properties (like `Title`), omit the `ON` clause.

For pluggable widget properties, use quoted property names (e.g., `'showLabel'`).

### INSERT BEFORE / INSERT AFTER

Inserts new widgets immediately before or after a named widget within its parent container. The new widgets use the same syntax as in `CREATE PAGE`.

### DROP WIDGET

Removes one or more widgets by name. The widget and all its children are removed from the tree.

### REPLACE

Replaces a widget (and its entire subtree) with one or more new widgets.

### SET Layout

Changes the page's layout without rebuilding the widget tree. Placeholder names are auto-mapped by default. If the new layout has different placeholder names, use `MAP` to specify the mapping.

Not supported for snippets (snippets don't have layouts).

### ADD Variables / DROP Variables

Adds or removes page-level variables. Page variables are typed values with default expressions, available for conditional visibility and dynamic behavior.

## Parameters

`module.Name`
:   The qualified name of the page or snippet to modify. Must already exist.

`widgetName`
:   The name of a widget within the page. Use `DESCRIBE PAGE` to list widget names.

`property`
:   A widget property name. Standard properties are unquoted. Pluggable widget properties use single quotes.

`value`
:   The new property value. Strings are quoted. Booleans and enumerations are unquoted.

## Examples

Change button caption and style:

```sql
ALTER PAGE Sales.Order_Edit {
    SET (Caption = 'Save & Close', ButtonStyle = Success) ON btnSave;
};
```

Remove an unused widget and add a new field:

```sql
ALTER PAGE Sales.Order_Edit {
    DROP WIDGET txtUnused;
    INSERT AFTER txtEmail {
        TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
    }
};
```

Replace a widget with multiple new widgets:

```sql
ALTER PAGE Sales.Order_Edit {
    REPLACE txtOldField WITH {
        TEXTBOX txtFirstName (Label: 'First Name', Attribute: FirstName)
        TEXTBOX txtLastName (Label: 'Last Name', Attribute: LastName)
    }
};
```

Set a page-level property:

```sql
ALTER PAGE Sales.Order_Edit {
    SET Title = 'Edit Order Details';
};
```

Set a pluggable widget property:

```sql
ALTER PAGE Sales.Order_Edit {
    SET 'showLabel' = false ON cbStatus;
};
```

Add and drop page variables:

```sql
ALTER PAGE Sales.Order_Edit {
    ADD Variables $showAdvanced : Boolean = 'false';
};

ALTER PAGE Sales.Order_Edit {
    DROP Variables $showAdvanced;
};
```

Change page layout (preserves all widgets):

```sql
ALTER PAGE Sales.Order_Edit {
    SET Layout = Atlas_Core.Atlas_Default;
};
```

Change layout with explicit placeholder mapping:

```sql
ALTER PAGE Sales.Order_Edit {
    SET Layout = Atlas_Core.Atlas_SideBar MAP (Main AS Content, Extra AS Sidebar);
};
```

Modify a snippet:

```sql
ALTER SNIPPET MyModule.NavMenu {
    SET Caption = 'Dashboard' ON btnHome;
    INSERT AFTER btnHome {
        ACTIONBUTTON btnReports (Caption: 'Reports', Action: PAGE MyModule.Reports)
    }
};
```

Combined operations in a single ALTER:

```sql
ALTER PAGE MyModule.Customer_Edit {
    SET Title = 'Customer Details';
    SET (Caption = 'Update', ButtonStyle = Primary) ON btnSave;
    DROP WIDGET txtObsolete;
    INSERT BEFORE txtEmail {
        TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
    }
    INSERT AFTER txtEmail {
        COMBOBOX cbCategory (Label: 'Category', Attribute: Category)
    }
};
```

## See Also

[CREATE PAGE](create-page.md), [CREATE SNIPPET](create-snippet.md), [DESCRIBE PAGE](/reference/query/describe-page.md), [DROP PAGE](drop-page.md)
