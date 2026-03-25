# MDL Basics

MDL (Mendix Definition Language) is a SQL-like language for reading and modifying Mendix application projects. It provides a text-based alternative to the visual editors in Mendix Studio Pro.

## What MDL Looks Like

MDL uses familiar SQL-style syntax with Mendix-specific extensions. Here is a simple example that creates an entity with attributes:

```sql
CREATE PERSISTENT ENTITY Sales.Customer (
  CustomerId: AutoNumber NOT NULL UNIQUE DEFAULT 1,
  Name: String(200) NOT NULL,
  Email: String(200) UNIQUE,
  IsActive: Boolean DEFAULT TRUE
)
INDEX (Name);
```

## Statement Termination

Statements are terminated with a semicolon (`;`) or a forward slash (`/`) on its own line (Oracle-style, useful for multi-line statements):

```sql
-- Semicolon terminator
CREATE MODULE OrderManagement;

-- Forward-slash terminator (useful for long statements)
CREATE PERSISTENT ENTITY Sales.Order (
  OrderId: AutoNumber NOT NULL UNIQUE,
  OrderDate: DateTime NOT NULL
)
INDEX (OrderDate DESC);
/
```

Simple commands such as `HELP`, `EXIT`, `STATUS`, `SHOW`, and `DESCRIBE` do not require a terminator.

## Case Insensitivity

All MDL **keywords** are case-insensitive. The following are equivalent:

```sql
CREATE PERSISTENT ENTITY Sales.Customer ( ... );
create persistent entity Sales.Customer ( ... );
Create Persistent Entity Sales.Customer ( ... );
```

Identifiers (module names, entity names, attribute names) are case-sensitive and must match the Mendix model exactly.

## Statement Categories

MDL statements fall into several categories:

| Category | Examples |
|----------|----------|
| **Query** | `SHOW ENTITIES`, `DESCRIBE ENTITY`, `SEARCH` |
| **Domain Model** | `CREATE ENTITY`, `CREATE ASSOCIATION`, `ALTER ENTITY` |
| **Enumerations** | `CREATE ENUMERATION`, `ALTER ENUMERATION` |
| **Microflows** | `CREATE MICROFLOW`, `DROP MICROFLOW` |
| **Pages** | `CREATE PAGE`, `ALTER PAGE`, `CREATE SNIPPET` |
| **Security** | `GRANT`, `REVOKE`, `CREATE USER ROLE` |
| **Navigation** | `CREATE OR REPLACE NAVIGATION` |
| **Connection** | `CONNECT LOCAL`, `DISCONNECT`, `STATUS` |

## Further Reading

- [Lexical Structure](./lexical-structure.md) -- keywords, literals, and tokens
- [Qualified Names](./qualified-names.md) -- how elements are referenced
- [Comments](./comments.md) -- comment syntax
- [Script Files](./script-files.md) -- running MDL from files
