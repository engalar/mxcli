# HelpDesk Seed Data + i18n Unified Design

**Date:** 2026-06-11  
**Files affected:**
- `mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl` (new)
- `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (modified: navigation + i18n section)

---

## Goals

1. **Seed data** — make the helpdesk app immediately runnable with realistic demo data after a single `mxcli exec helpdesk-seed.mdl`; accessible from a Manager navigation menu item.
2. **i18n unification** — make `helpdesk-app.mdl`'s multilingual demo internally consistent: English as the en_US base throughout (including navigation), comprehensive zh_CN translations for all user-visible enumerations and key page titles.

---

## Part 1 — `helpdesk-seed.mdl`

### Architecture

Single file, single microflow `HD.ACT_SeedDemoData()` with no parameters.  
Estimated size: ~130 lines.

**Execution flow:**

```
mxcli exec helpdesk-app.mdl -p app.mpr   # always first
mxcli exec helpdesk-seed.mdl -p app.mpr  # additive, idempotent
# start runtime → login as demo_manager → click "Initialize Demo Data"
```

### Microflow Structure

```
HD.ACT_SeedDemoData()  folder 'System'
│
├── 1. Idempotent guard
│     retrieve $existing from HD.Ticket limit 1;
│     if $existing != empty { return; }
│
├── 2. KB data (explicit creates)
│     KB.Category × 2  (General, Troubleshooting)
│     KB.Article  × 4  (1 Draft, 3 Published — linked to categories)
│     KB.Tag      × 2  (howto, faq)
│     KB.ArticleTag × 4 (junction: each Published article gets one tag)
│     → commit all
│
├── 3. HD users (explicit creates)
│     HD.Customer × 3  (diverse names + companies)
│     HD.Agent    × 2  (Alice Smith, Bob Jones)
│     → commit all
│
├── 4. Ticket creation (5 explicit creates, one per status)
│     T1: Subject='Cannot login'         Status=Draft     Priority=Low     Customer=$Cust1
│     T2: Subject='Slow response'        Status=Open      Priority=Normal   Customer=$Cust2
│     T3: Subject='Payment error'        Status=InProgress Priority=High   Customer=$Cust1  Agent=$Agent1
│     T4: Subject='Data export broken'   Status=Resolved  Priority=Critical Customer=$Cust3  IsOverSLA=true
│     T5: Subject='UI display glitch'    Status=Closed    Priority=Normal   Customer=$Cust2
│     → commit all
│
├── 5. Loop: add one TicketComment per Ticket
│     retrieve $Tickets from HD.Ticket limit 10;
│     loop $T in $Tickets {
│       $C = create HD.TicketComment (
│         Content    = 'Demo comment for: ' + $T/Subject,
│         IsInternal = false,
│         HD.TicketComment_Ticket = $T
│       );
│       commit $C;
│     }
│
├── 6. EscalationRequest for T3 (InProgress)
│     $ER = create HD.EscalationRequest (
│       Reason      = 'Customer threatening chargeback — needs manager review.',
│       RequestedAt = '[%CurrentDateTime%]',
│       HD.EscalationRequest_Ticket = $T3
│     );
│     commit $ER;
│
├── 7. FT data
│     FT.FieldTech    × 2  (Carlos Rivera IsAvailable=true, Dana Park IsAvailable=false)
│     FT.DispatchOrder × 1 (Status=OnSite, linked to $T3 + FT1)
│     → commit all
│
└── 8. return;
```

### Security & Navigation

```mdl
grant execute on microflow HD.ACT_SeedDemoData to HD.ManagerRole;

-- Rebuild Responsive navigation with English base + seed menu item
create or replace navigation Responsive
  home page HD.Ticket_Overview for Customer
  home page HD.Ticket_Overview for Agent
  home page HD.EscalationWorkflow_Overview for Manager
  home page MyFirstModule.Home_Web
  login page Administration.login
  menu (
    menu item 'My Tickets'       page HD.MyTickets_Overview;
    menu item 'Knowledge Base'   page KB.Article_Overview;
    menu 'Ticket Management' (
      menu item 'All Tickets'    page HD.Ticket_Overview;
      menu item 'Escalations'    page HD.EscalationWorkflow_Overview;
      menu item 'Workflow Admin' page HD.EscalationWorkflow_Overview;
    );
    menu 'Account Management' (
      menu item 'Accounts'         page HD.Account_Overview;
      menu item 'My Account'       page HD.ManageMyAccount;
      menu item 'Change Password'  microflow HD.Nav_ChangePassword;
    );
    menu 'Admin' (
      menu item 'User Management'    page Administration.Account_Overview;
      menu item 'Initialize Demo Data' microflow HD.ACT_SeedDemoData;
    );
  );
```

The seed navigation redefines the profile in English (matching the i18n unification in Part 2), replacing the Chinese-base navigation set by `helpdesk-app.mdl`.

---

## Part 2 — `helpdesk-app.mdl` changes

### 2a. Navigation Section (lines ~2158–2183)

Change all menu item captions from hardcoded Chinese to English. This makes en_US the consistent base language, with zh_CN translations applied via the `translate` commands in the i18n demo section.

**Before → After mapping:**

| Before (Chinese) | After (English) |
|------------------|-----------------|
| `'我的工单'` | `'My Tickets'` |
| `'知识库'` | `'Knowledge Base'` |
| `'工单管理'` | `'Ticket Management'` |
| `'所有工单'` | `'All Tickets'` |
| `'升级审批'` | `'Escalations'` |
| `'工作流管理'` | `'Workflow Admin'` |
| `'账户管理'` | `'Account Management'` |
| `'账户总览'` | `'Accounts'` |
| `'我的账户'` | `'My Account'` |
| `'修改密码'` | `'Change Password'` |
| `'系统管理'` | `'Admin'` |

### 2b. i18n Demo Section (lines ~2270–2337)

**Expand zh_CN coverage** to include all user-visible enumerations and the key pages a demo user will land on.

**New translations to add:**

```mdl
translate enumeration KB.ArticleStatus in zh_CN
  set Draft.caption     = '草稿',
      Published.caption = '已发布',
      Archived.caption  = '已归档';

translate enumeration FT.DispatchStatus in zh_CN
  set Pending.caption   = '待派遣',
      EnRoute.caption   = '前往中',
      OnSite.caption    = '到场',
      Completed.caption = '已完成',
      Cancelled.caption = '已取消';
```

**Expand page title translations:**

```mdl
translate page HD.Ticket_Overview in zh_CN
  set title = '工单列表';

translate page KB.Article_Overview in zh_CN
  set title = '知识库';

translate page HD.Account_Overview in zh_CN
  set title = '账户管理';

translate page FT.SLADashboard in zh_CN
  set title = 'SLA 仪表盘';
```

**Navigation caption translation:**

`translate navigation` is not currently implemented in mxcli. Navigation caption translation is documented as a comment in the i18n section with a TODO. Menu item zh_CN captions are visible only when the user's browser locale is zh_CN; the English base is always shown otherwise.

**Final language state (unchanged):** en_US (default) + zh_CN. The nl_NL/fr_FR add-then-drop demo pattern is retained as-is.

**Updated i18n section comment header** to reflect the expanded coverage:

```
-- Patterns covered in order:
--   1. alter settings language add 'code'          plain add
--   2. alter settings language add 'code' (checkCompleteness: true)
--   3. show languages
--   4. translate enumeration … in lang set …       3 enumerations: TicketStatus, TicketPriority,
--                                                   ArticleStatus, DispatchStatus
--   5. translate page … in lang set …              5 page titles
--   6. alter settings language drop 'code'         single language (no translations)
--   7. alter settings language drop 'code'         language with translations
--   8. describe settings
-- Final MPR state: en_US (default) + zh_CN with translations.
```

---

## Validation

After both changes:

```bash
# Syntax check
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl

# Full execution against testdata
./bin/mxcli exec mdl-examples/use-cases/helpdesk/helpdesk-app.mdl \
  -p testdata/helpdesk-clean-11.6.6/app.mpr
./bin/mxcli exec mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl \
  -p testdata/helpdesk-clean-11.6.6/app.mpr

# mx check — must produce 0 new StorageLoadException
~/.mxcli/mxbuild/11.6.6/modeler/mx check \
  testdata/helpdesk-clean-11.6.6/app.mpr 2>&1 | grep -i "StorageLoadException\|Invalid"
```

The seed microflow itself is not executed at `mxcli exec` time — it must be triggered from the app UI by a Manager user. The `mx check` step only validates the BSON structure.

---

## Non-Goals

- No changes to golden snapshot files (seed data is runtime-only, not part of MPR BSON)
- No `translate navigation` implementation (out of scope; left as comment/TODO)
- No FT navigation profile changes (OfflineNative remains commented-out)
