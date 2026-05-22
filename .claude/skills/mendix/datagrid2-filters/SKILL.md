---
name: datagrid2-filters
description: Use when adding filter widgets to DataGrid2 — textfilter
             numberfilter dropdownfilter datefilter column filter bar
             filtertype attributes datasource pluggablewidget datagrid2
---

## When to Use This Skill

- Adding `textfilter` / `numberfilter` / `datefilter` / `dropdownfilter` inside a DataGrid2 column or controlbar
- Configuring a filter bar above the DataGrid2 grid (in `controlbar {}`)
- Setting `filtertype:` default comparison (startsWith, equal, contains, etc.)
- Configuring advanced filter properties: placeholder, adjustable, multiselect, clearable
- Building a Gallery filter bar with multi-attribute search

## Checklist

- [ ] Identify the attribute type for each column → auto-selects filter widget kind (see Quick Reference)
- [ ] Column-level: place filter widget inside `column {}` body
- [ ] Filter bar: place filter widgets inside `controlbar {}` with `attributes: [Module.Entity.Attr]`
- [ ] For advanced properties (placeholder, adjustable, multiselect): use `PLUGGABLEWIDGET`
- [ ] Run `./bin/mxcli check script.mdl` to validate MDL syntax
- [ ] Run `mx check app.mpr` to confirm no BSON errors in Studio Pro

## Quick Syntax Reference

### Auto-Type Selection (column-level — no `attributes:` needed)

| Attribute type | Filter widget auto-selected |
|---|---|
| String | `textfilter` |
| Integer / Long / Decimal / AutoNumber | `numberfilter` |
| DateTime | `datefilter` |
| Enumeration / Boolean | `dropdownfilter` |

### Column-Level Filter Syntax

```sql
-- Widget goes inside column {} body; auto-wired to column attribute
column colName (attribute: AttrName, caption: 'Label') {
  textfilter     filtName                        -- String
  numberfilter   filtName                        -- Numeric
  datefilter     filtName                        -- DateTime
  dropdownfilter filtName                        -- Enum/Boolean
}

-- With explicit filtertype (default comparison)
column colName (attribute: AttrName) {
  textfilter filtName (filtertype: startsWith)
}
```

### Filter Bar Syntax (in controlbar)

```sql
datagrid dg1 (datasource: database Module.Order) {
  controlbar filterBar {
    textfilter     fSearch (attributes: [Module.Order.OrderNumber, Module.Order.CustomerName])
    dropdownfilter fStatus (attributes: [Module.Order.Status])
    numberfilter   fAmt    (attributes: [Module.Order.TotalAmount])
    datefilter     fDate   (attributes: [Module.Order.OrderDate])
  }
  column colNum  (attribute: OrderNumber,  caption: 'Order #')
  column colStat (attribute: Status,       caption: 'Status')
}
```

### Gallery Filter Bar (multi-attribute search)

```sql
gallery g1 (datasource: database Module.Product, selection: single) {
  filter filterBar {
    textfilter     f1 (attributes: [Module.Product.Name, Module.Product.Code])
    dropdownfilter f2 (attributes: [Module.Product.IsActive])
  }
  template template1 {
    dynamictext txt1 (content: '{1}', contentparams: [{1} = Name], rendermode: H4)
  }
}
```

### Advanced Properties via PLUGGABLEWIDGET

```sql
-- Inside column {} body
column colName (attribute: Name) {
  pluggablewidget 'com.mendix.widget.web.datagridtextfilter.DatagridTextFilter' filtName (
    defaultFilter: 'startsWith',
    placeholder:   'Search…',
    adjustable:    'yes',
    applyAfterMs:  '300'
  )
}

-- Dropdown: multi-select + clearable
column colStatus (attribute: Status) {
  pluggablewidget 'com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter' fStatus (
    multiselect:         'yes',
    clearable:           'yes',
    emptyCaption:        'All',
    showSelectedItemsAs: 'labels',
    selectionMethod:     'checkbox'
  )
}
```

### PLUGGABLEWIDGET Widget IDs

| Filter | Widget ID |
|--------|-----------|
| Text | `com.mendix.widget.web.datagridtextfilter.DatagridTextFilter` |
| Number | `com.mendix.widget.web.datagridnumberfilter.DatagridNumberFilter` |
| Date | `com.mendix.widget.web.datagriddatefilter.DatagridDateFilter` |
| Dropdown | `com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter` |

## Core Patterns

### Pattern 1: Column-level filters — all four types

```sql
create page MyMod.Order_Overview (
  title: 'Orders',
  layout: Atlas_Core.Atlas_Default,
  url: 'orders'
) {
  datagrid dgOrders (
    datasource: database from MyMod.Order sort by OrderDate desc,
    PageSize: 25,
    PagingPosition: both
  ) {
    column colNumber   (attribute: OrderNumber,  caption: 'Order #')          { textfilter     fNum      }
    column colCustomer (attribute: CustomerName, caption: 'Customer')          { textfilter     fCust     }
    column colAmount   (attribute: TotalAmount,  caption: 'Amount',
                        Alignment: right)                                      { numberfilter   fAmount   }
    column colDate     (attribute: OrderDate,    caption: 'Date')              { datefilter     fDate     }
    column colStatus   (attribute: Status,       caption: 'Status')            { dropdownfilter fStatus   }
    column colActive   (attribute: IsActive,     caption: 'Active')            { dropdownfilter fActive   }
  }
}
```

### Pattern 2: filtertype — default comparison type

```sql
datagrid dgOrders (datasource: database MyMod.Order) {
  -- Prefix search: "ORD-2024-" matches all orders starting with that prefix
  column colCode (attribute: OrderNumber, caption: 'Order #') {
    textfilter fCode (filtertype: startsWith)
  }
  -- Exact match: useful for IDs and codes
  column colEmail (attribute: Email, caption: 'Email') {
    textfilter fEmail (filtertype: equal)
  }
  -- Minimum threshold: show orders above entered amount
  column colAmount (attribute: TotalAmount, caption: 'Min Amount', Alignment: right) {
    numberfilter fAmt (filtertype: greaterEqual)
  }
}
```

Valid `filtertype` values: `contains` (default) | `startsWith` | `endsWith` | `equal` |
`notEqual` | `empty` | `notEmpty` | `greater` | `greaterEqual` | `smaller` | `smallerEqual`

### Pattern 3: DataGrid2 filter bar (in controlbar)

Filter bar sits above the data rows. Each widget must specify `attributes: [Module.Entity.Attr]`.
Multiple widgets combine with AND logic. Single `textfilter` with multiple attributes uses OR within that widget.

```sql
create page MyMod.Order_Overview (
  title: 'Orders',
  layout: Atlas_Core.Atlas_Default,
  url: 'orders'
) {
  datagrid dgOrders (
    datasource: database from MyMod.Order sort by OrderDate desc,
    PageSize: 20
  ) {
    controlbar filterBar {
      -- Single input searches OrderNumber OR CustomerName (OR within widget)
      textfilter fSearch (
        attributes: [MyMod.Order.OrderNumber, MyMod.Order.CustomerName]
      )
      -- Additional filters combine with AND
      dropdownfilter fStatus   (attributes: [MyMod.Order.Status])
      numberfilter   fAmount   (attributes: [MyMod.Order.TotalAmount])
      datefilter     fDate     (attributes: [MyMod.Order.OrderDate])
    }
    column colNumber   (attribute: OrderNumber,  caption: 'Order #')
    column colCustomer (attribute: CustomerName, caption: 'Customer')
    column colAmount   (attribute: TotalAmount,  caption: 'Amount',  Alignment: right)
    column colStatus   (attribute: Status,       caption: 'Status')
  }
}
```

### Pattern 4: Gallery filter bar (multi-attribute search)

Gallery uses `filter {}` container instead of `controlbar {}`:

```sql
gallery productGallery (datasource: database MyMod.Product, selection: single) {
  filter filterBar {
    -- OR match: shows row if Name OR Code OR Category contains text
    textfilter fSearch (
      attributes: [MyMod.Product.Name, MyMod.Product.Code, MyMod.Product.Category]
    )
    dropdownfilter fActive (attributes: [MyMod.Product.IsActive])
    numberfilter   fPrice  (attributes: [MyMod.Product.Price])
  }
  template template1 {
    dynamictext txtName  (content: '{1}', contentparams: [{1} = Name],     rendermode: H4)
    dynamictext txtCode  (content: 'SKU: {1}', contentparams: [{1} = Code])
    dynamictext txtPrice (content: '${1}',     contentparams: [{1} = Price])
  }
}
```

### Pattern 5: Column filters + CRUD actions

Action buttons go in `controlbar {}` as `actionbutton`; add filter widgets separately in columns or alongside actions in controlbar:

```sql
datagrid dgOrders (datasource: database MyMod.Order, PageSize: 20) {
  -- Mix: action button + filter widgets in same controlbar
  controlbar cb1 {
    actionbutton btnNew (
      caption: 'New Order',
      action: create_object MyMod.Order then show_page MyMod.Order_Edit,
      buttonstyle: primary
    )
    textfilter     fCust   (attributes: [MyMod.Order.CustomerName])
    dropdownfilter fStatus (attributes: [MyMod.Order.Status])
  }

  column colNumber   (attribute: OrderNumber,  caption: 'Order #') { textfilter fNum }
  column colCustomer (attribute: CustomerName, caption: 'Customer')
  column colStatus   (attribute: Status,       caption: 'Status')

  column colActions (caption: 'Actions', ShowContentAs: customContent) {
    actionbutton btnEdit   (caption: 'Edit',   action: show_page MyMod.Order_Edit (Order: $currentObject))
    actionbutton btnDelete (caption: 'Delete', action: delete_object, buttonstyle: danger)
  }
}
```

## Known Limitations

| Pattern | Status | Detail |
|---------|--------|--------|
| Column-level filter (`column {} { textfilter }`) | ✅ Works | Auto-wired to column attribute type |
| DataGrid2 filter bar (`controlbar {} { textfilter (attributes:[...]) }`) | ✅ Works | Fixed: `buildWidgetBSON()` now calls `BuildFilterWidgetGen()` (commit fc1b6ee3) |
| `filtertype:` property | ✅ Works | Fixed: forwarded via `FilterWidgetSpec.FilterType`; `applyFilterTypeToBSON()` applies it (commit fc1b6ee3) |
| `attributes:` on filter bar widgets | ✅ Works | Fixed: forwarded via `FilterWidgetSpec.Attributes` (commit fc1b6ee3) |
| Gallery filter bar (`filter {} { textfilter (attributes:[...]) }`) | ✅ Works | Routes through pluggable widget engine |
| PLUGGABLEWIDGET advanced props | ✅ Works | Full property access: placeholder, adjustable, multiselect, clearable, applyAfterMs |
| `Keep Selection` with NPE datasource | ❌ Not supported | NPE IDs change on every refresh; selection is always lost. Use persistent entities if selection persistence is needed. |

## Validation

```bash
# Syntax check (no project needed)
./bin/mxcli check your-page-script.mdl

# Full check with entity/association reference validation
./bin/mxcli check your-page-script.mdl -p path/to/app.mpr --references
```

## See Also

- `mendix:create-page` — Full widget syntax reference including DataGrid2
- `mendix:overview-pages` — CRUD page patterns
- `mendix:master-detail-pages` — Gallery filter bar and selection patterns
- `mendix:page-data-design` — Datasource strategy
