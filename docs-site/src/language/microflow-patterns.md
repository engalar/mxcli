# Common Patterns

This page shows frequently used microflow patterns in MDL, including CRUD operations, validation, batch processing, and important anti-patterns to avoid.

## CRUD Patterns

### Create and Commit

```sql
CREATE MICROFLOW Sales.ACT_CreateCustomer
FOLDER 'Customer'
BEGIN
  DECLARE $Name String;
  DECLARE $Email String;
  DECLARE $Customer Sales.Customer;

  $Customer = CREATE Sales.Customer (
    Name = $Name,
    Email = $Email,
    CreatedDate = [%CurrentDateTime%],
    IsActive = true
  );
  COMMIT $Customer;

  SHOW PAGE Sales.Customer_Edit ($Customer = $Customer);
  RETURN $Customer;
END;
```

### Retrieve and Update

```sql
CREATE MICROFLOW Sales.ACT_DeactivateCustomer
FOLDER 'Customer'
BEGIN
  DECLARE $Customer Sales.Customer;

  CHANGE $Customer (
    IsActive = false,
    DeactivatedDate = [%CurrentDateTime%]
  );
  COMMIT $Customer;
END;
```

### Retrieve with Filtering

```sql
CREATE MICROFLOW Sales.ACT_GetActiveOrders
FOLDER 'Orders'
BEGIN
  RETRIEVE $Orders FROM Sales.Order
    WHERE Status = 'Active'
    AND OrderDate > [%BeginOfCurrentMonth%];

  RETURN $Orders;
END;
```

### Delete with Confirmation

```sql
CREATE MICROFLOW Sales.ACT_DeleteOrder
FOLDER 'Orders'
BEGIN
  DECLARE $Order Sales.Order;

  -- Delete related order lines first
  RETRIEVE $Lines FROM $Order/Sales.OrderLine_Order;
  LOOP $Line IN $Lines
  BEGIN
    DELETE $Line;
  END LOOP;

  DELETE $Order;
END;
```

## Validation Pattern

Validate an object before saving, showing feedback on invalid fields:

```sql
CREATE MICROFLOW Sales.ACT_SaveCustomer
FOLDER 'Customer'
BEGIN
  DECLARE $Customer Sales.Customer;
  DECLARE $IsValid Boolean = true;

  -- Validate required fields
  IF $Customer/Name = empty THEN
    VALIDATION FEEDBACK $Customer/Name MESSAGE 'Name is required';
    SET $IsValid = false;
  END IF;

  IF $Customer/Email = empty THEN
    VALIDATION FEEDBACK $Customer/Email MESSAGE 'Email is required';
    SET $IsValid = false;
  END IF;

  -- Check for duplicate email
  IF $IsValid THEN
    RETRIEVE $Existing FROM Sales.Customer
      WHERE Email = $Customer/Email
      LIMIT 1;
    IF $Existing != empty THEN
      VALIDATION FEEDBACK $Customer/Email MESSAGE 'A customer with this email already exists';
      SET $IsValid = false;
    END IF;
  END IF;

  -- Save only if valid
  IF $IsValid THEN
    COMMIT $Customer;
    CLOSE PAGE;
  END IF;
END;
```

## Batch Processing Pattern

Process a list of items one at a time with error tracking:

```sql
CREATE MICROFLOW Sales.ACT_ProcessPendingOrders
FOLDER 'Batch'
BEGIN
  DECLARE $SuccessCount Integer = 0;
  DECLARE $ErrorCount Integer = 0;

  RETRIEVE $PendingOrders FROM Sales.Order
    WHERE Status = 'Pending';

  LOOP $Order IN $PendingOrders
  BEGIN
    CALL MICROFLOW Sales.SUB_ProcessOrder (
      Order = $Order
    ) ON ERROR {
      LOG ERROR NODE 'BatchProcess' 'Failed to process order: ' + $Order/OrderNumber;
      SET $ErrorCount = $ErrorCount + 1;
      CONTINUE;
    };

    SET $SuccessCount = $SuccessCount + 1;
  END LOOP;

  LOG INFO NODE 'BatchProcess' 'Batch complete: '
    + toString($SuccessCount) + ' succeeded, '
    + toString($ErrorCount) + ' failed';
END;
```

## Sub-Microflow Pattern

Break complex logic into reusable sub-microflows:

```sql
-- Main microflow delegates to sub-microflow
CREATE MICROFLOW Sales.ACT_PlaceOrder
FOLDER 'Orders'
BEGIN
  DECLARE $Order Sales.Order;

  -- Validate
  $IsValid = CALL MICROFLOW Sales.SUB_ValidateOrder (Order = $Order);
  IF NOT $IsValid THEN
    RETURN false;
  END IF;

  -- Calculate totals
  CALL MICROFLOW Sales.SUB_CalculateOrderTotals (Order = $Order);

  -- Finalize
  CHANGE $Order (Status = 'Placed');
  COMMIT $Order WITH EVENTS;

  RETURN true;
END;
```

```sql
-- Reusable sub-microflow
CREATE MICROFLOW Sales.SUB_CalculateOrderTotals
FOLDER 'Orders'
BEGIN
  DECLARE $Order Sales.Order;
  DECLARE $Total Decimal = 0;

  RETRIEVE $Lines FROM $Order/Sales.OrderLine_Order;
  LOOP $Line IN $Lines
  BEGIN
    SET $Total = $Total + ($Line/Quantity * $Line/UnitPrice);
  END LOOP;

  CHANGE $Order (TotalAmount = $Total);
END;
```

## Association Retrieval Pattern

Retrieve related objects through associations:

```sql
CREATE MICROFLOW Sales.ACT_GetOrderSummary
FOLDER 'Orders'
BEGIN
  DECLARE $Order Sales.Order;

  -- Retrieve related customer
  RETRIEVE $Customer FROM $Order/Sales.Order_Customer;

  -- Retrieve order lines
  RETRIEVE $Lines FROM $Order/Sales.OrderLine_Order;

  -- Use the retrieved data
  LOG INFO 'Order ' + $Order/OrderNumber
    + ' for customer ' + $Customer/Name
    + ' has ' + toString(length($Lines)) + ' lines';

  RETURN $Order;
END;
```

## Lookup Pattern (List Search)

When you have a list and need to find a matching item, retrieve from the list variable:

```sql
CREATE MICROFLOW Import.ACT_MatchDepartments
FOLDER 'Import'
BEGIN
  DECLARE $Employees List of Import.Employee = empty;
  DECLARE $Departments List of Import.Department = empty;

  -- Retrieve all departments for lookup
  RETRIEVE $Departments FROM Import.Department;

  LOOP $Employee IN $Employees
  BEGIN
    -- O(N) lookup: retrieve from in-memory list
    RETRIEVE $Dept FROM $Departments
      WHERE Name = $Employee/DepartmentName
      LIMIT 1;

    IF $Dept != empty THEN
      CHANGE $Employee (Employee_Department = $Dept);
      COMMIT $Employee;
    END IF;
  END LOOP;
END;
```

## Error Handling Pattern

Wrap risky operations with error handlers:

```sql
CREATE MICROFLOW Integration.ACT_SyncData
FOLDER 'Integration'
BEGIN
  DECLARE $Config Integration.SyncConfig;
  DECLARE $Success Boolean = false;

  RETRIEVE $Config FROM Integration.SyncConfig LIMIT 1;

  @caption 'Call external API'
  $Response = CALL MICROFLOW Integration.SUB_CallExternalAPI (
    Config = $Config
  ) ON ERROR {
    LOG ERROR NODE 'Integration' 'External API call failed';
    RETURN false;
  };

  IF $Response != empty THEN
    @caption 'Process response'
    CALL MICROFLOW Integration.SUB_ProcessResponse (
      Response = $Response
    ) ON ERROR {
      LOG ERROR NODE 'Integration' 'Failed to process response';
      RETURN false;
    };
    SET $Success = true;
  END IF;

  RETURN $Success;
END;
```

---

## Anti-Patterns (Avoid These)

The following patterns are common mistakes that cause bugs, performance problems, or parse errors. The `mxcli check` command detects these automatically.

### Never Create Empty List Variables as Loop Sources

Creating an empty list and then immediately looping over it is always wrong. If you need to process a list, accept it as a microflow parameter or retrieve it from the database.

```sql
-- WRONG: empty list means the loop never executes
DECLARE $Items List of Sales.Item = empty;
LOOP $Item IN $Items
BEGIN
  -- this code never runs!
END LOOP;
```

```sql
-- CORRECT: accept the list as a parameter
CREATE MICROFLOW Sales.ACT_ProcessItems
BEGIN
  DECLARE $Items List of Sales.Item = empty;  -- parameter, filled by caller

  RETRIEVE $Items FROM Sales.Item WHERE Status = 'Pending';  -- or retrieve from DB

  LOOP $Item IN $Items
  BEGIN
    -- process each item
  END LOOP;
END;
```

### Never Use Nested Loops for List Matching

Nested loops are O(N^2) and cause severe performance problems with large data sets. Instead, loop over the primary list and use `RETRIEVE ... FROM $List WHERE ... LIMIT 1` to find matches.

```sql
-- WRONG: O(N^2) nested loop
LOOP $Employee IN $Employees
BEGIN
  LOOP $Department IN $Departments
  BEGIN
    IF $Department/Name = $Employee/DeptName THEN
      CHANGE $Employee (Employee_Department = $Department);
    END IF;
  END LOOP;
END LOOP;
```

```sql
-- CORRECT: O(N) lookup from list
LOOP $Employee IN $Employees
BEGIN
  RETRIEVE $Dept FROM $Departments
    WHERE Name = $Employee/DeptName
    LIMIT 1;

  IF $Dept != empty THEN
    CHANGE $Employee (Employee_Department = $Dept);
  END IF;
END LOOP;
```

### Use Append Logic When Merging, Not Overwrite

When merging data from multiple sources, append new values rather than overwriting existing ones:

```sql
-- WRONG: overwrites existing notes
CHANGE $Customer (Notes = $NewNotes);
```

```sql
-- CORRECT: append with guard
IF $NewNotes != empty THEN
  CHANGE $Customer (Notes = $Customer/Notes + '\n' + $NewNotes);
END IF;
```

## Naming Conventions

Follow consistent naming patterns for microflows:

| Prefix | Purpose | Example |
|--------|---------|---------|
| `ACT_` | User-triggered action | `Sales.ACT_CreateOrder` |
| `SUB_` | Sub-microflow (called by other microflows) | `Sales.SUB_CalculateTotal` |
| `DS_` | Data source for a page/widget | `Sales.DS_GetActiveOrders` |
| `VAL_` | Validation logic | `Sales.VAL_ValidateOrder` |
| `SE_` | Scheduled event handler | `Sales.SE_ProcessBatch` |
| `BCO_` | Before commit event | `Sales.BCO_Order` |
| `ACO_` | After commit event | `Sales.ACO_Order` |

## Validation Checklist

Before presenting a microflow to the user, verify these rules:

1. Every `DECLARE` has a valid type (`String`, `Integer`, `Boolean`, `Decimal`, `DateTime`, `Module.Entity`, or `List of Module.Entity`)
2. Entity declarations do not use `= empty` (only list declarations do)
3. Every flow path ends with a `RETURN` statement (or the microflow returns void)
4. No empty list variables are used as loop sources
5. No nested loops for list matching
6. All entity and microflow qualified names use the `Module.Name` format
7. `VALIDATION FEEDBACK` uses the `$Variable/Attribute MESSAGE 'text'` syntax

Run the syntax checker to catch issues:

```bash
mxcli check script.mdl
mxcli check script.mdl -p app.mpr --references
```
