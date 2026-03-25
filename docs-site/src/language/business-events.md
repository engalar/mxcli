# Business Events

Business events enable event-driven integration between Mendix applications. An application can publish events when something happens (e.g., an order is placed) and other applications can subscribe to those events and react accordingly.

Business events are organized into **event services**, each of which defines one or more **messages** with typed attributes.

## Inspecting Business Events

```sql
-- List all business event services
SHOW BUSINESS EVENTS;
SHOW BUSINESS EVENTS IN MyModule;

-- View full definition
DESCRIBE BUSINESS EVENT SERVICE MyModule.OrderEvents;
```

## Quick Example

```sql
CREATE BUSINESS EVENT SERVICE Shop.OrderEvents (
  Version: '1.0',
  Description: 'Events related to order lifecycle'
) {
  MESSAGE OrderPlaced (
    OrderId: String,
    CustomerName: String,
    TotalAmount: Decimal,
    OrderDate: DateTime
  )
  MESSAGE OrderShipped (
    OrderId: String,
    TrackingNumber: String,
    ShippedDate: DateTime
  )
};
```

## See Also

- [Event Services](./event-services.md) -- CREATE and DROP syntax for event services
- [Publishing and Consuming Events](./pub-sub-events.md) -- event publishing and consumption patterns
