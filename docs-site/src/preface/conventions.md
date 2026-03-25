# Document Conventions

This page describes the formatting and notation conventions used throughout this documentation.

## Code Examples

MDL examples use `sql` code fencing because MDL's syntax is SQL-like and this provides appropriate syntax highlighting:

```sql
CREATE PERSISTENT ENTITY MyModule.Customer (
    Name: String(200) NOT NULL
);
```

Go code examples use `go` code fencing:

```go
reader, err := modelsdk.Open("/path/to/app.mpr")
defer reader.Close()
```

Shell commands use `bash` code fencing:

```bash
mxcli -p app.mpr -c "SHOW MODULES"
```

## Syntax Notation

When describing the syntax of MDL statements, the following conventions apply:

| Notation | Meaning |
|----------|---------|
| `UPPERCASE` | Keywords -- type them exactly as shown |
| `lowercase` | User-provided values (names, expressions, types) |
| `[brackets]` | Optional clause -- may be omitted |
| `...` | Repetition -- the preceding element may appear multiple times |
| `a | b` | Choice -- use one of the alternatives |
| `( )` | Grouping -- used to clarify precedence in syntax descriptions |

For example, the notation:

```
CREATE [PERSISTENT] ENTITY module.name (
    attribute_name: type [NOT NULL] [, ...]
);
```

means: `CREATE` and `ENTITY` are required keywords; `PERSISTENT` is optional; `module.name` is a user-provided qualified name; each attribute has a name and type, with an optional `NOT NULL` constraint; and additional attributes may follow, separated by commas.

## Cross-References

References to MDL statements link to their detailed pages in Part VI (MDL Statement Reference) using the format "See CREATE ENTITY" or "See GRANT". References to conceptual explanations link to the relevant section in Part II (The MDL Language).

## Terminology

Mendix-specific terms such as "entity", "microflow", "nanoflow", "module", and "domain model" follow standard Mendix terminology. See the [Glossary](../appendixes/glossary.md) for definitions.
