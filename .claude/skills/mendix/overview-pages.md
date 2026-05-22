---
name: overview-pages
description: Use when creating CRUD overview pages in Mendix — overview page
             datagrid crud new edit delete controlbar actionbutton column
             filter create_object show_page delete_object currentObject
---

## When to Use This Skill

- Creating an overview page that lists entities in a DataGrid2
- Adding New / Edit / Delete buttons (full CRUD flow)
- Adding a navigation snippet (reusable menu across overview pages)
- Creating the companion edit page opened from the DataGrid
- Adding column-level filters to searchable columns

## Checklist

- [ ] Ensure the entity and its enumerations exist (`create persistent entity ...`)
- [ ] Create the edit/popup page first (it's referenced by the overview)
- [ ] Create the overview page with DataGrid2 and `controlbar` New button
- [ ] Add `column colActions` with `ShowContentAs: customContent` for Edit and Delete
- [ ] Add column-level filters on searchable columns (optional — see `mendix:datagrid2-filters`)
- [ ] Run `./bin/mxcli check script.mdl` to validate
- [ ] Confirm `Atlas_Core.PopupLayout` is available in the project (`show pages in Atlas_Core`)

## Quick Syntax Reference

| Element | Syntax |
|---------|--------|
| New object + open edit page | `action: create_object Module.Entity then show_page Module.EditPage` |
| Edit row | `action: show_page Module.EditPage (Entity: $currentObject)` |
| Delete row | `action: delete_object` |
| Current row object | `$currentObject` (inside `column {}` or `controlbar {}`) |
| Custom content column | `column colName (caption: 'X', ShowContentAs: customContent) { ... }` |

## Core Patterns

### Pattern 1: Minimal CRUD Overview

```sql
-- Step 1: Edit page (popup, receives entity as parameter)
create page MyMod.Product_Edit (
  params: { $Product: MyMod.Product },
  title: 'Edit Product',
  layout: Atlas_Core.PopupLayout,
  folder: 'Products'
) {
  dataview dvProduct (datasource: $Product) {
    textbox txtName  (label: 'Name',  attribute: Name)
    textbox txtPrice (label: 'Price', attribute: Price)
    footer footer1 {
      actionbutton btnSave   (caption: 'Save',   action: save_changes close_page,   buttonstyle: primary)
      actionbutton btnCancel (caption: 'Cancel', action: cancel_changes close_page)
    }
  }
}

-- Step 2: Overview page with DataGrid2 CRUD
create page MyMod.Product_Overview (
  title: 'Products',
  layout: Atlas_Core.Atlas_Default,
  url: 'products',
  folder: 'Products'
) {
  datagrid dgProducts (
    datasource: database from MyMod.Product sort by Name asc,
    PageSize: 25,
    PagingPosition: both
  ) {
    controlbar cb1 {
      actionbutton btnNew (
        caption: 'New Product',
        action: create_object MyMod.Product then show_page MyMod.Product_Edit,
        buttonstyle: primary
      )
    }
    column colName     (attribute: Name,     caption: 'Name')
    column colPrice    (attribute: Price,    caption: 'Price', Alignment: right)
    column colIsActive (attribute: IsActive, caption: 'Active')
    column colActions (caption: 'Actions', ShowContentAs: customContent) {
      actionbutton btnEdit (
        caption: 'Edit',
        action: show_page MyMod.Product_Edit (Product: $currentObject),
        buttonstyle: default
      )
      actionbutton btnDelete (
        caption: 'Delete',
        action: delete_object,
        buttonstyle: danger
      )
    }
  }
}
```

### Pattern 2: Overview with column filters and sorting

```sql
create page MyMod.Order_Overview (
  title: 'Orders',
  layout: Atlas_Core.Atlas_Default,
  url: 'orders',
  folder: 'Orders'
) {
  datagrid dgOrders (
    datasource: database from MyMod.Order sort by OrderDate desc,
    PageSize: 20,
    PagingPosition: both
  ) {
    controlbar cb1 {
      actionbutton btnNew (
        caption: 'New Order',
        action: create_object MyMod.Order then show_page MyMod.Order_Edit,
        buttonstyle: primary
      )
    }
    -- Column-level filters
    column colNumber (attribute: OrderNumber, caption: 'Order #',
                      ColumnWidth: manual, Size: 130) {
      textfilter fNum (filtertype: startsWith)
    }
    column colCustomer (attribute: CustomerName, caption: 'Customer') {
      textfilter fCust
    }
    column colAmount (attribute: TotalAmount, caption: 'Amount',
                      Alignment: right, ColumnWidth: manual, Size: 110) {
      numberfilter fAmt
    }
    column colDate (attribute: OrderDate, caption: 'Date',
                    ColumnWidth: manual, Size: 130) {
      datefilter fDate
    }
    column colStatus (attribute: Status, caption: 'Status',
                      ColumnWidth: manual, Size: 100) {
      dropdownfilter fStatus
    }
    column colActions (caption: 'Actions', ShowContentAs: customContent,
                       ColumnWidth: manual, Size: 120) {
      actionbutton btnEdit (
        caption: 'Edit',
        action: show_page MyMod.Order_Edit (Order: $currentObject),
        buttonstyle: default
      )
      actionbutton btnDelete (caption: 'Delete', action: delete_object, buttonstyle: danger)
    }
  }
}
```

### Pattern 3: Overview with microflow datasource

```sql
create microflow MyMod.DSO_GetPendingOrders ()
  returns list of MyMod.Order
begin
  retrieve $Orders from MyMod.Order
    where [Status = MyMod.OrderStatus.Pending and DueDate < addDays('[%CurrentDateTime%]', 7)]
    sort by DueDate asc;
  return $Orders;
end;
/

create page MyMod.PendingOrders_Overview (
  title: 'Pending Orders',
  layout: Atlas_Core.Atlas_Default,
  url: 'pending-orders'
) {
  datagrid dgOrders (
    datasource: microflow MyMod.DSO_GetPendingOrders,
    PageSize: 25
  ) {
    column colNumber  (attribute: OrderNumber, caption: 'Order #')
    column colDue     (attribute: DueDate,     caption: 'Due Date')
    column colActions (caption: 'Actions', ShowContentAs: customContent) {
      actionbutton btnEdit (
        caption: 'Edit',
        action: show_page MyMod.Order_Edit (Order: $currentObject),
        buttonstyle: default
      )
    }
  }
}
```

### Pattern 4: Navigation snippet (reusable menu)

```sql
create snippet MyMod.NavMenu (folder: 'Navigation') {
  navigationlist navMenu {
    item itemProducts (action: show_page MyMod.Product_Overview) {
      dynamictext txtProducts (content: 'Products')
    }
    item itemOrders (action: show_page MyMod.Order_Overview) {
      dynamictext txtOrders (content: 'Orders')
    }
  }
}

-- Use in any overview page
create page MyMod.Dashboard (
  title: 'Dashboard',
  layout: Atlas_Core.Atlas_Default,
  url: 'dashboard'
) {
  layoutgrid lg1 {
    row row1 {
      column colNav  (desktopwidth: 3) { snippetcall navSnippet (snippet: MyMod.NavMenu) }
      column colMain (desktopwidth: 9) { dynamictext dtWelcome (content: 'Welcome', rendermode: H2) }
    }
  }
}
```

## Known Limitations

- Column-level filter widgets (`textfilter`, `dropdownfilter`, etc. in `column {}`) — work correctly
- Filter bar in controlbar (`textfilter (attributes:[...])` in `controlbar {}`) — fixed in commit fc1b6ee3
- `filtertype:` property on column filters — fixed in commit fc1b6ee3
- `ShowContentAs: customContent` action buttons and column filter widgets must be in **separate** columns
- Page parameters (`params:`) require Mendix 11.0+; use `-- @version: 11.0+` guard in MDL scripts

## Validation

```bash
./bin/mxcli check script.mdl
./bin/mxcli check script.mdl -p path/to/app.mpr --references
```

## See Also

- `mendix:create-page` — Full widget syntax reference
- `mendix:datagrid2-filters` — Column filter patterns
- `mendix:page-data-design` — Datasource strategy
- `mendix:master-detail-pages` — Selection binding and master-detail layouts
