# Helpdesk Account Management — MDL Extension Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` with an Account Management section that demonstrates 12 new MDL activity types across 9 requirements (R1–R9).

**Architecture:** All changes are purely additive MDL edits to one file. New content is inserted as a `-- MARK: Account Management` block before `-- MARK: Security — Module Roles` (line 1433). Security grants, navigation, and MOVE commands are appended to their existing sections. No Go code is modified.

**Tech Stack:** MDL (Mendix Definition Language), `mxcli check` for syntax validation, `make build` to rebuild the binary before validation.

**Spec:** `docs/superpowers/specs/2026-06-05-helpdesk-account-management-design.md`

---

## File Map

| File | Action | What changes |
|------|--------|-------------|
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` line 171–174 | Modify | Add `DeactivationReason` field to `HD.Agent` |
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` line 1432 | Insert after | New `-- MARK: Account Management` block (entities + microflows + pages) |
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` line 1546 | Insert after | Security grants for new pages + microflows |
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` line 1583 | Modify | Add `menu '账户管理'` group to navigation |
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` line 1653 | Insert after | MOVE commands for new Account microflows + pages |
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` line 1–35 | Modify | Add Note about Account Management section to file header |

---

## Validation Command

Used after every task:

```bash
make build && ./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
```

Expected output: `0 errors` (or the same pre-existing error count as baseline).

---

## Task 1: Verify Baseline

**Files:** `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` (read-only)

- [ ] **Step 1: Build and capture baseline error count**

```bash
make build
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | tee /tmp/helpdesk-baseline.txt
grep -c "\[error\]" /tmp/helpdesk-baseline.txt || echo "0 errors"
```

Expected: command exits cleanly. Note the exact error count — every subsequent task must not increase it.

---

## Task 2: Extend HD.Agent + Add HD.UserProfile and HD.PasswordForm

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:171-174`
- Insert after line 1432

**Why:** `HD.UserProfile extends System.User` enables the R1 `split type` + `cast` pattern. `HD.PasswordForm` is the non-persistent form holder for R2/R3 password pages. `DeactivationReason` on `HD.Agent` is needed for R8 conditional visibility.

- [ ] **Step 1: Add `DeactivationReason` to the existing HD.Agent definition at line 171**

Find this block (lines 171–174):
```mdl
create or modify persistent entity HD.Agent (
  Name:     string(200) not null,
  Email:    string(200) not null,
  IsActive: boolean default true
);
```

Replace with:
```mdl
create or modify persistent entity HD.Agent (
  Name:               string(200) not null,
  Email:              string(200) not null,
  IsActive:           boolean default true,
  DeactivationReason: string(500)
);
```

- [ ] **Step 2: Insert the Account Management entity block after line 1432 (the closing `};` of HD.TicketSearch_Results)**

Insert the following between the end of `HD.TicketSearch_Results` and `-- MARK: Security — Module Roles`:

```mdl

-- ============================================================================
-- MARK: Account Management — Entities, Microflows & Pages
-- ============================================================================
--
-- Patterns introduced in this section (not demonstrated elsewhere in this file):
--   Entities   : entity generalization (extends System.User)
--   Microflows : split type (InheritanceSplitStatement), cast (CastObjectStmt),
--                create java action, call java action, raise error,
--                show home page
--   Pages      : DataView with microflow datasource (DataView, not DataGrid),
--                tabcontainer + tabpage, ShowContentAs:customContent action button
--                column, nested DataView (association datasource),
--                conditional visibility (visible: [XPath])
--
-- Requirement coverage:
--   R1 ManageMyAccount  → HD.UserProfile + HD.DS_GetMyProfile + HD.ManageMyAccount
--   R2 ChangeMyPassword → HD.JA_HashPassword + HD.ACT_ChangePassword + HD.ChangeMyPassword
--   R3 ShowMyPasswordForm → HD.ShowMyPasswordForm (popup variant, no new activities)
--   R4 Account_Overview column filters → HD.Account_Overview datagrid columns
--   R5 Account_Overview action buttons → HD.Account_Overview customContent columns
--   R6 Account_Overview tabs           → HD.Account_Overview tabcontainer
--   R7 Account_New nested DataView     → HD.Customer_New
--   R8 Account_Edit conditional (Agent)    → HD.Agent_Edit
--   R9 Account_Edit conditional (Customer) → HD.Customer_Edit
-- ============================================================================

-- MARK: Account Management — Entities

-- R1: generalization of System.User — enables split type + cast in DS_GetMyProfile
create or modify persistent entity HD.UserProfile extends System.User (
  DisplayName: string(200)
);

-- R2/R3: non-persistent form holder for password change pages
create or modify non-persistent entity HD.PasswordForm (
  NewPassword:     string,
  ConfirmPassword: string
);
```

- [ ] **Step 3: Validate**

```bash
make build && ./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

Expected: same count as baseline.

- [ ] **Step 4: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add HD.UserProfile, HD.PasswordForm entities and DeactivationReason field"
```

---

## Task 3: Add R1 Microflow — HD.DS_GetMyProfile (split type + cast)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` — append to Account Management MARK block

**Why:** Demonstrates `split type` (InheritanceSplitStatement) and `cast` (CastObjectStmt) — the two new activities required for the ManageMyAccount pattern. Also demonstrates DataView with microflow datasource when used in Task 6.

- [ ] **Step 1: Append after the HD.PasswordForm entity definition**

```mdl

-- MARK: Account Management — Microflows

-- R1: DS_GetMyProfile
-- New activities: split type (route by runtime type) + cast (bind subtype variable)
-- Retrieves [%CurrentUser%] as System.User, splits by runtime type to HD.UserProfile,
-- casts to $Profile, and returns it. Used as DataView datasource in HD.ManageMyAccount.
create or modify microflow HD.DS_GetMyProfile
  ()
  returns HD.UserProfile as $Profile
  folder 'Account'
begin
  retrieve $Me from System.User
    where [id = '[%CurrentUser%]']
    limit 1;
  split type $Me
  case HD.UserProfile
    cast $Profile;
    return $Profile;
  else
    return empty;
  end split;
end;
/
```

- [ ] **Step 2: Validate**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

Expected: same count as baseline.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add HD.DS_GetMyProfile microflow (split type + cast)"
```

---

## Task 4: Add R2 Java Action + Microflow — HD.JA_HashPassword + HD.ACT_ChangePassword

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` — append to Account Management MARK block

**Why:** Demonstrates `create java action` (interface declaration), `call java action`, `raise error` (ErrorEvent path, distinct from validation feedback), and `show home page` (client navigation after security operation).

- [ ] **Step 1: Append after HD.DS_GetMyProfile**

```mdl

-- R2: HD.JA_HashPassword — Java Action definition
-- New pattern: create java action (declares interface; Java implementation loaded at runtime)
-- No source block needed — the definition alone demonstrates the MDL syntax.
create or modify java action HD.JA_HashPassword (
  PlainText: string
) returns string
folder 'Account';

-- R2: HD.ACT_ChangePassword
-- New activities:
--   call java action — invokes HD.JA_HashPassword to hash the new password
--   raise error      — terminates to ErrorEvent when passwords do not match
--                      (distinct from validation feedback: no field highlight, hard stop)
--   show home page   — redirects after successful password change (clears form state)
create or modify microflow HD.ACT_ChangePassword
  ($Form: HD.PasswordForm)
  returns boolean as $Success
  folder 'Account'
begin
  if $Form/NewPassword = '' or $Form/NewPassword = empty then
    validation feedback $Form/NewPassword message 'New password is required.';
    return false;
  end if;
  if $Form/NewPassword != $Form/ConfirmPassword then
    raise error;
  end if;
  $Hashed = call java action HD.JA_HashPassword (PlainText = $Form/NewPassword);
  set $Success = true;
  show home page;
  return $Success;
end;
/
```

- [ ] **Step 2: Validate**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

Expected: same count as baseline.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add HD.JA_HashPassword java action and HD.ACT_ChangePassword (raise error + call java action + show home page)"
```

---

## Task 5: Add R5 Support Microflows — ACT_Agent_Deactivate + ACT_Agent_Reactivate

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` — append to Account Management MARK block

**Why:** These are the microflows triggered by the action buttons in `HD.Account_Overview` (R5). They are simple change+commit microflows but must exist before the page that references them.

- [ ] **Step 1: Append after HD.ACT_ChangePassword**

```mdl

-- R5 support: action button microflows for Account_Overview
create or modify microflow HD.ACT_Agent_Deactivate
  ($Agent: HD.Agent)
  folder 'Account'
begin
  change $Agent (IsActive = false);
  commit $Agent;
  return;
end;
/

create or modify microflow HD.ACT_Agent_Reactivate
  ($Agent: HD.Agent)
  folder 'Account'
begin
  change $Agent (IsActive = true, DeactivationReason = empty);
  commit $Agent;
  return;
end;
/
```

- [ ] **Step 2: Validate**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add ACT_Agent_Deactivate and ACT_Agent_Reactivate support microflows"
```

---

## Task 6: Add R1/R2/R3 Pages — ManageMyAccount + ChangeMyPassword + ShowMyPasswordForm

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` — append to Account Management MARK block

**Why:**
- `HD.ManageMyAccount`: first demonstration of a **DataView with microflow datasource** (previously only DataGrid used microflow datasource).
- `HD.ChangeMyPassword`: full-page form that triggers `HD.ACT_ChangePassword`.
- `HD.ShowMyPasswordForm`: popup variant — same microflow, different layout, opened from the ManageMyAccount page.

- [ ] **Step 1: Append after the support microflows**

```mdl

-- MARK: Account Management — Pages

-- R1: HD.ManageMyAccount
-- New pattern: DataView with microflow datasource (datasource: microflow HD.DS_GetMyProfile)
-- Previously only DataGrid demonstrated microflow datasource; DataView is different.
-- DS_GetMyProfile internally runs split type + cast to return HD.UserProfile.
-- No page parameter — can be referenced directly as a menu item.
create or replace page HD.ManageMyAccount (
  title: 'My Account',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Account'
) {
  layoutgrid lgAcctMain {
    row rAcctMain {
      column cAcctMain (desktopwidth: 8) {
        dataview dvProfile (datasource: microflow HD.DS_GetMyProfile) {
          textbox tbDisplayName (label: 'Display Name', attribute: DisplayName)
          footer ftrAcctActions {
            actionbutton btnSaveProfile (
              caption: 'Save',
              action: save_changes,
              buttonstyle: primary
            )
            actionbutton btnChangePwd (
              caption: 'Change Password',
              action: show_page HD.ShowMyPasswordForm (Form: $currentObject),
              buttonstyle: default
            )
          }
        }
      }
    }
  }
};

-- R2: HD.ChangeMyPassword
-- Full-page password change form. Triggers HD.ACT_ChangePassword (java action + raise error).
create or replace page HD.ChangeMyPassword (
  title: 'Change Password',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Account',
  params: { $Form: HD.PasswordForm }
) {
  layoutgrid lgPwd {
    row rPwd {
      column cPwd (desktopwidth: 6) {
        dataview dvPwdFull (datasource: $Form) {
          textbox tbNewPwdFull     (label: 'New Password',     attribute: NewPassword)
          textbox tbConfirmPwdFull (label: 'Confirm Password', attribute: ConfirmPassword)
          footer ftrPwdFull {
            actionbutton btnSubmitPwdFull (
              caption: 'Change Password',
              action: microflow HD.ACT_ChangePassword (Form: $currentObject),
              buttonstyle: primary
            )
            actionbutton btnCancelPwdFull (
              caption: 'Cancel',
              action: cancel_changes close_page
            )
          }
        }
      }
    }
  }
};

-- R3: HD.ShowMyPasswordForm
-- Popup variant of ChangeMyPassword. Same HD.ACT_ChangePassword microflow,
-- Atlas_Core.PopupLayout entry point. Demonstrates one microflow callable from
-- two different page contexts (full-page vs. popup).
create or replace page HD.ShowMyPasswordForm (
  title: 'Change Password',
  layout: Atlas_Core.PopupLayout,
  folder: 'Account',
  params: { $Form: HD.PasswordForm }
) {
  dataview dvPwdPopup (datasource: $Form) {
    textbox tbNewPwdPopup     (label: 'New Password',     attribute: NewPassword)
    textbox tbConfirmPwdPopup (label: 'Confirm Password', attribute: ConfirmPassword)
    footer ftrPwdPopup {
      actionbutton btnSubmitPwdPopup (
        caption: 'Change Password',
        action: microflow HD.ACT_ChangePassword (Form: $currentObject) close_page,
        buttonstyle: primary
      )
      actionbutton btnCancelPwdPopup (
        caption: 'Cancel',
        action: cancel_changes close_page
      )
    }
  }
};
```

- [ ] **Step 2: Validate**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add ManageMyAccount (DataView+microflow datasource), ChangeMyPassword, ShowMyPasswordForm pages"
```

---

## Task 7: Add R4+R5+R6 Page — HD.Account_Overview (tabs + filters + action buttons)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` — append to Account Management MARK block

**Why:** Three new patterns in one page:
- R6: `tabcontainer` / `tabpage` — not demonstrated anywhere in the helpdesk file.
- R4: `dropdownfilter` on a boolean column (IsActive) — `textfilter` was already used; this adds a new column filter type in a meaningful context.
- R5: `ShowContentAs: customContent` column containing `actionbutton` — the only way to put a button inside a DataGrid2 column.

- [ ] **Step 1: Append after HD.ShowMyPasswordForm**

```mdl

-- R4 + R5 + R6: HD.Account_Overview
-- R6: tabcontainer tcAccounts with two tabpages (Active Accounts / Inactive Accounts)
-- R4: textfilter (Name, Email, Company) + dropdownfilter (IsActive boolean column)
-- R5: ShowContentAs: customContent column containing actionbutton widgets
create or replace page HD.Account_Overview (
  title: 'Account Management',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Account'
) {
  layoutgrid lgAcct {
    row rAcct {
      column cAcct (desktopwidth: 12) {
        tabcontainer tcAccounts {                              -- R6: tabs

          tabpage tabActive (caption: 'Active Accounts') {

            -- Agent datagrid: R4 (filters) + R5 (action button column)
            datagrid dgAgents (
              datasource: database from HD.Agent sort by Name asc,
              PageSize: 15
            ) {
              column colAgentName   (attribute: Name,     caption: 'Name') {
                textfilter fAgentName                         -- R4: textfilter
              }
              column colAgentEmail  (attribute: Email,    caption: 'Email') {
                textfilter fAgentEmail                        -- R4: textfilter
              }
              column colAgentActive (attribute: IsActive, caption: 'Active',
                                    ColumnWidth: manual, Size: 80) {
                dropdownfilter fAgentActive                   -- R4: dropdownfilter (boolean)
              }
              column colAgentAct (caption: 'Actions',
                ShowContentAs: customContent,                 -- R5: action button column
                ColumnWidth: manual, Size: 180) {
                actionbutton btnEditAgent (
                  caption: 'Edit',
                  action: show_page HD.Agent_Edit (Agent: $currentObject),
                  buttonstyle: default
                )
                actionbutton btnDeactivate (
                  caption: 'Deactivate',
                  action: microflow HD.ACT_Agent_Deactivate (Agent: $currentObject),
                  buttonstyle: default
                )
              }
            }

            -- Customer datagrid: R4 (filters) + R5 (action button column)
            datagrid dgCustomers (
              datasource: database from HD.Customer sort by Name asc,
              PageSize: 15
            ) {
              column colCustName    (attribute: Name,    caption: 'Name') {
                textfilter fCustName
              }
              column colCustEmail   (attribute: Email,   caption: 'Email') {
                textfilter fCustEmail
              }
              column colCustCompany (attribute: Company, caption: 'Company') {
                textfilter fCustCompany
              }
              column colCustAct (caption: 'Actions',
                ShowContentAs: customContent,
                ColumnWidth: manual, Size: 200) {
                actionbutton btnNewTicket (
                  caption: 'New Ticket',
                  action: show_page HD.Customer_New (Customer: $currentObject),
                  buttonstyle: primary
                )
                actionbutton btnEditCustomer (
                  caption: 'Edit',
                  action: show_page HD.Customer_Edit (Customer: $currentObject),
                  buttonstyle: default
                )
              }
            }

          }

          tabpage tabInactive (caption: 'Inactive Accounts') {

            datagrid dgInactiveAgents (
              datasource: database from HD.Agent sort by Name asc,
              PageSize: 15
            ) {
              column colIName   (attribute: Name,               caption: 'Name') {
                textfilter fIName
              }
              column colIActive (attribute: IsActive,           caption: 'Active',
                                ColumnWidth: manual, Size: 80) {
                dropdownfilter fIActive
              }
              column colReason  (attribute: DeactivationReason, caption: 'Reason')
              column colIAct    (caption: 'Actions',
                ShowContentAs: customContent,
                ColumnWidth: manual, Size: 120) {
                actionbutton btnReactivate (
                  caption: 'Reactivate',
                  action: microflow HD.ACT_Agent_Reactivate (Agent: $currentObject),
                  buttonstyle: default
                )
              }
            }

          }

        }
      }
    }
  }
};
```

- [ ] **Step 2: Validate**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add HD.Account_Overview page (tabs R6, column filters R4, action button columns R5)"
```

---

## Task 8: Add R7 Page — HD.Customer_New (nested DataView)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` — append to Account Management MARK block

**Why:** Demonstrates DataView nested inside DataView, with the inner DataView using an association path as datasource (`$Customer/HD.Ticket_Customer/HD.Ticket`). This is distinct from the existing `$Ticket/HD.TicketComment_Ticket/HD.TicketComment` usage which is a DataGrid datasource, not a nested DataView.

- [ ] **Step 1: Append after HD.Account_Overview**

```mdl

-- R7: HD.Customer_New — DataView nested inside DataView
-- Outer dvCustomer: datasource: $Customer (page parameter)
-- Inner dvFirstTicket: datasource: $Customer/HD.Ticket_Customer/HD.Ticket
--   — association path used as DataView datasource (distinct from DataGrid association datasource)
--   — shows Subject + Status of the customer's most recent associated ticket
create or replace page HD.Customer_New (
  title: 'Customer Profile',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Account',
  params: { $Customer: HD.Customer }
) {
  layoutgrid lgCust {
    row rCust {
      column cCust (desktopwidth: 12) {
        dataview dvCustomer (datasource: $Customer) {          -- outer DataView (R7)
          textbox tbCustName    (label: 'Name',    attribute: Name)
          textbox tbCustEmail   (label: 'Email',   attribute: Email)
          textbox tbCustCompany (label: 'Company', attribute: Company)
          dataview dvFirstTicket (                             -- inner DataView (R7)
            datasource: $Customer/HD.Ticket_Customer/HD.Ticket
          ) {
            dynamictext txtTicketSubject (
              content: 'Latest Ticket: {1}',
              contentparams: [{1} = Subject],
              rendermode: H4
            )
            dynamictext txtTicketStatus (
              content: '{1}',
              contentparams: [{1} = Status]
            )
          }
          footer ftrCustSave {
            actionbutton btnSaveCust (
              caption: 'Save',
              action: save_changes close_page,
              buttonstyle: primary
            )
            actionbutton btnCancelCust (
              caption: 'Cancel',
              action: cancel_changes close_page
            )
          }
        }
      }
    }
  }
};
```

- [ ] **Step 2: Validate**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add HD.Customer_New page (nested DataView R7)"
```

---

## Task 9: Add R8+R9 Pages — HD.Agent_Edit + HD.Customer_Edit (conditional visibility)

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` — append to Account Management MARK block

**Why:** Two distinct conditional visibility patterns using `visible: [XPath]`:
- R8 (Agent): `$Agent/IsActive = false` — attribute equality check.
- R9 (Customer): `$Customer/Company != empty` — empty-check on a nullable string.

- [ ] **Step 1: Append after HD.Customer_New**

```mdl

-- R8: HD.Agent_Edit — conditional visibility (attribute equality)
-- visible: [$Agent/IsActive = false] controls DeactivationReason textbox.
-- When IsActive is true the field is hidden; when false it appears and is editable.
create or replace page HD.Agent_Edit (
  title: 'Edit Agent',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Account',
  params: { $Agent: HD.Agent }
) {
  layoutgrid lgAgent {
    row rAgent {
      column cAgent (desktopwidth: 8) {
        dataview dvAgent (datasource: $Agent) {
          textbox tbAgentName  (label: 'Name',   attribute: Name)
          textbox tbAgentEmail (label: 'Email',  attribute: Email)
          textbox tbIsActive   (label: 'Active', attribute: Name)
          -- R8: conditional visibility — only visible when agent is deactivated
          textbox tbDeactivationReason (
            label: 'Deactivation Reason',
            attribute: DeactivationReason,
            visible: [$Agent/IsActive = false]
          )
          footer ftrAgent {
            actionbutton btnSaveAgent (
              caption: 'Save',
              action: save_changes close_page,
              buttonstyle: primary
            )
            actionbutton btnCancelAgent (
              caption: 'Cancel',
              action: cancel_changes close_page
            )
          }
        }
      }
    }
  }
};

-- R9: HD.Customer_Edit — conditional visibility (empty check)
-- visible: [$Customer/Company != empty] controls the corporate contract block.
-- The block appears only after the user fills in a company name.
create or replace page HD.Customer_Edit (
  title: 'Edit Customer',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Account',
  params: { $Customer: HD.Customer }
) {
  layoutgrid lgCustEdit {
    row rCustEdit {
      column cCustEdit (desktopwidth: 8) {
        dataview dvCustEdit (datasource: $Customer) {
          textbox tbCustEditName    (label: 'Name',    attribute: Name)
          textbox tbCustEditEmail   (label: 'Email',   attribute: Email)
          textbox tbCustEditCompany (label: 'Company', attribute: Company)
          -- R9: conditional visibility — corporate contract block
          dynamictext txtContractTitle (
            content: 'Corporate Contract',
            rendermode: H4,
            visible: [$Customer/Company != empty]
          )
          dynamictext txtContractInfo (
            content: 'Company on file: {1}',
            contentparams: [{1} = Company],
            visible: [$Customer/Company != empty]
          )
          footer ftrCustEdit {
            actionbutton btnSaveCustEdit (
              caption: 'Save',
              action: save_changes close_page,
              buttonstyle: primary
            )
            actionbutton btnCancelCustEdit (
              caption: 'Cancel',
              action: cancel_changes close_page
            )
          }
        }
      }
    }
  }
};
```

- [ ] **Step 2: Validate**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): add HD.Agent_Edit (R8 conditional visibility) and HD.Customer_Edit (R9 conditional visibility)"
```

---

## Task 10: Security Grants + Navigation + MOVE Commands

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` — three insertion points

**Why:** Completes the wiring: grants make pages accessible to roles, navigation adds menu entries, MOVE aligns folder declarations with the `folder` clauses already on each element.

- [ ] **Step 1: Insert security grants after line 1546 (`grant view on page KB.Article_Detail ...`)**

After this line:
```mdl
grant view on page KB.Article_Detail       to KB.Reader, KB.Contributor, KB.Admin;
```

Append:
```mdl

-- Account Management grants (R1–R9)
grant HD.ManagerRole on HD.UserProfile (create, read *, write *);
grant view on page HD.ManageMyAccount    to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page HD.ChangeMyPassword   to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page HD.ShowMyPasswordForm to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page HD.Account_Overview   to HD.ManagerRole;
grant view on page HD.Customer_New       to HD.AgentRole, HD.ManagerRole;
grant view on page HD.Agent_Edit         to HD.AgentRole, HD.ManagerRole;
grant view on page HD.Customer_Edit      to HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.DS_GetMyProfile      to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_ChangePassword   to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant execute on microflow HD.ACT_Agent_Deactivate to HD.ManagerRole;
grant execute on microflow HD.ACT_Agent_Reactivate to HD.ManagerRole;
```

- [ ] **Step 2: Add `menu '账户管理'` group to the navigation block**

Find the existing navigation menu block (around line 1575):
```mdl
  menu (
    menu item '我的工单'  page HD.MyTickets_Overview;
    menu item '知识库'    page KB.Article_Overview;
    menu '工单管理' (
      menu item '所有工单'  page HD.Ticket_Overview;
      menu item '升级审批'  page HD.EscalationWorkflow_Overview;
      menu item '工作流管理' page HD.EscalationWorkflow_Overview;
    );
    menu item '系统管理' page Administration.Account_Overview;
  );
```

Replace with (add `menu '账户管理'` before `menu item '系统管理'`):
```mdl
  menu (
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
      menu item '修改密码'  page HD.ChangeMyPassword;
    );
    menu item '系统管理' page Administration.Account_Overview;
  );
```

- [ ] **Step 3: Append MOVE commands after line 1653 (`move microflow HD.ACT_Workflow_Notify ...`)**

After this line:
```mdl
move microflow HD.ACT_Workflow_Notify            to folder 'Escalation/WorkflowAdmin';
```

Append:
```mdl

-- HD Module — Account
move microflow HD.DS_GetMyProfile       to folder 'Account';
move microflow HD.ACT_ChangePassword    to folder 'Account';
move microflow HD.ACT_Agent_Deactivate  to folder 'Account';
move microflow HD.ACT_Agent_Reactivate  to folder 'Account';
move page      HD.ManageMyAccount       to folder 'Account';
move page      HD.ChangeMyPassword      to folder 'Account';
move page      HD.ShowMyPasswordForm    to folder 'Account';
move page      HD.Account_Overview      to folder 'Account';
move page      HD.Customer_New          to folder 'Account';
move page      HD.Agent_Edit            to folder 'Account';
move page      HD.Customer_Edit         to folder 'Account';
```

- [ ] **Step 4: Validate**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -c "\[error\]" || echo "0 errors"
```

- [ ] **Step 5: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): wire Account Management security grants, navigation menu, and folder MOVE commands"
```

---

## Task 11: Update File Header + Final Validation

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl:1–35`

**Why:** The file header contains a running `-- Note:` log of every significant change. Keeping it current is part of the golden file convention.

- [ ] **Step 1: Add a Note entry to the file header (after the last existing `-- Note:` line, around line 34)**

Find the last `-- Note:` line (currently around line 34):
```mdl
-- Note    : msdkWrite now routes through writeUnitContents so EXECUTE SCRIPT security
```

Insert after it:
```mdl
-- Note    : Account Management section added (Tasks R1-R9): HD.UserProfile extends
--           System.User; split type/cast; create/call java action; raise error;
--           show home page; DataView microflow datasource; tabcontainer/tabpage;
--           customContent action button column; nested DataView; conditional visibility.
```

- [ ] **Step 2: Run full validation**

```bash
make build && ./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | tee /tmp/helpdesk-final.txt
grep -c "\[error\]" /tmp/helpdesk-final.txt || echo "0 errors"
diff <(grep "\[error\]" /tmp/helpdesk-baseline.txt) <(grep "\[error\]" /tmp/helpdesk-final.txt) && echo "No regression" || echo "REGRESSION — review diff above"
```

Expected: `No regression` — error count identical to baseline from Task 1.

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): complete Account Management extension — 9 requirements, 12 new MDL activity types"
```

---

## Self-Review Checklist

| Requirement | Task | Status |
|-------------|------|--------|
| R1 ManageMyAccount (split type + cast + DataView microflow datasource) | Task 3 + Task 6 | ✓ |
| R2 ChangeMyPassword (java action + raise error + show home page) | Task 4 + Task 6 | ✓ |
| R3 ShowMyPasswordForm (popup variant) | Task 6 | ✓ |
| R4 Account_Overview column filters (textfilter + dropdownfilter) | Task 7 | ✓ |
| R5 Account_Overview action button column (customContent) | Task 7 | ✓ |
| R6 Account_Overview tabs (tabcontainer + tabpage) | Task 7 | ✓ |
| R7 Customer_New nested DataView | Task 8 | ✓ |
| R8 Agent_Edit conditional visibility (IsActive = false) | Task 9 | ✓ |
| R9 Customer_Edit conditional visibility (Company != empty) | Task 9 | ✓ |
| Security grants | Task 10 | ✓ |
| Navigation | Task 10 | ✓ |
| MOVE commands | Task 10 | ✓ |
| File header note | Task 11 | ✓ |
| DeactivationReason on HD.Agent | Task 2 | ✓ |
