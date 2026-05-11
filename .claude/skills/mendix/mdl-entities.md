# MDL Entity Syntax Reference

Complete syntax reference for creating entities, attributes, and associations.

## Entity Types

| Type | Keyword | Stored in DB | Use Case |
|------|---------|--------------|----------|
| Persistent | `create persistent entity` | Yes | Business data |
| Non-Persistent | `create non-persistent entity` | No | Temporary/view data |
| View | `create view entity` | No (OQL query) | Aggregated/computed data |

## Persistent Entity

```mdl
/**
 * Customer entity for storing customer data
 */
create persistent entity Module.Customer (
  -- String attributes
  Name: string(100) not null,
  Email: string(200),
  Code: string(20) unique,

  -- Numeric attributes
  Age: integer,
  CreditLimit: decimal,

  -- Boolean
  IsActive: boolean default true,

  -- Date/Time
  BirthDate: date,
  -- Use autocreateddate (not datetime) to record when the object was created.
  -- 'CreatedDate' as a plain datetime triggers lint error MDL020.
  CreatedDate: autocreateddate,

  -- Enumeration
  status: Module.CustomerStatus default Active,

  -- Auto-number
  CustomerNumber: autonumber
);
/
```

## Non-Persistent Entity

Used for temporary data, form parameters, or calculated values.

```mdl
/**
 * Search parameters for customer search form
 */
create non-persistent entity Module.CustomerSearchParams (
  SearchName: string(100),
  SearchEmail: string(200),
  MinCreditLimit: decimal,
  IncludeInactive: boolean default false
);
/
```

## Attribute Types

| Type | Syntax | Example |
|------|--------|---------|
| String | `Name: string(length)` | `Name: string(100)` |
| Integer | `Name: integer` | `count: integer` |
| Long | `Name: long` | `BigNumber: long` |
| Decimal | `Name: decimal` | `Amount: decimal` |
| Boolean | `Name: boolean` | `IsActive: boolean` |
| DateTime | `Name: datetime` | `CreatedAt: datetime` |
| Date | `Name: date` | `BirthDate: date` |
| Enumeration | `Name: Module.EnumName` | `status: Module.Status` |
| AutoNumber | `Name: autonumber` | `Code: autonumber` |
| Binary | `Name: binary` | `FileData: binary` |
| Hashed String | `Name: hashedstring` | `password: hashedstring` |

## Attribute Modifiers

| Modifier | Meaning | Example |
|----------|---------|---------|
| `not null` | Required field | `Name: string(100) not null` |
| `unique` | Unique constraint | `Code: string(20) unique` |
| `default value` | Default value | `IsActive: boolean default true` |

**Note:** Boolean attributes auto-default to `false` when no `default` is specified.

## Generalization (Inheritance)

**CRITICAL: EXTENDS goes BEFORE the opening parenthesis, not after!**

```mdl
/**
 * Base entity
 */
create persistent entity Module.Person (
  PersonName: string(100) not null,
  Email: string(200)
);
/

/**
 * Customer extends Person - EXTENDS before (
 */
create persistent entity Module.Customer extends Module.Person (
  CustomerCode: string(20),
  CreditLimit: decimal
);
/
```

Common parent entities for file/image storage:
```mdl
-- Image entity (inherits Name, Size, Contents, thumbnail)
create persistent entity Module.ProductPhoto extends System.Image (
  PhotoCaption: string(200),
  SortOrder: integer default 0
);

-- File document (inherits Name, Size, Contents)
create persistent entity Module.Attachment extends System.FileDocument (
  AttachmentDescription: string(500)
);
```

**Wrong** (parse error):
```mdl
-- EXTENDS after ) = parse error!
create persistent entity Module.Photo (
  PhotoCaption: string(200)
) extends System.Image;
```

## Associations

### Reference (Many-to-One)

```mdl
/**
 * Order belongs to one Customer
 */
create association Module.Order_Customer (
  PARENT Module.Customer,
  CHILD Module.Order
);
/
```

### Reference Set (Many-to-Many)

```mdl
/**
 * Product can be in many Categories
 * Category can have many Products
 */
create association Module.Product_Category (
  PARENT Module.Category as reference set,
  CHILD Module.Product
);
/
```

### Association with Delete Behavior

```mdl
/**
 * Delete orders when customer is deleted
 */
create association Module.Order_Customer (
  PARENT Module.Customer,
  CHILD Module.Order,
  delete PARENT cascade  -- Delete orders when customer deleted
);
/
```

Delete behaviors:
- `delete PARENT cascade` - Delete children when parent deleted
- `delete PARENT prevent` - Prevent deletion if children exist
- `delete CHILD cascade` - Delete parent when last child deleted

## Enumerations

```mdl
/**
 * Order status values
 */
create enumeration Module.OrderStatus (
  Draft = 'Draft',
  Pending = 'Pending',
  Approved = 'Approved',
  Shipped = 'Shipped',
  Delivered = 'Delivered',
  Cancelled = 'Cancelled'
);
/
```

## View Entity (OQL)

```mdl
/**
 * Monthly sales summary by customer
 */
create view entity Module.CustomerSalesSummary (
  CustomerName: string(100),
  TotalOrders: integer,
  TotalAmount: decimal,
  LastOrderDate: datetime
)
as
  select
    c.Name as CustomerName,
    count(o.OrderID) as TotalOrders,
    sum(o.Amount) as TotalAmount,
    max(o.OrderDate) as LastOrderDate
  from Module.Customer c
  left join c/Module.Order_Customer/Module.Order o
  GROUP by c.Name;
/
```

## Entity with Index

```mdl
/**
 * Product with search index
 */
create persistent entity Module.Product (
  Code: string(20) not null,
  Name: string(100) not null,
  Category: string(50),
  Price: decimal
)
index idx_product_code on (Code)
index idx_product_category on (Category);
/
```

## Complete Domain Model Example

```mdl
-- Enumeration
create enumeration Shop.OrderStatus (
  Draft = 'Draft',
  Confirmed = 'Confirmed',
  Shipped = 'Shipped',
  Delivered = 'Delivered'
);
/

-- Customer entity
create persistent entity Shop.Customer (
  Name: string(100) not null,
  Email: string(200) not null unique,
  Phone: string(20),
  IsActive: boolean default true,
  CreatedDate: autocreateddate
);
/

-- Product entity
create persistent entity Shop.Product (
  Code: string(20) not null unique,
  Name: string(100) not null,
  description: string(500),
  Price: decimal not null,
  Stock: integer default 0,
  IsAvailable: boolean default true
);
/

-- Order entity
create persistent entity Shop.Order (
  OrderNumber: autonumber,
  OrderDate: datetime not null,
  status: Shop.OrderStatus default Draft,
  TotalAmount: decimal,
  Notes: string(500)
);
/

-- Order line entity
create persistent entity Shop.OrderLine (
  Quantity: integer not null,
  UnitPrice: decimal not null,
  LineTotal: decimal
);
/

-- Associations
create association Shop.Order_Customer (
  PARENT Shop.Customer,
  CHILD Shop.Order
);
/

create association Shop.OrderLine_Order (
  PARENT Shop.Order,
  CHILD Shop.OrderLine,
  delete PARENT cascade
);
/

create association Shop.OrderLine_Product (
  PARENT Shop.Product,
  CHILD Shop.OrderLine
);
/
```

## Quick Reference

### Entity Creation
```mdl
create persistent entity Module.Name (attributes);
create non-persistent entity Module.Name (attributes);
create view entity Module.Name (attributes) as select ...;
```

### Attribute Syntax
```mdl
attributename: type [(length)] [not null] [unique] [default value]
```

### Association Syntax
```mdl
create association Module.Name (
  PARENT Module.ParentEntity [as reference set],
  CHILD Module.ChildEntity
  [, delete PARENT cascade|prevent]
);
```

### Enumeration Syntax
```mdl
create enumeration Module.Name (
  Value1 = 'Caption1',
  Value2 = 'Caption2'
);
```
