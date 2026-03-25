# Control Flow

MDL microflows support conditional branching, loops, and error handling to control the execution path of your logic.

## IF / ELSE

Conditional branching executes different activities based on a boolean expression.

### Basic IF

```sql
IF $Order/TotalAmount > 1000 THEN
  CHANGE $Order (DiscountApplied = true);
END IF;
```

### IF / ELSE

```sql
IF $Customer/Email != empty THEN
  CALL MICROFLOW Sales.SUB_SendEmail (Customer = $Customer);
ELSE
  LOG WARNING 'Customer has no email address';
END IF;
```

### Nested IF

Since MDL does not support `CASE`/`WHEN` (switch statements), use nested `IF...ELSE` blocks:

```sql
IF $Order/Status = 'Draft' THEN
  CHANGE $Order (Status = 'Submitted');
ELSE
  IF $Order/Status = 'Submitted' THEN
    CHANGE $Order (Status = 'Approved');
  ELSE
    IF $Order/Status = 'Approved' THEN
      CHANGE $Order (Status = 'Shipped');
    ELSE
      LOG WARNING 'Unexpected order status: ' + $Order/Status;
    END IF;
  END IF;
END IF;
```

### Complex Conditions

Conditions support `AND`, `OR`, and parentheses:

```sql
IF $Amount > 0 AND $Customer != empty THEN
  COMMIT $Order;
END IF;

IF ($Status = 'Active' OR $Status = 'Pending') AND $IsValid = true THEN
  CALL MICROFLOW Sales.ProcessOrder (Order = $Order);
END IF;
```

## LOOP (FOR EACH)

Iterates over each item in a list:

```sql
LOOP $Line IN $OrderLines
BEGIN
  CHANGE $Line (
    LineTotal = $Line/Quantity * $Line/UnitPrice
  );
  COMMIT $Line;
END LOOP;
```

The loop variable (`$Line`) is automatically declared and takes the entity type of the list.

### LOOP with Nested Logic

```sql
LOOP $Order IN $PendingOrders
BEGIN
  IF $Order/TotalAmount > 0 THEN
    CHANGE $Order (Status = 'Confirmed');
    COMMIT $Order;
  ELSE
    DELETE $Order;
  END IF;
END LOOP;
```

### BREAK and CONTINUE

Use `BREAK` to exit a loop early, and `CONTINUE` to skip to the next iteration:

```sql
LOOP $Item IN $Items
BEGIN
  IF $Item/IsInvalid = true THEN
    CONTINUE;
  END IF;

  IF $Item/Type = 'StopSignal' THEN
    BREAK;
  END IF;

  CALL MICROFLOW Sales.ProcessItem (Item = $Item);
END LOOP;
```

## WHILE Loop

Executes a block repeatedly as long as a condition remains true:

```sql
DECLARE $Counter Integer = 0;

WHILE $Counter < 10
BEGIN
  SET $Counter = $Counter + 1;
  LOG INFO 'Iteration: ' + toString($Counter);
END WHILE;
```

> **Caution:** Ensure the condition will eventually become false to avoid infinite loops.

## Error Handling

### ON ERROR Suffix

Error handling is applied as a suffix to individual activities. There is no `TRY...CATCH` block in MDL. Instead, you specify error handling behavior on specific activities.

#### ON ERROR CONTINUE

Ignores the error and continues to the next activity:

```sql
COMMIT $Order ON ERROR CONTINUE;
```

#### ON ERROR ROLLBACK

Rolls back the current transaction and continues:

```sql
DELETE $Order ON ERROR ROLLBACK;
```

#### ON ERROR with Handler Block

Executes a custom error handling block when the activity fails:

```sql
COMMIT $Order ON ERROR {
  LOG ERROR 'Failed to commit order: ' + $Order/OrderNumber;
  ROLLBACK $Order;
};
```

The handler block can contain any activities -- logging, rollback, showing validation messages, etc.

### Error Handling Examples

```sql
-- Continue despite retrieval failure
RETRIEVE $Config FROM Admin.SystemConfig LIMIT 1 ON ERROR CONTINUE;

-- Custom error handler for external call
$Response = CALL MICROFLOW Integration.CallExternalAPI (
  Payload = $RequestBody
) ON ERROR {
  LOG ERROR NODE 'Integration' 'External API call failed';
  SET $Response = empty;
};

-- Rollback on commit failure
COMMIT $Order WITH EVENTS ON ERROR ROLLBACK;
```

> **Note:** `ON ERROR` is not supported on `EXECUTE DATABASE QUERY` activities.

## Unsupported Control Flow

The following constructs are **not** supported in MDL and will cause parse errors:

| Unsupported | Use Instead |
|-------------|-------------|
| `CASE ... WHEN ... END CASE` | Nested `IF ... ELSE ... END IF` |
| `TRY ... CATCH ... END TRY` | `ON ERROR { ... }` blocks on individual activities |

## Complete Example

```sql
CREATE MICROFLOW Sales.ACT_ProcessBatch
FOLDER 'Batch'
BEGIN
  DECLARE $Orders List of Sales.Order = empty;
  DECLARE $SuccessCount Integer = 0;
  DECLARE $ErrorCount Integer = 0;

  RETRIEVE $Orders FROM Sales.Order
    WHERE Status = 'Pending';

  LOOP $Order IN $Orders
  BEGIN
    IF $Order/TotalAmount <= 0 THEN
      LOG WARNING 'Skipping order with zero amount: ' + $Order/OrderNumber;
      CONTINUE;
    END IF;

    @caption 'Process order'
    CALL MICROFLOW Sales.SUB_ProcessSingleOrder (
      Order = $Order
    ) ON ERROR {
      LOG ERROR 'Failed to process order: ' + $Order/OrderNumber;
      SET $ErrorCount = $ErrorCount + 1;
      CONTINUE;
    };

    SET $SuccessCount = $SuccessCount + 1;
  END LOOP;

  LOG INFO 'Batch complete: ' + toString($SuccessCount) + ' processed, '
    + toString($ErrorCount) + ' errors';
END;
```
