# Available Tables

The catalog provides several tables that can be queried using standard SQL syntax. Use `SHOW CATALOG TABLES` to list all available tables in your catalog.

## Discovering Tables

```sql
SHOW CATALOG TABLES;
```

## Core Tables

### CATALOG.ENTITIES

Information about all entities in the project.

| Column | Description |
|--------|-------------|
| `Id` | Unique identifier |
| `Name` | Entity name |
| `ModuleName` | Module containing the entity |
| `QualifiedName` | Full qualified name (Module.Entity) |
| `EntityType` | Persistent, Non-Persistent, View, External |
| `Description` | Documentation text |
| `AttributeCount` | Number of attributes |
| `AccessRuleCount` | Number of access rules defined |

```sql
SELECT Name, EntityType, AttributeCount
FROM CATALOG.ENTITIES
WHERE ModuleName = 'Sales'
ORDER BY Name;
```

### CATALOG.ATTRIBUTES

Information about entity attributes.

| Column | Description |
|--------|-------------|
| `Id` | Unique identifier |
| `Name` | Attribute name |
| `EntityId` | Parent entity ID |
| `AttributeType` | String, Integer, Decimal, Boolean, DateTime, etc. |

```sql
SELECT a.Name, a.AttributeType
FROM CATALOG.ATTRIBUTES a
JOIN CATALOG.ENTITIES e ON a.EntityId = e.Id
WHERE e.QualifiedName = 'Sales.Customer';
```

### CATALOG.ASSOCIATIONS

Information about entity associations.

| Column | Description |
|--------|-------------|
| `Id` | Unique identifier |
| `Name` | Association name |
| `ParentEntity` | Parent (FROM) entity qualified name |
| `ChildEntity` | Child (TO) entity qualified name |
| `AssociationType` | Reference or ReferenceSet |

```sql
SELECT Name, ParentEntity, ChildEntity, AssociationType
FROM CATALOG.ASSOCIATIONS
WHERE ParentEntity LIKE '%Order%' OR ChildEntity LIKE '%Order%';
```

### CATALOG.MICROFLOWS

Information about microflows and nanoflows.

| Column | Description |
|--------|-------------|
| `Id` | Unique identifier |
| `Name` | Microflow name |
| `ModuleName` | Module containing the microflow |
| `QualifiedName` | Full qualified name |
| `ReturnType` | Return type of the microflow |
| `Description` | Documentation text |
| `Parameters` | Parameter information |
| `ObjectUsage` | Entities used in the microflow |

```sql
SELECT Name, ReturnType, Description
FROM CATALOG.MICROFLOWS
WHERE ModuleName = 'Sales'
ORDER BY Name;
```

### CATALOG.PAGES

Information about pages and their properties.

| Column | Description |
|--------|-------------|
| `Id` | Unique identifier |
| `Name` | Page name |
| `ModuleName` | Module containing the page |
| `QualifiedName` | Full qualified name |
| `URL` | Page URL if configured |
| `DataSource` | Primary data source |
| `WidgetTypes` | Types of widgets used |

```sql
SELECT Name, URL, DataSource
FROM CATALOG.PAGES
WHERE ModuleName = 'Sales'
ORDER BY Name;
```

### CATALOG.ACCESS_RULES

Information about entity access rules (available after full refresh).

| Column | Description |
|--------|-------------|
| `Id` | Unique identifier |
| `EntityId` | Entity this rule applies to |
| `UserRole` | Role this rule grants access to |
| `AllowRead` | Whether read access is granted |
| `AllowWrite` | Whether write access is granted |

```sql
SELECT e.QualifiedName, ar.UserRole, ar.AllowRead, ar.AllowWrite
FROM CATALOG.ACCESS_RULES ar
JOIN CATALOG.ENTITIES e ON ar.EntityId = e.Id
WHERE e.ModuleName = 'Sales';
```

## Listing All Tables

To see the complete list of available tables in your catalog (which may vary by project and refresh level):

```sql
SHOW CATALOG TABLES;
```

```bash
mxcli -p app.mpr -c "SHOW CATALOG TABLES"
```
