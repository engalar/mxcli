# SQL CONNECT

The `SQL CONNECT` command establishes a named connection to an external database.

## Syntax

```sql
SQL CONNECT <driver> '<dsn>' AS <alias>
```

**Parameters:**

| Parameter | Description |
|-----------|-------------|
| `<driver>` | Database driver name |
| `<dsn>` | Data Source Name (connection string) |
| `<alias>` | Short name to reference this connection |

## Supported Drivers

| Database | Driver Names |
|----------|-------------|
| PostgreSQL | `postgres`, `pg`, `postgresql` |
| Oracle | `oracle`, `ora` |
| SQL Server | `sqlserver`, `mssql` |

## Connection Examples

### PostgreSQL

```sql
SQL CONNECT postgres 'postgres://user:password@localhost:5432/mydb' AS source;

-- With SSL
SQL CONNECT postgres 'postgres://user:password@host:5432/mydb?sslmode=require' AS prod;
```

### Oracle

```sql
SQL CONNECT oracle 'oracle://user:password@host:1521/service_name' AS oradb;
```

### SQL Server

```sql
SQL CONNECT sqlserver 'sqlserver://user:password@host:1433?database=mydb' AS mssql;
```

## Managing Connections

### List Active Connections

```sql
SQL CONNECTIONS;
```

This shows alias and driver name only. The DSN is never displayed for security reasons.

### Disconnect

```sql
SQL DISCONNECT source;
```

## Multiple Connections

You can maintain multiple simultaneous connections with different aliases:

```sql
SQL CONNECT postgres 'postgres://...' AS source_db;
SQL CONNECT oracle 'oracle://...' AS legacy_db;
SQL CONNECT sqlserver 'sqlserver://...' AS reporting_db;

-- Query each by alias
SQL source_db SELECT count(*) FROM users;
SQL legacy_db SELECT count(*) FROM employees;
SQL reporting_db SELECT count(*) FROM reports;

-- List all connections
SQL CONNECTIONS;

-- Disconnect individually
SQL DISCONNECT legacy_db;
```

## CLI Subcommand

For one-off queries without establishing a named connection:

```bash
mxcli sql --driver postgres --dsn 'postgres://user:pass@localhost:5432/mydb' "SELECT 1"
```

## Security Notes

- The DSN (which contains credentials) is never shown in session output or logs
- Only the alias and driver type are displayed when listing connections
- See [Credential Management](credentials.md) for environment variable and YAML config alternatives
