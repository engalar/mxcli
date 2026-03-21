# Skill: Write OQL Queries for Mendix VIEW Entities

## Purpose
Generate correct OQL (Object Query Language) queries for Mendix VIEW entities. This skill helps you create VIEW entities with proper OQL syntax that will execute successfully in Mendix runtime.

## When to Use This Skill
- User asks to create a VIEW entity
- User requests help with OQL queries
- User wants to create analytics, reports, or aggregated data views
- User needs to join entities or create calculated fields
- You encounter OQL syntax errors when creating VIEW entities

## Critical OQL Syntax Rules

### 0. VIEW Entity Best Practices (CRITICAL)

**RULE 1: All SELECT columns MUST have explicit AS aliases**

Every column in the SELECT clause must have an alias that matches the entity attribute name:

```sql
-- ❌ WRONG - Missing aliases
CREATE VIEW ENTITY Finance.CashFlowProjection (
  ProjectionDate: DateTime,
  ProjectedIncome: Decimal,
  ProjectedExpense: Decimal
) AS (
  SELECT
    fl.ForecastDate,              -- Missing AS alias
    fl.ProjectedIncome,           -- Missing AS alias
    fl.ProjectedExpense           -- Missing AS alias
  FROM Finance.ForecastLine AS fl
);

-- ✅ CORRECT - All columns have explicit aliases
CREATE VIEW ENTITY Finance.CashFlowProjection (
  ProjectionDate: DateTime,
  ProjectedIncome: Decimal,
  ProjectedExpense: Decimal
) AS (
  SELECT
    fl.ForecastDate AS ProjectionDate,
    fl.ProjectedIncome AS ProjectedIncome,
    fl.ProjectedExpense AS ProjectedExpense
  FROM Finance.ForecastLine AS fl
);
```

**RULE 2: NEVER use ORDER BY or LIMIT in VIEW entity OQL**

The UI component or microflow using the view will handle sorting and pagination:

```sql
-- ❌ WRONG - Hardcoded sorting and limits
CREATE VIEW ENTITY Finance.TopCustomers (...) AS (
  SELECT c.Name AS CustomerName, sum(o.Amount) AS TotalSpent
  FROM Finance.Customer AS c
  INNER JOIN Finance.Order_Customer/Finance.Order AS o
  GROUP BY c.Name
  ORDER BY TotalSpent DESC        -- Remove this
  LIMIT 100                       -- Remove this
);

-- ✅ CORRECT - No ORDER BY or LIMIT
CREATE VIEW ENTITY Finance.TopCustomers (...) AS (
  SELECT c.Name AS CustomerName, sum(o.Amount) AS TotalSpent
  FROM Finance.Customer AS c
  INNER JOIN Finance.Order_Customer/Finance.Order AS o
  GROUP BY c.Name
  -- Let the UI component handle sorting and limits
);
```

**Why these rules matter:**
- **Explicit aliases**: Required for proper OQL-to-entity attribute mapping in Mendix
- **No ORDER BY/LIMIT**: Provides flexibility - different pages/microflows can sort and paginate the same view differently

### 1. Aggregate Functions (MUST BE LOWERCASE)
```sql
-- ❌ WRONG - Uppercase will fail
SUM(o.Amount)
AVG(o.Amount)
MAX(o.OrderDate)
MIN(o.Amount)

-- ✅ CORRECT - Lowercase
sum(o.Amount)
avg(o.Amount)
max(o.OrderDate)
min(o.Amount)
```

### 2. COUNT Function
```sql
-- ❌ WRONG - count(*) not supported in Mendix OQL
count(*)

-- ✅ CORRECT - Count by ID or entity
count(t.ID)           -- Count by ID attribute
count(t)              -- Count entity instances
```

### Aggregate Function Return Types

| Function | Input Type | Returns | MDL Declaration |
|----------|-----------|---------|-----------------|
| `count(expr)` | any | Integer | `Attr: Integer` |
| `sum(expr)` | Integer | Integer | `Attr: Integer` |
| `sum(expr)` | Decimal | Decimal | `Attr: Decimal` |
| `avg(expr)` | any numeric | Decimal | `Attr: Decimal` |
| `max(expr)` / `min(expr)` | Integer | Integer | `Attr: Integer` |
| `max(expr)` / `min(expr)` | Decimal | Decimal | `Attr: Decimal` |
| `max(expr)` / `min(expr)` | DateTime | DateTime | `Attr: DateTime` |
| `datepart(part, expr)` | DateTime | Integer | `Attr: Integer` |
| `length(expr)` | String | Integer | `Attr: Integer` |

**Key rule**: `count()` and `avg()` have fixed return types. `sum()`, `min()`, `max()` preserve the input type.

### 3. DATEPART Function (Comma Syntax)
```sql
-- ✅ CORRECT - Use comma syntax
datepart(YEAR, t.TransactionDate)
datepart(MONTH, t.TransactionDate)
datepart(QUARTER, t.TransactionDate)
datepart(WEEK, t.TransactionDate)
datepart(DAY, t.TransactionDate)

-- ❌ WRONG - FROM syntax not supported
DATEPART(YEAR FROM t.TransactionDate)
```

### 4. Enumeration Comparisons (String Literals)
```sql
-- ❌ WRONG - Qualified enum names
t.TransactionType = Finance.TransactionType.INCOME
t.Status != Finance.TransactionStatus.VOID

-- ✅ CORRECT - Use string literals
t.TransactionType = 'INCOME'
t.Status != 'VOID'
```

### 5. Division Operator (Colon, not Slash)
```sql
-- ❌ WRONG - Using / causes parsing errors
SELECT amount / quantity AS price
SELECT (total - discount) * 100.0 / total AS percentage

-- ✅ CORRECT - Use : for division
SELECT amount : quantity AS price
SELECT (total - discount) * 100.0 : total AS percentage
```

### 6. ORDER BY with Aliases
```sql
-- ❌ WRONG - Using expressions in ORDER BY
ORDER BY datepart(YEAR, t.TransactionDate) DESC

-- ✅ CORRECT - Use column aliases
SELECT
  datepart(YEAR, t.TransactionDate) AS OrderYear
FROM Finance.Transaction AS t
ORDER BY OrderYear DESC
```

### 7. Operators (Use != not <>)
```sql
-- ❌ WRONG - <> causes errors in Mendix
WHERE t.Status <> 'VOIDED'

-- ✅ CORRECT - Use !=
WHERE t.Status != 'VOIDED'
```

**Note:** Both `!=` and `<>` are valid in standard SQL, but Mendix OQL only accepts `!=`.

### 8. IN Expression Syntax
```sql
-- ✅ IN with value list
WHERE t.Status IN ('ACTIVE', 'PENDING', 'REVIEW')

-- ✅ IN with subquery
WHERE t.CustomerId IN (
  SELECT c.CustomerId FROM Shop.Customer AS c WHERE c.IsVIP = true
)

-- ✅ Enumeration values use identifiers, not captions
WHERE t.Priority IN ('HIGH', 'CRITICAL')  -- Not 'High', 'Critical'
```

### 9. Subqueries (Scalar and Correlated)
```sql
-- ✅ Scalar subquery in SELECT (returns single value)
SELECT
  p.Name AS ProductName,
  p.Price - (SELECT avg(p2.Price) FROM Shop.Product AS p2) AS DiffFromAvg
FROM Shop.Product AS p

-- ✅ Scalar subquery in WHERE
WHERE p.Price > (SELECT avg(p2.Price) FROM Shop.Product AS p2)

-- ✅ Correlated subquery (references outer query by attribute)
SELECT
  o.OrderNumber AS OrderNumber,
  (SELECT count(o2.OrderId) FROM Shop.Order AS o2 WHERE o2.CustomerId = o.CustomerId) AS CustomerOrderCount
FROM Shop.Order AS o

-- ✅ Correlated subquery via association (compare to .ID)
SELECT
  p.Name AS ProductName,
  (SELECT pr.PriceInEuro FROM Shop.Price AS pr
   WHERE pr/Shop.Price_Product = p.ID
   ORDER BY pr.StartDate DESC LIMIT 1) AS LatestPrice
FROM Shop.Product AS p

-- ❌ WRONG - bare alias without .ID
WHERE pr/Shop.Price_Product = p    -- Doesn't resolve

-- ✅ CORRECT - compare to entity .ID
WHERE pr/Shop.Price_Product = p.ID
```

### 10. Association Path Syntax
```sql
-- Association paths in OQL use '/' not '.'
-- ✅ CORRECT - slash prefix for association traversal
WHERE l/Library.Loan_Member = m.ID
JOIN l/Library.Loan_Book/Library.Book AS b

-- ❌ WRONG - dot instead of slash
WHERE l.Library.Loan_Member = m.ID     -- Error: does not resolve
```

### 11. JOIN Syntax (Association Traversal and ON Clause)

Mendix OQL supports both association traversal and SQL-style JOIN ON:

```sql
-- ✅ Association traversal (uses Mendix association path)
FROM Shop.Order AS o
INNER JOIN o/Shop.Order_Customer/Shop.Customer AS c

-- ✅ JOIN ON clause (SQL-style, for any condition)
FROM Shop.Order AS o
INNER JOIN Shop.Customer AS c ON o.CustomerId = c.CustomerId

-- ✅ LEFT OUTER JOIN with ON clause
FROM Shop.Product AS p
LEFT OUTER JOIN Shop.CompetitorProduct AS cp ON p.ProductCode = cp.ProductCode
```

**When to use each approach:**
- **Association traversal** (`alias/Module.Association/Entity`): When joining on a Mendix-defined association
- **JOIN ON** (`JOIN Entity ON condition`): When joining on arbitrary conditions or non-association fields

## Common OQL Patterns

### Pattern 1: Date-based Aggregation
```sql
CREATE VIEW ENTITY Finance.MonthlySummary (
  Year: Integer,
  Month: Integer,
  TotalAmount: Decimal,
  TransactionCount: Integer
) AS (
  SELECT
    datepart(YEAR, t.Date) AS Year,
    datepart(MONTH, t.Date) AS Month,
    sum(t.Amount) AS TotalAmount,
    count(t.ID) AS TransactionCount
  FROM Finance.Transaction AS t
  WHERE t.Status != 'VOIDED'
  GROUP BY datepart(YEAR, t.Date), datepart(MONTH, t.Date)
);
```

### Pattern 2: Conditional Aggregation
```sql
CREATE VIEW ENTITY Finance.CategorySummary (
  Category: String(200),
  Income: Decimal,
  Expense: Decimal,
  Net: Decimal
) AS (
  SELECT
    c.Name AS Category,
    sum(CASE WHEN t.Type = 'INCOME' THEN t.Amount ELSE 0 END) AS Income,
    sum(CASE WHEN t.Type = 'EXPENSE' THEN t.Amount ELSE 0 END) AS Expense,
    sum(CASE WHEN t.Type = 'INCOME' THEN t.Amount
             WHEN t.Type = 'EXPENSE' THEN -t.Amount ELSE 0 END) AS Net
  FROM Finance.Transaction AS t
  INNER JOIN Finance.Transaction_Category/Finance.Category AS c
  GROUP BY c.Name
);
```

### Pattern 3: Association Navigation
```sql
CREATE VIEW ENTITY Shop.OrderDetails (
  OrderId: Long,
  CustomerName: String(400),
  TotalItems: Integer,
  TotalPrice: Decimal
) AS (
  SELECT
    o.OrderId AS OrderId,
    c.FirstName + ' ' + c.LastName AS CustomerName,
    count(ol.OrderLineId) AS TotalItems,
    o.TotalPrice AS TotalPrice
  FROM Shop.CustomerOrder AS o
  INNER JOIN Shop.Order_Customer/Shop.Customer AS c
  LEFT JOIN Shop.OrderLine_Order/Shop.OrderLine AS ol
  GROUP BY o.OrderId, o.TotalPrice, c.FirstName, c.LastName
);
```

### Pattern 4: Calculations with Division
```sql
CREATE VIEW ENTITY Finance.BudgetVariance (
  Category: String(200),
  Budget: Decimal,
  Actual: Decimal,
  Variance: Decimal,
  VariancePercent: Decimal
) AS (
  SELECT
    c.Name AS Category,
    bl.PlannedAmount AS Budget,
    bl.ActualAmount AS Actual,
    bl.ActualAmount - bl.PlannedAmount AS Variance,
    (bl.ActualAmount - bl.PlannedAmount) * 100.0 : bl.PlannedAmount AS VariancePercent
  FROM Finance.BudgetLine AS bl
  INNER JOIN Finance.BudgetLine_Category/Finance.Category AS c
  WHERE bl.PlannedAmount > 0
);
```

### Pattern 5: IN Expression with Value List
```sql
CREATE VIEW ENTITY Shop.HighPriorityTasks (
  TaskId: Integer,
  TaskTitle: String(200),
  Priority: String(50)
) AS (
  SELECT
    t.TaskId AS TaskId,
    t.TaskTitle AS TaskTitle,
    t.TaskPriority AS Priority
  FROM Shop.Task AS t
  WHERE t.TaskPriority IN ('HIGH', 'CRITICAL')
);
```

### Pattern 6: IN Expression with Subquery
```sql
CREATE VIEW ENTITY Shop.CustomersWithOrders (
  CustomerId: Integer,
  CustomerName: String(200)
) AS (
  SELECT
    c.CustomerId AS CustomerId,
    c.Name AS CustomerName
  FROM Shop.Customer AS c
  WHERE c.CustomerId IN (
    SELECT DISTINCT o.CustomerId
    FROM Shop.Order AS o
    WHERE o.Status = 'COMPLETED'
  )
);
```

### Pattern 7: Scalar Subquery in SELECT
```sql
CREATE VIEW ENTITY Shop.ProductsAboveAverage (
  ProductId: Integer,
  Name: String(200),
  Price: Decimal,
  PriceDifferenceFromAvg: Decimal
) AS (
  SELECT
    p.ProductId AS ProductId,
    p.Name AS Name,
    p.Price AS Price,
    p.Price - (SELECT avg(p2.Price) FROM Shop.Product AS p2) AS PriceDifferenceFromAvg
  FROM Shop.Product AS p
  WHERE p.Price > (SELECT avg(p3.Price) FROM Shop.Product AS p3)
);
```

### Pattern 8: Correlated Subquery
```sql
CREATE VIEW ENTITY Shop.OrdersWithCustomerStats (
  OrderId: Integer,
  OrderNumber: String(50),
  CustomerTotalOrders: Integer,
  CustomerTotalSpend: Decimal
) AS (
  SELECT
    o.OrderId AS OrderId,
    o.OrderNumber AS OrderNumber,
    (SELECT count(o2.OrderId) FROM Shop.Order AS o2 WHERE o2.CustomerId = o.CustomerId) AS CustomerTotalOrders,
    (SELECT sum(o3.TotalAmount) FROM Shop.Order AS o3 WHERE o3.CustomerId = o.CustomerId) AS CustomerTotalSpend
  FROM Shop.Order AS o
);
```

### Pattern 9: Correlated Subquery via Association
```sql
-- Get the latest price for each product using association traversal
CREATE VIEW ENTITY Shop.ProductCurrentPrice (
  ProductId: String(50),
  Name: String(200),
  PriceInEuro: Decimal,
  IsActive: Boolean
) AS (
  SELECT
    p.ProductId AS ProductId,
    p.Name AS Name,
    (SELECT pr.PriceInEuro
     FROM Shop.Price AS pr
     WHERE pr.StartDate <= '[%BeginOfTomorrow%]'
     AND pr/Shop.Price_Product = p.ID
     ORDER BY pr.StartDate DESC
     LIMIT 1) AS PriceInEuro,
    p.IsActive AS IsActive
  FROM Shop.Product AS p
  WHERE p.IsActive
);
```

**Key points:**
- Use `pr/Shop.Price_Product = p.ID` (association path with `.ID`)
- Never use bare alias: `pr/Shop.Price_Product = p` will fail
- ORDER BY and LIMIT are valid inside correlated subqueries (just not at the view level)

### Pattern 10: JOIN with ON Clause (Non-Association)
```sql
-- When joining on arbitrary conditions (not Mendix associations)
CREATE VIEW ENTITY Shop.ProductComparison (
  ProductId: Integer,
  ProductName: String(200),
  CompetitorPrice: Decimal
) AS (
  SELECT
    p.ProductId AS ProductId,
    p.Name AS ProductName,
    cp.Price AS CompetitorPrice
  FROM Shop.Product AS p
  LEFT JOIN Shop.CompetitorProduct AS cp ON p.ProductCode = cp.ProductCode
  WHERE cp.CompetitorName = 'ACME'
);
```

## Step-by-Step Process

### Step 1: Define VIEW Entity Schema

**Always include @Position annotation:**

```sql
/**
 * View entity description
 *
 * @since 1.0.0
 */
@Position(300, 500)
CREATE VIEW ENTITY Module.ViewName (
  Attribute1: Type,
  Attribute2: Type,
  -- ... more attributes
) AS (
  -- OQL query goes here
);
```

### Step 2: Write SELECT Clause
- Use **lowercase** aggregate functions: `sum()`, `avg()`, `count()`
- Use `count(entity.ID)` not `count(*)`
- Create meaningful aliases for all columns
- Use `:` for division operations

### Step 3: Write FROM Clause
- Use table aliases (AS t, AS c, etc.)
- Navigate associations: `Entity_Association/TargetEntity`

### Step 4: Add JOINs if Needed
```sql
-- Association join syntax
INNER JOIN Shop.Order_Customer/Shop.Customer AS c
LEFT JOIN Shop.Product_Category/Shop.Category AS cat
```

### Step 5: Add WHERE Clause
- Use string literals for enum comparisons: `'VALUE'`
- Use standard comparison operators: `=`, `!=`, `>`, `<`, `>=`, `<=`

### Step 6: Add GROUP BY if Using Aggregates
- Include all non-aggregated columns
- Use same expressions as SELECT (e.g., `datepart()`)

### Step 7: Verify Aliases
- Ensure ALL SELECT columns have explicit AS aliases
- Aliases must match entity attribute names exactly

### Step 8: Validate Before Executing
```bash
mxcli check view.mdl -p app.mpr --references
```
This catches type mismatches (e.g., declaring `Long` for a `count()` column that returns `Integer`), missing module references, and OQL syntax errors — before they become MxBuild errors like CE6770 ("View Entity is out of sync with the OQL Query").

### Step 9: Final Check
- Remove any ORDER BY, LIMIT, or OFFSET clauses
- These should be handled by the UI component or microflow

## Common Mistakes to Avoid

### ❌ Mistake 1: Uppercase Aggregates
```sql
-- WRONG
SELECT SUM(amount) FROM ...

-- CORRECT
SELECT sum(amount) FROM ...
```

### ❌ Mistake 2: Using count(*)
```sql
-- WRONG
SELECT count(*) FROM Finance.Transaction

-- CORRECT
SELECT count(t.ID) FROM Finance.Transaction AS t
```

### ❌ Mistake 3: Qualified Enum Names
```sql
-- WRONG
WHERE t.Status = Finance.Status.ACTIVE

-- CORRECT
WHERE t.Status = 'ACTIVE'
```

### ❌ Mistake 4: Slash for Division
```sql
-- WRONG
SELECT total / count AS average

-- CORRECT
SELECT total : count AS average
```

### ❌ Mistake 5: Missing Column Aliases
```sql
-- WRONG
SELECT
  fl.ForecastDate,
  fl.ProjectedIncome
FROM Finance.ForecastLine AS fl

-- CORRECT
SELECT
  fl.ForecastDate AS ProjectionDate,
  fl.ProjectedIncome AS ProjectedIncome
FROM Finance.ForecastLine AS fl
```

### ❌ Mistake 6: Dot Instead of Slash for Association Paths
```sql
-- WRONG - dot notation for association
WHERE l.Library.Loan_Member = m.ID

-- CORRECT - slash notation
WHERE l/Library.Loan_Member = m.ID
```

### ❌ Mistake 7: Bare Alias in Association Comparison
```sql
-- WRONG - comparing association to bare entity alias
WHERE pr/Shop.Price_Product = p

-- CORRECT - compare to entity .ID
WHERE pr/Shop.Price_Product = p.ID
```

### ❌ Mistake 8: Using ORDER BY or LIMIT in VIEW
```sql
-- WRONG - Hardcoded in view
CREATE VIEW ENTITY Finance.TopItems (...) AS (
  SELECT ...
  ORDER BY Amount DESC
  LIMIT 100
);

-- CORRECT - Let UI handle it
CREATE VIEW ENTITY Finance.TopItems (...) AS (
  SELECT ...
  -- No ORDER BY or LIMIT
);
```

## Complete Example

### User Request
"Create a VIEW entity showing monthly revenue with order statistics"

### Response
```sql
/**
 * Monthly revenue summary with order statistics
 *
 * Time-series view of revenue and order metrics
 * aggregated by month and year.
 *
 * @since 1.0.0
 * @see Shop.CustomerOrder
 */
@Position(1400, 450)
CREATE VIEW ENTITY Shop.MonthlyRevenue (
  Year: Integer,
  Month: Integer,
  TotalOrders: Integer,
  TotalRevenue: Decimal,
  AverageOrderValue: Decimal
) AS (
  SELECT
    datepart(YEAR, o.OrderDate) AS Year,
    datepart(MONTH, o.OrderDate) AS Month,
    count(o.OrderId) AS TotalOrders,
    sum(o.TotalPrice) AS TotalRevenue,
    avg(o.TotalPrice) AS AverageOrderValue
  FROM Shop.CustomerOrder AS o
  GROUP BY datepart(YEAR, o.OrderDate), datepart(MONTH, o.OrderDate)
);
```

### Why This Works
1. ✅ All columns have explicit AS aliases
2. ✅ Lowercase aggregates: `sum()`, `avg()`, `count()`
3. ✅ Proper COUNT: `count(o.OrderId)` not `count(*)`
4. ✅ Comma syntax for DATEPART: `datepart(YEAR, o.OrderDate)`
5. ✅ GROUP BY matches SELECT non-aggregated expressions
6. ✅ No ORDER BY or LIMIT (UI will handle sorting)

## Testing OQL Queries

Use `mxcli oql` to test queries against a running Mendix runtime (read-only preview mode):

```bash
# Basic query (reads .docker/.env for connection settings)
mxcli oql -p app.mpr "SELECT Name, Email FROM MyModule.Customer"

# JSON output for piping to jq
mxcli oql -p app.mpr --json "SELECT count(c.ID) FROM MyModule.Order AS c" | jq '.[0]'

# Explicit connection (no project file needed)
mxcli oql --host localhost --port 8090 --token 'AdminPassword1!' "SELECT 1"

# Test a VIEW entity query before embedding it
mxcli oql -p app.mpr "SELECT datepart(YEAR, o.OrderDate) AS Year, sum(o.Total) AS Revenue FROM Sales.Order AS o GROUP BY datepart(YEAR, o.OrderDate)"
```

The app must be running first: `mxcli docker run -p app.mpr --wait`

> **Troubleshooting**: If you get "Action not found: preview_execute_oql", the Docker stack
> needs the `-Dmendix.live-preview=enabled` JVM flag. Re-initialize with:
> `mxcli docker init -p app.mpr --force`, then restart with `mxcli docker run -p app.mpr --wait`.

### Workflow: OQL → VIEW ENTITY

1. **Write and test interactively**: `mxcli oql -p app.mpr "SELECT ..."`
2. **Iterate** until the query returns expected results
3. **Embed** in a VIEW ENTITY with matching column aliases and attribute types
4. **Validate before executing**: `mxcli check view.mdl -p app.mpr --references` to catch type mismatches (e.g., `Long` vs `Integer` for `count()`)
5. **Apply and rebuild**: `mxcli exec view.mdl -p app.mpr && mxcli docker run -p app.mpr --fresh --wait`

## Integration with MDL Linter

The MDL linter checks for common OQL issues:

**Rule: `consistency/oql-syntax`**
- Validates VIEW entity OQL queries
- Checks for ORDER BY without LIMIT/OFFSET (CE0174)
- Checks for missing/empty SELECT or FROM clauses

**How to Fix Linter Errors:**
```bash
# Lint a file
mendix> lint file 'path/to/file.mdl';

# Common error: ORDER BY without LIMIT
# Error: VIEW entity X: ORDER BY requires LIMIT or OFFSET. Studio Pro error: CE0174
# Fix: Add LIMIT clause
```

## References

- [Mendix OQL Documentation](https://docs.mendix.com/refguide/oql/)
- [OQL Expressions](https://docs.mendix.com/refguide/oql-expressions/)
- [OQL Functions](https://docs.mendix.com/refguide/oql-expression-syntax/)
- Internal: `packages/mendix-repl/docs/syntax-proposals/OQL_SYNTAX_GUIDE.md`
- Internal: `packages/mendix-repl/examples/VIEW_ENTITY_VALIDATION.md`

## Summary Checklist

When writing OQL queries for VIEW entities, always verify:

- [ ] **CRITICAL**: Entity has @Position annotation (e.g., @Position(300, 500))
- [ ] **CRITICAL**: All SELECT columns have explicit AS aliases matching entity attributes
- [ ] **CRITICAL**: No ORDER BY, LIMIT, or OFFSET clauses (let UI handle sorting)
- [ ] Aggregate functions are lowercase (`sum`, `avg`, `count`, `max`, `min`)
- [ ] Using `count(entity.ID)` not `count(*)`
- [ ] DATEPART uses comma syntax: `datepart(YEAR, field)`
- [ ] Enum comparisons use enumeration **identifiers**, not captions: `'HIGH'` not `'High'`
- [ ] IN expressions use correct syntax: `IN ('VAL1', 'VAL2')` or `IN (SELECT ...)`
- [ ] Division uses colon: `amount : quantity`
- [ ] Inequality uses `!=` not `<>`
- [ ] All non-aggregated columns are in GROUP BY
- [ ] Association paths use `/` not `.`: `alias/Module.Assoc` not `alias.Module.Assoc`
- [ ] Association comparisons use `.ID`: `pr/Shop.Price_Product = p.ID` not `= p`
- [ ] Association navigation uses correct syntax: `Entity_Assoc/Target AS alias`
- [ ] JOIN ON clauses use comparison operators: `ON a.Field = b.Field`
- [ ] Subqueries are enclosed in parentheses and return appropriate values
- [ ] **Validate before executing**: Run `mxcli check script.mdl -p app.mpr --references` to catch type mismatches

Following these rules ensures your OQL queries will parse and execute correctly in Mendix runtime.
