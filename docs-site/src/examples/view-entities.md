# View Entities

View entities are read-only entities backed by an OQL query. They appear in the domain model but have no database table -- their data is computed from other entities via aggregation and joins.

## Sales Summary by Category

```sql
-- Source entities
CREATE PERSISTENT ENTITY Reports.ProductCategory (
  /** Category display name */
  CategoryName: String(200) NOT NULL
);
/

CREATE PERSISTENT ENTITY Reports.SaleTransaction (
  /** Transaction amount */
  Amount: Decimal NOT NULL,
  /** Date of the sale */
  SaleDate: DateTime NOT NULL
);
/

CREATE ASSOCIATION Reports.SaleTransaction_ProductCategory
  FROM Reports.ProductCategory TO Reports.SaleTransaction
  TYPE Reference OWNER Default;
/

-- View entity: aggregates sales by category
CREATE VIEW ENTITY Reports.SalesTotalByCategory (
  CategoryName: String(200),
  TotalAmount: Decimal,
  TransactionCount: Integer
) AS (
  SELECT
    c.CategoryName AS CategoryName,
    sum(s.Amount) AS TotalAmount,
    count(s.ID) AS TransactionCount
  FROM Reports.SaleTransaction AS s
  INNER JOIN s/Reports.SaleTransaction_ProductCategory/Reports.ProductCategory AS c
  GROUP BY c.CategoryName
);
/
```

## Querying a View Entity in a Microflow

View entities can be retrieved like any other entity, including with `WHERE` filters:

```sql
CREATE MICROFLOW Reports.GetSalesTotalForCategory (
  $Category: Reports.ProductCategory
)
RETURNS Decimal AS $TotalAmount
BEGIN
  DECLARE $TotalAmount Decimal = 0;

  RETRIEVE $Summary FROM Reports.SalesTotalByCategory
    WHERE CategoryName = $Category/CategoryName
    LIMIT 1;

  IF $Summary != empty THEN
    SET $TotalAmount = $Summary/TotalAmount;
  END IF;

  RETURN $TotalAmount;
END;
/
```

## Displaying in a Page

View entities work with data grids and list views like persistent entities:

```sql
CREATE PAGE Reports.SalesByCategory_Overview (
  Title: 'Sales by Category',
  Layout: Atlas_Core.Atlas_Default
) {
  DATAGRID2 ON Reports.SalesTotalByCategory (
    COLUMN CategoryName { Caption: 'Category' }
    COLUMN TotalAmount { Caption: 'Total Sales' }
    COLUMN TransactionCount { Caption: 'Transactions' }
  )
};
/
```
