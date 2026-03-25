# Expressions

Expressions in MDL are used in conditions, attribute assignments, variable assignments, and log messages within microflows. They follow Mendix expression syntax.

## Literals

### String Literals

Strings are enclosed in single quotes:

```sql
'Hello, world'
'Order #1234'
```

To include a single quote within a string, double it:

```sql
'it''s here'
'Customer''s address'
```

> **Important:** Do not use backslash escaping (`\'`). Mendix expression syntax requires doubled single quotes.

### Numeric Literals

```sql
42          -- Integer
3.14        -- Decimal
-100        -- Negative integer
1.5e10      -- Scientific notation
```

### Boolean Literals

```sql
true
false
```

### Empty

The `empty` literal represents a null/undefined value:

```sql
DECLARE $List List of Sales.Order = empty;

IF $Customer = empty THEN
  LOG WARNING 'No customer found';
END IF;
```

## Operators

### Arithmetic Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition / string concatenation | `$Price + $Tax` |
| `-` | Subtraction | `$Total - $Discount` |
| `*` | Multiplication | `$Quantity * $UnitPrice` |
| `div` | Division | `$Total div $Count` |
| `mod` | Modulo (remainder) | `$Index mod 2` |

String concatenation uses `+`:

```sql
SET $FullName = $FirstName + ' ' + $LastName;
LOG INFO 'Processing order: ' + $Order/OrderNumber;
```

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equal | `$Status = 'Active'` |
| `!=` | Not equal | `$Status != 'Closed'` |
| `>` | Greater than | `$Amount > 1000` |
| `<` | Less than | `$Count < 10` |
| `>=` | Greater than or equal | `$Age >= 18` |
| `<=` | Less than or equal | `$Score <= 100` |

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `AND` | Logical AND | `$IsActive AND $HasEmail` |
| `OR` | Logical OR | `$IsAdmin OR $IsManager` |
| `NOT` | Logical NOT | `NOT $IsDeleted` |

Parentheses control precedence:

```sql
IF ($Status = 'Active' OR $Status = 'Pending') AND $Amount > 0 THEN
  -- ...
END IF;
```

## Attribute Access

Access attributes of an object variable with `/`:

```sql
$Order/TotalAmount
$Customer/Email
$Line/Quantity
```

This is used in conditions, assignments, and expressions:

```sql
IF $Order/TotalAmount > 1000 THEN
  SET $Discount = $Order/TotalAmount * 0.1;
END IF;
```

## Date/Time Tokens

Mendix provides built-in date/time tokens enclosed in `[% ... %]`:

| Token | Description |
|-------|-------------|
| `[%CurrentDateTime%]` | Current date and time |
| `[%BeginOfCurrentDay%]` | Start of today (00:00) |
| `[%EndOfCurrentDay%]` | End of today (23:59:59) |
| `[%BeginOfCurrentWeek%]` | Start of the current week |
| `[%BeginOfCurrentMonth%]` | Start of the current month |
| `[%BeginOfCurrentYear%]` | Start of the current year |

Usage in expressions:

```sql
$Order = CREATE Sales.Order (
  OrderDate = [%CurrentDateTime%],
  DueDate = [%EndOfCurrentDay%]
);

RETRIEVE $TodayOrders FROM Sales.Order
  WHERE OrderDate > [%BeginOfCurrentDay%];
```

## String Templates

String concatenation with `+` is the primary way to build dynamic strings:

```sql
SET $Message = 'Order ' + $Order/OrderNumber + ' has been ' + $Order/Status;
LOG INFO NODE 'Orders' 'Total: ' + toString($Total) + ' for customer ' + $Customer/Name;
```

## Type Conversion

Use built-in functions for type conversion in expressions:

```sql
toString($IntValue)           -- Integer/Decimal/Boolean to String
```

## Enumeration Values

Reference enumeration values with their qualified name:

```sql
$Order = CREATE Sales.Order (
  Status = Sales.OrderStatus.Draft
);

IF $Order/Status = Sales.OrderStatus.Confirmed THEN
  -- process confirmed order
END IF;
```

Alternatively, enumeration values can be referenced as plain strings when the context is unambiguous:

```sql
$Order = CREATE Sales.Order (
  Status = 'Draft'
);
```

## Expression Contexts

Expressions appear in several places within microflow activities:

### In Conditions (IF, WHILE, WHERE)

```sql
IF $Order/TotalAmount > 0 AND $Order/Status != 'Cancelled' THEN ...
WHILE $Counter < $MaxRetries BEGIN ... END WHILE;
RETRIEVE $Active FROM Sales.Order WHERE Status = 'Active';
```

### In Attribute Assignments (CREATE, CHANGE)

```sql
$Order = CREATE Sales.Order (
  OrderDate = [%CurrentDateTime%],
  Description = 'Order for ' + $Customer/Name,
  TotalAmount = $Subtotal + $Tax
);
```

### In SET Assignments

```sql
SET $Counter = $Counter + 1;
SET $IsEligible = $Customer/Age >= 18 AND $Customer/IsActive;
SET $FullName = $Customer/FirstName + ' ' + $Customer/LastName;
```

### In LOG Messages

```sql
LOG INFO 'Processed ' + toString($Count) + ' orders';
LOG ERROR NODE 'Validation' 'Invalid amount: ' + toString($Order/TotalAmount);
```

### In VALIDATION FEEDBACK Messages

```sql
VALIDATION FEEDBACK $Customer/Email MESSAGE 'Please provide a valid email address';
```
