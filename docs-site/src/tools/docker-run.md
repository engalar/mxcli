# mxcli docker run

The `mxcli docker run` command runs a Mendix application in a Docker container, providing a local runtime environment for testing and development.

## Usage

```bash
mxcli docker run -p app.mpr
```

## What It Does

1. **Builds the application** if no build artifact exists
2. **Starts a Docker container** with the Mendix runtime
3. **Configures the runtime** with the project's settings (database, ports, etc.)
4. **Exposes the application** on the configured HTTP port

## Prerequisites

- Docker must be installed and running
- The project must be buildable (no errors in `mxcli docker check`)
- A PostgreSQL database must be available (Docker can provide one)

## Runtime Configuration

The runtime uses configuration from the project's settings. You can view and modify these with:

```sql
SHOW SETTINGS;
DESCRIBE SETTINGS;
ALTER SETTINGS CONFIGURATION 'default' DatabaseType = 'POSTGRESQL';
ALTER SETTINGS CONFIGURATION 'default' HttpPortNumber = '8080';
```

## Checking Project Health

Validate the project before running:

```bash
# Check for errors
mxcli docker check -p app.mpr
```

## Use with OQL

Once the application is running, you can query it with OQL:

```bash
mxcli oql -p app.mpr "SELECT * FROM Sales.Customer"
```

See [OQL Queries](oql.md) for details.
