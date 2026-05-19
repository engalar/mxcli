# Understanding Mendix Associations

Use this skill whenever working with associations: creating them, reading `SHOW ASSOCIATIONS` output, troubleshooting access rules, or designing domain models.

## The Core Mental Model

An association in Mendix is a **directed link from the entity that stores the reference TO the entity being referenced**.

- The **FROM entity** = the owner = stores the foreign key (or association table row)
- The **TO entity** = the referenced entity

```
FROM (owner/FK holder)  →  TO (referenced)
     Order              →      Customer
```

One Order belongs to one Customer. Order stores the FK. So: `from Order to Customer`.

**Naming convention:** `FromEntity_ToEntity` → `Order_Customer`

## The Parent/Child Paradox (Read This Carefully)

Mendix uses the terms **Parent** and **Child** in a way that is the **opposite** of most SQL/ORM conventions:

| Mendix term | Means | In 1-* example |
|-------------|-------|----------------|
| **Parent**  | The FROM entity — the **owner**, the one that stores the FK | `Order` (many side) |
| **Child**   | The TO entity — the **referenced** entity | `Customer` (one side) |

In SQL/ORM: "parent" usually means the *one* side (e.g., Customer is parent of Order). In Mendix it is the reverse: the *many* side is called Parent because it *owns* the association.

**In `SHOW ASSOCIATIONS` output:**
- `Parent` column = the FROM entity = the owner = the many side (for 1-*)
- `Child` column = the TO entity = the referenced = the one side (for 1-*)

```
-- SHOW ASSOCIATIONS output for Order_Customer:
-- Parent: Order   Child: Customer
-- This means: Order owns the FK → one Customer, many Orders
```

## The Four Association Patterns

| Multiplicity | Symbol  | Type | Owner | MDL owner keyword |
|---|---|---|---|---|
| **One-to-many** (most common) | `*-->1` | `reference` | FROM entity only | `owner default` |
| **One-to-one** | `1--1`  | `reference` | Both entities | `owner both` |
| **Many-to-many (one-way)** | `*-->*` | `ReferenceSet` | FROM entity only | `owner default` |
| **Many-to-many (bidirectional)** | `*--*`  | `ReferenceSet` | Both entities | `owner both` |

**Reading the symbol:** `>` always marks the non-owning (TO) side. When both sides own, `--` has no `>`.

### One-to-Many (Reference + Default)

One Customer has many Orders. Each Order stores a reference to its Customer.

```sql
-- Order is the FROM (owner/many side), Customer is the TO (one side)
create association Sales.Order_Customer
from Sales.Order to Sales.Customer
type reference
owner default;
```

- `SHOW ASSOCIATIONS`: Parent = Order, Child = Customer
- The selector in the UI must be inside an **Order** data view (the owner)
- Database (association table): stored in `sales$order_customer`; or with direct storage, as FK column in `sales$order`

### One-to-One (Reference + Both)

One Customer has one Profile, one Profile belongs to one Customer.

```sql
create association Sales.Customer_Profile
from Sales.Customer to Sales.Profile
type reference
owner both;
```

Mendix **does** enforce 1-1 uniqueness at the database level — both sides are validated by the database. This is a true structural constraint, not just behavioral.

The association is stored in **both** objects. Selectors can be placed in either a Customer or Profile data view.

### Many-to-Many with Default Ownership (ReferenceSet + Default)

One Customer can belong to many Groups; one Group can have many Customers. The Customer entity owns the association.

```sql
-- Customer stores the list of Groups it belongs to
create association Sales.Customer_Group
from Sales.Customer to Sales.Group
type ReferenceSet
owner default;
```

- The reference set selector must be inside a **Customer** data view
- Navigating from Group → Customer works but may be slower

### Many-to-Many with Dual Ownership (ReferenceSet + Both)

Both ends store the association. Use only when you genuinely need to add/remove the association from both sides in the UI.

```sql
create association Sales.Accountant_Group
from Sales.Accountant to Sales.Group
type ReferenceSet
owner both;
```

**Avoid unless necessary.** Dual ownership means committing either object saves the association — more overhead, more complexity.

## Reading SHOW ASSOCIATIONS Output

```
SHOW ASSOCIATIONS [in module Sales];

Name                   | FROM (owner)     | TO (referenced)  | Multiplicity | Type         | Owner
-----------------------+------------------+------------------+--------------+--------------+-------
Sales.Order_Customer   | Sales.Order      | Sales.Customer   | *-->1        | Reference    | Default
Sales.Customer_Profile | Sales.Customer   | Sales.Profile    | 1--1         | Reference    | Both
Sales.Customer_Group   | Sales.Customer   | Sales.Group      | *-->*        | ReferenceSet | Default
Sales.Accountant_Group | Sales.Accountant | Sales.Group      | *--*         | ReferenceSet | Both
```

**Multiplicity notation — the `>` always marks the non-owning (TO) side:**

| Symbol  | Meaning | Pattern |
|---------|---------|---------|
| `*-->1` | FROM owns many, TO has one, TO does not own | one-to-many |
| `1--1`  | both own one each (no `>` — neither is non-owner) | one-to-one |
| `*-->*` | FROM owns, TO does not own | many-to-many (default) |
| `*--*`  | both own (no `>` — neither is non-owner) | many-to-many (both) |

**`SHOW ASSOCIATION Sales.Order_Customer` (single):**
```
Association: Sales.Order_Customer
  Multiplicity: *-->1
  FROM (owner): Sales.Order
  TO (referenced): Sales.Customer
  Type: Reference
  Owner: Default
```

**`DESCRIBE ASSOCIATION Sales.Order_Customer`:**
```sql
-- one-to-many: Sales.Order *-->1 Sales.Customer (each Sales.Order belongs to one Sales.Customer)
create association Sales.Order_Customer
from Sales.Order to Sales.Customer
type reference
owner default
delete_behavior DELETE_BUT_KEEP_REFERENCES;
```

## Common Mistakes

### Wrong direction: putting FK on the "one" side

```sql
-- ❌ WRONG: Customer doesn't store order references
create association Sales.Customer_Order
from Sales.Customer to Sales.Order
type reference;

-- ✅ CORRECT: Order stores the customer reference (FK)
create association Sales.Order_Customer
from Sales.Order to Sales.Customer
type reference;
```

Ask yourself: "Which object *knows about* the other?" The Order knows which Customer it belongs to. The Customer does not store a list of Order IDs.

### Confusing one-to-many with ReferenceSet

`type reference` + `owner default` = one-to-many (each owner stores exactly **one** reference).
`type ReferenceSet` = many-to-many (each owner stores **many** references).

Use `reference` for "each X belongs to one Y". Use `ReferenceSet` for "each X can have many Y".

### Assuming 1-1 is just behavioral

`owner both` on a `reference` creates true one-to-one: **both sides are enforced as unique at the database level**. It is not merely a UI convention.

## Association Storage

By default, associations are stored in a separate **association table**. For one-to-many and one-to-one associations you can use **direct associations** instead (FK column on the owner entity's table):

```sql
create association Sales.Order_Customer
from Sales.Order to Sales.Customer
type reference
owner default
storage direct;    -- FK column on Sales$Order table
```

Direct associations are faster for most queries. They cannot be used for many-to-many (ReferenceSet).

## Non-Persistable Entities

Associations involving a non-persistable entity **must start at the non-persistable entity** and use `owner default`. The persistable entity **cannot** be the owner.

```sql
-- ✅ CORRECT: non-persistable is FROM (owner)
create association MyModule.SearchFilter_Result
from MyModule.SearchFilter to MyModule.SearchResult
type ReferenceSet
owner default;

-- ❌ WRONG: persistable entity cannot be owner when paired with non-persistable
create association MyModule.SearchResult_SearchFilter
from MyModule.SearchResult to MyModule.SearchFilter  -- SearchResult is persistable → error
type reference
owner default;
```

mxcli enforces this at `CREATE ASSOCIATION` time and returns an error if violated.

## Quick Decision Guide

```
What is the cardinality?
│
├─ Each A has exactly ONE B (and each B has many A)
│   → type reference, owner default
│   → from A to B  (A stores the FK)
│
├─ Each A has at most ONE B AND each B has at most ONE A
│   → type reference, owner both
│   → from either end (convention: from A to B)
│   ✓ Enforced at DB level (uniqueness on both sides)
│
└─ Each A can have MANY B (and each B can have many A)
    → type ReferenceSet
    ├─ Need selector from A side only → owner default, from A to B
    └─ Need selector from both sides → owner both
```
