# Helpdesk Module Folder Structure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add module folder declarations to `helpdesk-app.mdl` so all documents are organised into `Ticket/`, `Ticket/Search/`, `Escalation/`, and `Escalation/WorkflowAdmin/` folders (KB gets `Article/`), then regenerate the golden testdata.

**Architecture:** Inline `folder` keyword/property added to every `create microflow`, `create nanoflow`, and `create page` statement. A trailing `-- MARK: Folder Organization` section demonstrates equivalent `move` commands for reference. Workflows (`WF_*`) stay in module root — MDL has no folder support for the `create workflow` statement.

**Tech Stack:** Go, MDL CLI (`mxcli`), Mendix MPR v2, `mx check` (Mendix 11.6.4)

---

## File Structure

| File | Change |
|------|--------|
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` | Add `folder` to 47 create statements; append move reference section |
| `testdata/helpdesk-golden/minimal.mpr` | Regenerated |
| `testdata/helpdesk-golden/mprcontents/**` | Updated mxunit files for every document that gains a folder |

---

## Folder Assignments Reference

**KB:**
- `Article/` → `ACT_Article_Publish`, `ACT_Article_Archive`, `SUB_Article_TruncateContent`, `NF_Article_FormatPreview`, `Article_Overview`, `Article_Detail`, `Article_NewEdit`

**HD:**
- `Ticket/` → `ACT_Ticket_Submit/Assign/Resolve/Reopen/Close`, `ACT_Ticket_SafeCommit`, `ACT_Ticket_MarkCommentsRead`, `ACT_EscalationRequest_Cleanup`, `DS_OverdueTicketCount`, `NF_Ticket_QuickCreate`, `NF_Priority_GetLabel`, `Ticket_Overview`, `Ticket_Detail`, `Ticket_NewEdit`
- `Ticket/Search/` → `NF_TicketSearch_Apply`, `TicketSearch_Form`
- `Escalation/` → `WFA_GetManagerAssignees`, `WFS_SendReminder/Approve/Reject`, `WFS_Escalation_Initialize/AutoReject/UpdateTicketPriority/NotifyAgent`, `WFC_EscalationRequest_OnCreate`, `ACT_StartEscalation`, `EscalationReview_Form`, `EscalationWorkflow_Overview`, `EscalationStart_Form`
- `Escalation/WorkflowAdmin/` → `ACT_Workflow_ChangeState/CompleteTask/GenerateJumpTo/ApplyJumpTo/GetHistory/GetContext`, `DS_WorkflowInstances`, `ACT_Workflow_ShowTaskPage/ShowAdminPage/Lock/Unlock/Notify`

---

### Task 1: Build CLI and establish baseline

**Files:**
- Read: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (no changes)

- [ ] **Step 1: Build the CLI**

```bash
make build
```

Expected: `bin/mxcli` produced, no errors.

- [ ] **Step 2: Baseline syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0, no error lines. If errors appear, stop and fix before continuing.

- [ ] **Step 3: Note the current golden mx check baseline**

```bash
~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/helpdesk-golden/minimal.mpr 2>&1 | grep -c "Error\|StorageLoadException" || echo "0 errors"
```

Expected: `0 errors` (or a known count that will be the baseline throughout).

---

### Task 2: KB module — Article/ microflows and nanoflow

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

**Syntax rule:** For `create or modify microflow` / `create or modify nanoflow`, insert `  folder 'Article'` on a new line immediately before `begin`.

- [ ] **Step 1: Add folder to KB.ACT_Article_Publish**

Find:
```
create or modify microflow KB.ACT_Article_Publish
  ($Article: KB.Article)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow KB.ACT_Article_Publish
  ($Article: KB.Article)
  returns boolean as $Success
  folder 'Article'
begin
```

- [ ] **Step 2: Add folder to KB.ACT_Article_Archive**

Find:
```
create or modify microflow KB.ACT_Article_Archive
  ($Article: KB.Article)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow KB.ACT_Article_Archive
  ($Article: KB.Article)
  returns boolean as $Success
  folder 'Article'
begin
```

- [ ] **Step 3: Add folder to KB.SUB_Article_TruncateContent**

Find:
```
create or modify microflow KB.SUB_Article_TruncateContent
  ($Article: KB.Article)
  returns string as $Preview
begin
```

Replace with:
```
create or modify microflow KB.SUB_Article_TruncateContent
  ($Article: KB.Article)
  returns string as $Preview
  folder 'Article'
begin
```

- [ ] **Step 4: Add folder to KB.NF_Article_FormatPreview**

Find:
```
create or modify nanoflow KB.NF_Article_FormatPreview
  ($Article: KB.Article)
  returns string as $Preview
begin
```

Replace with:
```
create or modify nanoflow KB.NF_Article_FormatPreview
  ($Article: KB.Article)
  returns string as $Preview
  folder 'Article'
begin
```

- [ ] **Step 5: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

---

### Task 3: KB module — Article/ pages

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

**Syntax rule:** For `create page`, insert `  folder: 'Article',` as a new line after `layout: ...` and before any `params:` (or before the closing `)` if no params).

- [ ] **Step 1: Add folder to KB.Article_Overview**

Find:
```
create page KB.Article_Overview
(
  title: 'Knowledge Base',
  layout: Atlas_Core.Atlas_Default
)
```

Replace with:
```
create page KB.Article_Overview
(
  title: 'Knowledge Base',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Article'
)
```

- [ ] **Step 2: Add folder to KB.Article_Detail**

Find:
```
create page KB.Article_Detail
(
  title: 'Article',
  layout: Atlas_Core.Atlas_Default,
  params: { $Article: KB.Article }
)
```

Replace with:
```
create page KB.Article_Detail
(
  title: 'Article',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Article',
  params: { $Article: KB.Article }
)
```

- [ ] **Step 3: Add folder to KB.Article_NewEdit**

Find:
```
create page KB.Article_NewEdit
(
  title: 'New / Edit Article',
  layout: Atlas_Core.Atlas_Default,
  params: { $Article: KB.Article }
)
```

Replace with:
```
create page KB.Article_NewEdit
(
  title: 'New / Edit Article',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Article',
  params: { $Article: KB.Article }
)
```

- [ ] **Step 4: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

- [ ] **Step 5: Commit KB Article/ changes**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add Article/ folder to KB module documents"
```

---

### Task 4: HD module — Ticket/ microflows (9 microflows)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Add folder to HD.ACT_Ticket_Submit**

Find:
```
create or modify microflow HD.ACT_Ticket_Submit
  ($Ticket: HD.Ticket)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow HD.ACT_Ticket_Submit
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
begin
```

- [ ] **Step 2: Add folder to HD.ACT_Ticket_Assign**

Find:
```
create or modify microflow HD.ACT_Ticket_Assign
  ($Ticket: HD.Ticket, $Agent: HD.Agent)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow HD.ACT_Ticket_Assign
  ($Ticket: HD.Ticket, $Agent: HD.Agent)
  returns boolean as $Success
  folder 'Ticket'
begin
```

- [ ] **Step 3: Add folder to HD.ACT_Ticket_Resolve**

Find:
```
create or modify microflow HD.ACT_Ticket_Resolve
  ($Ticket: HD.Ticket)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow HD.ACT_Ticket_Resolve
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
begin
```

- [ ] **Step 4: Add folder to HD.ACT_Ticket_Reopen**

Find:
```
create or modify microflow HD.ACT_Ticket_Reopen
  ($Ticket: HD.Ticket)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow HD.ACT_Ticket_Reopen
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
begin
```

- [ ] **Step 5: Add folder to HD.ACT_Ticket_Close**

Find:
```
create or modify microflow HD.ACT_Ticket_Close
  ($Ticket: HD.Ticket)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow HD.ACT_Ticket_Close
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
begin
```

- [ ] **Step 6: Add folder to HD.ACT_Ticket_MarkCommentsRead**

Find:
```
create or modify microflow HD.ACT_Ticket_MarkCommentsRead
  ($Ticket: HD.Ticket)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Ticket_MarkCommentsRead
  ($Ticket: HD.Ticket)
  folder 'Ticket'
begin
```

- [ ] **Step 7: Add folder to HD.ACT_EscalationRequest_Cleanup**

Find:
```
create or modify microflow HD.ACT_EscalationRequest_Cleanup
  ($Ticket: HD.Ticket)
begin
```

Replace with:
```
create or modify microflow HD.ACT_EscalationRequest_Cleanup
  ($Ticket: HD.Ticket)
  folder 'Ticket'
begin
```

- [ ] **Step 8: Add folder to HD.ACT_Ticket_SafeCommit**

Find:
```
create or modify microflow HD.ACT_Ticket_SafeCommit
  ($Ticket: HD.Ticket)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow HD.ACT_Ticket_SafeCommit
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
begin
```

- [ ] **Step 9: Add folder to HD.DS_OverdueTicketCount**

Find:
```
create or modify microflow HD.DS_OverdueTicketCount
  ()
  returns integer as $Count
begin
```

Replace with:
```
create or modify microflow HD.DS_OverdueTicketCount
  ()
  returns integer as $Count
  folder 'Ticket'
begin
```

- [ ] **Step 10: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

---

### Task 5: HD module — Ticket/ nanoflows (2 nanoflows)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Add folder to HD.NF_Ticket_QuickCreate**

Find:
```
create or modify nanoflow HD.NF_Ticket_QuickCreate
  ($Customer: HD.Customer, $Subject: string)
  returns HD.Ticket as $Ticket
begin
```

Replace with:
```
create or modify nanoflow HD.NF_Ticket_QuickCreate
  ($Customer: HD.Customer, $Subject: string)
  returns HD.Ticket as $Ticket
  folder 'Ticket'
begin
```

- [ ] **Step 2: Add folder to HD.NF_Priority_GetLabel**

Find:
```
create or modify nanoflow HD.NF_Priority_GetLabel
  ($PriorityStr: string)
  returns string as $CSSClass
begin
```

Replace with:
```
create or modify nanoflow HD.NF_Priority_GetLabel
  ($PriorityStr: string)
  returns string as $CSSClass
  folder 'Ticket'
begin
```

- [ ] **Step 3: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

---

### Task 6: HD module — Ticket/ pages (3 pages)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Add folder to HD.Ticket_Overview**

Find:
```
create page HD.Ticket_Overview
(
  title: 'Tickets',
  layout: Atlas_Core.Atlas_Default
)
```

Replace with:
```
create page HD.Ticket_Overview
(
  title: 'Tickets',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket'
)
```

- [ ] **Step 2: Add folder to HD.Ticket_Detail**

Find:
```
create page HD.Ticket_Detail
(
  title: 'Ticket',
  layout: Atlas_Core.Atlas_Default,
  params: { $Ticket: HD.Ticket }
)
```

Replace with:
```
create page HD.Ticket_Detail
(
  title: 'Ticket',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
)
```

- [ ] **Step 3: Add folder to HD.Ticket_NewEdit**

Find:
```
create page HD.Ticket_NewEdit
(
  title: 'New / Edit Ticket',
  layout: Atlas_Core.Atlas_Default,
  params: { $Ticket: HD.Ticket }
)
```

Replace with:
```
create page HD.Ticket_NewEdit
(
  title: 'New / Edit Ticket',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
)
```

- [ ] **Step 4: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

- [ ] **Step 5: Commit HD Ticket/ changes**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add Ticket/ folder to HD ticket documents"
```

---

### Task 7: HD module — Ticket/Search/ (1 nanoflow + 1 page)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Add folder to HD.NF_TicketSearch_Apply**

Find:
```
create or modify nanoflow HD.NF_TicketSearch_Apply
  ($Search: HD.TicketSearch)
  returns list of HD.Ticket as $Tickets
begin
```

Replace with:
```
create or modify nanoflow HD.NF_TicketSearch_Apply
  ($Search: HD.TicketSearch)
  returns list of HD.Ticket as $Tickets
  folder 'Ticket/Search'
begin
```

- [ ] **Step 2: Add folder to HD.TicketSearch_Form**

Find:
```
create page HD.TicketSearch_Form
(
  title: 'Search Tickets',
  layout: Atlas_Core.PopupLayout,
  params: { $Search: HD.TicketSearch }
)
```

Replace with:
```
create page HD.TicketSearch_Form
(
  title: 'Search Tickets',
  layout: Atlas_Core.PopupLayout,
  folder: 'Ticket/Search',
  params: { $Search: HD.TicketSearch }
)
```

- [ ] **Step 3: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

---

### Task 8: HD module — Escalation/ workflow support microflows (10 microflows)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Add folder to HD.WFA_GetManagerAssignees**

Find:
```
create or modify microflow HD.WFA_GetManagerAssignees
  ($workflow: System.Workflow, $EscalationRequest: HD.EscalationRequest)
  returns list of System.User as $Users
begin
```

Replace with:
```
create or modify microflow HD.WFA_GetManagerAssignees
  ($workflow: System.Workflow, $EscalationRequest: HD.EscalationRequest)
  returns list of System.User as $Users
  folder 'Escalation'
begin
```

- [ ] **Step 2: Add folder to HD.WFS_SendReminder**

Find:
```
create or modify microflow HD.WFS_SendReminder
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.WFS_SendReminder
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation'
begin
```

- [ ] **Step 3: Add folder to HD.WFS_Approve**

Find:
```
create or modify microflow HD.WFS_Approve
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.WFS_Approve
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation'
begin
```

- [ ] **Step 4: Add folder to HD.WFS_Reject**

Find:
```
create or modify microflow HD.WFS_Reject
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.WFS_Reject
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation'
begin
```

- [ ] **Step 5: Add folder to HD.WFS_Escalation_Initialize**

Find:
```
create or modify microflow HD.WFS_Escalation_Initialize
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.WFS_Escalation_Initialize
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation'
begin
```

- [ ] **Step 6: Add folder to HD.WFS_AutoReject**

Find:
```
create or modify microflow HD.WFS_AutoReject
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.WFS_AutoReject
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation'
begin
```

- [ ] **Step 7: Add folder to HD.WFS_UpdateTicketPriority**

Find:
```
create or modify microflow HD.WFS_UpdateTicketPriority
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.WFS_UpdateTicketPriority
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation'
begin
```

- [ ] **Step 8: Add folder to HD.WFS_NotifyAgent**

Find:
```
create or modify microflow HD.WFS_NotifyAgent
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.WFS_NotifyAgent
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation'
begin
```

- [ ] **Step 9: Add folder to HD.ACT_StartEscalation**

Find:
```
create or modify microflow HD.ACT_StartEscalation
  ($Ticket: HD.Ticket, $Reason: string)
  returns boolean as $Success
begin
```

Replace with:
```
create or modify microflow HD.ACT_StartEscalation
  ($Ticket: HD.Ticket, $Reason: string)
  returns boolean as $Success
  folder 'Escalation'
begin
```

- [ ] **Step 10: Add folder to HD.WFC_EscalationRequest_OnCreate**

Find:
```
create or modify microflow HD.WFC_EscalationRequest_OnCreate
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.WFC_EscalationRequest_OnCreate
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation'
begin
```

- [ ] **Step 11: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

---

### Task 9: HD module — Escalation/ pages (3 pages)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Add folder to HD.EscalationReview_Form**

Find:
```
create page HD.EscalationReview_Form
(
  title: 'Escalation Review',
  layout: Atlas_Core.Atlas_Default,
  params: { $WorkflowUserTask: System.WorkflowUserTask }
)
```

Replace with:
```
create page HD.EscalationReview_Form
(
  title: 'Escalation Review',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Escalation',
  params: { $WorkflowUserTask: System.WorkflowUserTask }
)
```

- [ ] **Step 2: Add folder to HD.EscalationWorkflow_Overview**

Find:
```
create page HD.EscalationWorkflow_Overview
(
  title: 'Escalation Workflow',
  layout: Atlas_Core.Atlas_Default
)
```

Replace with:
```
create page HD.EscalationWorkflow_Overview
(
  title: 'Escalation Workflow',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Escalation'
)
```

- [ ] **Step 3: Add folder to HD.EscalationStart_Form**

Find:
```
create page HD.EscalationStart_Form
(
  title: 'Request Escalation',
  layout: Atlas_Core.PopupLayout
)
```

Replace with:
```
create page HD.EscalationStart_Form
(
  title: 'Request Escalation',
  layout: Atlas_Core.PopupLayout,
  folder: 'Escalation'
)
```

- [ ] **Step 4: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

- [ ] **Step 5: Commit Escalation/ changes**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add Escalation/ folder to HD escalation documents"
```

---

### Task 10: HD module — Escalation/WorkflowAdmin/ microflows (12 documents)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Add folder to HD.ACT_Workflow_ChangeState**

Find:
```
create or modify microflow HD.ACT_Workflow_ChangeState
  ($Workflow: System.Workflow, $Operation: string)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_ChangeState
  ($Workflow: System.Workflow, $Operation: string)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 2: Add folder to HD.ACT_Workflow_CompleteTask**

Find:
```
create or modify microflow HD.ACT_Workflow_CompleteTask
  ($UserTask: System.WorkflowUserTask, $Outcome: string)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_CompleteTask
  ($UserTask: System.WorkflowUserTask, $Outcome: string)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 3: Add folder to HD.ACT_Workflow_GenerateJumpTo**

Find:
```
create or modify microflow HD.ACT_Workflow_GenerateJumpTo
  ($Workflow: System.Workflow)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_GenerateJumpTo
  ($Workflow: System.Workflow)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 4: Add folder to HD.ACT_Workflow_ApplyJumpTo**

Find:
```
create or modify microflow HD.ACT_Workflow_ApplyJumpTo
  ($Workflow: System.Workflow)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_ApplyJumpTo
  ($Workflow: System.Workflow)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 5: Add folder to HD.ACT_Workflow_GetHistory**

Find:
```
create or modify microflow HD.ACT_Workflow_GetHistory
  ($Workflow: System.Workflow)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_GetHistory
  ($Workflow: System.Workflow)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 6: Add folder to HD.ACT_Workflow_GetContext**

Find:
```
create or modify microflow HD.ACT_Workflow_GetContext
  ($Workflow: System.Workflow)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_GetContext
  ($Workflow: System.Workflow)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 7: Add folder to HD.DS_WorkflowInstances**

Find:
```
create or modify microflow HD.DS_WorkflowInstances
  ($EscalationRequest: HD.EscalationRequest)
begin
```

Replace with:
```
create or modify microflow HD.DS_WorkflowInstances
  ($EscalationRequest: HD.EscalationRequest)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 8: Add folder to HD.ACT_Workflow_ShowTaskPage**

Find:
```
create or modify microflow HD.ACT_Workflow_ShowTaskPage
  ($UserTask: System.WorkflowUserTask)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_ShowTaskPage
  ($UserTask: System.WorkflowUserTask)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 9: Add folder to HD.ACT_Workflow_ShowAdminPage**

Find:
```
create or modify microflow HD.ACT_Workflow_ShowAdminPage
  ($Workflow: System.Workflow)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_ShowAdminPage
  ($Workflow: System.Workflow)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 10: Add folder to HD.ACT_Workflow_Lock**

Find:
```
create or modify microflow HD.ACT_Workflow_Lock
  ($Workflow: System.WorkflowDefinition)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_Lock
  ($Workflow: System.WorkflowDefinition)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 11: Add folder to HD.ACT_Workflow_Unlock**

Find:
```
create or modify microflow HD.ACT_Workflow_Unlock
  ($Workflow: System.WorkflowDefinition)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_Unlock
  ($Workflow: System.WorkflowDefinition)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 12: Add folder to HD.ACT_Workflow_Notify**

Find:
```
create or modify microflow HD.ACT_Workflow_Notify
  ($Workflow: System.Workflow)
begin
```

Replace with:
```
create or modify microflow HD.ACT_Workflow_Notify
  ($Workflow: System.Workflow)
  folder 'Escalation/WorkflowAdmin'
begin
```

- [ ] **Step 13: Run syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

- [ ] **Step 14: Commit WorkflowAdmin/ changes**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add Escalation/WorkflowAdmin/ folder to HD workflow admin microflows"
```

---

### Task 11: Add trailing move command reference section

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 1: Append the move section at the end of the file**

Append the following block after the last line of the file (after the navigation section):

```mdl

-- MARK: Folder Organization — equivalent move commands
-- These move commands are equivalent to the inline folder declarations above.
-- Use this pattern to reorganize an existing flat project without modifying
-- every create statement.

-- KB Module
move microflow KB.ACT_Article_Publish          to folder 'Article';
move microflow KB.ACT_Article_Archive          to folder 'Article';
move microflow KB.SUB_Article_TruncateContent  to folder 'Article';
move nanoflow  KB.NF_Article_FormatPreview     to folder 'Article';
move page      KB.Article_Overview             to folder 'Article';
move page      KB.Article_Detail               to folder 'Article';
move page      KB.Article_NewEdit              to folder 'Article';

-- HD Module — Ticket
move microflow HD.ACT_Ticket_Submit            to folder 'Ticket';
move microflow HD.ACT_Ticket_Assign            to folder 'Ticket';
move microflow HD.ACT_Ticket_Resolve           to folder 'Ticket';
move microflow HD.ACT_Ticket_Reopen            to folder 'Ticket';
move microflow HD.ACT_Ticket_Close             to folder 'Ticket';
move microflow HD.ACT_Ticket_SafeCommit        to folder 'Ticket';
move microflow HD.ACT_Ticket_MarkCommentsRead  to folder 'Ticket';
move microflow HD.ACT_EscalationRequest_Cleanup to folder 'Ticket';
move microflow HD.DS_OverdueTicketCount        to folder 'Ticket';
move nanoflow  HD.NF_Ticket_QuickCreate        to folder 'Ticket';
move nanoflow  HD.NF_Priority_GetLabel         to folder 'Ticket';
move page      HD.Ticket_Overview              to folder 'Ticket';
move page      HD.Ticket_Detail                to folder 'Ticket';
move page      HD.Ticket_NewEdit               to folder 'Ticket';

-- HD Module — Ticket/Search
move nanoflow  HD.NF_TicketSearch_Apply        to folder 'Ticket/Search';
move page      HD.TicketSearch_Form            to folder 'Ticket/Search';

-- HD Module — Escalation
move microflow HD.WFA_GetManagerAssignees          to folder 'Escalation';
move microflow HD.WFS_SendReminder                 to folder 'Escalation';
move microflow HD.WFS_Approve                      to folder 'Escalation';
move microflow HD.WFS_Reject                       to folder 'Escalation';
move microflow HD.WFS_Escalation_Initialize        to folder 'Escalation';
move microflow HD.WFS_AutoReject                   to folder 'Escalation';
move microflow HD.WFS_UpdateTicketPriority         to folder 'Escalation';
move microflow HD.WFS_NotifyAgent                  to folder 'Escalation';
move microflow HD.WFC_EscalationRequest_OnCreate   to folder 'Escalation';
move microflow HD.ACT_StartEscalation              to folder 'Escalation';
move page      HD.EscalationReview_Form            to folder 'Escalation';
move page      HD.EscalationWorkflow_Overview      to folder 'Escalation';
move page      HD.EscalationStart_Form             to folder 'Escalation';

-- HD Module — Escalation/WorkflowAdmin
move microflow HD.ACT_Workflow_ChangeState       to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_CompleteTask      to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_GenerateJumpTo    to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_ApplyJumpTo       to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_GetHistory        to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_GetContext        to folder 'Escalation/WorkflowAdmin';
move microflow HD.DS_WorkflowInstances           to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_ShowTaskPage      to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_ShowAdminPage     to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_Lock              to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_Unlock            to folder 'Escalation/WorkflowAdmin';
move microflow HD.ACT_Workflow_Notify            to folder 'Escalation/WorkflowAdmin';
```

**Note:** When this MDL is run against a project that already has the inline folder declarations above, the move commands are no-ops (documents are already in the correct folder). They serve as documentation of the move pattern.

- [ ] **Step 2: Run final syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): append move command reference section for folder organization"
```

---

### Task 12: Update golden testdata

**Files:**
- Modify: `testdata/helpdesk-golden/minimal.mpr` + `testdata/helpdesk-golden/mprcontents/**`

- [ ] **Step 1: Apply the updated MDL to the golden project**

```bash
./bin/mxcli -p testdata/helpdesk-golden/minimal.mpr \
    -c "$(cat mdl-examples/use-cases/helpdesk/helpdesk-app.mdl)"
```

Expected: runs without fatal errors. Warnings about existing documents are OK.

- [ ] **Step 2: Run mx check on the updated golden**

```bash
~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/helpdesk-golden/minimal.mpr \
    2>&1 | grep -i "StorageLoadException\|Error"
```

Expected: no new lines compared to the baseline from Task 1 Step 3.

- [ ] **Step 3: Review changed mxunit files**

```bash
git diff --name-only testdata/helpdesk-golden/
```

Expected: only `minimal.mpr` and `mprcontents/**` files changed. Each changed mxunit corresponds to a document that gained a folder assignment.

- [ ] **Step 4: Commit the golden update**

```bash
git add testdata/helpdesk-golden/
git commit -m "test(golden): update helpdesk golden after adding module folder structure"
```

---

## Self-Review

**Spec coverage:**
- ✅ KB `Article/` folder: Tasks 2–3
- ✅ HD `Ticket/` folder: Tasks 4–6
- ✅ HD `Ticket/Search/` folder: Task 7
- ✅ HD `Escalation/` folder: Tasks 8–9
- ✅ HD `Escalation/WorkflowAdmin/` folder: Task 10
- ✅ Trailing move section: Task 11
- ✅ Golden update: Task 12
- ✅ Workflow `WF_*` stay in root (no task needed — spec documents this as an exception)

**Placeholder scan:** No TBD/TODO/placeholder text in any step. All code blocks contain exact text.

**Type consistency:** All document names match the MDL file exactly (verified against the source). Folder paths use `/` separator throughout.
