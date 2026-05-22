# Mendix Skill Files Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert 8 Mendix skill files from plain-Markdown reference docs to Superpowers-format Claude Code skills with YAML frontmatter, When-to-Use, Checklists, Known Limitations, and new content for DataGrid2 filters, NPE datasource patterns, and page data container design.

**Architecture:** Each skill file gets: (1) YAML frontmatter with `name:` + `description:` trigger keywords, (2) `## When to Use This Skill` bullet list, (3) `## Checklist` of ordered executable steps, (4) `## Quick Syntax Reference` table, (5) `## Core Patterns` with MDL code blocks, (6) `## Known Limitations` with ✅/⚠️/❌ status markers, (7) `## Validation` bash block. Files are created in dependency order so cross-references are valid.

**Tech Stack:** Markdown, MDL (Mendix Definition Language), `./bin/mxcli check` for code block validation, `make sync-skills` for deployment.

---

## File Map

| Task | File | Operation | Target path |
|------|------|-----------|-------------|
| 1 | `datagrid2-filters.md` | CREATE | `cmd/mxcli/skills/datagrid2-filters.md` |
| 2 | `data-containers.md` | CREATE | `cmd/mxcli/skills/data-containers.md` |
| 3 | `create-page.md` | REWRITE | `cmd/mxcli/skills/create-page.md` |
| 4 | `generate-domain-model.md` | UPDATE | `cmd/mxcli/skills/generate-domain-model.md` |
| 5 | `write-microflows.md` | UPDATE | `cmd/mxcli/skills/write-microflows.md` |
| 6 | `overview-pages.md` | REWRITE | `cmd/mxcli/skills/overview-pages.md` |
| 7 | `master-detail-pages.md` | REWRITE | `cmd/mxcli/skills/master-detail-pages.md` |
| 8 | `alter-page.md` | UPDATE | `cmd/mxcli/skills/alter-page.md` |
| 9 | Sync + validate | — | `make sync-skills` |

**Validation approach:** Skill files contain MDL code blocks. After each file is written, extract and run the MDL examples through `./bin/mxcli check` using a scratch `.mdl` file. If the code block creates entities or modules that don't exist in `testdata/`, use `--no-references` (syntax-only check).

---

## Task 1: Create `datagrid2-filters.md`

**Files:**
- Create: `cmd/mxcli/skills/datagrid2-filters.md`

This is the format-establishing file. All subsequent files follow the same skeleton.

- [ ] **Step 1: Write the file**

Create `cmd/mxcli/skills/datagrid2-filters.md` with this complete content:

```markdown
---
name: datagrid2-filters
description: Use when adding filter widgets to DataGrid2 — textfilter
             numberfilter dropdownfilter datefilter column filter bar
             filtertype attributes datasource pluggablewidget datagrid2
---

## When to Use This Skill

- Adding `textfilter` / `numberfilter` / `datefilter` / `dropdownfilter` inside a DataGrid2 column
- Setting `filtertype:` default comparison (startsWith, equal, contains, etc.)
- Configuring advanced filter properties: placeholder, adjustable, multiselect, clearable
- Understanding why filter widgets in `controlbar {}` don't work as expected
- Building a Gallery filter bar with multi-attribute search

## Checklist

- [ ] Identify the attribute type for each column → auto-selects filter widget kind
- [ ] Place filter widget inside `column {}` body (not in `controlbar {}`)
- [ ] For advanced properties (placeholder, adjustable, multiselect): use `PLUGGABLEWIDGET`
- [ ] Run `./bin/mxcli check script.mdl` to validate MDL syntax
- [ ] Run `mx check app.mpr` to confirm no BSON errors in Studio Pro

## Quick Syntax Reference

### Column-Level Filter (auto-type selection)

| Attribute type | Filter widget | Notes |
|---|---|---|
| String | `textfilter` | Default filtertype: contains |
| Integer / Long / Decimal / AutoNumber | `numberfilter` | Default: greaterEqual |
| DateTime | `datefilter` | Default: between |
| Enumeration / Boolean | `dropdownfilter` | Auto-populates options |

### Filter Widget Syntax

```sql
-- Column-level: widget goes inside column {} body
column colName (attribute: AttrName, caption: 'Label') {
  textfilter     filtName                          -- String attrs
  numberfilter   filtName                          -- Numeric attrs
  datefilter     filtName                          -- DateTime attrs
  dropdownfilter filtName                          -- Enum/Boolean attrs
}

-- With explicit filtertype (default comparison)
column colName (attribute: AttrName) {
  textfilter filtName (filtertype: startsWith)
}
```

### Gallery Filter Bar (multi-attribute search)

```sql
gallery g1 (datasource: database Module.Entity, selection: single) {
  filter filterBar {
    textfilter     f1 (attributes: [Module.Entity.Name, Module.Entity.Code])
    numberfilter   f2 (attributes: [Module.Entity.Price])
    dropdownfilter f3 (attributes: [Module.Entity.Status])
    datefilter     f4 (attributes: [Module.Entity.CreatedDate])
  }
  template template1 {
    dynamictext txt1 (content: '{1}', contentparams: [{1} = Name], rendermode: H4)
  }
}
```

### Advanced Properties via PLUGGABLEWIDGET

```sql
-- Inside column {} body — full property access
column colName (attribute: AttrName) {
  pluggablewidget 'com.mendix.widget.web.datagridtextfilter.DatagridTextFilter' filtName (
    defaultFilter: 'startsWith',
    placeholder:   'Search…',
    adjustable:    'yes',
    applyAfterMs:  '300'
  )
}

-- Dropdown with multi-select and clearable
column colStatus (attribute: Status) {
  pluggablewidget 'com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter' fStatus (
    multiselect:         'yes',
    clearable:           'yes',
    emptyCaption:        'All',
    showSelectedItemsAs: 'labels',
    selectionMethod:     'checkbox'
  )
}

-- Date with adjustable comparison type
column colDate (attribute: OrderDate) {
  pluggablewidget 'com.mendix.widget.web.datagriddatefilter.DatagridDateFilter' fDate (
    defaultFilter: 'between',
    adjustable:    'yes'
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

Full DataGrid2 with one filter per column, auto-typed:

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
                        Alignment: right)                                      { numberfilter   fAmt      }
    column colDate     (attribute: OrderDate,    caption: 'Date')              { datefilter     fDate     }
    column colStatus   (attribute: Status,       caption: 'Status')            { dropdownfilter fStatus   }
    column colActive   (attribute: IsActive,     caption: 'Active')            { dropdownfilter fActive   }
  }
}
```

### Pattern 2: filtertype — default comparison type

```sql
datagrid dgOrders (datasource: database MyMod.Order) {
  -- Search from the start of the string
  column colCode (attribute: OrderNumber, caption: 'Order #') {
    textfilter fCode (filtertype: startsWith)
  }
  -- Exact match
  column colEmail (attribute: Email, caption: 'Email') {
    textfilter fEmail (filtertype: equal)
  }
  -- Greater-than-or-equal (useful for minimum thresholds)
  column colAmount (attribute: TotalAmount, caption: 'Min Amount', Alignment: right) {
    numberfilter fAmt (filtertype: greaterEqual)
  }
}
```

Valid `filtertype` values: `contains` (default) | `startsWith` | `endsWith` | `equal` |
`notEqual` | `empty` | `notEmpty` | `greater` | `greaterEqual` | `smaller` | `smallerEqual`

### Pattern 3: Gallery filter bar (multi-attribute OR search)

A single `textfilter` can search across multiple string attributes. Matches rows where ANY attribute contains the text.

```sql
gallery productGallery (datasource: database MyMod.Product, selection: single) {
  filter filterBar {
    -- OR match: shows row if Name OR Code OR Category contains text
    textfilter fSearch (
      attributes: [MyMod.Product.Name, MyMod.Product.Code, MyMod.Product.Category]
    )
    dropdownfilter fActive (attributes: [MyMod.Product.IsActive])
  }
  template template1 {
    dynamictext txtName (content: '{1}', contentparams: [{1} = Name], rendermode: H4)
    dynamictext txtCode (content: 'SKU: {1}', contentparams: [{1} = Code])
  }
}
```

Multiple filter widgets combine with AND: `fSearch AND fActive`.

### Pattern 4: Advanced filter properties (PLUGGABLEWIDGET)

Use when the shorthand (`textfilter filtName`) does not expose the property you need:

```sql
datagrid dgOrders (datasource: database MyMod.Order) {
  -- Placeholder text + 200ms debounce + user can switch comparison type
  column colCustomer (attribute: CustomerName, caption: 'Customer') {
    pluggablewidget 'com.mendix.widget.web.datagridtextfilter.DatagridTextFilter' fCust (
      defaultFilter: 'contains',
      placeholder:   'Search customer…',
      adjustable:    'yes',
      applyAfterMs:  '200'
    )
  }
  -- Multi-select + clearable + label badges
  column colStatus (attribute: Status, caption: 'Status') {
    pluggablewidget 'com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter' fStat (
      multiselect:         'yes',
      clearable:           'yes',
      emptyCaption:        'All statuses',
      showSelectedItemsAs: 'labels'
    )
  }
}
```

### Pattern 5: Column filters + CRUD actions

Action buttons go in `controlbar {}` as `actionbutton`; filter widgets stay in column bodies:

```sql
datagrid dgOrders (datasource: database MyMod.Order, PageSize: 20) {
  -- Action button in controlbar (NOT filter widgets)
  controlbar cb1 {
    actionbutton btnNew (
      caption: 'New Order',
      action: create_object MyMod.Order then show_page MyMod.Order_Edit,
      buttonstyle: primary
    )
  }

  -- Column-level filters
  column colNumber   (attribute: OrderNumber,  caption: 'Order #') { textfilter fNum }
  column colCustomer (attribute: CustomerName, caption: 'Customer') { textfilter fCust }
  column colStatus   (attribute: Status,       caption: 'Status')   { dropdownfilter fStatus }

  -- Inline row actions (no filter here)
  column colActions (caption: 'Actions', ShowContentAs: customContent) {
    actionbutton btnEdit (caption: 'Edit', action: show_page MyMod.Order_Edit (Order: $currentObject))
    actionbutton btnDel  (caption: 'Delete', action: delete_object, buttonstyle: danger)
  }
}
```

## Known Limitations

| Pattern | Status | Detail / Workaround |
|---------|--------|---------------------|
| Column-level filter (`column {} { textfilter }`) | ✅ Works | Auto-wired to column attribute type |
| Gallery filter bar (`filter {} { textfilter (attributes:[...]) }`) | ✅ Works | Routes through pluggable widget engine; `attributes:` list is applied |
| DataGrid2 filter bar (`controlbar {} { textfilter }`) | ⚠️ Gap | `buildWidgetBSON()` (line 2148, `cmd_pages_builder_v3.go`) falls back to DivContainer for unknown widget types. Fix requires: add filter-widget case to `buildWidgetBSON()` + extend `FilterWidgetSpec` with `Attributes []string` + `FilterType string`. Workaround: use column-level filters |
| `filtertype:` shorthand | ⚠️ Gap | Parsed by visitor (`visitor_page_v3.go:537`), stored in AST, NOT forwarded to `BuildFilterWidgetGen()`. Workaround: use `PLUGGABLEWIDGET` with `defaultFilter:` |
| PLUGGABLEWIDGET in column body | ✅ Works | Full property access to all filter widget properties |
| `Keep Selection` with NPE | ❌ Broken | NPE IDs change on every refresh; selection is always lost |

## Validation

```bash
# Syntax check (no project needed)
./bin/mxcli check your-page-script.mdl

# Full check with entity/association reference validation
./bin/mxcli check your-page-script.mdl -p path/to/app.mpr --references
```
```

- [ ] **Step 2: Validate the MDL code blocks**

Write the Pattern 1 code block to a scratch file and check it (entities don't need to exist for syntax-only check):

```bash
cat > /tmp/df-check.mdl << 'EOF'
create module MyMod;
create enumeration MyMod.StatusEnum (Active 'Active', Closed 'Closed');
create persistent entity MyMod.Order (
  OrderNumber: string(50) not null,
  CustomerName: string(200),
  TotalAmount: decimal,
  IsActive: boolean default true,
  Status: enumeration(MyMod.StatusEnum),
  OrderDate: datetime
);
create page MyMod.Order_Overview (
  title: 'Orders', layout: Atlas_Core.Atlas_Default, url: 'orders'
) {
  datagrid dgOrders (datasource: database from MyMod.Order sort by OrderDate desc, PageSize: 25, PagingPosition: both) {
    column colNumber   (attribute: OrderNumber,  caption: 'Order #')         { textfilter     fNum    }
    column colCustomer (attribute: CustomerName, caption: 'Customer')        { textfilter     fCust   }
    column colAmount   (attribute: TotalAmount,  caption: 'Amount', Alignment: right) { numberfilter fAmt }
    column colDate     (attribute: OrderDate,    caption: 'Date')            { datefilter     fDate   }
    column colStatus   (attribute: Status,       caption: 'Status')          { dropdownfilter fStatus }
  }
}
EOF
./bin/mxcli check /tmp/df-check.mdl
```

Expected: `OK` or no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/skills/datagrid2-filters.md
git commit -m "docs(skills): add datagrid2-filters.md — Superpowers format with filter widget patterns"
```

---

## Task 2: Create `data-containers.md`

**Files:**
- Create: `cmd/mxcli/skills/data-containers.md`

- [ ] **Step 1: Write the file**

Create `cmd/mxcli/skills/data-containers.md` with this complete content:

```markdown
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
- [ ] For lists: choose database source when possible (avoids extra microflow); use microflow/nanoflow for complex logic
- [ ] For NPEs: verify the datasource is microflow or nanoflow — database source will fail at runtime
- [ ] If nesting containers: ensure the parent provides context the child expects
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
         OR datasource: nanoflow Module.NF_GetOne (client-side only)

  LIST → Is the entity persistent?
    YES → datasource: database from Module.Entity [WHERE ...] [SORT BY ...]
          (prefer database — avoids extra microflow hop)
          OR datasource: microflow Module.DSO_GetList (for complex logic)
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

Use when the list requires business logic the WHERE clause can't express, or when the entity is non-persistent:

```sql
-- Microflow: returns list of persistent OR non-persistent entities
create microflow MyMod.DSO_GetPendingOrders ()
  returns list of MyMod.Order
begin
  retrieve $Orders from MyMod.Order
    where [Status = MyMod.OrderStatus.Pending and DueDate < [%CurrentDateTime%]]
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

-- Microflow builds and returns the NPE list (no commit)
create microflow MyMod.DSO_RunSearch ($Query: String)
  returns list of MyMod.SearchResult as $Results
begin
  -- build NPE objects from logic, external API, etc.
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
      -- Master list: Gallery with single selection enabled
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
      -- Detail panel: DataView listens to Gallery selection
      column colDetail (desktopwidth: 8) {
        dataview dvDetail (datasource: selection custList) {
          textbox txtName    (label: 'Name',    attribute: Name)
          textbox txtEmail   (label: 'Email',   attribute: Email)
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

Parent DataView provides context; child DataGrid retrieves via association path:

```sql
create page MyMod.Order_WithItems (
  params: { $Order: MyMod.Order },
  title: 'Order with Items',
  layout: Atlas_Core.Atlas_Default
) {
  dataview dvOrder (datasource: $Order, editable: false) {
    dynamictext dtNum (content: 'Order: {1}', contentparams: [{1} = OrderNumber], rendermode: H3)

    -- Child grid via association path: $Order → Order_OrderItem → OrderItem
    datagrid dgItems (
      datasource: $Order/MyMod.Order_OrderItem/MyMod.OrderItem,
      PageSize: 10
    ) {
      column colProduct  (attribute: ProductName, caption: 'Product')
      column colQty      (attribute: Quantity,    caption: 'Qty',   Alignment: center)
      column colTotal    (attribute: LineTotal,   caption: 'Total', Alignment: right)
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
| ⚠️ ListView PageSize | `PageSize:` on `listview` is parsed but NOT wired to the builder (hardcoded 20). Configure paging in Studio Pro if needed. |
| ⚠️ `datasource: selection` without selection mode | The source widget (Gallery / DataGrid) must have `selection: single` or `selection: multiple` enabled, otherwise the DataView shows nothing. |
| ✅ Nanoflow datasource | Works for both single object and list. Runs client-side — no server round-trip. Cannot access DB directly (use microflow for that). |
| ✅ Multi-hop association | `$Param/Module.Assoc1/Module.Entity1/Module.Assoc2/Module.Entity2` paths work for deeply nested retrievals. |

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
```

- [ ] **Step 2: Validate the MDL code blocks**

```bash
cat > /tmp/dc-check.mdl << 'EOF'
create module MyMod;
create persistent entity MyMod.Order (OrderNumber: string(50), TotalAmount: decimal, IsActive: boolean default true, OrderDate: datetime);
create persistent entity MyMod.Customer (Name: string(200), Email: string(200));
create non-persistent entity MyMod.SearchResult (Title: string(200), Score: decimal, Category: string(100));
create microflow MyMod.DSO_GetPendingOrders () returns list of MyMod.Order begin
  retrieve $Orders from MyMod.Order; return $Orders;
end;
/
create page MyMod.Order_Overview (title: 'Orders', layout: Atlas_Core.Atlas_Default, url: 'orders') {
  datagrid dgOrders (datasource: database from MyMod.Order sort by OrderDate desc, PageSize: 25, PagingPosition: both) {
    column colNumber (attribute: OrderNumber, caption: 'Order #')
    column colDate   (attribute: OrderDate,   caption: 'Date')
  }
}
EOF
./bin/mxcli check /tmp/dc-check.mdl
```

Expected: `OK`.

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/skills/data-containers.md
git commit -m "docs(skills): add data-containers.md — page data design patterns and NPE datasource rules"
```

---

## Task 3: Rewrite `create-page.md`

**Files:**
- Modify: `cmd/mxcli/skills/create-page.md` (full rewrite — current file is 895 lines)

**Strategy:** Keep all existing widget sections (DYNAMICTEXT, ACTIONBUTTON, LAYOUTGRID, DATAVIEW, input widgets, GALLERY, CONTAINER, SNIPPETCALL, IMAGE, PLUGGABLEWIDGET, BULK UPDATES, ALTER PAGE reference). Add frontmatter + When to Use + Checklist at top. Add new sections: LISTVIEW, GROUPBOX, TABCONTAINER/TABPAGE, STATICTEXT, TITLE. Expand DATAGRID section with filters + advanced patterns. Expand Known Limitations.

- [ ] **Step 1: Add frontmatter and new opening sections**

Prepend this block to the top of `cmd/mxcli/skills/create-page.md`, before the existing `# CREATE PAGE` heading:

```markdown
---
name: create-page
description: Use when writing CREATE PAGE or CREATE SNIPPET MDL statements —
             widget syntax dataview datagrid gallery listview groupbox tabcontainer
             textbox combobox dynamictext actionbutton layoutgrid snippetcall
             datasource params variables filter filtertype nanoflow microflow NPE
---

## When to Use This Skill

- Writing `create page` or `create or replace page` statements
- Adding widgets to a page: DataView, DataGrid2, ListView, Gallery, GroupBox, TabContainer
- Configuring datasources (database, microflow, nanoflow, page param, selection, association)
- Using page parameters (`params:`) or page variables (`variables:`)
- Applying conditional visibility (`visible:`), editability (`editable:`), styling (`class:`, `style:`)
- Displaying or editing non-persistent entities (NPE)

## Checklist

- [ ] Choose layout: `Atlas_Core.Atlas_Default` (full page) or `Atlas_Core.PopupLayout` (dialog)
- [ ] Declare `params:` if the page receives objects from the caller
- [ ] Choose the correct data container: DataView (1 object) / DataGrid2 (list table) / ListView (simple list) / Gallery (card grid) — see `mendix:page-data-design`
- [ ] Select datasource: `$param` → `database` → `microflow` → `nanoflow` → `selection` (in preference order for persistent entities); NPE must use `microflow` or `nanoflow`
- [ ] For DataGrid2: add column filters inside `column {}` — see `mendix:datagrid2-filters`
- [ ] Add `url:` only for pages with persistent entity params or no params (NPE params cannot deeplink)
- [ ] Run `./bin/mxcli check script.mdl` (syntax), then `mx check app.mpr` (BSON)

```

- [ ] **Step 2: Add LISTVIEW section**

After the existing `### DATAGRID Widget` section and before `### DATAVIEW Widget`, insert:

```markdown
### LISTVIEW Widget

Simple vertical list. Use when rows need custom template content and the table layout of DataGrid2 is too rigid.

```sql
listview lvItems (
  datasource: database from Module.Entity sort by Name asc
) {
  dynamictext txtName (content: '{1}', contentparams: [{1} = Name], rendermode: H4)
  dynamictext txtDesc (content: '{1}', contentparams: [{1} = Description])
  actionbutton btnView (
    caption: 'View',
    action: show_page Module.Detail (Entity: $currentObject),
    buttonstyle: default
  )
}
```

Supported datasource types: `database`, `microflow`, `nanoflow`, `association path`, `$pageParam`.

Use `$currentObject` inside the listview body to reference the current row's entity.

**Known limitation:** `PageSize:` is parsed but NOT wired to the builder — page size is always 20.
Configure paging in Studio Pro if a different size is needed.
```

- [ ] **Step 3: Add GROUPBOX section**

After the existing `### CONTAINER / CUSTOMCONTAINER Widgets` section, insert:

```markdown
### GROUPBOX Widget

Collapsible section with a captioned header. Use inside DataView to organize related fields.

```sql
dataview dvCustomer (datasource: $Customer) {
  groupbox gbPersonal (
    caption: 'Personal Info',
    HeaderMode: H3,
    Collapsible: YesExpanded
  ) {
    textbox txtName  (label: 'Name',  attribute: Name)
    textbox txtEmail (label: 'Email', attribute: Email)
  }

  groupbox gbAddress (
    caption: 'Address',
    HeaderMode: H4,
    Collapsible: YesCollapsed
  ) {
    textbox txtCity    (label: 'City',    attribute: City)
    textbox txtCountry (label: 'Country', attribute: Country)
  }

  footer footer1 {
    actionbutton btnSave (caption: 'Save', action: save_changes, buttonstyle: primary)
  }
}
```

**`Collapsible` values:**
- `No` — not collapsible (always visible)
- `YesExpanded` — collapsible, starts expanded (open by default)
- `YesCollapsed` — collapsible, starts collapsed (closed by default)

**`HeaderMode` values:** `Div` (default, no heading tag) | `H3` | `H4`
```

- [ ] **Step 4: Add TABCONTAINER / TABPAGE section**

After the GROUPBOX section, insert:

```markdown
### TABCONTAINER / TABPAGE Widgets

Organizes content into horizontal tabs. Use when a form or detail page has multiple parallel sections of equal weight (use GROUPBOX for collapsible sub-sections instead).

```sql
dataview dvOrder (datasource: $Order) {
  tabcontainer tcMain {
    tabpage tpDetails (caption: 'Details') {
      textbox txtNumber (label: 'Order #', attribute: OrderNumber)
      textbox txtAmount (label: 'Amount',  attribute: TotalAmount)
      combobox cmbStatus (label: 'Status', attribute: Status)
    }
    tabpage tpItems (caption: 'Line Items') {
      datagrid dgItems (
        datasource: $Order/MyMod.Order_OrderItem/MyMod.OrderItem,
        PageSize: 10
      ) {
        column colProduct (attribute: ProductName, caption: 'Product')
        column colQty     (attribute: Quantity,    caption: 'Qty', Alignment: center)
      }
    }
    tabpage tpHistory (caption: 'History') {
      listview lvHistory (datasource: database from MyMod.AuditLog) {
        dynamictext txtEvent (content: '{1}', contentparams: [{1} = EventDescription])
      }
    }
  }
  footer footer1 {
    actionbutton btnSave (caption: 'Save', action: save_changes, buttonstyle: primary)
  }
}
```

Rules:
- `tabcontainer` must have at least one `tabpage` child
- Each `tabpage` requires a `caption:` property (shown as tab label)
- `tabpage` as a top-level widget (outside `tabcontainer`) produces a validation error
```

- [ ] **Step 5: Add STATICTEXT and TITLE sections**

After the existing `### DYNAMICTEXT Widget` section, insert:

```markdown
### STATICTEXT Widget

Plain static label. No attribute binding, no ContentParams. Use for fixed instructional text or labels that never change.

```sql
statictext sLabel (content: 'All fields marked * are required')
```

Prefer `dynamictext` when the content might need ContentParams or attribute binding in future.

### TITLE Widget

Section heading stored as a caption (not a ClientTemplate). Use for page or section headings that are always static.

```sql
title tHeading (content: 'Customer Information')
```

`dynamictext` with `rendermode: H1` / `H2` / `H3` is preferred for headings that might later need ContentParams. Use `title` only when the heading is guaranteed static.
```

- [ ] **Step 6: Expand DATAGRID section — add column filters and advanced patterns**

In the existing `### DATAGRID Widget` section, append these subsections after the existing content:

```markdown
#### Column-Level Filter Widgets

Filter widgets auto-select based on the column's attribute type:

| Attribute type | Filter widget auto-selected |
|---|---|
| String | `textfilter` |
| Integer / Long / Decimal / AutoNumber | `numberfilter` |
| DateTime | `datefilter` |
| Enumeration / Boolean | `dropdownfilter` |

```sql
datagrid dgOrders (datasource: database from Module.Order sort by Date desc) {
  column colNum  (attribute: OrderNumber, caption: 'Order #') { textfilter     fNum  }
  column colAmt  (attribute: Amount,      caption: 'Amount',
                  Alignment: right)                          { numberfilter   fAmt  }
  column colDate (attribute: OrderDate,   caption: 'Date')   { datefilter     fDate }
  column colStat (attribute: Status,      caption: 'Status') { dropdownfilter fStat }
}
```

For advanced filter properties (placeholder, multiselect, adjustable), use `PLUGGABLEWIDGET`. See `mendix:datagrid2-filters`.

⚠️ **`filtertype:` is parsed but NOT forwarded to BSON.** Use `PLUGGABLEWIDGET` with `defaultFilter:` until the builder is updated.

⚠️ **DataGrid2 filter bar in `controlbar`** — filter widgets placed inside `controlbar {}` produce DivContainers, not real filter widgets. Use column-level filters as the working pattern.

#### Page Variables for Column Visibility

```sql
create page Module.ProductList (
  title: 'Products',
  layout: Atlas_Core.Atlas_Default,
  variables: { $showStock: boolean = 'true' }
) {
  datagrid dgProducts (datasource: database Module.Product) {
    column colName  (attribute: Name,  caption: 'Name')
    column colPrice (attribute: Price, caption: 'Price')
    -- Only visible when $showStock = true
    column colStock (attribute: Stock, caption: 'Stock', visible: '$showStock')
  }
}
```

#### ShowContentAs: url and email

```sql
datagrid dgContacts (datasource: database Module.Contact) {
  column colName  (attribute: Name,    caption: 'Name',    ShowContentAs: text)
  column colWeb   (attribute: Website, caption: 'Website', ShowContentAs: url)
  column colEmail (attribute: Email,   caption: 'Email',   ShowContentAs: email)
}
```
```

- [ ] **Step 7: Expand Known Limitations section**

Find the existing `## Known Limitations` section and append:

```markdown
- ⚠️ **DataGrid2 filter bar** — `textfilter`/`numberfilter`/etc. inside `controlbar {}` produce DivContainers (not real filters). Use column-level filters (`column {} { textfilter }`) instead.
- ⚠️ **`filtertype:` not wired** — parsed by the visitor but NOT written to BSON. Workaround: use `pluggablewidget 'com.mendix.widget.web.datagridtextfilter.DatagridTextFilter'` with `defaultFilter:` property.
- ⚠️ **NPE + `url:`** — pages with non-persistent entity parameters cannot have a `url:` field (no deeplink support). Remove `url:` from the page declaration.
- ⚠️ **NPE + `Keep Selection`** — `Keep Selection` on DataGrid2 does not work with NPE datasources; IDs change on every refresh, clearing selection.
- ⚠️ **ListView `PageSize:`** — parsed but not wired to the builder; always renders 20 rows regardless.
- ⚠️ **`TABPAGE` outside `TABCONTAINER`** — produces a runtime validation error. TABPAGE must always be a direct child of TABCONTAINER.
```

- [ ] **Step 8: Validate new MDL code blocks**

```bash
cat > /tmp/cp-check.mdl << 'EOF'
create module MyMod;
create enumeration MyMod.St (Active 'Active', Closed 'Closed');
create persistent entity MyMod.Order (OrderNumber: string(50), Amount: decimal, Status: enumeration(MyMod.St), OrderDate: datetime);
create persistent entity MyMod.Customer (Name: string(200), Email: string(200), City: string(100), Country: string(100));
create persistent entity MyMod.Contact (Name: string(200), Website: string(500), Email: string(200));
create persistent entity MyMod.Product (Name: string(200), Price: decimal, Stock: integer);
create page MyMod.ContactList (title: 'Contacts', layout: Atlas_Core.Atlas_Default, url: 'contacts') {
  datagrid dgContacts (datasource: database MyMod.Contact) {
    column colName  (attribute: Name,    caption: 'Name',    ShowContentAs: text)
    column colWeb   (attribute: Website, caption: 'Website', ShowContentAs: url)
    column colEmail (attribute: Email,   caption: 'Email',   ShowContentAs: email)
  }
}
create page MyMod.ProductList (title: 'Products', layout: Atlas_Core.Atlas_Default, variables: { $showStock: boolean = 'true' }) {
  datagrid dgProducts (datasource: database MyMod.Product) {
    column colName  (attribute: Name,  caption: 'Name')
    column colPrice (attribute: Price, caption: 'Price')
    column colStock (attribute: Stock, caption: 'Stock', visible: '$showStock')
  }
}
EOF
./bin/mxcli check /tmp/cp-check.mdl
```

Expected: `OK`.

- [ ] **Step 9: Commit**

```bash
git add cmd/mxcli/skills/create-page.md
git commit -m "docs(skills): rewrite create-page.md — Superpowers format, add LISTVIEW/GROUPBOX/TABCONTAINER/STATICTEXT/TITLE, expand DataGrid2 filters"
```

---

## Task 4: Update `generate-domain-model.md`

**Files:**
- Modify: `cmd/mxcli/skills/generate-domain-model.md` (add frontmatter + NPE section)

- [ ] **Step 1: Add frontmatter**

Prepend to the very top of `cmd/mxcli/skills/generate-domain-model.md` (before the existing `# Creating Mendix Domain Model MDL Scripts` heading):

```markdown
---
name: generate-domain-model
description: Use when creating entities, attributes, associations, enumerations,
             or non-persistent entities in MDL — create entity persistent
             non-persistent NPE enumeration association domain model attribute
---

```

- [ ] **Step 2: Add NPE section**

After the existing `### Entities` subsection (around line 296), add:

```markdown
### Non-Persistent Entities (NPE)

Non-persistent entities are stored in runtime memory only — no database table is created. Use for:
- Intermediate calculation results (validation output, search results)
- Data Transfer Objects (DTOs) between service calls
- Aggregation views that combine data from multiple sources

```sql
create non-persistent entity MyMod.SearchResult (
  Title:    String(200),
  Score:    Decimal,
  Category: String(100),
  IsValid:  Boolean default false
);

create non-persistent entity MyMod.ValidationResult (
  IsValid:  Boolean default false,
  Errors:   String(2000),
  Warnings: String(2000)
);
```

**Critical rules:**
- `datasource: database` on a page widget causes a runtime error for NPEs — they have no DB table
- `create list of NPE` in a microflow is blocked (CE0053) — build objects individually and collect them
- NPE objects do NOT need `commit` — they exist only in memory for the current request/session
- Pages with NPE parameters cannot have a `url:` field (no deeplink support)
- Return NPE objects from a microflow/nanoflow to display them in a page widget

```sql
-- Microflow returning a list of NPEs for a DataGrid datasource
create microflow MyMod.DSO_SearchProducts ($Query: String)
  returns list of MyMod.SearchResult as $Results
begin
  -- Build NPE objects (no commit required)
  $r1 = create MyMod.SearchResult (Title = 'Product A', Score = 0.95, Category = 'Electronics');
  $r2 = create MyMod.SearchResult (Title = 'Product B', Score = 0.80, Category = 'Books');
  return list($r1, $r2);
end;
/
```

See `mendix:page-data-design` for datasource patterns when displaying NPEs on pages.
```

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/skills/generate-domain-model.md
git commit -m "docs(skills): update generate-domain-model.md — add frontmatter + NPE section with CE0053 rules"
```

---

## Task 5: Update `write-microflows.md`

**Files:**
- Modify: `cmd/mxcli/skills/write-microflows.md` (add frontmatter + DSO_ section)

- [ ] **Step 1: Add frontmatter**

Prepend to the very top of `cmd/mxcli/skills/write-microflows.md`:

```markdown
---
name: write-microflows
description: Use when writing CREATE MICROFLOW MDL statements — microflow syntax
             create change commit retrieve loop if return DSO datasource nanoflow
             NPE non-persistent list return type parameter variable flow
---

```

- [ ] **Step 2: Add DSO_ naming pattern section**

After the existing `## When to Use This Skill` section, insert:

```markdown
## Page Datasource Microflows (DSO_ Pattern)

Microflows used as page widget datasources follow the `DSO_` prefix convention.
DSO = DataSource Object. These microflows return a list or single object for direct widget consumption.

```sql
-- Returns a list (for DataGrid / ListView / Gallery datasource)
create microflow MyMod.DSO_GetActiveOrders ()
  returns list of MyMod.Order
begin
  retrieve $Orders from MyMod.Order
    where [IsActive = true()] sort by OrderDate desc;
  return $Orders;
end;
/

-- Returns a single object (for DataView datasource)
create microflow MyMod.DSO_GetCurrentUserProfile ()
  returns MyMod.UserProfile as $Profile
begin
  retrieve $Profile from MyMod.UserProfile
    where [UserId = '[%CurrentUser%]'] limit 1;
  return $Profile;
end;
/

-- Returns a list of NPEs (no commit; objects live in memory only)
create microflow MyMod.DSO_BuildDashboardStats ()
  returns list of MyMod.DashboardStat as $Stats
begin
  $s1 = create MyMod.DashboardStat (Label = 'Open Orders', Count = 42);
  $s2 = create MyMod.DashboardStat (Label = 'Pending Items', Count = 17);
  return list($s1, $s2);
end;
/
```

**DSO_ rules:**
- No parameters (or minimal: search query only) — they are called by the runtime when the page opens, without user-provided context
- Must return the exact type the widget expects: `list of Entity` for list widgets, `Entity` for DataView
- For NPEs: no `commit` — just build and return; objects live in session memory only
- Name: `DSO_GetXxx` (for retrieves) or `DSO_BuildXxx` / `DSO_ComputeXxx` (for NPE construction)

## Nanoflow vs Microflow as Datasource

| Criteria | Microflow | Nanoflow |
|----------|-----------|----------|
| Runs on | Server | Client (browser) |
| Can access DB | ✅ Yes | ❌ No |
| Network round-trip | ✅ Yes | ❌ No (faster) |
| Java action calls | ✅ Yes | ❌ No |
| Best for NPE construction | ✅ From server data | ✅ From client-side state only |
| Best for DB retrieves | ✅ Use microflow | ❌ Cannot |

Use `nanoflow` as datasource when: pure client-side calculation, no DB access, reducing server load.
Use `microflow` when: DB retrieve is needed, complex server logic, Java action calls required.

```

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/skills/write-microflows.md
git commit -m "docs(skills): update write-microflows.md — add frontmatter + DSO_ naming pattern + nanoflow vs microflow datasource guidance"
```

---

## Task 6: Rewrite `overview-pages.md`

**Files:**
- Modify: `cmd/mxcli/skills/overview-pages.md` (Superpowers format rewrite — current file is 572 lines)

- [ ] **Step 1: Replace the file with Superpowers format**

Overwrite `cmd/mxcli/skills/overview-pages.md` with:

```markdown
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
| Current row object | `$currentObject` (inside `column {}` body) |
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
    column colName   (attribute: Name,     caption: 'Name')
    column colPrice  (attribute: Price,    caption: 'Price', Alignment: right)
    column colStatus (attribute: IsActive, caption: 'Active')
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

### Pattern 2: Overview with column filters + sorting

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

### Pattern 3: Overview with microflow datasource (complex filtering)

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

-- Use in overview pages
create page MyMod.Dashboard (title: 'Dashboard', layout: Atlas_Core.Atlas_Default, url: 'dashboard') {
  layoutgrid lg1 {
    row row1 {
      column colNav  (desktopwidth: 3) { snippetcall navSnippet (snippet: MyMod.NavMenu) }
      column colMain (desktopwidth: 9) { dynamictext dtWelcome (content: 'Welcome', rendermode: H2) }
    }
  }
}
```

## Known Limitations

- ⚠️ `ShowContentAs: customContent` columns with filter widgets inside produce nested DataGrid children — keep action buttons and filter widgets in separate columns
- ⚠️ Column-level `filtertype:` parsed but NOT wired to BSON — use `PLUGGABLEWIDGET` with `defaultFilter:` for explicit filter type
- ⚠️ `controlbar {}` filter widgets (textfilter/dropdownfilter) produce DivContainers — use column-body filters instead (see `mendix:datagrid2-filters`)

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
```

- [ ] **Step 2: Validate the MDL code blocks**

```bash
cat > /tmp/ovw-check.mdl << 'EOF'
create module MyMod;
create enumeration MyMod.OrderStatus (Pending 'Pending', Active 'Active');
create persistent entity MyMod.Product (Name: string(200) not null, Price: decimal, IsActive: boolean default true);
create persistent entity MyMod.Order (OrderNumber: string(50) not null, CustomerName: string(200), TotalAmount: decimal, Status: enumeration(MyMod.OrderStatus), DueDate: datetime, OrderDate: datetime);
create page MyMod.Product_Edit (params: { $Product: MyMod.Product }, title: 'Edit Product', layout: Atlas_Core.PopupLayout, folder: 'Products') {
  dataview dvProduct (datasource: $Product) {
    textbox txtName  (label: 'Name',  attribute: Name)
    footer footer1 {
      actionbutton btnSave (caption: 'Save', action: save_changes close_page, buttonstyle: primary)
    }
  }
}
create page MyMod.Product_Overview (title: 'Products', layout: Atlas_Core.Atlas_Default, url: 'products', folder: 'Products') {
  datagrid dgProducts (datasource: database from MyMod.Product sort by Name asc, PageSize: 25, PagingPosition: both) {
    controlbar cb1 {
      actionbutton btnNew (caption: 'New Product', action: create_object MyMod.Product then show_page MyMod.Product_Edit, buttonstyle: primary)
    }
    column colName   (attribute: Name,     caption: 'Name')
    column colPrice  (attribute: Price,    caption: 'Price', Alignment: right)
    column colActions (caption: 'Actions', ShowContentAs: customContent) {
      actionbutton btnEdit   (caption: 'Edit',   action: show_page MyMod.Product_Edit (Product: $currentObject), buttonstyle: default)
      actionbutton btnDelete (caption: 'Delete', action: delete_object, buttonstyle: danger)
    }
  }
}
EOF
./bin/mxcli check /tmp/ovw-check.mdl
```

Expected: `OK`.

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/skills/overview-pages.md
git commit -m "docs(skills): rewrite overview-pages.md — Superpowers format with CRUD patterns and column filter examples"
```

---

## Task 7: Rewrite `master-detail-pages.md`

**Files:**
- Modify: `cmd/mxcli/skills/master-detail-pages.md` (Superpowers format rewrite — current 184 lines)

- [ ] **Step 1: Replace the file with Superpowers format**

Overwrite `cmd/mxcli/skills/master-detail-pages.md` with:

```markdown
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
- [ ] For Gallery filter bar: use `filter filterBarName { textfilter ... }` inside the gallery (NOT in controlbar)
- [ ] Run `./bin/mxcli check script.mdl` to validate

## Quick Syntax Reference

| Element | Syntax | Notes |
|---------|--------|-------|
| Enable selection | `selection: single` or `selection: multiple` on Gallery/DataGrid | Required for `datasource: selection` |
| Listen to selection | `dataview dv (datasource: selection masterWidgetName)` | Widget names must match exactly |
| Association datasource | `datagrid dg (datasource: $Param/Module.Assoc/Module.Entity)` | Path from page param through association |
| Gallery filter bar | `gallery g1 (...) { filter fb { textfilter f1 (attributes: [...]) } template t1 { ... } }` | `attributes:` is required for grid-level filters |
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
      -- LEFT: master list
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

Use when the list is tabular and the detail is a form below:

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

The `filter {}` container inside a Gallery enables a search bar. `attributes: [A, B]` means the filter shows rows where ANY listed attribute contains the text (OR match):

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
      -- Single input searches Name, Code, AND Category (OR match)
      textfilter fSearch (
        attributes: [MyMod.Product.Name, MyMod.Product.Code, MyMod.Product.Category]
      )
      -- Additional AND filters
      dropdownfilter fActive   (attributes: [MyMod.Product.IsActive])
      numberfilter   fMinPrice (attributes: [MyMod.Product.Price])
    }
    template template1 {
      dynamictext txtName     (content: '{1}',  contentparams: [{1} = Name],     rendermode: H4)
      dynamictext txtCode     (content: 'SKU: {1}', contentparams: [{1} = Code])
      dynamictext txtPrice    (content: '${1}', contentparams: [{1} = Price])
    }
  }
}
```

Multiple filters combine with AND: `fSearch AND fActive AND fMinPrice`.

### Pattern 4: DataView (page param) → nested DataGrid via association

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

- ⚠️ **`datasource: selection` requires selection enabled** — the source widget (Gallery/DataGrid) must have `selection: single` or `selection: multiple`; otherwise the DataView shows nothing at runtime
- ⚠️ **Selection binding is page-scoped** — master widget and detail DataView must be on the same page; cross-page selection is not possible
- ⚠️ **NPE lists + `Keep Selection`** — DataGrid2 `Keep Selection` does not work with NPE datasources; selection is lost on filter or refresh
- ⚠️ **Gallery `filter {}` vs DataGrid `controlbar {}`** — `filter {}` inside a gallery routes through the pluggable widget engine and correctly applies `attributes:[...]`. The same filter widgets inside a DataGrid `controlbar {}` produce DivContainers — use column-level filters for DataGrid instead (see `mendix:datagrid2-filters`)
- ✅ **Gallery filter bar `attributes:[]`** — multi-attribute text search (OR within the filter) works correctly in gallery context

## Validation

```bash
./bin/mxcli check script.mdl
./bin/mxcli check script.mdl -p path/to/app.mpr --references
```

## See Also

- `mendix:create-page` — Full widget syntax reference
- `mendix:overview-pages` — CRUD overview + DataGrid CRUD patterns
- `mendix:datagrid2-filters` — Column filter widgets
- `mendix:page-data-design` — Datasource strategy
```

- [ ] **Step 2: Validate the MDL code blocks**

```bash
cat > /tmp/md-check.mdl << 'EOF'
create module MyMod;
create enumeration MyMod.OrderStatus (Pending 'Pending', Active 'Active');
create persistent entity MyMod.Customer (Name: string(200) not null, Email: string(200), Phone: string(50));
create persistent entity MyMod.Order (OrderNumber: string(50) not null, TotalAmount: decimal, Status: enumeration(MyMod.OrderStatus), OrderDate: datetime);
create persistent entity MyMod.Product (Name: string(200) not null, Code: string(50), Price: decimal, IsActive: boolean default true, Category: string(100));
create association MyMod.Order_Customer from MyMod.Order to MyMod.Customer;
create page MyMod.Customer_MasterDetail (title: 'Customers', layout: Atlas_Core.Atlas_Default, url: 'customers') {
  layoutgrid lg1 {
    row row1 {
      column colMaster (desktopwidth: 4) {
        gallery custList (datasource: database from MyMod.Customer sort by Name asc, selection: single) {
          template template1 {
            dynamictext txtName (content: '{1}', contentparams: [{1} = Name], rendermode: H4)
          }
        }
      }
      column colDetail (desktopwidth: 8) {
        dataview dvDetail (datasource: selection custList) {
          textbox txtName (label: 'Name', attribute: Name)
          footer footer1 {
            actionbutton btnSave (caption: 'Save', action: save_changes, buttonstyle: primary)
          }
        }
      }
    }
  }
}
EOF
./bin/mxcli check /tmp/md-check.mdl
```

Expected: `OK`.

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/skills/master-detail-pages.md
git commit -m "docs(skills): rewrite master-detail-pages.md — Superpowers format, gallery filter bar, selection binding patterns"
```

---

## Task 8: Update `alter-page.md`

**Files:**
- Modify: `cmd/mxcli/skills/alter-page.md` (add frontmatter + DataGrid2 column operations section)

- [ ] **Step 1: Add frontmatter**

Prepend to the very top of `cmd/mxcli/skills/alter-page.md`:

```markdown
---
name: alter-page
description: Use when modifying existing pages or snippets in-place — alter page
             snippet set drop insert replace widget datagrid column variable
             layout filter conditional visibility actionbutton textbox
---

```

- [ ] **Step 2: Extend the existing DataGrid Column Operations section**

Find the `### DataGrid Column Operations` section (around line 129 in the current file) and append after its existing content:

```markdown
#### Adding and removing column filters

Column-level filter widgets (inside `column {}`) cannot currently be addressed by dotted notation in ALTER PAGE. To add or remove a filter, replace the whole column:

```sql
-- Replace a column to add a filter widget
alter page MyMod.Order_Overview {
  replace colStatus with {
    column colStatus (attribute: Status, caption: 'Status') {
      dropdownfilter fStatus
    }
  }
};

-- Replace a column to remove its filter widget
alter page MyMod.Order_Overview {
  replace colStatus with {
    column colStatus (attribute: Status, caption: 'Status')
  }
};
```

#### Notes on filter widgets and ALTER

- Filter widgets in `column {}` are children of the DataGrid2 column BSON — not standalone widgets addressable by name
- Use `DESCRIBE PAGE Module.PageName` to inspect column names before writing ALTER statements
- If a column dotted reference (`dgName.colName`) resolves to the wrong widget, fall back to `REPLACE` on the entire column
```

- [ ] **Step 3: Commit**

```bash
git add cmd/mxcli/skills/alter-page.md
git commit -m "docs(skills): update alter-page.md — add frontmatter + column filter ALTER patterns"
```

---

## Task 9: Sync and validate

**Files:**
- Run: `make sync-skills`
- Verify: `.claude/skills/mendix/` contains all 8 updated/new files

- [ ] **Step 1: Run sync**

```bash
make sync-skills
```

Expected output: files copied from `cmd/mxcli/skills/` to `.claude/skills/mendix/`. No errors.

- [ ] **Step 2: Verify all 8 files are present in `.claude/skills/mendix/`**

```bash
ls .claude/skills/mendix/ | grep -E "datagrid2-filters|data-containers|create-page|generate-domain|write-microflows|overview-pages|master-detail|alter-page"
```

Expected: all 8 names listed.

- [ ] **Step 3: Verify frontmatter in all 8 files**

```bash
for f in datagrid2-filters data-containers create-page generate-domain-model write-microflows overview-pages master-detail-pages alter-page; do
  echo "=== $f.md ==="
  head -4 .claude/skills/mendix/$f.md
done
```

Expected: each file starts with `---`, has `name:` and `description:` lines.

- [ ] **Step 4: Final commit**

```bash
git add .claude/skills/mendix/
git commit -m "chore(skills): sync 8 updated Mendix skills to .claude/skills/mendix/"
```

---

## Self-Review

### Spec coverage check

| Spec requirement | Covered by task |
|---|---|
| YAML frontmatter on all 8 files | Tasks 1–8 (each adds frontmatter) |
| `datagrid2-filters.md` NEW with 5 patterns | Task 1 |
| `data-containers.md` NEW with decision tree, NPE constraints | Task 2 |
| `create-page.md` REWRITE with LISTVIEW/GROUPBOX/TABCONTAINER/STATICTEXT/TITLE | Task 3 |
| `create-page.md` — expanded DataGrid2: filters, ShowContentAs, page variables | Task 3 steps 6–7 |
| `generate-domain-model.md` — NPE section with CE0053 rules | Task 4 |
| `write-microflows.md` — DSO_ pattern, nanoflow vs microflow guidance | Task 5 |
| `overview-pages.md` REWRITE | Task 6 |
| `master-detail-pages.md` REWRITE | Task 7 |
| `alter-page.md` — DataGrid2 column filter ALTER patterns | Task 8 |
| `make sync-skills` after all changes | Task 9 |
| Known Limitations with ✅/⚠️/❌ markers | Tasks 1, 2, 3, 6, 7 |
| Validation bash blocks in every file | Tasks 1, 2, 3, 6, 7 |

### No placeholder scan

Reviewed plan for "TBD", "TODO", "similar to above", "handle edge cases" — none found.

### Type consistency

- `$currentObject` used consistently for DataGrid row reference (Tasks 1, 3, 6)
- `datasource: selection widgetName` uses the exact widget name from the same code block (Task 7)
- Association path format `$Param/Module.Assoc/Module.Entity` consistent across Tasks 2, 3, 7
- `FilterWidgetSpec` naming consistent with Go code references (Task 1 Known Limitations)
