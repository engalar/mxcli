---
name: master-detail-pages
description: Use when building master-detail layouts — gallery selection dataview
             listen to widget selection binding datasource association filter bar
             attributes multi-attribute search master detail split layout
---

## When to Use This Skill

- Building a side-by-side list + detail panel (master-detail layout)
- Using `datasource: selection widgetName` to sync a DataView with a Gallery or DataGrid row
- Adding a Gallery filter bar with multi-attribute text search
- Nesting a DataGrid inside a DataView via association path
- Displaying related objects when a parent object is selected

## Checklist

- [ ] Enable `selection: single` (or `multiple`) on the source widget (Gallery/DataGrid)
- [ ] Set the detail DataView datasource to `selection sourceWidgetName`
- [ ] Both master and detail widgets must be on the same page (selection binding is page-scoped)
- [ ] For Gallery filter bar: use `filter filterBarName { textfilter ... }` inside the gallery
- [ ] Run `./bin/mxcli check script.mdl` to validate

## Quick Syntax Reference

| Element | Syntax | Notes |
|---------|--------|-------|
| Enable selection | `selection: single` or `selection: multiple` on Gallery/DataGrid | Required for `datasource: selection` |
| Listen to selection | `dataview dv (datasource: selection masterWidgetName)` | Widget names must match exactly |
| Association datasource | `datagrid dg (datasource: $Param/Module.Assoc/Module.Entity)` | Path from page param through association |
| Gallery filter bar | `gallery g1 (...) { filter fb { textfilter f1 (attributes: [...]) } template t1 { ... } }` | `attributes:` required for gallery filter bar |
| Multi-attribute OR search | `textfilter f1 (attributes: [Mod.Entity.A1, Mod.Entity.A2])` | Matches if ANY attribute contains text |

## Core Patterns

### Pattern 1: Gallery (left) + DataView detail (right)

Classic side-by-side master-detail:

```sql
create page MyMod.Customer_MasterDetail (
  title: 'Customers',
  layout: Atlas_Core.Atlas_Default,
  url: 'customers'
) {
  layoutgrid lg1 {
    row row1 {
      -- LEFT: master list with selection enabled
      column colMaster (desktopwidth: 4) {
        gallery custList (
          datasource: database from MyMod.Customer sort by Name asc,
          selection: single
        ) {
          template template1 {
            dynamictext txtName  (content: '{1}', contentparams: [{1} = Name],  rendermode: H4)
            dynamictext txtEmail (content: '{1}', contentparams: [{1} = Email])
          }
        }
      }
      -- RIGHT: detail panel listens to Gallery selection
      column colDetail (desktopwidth: 8) {
        dataview dvDetail (datasource: selection custList) {
          textbox txtName    (label: 'Name',    attribute: Name)
          textbox txtEmail   (label: 'Email',   attribute: Email)
          textbox txtPhone   (label: 'Phone',   attribute: Phone)
          footer footer1 {
            actionbutton btnSave   (caption: 'Save',   action: save_changes, buttonstyle: primary)
            actionbutton btnCancel (caption: 'Cancel', action: cancel_changes)
          }
        }
      }
    }
  }
}
```

### Pattern 2: DataGrid (top) + DataView detail (bottom)

Use when the list is tabular and the detail form sits below:

```sql
create page MyMod.Order_Split (
  title: 'Order Management',
  layout: Atlas_Core.Atlas_Default,
  url: 'order-management'
) {
  layoutgrid lg1 {
    row rowGrid {
      column colGrid (desktopwidth: 12) {
        datagrid dgOrders (
          datasource: database from MyMod.Order sort by OrderDate desc,
          selection: single,
          PageSize: 15
        ) {
          column colNum    (attribute: OrderNumber, caption: 'Order #')
          column colDate   (attribute: OrderDate,   caption: 'Date')
          column colStatus (attribute: Status,      caption: 'Status')
        }
      }
    }
    row rowDetail {
      column colDetail (desktopwidth: 12) {
        dataview dvOrder (datasource: selection dgOrders) {
          textbox txtNumber (label: 'Order #', attribute: OrderNumber)
          textbox txtAmount (label: 'Amount',  attribute: TotalAmount)
          combobox cmbStatus (label: 'Status', attribute: Status)
          footer footer1 {
            actionbutton btnSave   (caption: 'Save',   action: save_changes, buttonstyle: primary)
            actionbutton btnCancel (caption: 'Cancel', action: cancel_changes)
          }
        }
      }
    }
  }
}
```

### Pattern 3: Gallery filter bar with multi-attribute search

The `filter {}` container inside a Gallery enables a search bar above the cards.
`attributes: [A, B]` means the filter shows cards where ANY listed attribute contains the text (OR match).
Multiple filter widgets combine with AND logic.

```sql
create page MyMod.Product_Gallery (
  title: 'Products',
  layout: Atlas_Core.Atlas_Default,
  url: 'product-gallery'
) {
  gallery productGallery (
    datasource: database from MyMod.Product sort by Name asc,
    selection: single,
    DesktopColumns: 3,
    TabletColumns: 2,
    PhoneColumns: 1
  ) {
    filter filterBar {
      -- Single input searches Name, Code, AND Category (OR match within this widget)
      textfilter fSearch (
        attributes: [MyMod.Product.Name, MyMod.Product.Code, MyMod.Product.Category]
      )
      -- Additional AND filters
      dropdownfilter fActive   (attributes: [MyMod.Product.IsActive])
      numberfilter   fMinPrice (attributes: [MyMod.Product.Price])
    }
    template template1 {
      dynamictext txtName  (content: '{1}', contentparams: [{1} = Name],  rendermode: H4)
      dynamictext txtCode  (content: 'SKU: {1}', contentparams: [{1} = Code])
      dynamictext txtPrice (content: '${1}',     contentparams: [{1} = Price])
    }
  }
}
```

### Pattern 4: DataView (page param) + nested DataGrid via association

Parent page receives an entity; child DataGrid shows related objects via association path:

```sql
create page MyMod.Customer_Detail (
  params: { $Customer: MyMod.Customer },
  title: 'Customer Detail',
  layout: Atlas_Core.Atlas_Default,
  url: 'customer/{Customer/Name}'
) {
  dataview dvCustomer (datasource: $Customer, editable: false) {
    dynamictext dtName (content: '{1}', contentparams: [{1} = Name], rendermode: H2)

    datagrid dgOrders (
      datasource: $Customer/MyMod.Order_Customer/MyMod.Order,
      PageSize: 10
    ) {
      column colNum    (attribute: OrderNumber, caption: 'Order #') { textfilter fNum }
      column colDate   (attribute: OrderDate,   caption: 'Date')    { datefilter fDate }
      column colStatus (attribute: Status,      caption: 'Status')  { dropdownfilter fStat }
    }
  }
}
```

## Known Limitations

- ⚠️ **`datasource: selection` requires selection enabled** — source widget must have `selection: single` or `selection: multiple`; DataView shows nothing otherwise
- ⚠️ **Selection binding is page-scoped** — master and detail must be on the same page; cross-page selection is not supported
- ⚠️ **NPE lists + `Keep Selection`** — DataGrid2 `Keep Selection` does not work with NPE datasources; selection is lost on filter or refresh
- ✅ **Gallery `filter {}` with `attributes:[...]`** — multi-attribute text search (OR within the filter widget) works correctly
- ✅ **DataGrid2 filter bar** (`controlbar {} { textfilter (attributes:[...]) }`) — fixed in commit fc1b6ee3; now emits real filter BSON

## Validation

```bash
./bin/mxcli check script.mdl
./bin/mxcli check script.mdl -p path/to/app.mpr --references
```

## See Also

- `mendix:create-page` — Full widget syntax reference
- `mendix:overview-pages` — CRUD overview + DataGrid CRUD patterns
- `mendix:datagrid2-filters` — Column and filter bar filter widgets
- `mendix:page-data-design` — Datasource strategy
