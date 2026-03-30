# Creating Mendix Domain Model MDL Scripts

Use this skill to generate Mendix domain model scripts in MDL (Mendix Definition Language) format and validate them with the linter.

## When to Use This Skill

- User asks to create a domain model for a specific use case
- User wants to generate entities, associations, and enumerations
- User requests a complete e-commerce, HR, CRM, or other business domain model
- User needs validation of generated MDL scripts

## MDL Syntax Reference

**CRITICAL: All CREATE statements MUST have JavaDoc-style documentation**

Every CREATE statement (modules, entities, associations, enumerations, microflows) should have a /** ... */ comment explaining its purpose. This is essential for:
- Team collaboration and knowledge transfer
- Understanding domain model structure
- Long-term maintainability
- Auto-generated documentation

### Module Creation

```sql
/**
 * Module for financial transaction management
 *
 * Handles accounts, transactions, budgets, and reporting.
 *
 * @since 1.0.0
 */
CREATE MODULE Finance;
```

### Minimap Section Headers (MARK Comments)

**IMPORTANT: Large MDL files (300+ lines) MUST use MARK comments for navigation**

Use `-- MARK: Section Name` comments to create collapsible sections in code editors. This dramatically improves navigation and organization in large domain model files.

**Format**: `-- MARK: Section Name`

**Required for files:**
- 300+ lines: At least 3 MARK comments
- 500+ lines: At least 5 MARK comments

**Recommended sections:**
```sql
-- MARK: ENUMERATIONS

-- MARK: CORE ENTITIES

-- MARK: ASSOCIATIONS

-- MARK: VIEW ENTITIES

-- MARK: MICROFLOWS
```

**With subsections:**
```sql
-- MARK: - Core Entities (Persistent)

-- MARK: - View Entities for Reporting
```

**Benefits:**
- Creates outline/minimap view in VS Code, Xcode-style editors
- Makes large files navigable with jump-to-section
- Groups related code logically
- Improves team collaboration on complex models

### Enumerations

```sql
/**
 * Transaction type classification
 *
 * Categorizes financial transactions as income or expense
 * for proper accounting and reporting.
 *
 * @since 1.0.0
 */
CREATE ENUMERATION Module.TransactionType (
  INCOME 'Income',
  EXPENSE 'Expense'
);
```

### Entities

**IMPORTANT: All entities MUST have @Position annotation**

The `@Position(x, y)` annotation specifies where the entity appears in the domain model diagram. Without it, entities appear at (0,0) or random locations.

**Position Guidelines:**
- Use increments of 50 or 100 for spacing (e.g., 100, 200, 300)
- Leave space between entities (at least 200 pixels)
- Organize related entities in logical groups
- Example layout: Categories at y=100, Transactions at y=300, Reports at y=500

#### Persistent Entity

```sql
/**
 * Entity description
 *
 * Detailed explanation of what this entity represents.
 *
 * @since 1.0.0
 * @see Module.RelatedEntity
 */
@Position(100, 100)
CREATE PERSISTENT ENTITY Module.EntityName (
  /** Unique identifier */
  Id: Long NOT NULL ERROR 'ID is required' UNIQUE ERROR 'ID must be unique',
  /** Attribute description */
  AttributeName: String(200) NOT NULL ERROR 'Attribute name is required',
  /** Numeric value */
  Amount: Decimal,
  /** Date field */
  CreationDate: Date,
  /** Boolean flag */
  IsActive: Boolean NOT NULL ERROR 'IsActive flag is required' DEFAULT true,
  /** Enumeration field */
  Status: Enumeration(Module.StatusEnum) NOT NULL ERROR 'Status is required'
);
```

#### Entity Indexes (Performance Optimization)

**CRITICAL: INDEX syntax goes AFTER the closing parenthesis, with NO comma before**

Indexes improve query performance for frequently filtered or sorted columns. Add them to persistent entities when:
- Column is used in WHERE clauses frequently
- Column is used for sorting (ORDER BY)
- Composite indexes for multi-column filters

**Syntax:**
```sql
CREATE PERSISTENT ENTITY Module.Transaction (
  TransactionDate: DateTime NOT NULL,
  Status: Enumeration(Module.Status) NOT NULL,
  Amount: Decimal NOT NULL,
  IsRecurring: Boolean DEFAULT false
)
INDEX (TransactionDate DESC)
INDEX (Status, TransactionDate)
INDEX (IsRecurring);
```

**Index Guidelines:**
- **Position**: AFTER closing parenthesis, NO comma before first INDEX
- **No names**: Unlike SQL CREATE INDEX, MDL indexes don't have names
- **Sort direction**: ASC or DESC are optional (default is ASC)
- **Composite indexes**: Order matters - put most selective columns first
- **Limit**: Don't over-index - each index has storage/write overhead

**Common index patterns:**
- Date fields: `INDEX (CreatedDate DESC)` - for recent-first queries
- Status filters: `INDEX (Status, CreatedDate DESC)` - for filtered date ranges
- Boolean flags: `INDEX (IsActive)` - for active/inactive filtering
- Foreign keys: Automatically indexed by associations

#### Entity Generalization (EXTENDS)

**CRITICAL: EXTENDS goes BEFORE the opening parenthesis, not after!**

Use `EXTENDS` to inherit from a parent entity. Common for file/image storage using System entities.

```sql
-- Correct: EXTENDS before (
CREATE PERSISTENT ENTITY Module.ProductPhoto EXTENDS System.Image (
  PhotoCaption: String(200),
  SortOrder: Integer DEFAULT 0
);

-- Correct: File document specialization
CREATE PERSISTENT ENTITY Module.Attachment EXTENDS System.FileDocument (
  AttachmentDescription: String(500)
);

-- Correct: Custom entity inheritance
CREATE PERSISTENT ENTITY Module.Employee EXTENDS Module.Person (
  EmployeeNumber: String(20)
);
```

**Wrong** (parse error):
```sql
-- EXTENDS after ) = parse error!
CREATE PERSISTENT ENTITY Module.Photo (
  PhotoCaption: String(200)
) EXTENDS System.Image;
```

**Note:** `mxcli syntax entity` output may show EXTENDS after `)` — this is misleading. Always place EXTENDS before `(`.

#### Non-Persistent Entity

**IMPORTANT: Non-persistent entities cannot have validation rules** (`NOT NULL ERROR`, `UNIQUE ERROR`) on attributes. They can only have `DEFAULT` values.

```sql
/**
 * Non-persistent entity description
 *
 * @since 1.0.0
 */
@Position(200, 100)
CREATE NON-PERSISTENT ENTITY Module.TemporaryData (
  SessionId: String(100),
  Data: String(1000),
  IsActive: Boolean DEFAULT false
);
```

#### View Entity (with OQL)

```sql
/**
 * View entity description
 *
 * @since 1.0.0
 */
@Position(300, 500)
CREATE VIEW ENTITY Module.ViewName (
  Attribute1: Type,
  Attribute2: Type
) AS (
  SELECT
    e.Id AS Id,
    e.Name AS Name,
    e.Amount AS Amount
  FROM Module.Entity AS e
  WHERE e.IsActive = true
);
```

**Enumeration Comparisons in OQL:**

When comparing enumeration attributes in OQL WHERE clauses, use the **enumeration value** (identifier), not the caption:

```sql
-- Enumeration definition
CREATE ENUMERATION Module.OrderStatus (
  PENDING 'Pending',
  PROCESSING 'Processing',
  CANCELLED 'Cancelled'
);

-- OQL comparison - use the VALUE, not the caption
WHERE e.Status != 'CANCELLED'   -- Correct: uses enum value
WHERE e.Status != 'Cancelled'   -- Wrong: this is the caption
```

### Associations

**CRITICAL: Association Directionality**

In Mendix, associations are defined **FROM the entity that contains the foreign key TO the entity that is referenced**.

Think of it like this:
- A `Transaction` knows which `Account` it belongs to → Transaction contains the foreign key
- Therefore: `FROM Transaction TO Account`
- **NOT** `FROM Account TO Transaction` ❌

**Common Patterns**:

```sql
-- ❌ INCORRECT: Account doesn't store transaction references
CREATE ASSOCIATION Finance.Account_Transaction
FROM Finance.Account TO Finance.Transaction
TYPE REFERENCE;

-- ✅ CORRECT: Transaction stores the account reference (foreign key)
CREATE ASSOCIATION Finance.Transaction_Account
FROM Finance.Transaction TO Finance.Account
TYPE REFERENCE;

-- ✅ One-to-Many: Customer has many Orders (each order knows its customer)
CREATE ASSOCIATION Sales.Order_Customer
FROM Sales.Order TO Sales.Customer
TYPE REFERENCE;

-- ✅ Many-to-Many: Use ReferenceSet and choose which side stores the relationship
CREATE ASSOCIATION Sales.Order_Products
FROM Sales.Order TO Sales.Product
TYPE ReferenceSet
OWNER Both;
```

**Full Association Syntax**:

```sql
/**
 * Association description
 *
 * Explain the relationship and directionality.
 *
 * @since 1.0.0
 */
CREATE ASSOCIATION Module.EntityWithFK_ReferencedEntity
FROM Module.EntityWithFK TO Module.ReferencedEntity
TYPE Reference
OWNER Default
DELETE_BEHAVIOR DELETE_BUT_KEEP_REFERENCES
COMMENT 'Additional documentation';
```

**Association Types**:
- `Reference` - One-to-one or many-to-one (foreign key on FROM entity)
- `ReferenceSet` - One-to-many or many-to-many (collection)

**Owner Options**:
- `Default` - Standard ownership (FROM entity owns the reference)
- `Both` - Both sides own the association (bidirectional)
- `Parent` - Only parent (TO) entity owns
- `Child` - Only child (FROM) entity owns

**Delete Behaviors**:
- `DELETE_AND_REFERENCES` - Delete object and all referencing objects
- `DELETE_BUT_KEEP_REFERENCES` - Delete object, keep references (nullify)
- `DELETE_IF_NO_REFERENCES` - Only delete if no objects reference it
- `CASCADE` - Cascade delete to associated objects
- `PREVENT` - Prevent deletion if references exist

**Naming Convention**: `{FromEntity}_{ToEntity}` (e.g., `Order_Customer`, `Transaction_Account`)

#### Calculated Attributes

Calculated attributes derive their value from a microflow at runtime. Use `CALCULATED BY Module.Microflow` to specify the calculation microflow.

**IMPORTANT: CALCULATED attributes are only supported on PERSISTENT entities.** Using CALCULATED on non-persistent entities will produce a validation error.

```sql
@Position(100, 100)
CREATE PERSISTENT ENTITY Module.OrderLine (
  /** Unit price */
  UnitPrice: Decimal NOT NULL,
  /** Quantity ordered */
  Quantity: Integer NOT NULL,
  /** Total price, calculated by microflow */
  TotalPrice: Decimal CALCULATED BY Module.CalcTotalPrice
);
```

**Syntax variants:**
- `CALCULATED BY Module.Microflow` — recommended, binds the calculation microflow directly
- `CALCULATED Module.Microflow` — also valid (`BY` keyword is optional)
- `CALCULATED` — bare form, marks as calculated but requires manual microflow binding in Studio Pro

### Data Types

| Type | Example | Description |
|------|---------|-------------|
| `String(length)` | `String(200)` | Text field with max length |
| `Integer` | `Integer` | 32-bit integer |
| `Long` | `Long` | 64-bit integer (use for IDs) |
| `Decimal` | `Decimal` | Decimal number |
| `Boolean` | `Boolean` | True/false |
| `DateTime` | `DateTime` | Date and time |
| `Date` | `Date` | Date only |
| `Binary` | `Binary` | Binary data |
| `AutoNumber` | `AutoNumber DEFAULT 1` | Auto-incrementing number (requires DEFAULT start value) |
| `Enumeration(Module.Enum)` | `Enumeration(Shop.Status)` | Enumeration reference |

### Constraints

**Basic Constraints:**
- `NOT NULL` - Field is required
- `UNIQUE` - Value must be unique
- `DEFAULT value` - Default value

**Validation Error Messages:**

Each constraint can have a custom error message using `ERROR 'message'` syntax:

```sql
CREATE PERSISTENT ENTITY Module.Customer (
  /** Customer name - required with custom error */
  Name: String(200) NOT NULL ERROR 'Name is required',
  /** Email - required and unique with separate error messages */
  Email: String(200) NOT NULL ERROR 'Email is required' UNIQUE ERROR 'Email must be unique',
  /** Age with default value */
  Age: Integer DEFAULT 0,
  /** Active status flag */
  IsActive: Boolean NOT NULL ERROR 'IsActive flag is required' DEFAULT true
);
```

**Error Message Guidelines:**
- Place `ERROR 'message'` immediately after the constraint
- Multiple constraints can each have their own error message
- Keep messages clear and user-friendly
- Follow the pattern: `NOT NULL ERROR 'X is required'` for required fields
- For UNIQUE: `UNIQUE ERROR 'X must be unique'`
- Error messages are shown to end users during validation

**Common patterns:**
```sql
-- Required field
Name: String(200) NOT NULL ERROR 'Name is required',

-- Required and unique
Email: String(200) NOT NULL ERROR 'Email is required' UNIQUE ERROR 'Email must be unique',

-- Required with default
IsActive: Boolean NOT NULL ERROR 'IsActive flag is required' DEFAULT true,

-- Enum with required error
Status: Enumeration(Module.Status) NOT NULL ERROR 'Status is required',

-- Enum with default value (use fully qualified Module.Enum.Value)
Priority: Enumeration(Module.Priority) DEFAULT Module.Priority.Normal
```

## Reserved Keywords

**Best practice: Always quote all identifiers** (entity names, attribute names) with double quotes. This eliminates all reserved keyword conflicts and is always safe — quotes are stripped automatically by the parser.

```sql
CREATE PERSISTENT ENTITY Module."VATRate" (
  "Create": DateTime,
  "Rate": Decimal,
  "Status": String(50)
);
```

Both `"Name"` and `` `Name` `` syntax are supported. Prefer double quotes for consistency.

**Boolean attributes** auto-default to `false` when no `DEFAULT` is specified:
```sql
CREATE PERSISTENT ENTITY Module.Item (
  IsActive: Boolean,           -- auto-defaults to false
  IsPublished: Boolean DEFAULT true
);
```

## Entity Positioning

Use `@Position(x, y)` to control layout in Studio Pro:
- Place related entities near each other
- Use consistent spacing (e.g., 250 pixels horizontal, 200 vertical)
- Group by domain concept

Example layout:
```sql
@Position(50, 50)      -- Top-left: Core entity
CREATE PERSISTENT ENTITY Module.Customer (...);

@Position(300, 50)     -- Same row: Related entity
CREATE PERSISTENT ENTITY Module.Address (...);

@Position(50, 250)     -- Below: Dependent entity
CREATE PERSISTENT ENTITY Module.Order (...);
```

## Documentation Best Practices

### Entity Documentation

```sql
/**
 * Brief one-line summary
 *
 * Detailed multi-line description explaining:
 * - What the entity represents
 * - Key business rules
 * - Relationships to other entities
 *
 * @since 1.0.0
 * @see Module.RelatedEntity
 */
```

### Attribute Documentation

```sql
/** Brief description of what this attribute stores */
AttributeName: Type,
```

### Association Documentation

```sql
/**
 * Relationship description
 *
 * Explains the business meaning of this association.
 *
 * @since 1.0.0
 */
```

## Step-by-Step Process

### 1. Analyze Requirements

When user requests a domain model:
1. Identify core entities (nouns)
2. Identify enumerations (status, types, categories)
3. Identify relationships (associations)
4. Identify attributes for each entity
5. Check for reserved keyword conflicts

### 2. Generate MDL Script

Create script with this structure:
```sql
-- ============================================================================
-- Domain Model Name
-- ============================================================================
-- Description of the domain
-- ============================================================================

-- MARK: ENUMERATIONS

CREATE ENUMERATION Module.Enum1 (...);
CREATE ENUMERATION Module.Enum2 (...);

-- MARK: CORE ENTITIES

-- MARK: - Entity Group 1

CREATE PERSISTENT ENTITY Module.Entity1 (...);
CREATE PERSISTENT ENTITY Module.Entity2 (...);

-- MARK: - Entity Group 2

CREATE PERSISTENT ENTITY Module.Entity3 (...);

-- MARK: VIEW ENTITIES

CREATE VIEW ENTITY Module.View1 AS ...;

-- MARK: ASSOCIATIONS

-- MARK: - Entity Group 1 Associations

CREATE ASSOCIATION Module.Assoc1 ...;
CREATE ASSOCIATION Module.Assoc2 ...;
```

### 3. Validate with Linter

Run the linter to check for issues:

```bash
# Standalone test
node dist/test-linter-standalone.js

# Or create a custom test file
```

The linter will detect:
- ✅ Reserved keywords (CE7247)
- ✅ Duplicate names (CE0065)
- ✅ OQL syntax errors (CE0174)

### 4. Review and Fix Issues

**Common Issues**:

1. **Reserved Keyword Error**:
   ```
   error: Reserved keyword 'CreatedDate' used as attribute name
   💡 Rename to 'CreationDate'
   ```
   Fix: Rename to suggested alternative

2. **Duplicate Name Error**:
   ```
   error: Duplicate name 'Status' in module 'Shop'
   💡 Rename one of the enumeration, entity to avoid conflict
   ```
   Fix: Rename entity to `OrderStatus` or similar

3. **OQL Syntax Error**:
   ```
   error: ORDER BY requires LIMIT or OFFSET
   💡 Add LIMIT clause to query
   ```
   Fix: Add `LIMIT 100` to view entity query

### 5. Generate Complete Script

Ensure:
- ✅ All entities have JavaDoc documentation
- ✅ All attributes have inline comments
- ✅ All associations have descriptions
- ✅ Position annotations for all entities
- ✅ No reserved keywords
- ✅ No duplicate names
- ✅ Valid OQL queries

## Example: E-Commerce Domain Model

```sql
-- ============================================================================
-- E-Commerce Domain Model
-- ============================================================================

CREATE MODULE ECommerce;

-- Enumerations
-- ============================================================================

/**
 * Order status enumeration
 *
 * @since 1.0.0
 */
CREATE ENUMERATION ECommerce.OrderStatus (
  Draft 'Draft',
  Submitted 'Submitted',
  Paid 'Paid',
  Shipped 'Shipped',
  Delivered 'Delivered',
  Cancelled 'Cancelled'
);

-- Entities
-- ============================================================================

-- Customer Management
-- ----------------------------------------------------------------------------

/**
 * Customer entity
 *
 * Stores customer information for e-commerce platform.
 *
 * @since 1.0.0
 * @see ECommerce.SalesOrder
 */
@Position(50, 50)
CREATE PERSISTENT ENTITY ECommerce.Customer (
  /** Unique customer identifier */
  CustomerId: Long NOT NULL ERROR 'Customer ID is required' UNIQUE ERROR 'Customer ID must be unique',
  /** Customer full name */
  FullName: String(200) NOT NULL ERROR 'Full name is required',
  /** Email address */
  Email: String(200) NOT NULL ERROR 'Email is required' UNIQUE ERROR 'Email must be unique',
  /** Registration date */
  RegistrationDate: DateTime NOT NULL ERROR 'Registration date is required'
);

/**
 * Product entity
 *
 * Catalog of products available for purchase.
 *
 * @since 1.0.0
 */
@Position(50, 250)
CREATE PERSISTENT ENTITY ECommerce.Product (
  /** Unique product identifier */
  ProductId: Long NOT NULL ERROR 'Product ID is required' UNIQUE ERROR 'Product ID must be unique',
  /** Product name */
  ProductName: String(200) NOT NULL ERROR 'Product name is required',
  /** Product SKU */
  SKU: String(50) NOT NULL ERROR 'SKU is required' UNIQUE ERROR 'SKU must be unique',
  /** Unit price */
  Price: Decimal NOT NULL ERROR 'Price is required',
  /** Stock quantity */
  StockQuantity: Integer NOT NULL ERROR 'Stock quantity is required'
);

/**
 * Sales order entity
 *
 * Customer orders for products.
 *
 * @since 1.0.0
 */
@Position(300, 150)
CREATE PERSISTENT ENTITY ECommerce.SalesOrder (
  /** Unique order identifier */
  OrderId: Long NOT NULL ERROR 'Order ID is required' UNIQUE ERROR 'Order ID must be unique',
  /** Order number */
  OrderNumber: String(50) NOT NULL ERROR 'Order number is required' UNIQUE ERROR 'Order number must be unique',
  /** Order date */
  OrderDate: DateTime NOT NULL ERROR 'Order date is required',
  /** Total amount */
  TotalAmount: Decimal NOT NULL ERROR 'Total amount is required',
  /** Order status */
  Status: Enumeration(ECommerce.OrderStatus) NOT NULL ERROR 'Status is required'
);

-- Associations
-- ============================================================================

/**
 * Customer orders
 *
 * Links customers to their orders.
 *
 * @since 1.0.0
 */
CREATE ASSOCIATION ECommerce.Customer_Orders
FROM ECommerce.Customer TO ECommerce.SalesOrder
TYPE ReferenceSet
OWNER Both;
```

## Testing the Script

1. **Save to file**: Save as `examples/my-domain-model.mdl`

2. **Run standalone linter**:
   ```bash
   node dist/test-linter-standalone.js
   ```

3. **Execute in REPL**:
   ```sql
   mendix> CONNECT TO FILESYSTEM 'path/to/project.mpr';
   mendix> EXECUTE SCRIPT 'examples/my-domain-model.mdl';
   ```

4. **Check Studio Pro**: Open project and verify entities appear correctly

## Common Patterns

### One-to-Many Relationship
```sql
-- Parent entity
CREATE PERSISTENT ENTITY Module.Parent (Id: Long NOT NULL UNIQUE);

-- Child entity
CREATE PERSISTENT ENTITY Module.Child (
  Id: Long NOT NULL UNIQUE,
  ChildData: String(200)
);

-- Association (Parent has many Children)
CREATE ASSOCIATION Module.Parent_Children
FROM Module.Parent TO Module.Child
TYPE ReferenceSet
OWNER Both;
```

### Many-to-Many Relationship
```sql
-- Entity A
CREATE PERSISTENT ENTITY Module.EntityA (Id: Long NOT NULL UNIQUE);

-- Entity B
CREATE PERSISTENT ENTITY Module.EntityB (Id: Long NOT NULL UNIQUE);

-- Bidirectional association
CREATE ASSOCIATION Module.EntityA_EntityB
FROM Module.EntityA TO Module.EntityB
TYPE ReferenceSet
OWNER Both;
```

### Hierarchical Relationship (Self-Reference)

**IMPORTANT: Self-referencing associations must use `OWNER Default`** (one-to-many). Using `OWNER Both` is not supported for self-references.

```sql
/**
 * Category with parent-child hierarchy
 */
CREATE PERSISTENT ENTITY Module.Category (
  Id: Long NOT NULL UNIQUE,
  CategoryName: String(200) NOT NULL
);

/**
 * Parent category link (self-reference)
 */
CREATE ASSOCIATION Module.Category_ParentCategory
FROM Module.Category TO Module.Category
TYPE Reference
OWNER Default;
```

### ALTER ENTITY (Incremental Modifications)

Use `ALTER ENTITY` to make targeted changes to existing entities without redefining the entire entity:

```sql
-- Add a new attribute
ALTER ENTITY Module.Customer
  ADD ATTRIBUTE PhoneNumber: String(20);

-- Add multiple attributes at once
ALTER ENTITY Module.Order
  ADD ATTRIBUTE VATRate: Decimal
  ADD ATTRIBUTE VATAmount: Decimal;

-- Rename an attribute (preserves data)
ALTER ENTITY Module.Order
  RENAME ATTRIBUTE CreatedDate TO OrderDate;

-- Drop an attribute
ALTER ENTITY Module.Product
  DROP ATTRIBUTE LegacyCode;

-- Modify attribute type
ALTER ENTITY Module.Customer
  MODIFY ATTRIBUTE Address: String(500);

-- Set entity documentation
ALTER ENTITY Module.Customer
  SET DOCUMENTATION 'Core customer entity for CRM module';

-- Add an index
ALTER ENTITY Module.Customer
  ADD INDEX idx_email (Email ASC);

-- Reposition entity on domain model canvas
ALTER ENTITY Module.Customer
  SET POSITION (100, 200);
```

**Supported operations:** ADD ATTRIBUTE, RENAME ATTRIBUTE, MODIFY ATTRIBUTE, DROP ATTRIBUTE, SET DOCUMENTATION, SET COMMENT, ADD INDEX, DROP INDEX, SET POSITION.

### Entity Migration with CREATE OR MODIFY

Use `CREATE OR MODIFY` to update existing entities without losing data. The REPL computes differences and applies incremental changes.

```sql
/**
 * Customer entity migration - rename CustomerName to FullName
 */
CREATE OR MODIFY PERSISTENT ENTITY Module.Customer (
  /** Unique identifier (unchanged) */
  CustomerId: Long NOT NULL UNIQUE,

  /** Renamed from CustomerName - data preserved */
  @RenamedFrom('CustomerName')
  FullName: String(200) NOT NULL,

  /** New field */
  Email: String(255) UNIQUE,

  /** Type widened from String(100) to String(200) */
  Address: String(200)
);
```

**Key features:**
- `@RenamedFrom('oldName')` - renames attribute, preserves data
- Auto-removes attributes not in new definition
- Allows compatible type changes (e.g., String length increase)
- Preserves entity UUID (no data loss)

### Status-Driven Entity
```sql
-- Status enumeration
CREATE ENUMERATION Module.TaskStatus (
  Todo 'To Do',
  InProgress 'In Progress',
  Done 'Done'
);

-- Entity with status
CREATE PERSISTENT ENTITY Module.Task (
  Id: Long NOT NULL UNIQUE,
  TaskName: String(200) NOT NULL,
  Status: Enumeration(Module.TaskStatus) NOT NULL
);
```

## Checklist

Before finalizing an MDL script:

- [ ] All entities have JavaDoc documentation
- [ ] All attributes have inline comments
- [ ] All associations have descriptions
- [ ] Position annotations on all entities
- [ ] MARK comments for files 300+ lines (at least 3 sections)
- [ ] All identifiers quoted with double quotes
- [ ] No duplicate names (run linter)
- [ ] Valid OQL queries in view entities (run linter)
- [ ] Consistent naming conventions (PascalCase)
- [ ] Appropriate data types and lengths
- [ ] Required fields marked with NOT NULL
- [ ] Validation error messages added for NOT NULL and UNIQUE constraints
- [ ] IDs marked with NOT NULL UNIQUE
- [ ] Email/unique fields marked with UNIQUE

## References

- **Reserved Keywords**: `packages/mendix-repl/docs/reference/reserved-keywords.md`
- **Linter Proposal**: `packages/mendix-repl/docs/proposals/mdl-linter-proposal.md`
- **Example Scripts**:
  - `packages/mendix-repl/examples/shop-domain-model.mdl`
  - `packages/mendix-repl/examples/pet-store-domain-model.mdl`
- **Linter Test**: `packages/mendix-repl/src/test-linter-standalone.ts`

## Tips for AI Assistants

1. **Always quote all identifiers** with double quotes to avoid reserved keyword conflicts
2. **Use descriptive names** (ServiceType, CustomerOrder)
3. **Run linter** on generated scripts before presenting to user
4. **Fix all errors** reported by linter before finalizing
5. **Follow examples** in shop-domain-model.mdl and pet-store-domain-model.mdl
6. **Document thoroughly** - Studio Pro users benefit from good documentation
7. **Position thoughtfully** - Related entities should be visually grouped
8. **Test incrementally** - Generate in sections and validate each part