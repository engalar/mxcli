# Helpdesk App — Module Folder Structure Design

**Date:** 2026-05-23  
**Scope:** `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` + golden testdata  
**Approach:** Process-based grouping (Mendix best practice), 3-level mixed hierarchy  

---

## Goal

Add module folder structure to the helpdesk baseline MDL so that Studio Pro's App Explorer is readable at a glance and follows the Mendix [naming conventions best practice](https://docs.mendix.com/refguide/modeling/best-practices/dev-best-practices/naming-conventions/#folder-structure).

Documents are grouped by **business process**, not by document type — so a developer working on "Escalation" finds all related pages, microflows, nanoflows, and workflow support logic in one place.

---

## Folder Assignments

### HD Module

| Folder | Documents |
|--------|-----------|
| `Ticket/` | `ACT_Ticket_Submit`, `ACT_Ticket_Assign`, `ACT_Ticket_Resolve`, `ACT_Ticket_Reopen`, `ACT_Ticket_Close`, `ACT_Ticket_SafeCommit`, `ACT_Ticket_MarkCommentsRead`, `ACT_EscalationRequest_Cleanup`, `DS_OverdueTicketCount`, `NF_Ticket_QuickCreate`, `NF_Priority_GetLabel`, `Ticket_Overview`, `Ticket_Detail`, `Ticket_NewEdit` |
| `Ticket/Search/` | `NF_TicketSearch_Apply`, `TicketSearch_Form` |
| `Escalation/` | `ACT_StartEscalation`, `WFA_GetManagerAssignees`, `WFS_SendReminder`, `WFS_Approve`, `WFS_Reject`, `WFS_Escalation_Initialize`, `WFS_AutoReject`, `WFS_UpdateTicketPriority`, `WFS_NotifyAgent`, `WFC_EscalationRequest_OnCreate`, `EscalationStart_Form`, `EscalationReview_Form`, `EscalationWorkflow_Overview` |
| `Escalation/WorkflowAdmin/` | `ACT_Workflow_ChangeState`, `ACT_Workflow_CompleteTask`, `ACT_Workflow_GenerateJumpTo`, `ACT_Workflow_ApplyJumpTo`, `ACT_Workflow_GetHistory`, `ACT_Workflow_GetContext`, `DS_WorkflowInstances`, `ACT_Workflow_ShowTaskPage`, `ACT_Workflow_ShowAdminPage`, `ACT_Workflow_Lock`, `ACT_Workflow_Unlock`, `ACT_Workflow_Notify` |
| Root (no folder support) | Entities, Enumerations, Constants, `WF_SUB_ManagerReview`, `WF_TicketEscalation` |

**Rationale for `Ticket/Search/` sub-folder:** The search/filter feature (`TicketSearch_Form` + `NF_TicketSearch_Apply`) is a distinct UX concern from the ticket lifecycle actions. Isolating it makes both groups easier to scan.

**Rationale for `Escalation/WorkflowAdmin/` sub-folder:** The 12 `ACT_Workflow_*` microflows are technical demonstrations of Mendix workflow API activities (lock, unlock, jump-to, etc.) rather than business escalation logic. Separating them prevents cluttering the business-oriented `Escalation/` folder.

**Workflow exception:** `WF_SUB_ManagerReview` and `WF_TicketEscalation` remain in the module root because the MDL `create or replace workflow` statement has no `folder` keyword and `move workflow` is not yet in the organize-project skill's supported type list.

### KB Module

| Folder | Documents |
|--------|-----------|
| `Article/` | `ACT_Article_Publish`, `ACT_Article_Archive`, `SUB_Article_TruncateContent`, `NF_Article_FormatPreview`, `Article_Overview`, `Article_Detail`, `Article_NewEdit` |
| Root (no folder support) | Entities (`Category`, `Tag`, `Article`, `ArticleTag`, `ArticleRating`), Enumerations |

---

## Implementation Strategy

### 1. Inline `folder` declarations on `create`

Add `folder` at the point of document creation so the MDL file reflects final state directly:

- **Microflows / Nanoflows:** `folder 'Ticket'` keyword before `begin`  
  ```mdl
  create or modify microflow HD.ACT_Ticket_Submit ($Ticket: HD.Ticket)
    returns boolean as $Success
    folder 'Ticket'
  begin
    ...
  end;
  ```

- **Pages:** `folder: 'Ticket'` property inside the properties block  
  ```mdl
  create page HD.Ticket_Overview
  (
    title: 'Tickets',
    layout: Atlas_Core.Atlas_Default,
    folder: 'Ticket'
  )
  { ... }
  ```

### 2. Trailing `move` command section

A `-- MARK: Folder Organization` section at the file end shows the equivalent `move` command form for reference. This is useful as a regression test for the `move` MDL command and demonstrates the reorganization pattern for projects that already have flat documents.

### 3. Document ordering

Inline `folder` declarations do not require any reordering of documents in the MDL file. The order of `create` statements remains unchanged to minimize diff noise.

---

## Files Changed

| File | Change |
|------|--------|
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` | Add `folder` to all create statements; add trailing move section |
| `testdata/helpdesk-golden/minimal.mpr` | Regenerated golden (run the MDL, update golden) |
| `testdata/helpdesk-golden/mprcontents/` | Updated unit files for documents that now have folder assignments |

---

## Validation

```bash
# 1. Syntax check
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl

# 2. Apply to testdata project and mx check
./bin/mxcli -p testdata/helpdesk-golden/minimal.mpr \
    -c "$(cat mdl-examples/use-cases/helpdesk/helpdesk-app.mdl)"

~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/helpdesk-golden/minimal.mpr \
    2>&1 | grep -i "StorageLoadException\|Error"

# 3. Restore if needed
git restore testdata/helpdesk-golden/
```
