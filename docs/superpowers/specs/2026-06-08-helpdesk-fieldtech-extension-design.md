# HelpDesk FieldTech Extension — MDL Coverage Design

**Date**: 2026-06-08  
**File**: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`  
**Status**: Design approved, pending implementation

## Goal

Extend `helpdesk-app.mdl` with a `FT` (Field Technician) module that naturally demonstrates all MDL features documented in `.claude/skills/mendix/` but not yet covered in the baseline app. Every new MDL construct must serve a real business purpose — no syntax-only demos.

## Business Story

The HelpDesk app grows to support on-site field service. When a ticket requires a physical visit, an Agent dispatches a FieldTech. FieldTechs work on mobile devices, often offline. On completion they sync their results; the system notifies an external monitoring platform via Business Events and sends an SMS confirmation via REST.

This single narrative arc requires:
- New entities, associations, and delete-cascade behavior (FT domain)
- OQL view entities for SLA + dispatch analytics
- REST client + JSON structures for SMS gateway integration
- Import/Export mappings for external work-order ingestion (JSON webhook)
- Business Events for event-driven integration with external monitoring
- Offline nanoflows with SYNCHRONIZE and JavaScript Action (GPS)
- ALTER PAGE to add dispatch controls to the existing Ticket_Detail page
- REVOKE to tighten early-grant security mistakes
- ALTER SETTINGS MODEL for AfterStartup and HealthCheck microflows

## New Module: FT

### Entities

| Entity | Type | Key Attributes |
|--------|------|----------------|
| `FT.FieldTech` | persistent | Name, Phone, Region, IsAvailable (boolean) |
| `FT.DispatchOrder` | persistent | DispatchedAt, CompletedAt, SiteNotes, Status (enum) |
| `FT.CheckInRecord` | non-persistent | Latitude, Longitude, CheckedInAt — created offline, synced |
| `FT.WorkOrderImport` | non-persistent | RawPayload (string) — JSON webhook buffer |

### Associations

| Association | From → To | Delete Behavior |
|-------------|-----------|-----------------|
| `FT.DispatchOrder_Ticket` | FT.DispatchOrder → HD.Ticket | delete PARENT cascade (ticket deleted → orders deleted) |
| `FT.DispatchOrder_FieldTech` | FT.DispatchOrder → FT.FieldTech | prevent |
| `FT.CheckInRecord_DispatchOrder` | FT.CheckInRecord → FT.DispatchOrder | cascade |

### Indexes

`FT.DispatchOrder`: `index (Status)`, `index (DispatchedAt, CompletedAt)` — multi-column index demo.

### Enumeration

`FT.DispatchStatus`: Pending, EnRoute, OnSite, Completed, Cancelled

## OQL Analytics (VIEW ENTITY)

Two view entities aggregated over existing data:

```mdl
-- SLA compliance rate per agent
CREATE VIEW ENTITY FT.TicketSLAStats AS
  SELECT COUNT(t.ID) AS TotalTickets,
         SUM(CASE WHEN t.IsOverSLA THEN 1 ELSE 0 END) AS OverSLACount
  FROM HD.Ticket t
  LEFT JOIN HD.Agent a ON HD.Ticket_Agent
  GROUP BY a.ID;

-- Dispatch completion summary per FieldTech
CREATE VIEW ENTITY FT.DispatchSummary AS
  SELECT ft.Name, COUNT(d.ID) AS OrderCount,
         avg(DATEPART(HOUR, d.CompletedAt) - DATEPART(HOUR, d.DispatchedAt)) AS AvgHours
  FROM FT.DispatchOrder d
  LEFT JOIN FT.FieldTech ft ON FT.DispatchOrder_FieldTech
  WHERE d.Status = 'Completed'
  GROUP BY ft.ID;
```

Demonstrates: `CREATE VIEW ENTITY`, `LEFT JOIN`, `GROUP BY`, `count()`, `avg()`, `DATEPART()`.

## Microflows

| Microflow | Purpose | New MDL Activities |
|-----------|---------|-------------------|
| `FT.ACT_Dispatch_Assign` | Agent dispatches a FieldTech to a ticket | `CREATE LIST` (collect candidate techs), `SEND REST REQUEST` (SMS notification) |
| `FT.ACT_Dispatch_Complete` | Mark dispatch completed, trigger SMS + Business Event | `SEND REST REQUEST`, `publish business event` |
| `FT.ACT_WorkOrder_Import` | Parse JSON webhook payload into DispatchOrder | `IMPORT MAPPING` activity |
| `FT.ACT_FT_Initialize` | AfterStartup: reset IsAvailable flags | `ALTER SETTINGS MODEL` target |
| `FT.DS_HealthCheck` | HealthCheck: return count of pending orders | `ALTER SETTINGS MODEL` target |
| `FT.DS_MyDispatchOrders` | FieldTech retrieves own orders | `reversed()` XPath, `[%UserRole_FieldTechRole%]` token |

## Nanoflows (Offline)

| Nanoflow | Purpose | New MDL Activities |
|----------|---------|-------------------|
| `FT.NF_FT_CheckIn` | Record GPS location on arrival | `CALL JAVASCRIPT ACTION` (GPS coords), create `CheckInRecord` |
| `FT.NF_FT_CompleteOffline` | Mark dispatch complete while offline | change `DispatchOrder`, `SYNCHRONIZE` |
| `FT.NF_FT_LoadOrders` | Refresh order list when back online | retrieve `DispatchOrder`, `SYNCHRONIZE` |

## REST & JSON Integration

```mdl
-- SMS gateway client
CREATE JSON STRUCTURE FT.SMSPayload
  { "to": "string", "message": "string" };

CREATE REST CLIENT FT.SMSGateway
  base url 'https://sms.example.com/api/v1'
  authentication none;

-- External work-order webhook
CREATE JSON STRUCTURE FT.WorkOrderPayload
  { "ticketRef": "string", "techName": "string", "scheduledAt": "string" };

CREATE IMPORT MAPPING FT.WorkOrderImport_Mapping
  from FT.WorkOrderPayload
  map to FT.DispatchOrder (...);

CREATE EXPORT MAPPING FT.DispatchOrder_Export
  from FT.DispatchOrder
  to FT.WorkOrderPayload (...);
```

Demonstrates: `CREATE REST CLIENT`, `CREATE JSON STRUCTURE`, `CREATE IMPORT MAPPING`, `CREATE EXPORT MAPPING`, `SEND REST REQUEST`.

## Business Events

```mdl
CREATE BUSINESS EVENT SERVICE FT.FieldWorkEvents
  channel 'field-work-events';

message DispatchCompleted (
  DispatchOrderId: integer,
  TicketId:        integer,
  CompletedAt:     string
) publish entity FT.DispatchOrder;

message ExternalWorkOrderReceived (
  TicketRef:   string,
  TechName:    string,
  ScheduledAt: string
) subscribe microflow FT.ACT_WorkOrder_Import;
```

Demonstrates: `CREATE BUSINESS EVENT SERVICE`, `message ... publish entity`, `message ... subscribe microflow`.

## Pages

| Page | Layout | Datasource Pattern | New Widget/Pattern |
|------|--------|-------------------|-------------------|
| `FT.DispatchQueue` | Atlas_Core.Atlas_Default | database from FT.DispatchOrder (FieldTech role filtered) | url slug, page variables (toggle advanced filter) |
| `FT.DispatchOrder_Detail` | Atlas_Core.Atlas_Default | $DispatchOrder param | nested DataView, conditional visibility |
| `FT.SLADashboard` | Atlas_Core.Atlas_Default | VIEW ENTITY FT.TicketSLAStats + FT.DispatchSummary | DataGrid on VIEW ENTITY |

`FT.DispatchQueue` introduces `url: 'dispatch-queue'` (url slug) and `variables: { $ShowAdvanced: boolean = false }` (page variables) — both previously absent from the baseline.

## Security

### New grants
- `FT.FieldTechRole` on `FT.DispatchOrder` — read own orders only via `reversed()` XPath:
  `where '[FT.DispatchOrder_FieldTech/FT.FieldTech[reversed()]/System.owner = ''[%CurrentUser%]'']'`
- `[%UserRole_FieldTechRole%]` token used in navigation visibility expression

### REVOKE (security tightening)
```mdl
-- KB.Reader was granted write on ArticleRating in the baseline; tighten:
revoke write on KB.ArticleRating from KB.Reader;
-- FieldTech cannot execute Agent-only escalation microflows:
revoke execute on microflow HD.ACT_StartEscalation from HD.AgentRole;
grant execute on microflow HD.ACT_StartEscalation to HD.AgentRole, HD.ManagerRole;
```

Demonstrates: `REVOKE` statement.

### User Role
```mdl
create or modify user role FieldTech (System.User, FT.FieldTechRole);
```

## Navigation & Settings

### Offline Native Profile
```mdl
create or replace navigation OfflineNative
  home page FT.DispatchQueue for FieldTech
  menu (
    menu item 'My Orders' page FT.DispatchQueue;
    menu item 'Check In'  nanoflow FT.NF_FT_CheckIn;
  );
```

Demonstrates: offline navigation profile (separate from Responsive).

### ALTER SETTINGS MODEL
```mdl
alter settings model (
  AfterStartupMicroflow:  FT.ACT_FT_Initialize,
  HealthCheckMicroflow:   FT.DS_HealthCheck
);
```

## ALTER PAGE (HD Evolution)

After the FT module is fully defined, retrofit the existing Ticket_Detail page:

```mdl
-- Add "Dispatch" button to Ticket_Detail footer
alter page HD.Ticket_Detail
  insert actionbutton btnDispatch (
    caption: 'Dispatch FieldTech',
    action: show_page FT.DispatchOrder_New (Ticket: $Ticket),
    buttonstyle: default
  ) after btnAssignAgent in dvTicket.ftrActions;

-- Add dispatch history grid below comments
alter page HD.Ticket_Detail
  insert datagrid dgDispatchHistory (
    datasource: $Ticket/FT.DispatchOrder_Ticket/FT.DispatchOrder
  ) {
    column colTech      (attribute: FT.DispatchOrder_FieldTech, caption: 'Technician')
    column colStatus    (attribute: Status,      caption: 'Status')
    column colCompleted (attribute: CompletedAt, caption: 'Completed')
  } after rComments;
```

Demonstrates: `ALTER PAGE ... INSERT ... after`.

## File Layout (append to helpdesk-app.mdl)

New sections in dependency order, estimated line counts:

| MARK Section | Est. Lines |
|-------------|-----------|
| FieldTech — Module & Entities | ~80 |
| FieldTech — OQL Analytics | ~40 |
| FieldTech — Microflows | ~120 |
| FieldTech — Nanoflows (Offline) | ~80 |
| FieldTech — REST & JSON Integration | ~60 |
| FieldTech — Business Events | ~40 |
| FieldTech — Pages | ~80 |
| FieldTech — Security | ~60 |
| FieldTech — Navigation & Settings | ~40 |
| HD — Page Evolution (ALTER PAGE) | ~40 |
| **Total new** | **~640** |
| **Grand total** | **~2915** |

## MDL Coverage Checklist

Features introduced by this extension that were absent from the baseline:

- [x] `CREATE VIEW ENTITY ... AS SELECT ... GROUP BY` (OQL view entity)
- [x] `LEFT JOIN`, `avg()`, `DATEPART()` in OQL
- [x] `delete PARENT cascade` / `delete CHILD cascade` (association delete behavior)
- [x] Multi-column `index (A, B)`
- [x] `REVOKE` (security tightening)
- [x] `reversed()` in XPath constraint
- [x] `[%UserRole_RoleName%]` system token
- [x] `ALTER SETTINGS MODEL` (AfterStartup, HealthCheck)
- [x] `CREATE REST CLIENT` + `SEND REST REQUEST`
- [x] `CREATE JSON STRUCTURE`
- [x] `CREATE IMPORT MAPPING` + `CREATE EXPORT MAPPING`
- [x] `CREATE BUSINESS EVENT SERVICE` + `message ... publish` + `message ... subscribe`
- [x] `SYNCHRONIZE` (offline nanoflow)
- [x] `CALL JAVASCRIPT ACTION`
- [x] Offline navigation profile (`OfflineNative`)
- [x] `url:` slug on page
- [x] `variables:` page-level variables
- [x] `ALTER PAGE ... INSERT ... after` (page evolution)
- [x] `CREATE LIST` microflow activity

## Out of Scope

- `ALTER SNIPPET`, `DEFINE FRAGMENT` — no natural business hook in this story; defer to a separate snippet/fragment demo
- `SHOW WIDGETS` / `UPDATE WIDGETS` — developer tooling, not business logic
- Test annotations (`.test.mdl`) — separate test suite file, not the app baseline
- `DESCRIBE` / `SHOW` query commands — read-only CLI commands, not part of the app script
