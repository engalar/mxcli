# Event Services

An event service is a named container for business event messages. It defines the contract between publishers and consumers: what events exist and what data each event carries.

## CREATE BUSINESS EVENT SERVICE

```sql
CREATE BUSINESS EVENT SERVICE <Module>.<Name> (
  [Version: '<version>',]
  [Description: '<text>']
) {
  MESSAGE <MessageName> (
    <AttributeName>: <Type> [, ...]
  )
  [MESSAGE <MessageName> ( ... )]
};
```

| Element | Description |
|---------|-------------|
| `Version` | Service version string |
| `Description` | Human-readable description of the service |
| `MESSAGE` | A named event type with typed attributes |

### Attribute Types

Message attributes support standard Mendix types:

| Type | Description |
|------|-------------|
| `String` | Text value |
| `Integer` | 32-bit integer |
| `Long` | 64-bit integer |
| `Decimal` | Precise decimal number |
| `Boolean` | True or false |
| `DateTime` | Date and time value |

### Example

```sql
CREATE BUSINESS EVENT SERVICE HR.EmployeeEvents (
  Version: '2.0',
  Description: 'Employee lifecycle events'
) {
  MESSAGE EmployeeHired (
    EmployeeId: String,
    FullName: String,
    Department: String,
    StartDate: DateTime
  )
  MESSAGE EmployeeTerminated (
    EmployeeId: String,
    TerminationDate: DateTime,
    Reason: String
  )
};
```

## DROP BUSINESS EVENT SERVICE

Remove an event service:

```sql
DROP BUSINESS EVENT SERVICE HR.EmployeeEvents;
```

## Listing and Inspecting

```sql
-- List all services
SHOW BUSINESS EVENTS;

-- Filter by module
SHOW BUSINESS EVENTS IN HR;

-- Full MDL definition
DESCRIBE BUSINESS EVENT SERVICE HR.EmployeeEvents;
```

The `DESCRIBE` output is round-trippable -- it can be copied, modified, and executed as a `CREATE` statement.

## See Also

- [Business Events](./business-events.md) -- overview of business events
- [Publishing and Consuming Events](./pub-sub-events.md) -- event publishing and consumption patterns
