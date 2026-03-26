# DROP IMAGE COLLECTION

## Synopsis

    DROP IMAGE COLLECTION module.name;

## Description

Removes an image collection and all its embedded images from the module.

## Parameters

**module.name**
: The qualified name of the collection to drop (e.g., `MyModule.AppIcons`).

## Examples

```sql
DROP IMAGE COLLECTION MyModule.StatusIcons;
```

### Check references before dropping

```sql
SHOW REFERENCES TO MyModule.AppIcons;
DROP IMAGE COLLECTION MyModule.AppIcons;
```

## See Also

[CREATE IMAGE COLLECTION](create-image-collection.md), [SHOW / DESCRIBE IMAGE COLLECTION](show-describe-image-collection.md)
