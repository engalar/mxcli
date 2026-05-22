---
name: page-data-design
description: Use when designing how data flows through a Mendix page — dataview
             datasource database microflow nanoflow selection association NPE
             non-persistent data container design page data flow listview gallery
---

## When to Use This Skill

- Choosing between DataView, DataGrid2, ListView, or Gallery for a page section
- Deciding which datasource type to use (database / microflow / nanoflow / $param / selection / association)
- Designing pages that display or edit non-persistent entities (NPEs)
- Nesting data containers (DataView wrapping a DataGrid, master-detail layout)
- Avoiding N+1 queries and unnecessary server round-trips

## Checklist

- [ ] Identify how many objects the section displays (1 → DataView; list → DataGrid/ListView/Gallery)
- [ ] Determine if the object is already available as a page parameter or must be loaded
- [ ] For lists: prefer `database` source for persistent entities (avoids extra microflow hop)
- [ ] For NPEs: verify datasource is `microflow` or `nanoflow` — `database` source fails at runtime
- [ ] If nesting containers: ensure the parent provides the context the child expects
- [ ] Run `./bin/mxcli check script.mdl` to validate all datasource references

## Quick Syntax Reference

### Container Type Selection

| Widget | Holds | Best for |
|--------|-------|----------|
| `dataview` | 1 object | Edit forms, detail panels, header cards |
| `datagrid` | List | Tabular overview with sorting, paging, column filters |
| `listview` | List | Simple vertical list, custom row templates |
| `gallery` | List | Card grid with selection, image-heavy layouts |

### Datasource Decision Tree

```
Is the object/list already a page parameter?
  YES → datasource: $paramName

Is it a single object or a list?
  SINGLE → datasource: microflow Module.DSO_GetOne
         OR datasource: nanoflow Module.NF_GetOne  (client-side, no server round-trip)

  LIST → Is the entity persistent?
    YES → datasource: database from Module.Entity [WHERE ...] [SORT BY ...]
          (prefer database — avoids extra microflow hop)
          OR datasource: microflow Module.DSO_GetList  (for complex logic)
    NO (NPE) → MUST use microflow or nanoflow
               (NPEs have no DB table; database source fails at runtime)
```

### Datasource Syntax Summary

```sql
-- Page parameter (persistent or NPE, passed from caller)
dataview dv1 (datasource: $MyParam) { ... }

-- Database query (persistent entities only)
datagrid dg1 (datasource: database from Module.Entity
              where [IsActive = true()] sort by Name asc) { ... }

-- Microflow datasource (returns single object or list)
dataview dv1 (datasource: microflow Module.DSO_GetSummary) { ... }
datagrid dg1 (datasource: microflow Module.DSO_GetActiveItems) { ... }

-- Nanoflow datasource (client-side, no server round-trip)
dataview dv1 (datasource: nanoflow Module.NF_GetDraft) { ... }
listview lv1 (datasource: nanoflow Module.NF_GetLocalItems) { ... }

-- Selection binding (depends on another widget's selection)
dataview dvDetail (datasource: selection masterGallery) { ... }

-- Association path (from parent context via association)
datagrid dgItems (datasource: $Order/Module.Order_OrderItem/Module.OrderItem) { ... }
```

## Core Patterns

### Pattern 1: DataView from page parameter

Standard detail/edit page. The caller passes the object; this page displays or edits it.

```sql
create page MyMod.Order_Detail (
  params: { $Order: MyMod.Order },
  title: 'Order Details',
  layout: Atlas_Core.PopupLayout
) {
  dataview dvOrder (datasource: $Order) {
    textbox txtNumber (label: 'Order #', attribute: OrderNumber)
    textbox txtAmount (label: 'Amount',  attribute: TotalAmount)
    footer footer1 {
      actionbutton btnSave   (caption: 'Save',   action: save_changes close_page, buttonstyle: primary)
      actionbutton btnCancel (caption: 'Cancel', action: cancel_changes close_page)
    }
  }
}
```

### Pattern 2: DataGrid from database (preferred for persistent lists)

No microflow needed when the query is a straightforward retrieve:

```sql
create page MyMod.Order_Overview (
  title: 'Orders',
  layout: Atlas_Core.Atlas_Default,
  url: 'orders'
) {
  datagrid dgOrders (
    datasource: database from MyMod.Order
      where [IsActive = true()] sort by OrderDate desc,
    PageSize: 25,
    PagingPosition: both
  ) {
    column colNumber (attribute: OrderNumber, caption: 'Order #')
    column colDate   (attribute: OrderDate,   caption: 'Date')
    column colStatus (attribute: Status,      caption: 'Status')
  }
}
```

### Pattern 3: DataGrid from microflow (complex logic or NPE)

Use when the list requires business logic the WHERE clause cannot express, or when the entity is non-persistent:

```sql
-- Microflow: complex filter logic
create microflow MyMod.DSO_GetPendingOrders ()
  returns list of MyMod.Order
begin
  retrieve $Orders from MyMod.Order
    where [Status = MyMod.OrderStatus.Pending and DueDate < addDays('[%CurrentDateTime%]', 7)]
    sort by DueDate asc;
  return $Orders;
end;
/

create page MyMod.PendingOrders (
  title: 'Pending Orders',
  layout: Atlas_Core.Atlas_Default
) {
  datagrid dgOrders (datasource: microflow MyMod.DSO_GetPendingOrders) {
    column colNumber (attribute: OrderNumber, caption: 'Order #')
    column colDue    (attribute: DueDate,     caption: 'Due')
  }
}
```

### Pattern 4: NPE as datasource (microflow bridge)

Non-persistent entities MUST go through microflow or nanoflow — no direct `database` source:

```sql
-- NPE declaration
create non-persistent entity MyMod.SearchResult (
  Title:    String(200),
  Score:    Decimal,
  Category: String(100)
);

-- Microflow builds and returns the NPE list (no commit needed)
create microflow MyMod.DSO_RunSearch ($Query: String)
  returns list of MyMod.SearchResult as $Results
begin
  $r1 = create MyMod.SearchResult (Title = 'Item A', Score = 0.95, Category = 'Books');
  return list($r1);
end;
/

-- Page: microflow datasource is the ONLY valid option for NPE lists
create page MyMod.SearchResults (
  title: 'Search Results',
  layout: Atlas_Core.Atlas_Default
) {
  datagrid dgResults (datasource: microflow MyMod.DSO_RunSearch) {
    column colTitle    (attribute: Title,    caption: 'Title')
    column colScore    (attribute: Score,    caption: 'Score',    Alignment: right)
    column colCategory (attribute: Category, caption: 'Category')
  }
}
```

### Pattern 5: Master-detail (Gallery selection → DataView)

```sql
create page MyMod.Customer_MasterDetail (
  title: 'Customers',
  layout: Atlas_Core.Atlas_Default,
  url: 'customers'
) {
  layoutgrid lg1 {
    row row1 {
      -- Master: Gallery with selection enabled
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
      -- Detail: DataView listens to Gallery selection
      column colDetail (desktopwidth: 8) {
        dataview dvDetail (datasource: selection custList) {
          textbox txtName  (label: 'Name',  attribute: Name)
          textbox txtEmail (label: 'Email', attribute: Email)
          footer footer1 {
            actionbutton btnSave (caption: 'Save', action: save_changes, buttonstyle: primary)
          }
        }
      }
    }
  }
}
```

### Pattern 6: Nested DataView → DataGrid via association

Parent DataView provides context; child DataGrid retrieves related objects via association path:

```sql
create page MyMod.Order_WithItems (
  params: { $Order: MyMod.Order },
  title: 'Order with Items',
  layout: Atlas_Core.Atlas_Default
) {
  dataview dvOrder (datasource: $Order, editable: false) {
    dynamictext dtNum (content: 'Order: {1}', contentparams: [{1} = OrderNumber], rendermode: H3)

    -- Child grid via association: $Order → Order_OrderItem → OrderItem
    datagrid dgItems (
      datasource: $Order/MyMod.Order_OrderItem/MyMod.OrderItem,
      PageSize: 10
    ) {
      column colProduct (attribute: ProductName, caption: 'Product')
      column colQty     (attribute: Quantity,    caption: 'Qty',   Alignment: center)
      column colTotal   (attribute: LineTotal,   caption: 'Total', Alignment: right)
    }
  }
}
```

## Known Limitations

| Constraint | Detail |
|-----------|--------|
| ⚠️ NPE + `database` source | Runtime error — NPEs have no DB table. Always use `microflow` or `nanoflow`. |
| ⚠️ NPE + `url:` on page | Pages with NPE parameters cannot use `url:` — no deeplink support. Remove `url:` or use a persistent parameter. |
| ⚠️ NPE + `Keep Selection` | DataGrid2 `Keep Selection` breaks with NPEs — IDs change on every refresh. |
| ⚠️ ListView `PageSize:` | Parsed but NOT wired to the builder (hardcoded 20). Configure paging in Studio Pro if needed. |
| ⚠️ `datasource: selection` without selection mode | The source widget (Gallery/DataGrid) must have `selection: single` or `selection: multiple` enabled, otherwise the DataView shows nothing. |
| ✅ Nanoflow datasource | Works for single object and list. Runs client-side — no server round-trip. Cannot access DB directly. |
| ✅ Multi-hop association path | `$Param/Module.Assoc1/Module.Entity1/Module.Assoc2/Module.Entity2` works for deeply nested retrievals. |

## Validation

```bash
./bin/mxcli check script.mdl
./bin/mxcli check script.mdl -p path/to/app.mpr --references
```

## See Also

- `mendix:create-page` — Full widget syntax reference
- `mendix:datagrid2-filters` — Filter widgets for DataGrid2
- `mendix:write-microflows` — DSO_ pattern for page datasource microflows
- `mendix:generate-domain-model` — Non-persistent entity declaration
