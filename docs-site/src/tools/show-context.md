# SHOW CONTEXT

The `SHOW CONTEXT` command assembles the surrounding context of an element within its module and project hierarchy. This is particularly useful for AI-assisted development, where an LLM needs to understand the neighborhood of an element to generate accurate code.

## Prerequisites

Requires a full catalog refresh:

```sql
REFRESH CATALOG FULL;
```

## Syntax

```sql
SHOW CONTEXT OF <qualified-name>
```

## Examples

```sql
-- Show context of a microflow
SHOW CONTEXT OF Sales.ACT_ProcessOrder;

-- Show context of an entity
SHOW CONTEXT OF Sales.Customer;

-- Show context of a page
SHOW CONTEXT OF Sales.CustomerOverview;
```

## CLI Usage

```bash
# Basic context retrieval
mxcli context -p app.mpr Module.MyMicroflow

# With depth control
mxcli context -p app.mpr Module.MyMicroflow --depth 3
```

The `--depth` flag controls how many levels of related elements are included in the context output. A higher depth gives a broader picture but produces more output.

## What Context Includes

The context command gathers information about:

- The element's own definition (MDL source)
- Elements that call it (callers)
- Elements it calls (callees)
- Related entities and associations
- Pages that display or edit the same data
- The module structure surrounding the element

## Use Case: AI-Assisted Development

When an AI assistant needs to modify a microflow, it uses `SHOW CONTEXT` to understand:

1. What parameters the microflow expects
2. What entities and associations are involved
3. What other microflows it interacts with
4. What pages display the same data

This information allows the AI to generate code that fits naturally into the existing project structure.

```bash
# Gather context for an AI prompt
mxcli context -p app.mpr Sales.ACT_ProcessOrder --depth 2
```
