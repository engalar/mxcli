# Generalization

Generalization (inheritance) allows an entity to extend another entity, inheriting all of its attributes and associations. The child entity can add its own attributes on top.

## Syntax

```sql
CREATE PERSISTENT ENTITY <Module>.<Name>
  EXTENDS <ParentEntity>
(
  <additional-attributes>
);
```

Both `EXTENDS` (preferred) and `GENERALIZATION` (legacy) keywords are supported.

## Examples

### Extending System.User

The most common use of generalization is creating an application-specific user entity:

```sql
/** Employee extends User with additional fields */
CREATE PERSISTENT ENTITY HR.Employee EXTENDS System.User (
  EmployeeNumber: String(20) NOT NULL UNIQUE,
  Department: String(100),
  HireDate: Date
);
```

The `HR.Employee` entity inherits all attributes from `System.User` (Name, Password, etc.) and adds `EmployeeNumber`, `Department`, and `HireDate`.

### Extending System.Image

For entities that store images:

```sql
/** Product photo entity */
CREATE PERSISTENT ENTITY Catalog.ProductPhoto EXTENDS System.Image (
  Caption: String(200),
  SortOrder: Integer DEFAULT 0
);
```

### Extending System.FileDocument

For entities that store file attachments:

```sql
/** File attachment entity */
CREATE PERSISTENT ENTITY Docs.Attachment EXTENDS System.FileDocument (
  Description: String(500)
);
```

## Common System Generalizations

| Parent Entity | Purpose |
|---------------|---------|
| `System.User` | User accounts with authentication |
| `System.FileDocument` | File storage (name, size, content) |
| `System.Image` | Image storage (extends FileDocument with dimensions) |

## Inheritance Rules

- A persistent entity can extend another persistent entity
- The child entity inherits all attributes and associations of the parent
- The child entity can add new attributes but cannot remove inherited ones
- Associations to the parent entity also apply to child objects
- Database queries on the parent entity include child objects

## See Also

- [Entities](./entities.md) -- entity types and CREATE ENTITY syntax
- [Associations](./associations.md) -- relationships between entities
- [Domain Model](./domain-model.md) -- domain model overview
