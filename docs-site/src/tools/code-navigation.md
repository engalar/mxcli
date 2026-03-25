# Code Navigation

mxcli provides a suite of cross-reference navigation commands that let you explore relationships between elements in a Mendix project. These commands answer questions like "what calls this microflow?", "what would break if I change this entity?", and "what does this element depend on?"

## Prerequisites

Cross-reference navigation commands require a **full catalog refresh** before use:

```sql
REFRESH CATALOG FULL
```

This populates the reference data needed for callers, callees, references, impact, and context queries. A basic `REFRESH CATALOG` only builds metadata tables and is not sufficient for cross-reference navigation.

## Available Commands

| Command | Purpose |
|---------|---------|
| `SHOW CALLERS OF` | Find what calls a given element |
| `SHOW CALLEES OF` | Find what a given element calls |
| `SHOW REFERENCES OF` | Find all references to and from an element |
| `SHOW IMPACT OF` | Analyze the impact of changing an element |
| `SHOW CONTEXT OF` | Show the surrounding context of an element |
| `SEARCH` | Full-text search across all strings and source |

## CLI Subcommands

These commands are also available as CLI subcommands for scripting and piping:

```bash
mxcli callers -p app.mpr Module.MyMicroflow
mxcli callees -p app.mpr Module.MyMicroflow
mxcli refs -p app.mpr Module.Customer
mxcli impact -p app.mpr Module.Customer
mxcli context -p app.mpr Module.MyMicroflow --depth 3
mxcli search -p app.mpr "validation"
```

## Typical Workflow

1. **Refresh the catalog** to ensure cross-reference data is current
2. **Explore references** to understand how elements are connected
3. **Analyze impact** before making changes
4. **Use context** to gather surrounding information for AI-assisted development

```sql
-- Step 1: Build the catalog
REFRESH CATALOG FULL;

-- Step 2: Understand what uses a microflow
SHOW CALLERS OF Sales.ACT_ProcessOrder;

-- Step 3: Check impact before renaming an entity
SHOW IMPACT OF Sales.Customer;

-- Step 4: Gather context for AI consumption
SHOW CONTEXT OF Sales.SubmitOrder;
```

## How AI Assistants Use This

When you ask an AI assistant to modify an element, it uses these commands to:

1. **Discover dependencies** with `SHOW CALLERS` and `SHOW REFERENCES`
2. **Assess risk** with `SHOW IMPACT` before making changes
3. **Gather context** with `SHOW CONTEXT` for informed code generation
4. **Find related elements** with `SEARCH` to understand patterns
