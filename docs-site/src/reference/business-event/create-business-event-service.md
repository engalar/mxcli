# CREATE BUSINESS EVENT SERVICE

## Synopsis

```sql
CREATE [ OR MODIFY ] BUSINESS EVENT SERVICE module.Name
(
    ServiceName: 'service_name',
    EventNamePrefix: 'prefix'
)
{
    MESSAGE MessageName ( attr: Type [, ...] ) PUBLISH | SUBSCRIBE
        [ ENTITY module.Entity ] ;
    [ ... ]
}
```

## Description

Creates a business event service that defines one or more messages for event-driven communication between applications.

Each message has a name, a set of typed attributes, and an operation mode (PUBLISH or SUBSCRIBE). A publishing message means this application sends events of that type. A subscribing message means this application receives and handles events of that type.

The `OR MODIFY` option updates the existing service in-place, preserving its UUID so that references from other documents remain valid. `OR REPLACE` is also accepted as a synonym.

### Service Properties

The service declaration includes two required properties:

- **ServiceName** -- The logical service name used for event routing on the Mendix Business Events broker.
- **EventNamePrefix** -- A prefix prepended to message names to form the fully qualified event name. Can be empty (`''`).

### Message Attributes

Each message defines zero or more attributes with the following supported types:

| Type | Description |
|------|-------------|
| `String` | Text value |
| `Integer` | 32-bit integer |
| `Long` | 64-bit integer |
| `Boolean` | True/false |
| `DateTime` | Date and time |
| `Decimal` | Arbitrary-precision decimal |

### Entity Mapping

The optional `ENTITY` clause on a message links the message to a Mendix entity. For SUBSCRIBE messages, incoming events are mapped to instances of this entity. For PUBLISH messages, entity instances are serialized into outgoing events.

## Parameters

`module.Name`
:   Qualified name of the business event service (`Module.ServiceName`).

`ServiceName: 'service_name'`
:   The logical service identifier used by the Business Events broker.

`EventNamePrefix: 'prefix'`
:   A prefix for event names. Use an empty string (`''`) for no prefix.

`MESSAGE MessageName`
:   The name of a message within the service.

`attr: Type`
:   An attribute definition within a message. Multiple attributes are comma-separated.

`PUBLISH`
:   This application publishes (sends) this message type.

`SUBSCRIBE`
:   This application subscribes to (receives) this message type.

`ENTITY module.Entity`
:   Optional entity linked to the message for data mapping.

## Examples

Create a service that publishes order events:

```sql
CREATE BUSINESS EVENT SERVICE Shop.OrderEvents
(
    ServiceName: 'com.example.shop.orders',
    EventNamePrefix: 'shop'
)
{
    MESSAGE OrderCreated (OrderId: Long, CustomerName: String, Total: Decimal) PUBLISH
        ENTITY Shop.Order;
    MESSAGE OrderShipped (OrderId: Long, TrackingNumber: String) PUBLISH
        ENTITY Shop.Shipment;
};
```

Create a service that subscribes to external events:

```sql
CREATE BUSINESS EVENT SERVICE Inventory.StockUpdates
(
    ServiceName: 'com.example.warehouse.stock',
    EventNamePrefix: ''
)
{
    MESSAGE StockChanged (ProductId: Long, NewQuantity: Integer) SUBSCRIBE
        ENTITY Inventory.StockLevel;
};
```

Replace an existing service:

```sql
CREATE OR REPLACE BUSINESS EVENT SERVICE Shop.OrderEvents
(
    ServiceName: 'com.example.shop.orders',
    EventNamePrefix: 'shop'
)
{
    MESSAGE OrderCreated (OrderId: Long, CustomerName: String, Total: Decimal) PUBLISH;
    MESSAGE OrderCancelled (OrderId: Long, Reason: String) PUBLISH;
};
```

## See Also

[DROP BUSINESS EVENT SERVICE](drop-business-event-service.md)
