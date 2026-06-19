# Capstone HelpDesk Refinement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete missing operations and apply professional UI to the HelpDeskE2E capstone reference implementation.

**Architecture:** Four incremental MDL files (13–16) executed in order against the existing project, plus SCSS brand theme appended to main.scss.

**Tech Stack:** MDL (Mendix Definition Language), Mendix Atlas_Core theme, SCSS, TicketStatusBadge pluggable widget (installed by 10-widget.mdl).

## Global Constraints

- All MDL files must be idempotent (`create or modify` / `create or replace`)
- No Java/JS Action modifications
- No new external widget packages
- Run order: 13 → 14 → 15 → 16
- All files added to `scripts/validate-academy-capstone.sh` exec list
- Theme applied via `main.scss` append (not MDL)

---

### Task 1: Assign Agent Workflow (13-improve-operations.mdl)

**Files:**
- Create: `academy/zh/capstone-helpdesk/参考实现/13-improve-operations.mdl`

**Interfaces:**
- Consumes: `HD.Ticket`, `HD.Agent` entities from 01-domain.mdl; `HD.ACT_Ticket_Assign` microflow from 02-microflows.mdl
- Produces: `HD.SelectAgent_Form` page, `HD.ACT_OpenAssignForm` microflow, `HD.ACT_DoAssign` microflow

- [ ] **Step 1: Write the new MDL file header + entity-less microflows**

Add to `13-improve-operations.mdl`:
```mdl
-- ============================================================
-- 模块 13：操作补全 — Assign 指配、Delete 删除、Search 搜索
-- 前提：01-12 已执行
-- ============================================================

-- 打开指配选择弹窗
create or modify microflow HD.ACT_OpenAssignForm
  ($Ticket: HD.Ticket)
  returns nothing
  folder 'Ticket'
{
  show page HD.SelectAgent_Form;
}
/
```

- [ ] **Step 2: Add the SelectAgent popup page**

```mdl
create or replace page HD.SelectAgent_Form
(
  title:  'Select Agent',
  layout: Atlas_Core.PopupLayout,
  folder: 'Ticket'
)
{
  layoutgrid lgMain {
    row rHeader {
      column cTitle (desktopwidth: 12) {
        dynamictext txtTitle (content: 'Select an Agent to Assign', rendermode: H3)
      }
    }
    row rList {
      column cList (desktopwidth: 12) {
        datagrid dgAgents (
          datasource: database from HD.Agent
        ) {
          column colName  (attribute: Name,      caption: 'Name')
          column colEmail (attribute: Email,     caption: 'Email')
          column colActive(attribute: IsActive,  caption: 'Active')
          column colAction (caption: '', ShowContentAs: customContent) {
            actionbutton btnSelect (
              caption:     'Select',
              action:      microflow HD.ACT_DoAssign (Ticket: $Ticket, Agent: $currentObject) close_page,
              buttonstyle: primary
            )
          }
        }
      }
    }
  }
};
```

- [ ] **Step 3: Add the DoAssign microflow**

```mdl
create or modify microflow HD.ACT_DoAssign
  ($Ticket: HD.Ticket, $Agent: HD.Agent)
  returns boolean as $Success
  folder 'Ticket'
{
  $Success = call microflow HD.ACT_Ticket_Assign(Ticket = $Ticket, Agent = $Agent);
  return $Success;
}
/
```

- [ ] **Step 4: Wire Assign button to Ticket_Detail**

```mdl
alter page HD.Ticket_Detail {
  insert after btnSubmit {
    actionbutton btnAssign (
      caption:     'Assign',
      action:      microflow HD.ACT_OpenAssignForm (Ticket: $currentObject),
      buttonstyle: default
    )
  }
};
```

- [ ] **Step 5: Add Priority dropdown to Ticket_NewEdit**

```mdl
alter page HD.Ticket_NewEdit {
  insert after taDescription {
    dropdown ddPriority (
      label:     'Priority',
      attribute: Priority
    )
  }
};
```

- [ ] **Step 6: Add Ticket Delete operation**

```mdl
create or modify microflow HD.ACT_Ticket_Delete
  ($Ticket: HD.Ticket)
  returns boolean as $Success
  folder 'Ticket'
{
  show message 'Are you sure you want to delete this ticket?' type confirmation;
  if get confirmation {
    delete $Ticket;
    commit $Ticket;
    return true;
  }
  return false;
}
/

grant execute on microflow HD.ACT_Ticket_Delete to HD.ManagerRole;

alter page HD.Ticket_Detail {
  insert after btnClose {
    actionbutton btnDelete (
      caption:     'Delete',
      action:      microflow HD.ACT_Ticket_Delete (Ticket: $currentObject) close_page,
      buttonstyle: danger
    )
  }
};
```

- [ ] **Step 7: Grant execute access for new microflows**

```mdl
grant execute on microflow HD.ACT_OpenAssignForm to HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_DoAssign       to HD.AgentRole, HD.ManagerRole;
```

- [ ] **Step 8: Verify 13-improve-operations.mdl**

```bash
cd /mnt/data_sdb/mxcli && go run ./cmd/mdlrun -p HelpDeskE2E/HelpDeskE2E.mpr academy/zh/capstone-helpdesk/参考实现/13-improve-operations.mdl
```
Expected: All statements succeed, no errors.

---

### Task 2: Beautify Pages (14-beautify-pages.mdl)

**Files:**
- Create: `academy/zh/capstone-helpdesk/参考实现/14-beautify-pages.mdl`

**Interfaces:**
- Consumes: TicketStatusBadge widget (installed by 10-widget.mdl), all HD + KB pages from 04 + 06
- Produces: ALTER PAGE statements for all beautified pages

- [ ] **Step 1: Beautify Ticket_Overview — replace Status column with custom badge widget**

```mdl
-- ============================================================
-- 模块 14：页面美化
-- 前提：01-13 已执行
-- ============================================================

alter page HD.Ticket_Overview {
  replace colStatus with column colStatus (
    caption: 'Status', ColumnWidth: manual, Size: 120
  ) {
    ticketstatusbadge badgeStatus (
      widget: com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge,
      datasource: $currentObject/Status
    )
  }
};
```

- [ ] **Step 2: Add priority color labels to Ticket_Overview**

```mdl
alter page HD.Ticket_Overview {
  replace colPriority with column colPriority (
    caption: 'Priority', ColumnWidth: manual, Size: 100
  ) {
    dynamictext txtPriority (
      content: '{1}', contentparams: [{1} = Priority],
      rendermode: Text
    )
  }
};
```

- [ ] **Step 3: Beautify MyTickets_Overview (same changes)**

```mdl
alter page HD.MyTickets_Overview {
  replace colStatus with column colStatus (
    caption: 'Status', ColumnWidth: manual, Size: 120
  ) {
    ticketstatusbadge badgeStatus (
      widget: com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge,
      datasource: $currentObject/Status
    )
  }
};

alter page HD.MyTickets_Overview {
  replace colPriority with column colPriority (
    caption: 'Priority', ColumnWidth: manual, Size: 100
  ) {
    dynamictext txtPriority (
      content: '{1}', contentparams: [{1} = Priority],
      rendermode: Text
    )
  }
};
```

- [ ] **Step 4: Beautify Ticket_Detail — status badge, priority color, comment timestamps**

```mdl
alter page HD.Ticket_Detail {
  replace txtStatus with dynamictext txtStatus (
    content: 'Status: {1}', contentparams: [{1} = Status]
  );
  insert after txtStatus {
    ticketstatusbadge badgeStatus (
      widget: com.helpdesk.widget.ticketstatusbadge.TicketStatusBadge,
      datasource: $currentObject/Status
    )
  }
};

alter page HD.Ticket_Detail {
  insert after badgeStatus {
    dynamictext txtPriorityColor (
      content: 'Priority: {1}', contentparams: [{1} = Priority]
    )
  }
};
```

- [ ] **Step 5: Beautify KB Article pages**

```mdl
alter page KB.Article_Overview {
  replace colDate with column colDate (
    attribute: PublishedAt, caption: 'Published', ColumnWidth: manual, Size: 140
  )
};

alter page KB.Article_Detail {
  replace ftrActions with footer ftrActions {
    actionbutton btnPublish (
      caption:     'Publish',
      action:      microflow KB.ACT_Article_Publish (Article: $currentObject),
      buttonstyle: primary
    )
    actionbutton btnArchive (
      caption:     'Archive',
      action:      microflow KB.ACT_Article_Archive (Article: $currentObject),
      buttonstyle: default
    )
    actionbutton btnEdit (
      caption:     'Edit',
      action:      show_page KB.Article_NewEdit (Article: $currentObject),
      buttonstyle: default
    )
  }
};
```

- [ ] **Step 6: Beautify Ticket_NewEdit — form section dividers**

```mdl
alter page HD.Ticket_NewEdit {
  insert after ddPriority {
    dynamictext txtDivider (content: '───', rendermode: Text)
  }
};
```

- [ ] **Step 7: Verify 14-beautify-pages.mdl**

```bash
cd /mnt/data_sdb/mxcli && go run ./cmd/mdlrun -p HelpDeskE2E/HelpDeskE2E.mpr academy/zh/capstone-helpdesk/参考实现/14-beautify-pages.mdl
```

---

### Task 3: Dashboard (15-dashboard.mdl)

**Files:**
- Create: `academy/zh/capstone-helpdesk/参考实现/15-dashboard.mdl`

**Interfaces:**
- Consumes: `HD.Ticket` entity, ticket microflows
- Produces: `HD.DashboardStats` non-persistent entity, `HD.DS_DashboardStats` microflow, `HD.Dashboard_Home` page

- [ ] **Step 1: Create non-persistent entity + data source microflow**

```mdl
-- ============================================================
-- 模块 15：Dashboard 仪表盘
-- 前提：01-14 已执行
-- ============================================================

create or modify non-persistent entity HD.DashboardStats (
  OpenCount:        integer,
  OverdueCount:     integer,
  NewTodayCount:    integer,
  AvgResponseHours: decimal(4, 1)
);

create or modify microflow HD.DS_DashboardStats
  ()
  returns HD.DashboardStats as $Stats
  folder 'Dashboard'
{
  $Stats = create HD.DashboardStats;

  retrieve $Open from HD.Ticket
    where [Status != 'Closed' and Status != 'Resolved']
    limit 0;
  $Stats/OpenCount = count($Open);

  retrieve $Overdue from HD.Ticket where [IsOverSLA = true()] limit 0;
  $Stats/OverdueCount = count($Overdue);

  retrieve $Today from HD.Ticket
    where [System.changedDate >= '[%BeginOfCurrentDay%]']
    limit 0;
  $Stats/NewTodayCount = count($Today);

  retrieve $Resolved from HD.Ticket
    where [ResolvedAt != empty]
    limit 0;
  declare $Total: decimal(10, 1) = 0;
  loop $T in $Resolved {
    $Total = $Total + (cast(cast($T/ResolvedAt, datetime) - cast($T/System.changedDate, datetime), decimal) / 3600000000);
  }
  if count($Resolved) > 0 {
    $Stats/AvgResponseHours = $Total / count($Resolved);
  } else {
    $Stats/AvgResponseHours = 0;
  }

  return $Stats;
}
/
```

- [ ] **Step 2: Create Dashboard home page**

```mdl
create or replace page HD.Dashboard_Home
(
  title:  'Dashboard',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Dashboard'
)
{
  layoutgrid lgMain {
    row rHeader {
      column cTitle (desktopwidth: 12) {
        dynamictext txtTitle (content: 'HelpDesk Dashboard', rendermode: H2)
      }
    }
    row rStats {
      column cOpen (desktopwidth: 3) {
        dataview dvOpen (datasource: microflow HD.DS_DashboardStats) {
          dynamictext txtOpen (content: '{1}', contentparams: [{1} = OpenCount], rendermode: H1)
          dynamictext txtOpenLabel (content: 'Open Tickets', rendermode: Text)
        }
      }
      column cOverdue (desktopwidth: 3) {
        dataview dvOverdue (datasource: microflow HD.DS_DashboardStats) {
          dynamictext txtOverdue (content: '{1}', contentparams: [{1} = OverdueCount], rendermode: H1)
          dynamictext txtOverdueLabel (content: 'Overdue', rendermode: Text)
        }
      }
      column cNew (desktopwidth: 3) {
        dataview dvNew (datasource: microflow HD.DS_DashboardStats) {
          dynamictext txtNew (content: '{1}', contentparams: [{1} = NewTodayCount], rendermode: H1)
          dynamictext txtNewLabel (content: 'New Today', rendermode: Text)
        }
      }
      column cAvg (desktopwidth: 3) {
        dataview dvAvg (datasource: microflow HD.DS_DashboardStats) {
          dynamictext txtAvg (content: '{1}h', contentparams: [{1} = AvgResponseHours], rendermode: H1)
          dynamictext txtAvgLabel (content: 'Avg Response', rendermode: Text)
        }
      }
    }
    row rOverdue {
      column cOverdueList (desktopwidth: 12) {
        dynamictext txtOverdueTitle (content: '⚠ Overdue Alerts', rendermode: H3)
        datagrid dgOverdue (
          datasource: database from HD.Ticket
            where [IsOverSLA = true()]
            sort by SLADueAt asc,
          PageSize: 5
        ) {
          column colSubject  (attribute: Subject,          caption: 'Ticket')
          column colPriority (attribute: Priority,         caption: 'Priority', ColumnWidth: manual, Size: 100)
          column colSLADue   (attribute: SLADueAt,         caption: 'SLA Due',  ColumnWidth: manual, Size: 140)
          column colActions  (caption: '', ShowContentAs: customContent, ColumnWidth: manual, Size: 60) {
            actionbutton btnOpen (
              caption:     'Open',
              action:      show_page HD.Ticket_Detail (Ticket: $currentObject),
              buttonstyle: link
            )
          }
        }
      }
    }
    row rActions {
      column cActions (desktopwidth: 12) {
        dynamictext txtQuickTitle (content: 'Quick Actions', rendermode: H3)
        actionbutton btnNewTicket (
          caption:     'New Ticket',
          action:      create_object HD.Ticket then show_page HD.Ticket_NewEdit,
          buttonstyle: primary
        )
        actionbutton btnAllTickets (
          caption:     'All Tickets',
          action:      show_page HD.Ticket_Overview,
          buttonstyle: default
        )
        actionbutton btnKnowledgeBase (
          caption:     'Knowledge Base',
          action:      show_page KB.Article_Overview,
          buttonstyle: default
        )
      }
    }
  }
};
```

- [ ] **Step 3: Update navigation — Agent/Manager sees Dashboard first**

```mdl
create or replace navigation Responsive
  home page HD.MyTickets_Overview    for Customer
  home page HD.Dashboard_Home        for Agent
  home page HD.Dashboard_Home        for Manager
  home page HD.MyTickets_Overview
  login page Administration.login
  menu (
    menu item 'Dashboard'       page HD.Dashboard_Home;
    menu item 'My Tickets'      page HD.MyTickets_Overview;
    menu item 'All Tickets'     page HD.Ticket_Overview;
    menu item 'Knowledge Base'  page KB.Article_Overview;
  );
```

- [ ] **Step 4: Grant execute and view access**

```mdl
grant execute on microflow HD.DS_DashboardStats to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page HD.Dashboard_Home to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
```

- [ ] **Step 5: Verify 15-dashboard.mdl**

```bash
cd /mnt/data_sdb/mxcli && go run ./cmd/mdlrun -p HelpDeskE2E/HelpDeskE2E.mpr academy/zh/capstone-helpdesk/参考实现/15-dashboard.mdl
```

---

### Task 4: Brand Theme SCSS (16-brand-theme.mdl)

**Files:**
- Create: `academy/zh/capstone-helpdesk/参考实现/16-brand-theme.mdl`

**Interfaces:**
- Consumes: `HelpDeskE2E/theme/web/main.scss`
- Produces: Brand SCSS appended to main.scss

- [ ] **Step 1: Create 16-brand-theme.mdl that appends SCSS**

Note: This MDL file's exec runs the SCSS append client-side via the bridge.

```mdl
-- ============================================================
-- 模块 16：品牌主题
-- 前提：01-15 已执行
-- 注意：本模块在 mxcli 执行后将 SCSS 样式追加到 main.scss 文件
-- ============================================================

-- 品牌主题色、字体、状态/优先级配色通过 theme/web/main.scss 覆盖
-- Atlas_Core 主题变量实现。参考 academy/zh/11-主题扩展 的 theme 目录。
```

- [ ] **Step 2: Append brand SCSS to main.scss**

The validation script's existing `EXT_THEME_SRC` mechanism handles this. Add a comment block to the theme source file or create a dedicated brand scss file:

```bash
cat >> HelpDeskE2E/theme/web/main.scss << 'SCSS'

// ============================================================
// HelpDesk Brand Theme (模块 16)
// ============================================================

// Fonts
@import url('https://fonts.googleapis.com/css2?family=Lexend:wght@400;500;600;700&family=Source+Sans+3:wght@400;600&display=swap');

$font-family-headings: 'Lexend', sans-serif;
$font-family-body:     'Source Sans 3', sans-serif;

// Theme colors
$brand-primary: #0891B2;
$brand-secondary: #22D3EE;
$brand-success: #22C55E;
$brand-background: #ECFEFF;
$brand-text: #164E63;

// Status indicator colors
$status-draft: #94A3B8;
$status-open: #3B82F6;
$status-inprogress: #F59E0B;
$status-resolved: #22C55E;
$status-closed: #6B7280;

// Priority indicator colors
$priority-low: #94A3B8;
$priority-normal: #3B82F6;
$priority-high: #F97316;
$priority-critical: #EF4444;
SCSS
```

---

### Task 5: Wire New Files into Validation Script

**Files:**
- Modify: `scripts/validate-academy-capstone.sh`

- [ ] **Step 1: Add new MDL files to the exec list**

In the `mdl_files=(...)` array in `scripts/validate-academy-capstone.sh`, add after `12-integrate-actions.mdl`:
```bash
        "$CAPSTONE_DIR/13-improve-operations.mdl"
        "$CAPSTONE_DIR/14-beautify-pages.mdl"
        "$CAPSTONE_DIR/15-dashboard.mdl"
        "$CAPSTONE_DIR/16-brand-theme.mdl"
```

- [ ] **Step 2: Verify full validation**

```bash
cd /mnt/data_sdb/mxcli && rm -rf HelpDeskE2E && make sync-all 2>/dev/null && bash scripts/validate-academy-capstone.sh --from new 2>&1 | tail -20
```
Expected: `16/16 files succeeded, 0 failed`

- [ ] **Step 3: Commit all changes**

```bash
git add academy/zh/capstone-helpdesk/参考实现/1*-*.mdl scripts/validate-academy-capstone.sh
git commit -m "feat: capstone helpdesk refinement — operations, UI, dashboard, theme"
```
