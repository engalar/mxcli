# Data Type Mapping Table

Comprehensive mapping between MDL data types, Mendix internal types, and backend representations.

## Primitive Types

| MDL Type | Description | Range/Limits |
|----------|-------------|--------------|
| `String` | Variable-length text (default length 200) | 1 to `unlimited` characters |
| `String(n)` | Variable-length text with explicit length | 1 to `unlimited` characters |
| `Integer` | 32-bit signed integer | -2,147,483,648 to 2,147,483,647 |
| `Long` | 64-bit signed integer | -9,223,372,036,854,775,808 to 9,223,372,036,854,775,807 |
| `Decimal` | High-precision decimal number | Up to 20 digits total |
| `Boolean` | True/false value | `true` or `false` (must have DEFAULT) |
| `DateTime` | Date and time with timezone awareness | UTC timestamp |
| `Date` | Date only (no time component) | Internally stored as DateTime |
| `AutoNumber` | Auto-incrementing integer | Typically combined with NOT NULL UNIQUE |
| `Binary` | Binary data (files, images) | Configurable max size |
| `HashedString` | Securely hashed string (passwords) | One-way hash, cannot be retrieved |

## Complex Types

| MDL Type | Description | Example |
|----------|-------------|---------|
| `Enumeration(Module.EnumName)` | Reference to an enumeration type | `Status: Enumeration(Sales.OrderStatus)` |

## MDL to Backend Type Mapping

| MDL Type | BSON $Type | Go Type (SDK) |
|----------|------------|---------------|
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

## Default Value Mapping

| MDL Default | BSON Structure | Go Structure |
|-------------|----------------|--------------|
| `DEFAULT 'text'` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "text"}` | `Value: &AttributeValue{DefaultValue: "text"}` |
| `DEFAULT 123` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "123"}` | `Value: &AttributeValue{DefaultValue: "123"}` |
| `DEFAULT TRUE` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "true"}` | `Value: &AttributeValue{DefaultValue: "true"}` |
| (calculated) | `Value: {$Type: "DomainModels$CalculatedValue", Microflow: <id>}` | `Value: &AttributeValue{Type: "CalculatedValue", MicroflowID: id}` |

## Default Value Syntax by Type

| Type | Default Syntax | Examples |
|------|----------------|----------|
| String | `DEFAULT 'value'` | `DEFAULT ''`, `DEFAULT 'Unknown'` |
| Integer | `DEFAULT n` | `DEFAULT 0`, `DEFAULT -1` |
| Long | `DEFAULT n` | `DEFAULT 0` |
| Decimal | `DEFAULT n.n` | `DEFAULT 0`, `DEFAULT 0.00`, `DEFAULT 99.99` |
| Boolean | `DEFAULT TRUE/FALSE` | `DEFAULT TRUE`, `DEFAULT FALSE` |
| AutoNumber | `DEFAULT n` | `DEFAULT 1` (starting value) |
| Enumeration | `DEFAULT Module.Enum.Value` | `DEFAULT Shop.Status.Active`, `DEFAULT 'Pending'` |

## Constraints

| Constraint | Syntax | Description |
|------------|--------|-------------|
| Not Null | `NOT NULL` | Value is required |
| Not Null with Error | `NOT NULL ERROR 'message'` | Required with custom error |
| Unique | `UNIQUE` | Value must be unique |
| Unique with Error | `UNIQUE ERROR 'message'` | Unique with custom error |

Constraints must appear in this order:
1. `NOT NULL [ERROR '...']`
2. `UNIQUE [ERROR '...']`
3. `DEFAULT <value>`

## Type Compatibility

MDL does not perform implicit type conversions. Types must match exactly.

**Compatible type changes:**
- `String(100)` to `String(200)` -- Increasing length
- `Integer` to `Long` -- Widening numeric type

**Incompatible type changes:**
- `String` to `Integer`
- `Boolean` to `String`
- Any type to `AutoNumber` (if not empty)

## Complete Example

```sql
/** Example entity demonstrating all data types */
@Position(100, 100)
CREATE PERSISTENT ENTITY Demo.AllTypes (
  /** Auto-generated ID */
  Id: AutoNumber NOT NULL UNIQUE DEFAULT 1,

  /** Short text field */
  Code: String(10) NOT NULL UNIQUE,

  /** Standard text field */
  Name: String(200) NOT NULL,

  /** Long text field */
  Description: String(unlimited),

  /** Integer counter */
  Counter: Integer DEFAULT 0,

  /** Large number */
  BigNumber: Long,

  /** Money amount */
  Amount: Decimal DEFAULT 0.00,

  /** Flag field */
  IsActive: Boolean DEFAULT TRUE,

  /** Timestamp */
  CreatedAt: DateTime,

  /** Date only */
  BirthDate: Date,

  /** File attachment */
  Attachment: Binary,

  /** Status from enumeration */
  Status: Enumeration(Demo.Status) DEFAULT 'Active'
)
INDEX (Code)
INDEX (Name, CreatedAt DESC);
```
