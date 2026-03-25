# ALTER ENTITY

`ALTER ENTITY` modifies an existing entity's attributes, indexes, or documentation without recreating it. This is useful for incremental changes to entities that already contain data.

## ADD Attributes

Add one or more new attributes:

```sql
ALTER ENTITY Sales.Customer
  ADD (Phone: String(50), Notes: String(unlimited));
```

New attributes support the same [constraints](./constraints.md) as in `CREATE ENTITY`:

```sql
ALTER ENTITY Sales.Customer
  ADD (
    LoyaltyPoints: Integer DEFAULT 0,
    MemberSince: DateTime NOT NULL
  );
```

## DROP Attributes

Remove one or more attributes:

```sql
ALTER ENTITY Sales.Customer
  DROP (Notes);
```

Multiple attributes can be dropped at once:

```sql
ALTER ENTITY Sales.Customer
  DROP (Notes, TempField, OldStatus);
```

## MODIFY Attributes

Change the type or constraints of existing attributes:

```sql
ALTER ENTITY Sales.Customer
  MODIFY (Name: String(400) NOT NULL);
```

## RENAME Attributes

Rename an attribute:

```sql
ALTER ENTITY Sales.Customer
  RENAME Phone TO PhoneNumber;
```

## ADD INDEX

Add an index to the entity:

```sql
ALTER ENTITY Sales.Customer
  ADD INDEX (Email);
```

Composite indexes:

```sql
ALTER ENTITY Sales.Customer
  ADD INDEX (Name, CreatedAt DESC);
```

## DROP INDEX

Remove an index:

```sql
ALTER ENTITY Sales.Customer
  DROP INDEX (Email);
```

## SET DOCUMENTATION

Update the entity's documentation text:

```sql
ALTER ENTITY Sales.Customer
  SET DOCUMENTATION 'Customer master data for the Sales module';
```

## Syntax Summary

```sql
ALTER ENTITY <Module>.<Entity>
  ADD (<attribute-definition> [, ...])

ALTER ENTITY <Module>.<Entity>
  DROP (<attribute-name> [, ...])

ALTER ENTITY <Module>.<Entity>
  MODIFY (<attribute-definition> [, ...])

ALTER ENTITY <Module>.<Entity>
  RENAME <old-name> TO <new-name>

ALTER ENTITY <Module>.<Entity>
  ADD INDEX (<column-list>)

ALTER ENTITY <Module>.<Entity>
  DROP INDEX (<column-list>)

ALTER ENTITY <Module>.<Entity>
  SET DOCUMENTATION '<text>'
```

## See Also

- [Entities](./entities.md) -- CREATE ENTITY syntax
- [Attributes](./attributes.md) -- attribute definition format
- [Indexes](./indexes.md) -- index creation and management
- [Constraints](./constraints.md) -- NOT NULL, UNIQUE, DEFAULT
