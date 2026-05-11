# CREATE IMPORT MAPPING

## Synopsis

```sql
CREATE [ OR MODIFY ] IMPORT MAPPING module.Name
    [ WITH JSON STRUCTURE module.JsonStructure ]
    [ WITH XML SCHEMA module.XmlSchema ]
{
    { CREATE | FIND | FIND OR CREATE } module.Entity {
        Attribute = jsonField [ KEY ],
        ...
        [ { CREATE | FIND | FIND OR CREATE } Assoc/module.ChildEntity = nestedKey {
            ChildAttr = childField [ KEY ],
            ...
        } ]
    }
};
```

## Description

Creates an import mapping that reads JSON or XML data and creates or finds Mendix entity objects from it.

An import mapping is invoked in a microflow using `IMPORT FROM MAPPING module.Mapping($json)`. The result is the root entity object populated from the parsed data.

### Object Handling

Each entity block specifies one of three strategies:

| Keyword | Behaviour |
|---------|-----------|
| `CREATE` | Always create a new object |
| `FIND` | Find an existing object by the KEY attribute; error if not found |
| `FIND OR CREATE` | Find an existing object by the KEY attribute; create if not found (upsert) |

Marking an attribute with `KEY` designates it as the lookup key for FIND and FIND OR CREATE.

### Nested Objects and Arrays

Child entities are nested inside the parent's `{ }` block using the association path:

```
AssociationName/module.ChildEntity = jsonKey { ... }
```

The association must be defined such that the child entity owns the foreign key (FROM child TO parent). Arrays in the JSON sample automatically map to multiple child objects.

### OR MODIFY

If `OR MODIFY` is specified and the mapping already exists, it is updated in place. The document UUID is preserved, which protects runtime microflow references that call `IMPORT FROM MAPPING`.

## Parameters

`module.Name`
:   The qualified name of the import mapping.

`WITH JSON STRUCTURE module.JsonStructure`
:   Associates the mapping with the named JSON structure. The structure defines the JSON shape.

`WITH XML SCHEMA module.XmlSchema`
:   Associates the mapping with the named XML schema instead of a JSON structure.

`CREATE | FIND | FIND OR CREATE`
:   Object handling strategy for each entity block.

`Attribute = jsonField [ KEY ]`
:   Maps the JSON field `jsonField` to entity attribute `Attribute`. `KEY` marks this attribute for FIND lookups.

`AssociationName/module.ChildEntity = nestedKey`
:   Maps a nested JSON object or array element to a child entity via the named association.

## Examples

### Simple flat mapping

```sql
CREATE IMPORT MAPPING MyModule.IMM_Pet
    WITH JSON STRUCTURE MyModule.JSON_Pet
{
    CREATE MyModule.PetResponse {
        PetId = id,
        Name  = name,
        Status = status
    }
};
```

### Upsert by key attribute

```sql
CREATE IMPORT MAPPING MyModule.IMM_UpsertPet
    WITH JSON STRUCTURE MyModule.JSON_Pet
{
    FIND OR CREATE MyModule.PetResponse {
        PetId = id KEY,
        Name  = name,
        Status = status
    }
};
```

### Nested JSON with child entity

```sql
CREATE IMPORT MAPPING MyModule.IMM_Order
    WITH JSON STRUCTURE MyModule.JSON_Order
{
    CREATE MyModule.OrderResponse {
        OrderId = orderId,
        CREATE MyModule.CustomerInfo_OrderResponse/MyModule.CustomerInfo = customer {
            Name  = name,
            Email = email
        },
        CREATE MyModule.OrderItem_OrderResponse/MyModule.OrderItem = items {
            Sku      = sku,
            Quantity = quantity,
            Price    = price
        }
    }
};
```

### Idempotent update

```sql
CREATE OR MODIFY IMPORT MAPPING MyModule.IMM_Pet
    WITH JSON STRUCTURE MyModule.JSON_Pet
{
    CREATE MyModule.PetResponse {
        PetId  = id,
        Name   = name,
        Status = status
    }
};
```

## Notes

- The child entity must own the FK (`FROM child TO parent`). Using the wrong association direction causes validation errors at runtime.
- Arrays in the JSON sample map directly to child entity objects — there is no intermediate container entity (unlike export mappings).
- Import and export of the same JSON typically require different entity structures because of the FK direction difference.

## See Also

[CREATE JSON STRUCTURE](create-json-structure.md), [CREATE EXPORT MAPPING](create-export-mapping.md)
