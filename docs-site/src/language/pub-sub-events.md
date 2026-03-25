# Publishing and Consuming Events

Business events follow a publish-subscribe pattern. One application publishes events when significant actions occur, and other applications consume those events to trigger their own logic.

## Publishing Events

A publishing application defines a business event service with its message types:

```sql
CREATE BUSINESS EVENT SERVICE Shop.OrderEvents (
  Version: '1.0',
  Description: 'Order lifecycle events'
) {
  MESSAGE OrderPlaced (
    OrderId: String,
    CustomerName: String,
    TotalAmount: Decimal,
    OrderDate: DateTime
  )
  MESSAGE OrderCancelled (
    OrderId: String,
    Reason: String,
    CancelledDate: DateTime
  )
};
```

The event service acts as a contract. Consuming applications rely on the message structure, so changes to attribute names or types should be accompanied by a version increment.

## Consuming Events

A consuming application subscribes to events from another application's event service. When an event is received, a configured microflow is triggered to handle it.

The consumption side is typically configured in Mendix Studio Pro by selecting the published event service and mapping each message to a handler microflow. The handler microflow receives the event attributes as parameters.

### Handler Pattern

A typical handler microflow processes the incoming event data:

```sql
CREATE MICROFLOW Warehouse.ACT_HandleOrderPlaced
BEGIN
  -- Parameters: $OrderId (String), $CustomerName (String), etc.
  DECLARE $ShipmentRequest Warehouse.ShipmentRequest;
  $ShipmentRequest = CREATE Warehouse.ShipmentRequest (
    ExternalOrderId = $OrderId,
    CustomerName = $CustomerName,
    Status = 'Pending'
  );
  COMMIT $ShipmentRequest;
END;
```

## Versioning

Event services include a version string to track contract changes:

```sql
CREATE BUSINESS EVENT SERVICE Shop.OrderEvents (
  Version: '2.0',
  Description: 'Order events - v2 adds ShippingAddress'
) {
  MESSAGE OrderPlaced (
    OrderId: String,
    CustomerName: String,
    TotalAmount: Decimal,
    OrderDate: DateTime,
    ShippingAddress: String
  )
};
```

When adding new attributes, increment the version so consuming applications know to update their handlers.

## Use Cases

| Scenario | Publisher | Consumer |
|----------|-----------|----------|
| Order fulfillment | Shop app publishes `OrderPlaced` | Warehouse app creates shipment request |
| Customer sync | CRM app publishes `CustomerUpdated` | Billing app updates customer record |
| Compliance | HR app publishes `EmployeeTerminated` | IT app revokes access |
| Notifications | Any app publishes events | Notification service sends emails/SMS |

## See Also

- [Business Events](./business-events.md) -- overview of business events
- [Event Services](./event-services.md) -- CREATE and DROP syntax
