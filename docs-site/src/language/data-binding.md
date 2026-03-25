# Data Binding

Data binding connects widgets to entity attributes and data sources. In MDL, binding is configured through widget properties rather than a special operator.

## Attribute Binding

Input widgets bind to entity attributes using the `Attribute` property. The attribute name is resolved relative to the containing DataView's entity context:

```sql
DATAVIEW dvCustomer (DataSource: $Customer) {
  -- These attribute names are resolved against MyModule.Customer
  TEXTBOX txtName (Label: 'Name', Attribute: Name)
  TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
  CHECKBOX cbActive (Label: 'Active', Attribute: IsActive)
  DATEPICKER dpCreated (Label: 'Created', Attribute: CreatedDate)
  COMBOBOX cbStatus (Label: 'Status', Attribute: Status)
}
```

The SDK automatically resolves short attribute names to fully qualified paths. For example, `Name` is resolved to `MyModule.Customer.Name` based on the DataView's entity context.

## Data Sources

Data sources determine where a container widget gets its data. Different widget types support different data source types.

### Page Parameter Source

The simplest binding -- connects a DataView to a page parameter:

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
  }
}
```

### Database Source

Retrieves entities directly from the database. Works with list widgets (DataGrid, ListView, Gallery):

```sql
DATAGRID dgCustomers (DataSource: DATABASE MyModule.Customer) {
  COLUMN colName (Attribute: Name, Caption: 'Customer Name')
  COLUMN colEmail (Attribute: Email, Caption: 'Email')
}
```

### Microflow Source

Calls a microflow to retrieve data. The microflow must return the appropriate type (single object for DataView, list for DataGrid/ListView):

```sql
-- Single object for a DataView
DATAVIEW dvStats (DataSource: MICROFLOW MyModule.DS_GetStatistics) {
  DYNAMICTEXT txtCount (Content: 'Total: {1}', Attribute: TotalCount)
}

-- List for a DataGrid
DATAGRID dgFiltered (DataSource: MICROFLOW MyModule.DS_GetFilteredOrders) {
  COLUMN colId (Attribute: OrderId)
}
```

### Nanoflow Source

Same as microflow source but runs on the client side:

```sql
LISTVIEW lvRecent (DataSource: NANOFLOW MyModule.DS_GetRecentItems) {
  DYNAMICTEXT txtItem (Content: '{1}', Attribute: Name)
}
```

### Association Source

Follows an association from a parent DataView to retrieve related objects:

```sql
DATAVIEW dvCustomer (DataSource: $Customer) {
  TEXTBOX txtName (Label: 'Name', Attribute: Name)

  -- Follow Customer_Order association to list orders
  LISTVIEW lvOrders (DataSource: ASSOCIATION Customer_Order) {
    DYNAMICTEXT txtOrderDate (Content: '{1}', Attribute: OrderDate)
    DYNAMICTEXT txtAmount (Content: '${1}', Attribute: Amount)
  }
}
```

### Selection Source

Binds to the currently selected item in another list widget. Enables master-detail patterns:

```sql
-- Master: list of products
DATAGRID dgProducts (DataSource: DATABASE MyModule.Product) {
  COLUMN colName (Attribute: Name)
  COLUMN colPrice (Attribute: Price)
}

-- Detail: shows the selected product
DATAVIEW dvDetail (DataSource: SELECTION dgProducts) {
  TEXTBOX txtDescription (Label: 'Description', Attribute: Description)
  TEXTBOX txtCategory (Label: 'Category', Attribute: Category)
}
```

## Nested Data Contexts

Widgets inherit the data context from their parent container. A DataView inside a DataView creates a nested context:

```sql
DATAVIEW dvOrder (DataSource: $Order) {
  TEXTBOX txtOrderId (Label: 'Order #', Attribute: OrderId)

  -- Nested DataView for the order's customer (via association)
  DATAVIEW dvCustomer (DataSource: ASSOCIATION Order_Customer) {
    -- Attributes here resolve against Customer
    DYNAMICTEXT txtCustomerName (Content: '{1}', Attribute: Name)
  }
}
```

## Dynamic Text Content

The `DYNAMICTEXT` widget uses `{1}`, `{2}`, etc. as placeholders for attribute values in the `Content` property. The `Attribute` property specifies which attribute fills the first placeholder:

```sql
DYNAMICTEXT txtPrice (Content: 'Price: ${1}', Attribute: Price)
```

## Button Actions with Parameters

Action buttons can pass the current data context to microflows and pages:

```sql
DATAVIEW dvOrder (DataSource: $Order) {
  ACTIONBUTTON btnProcess (
    Caption: 'Process Order',
    Action: MICROFLOW Sales.ACT_ProcessOrder(Order: $Order),
    ButtonStyle: Primary
  )

  ACTIONBUTTON btnEdit (
    Caption: 'Edit',
    Action: PAGE Sales.Order_Edit
  )
}
```

## See Also

- [Pages](./pages.md) -- page overview
- [Page Structure](./page-structure.md) -- data source types in detail
- [Widget Types](./widget-types.md) -- available widgets and their properties
- [Common Patterns](./page-patterns.md) -- master-detail and other binding patterns
