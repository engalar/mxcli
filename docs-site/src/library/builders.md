# Builders

The `api` package provides fluent builder types for constructing Mendix model elements. Each builder follows the pattern: create, configure with chained methods, then call `Build()`.

## EntityBuilder

Creates entities with attributes, persistence settings, and documentation.

```go
entity, err := modelAPI.DomainModels.CreateEntity("Customer").
    Persistent().
    WithStringAttribute("Name", 200).
    WithStringAttribute("Email", 254).
    WithIntegerAttribute("Age").
    WithBooleanAttribute("IsActive").
    WithDateTimeAttribute("CreatedDate", true).
    WithDecimalAttribute("Revenue").
    WithEnumerationAttribute("Status", "MyModule.CustomerStatus").
    Build()
```

### EntityBuilder Methods

| Method | Description |
|--------|-------------|
| `Persistent()` | Mark as persistent (stored in database) |
| `NonPersistent()` | Mark as non-persistent (in-memory only) |
| `WithStringAttribute(name, length)` | Add a string attribute |
| `WithIntegerAttribute(name)` | Add an integer attribute |
| `WithLongAttribute(name)` | Add a long attribute |
| `WithDecimalAttribute(name)` | Add a decimal attribute |
| `WithBooleanAttribute(name)` | Add a boolean attribute |
| `WithDateTimeAttribute(name, localize)` | Add a datetime attribute |
| `WithAutoNumberAttribute(name)` | Add an auto-number attribute |
| `WithEnumerationAttribute(name, enumRef)` | Add an enumeration attribute |
| `Build()` | Create the entity and write it to the project |

## AssociationBuilder

Creates associations (relationships) between entities.

```go
assoc, err := modelAPI.DomainModels.CreateAssociation("Customer_Orders").
    From("Customer").
    To("Order").
    OneToMany().
    Build()
```

### AssociationBuilder Methods

| Method | Description |
|--------|-------------|
| `From(entityName)` | Set the FROM entity (FK owner) |
| `To(entityName)` | Set the TO entity (referenced) |
| `OneToMany()` | Reference type with Default owner (1:N) |
| `ManyToMany()` | ReferenceSet type (M:N) |
| `Build()` | Create the association and write it to the project |

## EnumerationBuilder

Creates enumerations with named values and captions.

```go
enum, err := modelAPI.Enumerations.CreateEnumeration("OrderStatus").
    WithValue("Draft", "Draft Order").
    WithValue("Pending", "Pending Approval").
    WithValue("Completed", "Completed").
    WithValue("Cancelled", "Cancelled").
    Build()
```

### EnumerationBuilder Methods

| Method | Description |
|--------|-------------|
| `WithValue(name, caption)` | Add a value with its display caption |
| `Build()` | Create the enumeration and write it to the project |

## MicroflowBuilder

Creates microflows with parameters and return types.

```go
mf, err := modelAPI.Microflows.CreateMicroflow("ACT_ProcessOrder").
    WithParameter("Order", "Sales.Order").
    WithStringParameter("Message").
    ReturnsBoolean().
    Build()
```

### MicroflowBuilder Methods

| Method | Description |
|--------|-------------|
| `WithParameter(name, entityType)` | Add an object parameter |
| `WithStringParameter(name)` | Add a string parameter |
| `ReturnsBoolean()` | Set return type to Boolean |
| `ReturnsString()` | Set return type to String |
| `ReturnsVoid()` | Set return type to Nothing (void) |
| `Build()` | Create the microflow and write it to the project |

## PageBuilder

Creates pages with title and layout configuration.

```go
page, err := modelAPI.Pages.CreatePage("CustomerOverview").
    WithTitle("Customer Overview").
    Build()
```

### PageBuilder Methods

| Method | Description |
|--------|-------------|
| `WithTitle(title)` | Set the page title |
| `Build()` | Create the page and write it to the project |

## Error Handling

All `Build()` methods return `(result, error)`. Common errors include:

- Module context not set (`modelAPI.SetModule()` not called)
- Entity not found (for associations referencing nonexistent entities)
- Duplicate names within the same module
- Invalid attribute types or references

```go
entity, err := modelAPI.DomainModels.CreateEntity("Customer").
    Persistent().
    Build()
if err != nil {
    log.Fatalf("Failed to create entity: %v", err)
}
```
