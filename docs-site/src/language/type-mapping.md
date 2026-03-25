# Type Mapping

This page shows how MDL types correspond to Mendix internal types (BSON storage) and the Go SDK types used by modelsdk-go.

## MDL to Backend Type Mapping

| MDL Type | BSON `$Type` | Go SDK Type |
|----------|------------|-------------|
| `String` | `DomainModels$StringAttributeType` | `*StringAttributeType` |
| `String(n)` | `DomainModels$StringAttributeType` + Length | `*StringAttributeType{Length: n}` |
| `Integer` | `DomainModels$IntegerAttributeType` | `*IntegerAttributeType` |
| `Long` | `DomainModels$LongAttributeType` | `*LongAttributeType` |
| `Decimal` | `DomainModels$DecimalAttributeType` | `*DecimalAttributeType` |
| `Boolean` | `DomainModels$BooleanAttributeType` | `*BooleanAttributeType` |
| `DateTime` | `DomainModels$DateTimeAttributeType` | `*DateTimeAttributeType` |
| `Date` | `DomainModels$DateTimeAttributeType` | `*DateTimeAttributeType` |
| `AutoNumber` | `DomainModels$AutoNumberAttributeType` | `*AutoNumberAttributeType` |
| `Binary` | `DomainModels$BinaryAttributeType` | `*BinaryAttributeType` |
| `Enumeration` | `DomainModels$EnumerationAttributeType` | `*EnumerationAttributeType` |
| `HashedString` | `DomainModels$HashedStringAttributeType` | `*HashedStringAttributeType` |

Note that `Date` and `DateTime` both map to the same underlying BSON type. The distinction is handled at the UI layer.

## Default Value Mapping

Default values are stored in BSON as `StoredValue` structures:

| MDL Default | BSON Structure |
|-------------|----------------|
| `DEFAULT 'text'` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "text"}` |
| `DEFAULT 123` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "123"}` |
| `DEFAULT TRUE` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "true"}` |
| (calculated) | `Value: {$Type: "DomainModels$CalculatedValue", Microflow: <id>}` |

All default values are serialized as strings in the BSON storage, regardless of the attribute type.

## See Also

- [Primitive Types](./primitive-types.md) -- MDL type syntax and usage
- [Data Types](./data-types.md) -- type system overview
