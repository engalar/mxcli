# Helpdesk Seed Data + i18n Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `helpdesk-seed.mdl` file with idempotent demo-data seeding via a Manager menu item, and unify `helpdesk-app.mdl`'s i18n demo to use English as the en_US base with expanded zh_CN translations.

**Architecture:** Two pure-MDL deliverables — no Go code changes. `helpdesk-app.mdl` gets English navigation captions and more `translate` statements. `helpdesk-seed.mdl` defines a single `HD.ACT_SeedDemoData` microflow (idempotent, loop-based comment creation) plus a security grant and updated navigation.

**Tech Stack:** MDL (Mendix Definition Language), `mxcli check` (syntax), `mx check` (BSON), `TestHelpdeskGolden_Update` (golden rebuild)

---

## File Map

| Action | Path |
|--------|------|
| Modify | `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` |
| Modify | `cmd/mxcli/examples/helpdesk-app.mdl` (identical copy — must stay in sync) |
| Create | `mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl` |

**Both `helpdesk-app.mdl` copies must receive identical edits.** Verify with `diff mdl-examples/use-cases/helpdesk/helpdesk-app.mdl cmd/mxcli/examples/helpdesk-app.mdl` after each task — expected: no output.

---

## Task 1: helpdesk-app.mdl — Navigation English captions

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (navigation section ~lines 2169-2183)
- Modify: `cmd/mxcli/examples/helpdesk-app.mdl` (same section)

- [ ] **Step 1: Edit navigation in `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`**

Replace the menu item block (inside `create or replace navigation Responsive`):

**Old (lines 2169–2183):**
```mdl
    menu item '我的工单'  page HD.MyTickets_Overview;
    menu item '知识库'    page KB.Article_Overview;
    menu '工单管理' (
      menu item '所有工单'  page HD.Ticket_Overview;
      menu item '升级审批'  page HD.EscalationWorkflow_Overview;
      menu item '工作流管理' page HD.EscalationWorkflow_Overview;
    );
    menu '账户管理' (
      menu item '账户总览'  page HD.Account_Overview;
      menu item '我的账户'  page HD.ManageMyAccount;
      -- Issue 5 fix: HD.ChangeMyPassword requires $Form parameter; use no-param microflow as entry
      menu item '修改密码'  microflow HD.Nav_ChangePassword;
    );
    menu item '系统管理' page Administration.Account_Overview;
```

**New:**
```mdl
    menu item 'My Tickets'     page HD.MyTickets_Overview;
    menu item 'Knowledge Base' page KB.Article_Overview;
    menu 'Ticket Management' (
      menu item 'All Tickets'    page HD.Ticket_Overview;
      menu item 'Escalations'    page HD.EscalationWorkflow_Overview;
      menu item 'Workflow Admin' page HD.EscalationWorkflow_Overview;
    );
    menu 'Account Management' (
      menu item 'Accounts'        page HD.Account_Overview;
      menu item 'My Account'      page HD.ManageMyAccount;
      -- Issue 5 fix: HD.ChangeMyPassword requires $Form parameter; use no-param microflow as entry
      menu item 'Change Password' microflow HD.Nav_ChangePassword;
    );
    menu item 'Admin' page Administration.Account_Overview;
```

- [ ] **Step 2: Apply the same edit to `cmd/mxcli/examples/helpdesk-app.mdl`**

Same old → new replacement as Step 1.

- [ ] **Step 3: Verify both files are identical**

```bash
diff mdl-examples/use-cases/helpdesk/helpdesk-app.mdl cmd/mxcli/examples/helpdesk-app.mdl
```
Expected: no output.

- [ ] **Step 4: Syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```
Expected: `OK` with 0 errors.

- [ ] **Step 5: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl cmd/mxcli/examples/helpdesk-app.mdl
git commit -m "fix(helpdesk-app): navigation captions to English (en_US base for i18n)"
```

---

## Task 2: helpdesk-app.mdl — Expand zh_CN i18n translations

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (i18n section ~lines 2272–2333)
- Modify: `cmd/mxcli/examples/helpdesk-app.mdl` (same section)

- [ ] **Step 1: Update the i18n section comment header in `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`**

**Old:**
```mdl
-- Patterns covered in order:
--   1. alter settings language add 'code'               plain add
--   2. alter settings language add 'code' (checkCompleteness: true)
--   3. show languages                                   inspect all registered
--   4. translate enumeration … in lang set …            enum caption translation
--   5. translate page … in lang set …                   page widget translation
--   6. alter settings language drop 'code'              单个语言的删除 (no translations)
--   7. alter settings language drop 'code'              整个语言的删除 (with translations)
--   8. describe settings                                 re-executable language state
-- Final MPR state: en_US (default) + zh_CN with translations.
```

**New:**
```mdl
-- Patterns covered in order:
--   1. alter settings language add 'code'               plain add
--   2. alter settings language add 'code' (checkCompleteness: true)
--   3. show languages                                   inspect all registered
--   4. translate enumeration … in lang set …            4 enumerations (zh_CN):
--                                                        TicketStatus, TicketPriority,
--                                                        ArticleStatus, DispatchStatus
--   5. translate page … in lang set …                   5 page titles (zh_CN)
--   6. alter settings language drop 'code'              单个语言的删除 (no translations)
--   7. alter settings language drop 'code'              整个语言的删除 (with translations)
--   8. describe settings                                 re-executable language state
-- Final MPR state: en_US (default) + zh_CN with translations.
```

- [ ] **Step 2: Insert new zh_CN translations after the existing page translation**

Find the block ending with:
```mdl
translate page HD.Ticket_NewEdit in zh_CN
  set title = '新建/编辑工单';

-- Step 4: Apply nl_NL translations (partial — checkCompleteness will flag the rest).
```

Insert between those two blocks:
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

translate page HD.Ticket_Overview in zh_CN
  set title = '工单列表';

translate page KB.Article_Overview in zh_CN
  set title = '知识库';

translate page HD.Account_Overview in zh_CN
  set title = '账户管理';

translate page FT.SLADashboard in zh_CN
  set title = 'SLA 仪表盘';

```

So the full replacement for that boundary:
**Old:**
```mdl
translate page HD.Ticket_NewEdit in zh_CN
  set title = '新建/编辑工单';

-- Step 4: Apply nl_NL translations (partial — checkCompleteness will flag the rest).
```

**New:**
```mdl
translate page HD.Ticket_NewEdit in zh_CN
  set title = '新建/编辑工单';

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

translate page HD.Ticket_Overview in zh_CN
  set title = '工单列表';

translate page KB.Article_Overview in zh_CN
  set title = '知识库';

translate page HD.Account_Overview in zh_CN
  set title = '账户管理';

translate page FT.SLADashboard in zh_CN
  set title = 'SLA 仪表盘';

-- Step 4: Apply nl_NL translations (partial — checkCompleteness will flag the rest).
```

- [ ] **Step 3: Apply the same edits to `cmd/mxcli/examples/helpdesk-app.mdl`**

Identical old → new replacements as Steps 1 and 2.

- [ ] **Step 4: Verify both files are identical**

```bash
diff mdl-examples/use-cases/helpdesk/helpdesk-app.mdl cmd/mxcli/examples/helpdesk-app.mdl
```
Expected: no output.

- [ ] **Step 5: Syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```
Expected: `OK` with 0 errors.

- [ ] **Step 6: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl cmd/mxcli/examples/helpdesk-app.mdl
git commit -m "feat(helpdesk-app): expand zh_CN translations — ArticleStatus, DispatchStatus, 4 page titles"
```

---

## Task 3: Create helpdesk-seed.mdl

**Files:**
- Create: `mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl`

- [ ] **Step 1: Create the file with the following content**

```mdl
-- ============================================================================
-- HelpDesk Seed Data
-- ============================================================================
--
-- Purpose : Populate the helpdesk app with realistic demo data for all roles.
-- Requires: helpdesk-app.mdl executed first (entities, roles, navigation).
-- Trigger : Manager menu item "Initialize Demo Data" (HD.ACT_SeedDemoData).
-- Idempotent: returns immediately if HD.Ticket rows already exist.
--
-- Usage:
--   ./bin/mxcli exec mdl-examples/use-cases/helpdesk/helpdesk-app.mdl -p app.mpr
--   ./bin/mxcli exec mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl -p app.mpr
--   # Start runtime, login as demo_manager, click "Initialize Demo Data"
-- ============================================================================

create or modify microflow HD.ACT_SeedDemoData ()
  returns boolean as $OK
  folder 'System'
{
  -- Idempotent guard: skip if data already exists.
  retrieve $existing from HD.Ticket limit 1;
  if $existing != empty {
    return true;
  }

  -- KB Categories
  $Cat1 = create KB.Category (Name = 'Getting Started',  Description = 'Guides and tutorials for new users.');
  $Cat2 = create KB.Category (Name = 'Troubleshooting',  Description = 'Common issues and step-by-step fixes.');
  commit $Cat1;
  commit $Cat2;

  -- KB Tags
  $Tag1 = create KB.Tag (Name = 'howto');
  $Tag2 = create KB.Tag (Name = 'faq');
  commit $Tag1;
  commit $Tag2;

  -- KB Articles (1 Draft + 3 Published)
  $Art1 = create KB.Article (
    Title       = 'How to submit a support ticket',
    Content     = 'Navigate to Ticket Management > All Tickets, then click New Ticket. Fill in Subject, Description, and Priority, then click Submit.',
    Status      = KB.ArticleStatus.Published,
    PublishedAt = '[%CurrentDateTime%]',
    KB.Article_Category = $Cat1
  );
  $Art2 = create KB.Article (
    Title       = 'Resetting your password',
    Content     = 'Go to Account Management > Change Password. Enter your new password twice and click Change Password.',
    Status      = KB.ArticleStatus.Published,
    PublishedAt = '[%CurrentDateTime%]',
    KB.Article_Category = $Cat1
  );
  $Art3 = create KB.Article (
    Title       = 'Common login errors and fixes',
    Content     = 'Clear browser cache, disable extensions, and try incognito mode. If the problem persists, contact support.',
    Status      = KB.ArticleStatus.Published,
    PublishedAt = '[%CurrentDateTime%]',
    KB.Article_Category = $Cat2
  );
  $Art4 = create KB.Article (
    Title       = 'SLA policy overview',
    Content     = 'Draft — under review by the support team.',
    Status      = KB.ArticleStatus.Draft,
    KB.Article_Category = $Cat2
  );
  commit $Art1;
  commit $Art2;
  commit $Art3;
  commit $Art4;

  -- ArticleTag junction records (many-to-many)
  $AT1 = create KB.ArticleTag (KB.ArticleTag_Article = $Art1, KB.ArticleTag_Tag = $Tag1);
  $AT2 = create KB.ArticleTag (KB.ArticleTag_Article = $Art2, KB.ArticleTag_Tag = $Tag2);
  $AT3 = create KB.ArticleTag (KB.ArticleTag_Article = $Art3, KB.ArticleTag_Tag = $Tag2);
  $AT4 = create KB.ArticleTag (KB.ArticleTag_Article = $Art3, KB.ArticleTag_Tag = $Tag1);
  commit $AT1;
  commit $AT2;
  commit $AT3;
  commit $AT4;

  -- HD Customers
  $Cust1 = create HD.Customer (Name = 'Alice Tan',  Email = 'alice@acme.com',    Company = 'Acme Corp');
  $Cust2 = create HD.Customer (Name = 'Bob Lee',    Email = 'bob@globex.com',    Company = 'Globex Inc');
  $Cust3 = create HD.Customer (Name = 'Carol Wu',   Email = 'carol@initech.com', Company = 'Initech Ltd');
  commit $Cust1;
  commit $Cust2;
  commit $Cust3;

  -- HD Agents
  $Agent1 = create HD.Agent (Name = 'Alice Smith', Email = 'alice.smith@helpdesk.internal', IsActive = true);
  $Agent2 = create HD.Agent (Name = 'Bob Jones',   Email = 'bob.jones@helpdesk.internal',  IsActive = true);
  commit $Agent1;
  commit $Agent2;

  -- HD Tickets — one per status, covering all five states
  $T1 = create HD.Ticket (
    Subject     = 'Cannot login to portal',
    Description = 'Getting 403 error on the login page after recent password change.',
    Status      = HD.TicketStatus.Draft,
    Priority    = HD.TicketPriority.Low,
    HD.Ticket_Customer = $Cust1
  );
  $T2 = create HD.Ticket (
    Subject     = 'Dashboard loads slowly',
    Description = 'Main dashboard takes 30 seconds to render. Happens on all browsers.',
    Status      = HD.TicketStatus.Open,
    Priority    = HD.TicketPriority.Normal,
    SLADueAt    = addHours('[%CurrentDateTime%]', 24),
    HD.Ticket_Customer = $Cust2
  );
  $T3 = create HD.Ticket (
    Subject     = 'Payment error on checkout',
    Description = 'Card declined for a valid Visa card. Error code: PAY_003.',
    Status      = HD.TicketStatus.InProgress,
    Priority    = HD.TicketPriority.High,
    SLADueAt    = addHours('[%CurrentDateTime%]', 8),
    HD.Ticket_Customer  = $Cust1,
    HD.Ticket_Agent     = $Agent1
  );
  $T4 = create HD.Ticket (
    Subject     = 'Data export returns empty file',
    Description = 'CSV export from the report page downloads a 0-byte file.',
    Status      = HD.TicketStatus.Resolved,
    Priority    = HD.TicketPriority.Critical,
    SLADueAt    = '[%CurrentDateTime%]',
    ResolvedAt  = '[%CurrentDateTime%]',
    IsOverSLA   = true,
    HD.Ticket_Customer  = $Cust3,
    HD.Ticket_Agent     = $Agent2
  );
  $T5 = create HD.Ticket (
    Subject     = 'Table columns misaligned in Firefox',
    Description = 'The ticket list table header does not align with rows in Firefox 124.',
    Status      = HD.TicketStatus.Closed,
    Priority    = HD.TicketPriority.Normal,
    HD.Ticket_Customer = $Cust2
  );
  commit $T1;
  commit $T2;
  commit $T3;
  commit $T4;
  commit $T5;

  -- Add one TicketComment per Ticket via loop (avoids repeating 5 identical create+commit blocks)
  retrieve $AllTickets from HD.Ticket limit 10;
  loop $T in $AllTickets {
    $C = create HD.TicketComment (
      Content    = 'Demo comment — please review this ticket: ' + $T/Subject,
      IsInternal = false,
      HD.TicketComment_Ticket = $T
    );
    commit $C;
  }

  -- EscalationRequest for T3 (InProgress / High — realistic escalation scenario)
  $ER = create HD.EscalationRequest (
    Reason          = 'Customer threatening chargeback. Payment team cannot reproduce locally. Manager review required.',
    RequestedAt     = '[%CurrentDateTime%]',
    HD.EscalationRequest_Ticket = $T3
  );
  commit $ER;

  -- FT FieldTech records
  $FT1 = create FT.FieldTech (Name = 'Carlos Rivera', Phone = '+1-555-0101', Region = 'West', IsAvailable = true);
  $FT2 = create FT.FieldTech (Name = 'Dana Park',     Phone = '+1-555-0202', Region = 'East', IsAvailable = false);
  commit $FT1;
  commit $FT2;

  -- DispatchOrder: FT1 currently on-site for T3's physical investigation
  $DO1 = create FT.DispatchOrder (
    Status       = FT.DispatchStatus.OnSite,
    DispatchedAt = '[%CurrentDateTime%]',
    SiteNotes    = 'On-site payment terminal inspection in progress.',
    FT.DispatchOrder_Ticket    = $T3,
    FT.DispatchOrder_FieldTech = $FT1
  );
  commit $DO1;

  return true;
}
/

grant execute on microflow HD.ACT_SeedDemoData to HD.ManagerRole;

-- Rebuild Responsive navigation with English base captions + seed menu item.
-- This replaces the navigation set by helpdesk-app.mdl; the only change is
-- converting '系统管理' to a submenu 'Admin' with a second item.
create or replace navigation Responsive
  home page HD.Ticket_Overview for Customer
  home page HD.Ticket_Overview for Agent
  home page HD.EscalationWorkflow_Overview for Manager
  home page MyFirstModule.Home_Web
  login page Administration.login
  menu (
    menu item 'My Tickets'     page HD.MyTickets_Overview;
    menu item 'Knowledge Base' page KB.Article_Overview;
    menu 'Ticket Management' (
      menu item 'All Tickets'    page HD.Ticket_Overview;
      menu item 'Escalations'    page HD.EscalationWorkflow_Overview;
      menu item 'Workflow Admin' page HD.EscalationWorkflow_Overview;
    );
    menu 'Account Management' (
      menu item 'Accounts'        page HD.Account_Overview;
      menu item 'My Account'      page HD.ManageMyAccount;
      menu item 'Change Password' microflow HD.Nav_ChangePassword;
    );
    menu 'Admin' (
      menu item 'User Management'      page Administration.Account_Overview;
      menu item 'Initialize Demo Data' microflow HD.ACT_SeedDemoData;
    );
  );
```

- [ ] **Step 2: Syntax check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl
```
Expected: `OK` with 0 errors. If you see "unknown entity" or reference errors, that is normal — syntax-only check does not resolve cross-file references.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl
git commit -m "feat(helpdesk): add helpdesk-seed.mdl — idempotent demo data via Manager menu"
```

---

## Task 4: Integration test — exec both files + mx check

**Files:** none modified (restore testdata after testing)

- [ ] **Step 1: Build the CLI**

```bash
make build
```

- [ ] **Step 2: Exec helpdesk-app.mdl against testdata**

```bash
./bin/mxcli exec mdl-examples/use-cases/helpdesk/helpdesk-app.mdl \
  -p testdata/helpdesk-clean-11.6.6/minimal.mpr
```
Expected: completes without error.

- [ ] **Step 3: Exec helpdesk-seed.mdl against the same testdata**

```bash
./bin/mxcli exec mdl-examples/use-cases/helpdesk/helpdesk-seed.mdl \
  -p testdata/helpdesk-clean-11.6.6/minimal.mpr
```
Expected: completes without error. This writes `HD.ACT_SeedDemoData` BSON, the grant, and updated navigation into the MPR.

- [ ] **Step 4: mx check — verify 0 new StorageLoadException**

```bash
~/.mxcli/mxbuild/11.6.6/modeler/mx check \
  testdata/helpdesk-clean-11.6.6/minimal.mpr 2>&1 | grep "\[error\]" | wc -l
```
Expected: `0` (same baseline error count as before). If you see new `[error]` lines, compare against:
```bash
git stash && ./bin/mxcli exec mdl-examples/use-cases/helpdesk/helpdesk-app.mdl \
  -p testdata/helpdesk-clean-11.6.6/minimal.mpr
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/helpdesk-clean-11.6.6/minimal.mpr 2>&1 | grep "\[error\]" | wc -l
git stash pop
```
Any delta means the new seed navigation or translations introduced a BSON error — debug with the `debug-bson.md` skill.

- [ ] **Step 5: Restore testdata**

```bash
git restore testdata/helpdesk-clean-11.6.6/
git clean -fd testdata/helpdesk-clean-11.6.6/
```

---

## Task 5: Rebuild golden snapshot

**Why:** The navigation unit BSON in `helpdesk-app.mdl` changed (English captions, different menu structure). The golden file at `testdata/helpdesk-golden-11.6.6/` must be regenerated to reflect the new expected state. Without this, `TestHelpdeskGolden_Regression_BSON` will fail on CI.

**Files:**
- Regenerated by test: `testdata/helpdesk-golden-11.6.6/` (all mxunit files)
- Regenerated by test: `testdata/helpdesk-golden-11.6.6/describe-snapshot.mdl`
- Regenerated by test: `testdata/helpdesk-clean-11.6.6/describe-snapshot.mdl`

- [ ] **Step 1: Rebuild golden (runs in-process, no install-daemon needed)**

```bash
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run '^TestHelpdeskGolden_Update$' \
  -update-golden \
  -v -timeout 10m
```
Expected output ends with:
```
Golden updated: testdata/helpdesk-golden-11.6.6
Next step: git add testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-clean-11.6.6/describe-snapshot.mdl && git commit
PASS
```

- [ ] **Step 2: mx check the rebuilt golden**

```bash
~/.mxcli/mxbuild/11.6.6/modeler/mx check \
  testdata/helpdesk-golden-11.6.6/minimal.mpr 2>&1 | grep "\[error\]" | wc -l
```
Expected: `0`. If you see new errors, the navigation or translation BSON is malformed — revert the golden rebuild (`git restore testdata/`) and debug with `go run ./cmd/mdlrun`.

- [ ] **Step 3: Run the regression test against the new golden**

```bash
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run '^TestHelpdeskGolden_Regression_BSON$' \
  -v -timeout 10m
```
Expected: `PASS` (B2 matches B1, which is the golden we just rebuilt).

- [ ] **Step 4: Commit the updated golden**

```bash
git add testdata/helpdesk-golden-11.6.6/ testdata/helpdesk-clean-11.6.6/describe-snapshot.mdl
git commit -m "chore(golden): rebuild helpdesk-golden-11.6.6 — English navigation + expanded zh_CN"
```

---

## Self-Review Notes

**Spec coverage:**
- ✅ `helpdesk-seed.mdl` created with single microflow (Task 3)
- ✅ Idempotent guard via `retrieve HD.Ticket limit 1` (Task 3, step 1)
- ✅ KB data: 2 Category, 4 Article, 2 Tag, 4 ArticleTag (Task 3)
- ✅ HD users: 3 Customer, 2 Agent (Task 3)
- ✅ 5 Tickets covering all statuses (Task 3)
- ✅ Loop for TicketComment creation (Task 3)
- ✅ EscalationRequest for InProgress ticket (Task 3)
- ✅ FT data: 2 FieldTech, 1 DispatchOrder (Task 3)
- ✅ Grant to HD.ManagerRole (Task 3)
- ✅ Navigation menu item "Initialize Demo Data" (Task 3)
- ✅ Navigation English captions in helpdesk-app.mdl (Task 1)
- ✅ KB.ArticleStatus zh_CN translations (Task 2)
- ✅ FT.DispatchStatus zh_CN translations (Task 2)
- ✅ 4 additional page title translations (Task 2)
- ✅ Both helpdesk-app.mdl copies updated (Tasks 1 & 2)
- ✅ Golden rebuild (Task 5)

**No-Go code changes** — this is pure MDL, no unit tests to write. Validation is mx check + golden regression.
