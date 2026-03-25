# OQL Queries

The `mxcli oql` command executes OQL (Object Query Language) queries against a running Mendix runtime via the M2EE admin API. OQL is Mendix's query language for retrieving data from the runtime, similar to SQL but operating on Mendix entities rather than database tables.

## Usage

```bash
mxcli oql -p app.mpr "SELECT * FROM Sales.Customer"
```

## Prerequisites

- A Mendix application must be running (via `mxcli docker run` or another deployment)
- The M2EE admin API must be accessible
- The project file is needed to resolve entity and attribute names

## Query Syntax

OQL uses Mendix entity names and attribute names, not database table names:

```bash
# Select all customers
mxcli oql -p app.mpr "SELECT * FROM Sales.Customer"

# Select specific attributes
mxcli oql -p app.mpr "SELECT Name, Email FROM Sales.Customer"

# Filter with WHERE
mxcli oql -p app.mpr "SELECT * FROM Sales.Order WHERE Status = 'Open'"

# Join entities via associations
mxcli oql -p app.mpr "SELECT o.OrderNumber, c.Name FROM Sales.Order o JOIN Sales.Customer c ON o.Sales.Order_Customer = c"

# Aggregation
mxcli oql -p app.mpr "SELECT COUNT(*) FROM Sales.Customer"
```

## Difference from Catalog Queries

| Feature | OQL (`mxcli oql`) | Catalog (`SELECT FROM CATALOG.*`) |
|---------|-------------------|----------------------------------|
| Data source | Running Mendix runtime | Project metadata (MPR file) |
| Queries | Instance data (rows) | Model structure (entities, microflows) |
| Requires | Running application | Only the MPR file |
| Language | OQL (Mendix-specific) | SQL (SQLite) |

- Use **OQL** to query actual data in a running application
- Use **catalog queries** to analyze the project structure and metadata

## Use Cases

### Verifying Imported Data

After running `IMPORT FROM`, verify the data was imported correctly:

```bash
mxcli oql -p app.mpr "SELECT COUNT(*) FROM HR.Employee"
mxcli oql -p app.mpr "SELECT Name, Email FROM HR.Employee LIMIT 10"
```

### Debugging Business Logic

Check data state while debugging microflows:

```bash
mxcli oql -p app.mpr "SELECT * FROM Sales.Order WHERE Status = 'Draft'"
```

### Data Exploration

Explore data patterns in a running application:

```bash
mxcli oql -p app.mpr "SELECT Status, COUNT(*) FROM Sales.Order GROUP BY Status"
```
