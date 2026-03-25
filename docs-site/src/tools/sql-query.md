# Querying External Databases

Once connected to an external database, you can explore its schema and run arbitrary SQL queries.

## Schema Exploration

### Show Tables

```sql
SQL <alias> SHOW TABLES;
```

Lists all user tables in the connected database.

### Show Views

```sql
SQL <alias> SHOW VIEWS;
```

Lists all user views.

### Show Functions

```sql
SQL <alias> SHOW FUNCTIONS;
```

Lists functions and stored procedures.

### Describe Table

```sql
SQL <alias> DESCRIBE <table>;
```

Shows column details including name, type, and nullability.

## Running Queries

Any SQL statement can be passed through to the connected database:

```sql
SQL <alias> <any-sql>;
```

### Examples

```sql
-- Connect
SQL CONNECT postgres 'postgres://user:pass@localhost:5432/mydb' AS source;

-- Explore schema
SQL source SHOW TABLES;
SQL source DESCRIBE employees;

-- Simple queries
SQL source SELECT * FROM employees WHERE active = true LIMIT 10;

-- Aggregation
SQL source SELECT department, COUNT(*) as cnt FROM employees GROUP BY department ORDER BY cnt DESC;

-- Joins
SQL source SELECT e.name, d.name AS department
  FROM employees e
  JOIN departments d ON e.dept_id = d.id
  WHERE e.active = true;

-- DDL (if you have permissions)
SQL source CREATE INDEX idx_employees_email ON employees(email);
```

## CLI Subcommand

For one-off queries from the command line:

```bash
# Direct query
mxcli sql --driver postgres --dsn 'postgres://user:pass@localhost:5432/mydb' "SELECT * FROM users LIMIT 5"
```

## Interactive Workflow

Within the REPL, use SQL commands interactively:

```sql
-- Connect
SQL CONNECT postgres 'postgres://user:pass@localhost:5432/mydb' AS source;

-- Explore
SQL source SHOW TABLES;
SQL source DESCRIBE users;

-- Query
SQL source SELECT id, name, email FROM users WHERE created_at > '2024-01-01' LIMIT 10;

-- Done
SQL DISCONNECT source;
```

## Output

Query results are displayed in a formatted table by default. The output format can be controlled with session variables:

```sql
SET output_format = 'json';
SQL source SELECT * FROM users LIMIT 5;
```
