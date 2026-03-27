# Proposal: Integration Pane — Unified View of External Service Assets

## Overview

Mendix Studio Pro has an **Integration Pane** that shows all connected services and lists the available assets from their contracts (OData `$metadata`, OpenAPI specs, AsyncAPI definitions, external database schemas). We need an equivalent in mxcli — both as **MDL commands** and **catalog tables** — to give users a unified way to discover what integrations exist and what assets they expose.

### Service Types

| Service Type | Contract Format | Assets |
|---|---|---|
| **OData Client** | `$metadata` (EDMX/CSDL) | External entities, external actions/functions |
| **REST Client** | OpenAPI / manual operations | REST operations (resources, methods, parameters) |
| **Published OData** | Self-defined | Published entity sets, exposed members |
| **Published REST** | Self-defined | Resources, operations |
| **Business Events** | AsyncAPI | Channels, messages, attributes |
| **Database Connection** | DB schema | Queries, table mappings, parameters |
| **SOAP Client** (consumed) | WSDL | Imported operations, parameters |
| **SOAP Service** (published) | WSDL | Published operations, parameters, microflow handlers |

### Two Levels of Discovery

1. **MPR-stored assets** (Phase 1) — assets that are already imported/configured and persisted in the MPR file. This is what we can show without network access.
2. **Contract-fetched assets** (Phase 2, future) — fetching `$metadata` or OpenAPI specs from remote URLs to show what's *available but not yet imported*. This requires network access and is out of scope for Phase 1.

---

## Current State

### SHOW/DESCRIBE Commands

| Service Type | SHOW | DESCRIBE | Catalog Table |
|---|---|---|---|
| OData Client | `SHOW ODATA CLIENTS` | `DESCRIBE ODATA CLIENT` | `odata_clients` |
| OData Service (published) | `SHOW ODATA SERVICES` | `DESCRIBE ODATA SERVICE` | `odata_services` |
| External Entity | `SHOW EXTERNAL ENTITIES` | `DESCRIBE EXTERNAL ENTITY` | — (in `entities` table) |
| External Action | — | — | — |
| SOAP Client (consumed) | — | — | — |
| SOAP Service (published) | — | — | — |
| REST Client | `SHOW REST CLIENTS` | `DESCRIBE REST CLIENT` | — |
| Published REST | — | — | — |
| Business Event Service | `SHOW BUSINESS EVENT SERVICES` | `DESCRIBE BUSINESS EVENT SERVICE` | `business_event_services` |
| Business Event Messages | `SHOW BUSINESS EVENTS` | — | — |
| Business Event Client | `SHOW BUSINESS EVENT CLIENTS` (stub) | — | — |
| Database Connection | `SHOW DATABASE CONNECTIONS` | `DESCRIBE DATABASE CONNECTION` | `database_connections` |

### Catalog Gaps

| Gap | Details |
|---|---|
| No `rest_clients` table | Consumed REST services not queryable via SQL |
| No `rest_operations` table | Individual operations not queryable |
| No `published_rest_services` table | Published REST services not queryable |
| No `published_rest_operations` table | Published REST operations not queryable |
| No `external_entities` table | External entities mixed into `entities` table with no easy filter |
| No `external_actions` table | No way to discover OData action usage |
| No `business_events` table | Individual messages not queryable |
| `CallExternalAction` not in activity type switch | Catalog builder doesn't recognize this activity type |
| `activities` table missing service/action columns | No OData service or action name stored per activity |
| No SOAP parsing at all | `WebServices$ImportedWebService` and `WebServices$PublishedWebService` BSON types exist in metamodel but no parser, reader, model types, or commands |
| No `soap_clients` / `soap_services` tables | SOAP services not queryable |

---

## Phase 1: MPR-Stored Assets

### 1.1 New MDL Commands

#### OData External Actions (from microflow usage)
```sql
SHOW EXTERNAL ACTIONS;                    -- All external actions used in microflows
SHOW EXTERNAL ACTIONS IN MyModule;        -- Filter by module
```

Output columns: `Service`, `Action`, `Parameters`, `UsedBy` (microflow names)

> **Note:** OData action definitions from `$metadata` are NOT stored in the MPR. We can only show actions that are actually *used* in microflows. Phase 2 would add `$metadata` fetching.

#### Published REST Services
```sql
SHOW PUBLISHED REST SERVICES;             -- All published REST services
SHOW PUBLISHED REST SERVICES IN MyModule; -- Filter by module
DESCRIBE PUBLISHED REST SERVICE MyModule.CustomerAPI;
```

Output columns: `Module`, `QualifiedName`, `Path`, `Version`, `Resources`, `Operations`

### 1.2 New Catalog Tables

#### `rest_clients` — Consumed REST Services
```sql
CREATE TABLE IF NOT EXISTS rest_clients (
    Id TEXT PRIMARY KEY,
    Name TEXT,
    QualifiedName TEXT,
    ModuleName TEXT,
    Folder TEXT,
    BaseUrl TEXT,
    AuthScheme TEXT,           -- "Basic", "None"
    OperationCount INTEGER,
    Documentation TEXT,
    ProjectId TEXT,
    ProjectName TEXT,
    SnapshotId TEXT,
    SnapshotDate TEXT,
    SnapshotSource TEXT
);
```

#### `rest_operations` — REST Client Operations (detail table)
```sql
CREATE TABLE IF NOT EXISTS rest_operations (
    Id TEXT PRIMARY KEY,
    ServiceId TEXT,             -- FK to rest_clients.Id
    ServiceQualifiedName TEXT,
    Name TEXT,
    HttpMethod TEXT,            -- GET, POST, PUT, PATCH, DELETE
    Path TEXT,
    ParameterCount INTEGER,
    HasBody INTEGER,
    ResponseType TEXT,          -- "ImplicitMapping", "NoResponse"
    Timeout INTEGER,
    ModuleName TEXT,
    ProjectId TEXT,
    SnapshotId TEXT,
    SnapshotDate TEXT,
    SnapshotSource TEXT
);
```

#### `published_rest_services` — Published REST Services
```sql
CREATE TABLE IF NOT EXISTS published_rest_services (
    Id TEXT PRIMARY KEY,
    Name TEXT,
    QualifiedName TEXT,
    ModuleName TEXT,
    Folder TEXT,
    Path TEXT,
    Version TEXT,
    ServiceName TEXT,
    ResourceCount INTEGER,
    OperationCount INTEGER,
    Documentation TEXT,
    ProjectId TEXT,
    ProjectName TEXT,
    SnapshotId TEXT,
    SnapshotDate TEXT,
    SnapshotSource TEXT
);
```

#### `published_rest_operations` — Published REST Operations (detail table)
```sql
CREATE TABLE IF NOT EXISTS published_rest_operations (
    Id TEXT PRIMARY KEY,
    ServiceId TEXT,             -- FK to published_rest_services.Id
    ServiceQualifiedName TEXT,
    ResourceName TEXT,
    HttpMethod TEXT,
    Path TEXT,
    Summary TEXT,
    Microflow TEXT,             -- Implementation microflow (BY_NAME)
    Deprecated INTEGER,
    ModuleName TEXT,
    ProjectId TEXT,
    SnapshotId TEXT,
    SnapshotDate TEXT,
    SnapshotSource TEXT
);
```

#### `external_entities` — OData External Entities (view on entities)
```sql
CREATE TABLE IF NOT EXISTS external_entities (
    Id TEXT PRIMARY KEY,
    Name TEXT,
    QualifiedName TEXT,
    ModuleName TEXT,
    ServiceName TEXT,           -- Consumed OData service qualified name
    EntitySet TEXT,             -- Remote entity set name
    RemoteName TEXT,            -- Remote entity name
    Countable INTEGER,
    Creatable INTEGER,
    Deletable INTEGER,
    Updatable INTEGER,
    AttributeCount INTEGER,
    ProjectId TEXT,
    ProjectName TEXT,
    SnapshotId TEXT,
    SnapshotDate TEXT,
    SnapshotSource TEXT
);
```

#### `external_actions` — OData External Actions (from microflow usage)
```sql
CREATE TABLE IF NOT EXISTS external_actions (
    Id TEXT PRIMARY KEY,        -- Synthetic: hash of service+action
    ServiceName TEXT,           -- Consumed OData service qualified name
    ActionName TEXT,
    ModuleName TEXT,            -- Module where the calling microflow lives
    UsageCount INTEGER,         -- Number of microflow activities calling this
    CallerNames TEXT,           -- Comma-separated list of calling microflows
    ParameterNames TEXT,        -- Comma-separated parameter names
    ProjectId TEXT,
    ProjectName TEXT,
    SnapshotId TEXT,
    SnapshotDate TEXT,
    SnapshotSource TEXT
);
```

#### `business_events` — Individual Business Event Messages (detail table)
```sql
CREATE TABLE IF NOT EXISTS business_events (
    Id TEXT PRIMARY KEY,
    ServiceId TEXT,             -- FK to business_event_services.Id
    ServiceQualifiedName TEXT,
    ChannelName TEXT,
    MessageName TEXT,
    CanPublish INTEGER,
    CanSubscribe INTEGER,
    AttributeCount INTEGER,
    Entity TEXT,                -- Associated entity (BY_NAME)
    PublishMicroflow TEXT,      -- Publisher microflow (BY_NAME)
    SubscribeMicroflow TEXT,    -- Subscriber microflow (BY_NAME)
    ModuleName TEXT,
    ProjectId TEXT,
    SnapshotId TEXT,
    SnapshotDate TEXT,
    SnapshotSource TEXT
);
```

### 1.3 Fix Existing Catalog Gaps

1. **Add `CallExternalAction` to `getMicroflowActionType()`** in `mdl/catalog/builder_microflows.go`
2. **Add service/action columns to `activities` table** — `ServiceRef TEXT`, `ActionRef TEXT` for CallExternalAction, CallRestAction, etc.
3. **Populate `external_entities` table** during catalog refresh from domain models with `Source = "Rest$ODataRemoteEntitySource"`
4. **Add `IsExternal` column to `entities` table** for easy filtering

### 1.4 OBJECTS View Integration

Add all new tables to the `CATALOG.OBJECTS` UNION ALL view so `SHOW CATALOG TABLES` and `SELECT * FROM CATALOG.OBJECTS` include them:

```sql
UNION ALL SELECT Id, 'RestClient' AS ObjectType, Name, QualifiedName, ModuleName, ... FROM rest_clients
UNION ALL SELECT Id, 'RestOperation' AS ObjectType, Name, ... FROM rest_operations
UNION ALL SELECT Id, 'PublishedRestService' AS ObjectType, Name, ... FROM published_rest_services
UNION ALL SELECT Id, 'ExternalEntity' AS ObjectType, Name, ... FROM external_entities
UNION ALL SELECT Id, 'ExternalAction' AS ObjectType, ActionName AS Name, ... FROM external_actions
UNION ALL SELECT Id, 'BusinessEvent' AS ObjectType, MessageName AS Name, ... FROM business_events
```

---

## Phase 1 Implementation Plan

### Step 1: Catalog tables and builders (foundation)

- [ ] Add table DDL for `rest_clients`, `rest_operations`, `published_rest_services`, `published_rest_operations`, `external_entities`, `external_actions`, `business_events` to `mdl/catalog/tables.go`
- [ ] Add `IsExternal` column to existing `entities` table
- [ ] Add `ServiceRef`, `ActionRef` columns to existing `activities` table
- [ ] Add `CallExternalAction` case to `getMicroflowActionType()` in `builder_microflows.go`
- [ ] Create `builder_rest.go` — populate `rest_clients` and `rest_operations`
- [ ] Create `builder_published_rest.go` — populate `published_rest_services` and `published_rest_operations`
- [ ] Create `builder_external_entities.go` — populate `external_entities` from domain models
- [ ] Create `builder_external_actions.go` — populate `external_actions` by scanning microflows
- [ ] Extend `builder_businessevents.go` — populate `business_events` detail table
- [ ] Update OBJECTS view with new table UNIONs
- [ ] Wire builders into `builder.go` refresh pipeline

### Step 2: SHOW EXTERNAL ACTIONS command

- [ ] Add `ShowExternalActions` to AST `ShowObjectType` enum
- [ ] Add grammar rule `SHOW EXTERNAL ACTIONS [IN module]` to `MDLParser.g4`
- [ ] Add visitor handler in `visitor_query.go`
- [ ] Implement `showExternalActions()` in `cmd_odata.go` (scan microflows for CallExternalAction)
- [ ] Add executor dispatch in `executor.go`
- [ ] Regenerate ANTLR parser

### Step 3: SHOW/DESCRIBE PUBLISHED REST SERVICES

- [ ] Add `ShowPublishedRestServices` to AST `ShowObjectType` enum
- [ ] Add `DescribePublishedRestService` to AST `DescribeObjectType` enum
- [ ] Add grammar rules to `MDLParser.g4`
- [ ] Add visitor handlers in `visitor_query.go`
- [ ] Implement `showPublishedRestServices()` in new `cmd_published_rest.go`
- [ ] Implement `describePublishedRestService()` with MDL output format
- [ ] Add executor dispatch in `executor.go`
- [ ] Regenerate ANTLR parser

### Step 4: Help text and examples

- [ ] Update `cmd/mxcli/help_topics/odata.txt` with SHOW EXTERNAL ACTIONS
- [ ] Create `cmd/mxcli/help_topics/rest.txt` or update existing with published REST
- [ ] Add examples to `mdl-examples/doctype-tests/`
- [ ] Update CLAUDE.md Current Implementation Status section

### Step 5: Tests

- [ ] Add catalog builder tests for new tables
- [ ] Add visitor parsing tests for new grammar rules
- [ ] Verify `make test` passes

---

## Phase 2: Contract Fetching (Future)

Phase 2 would add the ability to fetch remote contracts and show *available but not yet imported* assets:

### OData `$metadata` Fetching
```sql
-- Fetch and parse the $metadata from the service URL
REFRESH ODATA CLIENT MyModule.SalesforceAPI;

-- Show all available entities/actions from the contract (including not-yet-imported)
SHOW AVAILABLE ENTITIES FROM MyModule.SalesforceAPI;
SHOW AVAILABLE ACTIONS FROM MyModule.SalesforceAPI;
```

### OpenAPI Fetching
```sql
-- If ConsumedRestService has an OpenApiFile, parse it
SHOW AVAILABLE OPERATIONS FROM MyModule.ExternalAPI;
```

### Database Schema Discovery
```sql
-- Already partially implemented via SQL CONNECT + SQL <alias> SHOW TABLES
SQL mydb SHOW TABLES;
SQL mydb DESCRIBE TABLE orders;
```

Phase 2 is out of scope for this PR.

---

## Phase 3: SOAP Web Services (Future)

Mendix supports both consuming and publishing SOAP web services. The BSON types are fully defined in the metamodel (`WebServices$ImportedWebService`, `WebServices$PublishedWebService`) but **no parser, reader, model types, or commands exist yet**. This requires building the full stack from scratch.

### BSON Types Available

| Type | Description |
|---|---|
| `WebServices$ImportedWebService` | Consumed SOAP service — holds WSDL, operations, port/binding info |
| `WebServices$PublishedWebService` | Published SOAP service — versioned services with published operations |
| `WebServices$WsdlDescription` | WSDL metadata (schema entries, target namespace) |
| `WebServices$ServiceInfo` | SOAP service definition (location, port, SOAP version, operations) |
| `WebServices$OperationInfo` | Individual SOAP operation (request/response body, SOAP action) |
| `WebServices$PublishedOperation` | Published operation (microflow handler, parameters, return type) |
| `WebServices$VersionedService` | Versioned published service (caption, authentication, validation) |
| `Microflows$WebServiceCallAction` | Microflow activity that calls a consumed SOAP operation |

### Implementation Steps

1. **Model types** — Add `ImportedWebService`, `PublishedWebService` to `model/types.go`
2. **Parser** — Create `sdk/mpr/parser_webservices.go` for BSON deserialization
3. **Reader** — Add `ListImportedWebServices()`, `ListPublishedWebServices()` to reader
4. **SHOW/DESCRIBE** — `SHOW SOAP CLIENTS`, `SHOW SOAP SERVICES`, `DESCRIBE SOAP CLIENT`, `DESCRIBE SOAP SERVICE`
5. **Catalog tables** — `soap_clients` (with operation count, WSDL URL, SOAP version), `soap_services` (with versioned service info), `soap_operations` (detail table)
6. **Microflow integration** — Add `WebServiceCallAction` to `getMicroflowActionType()` and populate `ServiceRef`/`ActionRef`

### Target MDL Commands
```sql
SHOW SOAP CLIENTS;                           -- List consumed SOAP services
SHOW SOAP CLIENTS IN MyModule;
SHOW SOAP SERVICES;                          -- List published SOAP services
DESCRIBE SOAP CLIENT MyModule.WeatherService;
DESCRIBE SOAP SERVICE MyModule.OrderService;

-- Catalog queries
SELECT * FROM CATALOG.SOAP_CLIENTS;
SELECT * FROM CATALOG.SOAP_SERVICES;
SELECT * FROM CATALOG.SOAP_OPERATIONS;
```

---

## Example Queries After Phase 1

```sql
-- Integration overview: all external services
SELECT ObjectType, QualifiedName, ModuleName
FROM CATALOG.OBJECTS
WHERE ObjectType IN ('ODataClient', 'RestClient', 'PublishedODataService',
                     'PublishedRestService', 'BusinessEventService', 'DatabaseConnection',
                     'SoapClient', 'SoapService');  -- Phase 3

-- All external entities and their source services
SELECT QualifiedName, ServiceName, EntitySet, RemoteName
FROM CATALOG.EXTERNAL_ENTITIES;

-- All external actions and where they're called from
SELECT ServiceName, ActionName, UsageCount, CallerNames
FROM CATALOG.EXTERNAL_ACTIONS;

-- All REST operations across all consumed services
SELECT ServiceQualifiedName, HttpMethod, Path, Name
FROM CATALOG.REST_OPERATIONS
ORDER BY ServiceQualifiedName, Path;

-- Published API surface area
SELECT ServiceQualifiedName, HttpMethod, Path, Microflow
FROM CATALOG.PUBLISHED_REST_OPERATIONS;

-- Business event messages with their handlers
SELECT ServiceQualifiedName, MessageName, CanPublish, CanSubscribe, Entity
FROM CATALOG.BUSINESS_EVENTS;

-- Cross-cutting: find all integration touchpoints in a module
SELECT 'OData Client' AS Type, QualifiedName FROM CATALOG.ODATA_CLIENTS WHERE ModuleName = 'Integration'
UNION ALL
SELECT 'REST Client', QualifiedName FROM CATALOG.REST_CLIENTS WHERE ModuleName = 'Integration'
UNION ALL
SELECT 'External Entity', QualifiedName FROM CATALOG.EXTERNAL_ENTITIES WHERE ModuleName = 'Integration'
UNION ALL
SELECT 'Business Event', ServiceQualifiedName || '.' || MessageName FROM CATALOG.BUSINESS_EVENTS WHERE ModuleName = 'Integration';
```
