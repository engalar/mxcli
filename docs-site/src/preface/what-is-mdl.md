# What is MDL?

**MDL (Mendix Definition Language)** is a text-based, SQL-like language for describing and manipulating Mendix application models. It provides a human-readable representation of Mendix project elements -- entities, microflows, pages, security rules, and more -- at the same abstraction level as the visual models in Studio Pro.

## Why a Text-Based Language?

Mendix projects are stored in a binary format (`.mpr` files containing BSON documents in SQLite). While Studio Pro provides a visual interface, there is no standard text format for Mendix models. This creates problems for:

- **AI assistants** that need to read and generate model elements
- **Code review** of changes to Mendix projects
- **Scripting and automation** of repetitive modeling tasks
- **Documentation** of project structure and design patterns

MDL addresses these problems by providing a syntax that is both human-readable and machine-parseable.

## Characteristics of MDL

- **SQL-like syntax** -- uses familiar keywords like `CREATE`, `ALTER`, `DROP`, `SHOW`, `DESCRIBE`, `GRANT`.
- **Declarative** -- you describe what the model should look like, not how to construct it step by step.
- **Complete** -- covers domain models, microflows, pages, security, navigation, workflows, and business events.
- **Validated** -- mxcli can check MDL scripts for syntax errors and reference validity before applying them.

## Token Efficiency

When working with AI coding assistants, context window size is a critical constraint. MDL's compact syntax uses significantly fewer tokens than equivalent JSON model representations:

| Representation | Tokens for a 10-entity module |
|----------------|-------------------------------|
| JSON (raw model) | ~15,000--25,000 tokens |
| MDL | ~2,000--4,000 tokens |
| **Reduction** | **5--10x fewer tokens** |

Fewer tokens means lower API costs, more application context fits in a single prompt, and AI assistants produce more accurate output because there is less noise in their input.

## Example: JSON vs MDL

A simple entity definition illustrates the difference.

**JSON representation (~180 tokens):**

```json
{
  "entity": {
    "name": "Customer",
    "documentation": "Customer master data",
    "attributes": [
      {"name": "Name", "type": {"type": "String", "length": 200}},
      {"name": "Email", "type": {"type": "String", "length": 200}}
    ]
  }
}
```

**MDL representation (~35 tokens):**

```sql
/** Customer master data */
CREATE PERSISTENT ENTITY Sales.Customer (
    Name: String(200),
    Email: String(200)
);
```

The MDL version conveys the same information in roughly one-fifth the space, and its structure is immediately recognizable to anyone familiar with SQL DDL.

## What MDL Covers

MDL provides statements for the following areas of a Mendix project:

| Area | Example Statements |
|------|--------------------|
| Domain model | `CREATE ENTITY`, `ALTER ENTITY`, `CREATE ASSOCIATION` |
| Microflows | `CREATE MICROFLOW`, `CREATE NANOFLOW` |
| Pages | `CREATE PAGE`, `CREATE SNIPPET`, `ALTER PAGE` |
| Security | `CREATE MODULE ROLE`, `GRANT`, `REVOKE` |
| Navigation | `ALTER NAVIGATION` |
| Workflows | `CREATE WORKFLOW` |
| Business events | `CREATE BUSINESS EVENT SERVICE` |
| Project queries | `SHOW MODULES`, `DESCRIBE ENTITY`, `SEARCH` |
| Catalog | `REFRESH CATALOG`, `SELECT FROM CATALOG` |

See Part II (The MDL Language) for a complete guide, or Part VI (MDL Statement Reference) for detailed syntax of each statement.
