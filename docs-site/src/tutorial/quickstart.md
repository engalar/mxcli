# 5-Minute Quickstart

Get from zero to modifying a Mendix project in 5 minutes. No prior MDL knowledge needed.

## 1. Install mxcli

```bash
# macOS / Linux
curl -fsSL https://mxcli.org/install.sh | bash

# Or with Go
go install github.com/mendixlabs/mxcli/cmd/mxcli@latest
```

Verify: `mxcli --version` should print the version number.

## 2. Open your project

```bash
mxcli -p /path/to/your-app.mpr
```

You'll see:
```
Connected to: your-app.mpr (Mendix 11.6.3)
MDL REPL - Mendix Definition Language
mdl>
```

> **Don't have a Mendix project?** Create one in Studio Pro first, or use `mx create-project --app-name test-app`.

## 3. Explore what's there

```sql
-- List all modules
SHOW MODULES;

-- List entities in a module
SHOW ENTITIES IN MyFirstModule;

-- See the full structure at a glance
SHOW STRUCTURE;
```

## 4. Create something

```sql
-- Create a new entity
CREATE PERSISTENT ENTITY MyFirstModule.Customer (
  Name: String(200) NOT NULL,
  Email: String(200),
  IsActive: Boolean DEFAULT true
);
```

Expected output:
```
Created entity: MyFirstModule.Customer
```

## 5. Verify it works

```sql
-- See what you created
DESCRIBE ENTITY MyFirstModule.Customer;
```

Output:
```
@Position(100, 100)
CREATE OR REPLACE PERSISTENT ENTITY MyFirstModule.Customer (
  Name: String(200) NOT NULL,
  Email: String(200),
  IsActive: Boolean DEFAULT true
);
```

Open the project in Studio Pro -- your entity is there.

## 6. Create a microflow

```sql
CREATE MICROFLOW MyFirstModule.ACT_DeactivateCustomer (
  $Customer: MyFirstModule.Customer
)
BEGIN
  CHANGE $Customer (IsActive = false);
  COMMIT $Customer;
END;
```

## 7. Validate

```bash
# Check your project for errors (from a separate terminal)
mxcli check -p /path/to/your-app.mpr
```

No errors? You're done. Open in Studio Pro and everything is there.

## What's next?

| I want to... | Read... |
|---|---|
| Explore my project deeper | [SHOW Commands](show-commands.md) |
| Create pages with widgets | [Creating a Page](create-page.md) |
| Use AI to generate code | [Claude Code Integration](claude-code.md) |
| Set up for a team | [Skills and CLAUDE.md](skills.md) |
| See everything mxcli can do | [Capabilities Overview](../reference/capabilities.md) |
| Customize AI generation | [Customizing AI Generation](../guides/customizing-ai.md) |

## Common issues

**"mx command not available"** -- Install mxbuild for validation:
```bash
mxcli setup mxbuild -p your-app.mpr
```

**"CGO not available"** -- mxcli uses pure Go SQLite. No C compiler needed. If you see CGO errors, ensure you're using the official binary or `go install`.

**Project won't open in Studio Pro after changes** -- Close Studio Pro before running mxcli write commands, then reopen. See [F4 sync support](../appendixes/version-compatibility.md) for details.
