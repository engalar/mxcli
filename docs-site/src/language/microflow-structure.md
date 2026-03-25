# Structure

This page covers the structural elements of a microflow definition: the `CREATE MICROFLOW` syntax, parameters, variable declarations, return values, and annotations.

## Full CREATE MICROFLOW Syntax

```sql
CREATE [OR REPLACE] MICROFLOW <Module.Name>
  [FOLDER '<path>']
BEGIN
  [<declarations>]
  [<activities>]
  [RETURN <value>;]
END;
```

The `OR REPLACE` modifier overwrites an existing microflow of the same name. The `FOLDER` clause organizes the microflow within the module's folder structure.

## Parameters

Parameters are declared with `DECLARE` at the top of the `BEGIN...END` block. They define the inputs to the microflow.

### Primitive Parameters

```sql
DECLARE $Name String;
DECLARE $Count Integer;
DECLARE $IsActive Boolean;
DECLARE $Amount Decimal;
DECLARE $StartDate DateTime;
```

Supported primitive types: `String`, `Integer`, `Long`, `Boolean`, `Decimal`, `DateTime`.

### Entity Parameters

Entity parameters receive a single object:

```sql
DECLARE $Customer MyModule.Customer;
```

> **Important:** Do not use `= empty` or `AS` with entity declarations. The correct syntax is simply `DECLARE $Var Module.Entity;`.

### List Parameters

List parameters receive a list of objects:

```sql
DECLARE $Orders List of Sales.Order = empty;
```

The `= empty` initializer creates an empty list. This is required for list declarations.

### Parameter vs. Local Variable

In MDL, all `DECLARE` statements at the top of a microflow are treated as parameters. Variables created by activities (such as `$Var = CREATE ...` or `RETRIEVE $Var ...`) are local variables. If you need a local variable with a default value, use `DECLARE` followed by `SET`:

```sql
DECLARE $Counter Integer = 0;
SET $Counter = 10;
```

## Variables and Assignment

### Variable Declaration

```sql
DECLARE $Message String = 'Hello';
DECLARE $Total Decimal = 0;
DECLARE $Found Boolean = false;
```

### Assignment with SET

Change the value of an already-declared variable:

```sql
SET $Counter = $Counter + 1;
SET $FullName = $FirstName + ' ' + $LastName;
SET $IsValid = $Amount > 0;
```

The variable must be declared before it can be assigned.

### Variables from Activities

Activities like CREATE and RETRIEVE produce result variables:

```sql
$Order = CREATE Sales.Order (Status = 'New');
RETRIEVE $Customer FROM Sales.Customer WHERE Email = $Email LIMIT 1;
$Result = CALL MICROFLOW Sales.CalculateTotal (Order = $Order);
```

## Return Values

Every flow path should end with a `RETURN` statement. The return type is inferred from the returned value.

### Returning a Primitive

```sql
RETURN true;
RETURN $Total;
RETURN 'Success';
```

### Returning an Object

```sql
RETURN $Order;
```

### Returning a List

```sql
RETURN $FilteredOrders;
```

### Returning Nothing

If the microflow returns nothing (void), you can omit the `RETURN` or use:

```sql
RETURN;
```

## Annotations

Annotations are metadata decorators placed **before** an activity. They control visual layout and documentation in Mendix Studio Pro.

### Position

Set the canvas position of the next activity:

```sql
@position(200, 100)
$Order = CREATE Sales.Order (Status = 'New');
```

### Caption

Set a custom caption displayed on the activity in the canvas:

```sql
@caption 'Create new order'
$Order = CREATE Sales.Order (Status = 'New');
```

### Color

Set the background color of the activity:

```sql
@color Green
COMMIT $Order;
```

### Annotation (Visual Note)

Attach a visual annotation note to the next activity:

```sql
@annotation 'This step validates the input before saving'
VALIDATION FEEDBACK $Customer/Email MESSAGE 'Email is required';
```

### Combining Annotations

Multiple annotations can be stacked before a single activity:

```sql
@position(300, 200)
@caption 'Validate and save'
@color Green
@annotation 'Final step: commit the validated order'
COMMIT $Order WITH EVENTS;
```

## Complete Example

```sql
CREATE MICROFLOW Sales.ACT_ProcessOrder
FOLDER 'Orders/Processing'
BEGIN
  -- Parameters
  DECLARE $Order Sales.Order;
  DECLARE $ApplyDiscount Boolean;

  -- Local variable
  DECLARE $Total Decimal = 0;

  -- Retrieve order lines
  RETRIEVE $Lines FROM $Order/Sales.OrderLine_Order;

  -- Calculate total
  @caption 'Calculate total'
  $Total = CALL MICROFLOW Sales.SUB_CalculateTotal (
    OrderLines = $Lines
  );

  -- Apply discount if requested
  IF $ApplyDiscount THEN
    SET $Total = $Total * 0.9;
  END IF;

  -- Update order
  CHANGE $Order (
    TotalAmount = $Total,
    Status = 'Processed'
  );
  COMMIT $Order;

  RETURN $Order;
END;
```
