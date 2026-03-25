# Qualified Names

MDL uses dot-separated qualified names to reference elements within a Mendix project. This naming convention ensures elements are unambiguous across modules.

## Simple Identifiers

Valid identifier characters:

- Letters: `A-Z`, `a-z`
- Digits: `0-9` (not as first character)
- Underscore: `_`

```sql
MyEntity
my_attribute
Attribute123
```

## Qualified Name Format

The general format is `Module.Element` or `Module.Entity.Attribute`:

```sql
MyModule.Customer           -- Entity in module
MyModule.OrderStatus        -- Enumeration in module
MyModule.Customer.Name      -- Attribute in entity
```

### Two-Part Names

Most elements use a two-part name: `Module.ElementName`.

```sql
Sales.Customer              -- entity
Sales.OrderStatus           -- enumeration
Sales.ACT_CreateOrder       -- microflow
Sales.Customer_Edit         -- page
Sales.Order_Customer        -- association
```

### Three-Part Names

Attribute references use three parts: `Module.Entity.Attribute`.

```sql
Sales.Customer.Name         -- attribute on entity
Sales.Order.TotalAmount     -- attribute on entity
```

## Quoting Rules

When a name segment is a reserved keyword, wrap it in double quotes or backticks:

```sql
"ComboBox"."CategoryTreeVE"
`Order`.`Status`
```

Mixed quoting is allowed -- you only need to quote the segments that conflict:

```sql
"ComboBox".CategoryTreeVE
Sales."Order"
```

Identifiers that contain only letters, digits, and underscores (and do not start with a digit) never need quoting.

## Usage in Statements

Qualified names appear throughout MDL:

```sql
-- Entity creation
CREATE PERSISTENT ENTITY Sales.Customer ( ... );

-- Association referencing two entities
CREATE ASSOCIATION Sales.Order_Customer
  FROM Sales.Customer
  TO Sales.Order
  TYPE Reference;

-- Enumeration reference in an attribute type
Status: Enumeration(Sales.OrderStatus) DEFAULT 'Active'

-- Microflow call
$Result = CALL MICROFLOW Sales.ACT_ProcessOrder ($Order = $Order);
```

## See Also

- [Lexical Structure](./lexical-structure.md) -- keywords and quoting
- [Entities](./entities.md) -- entity qualified names
- [Associations](./associations.md) -- association naming conventions
