# SHOW / DESCRIBE IMAGE COLLECTION

## Synopsis

    SHOW IMAGE COLLECTION;
    SHOW IMAGE COLLECTION IN module;
    DESCRIBE IMAGE COLLECTION module.name;

## Description

`SHOW IMAGE COLLECTION` lists all image collections in the project, optionally filtered by module. `DESCRIBE IMAGE COLLECTION` shows the full definition of a specific collection, including its images as a re-executable `CREATE` statement.

In the TUI, images are rendered inline when the terminal supports it (Kitty, iTerm2, Sixel).

## Parameters

**IN module**
: Filters the listing to a single module.

**module.name**
: The qualified name of the collection to describe (e.g., `MyModule.AppIcons`).

## Examples

### List all image collections

```sql
SHOW IMAGE COLLECTION;
```

### Filter by module

```sql
SHOW IMAGE COLLECTION IN MyModule;
```

### View full definition

```sql
DESCRIBE IMAGE COLLECTION MyModule.AppIcons;
```

The output includes the complete `CREATE` statement with all `IMAGE ... FROM FILE` entries, which can be copied and re-executed.

## See Also

[CREATE IMAGE COLLECTION](create-image-collection.md), [DROP IMAGE COLLECTION](drop-image-collection.md)
