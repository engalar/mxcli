# SHOW IMAGE COLLECTION

## Synopsis

    SHOW IMAGE COLLECTION;
    SHOW IMAGE COLLECTION IN module;

## Description

Lists all image collections in the project. Use `IN module` to filter to a specific module. For full details including embedded images, use `DESCRIBE IMAGE COLLECTION`.

## Parameters

**IN module**
: Optional. Restricts the listing to collections in the specified module.

## Examples

### List all collections

```sql
SHOW IMAGE COLLECTION;
```

### Filter by module

```sql
SHOW IMAGE COLLECTION IN MyModule;
```

### Describe a specific collection

```sql
DESCRIBE IMAGE COLLECTION MyModule.AppIcons;
```

## See Also

[DESCRIBE IMAGE COLLECTION](../image-collection/show-describe-image-collection.md), [CREATE IMAGE COLLECTION](../image-collection/create-image-collection.md), [SHOW MODULES](show-modules.md)
