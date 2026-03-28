# Data Import Pipeline

Connect to an external database, explore its schema, import data into Mendix, and generate database connectors -- all from the mxcli REPL.

## Connect and Explore

```sql
-- Connect to the legacy database
SQL CONNECT postgres 'host=legacy-db port=5432 dbname=crm user=readonly' AS legacy;

-- Discover the schema
SQL legacy SHOW TABLES;
SQL legacy DESCRIBE customers;
SQL legacy DESCRIBE orders;

-- Preview the data
SQL legacy SELECT * FROM customers LIMIT 10;
```

## Import Data

The `IMPORT FROM` statement reads from the external database and inserts into the Mendix application's PostgreSQL database with proper Mendix ID generation:

```sql
-- Import customers
IMPORT FROM legacy
  QUERY 'SELECT name, email, phone, active FROM customers'
  INTO CRM.Customer
  MAP (name AS Name, email AS Email, phone AS Phone, active AS IsActive)
  BATCH 500;

-- Import orders with association linking
-- The LINK clause looks up Customer by Email and creates the association
IMPORT FROM legacy
  QUERY 'SELECT order_number, order_date, total, customer_email FROM orders'
  INTO Sales.Order
  MAP (order_number AS OrderNumber, order_date AS OrderDate, total AS TotalAmount)
  LINK (customer_email TO Sales.Order_Customer ON Email)
  BATCH 500;
```

## Generate Database Connectors

For ongoing integration (querying the external database from microflows at runtime), generate Database Connector entities and queries:

```sql
-- Auto-generate non-persistent entities and query microflows
SQL legacy GENERATE CONNECTOR INTO Integration
  TABLES (customers, orders, products);
```

This generates:

- Constants for JDBC connection string, username, and password
- Non-persistent entities with mapped attributes (SQL types to Mendix types)
- A `DATABASE CONNECTION` definition with query microflows

The output is valid MDL that can be reviewed before execution. Add the `EXEC` flag to execute immediately:

```sql
SQL legacy GENERATE CONNECTOR INTO Integration
  TABLES (customers, orders)
  EXEC;
```

## Execute Queries from Microflows

Once a database connection is defined, microflows can execute queries at runtime:

```sql
-- Non-persistent entity for query results
CREATE NON-PERSISTENT ENTITY Integration.Customer (
  /** Customer name from external system */
  Name: String(200),
  /** Email address */
  Email: String(200),
  /** Account balance */
  Balance: Decimal
);
/

-- Database connection with parameterized query
CREATE DATABASE CONNECTION Integration.LegacyDatabase
TYPE 'PostgreSQL'
CONNECTION STRING @Integration.LegacyDatabase_DBSource
USERNAME @Integration.LegacyDatabase_DBUsername
PASSWORD @Integration.LegacyDatabase_DBPassword
BEGIN
  QUERY SearchCustomers
    SQL 'SELECT name, email, balance FROM customers WHERE name ILIKE {search}'
    PARAMETER search: String DEFAULT '%'
    RETURNS Integration.Customer
    MAP (name AS Name, email AS Email, balance AS Balance);
END;
/

-- Microflow that executes the query
CREATE MICROFLOW Integration.SearchLegacyCustomers ($SearchTerm: String)
RETURNS List of Integration.Customer AS $Results
BEGIN
  $Results = EXECUTE DATABASE QUERY Integration.LegacyDatabase.SearchCustomers
    (search = '%' + $SearchTerm + '%');
  RETURN $Results;
END;
/
```
