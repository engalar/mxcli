# CREATE CONSUMED MCP SERVICE

## Synopsis

```sql
CREATE [ OR MODIFY ] CONSUMED MCP SERVICE module.Name (
    ProtocolVersion: v2025_03_26,
    version: '1.0'
    [, ConnectionTimeoutSeconds: 30 ]
    [, documentation: 'description' ]
);

DROP CONSUMED MCP SERVICE module.Name
```

Requires Mendix 11.9+.

## Description

Creates a Consumed MCP Service document. A consumed MCP service represents a remote tool server that implements the Model Context Protocol (MCP). Once registered, agents can use the tools provided by the MCP server.

Consumed MCP services are referenced inside `CREATE AGENT` body blocks using the `MCP SERVICE` keyword.

If `OR MODIFY` is specified and the service already exists, its properties are updated in place. The document UUID is preserved.

## Parameters

`module.Name`
:   The qualified name of the consumed MCP service document.

`ProtocolVersion: v2025_03_26`
:   The MCP protocol version. Use the token form (e.g. `v2025_03_26`), not a quoted string.

`version: '1.0'`
:   The service version as a quoted string.

`ConnectionTimeoutSeconds: 30`
:   Optional. Connection timeout in seconds. Defaults to 30.

`documentation: 'description'`
:   Optional. A human-readable description of the MCP service.

## Examples

### Register an MCP service

```sql
CREATE CONSUMED MCP SERVICE MyModule."WebSearch" (
    ProtocolVersion: v2025_03_26,
    version: '1.0',
    ConnectionTimeoutSeconds: 30,
    documentation: 'Web search MCP server for research tasks'
);
/
```

### Idempotent upsert

```sql
CREATE OR MODIFY CONSUMED MCP SERVICE MyModule."WebSearch" (
    ProtocolVersion: v2025_03_26,
    version: '1.1',
    ConnectionTimeoutSeconds: 60,
    documentation: 'Updated web search MCP service'
);
/
```

## See Also

[CREATE AGENT](create-agent.md), [CREATE MODEL](create-model.md)
