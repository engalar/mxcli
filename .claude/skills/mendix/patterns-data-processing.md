# Data Processing Patterns

Patterns for loops, aggregates, batch processing, and data transformation.

## Loop Patterns

### Basic Loop

```mdl
/**
 * Process all items in a list
 */
CREATE MICROFLOW Module.ProcessItems (
  $Items: List of Module.Item
)
RETURNS Boolean
BEGIN
  DECLARE $ProcessedCount Integer = 0;

  LOOP $Item IN $Items
  BEGIN
    -- Process each item
    CHANGE $Item (ProcessedDate = [%CurrentDateTime%]);
    COMMIT $Item;
    SET $ProcessedCount = $ProcessedCount + 1;
  END LOOP;

  LOG INFO NODE 'Processing' 'Processed ' + $ProcessedCount + ' items';
  RETURN true;
END;
/
```

### Loop with Filtering

```mdl
/**
 * Process only active items
 */
CREATE MICROFLOW Module.ProcessActiveItems (
  $Items: List of Module.Item
)
RETURNS Integer
BEGIN
  DECLARE $Count Integer = 0;

  LOOP $Item IN $Items
  BEGIN
    IF $Item/IsActive THEN
      -- Process active item
      CHANGE $Item (LastProcessed = [%CurrentDateTime%]);
      COMMIT $Item;
      SET $Count = $Count + 1;
    END IF;
  END LOOP;

  RETURN $Count;
END;
/
```

### Loop with Accumulator

```mdl
/**
 * Calculate total value of all orders
 */
CREATE MICROFLOW Module.CalculateOrderTotal (
  $Orders: List of Module.Order
)
RETURNS Decimal
BEGIN
  DECLARE $Total Decimal = 0;

  LOOP $Order IN $Orders
  BEGIN
    SET $Total = $Total + $Order/Amount;
  END LOOP;

  RETURN $Total;
END;
/
```

## Retrieve by Association

Use `RETRIEVE $List FROM $Parent/Module.AssociationName` to retrieve related objects
via association instead of a database XPath query. This is **required** for:

- **Non-persistent entities (NPEs)** — database XPath queries always return empty for NPEs
- **Uncommitted objects** — objects not yet committed to the database
- **JSON mapping results** — imported data structures held in memory

### Persistent Entity Example

```mdl
/**
 * Get all orders for a customer via association
 */
CREATE MICROFLOW Module.GetCustomerOrders (
  $Customer : Module.Customer
)
RETURNS List of Module.Order
BEGIN
  RETRIEVE $Orders FROM $Customer/Module.Order_Customer;
  RETURN $Orders;
END;
/
```

### Non-Persistent Entity Example (NPE)

```mdl
/**
 * Process imported rows from an in-memory result object.
 * Database RETRIEVE would return empty for NPEs — use association retrieve.
 */
CREATE MICROFLOW Module.ProcessImportRows (
  $ImportResult : Module.ImportResult
)
RETURNS Integer
BEGIN
  -- Association retrieve is the ONLY way to get related NPEs
  RETRIEVE $Rows FROM $ImportResult/Module.ImportResult_ImportRow;

  DECLARE $ValidCount Integer = 0;

  LOOP $Row IN $Rows
  BEGIN
    IF $Row/IsValid THEN
      SET $ValidCount = $ValidCount + 1;
    END IF;
  END LOOP;

  RETURN $ValidCount;
END;
/
```

### When to Use Which Retrieve

| Scenario | Syntax | Why |
|----------|--------|-----|
| Query persistent entities by attribute | `RETRIEVE $List FROM Module.Entity WHERE ...` | Database XPath query |
| Get related persistent objects | `RETRIEVE $List FROM $Parent/Module.Association` | Simpler, no XPath needed |
| Get related NPEs / uncommitted objects | `RETRIEVE $List FROM $Parent/Module.Association` | **Only option** — database has no data |
| JSON mapping results (import) | `RETRIEVE $List FROM $Parent/Module.Association` | Mapping creates in-memory NPEs |

**Important:** Association retrieve always returns a list. It does not support WHERE, SORT BY, LIMIT, or OFFSET clauses.

## Aggregate Patterns

Aggregates use **function-call syntax** — there is no `AGGREGATE` keyword.

| Function | Syntax | Returns |
|----------|--------|---------|
| COUNT | `$n = COUNT($list)` | Integer |
| SUM | `$n = SUM($list.Attr)` | Decimal |
| AVERAGE | `$n = AVERAGE($list.Attr)` | Decimal |
| MINIMUM | `$n = MINIMUM($list.Attr)` | Same as attribute |
| MAXIMUM | `$n = MAXIMUM($list.Attr)` | Same as attribute |

**Important:** RETRIEVE implicitly declares its variable — do NOT add a separate DECLARE
before RETRIEVE, or you'll get CE0111 "Duplicate variable name".

### Count Items

```mdl
/**
 * Count active customers
 */
CREATE MICROFLOW Module.CountActiveCustomers ()
RETURNS Integer
BEGIN
  RETRIEVE $Customers FROM Module.Customer
    WHERE IsActive = true;

  $Count = COUNT($Customers);
  RETURN $Count;
END;
/
```

### Sum Values

```mdl
/**
 * Sum order amounts for a customer
 */
CREATE MICROFLOW Module.GetCustomerTotalOrders (
  $Customer: Module.Customer
)
RETURNS Decimal
BEGIN
  RETRIEVE $Orders FROM Module.Order
    WHERE Module.Order_Customer = $Customer;

  $Total = SUM($Orders.Amount);
  RETURN $Total;
END;
/
```

### Average Calculation

```mdl
/**
 * Calculate average order value
 */
CREATE MICROFLOW Module.GetAverageOrderValue ()
RETURNS Decimal
BEGIN
  RETRIEVE $Orders FROM Module.Order;

  $Average = AVERAGE($Orders.Amount);
  RETURN $Average;
END;
/
```

### Min/Max

```mdl
$MinPrice = MINIMUM($Products.Price);
$MaxPrice = MAXIMUM($Products.Price);
```

## List Operations

### Add to List

```mdl
/**
 * Collect matching items into a list
 */
CREATE MICROFLOW Module.CollectHighValueOrders (
  $Orders: List of Module.Order,
  $Threshold: Decimal
)
RETURNS List of Module.Order
BEGIN
  DECLARE $HighValue List of Module.Order = empty;

  LOOP $Order IN $Orders
  BEGIN
    IF $Order/Amount > $Threshold THEN
      ADD $Order TO $HighValue;
    END IF;
  END LOOP;

  RETURN $HighValue;
END;
/
```

### Remove from List

```mdl
/**
 * Remove inactive items from list
 */
CREATE MICROFLOW Module.FilterActiveItems (
  $Items: List of Module.Item
)
RETURNS List of Module.Item
BEGIN
  DECLARE $ToRemove List of Module.Item = empty;

  -- Collect items to remove
  LOOP $Item IN $Items
  BEGIN
    IF NOT $Item/IsActive THEN
      ADD $Item TO $ToRemove;
    END IF;
  END LOOP;

  -- Remove collected items
  LOOP $Item IN $ToRemove
  BEGIN
    REMOVE $Item FROM $Items;
  END LOOP;

  RETURN $Items;
END;
/
```

## Batch Processing

### Process in Batches

```mdl
/**
 * Process large dataset in batches
 * Commits after each batch to avoid memory issues
 */
CREATE MICROFLOW Module.BatchProcess (
  $Items: List of Module.Item,
  $BatchSize: Integer
)
RETURNS Integer
BEGIN
  DECLARE $Processed Integer = 0;
  DECLARE $BatchCount Integer = 0;

  LOOP $Item IN $Items
  BEGIN
    -- Process item
    CHANGE $Item (Status = 'Processed');

    SET $BatchCount = $BatchCount + 1;
    SET $Processed = $Processed + 1;

    -- Commit batch
    IF $BatchCount >= $BatchSize THEN
      COMMIT $Item;
      SET $BatchCount = 0;
      LOG INFO NODE 'Batch' 'Processed ' + $Processed + ' items';
    END IF;
  END LOOP;

  -- Final commit for remaining items
  IF $BatchCount > 0 THEN
    LOG INFO NODE 'Batch' 'Final batch: ' + $Processed + ' total';
  END IF;

  RETURN $Processed;
END;
/
```

## Data Transformation

### Copy Entity

```mdl
/**
 * Create a copy of an order
 */
CREATE MICROFLOW Module.CopyOrder (
  $Source: Module.Order
)
RETURNS Module.Order
BEGIN
  DECLARE $Copy AS Module.Order;

  $Copy = CREATE Module.Order (
    OrderNumber = 'COPY-' + $Source/OrderNumber,
    Amount = $Source/Amount,
    Status = 'Draft',
    CreatedDate = [%CurrentDateTime%]
  );

  -- Copy association
  SET $Copy/Module.Order_Customer = $Source/Module.Order_Customer;

  COMMIT $Copy;
  RETURN $Copy;
END;
/
```

### Transform List

```mdl
/**
 * Create summary records from detail records
 */
CREATE MICROFLOW Module.CreateOrderSummaries (
  $Orders: List of Module.Order
)
RETURNS List of Module.OrderSummary
BEGIN
  DECLARE $Summaries List of Module.OrderSummary = empty;
  DECLARE $Summary AS Module.OrderSummary;

  LOOP $Order IN $Orders
  BEGIN
    $Summary = CREATE Module.OrderSummary (
      OrderNumber = $Order/OrderNumber,
      TotalAmount = $Order/Amount,
      CustomerName = $Order/Module.Order_Customer/Name
    );
    ADD $Summary TO $Summaries;
  END LOOP;

  RETURN $Summaries;
END;
/
```

## Error Handling in Loops

### Continue on Error

```mdl
/**
 * Process items, log errors but continue
 */
CREATE MICROFLOW Module.ProcessWithErrorHandling (
  $Items: List of Module.Item
)
RETURNS Integer
BEGIN
  DECLARE $Processed Integer = 0;
  DECLARE $Errors Integer = 0;

  LOOP $Item IN $Items
  BEGIN
    IF $Item/Data = empty THEN
      LOG WARNING NODE 'Process' 'Skipping item with empty data: ' + $Item/Code;
      SET $Errors = $Errors + 1;
    ELSE
      CHANGE $Item (Status = 'Processed');
      COMMIT $Item;
      SET $Processed = $Processed + 1;
    END IF;
  END LOOP;

  LOG INFO NODE 'Process' 'Completed: ' + $Processed + ' processed, ' + $Errors + ' errors';
  RETURN $Processed;
END;
/
```

## Best Practices

1. **Commit inside loops carefully**: Can cause performance issues on large sets
2. **Use batch commits**: Commit every N records for large datasets
3. **Log progress**: Add logging for long-running operations
4. **Handle errors gracefully**: Don't let one bad record stop the whole process
5. **Return counts**: Help callers know what was processed
6. **Use meaningful variable names**: `$ProcessedCount` not `$c`
