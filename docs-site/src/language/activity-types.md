# Activity Types

This page documents every activity type available in MDL microflows. Activities are the individual steps that make up a microflow's logic.

## Object Operations

### CREATE

Creates a new object of the specified entity type and assigns attribute values:

```sql
$Order = CREATE Sales.Order (
  OrderDate = [%CurrentDateTime%],
  Status = 'Draft',
  TotalAmount = 0
);
```

Attribute assignments are comma-separated `Name = value` pairs. The result is assigned to a variable.

### CHANGE

Modifies attributes of an existing object:

```sql
CHANGE $Order (
  Status = 'Confirmed',
  TotalAmount = $CalculatedTotal
);
```

Multiple attributes can be changed in a single `CHANGE` statement.

### COMMIT

Persists an object (or its changes) to the database:

```sql
COMMIT $Order;
```

Options:
- `WITH EVENTS` -- triggers event handlers (before/after commit microflows)
- `REFRESH` -- refreshes the object in the client after committing

```sql
COMMIT $Order WITH EVENTS;
COMMIT $Order REFRESH;
COMMIT $Order WITH EVENTS REFRESH;
```

### DELETE

Deletes an object from the database:

```sql
DELETE $Order;
```

### ROLLBACK

Reverts uncommitted changes to an object, restoring it to its last committed state:

```sql
ROLLBACK $Order;
ROLLBACK $Order REFRESH;
```

The `REFRESH` option refreshes the object in the client.

## Retrieval

### RETRIEVE from Database (XPath)

Retrieves objects from the database using an optional XPath-style WHERE clause:

```sql
-- Retrieve a single object
RETRIEVE $Customer FROM Sales.Customer
  WHERE Email = $InputEmail
  LIMIT 1;

-- Retrieve a list of objects
RETRIEVE $ActiveOrders FROM Sales.Order
  WHERE Status = 'Active';

-- Retrieve all objects of an entity
RETRIEVE $AllProducts FROM Sales.Product;

-- Retrieve with multiple conditions
RETRIEVE $RecentOrders FROM Sales.Order
  WHERE Status = 'Active'
  AND OrderDate > [%BeginOfCurrentDay%]
  LIMIT 50;
```

When `LIMIT 1` is specified, the result is a single entity object. Otherwise, the result is a list.

### RETRIEVE by Association

Retrieves objects by following an association from an existing object:

```sql
-- Retrieve related objects via association
RETRIEVE $Lines FROM $Order/Sales.OrderLine_Order;

-- Retrieve a single associated object
RETRIEVE $Customer FROM $Order/Sales.Order_Customer;
```

The association path uses the format `$Variable/Module.AssociationName`.

### RETRIEVE from Variable List

Retrieves from an in-memory list variable:

```sql
RETRIEVE $Match FROM $CustomerList
  WHERE Email = $SearchEmail
  LIMIT 1;
```

## Call Activities

### CALL MICROFLOW

Calls another microflow, passing parameters and optionally receiving a return value:

```sql
-- Call with return value
$Total = CALL MICROFLOW Sales.SUB_CalculateTotal (
  OrderLines = $Lines
);

-- Call without return value
CALL MICROFLOW Sales.SUB_SendNotification (
  Customer = $Customer,
  Message = 'Order confirmed'
);

-- Call with no parameters
$Config = CALL MICROFLOW Admin.GetSystemConfig ();
```

### CALL NANOFLOW

Calls a nanoflow from within a microflow:

```sql
$Result = CALL NANOFLOW MyModule.ValidateInput (
  InputValue = $Value
);
```

### CALL JAVA ACTION

Calls a Java action:

```sql
$Hash = CALL JAVA ACTION MyModule.HashPassword (
  Password = $RawPassword
);
```

Java action parameters follow the same `Name = value` syntax.

## UI Activities

### SHOW PAGE

Opens a page, passing parameters:

```sql
SHOW PAGE Sales.Order_Edit ($Order = $Order);
```

The parameter syntax uses `$PageParam = $MicroflowVar`. Multiple parameters are comma-separated:

```sql
SHOW PAGE Sales.OrderDetail (
  $Order = $Order,
  $Customer = $Customer
);
```

An alternate syntax using colon notation is also supported:

```sql
SHOW PAGE Sales.Order_Edit (Order: $Order);
```

### CLOSE PAGE

Closes the current page:

```sql
CLOSE PAGE;
```

## Validation

### VALIDATION FEEDBACK

Displays a validation error message on a specific attribute of an object:

```sql
VALIDATION FEEDBACK $Customer/Email MESSAGE 'Email address is required';
VALIDATION FEEDBACK $Order/TotalAmount MESSAGE 'Total must be greater than zero';
```

The syntax is `VALIDATION FEEDBACK $Variable/AttributeName MESSAGE 'message text'`.

## Logging

### LOG

Writes a message to the Mendix runtime log:

```sql
LOG INFO 'Order created successfully';
LOG WARNING 'Customer has no email address';
LOG ERROR 'Failed to process payment';
```

Log levels: `INFO`, `WARNING`, `ERROR`.

Optionally specify a log node name:

```sql
LOG INFO NODE 'OrderProcessing' 'Order created: ' + $Order/OrderNumber;
LOG ERROR NODE 'PaymentGateway' 'Payment failed for order ' + $Order/OrderNumber;
```

## Database Query Execution

### EXECUTE DATABASE QUERY

Executes a Database Connector query defined in the project:

```sql
-- Basic execution (3-part qualified name: Module.Connection.Query)
$Result = EXECUTE DATABASE QUERY MyModule.MyConn.GetCustomers;

-- Dynamic SQL query
$Result = EXECUTE DATABASE QUERY MyModule.MyConn.SearchQuery
  DYNAMIC 'SELECT * FROM customers WHERE name LIKE ?';

-- With parameters
$Result = EXECUTE DATABASE QUERY MyModule.MyConn.GetByEmail
  PARAMETERS ($EmailParam = $Email);

-- With runtime connection override
$Result = EXECUTE DATABASE QUERY MyModule.MyConn.GetData
  CONNECTION $RuntimeConnString;
```

The query name follows a three-part naming convention: `Module.ConnectionName.QueryName`.

> **Note:** Error handling (`ON ERROR`) is not supported on `EXECUTE DATABASE QUERY` activities.

## Summary Table

| Activity | Syntax | Returns |
|----------|--------|---------|
| Create object | `$Var = CREATE Module.Entity (Attr = val);` | Entity object |
| Change object | `CHANGE $Var (Attr = val);` | -- |
| Commit | `COMMIT $Var [WITH EVENTS] [REFRESH];` | -- |
| Delete | `DELETE $Var;` | -- |
| Rollback | `ROLLBACK $Var [REFRESH];` | -- |
| Retrieve (DB) | `RETRIEVE $Var FROM Module.Entity [WHERE ...] [LIMIT n];` | Entity or list |
| Retrieve (assoc) | `RETRIEVE $Var FROM $Obj/Module.Assoc;` | Entity or list |
| Call microflow | `$Var = CALL MICROFLOW Module.Name (Param = $val);` | Any type |
| Call nanoflow | `$Var = CALL NANOFLOW Module.Name (Param = $val);` | Any type |
| Call Java action | `$Var = CALL JAVA ACTION Module.Name (Param = val);` | Any type |
| Show page | `SHOW PAGE Module.Page ($Param = $val);` | -- |
| Close page | `CLOSE PAGE;` | -- |
| Validation | `VALIDATION FEEDBACK $Var/Attr MESSAGE 'msg';` | -- |
| Log | `LOG INFO\|WARNING\|ERROR [NODE 'name'] 'msg';` | -- |
| DB query | `$Var = EXECUTE DATABASE QUERY Module.Conn.Query;` | Result set |
| Assignment | `SET $Var = expression;` | -- |
