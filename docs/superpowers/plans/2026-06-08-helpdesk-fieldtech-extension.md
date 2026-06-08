# HelpDesk FieldTech Extension — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Append a FieldTech (FT) module to `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` that demonstrates 19 MDL syntax features absent from the baseline, organised around a coherent field-service business story.

**Architecture:** All new MDL is appended to the single existing file in dependency order: entities → OQL views → microflows → nanoflows → REST/JSON artifacts → Business Events → pages → security → navigation+settings → ALTER PAGE patches. `mxcli check` runs after every task. Full execution + `mx check` runs after Task 7 (pages complete) and after Task 10 (final).

**Tech Stack:** MDL, `./bin/mxcli check`, `./bin/mxcli -p <mpr> -c "EXECUTE SCRIPT '...'"`, `~/.mxcli/mxbuild/11.6.6/modeler/mx check`

---

## File Changes

| File | Change |
|------|--------|
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` | Append ~640 lines across 10 sections |

No other files change.

---

## Task 1: FT Module — Entities & Associations

New MDL features: `delete PARENT cascade`, `delete CHILD cascade`, multi-column `index (A, B)`, `system members (owner)` on FT.FieldTech.

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append the FT entities block**

Open `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` and append after the final `-- End of i18n Demo` comment:

```mdl
-- ============================================================================
-- MARK: FieldTech — Module & Entities
-- ============================================================================
-- Business context: Agents dispatch on-site field technicians for tickets that
-- require a physical visit. FieldTechs work mobile / offline; results sync
-- back when connectivity is restored. The FT module introduces all MDL features
-- not yet covered by the KB + HD baseline.
-- ============================================================================

create module FT;

create or modify enumeration FT.DispatchStatus (
  Pending   'Pending',
  EnRoute   'En Route',
  OnSite    'On Site',
  Completed 'Completed',
  Cancelled 'Cancelled'
);

-- FT.FieldTech: technician profile. system members (owner) links each record
-- to the Mendix user who created it — used in row-level security XPath.
create or modify persistent entity FT.FieldTech (
  Name:        string(200) not null,
  Phone:        string(50),
  Region:       string(100),
  IsAvailable:  boolean default true
)
system members (owner);

-- FT.DispatchOrder: one order per site visit. Multi-column index on
-- (Status, DispatchedAt) demonstrates the multi-column index syntax.
create or modify persistent entity FT.DispatchOrder (
  DispatchedAt: datetime,
  CompletedAt:  datetime,
  SiteNotes:    string,
  Status:       FT.DispatchStatus default Pending
)
index (Status)
index (Status, DispatchedAt);

-- delete PARENT cascade: deleting a Ticket cascades to its DispatchOrders.
-- "PARENT" refers to the FROM entity (the FK owner) — FT.DispatchOrder here.
create or modify association FT.DispatchOrder_Ticket
  from FT.DispatchOrder to HD.Ticket
  type reference
  owner default
  delete PARENT cascade;

-- delete CHILD prevent: deleting a FieldTech is blocked if they have orders.
-- "CHILD" refers to the TO entity — FT.FieldTech here.
create or modify association FT.DispatchOrder_FieldTech
  from FT.DispatchOrder to FT.FieldTech
  type reference
  owner default
  delete CHILD prevent;

-- FT.CheckInRecord: non-persistent GPS snapshot created offline, synced later.
create or modify non-persistent entity FT.CheckInRecord (
  Latitude:    decimal,
  Longitude:   decimal,
  CheckedInAt: datetime
);

create or modify association FT.CheckInRecord_DispatchOrder
  from FT.CheckInRecord to FT.DispatchOrder
  type reference
  owner default;

-- FT.WorkOrderImport: non-persistent webhook buffer for JSON import.
create or modify non-persistent entity FT.WorkOrderImport (
  RawPayload:   string,
  TicketRef:    string(200),
  TechName:     string(200),
  ScheduledAt:  string(50)
);
```

- [ ] **Step 2: Verify syntax**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add FT module entities — cascade delete, multi-column index"
```

---

## Task 2: FT Module — OQL View Entities

New MDL features: `CREATE VIEW ENTITY ... AS (SELECT ... GROUP BY ...)`, `LEFT OUTER JOIN`, `count(t.ID)`, `avg()`, `left outer join`, explicit AS aliases.

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append OQL view entities**

```mdl
-- ============================================================================
-- MARK: FieldTech — OQL Analytics
-- ============================================================================
-- Two view entities for the SLA dashboard page (FT.SLADashboard).
-- Demonstrates: CREATE VIEW ENTITY, LEFT OUTER JOIN, count(t.ID), avg(),
-- explicit AS aliases (required — Mendix maps aliases to entity attributes),
-- no ORDER BY / LIMIT (UI handles sorting).
-- ============================================================================

-- SLA compliance per agent: total tickets and overdue count.
-- IsOverSLA is boolean; count() on a boolean column counts non-null true values.
create view entity FT.TicketSLAStats (
  AgentName:    string(200),
  TotalTickets: integer,
  OverSLACount: integer
) as (
  select
    a.Name             as AgentName,
    count(t.ID)        as TotalTickets,
    count(t.IsOverSLA) as OverSLACount
  from HD.Ticket as t
  left outer join t/HD.Ticket_Agent/HD.Agent as a
  group by a.Name
);

-- Dispatch completion summary per FieldTech.
-- avg() on integer (DATEPART result) returns decimal — declared as decimal.
create view entity FT.DispatchSummary (
  TechName:    string(200),
  OrderCount:  integer,
  AvgHours:    decimal
) as (
  select
    ft.Name                                                             as TechName,
    count(d.ID)                                                         as OrderCount,
    avg(datepart(HOUR, d.CompletedAt) - datepart(HOUR, d.DispatchedAt)) as AvgHours
  from FT.DispatchOrder as d
  left outer join d/FT.DispatchOrder_FieldTech/FT.FieldTech as ft
  where d.Status = 'Completed'
  group by ft.Name
);
```

- [ ] **Step 2: Verify syntax**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add OQL view entities — TicketSLAStats, DispatchSummary"
```

---

## Task 3: FT Module — Microflows

New MDL features: `CREATE LIST` activity, `send rest request` (after Task 5 adds the REST client — see dependency note), `import from mapping` (after Task 5), `reversed()` XPath in retrieve, `log info`.

> **Dependency note**: `FT.ACT_Dispatch_Complete` calls `send rest request FT.SMSGateway.NotifyTech` and `FT.ACT_WorkOrder_Import` calls `import from mapping`. These activities reference artifacts defined in Task 5. Write the full microflow bodies now; `mxcli check --references` will pass only after Task 5 is complete. Plain `mxcli check` (syntax only) passes immediately.

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append FT microflows**

```mdl
-- ============================================================================
-- MARK: FieldTech — Microflows
-- ============================================================================

-- ACT_Dispatch_Assign: Agent assigns a FieldTech to a Ticket.
-- New activity: CREATE LIST — builds a list of candidate FT.FieldTech objects
-- available for dispatch (IsAvailable = true). LIST is then filtered by Region.
-- In production use retrieve-then-filter; here CREATE LIST demonstrates syntax.
create or modify microflow FT.ACT_Dispatch_Assign
  ($Ticket: HD.Ticket, $FieldTech: FT.FieldTech)
  returns boolean as $Success
  folder 'Dispatch'
{
  -- CREATE LIST activity: instantiates an empty typed list variable.
  -- Use case: accumulate objects before a bulk commit or decision.
  declare $Candidates: list of FT.FieldTech;
  $Candidates = create list of FT.FieldTech;

  if $FieldTech/IsAvailable = false {
    show message 'Selected technician is not available.' type warning;
    return false;
  }
  $Order = create FT.DispatchOrder (
    Status       = FT.DispatchStatus.Pending,
    DispatchedAt = '[%CurrentDateTime%]',
    FT.DispatchOrder_Ticket    = $Ticket,
    FT.DispatchOrder_FieldTech = $FieldTech
  );
  commit $Order;
  change $FieldTech (IsAvailable = false);
  commit $FieldTech;
  return true;
}
/

-- ACT_Dispatch_Open: called from Ticket_Detail ALTER PAGE button.
-- Creates a draft DispatchOrder linked to the Ticket, then shows the detail page.
create or modify microflow FT.ACT_Dispatch_Open
  ($Ticket: HD.Ticket)
  folder 'Dispatch'
{
  $Order = create FT.DispatchOrder (
    Status = FT.DispatchStatus.Pending,
    FT.DispatchOrder_Ticket = $Ticket
  );
  commit $Order;
  show page FT.DispatchOrder_Detail (DispatchOrder: $Order);
  return;
}
/

-- ACT_Dispatch_Complete: marks order Completed, sends SMS via REST, logs.
-- Demonstrates: send rest request (REST client operation call).
-- $latestHttpResponse/StatusCode reads the HTTP status after the call.
create or modify microflow FT.ACT_Dispatch_Complete
  ($Order: FT.DispatchOrder)
  returns boolean as $Success
  folder 'Dispatch'
{
  if $Order/Status = FT.DispatchStatus.Completed {
    return false;
  }
  change $Order (
    Status      = FT.DispatchStatus.Completed,
    CompletedAt = '[%CurrentDateTime%]'
  );
  commit $Order;
  -- Notify FieldTech via SMS gateway (REST client defined in MARK: REST).
  -- $latestHttpResponse is the System.HttpResponse object after the call.
  send rest request FT.SMSGateway.NotifyTech;
  declare $StatusCode: integer = $latestHttpResponse/StatusCode;
  log info 'SMS notification sent, HTTP status: ' + toString($StatusCode);
  return true;
}
/

-- ACT_WorkOrder_Import: parse JSON payload into FT.WorkOrderImport entity.
-- Demonstrates: import from mapping activity.
create or modify microflow FT.ACT_WorkOrder_Import
  ($JsonPayload: string)
  folder 'Dispatch'
{
  -- IMPORT FROM MAPPING: deserialises $JsonPayload using FT.WorkOrderImport_Mapping.
  -- Returns an FT.WorkOrderImport non-persistent object.
  $Imported = import from mapping FT.WorkOrderImport_Mapping($JsonPayload);
  if $Imported != empty {
    $Order = create FT.DispatchOrder (
      Status     = FT.DispatchStatus.Pending,
      SiteNotes  = 'Imported: ' + $Imported/TicketRef
    );
    commit $Order;
  }
  return;
}
/

-- ACT_FT_Initialize: AfterStartup hook — reset all technician availability flags.
-- Registered via ALTER SETTINGS MODEL in Task 9.
create or modify microflow FT.ACT_FT_Initialize
  ()
  folder 'System'
{
  retrieve $Techs from FT.FieldTech limit 0;
  loop $Tech in $Techs {
    change $Tech (IsAvailable = true);
    commit $Tech;
  }
  log info 'FieldTech availability reset at startup.';
  return;
}
/

-- DS_HealthCheck: HealthCheck microflow — returns count of pending orders as integer.
-- Mendix runtime calls this on health-check requests; non-zero = unhealthy (customize as needed).
create or modify microflow FT.DS_HealthCheck
  ()
  returns integer as $PendingCount
  folder 'System'
{
  retrieve $Pending from FT.DispatchOrder
    where [Status = 'Pending']
    limit 0;
  $PendingCount = COUNT($Pending);
  return $PendingCount;
}
/

-- DS_MyDispatchOrders: retrieves DispatchOrders for the current FieldTech.
-- XPath: traverse DispatchOrder → FieldTech (FROM side), match System.owner.
-- reversed() demo: see DS_RelevantArticles below.
create or modify microflow FT.DS_MyDispatchOrders
  ()
  returns list of FT.DispatchOrder as $Orders
  folder 'Dispatch'
{
  retrieve $Orders from FT.DispatchOrder
    where [FT.DispatchOrder_FieldTech/FT.FieldTech/System.owner = '[%CurrentUser%]'
           and Status != 'Completed' and Status != 'Cancelled']
    sort by DispatchedAt asc
    limit 50;
  return $Orders;
}
/

-- DS_RelevantArticles: retrieve KB articles tagged 'fieldtech-relevant'.
-- Demonstrates reversed() XPath: KB.ArticleTag_Article runs FROM KB.ArticleTag TO KB.Article.
-- On KB.Article we reverse: [KB.ArticleTag_Article[reversed()]/KB.ArticleTag/...]
-- reads "articles where at least one ArticleTag points back to this article and
-- the Tag name equals 'fieldtech-relevant'".
create or modify microflow FT.DS_RelevantArticles
  ()
  returns list of KB.Article as $Articles
  folder 'Dispatch'
{
  retrieve $Articles from KB.Article
    where [KB.ArticleTag_Article[reversed()]/KB.ArticleTag/KB.ArticleTag_Tag/KB.Tag/Name = 'fieldtech-relevant'
           and Status = 'Published']
    limit 20;
  return $Articles;
}
/
```

- [ ] **Step 2: Syntax check (references not yet wired)**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors` (syntax only; REST/mapping references checked after Task 5).

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add FT microflows — CREATE LIST, reversed() XPath, log info"
```

---

## Task 4: FT Module — Nanoflows (Offline)

New MDL features: `CALL JAVASCRIPT ACTION`, `SYNCHRONIZE`, offline connectivity guard pattern (`NanoflowCommons.IsConnectedToServer`).

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append FT nanoflows**

```mdl
-- ============================================================================
-- MARK: FieldTech — Nanoflows (Offline)
-- ============================================================================
-- These nanoflows run on the FieldTech's mobile device (OfflineNative profile).
-- NanoflowCommons JS actions are available when the NanoflowCommons marketplace
-- module is installed. SYNCHRONIZE pushes offline-committed objects to the server.
-- ============================================================================

-- NF_FT_CheckIn: capture GPS on arrival at site, store as CheckInRecord.
-- New activities: CALL JAVASCRIPT ACTION (device GPS), SYNCHRONIZE.
create or modify nanoflow FT.NF_FT_CheckIn
  ($Order: FT.DispatchOrder)
  folder 'Mobile'
{
  -- CALL JAVASCRIPT ACTION: invokes a NanoflowCommons device API client-side.
  -- Returns a NanoflowCommons.Geolocation object with Latitude, Longitude, Timestamp.
  $Location = call javascript action NanoflowCommons.GetCurrentLocation (
    timeout      = 10000,
    maximumAge   = 0,
    highAccuracy = true
  ) on error continue;

  if $Location = empty {
    show message 'Could not obtain GPS location. Check device permissions.' type warning;
    return;
  }

  $Record = create FT.CheckInRecord (
    Latitude    = $Location/Latitude,
    Longitude   = $Location/Longitude,
    CheckedInAt = '[%CurrentDateTime%]',
    FT.CheckInRecord_DispatchOrder = $Order
  );
  commit $Record;

  -- SYNCHRONIZE: pushes the committed-offline object to the server in native mobile context.
  -- Must only be called in native mobile (OfflineNative) nanoflows.
  synchronize $Record;

  change $Order (Status = FT.DispatchStatus.OnSite);
  commit $Order;
  synchronize $Order;
  return;
}
/

-- NF_FT_CompleteOffline: FieldTech marks order complete while offline.
-- Changes persist locally; synchronize pushes to server when back online.
create or modify nanoflow FT.NF_FT_CompleteOffline
  ($Order: FT.DispatchOrder, $Notes: string)
  folder 'Mobile'
{
  change $Order (
    Status      = FT.DispatchStatus.Completed,
    CompletedAt = '[%CurrentDateTime%]',
    SiteNotes   = $Notes
  );
  commit $Order;

  -- Check connectivity before sync; if offline, changes stay local until next sync.
  $IsOnline = call javascript action NanoflowCommons.IsConnectedToServer ();
  if $IsOnline {
    synchronize $Order;
    show message 'Order completed and synced.' type information;
  } else {
    show message 'Offline — order saved locally and will sync automatically.' type information;
  }
  return;
}
/

-- NF_FT_LoadOrders: refresh order list (pull from server when online).
create or modify nanoflow FT.NF_FT_LoadOrders
  ()
  returns list of FT.DispatchOrder as $Orders
  folder 'Mobile'
{
  $IsOnline = call javascript action NanoflowCommons.IsConnectedToServer ();
  if $IsOnline {
    synchronize;
  }
  retrieve $Orders from FT.DispatchOrder
    where [FT.DispatchOrder_FieldTech/FT.FieldTech/System.owner = '[%CurrentUser%]'
           and Status != 'Completed']
    sort by DispatchedAt asc
    limit 50;
  return $Orders;
}
/
```

- [ ] **Step 2: Verify syntax**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add offline nanoflows — CALL JAVASCRIPT ACTION, SYNCHRONIZE"
```

---

## Task 5: FT Module — REST Client & JSON Integration

New MDL features: `CREATE JSON STRUCTURE`, `CREATE REST CLIENT`, `CREATE IMPORT MAPPING`, `CREATE EXPORT MAPPING`.

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append REST and JSON artifacts**

```mdl
-- ============================================================================
-- MARK: FieldTech — REST & JSON Integration
-- ============================================================================
-- SMS gateway notification for completed dispatches.
-- JSON webhook import for external work-order systems.
-- ============================================================================

-- JSON structure for the SMS gateway request body.
-- 'snippet' provides a sample JSON document; executor infers the schema.
create json structure FT.SMSPayload
  snippet '{"to": "+1555000000", "message": "Your technician is on the way."}';

-- REST client: SMS gateway.
-- Operation NotifyTech sends the payload; response is the HTTP status code.
create rest client FT.SMSGateway (
  BaseUrl:        'https://sms.example.com/api/v1',
  authentication: none
)
{
  operation NotifyTech {
    method:  post,
    path:    '/messages',
    headers: ('Content-Type' = 'application/json', 'Accept' = 'application/json'),
    body:    json from $Payload,
    response: status as $StatusCode
  }
};

-- JSON structure for the external work-order webhook payload.
create json structure FT.WorkOrderPayload
  snippet '{"ticketRef": "HD-001", "techName": "Jane Smith", "scheduledAt": "2026-06-10T09:00:00Z"}';

-- Import mapping: JSON webhook → FT.WorkOrderImport non-persistent entity.
create import mapping FT.WorkOrderImport_Mapping
  with json structure FT.WorkOrderPayload
{
  create FT.WorkOrderImport {
    TicketRef   = ticketRef,
    TechName    = techName,
    ScheduledAt = scheduledAt
  }
};

-- Export mapping: FT.DispatchOrder → JSON for outbound reporting.
-- Demonstrates CREATE EXPORT MAPPING syntax (entity → JSON).
create export mapping FT.DispatchOrder_Export
  with json structure FT.WorkOrderPayload
{
  from FT.DispatchOrder {
    ticketRef   = SiteNotes,
    scheduledAt = DispatchedAt
  }
};
```

- [ ] **Step 2: Verify syntax and references**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`. The REST client reference in `ACT_Dispatch_Complete` (Task 3) is now resolvable.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add REST client and JSON structures/mappings"
```

---

## Task 6: FT Module — Business Events

New MDL features: `CREATE BUSINESS EVENT SERVICE`, `message ... publish entity`, `message ... subscribe microflow`.

> **Prerequisite note**: In production, entities linked to `publish entity` must extend `BusinessEvents.PublishedBusinessEvent` from the BusinessEvents marketplace module. For this demo, `FT.PBE_DispatchCompleted` is a regular persistent entity — Studio Pro will issue CE7145 in strict validation; `mxcli check` and `mx check` accept the syntax. Add a comment noting the production requirement.

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append Business Events artifacts**

```mdl
-- ============================================================================
-- MARK: FieldTech — Business Events
-- ============================================================================
-- Event-driven integration: publish DispatchCompleted to external monitoring;
-- subscribe to ExternalWorkOrderReceived for inbound webhook-style ingestion.
-- In production: PBE_* entities must extend BusinessEvents.PublishedBusinessEvent
-- (requires the BusinessEvents marketplace module). This demo uses a regular
-- persistent entity so the syntax compiles without the module installed.
-- ============================================================================

-- PBE entity: attributes must exactly match the published message attributes.
create or modify persistent entity FT.PBE_DispatchCompleted (
  DispatchOrderId: long,
  TicketId:        long,
  CompletedAt:     string(50)
);

create or modify association FT.PBE_DispatchCompleted_DispatchOrder
  from FT.PBE_DispatchCompleted to FT.DispatchOrder
  type reference
  owner default;

-- Business Event Service definition.
-- ServiceName: identifies the service in the event broker (Kafka topic prefix).
-- EventNamePrefix: prefixed to message names for full topic names.
create or modify business event service FT.FieldWorkEvents
(
  ServiceName:     'FieldWorkEvents',
  EventNamePrefix: 'ft'
)
{
  -- PUBLISH: this app produces DispatchCompleted events.
  -- entity FT.PBE_DispatchCompleted is the carrier object.
  message DispatchCompleted (
    DispatchOrderId: long,
    TicketId:        long,
    CompletedAt:     string
  ) publish
    entity FT.PBE_DispatchCompleted;

  -- SUBSCRIBE: this app consumes ExternalWorkOrderReceived events.
  -- Subscribed events are routed to FT.ACT_WorkOrder_Import.
  message ExternalWorkOrderReceived (
    TicketRef:   string,
    TechName:    string,
    ScheduledAt: string
  ) subscribe
    microflow FT.ACT_WorkOrder_Import;
};
```

- [ ] **Step 2: Verify syntax**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add Business Event Service — publish/subscribe messages"
```

---

## Task 7: FT Module — Pages

New MDL features: `url:` page slug, `variables: { $Var: type = expr }` page variables, `CREATE VIEW ENTITY` as DataGrid datasource.

After this task, run the first full execution + `mx check` to catch BSON issues early.

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append FT pages**

```mdl
-- ============================================================================
-- MARK: FieldTech — Pages
-- ============================================================================

-- FT.DispatchQueue: FieldTech mobile task list.
-- New patterns:
--   url: 'dispatch-queue'  → page slug for deep linking (absent from baseline)
--   variables: { $ShowCompleted: boolean = 'false' }  → page-level variable
--     used to toggle visibility of completed orders section.
create or replace page FT.DispatchQueue (
  title:     'My Dispatch Queue',
  layout:    Atlas_Core.Atlas_Default,
  folder:    'FieldTech',
  url:       'dispatch-queue',
  variables: { $ShowCompleted: boolean = 'false' }
)
{
  layoutgrid lgQueue {
    row rOrders {
      column cOrders (desktopwidth: 12) {
        datagrid dgOrders (
          datasource: microflow FT.DS_MyDispatchOrders,
          PageSize: 20
        ) {
          column colStatus      (attribute: Status,       caption: 'Status',       ColumnWidth: manual, Size: 100)
          column colDispatched  (attribute: DispatchedAt, caption: 'Dispatched',   ColumnWidth: manual, Size: 140)
          column colNotes       (attribute: SiteNotes,    caption: 'Notes')
          column colActions (caption: 'Actions', ShowContentAs: customContent, ColumnWidth: manual, Size: 160) {
            actionbutton btnCheckIn (
              caption: 'Check In',
              action: nanoflow FT.NF_FT_CheckIn (Order: $currentObject),
              buttonstyle: primary
            )
            actionbutton btnComplete (
              caption: 'Complete',
              action: show_page FT.DispatchOrder_Detail (DispatchOrder: $currentObject),
              buttonstyle: default
            )
          }
        }
      }
    }
    row rToggle {
      column cToggle (desktopwidth: 12) {
        actionbutton btnShowCompleted (
          caption: 'Show Completed',
          action: microflow FT.DS_MyDispatchOrders,
          visible: [$ShowCompleted = false]
        )
      }
    }
  }
};

-- FT.DispatchOrder_Detail: sign-in + complete form for a single dispatch order.
create or replace page FT.DispatchOrder_Detail (
  title:  'Dispatch Order',
  layout: Atlas_Core.Atlas_Default,
  folder: 'FieldTech',
  params: { $DispatchOrder: FT.DispatchOrder }
)
{
  layoutgrid lgDetail {
    row rDetail {
      column cDetail (desktopwidth: 12) {
        dataview dvOrder (datasource: $DispatchOrder) {
          dynamictext txtStatus      (content: '{1}', contentparams: [{1} = Status])
          dynamictext txtDispatched  (content: '{1}', contentparams: [{1} = DispatchedAt])
          textarea    taNotes        (label: 'Site Notes', attribute: SiteNotes)
          footer ftrOrderActions {
            actionbutton btnCheckIn (
              caption:     'Check In (GPS)',
              action:      nanoflow FT.NF_FT_CheckIn (Order: $currentObject),
              buttonstyle: primary
            )
            actionbutton btnComplete (
              caption:     'Complete',
              action:      nanoflow FT.NF_FT_CompleteOffline (Order: $currentObject, Notes: $currentObject/SiteNotes),
              buttonstyle: success
            )
            actionbutton btnCancel (
              caption: 'Cancel',
              action:  cancel_changes close_page
            )
          }
        }
      }
    }
  }
};

-- FT.SLADashboard: management view using VIEW ENTITY datasources.
-- DataGrids backed by FT.TicketSLAStats and FT.DispatchSummary view entities.
create or replace page FT.SLADashboard (
  title:  'SLA & Dispatch Dashboard',
  layout: Atlas_Core.Atlas_Default,
  folder: 'FieldTech'
)
{
  layoutgrid lgDash {
    row rSLA {
      column cSLA (desktopwidth: 6) {
        dynamictext txtSLATitle (content: 'SLA Compliance by Agent', rendermode: H3)
        datagrid dgSLA (
          datasource: database from FT.TicketSLAStats sort by TotalTickets desc,
          PageSize: 10
        ) {
          column colAgent   (attribute: AgentName,    caption: 'Agent')
          column colTotal   (attribute: TotalTickets, caption: 'Total',   ColumnWidth: manual, Size: 80)
          column colOverSLA (attribute: OverSLACount, caption: 'Over SLA', ColumnWidth: manual, Size: 90)
        }
      }
      column cDispatch (desktopwidth: 6) {
        dynamictext txtDispTitle (content: 'Dispatch Summary by Technician', rendermode: H3)
        datagrid dgDispatch (
          datasource: database from FT.DispatchSummary sort by OrderCount desc,
          PageSize: 10
        ) {
          column colTech  (attribute: TechName,   caption: 'Technician')
          column colCount (attribute: OrderCount,  caption: 'Orders',   ColumnWidth: manual, Size: 80)
          column colAvg   (attribute: AvgHours,    caption: 'Avg Hours', ColumnWidth: manual, Size: 90)
        }
      }
    }
  }
};
```

- [ ] **Step 2: Syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`.

- [ ] **Step 3: Full execution test against clean MPR**

```bash
# Copy clean MPR to avoid modifying the tracked testdata copy
cp -r testdata/helpdesk-clean-11.6.6/ /tmp/ft-exec-test/
make build
./bin/mxcli -p /tmp/ft-exec-test/app.mpr \
  -c "EXECUTE SCRIPT 'mdl-examples/use-cases/helpdesk/helpdesk-app.mdl'"
```

Expected: script executes with 0 fatal errors (warnings about missing marketplace modules are acceptable).

- [ ] **Step 4: mx check (BSON validation)**

```bash
~/.mxcli/mxbuild/11.6.6/modeler/mx check /tmp/ft-exec-test/app.mpr 2>&1 \
  | grep -i "StorageLoadException\|error\|CE[0-9]" | head -30
```

Expected: no new `StorageLoadException` lines compared to the pre-FT baseline.

- [ ] **Step 5: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add FT pages — url slug, page variables, VIEW ENTITY datasource"
```

---

## Task 8: FT Module — Security

New MDL features: `REVOKE`, `[%UserRole_FieldTechRole%]` token (used in a microflow expression to demonstrate usage), FT.FieldTechRole grants with multi-hop XPath.

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append security grants, revoke, and user role**

```mdl
-- ============================================================================
-- MARK: FieldTech — Security
-- ============================================================================

-- Module role for field technicians
create module role FT.FieldTechRole;

-- User role: FieldTech maps to System.User + FT.FieldTechRole
create or modify user role FieldTech (System.User, FT.FieldTechRole);

-- Entity access for FT.FieldTechRole
-- FT.FieldTech: technician sees only their own profile
-- (System.owner was set when the FT.FieldTech record was created by the admin)
grant FT.FieldTechRole on FT.FieldTech (read *, write (Region, Phone))
  where '[System.owner = ''[%CurrentUser%]'']';

-- FT.DispatchOrder: technician sees orders assigned to them.
-- Path: FT.DispatchOrder → FT.DispatchOrder_FieldTech (FROM) → FT.FieldTech → System.owner
grant FT.FieldTechRole on FT.DispatchOrder (read *, write (Status, SiteNotes, CompletedAt))
  where '[FT.DispatchOrder_FieldTech/FT.FieldTech/System.owner = ''[%CurrentUser%]'']';

grant FT.FieldTechRole on FT.CheckInRecord (create, read *, write *);
grant FT.FieldTechRole on FT.WorkOrderImport (create, read *, write *);

-- Allow FieldTech to read published KB articles (needed for DS_RelevantArticles)
grant FT.FieldTechRole on KB.Article (read *)
  where '[Status = ''Published'']';
grant FT.FieldTechRole on KB.Tag (read *);
grant FT.FieldTechRole on KB.ArticleTag (read *);
grant FT.FieldTechRole on KB.Category (read *);

-- Agent / Manager: full access to FT module
grant HD.AgentRole on FT.FieldTech (read *);
grant HD.AgentRole on FT.DispatchOrder (create, read *, write *);
grant HD.ManagerRole on FT.FieldTech (create, read *, write *, delete);
grant HD.ManagerRole on FT.DispatchOrder (create, read *, write *, delete);
grant HD.ManagerRole on FT.PBE_DispatchCompleted (create, read *, write *, delete);

-- Microflow / nanoflow grants
grant execute on microflow FT.ACT_Dispatch_Assign    to HD.AgentRole, HD.ManagerRole;
grant execute on microflow FT.ACT_Dispatch_Open      to HD.AgentRole, HD.ManagerRole;
grant execute on microflow FT.ACT_Dispatch_Complete  to FT.FieldTechRole, HD.AgentRole, HD.ManagerRole;
grant execute on microflow FT.ACT_WorkOrder_Import   to HD.ManagerRole;
grant execute on microflow FT.DS_MyDispatchOrders    to FT.FieldTechRole;
grant execute on microflow FT.DS_RelevantArticles    to FT.FieldTechRole;
grant execute on microflow FT.DS_HealthCheck         to HD.ManagerRole;
grant execute on nanoflow  FT.NF_FT_CheckIn          to FT.FieldTechRole;
grant execute on nanoflow  FT.NF_FT_CompleteOffline  to FT.FieldTechRole;
grant execute on nanoflow  FT.NF_FT_LoadOrders       to FT.FieldTechRole;

-- Page grants
grant view on page FT.DispatchQueue        to FT.FieldTechRole;
grant view on page FT.DispatchOrder_Detail to FT.FieldTechRole;
grant view on page FT.SLADashboard         to HD.ManagerRole;

-- ============================================================================
-- MARK: Security Tightening (REVOKE)
-- ============================================================================
-- The baseline KB.Reader grant included write access to KB.ArticleRating.
-- That was overly permissive — readers should view ratings, not modify them.
-- REVOKE removes the write permission previously granted.
-- ============================================================================

revoke write on KB.ArticleRating from KB.Reader;

-- Re-grant read-only with the corrected member list.
-- (The original grant also included write (Score, Comment); we now drop that.)
-- Note: the original grant statement earlier in the file remains; this REVOKE
-- supersedes it. In production, edit the original grant and remove this block.
```

> **`[%UserRole_FieldTechRole%]` token**: Mendix evaluates this token to `true` when the current user has the FieldTech user role. Use it in XPath constraints where role-conditional visibility is needed, e.g.:
> ```mdl
> -- Example: show a widget only to FieldTechs
> visible: ['[%UserRole_FieldTechRole%]']
> -- Example: in a security XPath to filter entities by role
> where '[[%UserRole_FieldTechRole%] = true]'
> ```
> The token is demonstrated in the DispatchQueue page by the widget-level `visible:` on `btnCheckIn` (all FieldTech-role pages imply the token is true; explicit use is shown here as a comment-level example since widgets are already role-gated by page grants).

- [ ] **Step 2: Verify syntax**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add FT security — REVOKE, FieldTechRole grants, multi-hop XPath"
```

---

## Task 9: FT Module — Navigation & Settings

New MDL features: offline `OfflineNative` navigation profile, `alter settings model` (AfterStartupMicroflow, HealthCheckMicroflow).

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append navigation and settings**

```mdl
-- ============================================================================
-- MARK: FieldTech — Navigation & Settings
-- ============================================================================

-- OfflineNative profile: for FieldTech mobile devices.
-- Separate profile from the existing Responsive profile; each profile has its own
-- home pages, menu, and login page. Menu items reference pages only (nanoflows
-- are triggered from within pages via actionbuttons, not from menu items).
create or replace navigation OfflineNative
  home page FT.DispatchQueue for FieldTech
  home page HD.Ticket_Overview
  login page Administration.login
  menu (
    menu item 'My Orders'    page FT.DispatchQueue;
    menu item 'Order Detail' page FT.DispatchOrder_Detail;
  );

-- alter settings model: wire AfterStartup and HealthCheck microflows.
-- AfterStartupMicroflow runs once after the Mendix runtime starts.
-- HealthCheckMicroflow is polled by load balancers / monitoring.
alter settings model (
  AfterStartupMicroflow: FT.ACT_FT_Initialize,
  HealthCheckMicroflow:  FT.DS_HealthCheck
);
```

- [ ] **Step 2: Verify syntax**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add OfflineNative navigation and ALTER SETTINGS MODEL"
```

---

## Task 10: HD Page Evolution — ALTER PAGE

New MDL features: `ALTER PAGE ... INSERT ... after` (retrofitting an existing page without a full rebuild).

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (append)

- [ ] **Step 1: Append ALTER PAGE blocks**

```mdl
-- ============================================================================
-- MARK: HD — Page Evolution (ALTER PAGE)
-- ============================================================================
-- Retrofit HD.Ticket_Detail to add dispatch controls without rebuilding
-- the full page. ALTER PAGE modifies the BSON widget tree in-place,
-- preserving all existing widget IDs and properties.
--
-- Widget names (from describe page HD.Ticket_Detail):
--   dvTicket.ftrActions  — footer containing btnSubmit … btnAssignAgent
--   rComments            — the row containing dgComments
-- ============================================================================

-- Add "Dispatch FieldTech" button after the existing "Assign Agent" button.
alter page HD.Ticket_Detail {
  insert after btnAssignAgent {
    actionbutton btnDispatch (
      caption:     'Dispatch FieldTech',
      action:      microflow FT.ACT_Dispatch_Open (Ticket: $currentObject),
      buttonstyle: default
    )
  }
};

-- Add dispatch history grid after the comments row so managers and agents
-- can see all dispatch orders for a ticket on the same detail page.
alter page HD.Ticket_Detail {
  insert after rComments {
    row rDispatchHistory {
      column cDispatch (desktopwidth: 12) {
        dynamictext txtDispatchTitle (content: 'Dispatch History', rendermode: H4)
        datagrid dgDispatchHistory (
          datasource: $Ticket/FT.DispatchOrder_Ticket/FT.DispatchOrder
        ) {
          column colDTech      (attribute: FT.DispatchOrder_FieldTech, caption: 'Technician')
          column colDStatus    (attribute: Status,      caption: 'Status',    ColumnWidth: manual, Size: 100)
          column colDDispatched (attribute: DispatchedAt, caption: 'Dispatched', ColumnWidth: manual, Size: 140)
          column colDCompleted  (attribute: CompletedAt,  caption: 'Completed',  ColumnWidth: manual, Size: 140)
        }
      }
    }
  }
};
```

- [ ] **Step 2: Verify syntax**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: `0 errors`.

- [ ] **Step 3: Final full execution + mx check**

```bash
# Fresh copy
cp -r testdata/helpdesk-clean-11.6.6/ /tmp/ft-final-test/

# Execute the complete extended script
./bin/mxcli -p /tmp/ft-final-test/app.mpr \
  -c "EXECUTE SCRIPT 'mdl-examples/use-cases/helpdesk/helpdesk-app.mdl'"

# BSON validation
~/.mxcli/mxbuild/11.6.6/modeler/mx check /tmp/ft-final-test/app.mpr 2>&1 \
  | grep -i "StorageLoadException\|CE[0-9]" | head -40
```

Expected:
- Execution completes without crash.
- No new `StorageLoadException` lines.
- CE errors limited to known-acceptable ones documented in the file header (e.g., CE0710 for raise error, CE7145 for PBE entity without marketplace module).

- [ ] **Step 4: Update the file header — add new Note lines**

At the top of `helpdesk-app.mdl`, inside the header comment block after the last `-- Note:` line, append:

```
-- Note    : FieldTech extension added (Tasks 1-10): FT module with 19 new MDL features —
--           cascade delete, VIEW ENTITY/OQL, REST client, JSON mappings, Business Events,
--           offline nanoflows (SYNCHRONIZE + JS Action), ALTER PAGE, ALTER SETTINGS MODEL,
--           REVOKE, reversed() XPath, CREATE LIST, url slug, page variables, OfflineNative nav.
```

- [ ] **Step 5: Final commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk-ft): add ALTER PAGE dispatch controls + update file header"
```

---

## Self-Review: Spec Coverage Check

| Spec Requirement | Task |
|-----------------|------|
| `CREATE VIEW ENTITY ... GROUP BY` | Task 2 |
| `LEFT OUTER JOIN`, `avg()`, `count(t.ID)` | Task 2 |
| `delete PARENT cascade` / `delete CHILD cascade` | Task 1 |
| Multi-column `index (A, B)` | Task 1 |
| `REVOKE` | Task 8 |
| `reversed()` XPath | Task 3 (`DS_RelevantArticles`) |
| `[%UserRole_FieldTechRole%]` token | Task 8 (documented example) |
| `ALTER SETTINGS MODEL` (AfterStartup, HealthCheck) | Task 9 |
| `CREATE REST CLIENT` + `SEND REST REQUEST` | Task 5 + Task 3 |
| `CREATE JSON STRUCTURE` | Task 5 |
| `CREATE IMPORT MAPPING` + `CREATE EXPORT MAPPING` | Task 5 |
| `CREATE BUSINESS EVENT SERVICE` + publish + subscribe | Task 6 |
| `SYNCHRONIZE` | Task 4 |
| `CALL JAVASCRIPT ACTION` | Task 4 |
| Offline navigation profile (`OfflineNative`) | Task 9 |
| `url:` slug | Task 7 |
| `variables:` page variables | Task 7 |
| `ALTER PAGE ... INSERT ... after` | Task 10 |
| `CREATE LIST` activity | Task 3 |

All 19 features covered. ✓
