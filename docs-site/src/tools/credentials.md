# Credential Management

mxcli provides secure credential management for external database connections. Credentials are isolated from session output, logs, and error messages.

## Security Principles

- **DSN isolation** -- Connection strings (which contain passwords) are never displayed in session output
- **Alias-only display** -- `SQL CONNECTIONS` only shows alias and driver type
- **Log sanitization** -- Credentials are stripped from log output and error messages

## Connection Methods

### Inline DSN

The simplest approach, suitable for development:

```sql
SQL CONNECT postgres 'postgres://user:password@localhost:5432/mydb' AS source;
```

### Environment Variables

Store credentials in environment variables for CI/CD and production:

```bash
# Set environment variable
export MYDB_DSN='postgres://user:password@host:5432/mydb'

# Use in mxcli (depends on your shell expanding the variable)
SQL CONNECT postgres '$MYDB_DSN' AS source;
```

### YAML Configuration

mxcli supports YAML configuration files for managing multiple database connections. The configuration file stores DSN information that can be referenced by name.

Configuration is resolved from:
1. Environment variables
2. YAML config files
3. Inline DSN strings

## CLI Subcommand

The `mxcli sql` subcommand accepts credentials via command-line flags:

```bash
mxcli sql --driver postgres --dsn 'postgres://user:pass@host:5432/db' "SELECT 1"
```

For CI/CD pipelines, use environment variables:

```bash
export DB_DSN='postgres://user:pass@host:5432/db'
mxcli sql --driver postgres --dsn "$DB_DSN" "SELECT * FROM users"
```

## Best Practices

1. **Never commit credentials** to version control
2. **Use environment variables** in CI/CD pipelines
3. **Use YAML config** for local development with multiple databases
4. **Rotate credentials** regularly
5. **Use read-only database users** when only querying (not importing)

## Mendix Application Database

For `IMPORT` commands that write to the Mendix application database, the connection is automatically established using the project's configuration settings (from `DESCRIBE SETTINGS`). The Mendix database DSN is built from the project's DatabaseType, DatabaseUrl, DatabaseName, DatabaseUserName, and DatabasePassword settings.
