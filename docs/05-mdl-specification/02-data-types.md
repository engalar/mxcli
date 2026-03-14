# MDL Data Types

This document describes the data type system in MDL and how types map to different backends.

## Table of Contents

1. [Primitive Types](#primitive-types)
2. [Complex Types](#complex-types)
3. [Type Syntax](#type-syntax)
4. [Default Values](#default-values)
5. [Type Mapping Table](#type-mapping-table)

---

## Primitive Types

### String

Variable-length text data.

**Syntax:**
```sql
String              -- Default length (200)
String(n)           -- Specific length (1-unlimited)
```

**Parameters:**
- `n` - Maximum length in characters. Use `unlimited` for unlimited length.

**Examples:**
```sql
Name: String(200)
Description: String(unlimited)
Code: String(10)
```

**Default value format:**
```sql
DEFAULT 'text value'
DEFAULT ''
```

---

### Integer

32-bit signed integer.

**Syntax:**
```sql
Integer
```

**Range:** -2,147,483,648 to 2,147,483,647

**Examples:**
```sql
Quantity: Integer
Age: Integer DEFAULT 0
Priority: Integer NOT NULL DEFAULT 1
```

---

### Long

64-bit signed integer.

**Syntax:**
```sql
Long
```

**Range:** -9,223,372,036,854,775,808 to 9,223,372,036,854,775,807

**Examples:**
```sql
FileSize: Long
TotalCount: Long DEFAULT 0
```

---

### Decimal

High-precision decimal number.

**Syntax:**
```sql
Decimal
```

**Precision:** Up to 20 digits total, with configurable decimal places.

**Examples:**
```sql
Price: Decimal
Amount: Decimal DEFAULT 0
TaxRate: Decimal DEFAULT 0.21
```

---

### Boolean

True/false value.

**Syntax:**
```sql
Boolean DEFAULT true|false
```

> **Required:** Boolean attributes must have a DEFAULT value. This is enforced by Mendix Studio Pro.

**Examples:**
```sql
IsActive: Boolean DEFAULT true
Enabled: Boolean DEFAULT true
Deleted: Boolean DEFAULT false
```

---

### DateTime

Date and time with timezone awareness.

**Syntax:**
```sql
DateTime
```

**Storage:** UTC timestamp with optional localization.

**Examples:**
```sql
CreatedAt: DateTime
ModifiedAt: DateTime
ScheduledFor: DateTime
```

**Note:** DateTime values include both date and time components. For date-only values, still use DateTime but only populate the date portion.

---

### Date

Date only (no time component).

**Syntax:**
```sql
Date
```

**Note:** Internally stored as DateTime in Mendix, but UI only shows date.

**Examples:**
```sql
BirthDate: Date
ExpiryDate: Date
```

---

### AutoNumber

Auto-incrementing integer, typically used for IDs.

**Syntax:**
```sql
AutoNumber
```

**Examples:**
```sql
OrderId: AutoNumber NOT NULL UNIQUE DEFAULT 1
CustomerId: AutoNumber
```

**Notes:**
- AutoNumber attributes automatically get the next value on object creation
- The DEFAULT value specifies the starting number
- Typically combined with NOT NULL and UNIQUE constraints

---

### Binary

Binary data (file contents, images, etc.).

**Syntax:**
```sql
Binary
```

**Examples:**
```sql
ProfileImage: Binary
Document: Binary
Thumbnail: Binary
```

**Notes:**
- Binary attributes can store files of any type
- File metadata (name, size, mime type) is stored separately by Mendix
- Maximum size is configurable per attribute

---

## Complex Types

### Enumeration

Reference to an enumeration type.

**Syntax:**
```sql
Enumeration(<qualified-name>)
```

**Parameters:**
- `<qualified-name>` - The Module.EnumerationName of the enumeration

**Examples:**
```sql
Status: Enumeration(Sales.OrderStatus)
Priority: Enumeration(Core.Priority) DEFAULT Core.Priority.Normal
Type: Enumeration(MyModule.ItemType) NOT NULL
```

**Default value format:**
```sql
DEFAULT Module.EnumName.ValueName
-- or legacy string literal form:
DEFAULT 'ValueName'
```

The default value is the **name** of the enumeration value (not the caption). The fully qualified form `Module.EnumName.ValueName` is preferred as it is explicit and unambiguous.

---

### HashedString

Securely hashed string (for passwords).

**Syntax:**
```sql
HashedString
```

**Examples:**
```sql
Password: HashedString
```

**Notes:**
- Values are one-way hashed and cannot be retrieved
- Comparison is done by hashing the input and comparing hashes
- Used primarily for password storage

---

## Type Syntax

### Full Attribute Definition

```sql
[/** documentation */]
<name>: <type> [constraints] [DEFAULT <value>]
```

### Constraints

| Constraint | Syntax | Description |
|------------|--------|-------------|
| Not Null | `NOT NULL` | Value is required |
| Not Null with Error | `NOT NULL ERROR 'message'` | Required with custom error |
| Unique | `UNIQUE` | Value must be unique |
| Unique with Error | `UNIQUE ERROR 'message'` | Unique with custom error |

### Constraint Order

Constraints must appear in this order:
1. `NOT NULL [ERROR '...']`
2. `UNIQUE [ERROR '...']`
3. `DEFAULT <value>`

**Examples:**
```sql
-- All constraints
Email: String(200) NOT NULL ERROR 'Email is required' UNIQUE ERROR 'Email already exists' DEFAULT ''

-- Some constraints
Name: String(200) NOT NULL
Code: String(10) UNIQUE
Count: Integer DEFAULT 0

-- No constraints
Description: String(unlimited)
```

---

## Default Values

### Syntax by Type

| Type | Default Syntax | Examples |
|------|----------------|----------|
| String | `DEFAULT 'value'` | `DEFAULT ''`, `DEFAULT 'Unknown'` |
| Integer | `DEFAULT n` | `DEFAULT 0`, `DEFAULT -1` |
| Long | `DEFAULT n` | `DEFAULT 0` |
| Decimal | `DEFAULT n.n` | `DEFAULT 0`, `DEFAULT 0.00`, `DEFAULT 99.99` |
| Boolean | `DEFAULT TRUE/FALSE` | `DEFAULT TRUE`, `DEFAULT FALSE` |
| AutoNumber | `DEFAULT n` | `DEFAULT 1` (starting value) |
| Enumeration | `DEFAULT Module.Enum.Value` | `DEFAULT Shop.Status.Active`, `DEFAULT 'Pending'` |

### No Default

Omit the DEFAULT clause for no default value:
```sql
OptionalField: String(200)
```

---

## Type Mapping Table

### MDL to Backend Type Mapping

| MDL Type | BSON $Type | Go Type (SDK) | Model API Type |
|----------|------------|---------------|----------------|
| String | `DomainModels$StringAttributeType` | `*StringAttributeType` | TBD |
| String(n) | `DomainModels$StringAttributeType` + Length | `*StringAttributeType{Length: n}` | TBD |
| Integer | `DomainModels$IntegerAttributeType` | `*IntegerAttributeType` | TBD |
| Long | `DomainModels$LongAttributeType` | `*LongAttributeType` | TBD |
| Decimal | `DomainModels$DecimalAttributeType` | `*DecimalAttributeType` | TBD |
| Boolean | `DomainModels$BooleanAttributeType` | `*BooleanAttributeType` | TBD |
| DateTime | `DomainModels$DateTimeAttributeType` | `*DateTimeAttributeType` | TBD |
| Date | `DomainModels$DateTimeAttributeType` | `*DateTimeAttributeType` | TBD |
| AutoNumber | `DomainModels$AutoNumberAttributeType` | `*AutoNumberAttributeType` | TBD |
| Binary | `DomainModels$BinaryAttributeType` | `*BinaryAttributeType` | TBD |
| Enumeration | `DomainModels$EnumerationAttributeType` | `*EnumerationAttributeType` | TBD |
| HashedString | `DomainModels$HashedStringAttributeType` | `*HashedStringAttributeType` | TBD |

### Default Value Mapping

| MDL Default | BSON Structure | Go Structure |
|-------------|----------------|--------------|
| `DEFAULT 'text'` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "text"}` | `Value: &AttributeValue{DefaultValue: "text"}` |
| `DEFAULT 123` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "123"}` | `Value: &AttributeValue{DefaultValue: "123"}` |
| `DEFAULT TRUE` | `Value: {$Type: "DomainModels$StoredValue", DefaultValue: "true"}` | `Value: &AttributeValue{DefaultValue: "true"}` |
| (calculated) | `Value: {$Type: "DomainModels$CalculatedValue", Microflow: <id>}` | `Value: &AttributeValue{Type: "CalculatedValue", MicroflowID: id}` |

---

## Type Compatibility

### Implicit Conversions

MDL does not perform implicit type conversions. Types must match exactly.

### Attribute Type Changes

When modifying an entity with `CREATE OR MODIFY`, attribute types cannot be changed if:
- The attribute contains data
- The new type is incompatible

Compatible type changes:
- `String(100)` to `String(200)` - Increasing length
- `Integer` to `Long` - Widening numeric type

Incompatible type changes:
- `String` to `Integer`
- `Boolean` to `String`
- Any type to `AutoNumber` (if not empty)

---

## Examples

### Complete Entity with All Types

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
/
```
