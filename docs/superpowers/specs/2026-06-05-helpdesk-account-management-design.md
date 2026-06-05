# Helpdesk Account Management — MDL Pattern Extension Design

**Date:** 2026-06-05  
**File:** `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`  
**Scope:** Add Account Management section to the helpdesk baseline, covering 9 requirements and 12 previously-undemonstrated MDL activity types.

---

## Purpose

The helpdesk golden file is a regression baseline and syntax showcase. This design extends it with an Account Management section that:

1. Mirrors the functionality of Mendix's built-in `Administration.ManageMyAccount`, `Administration.ChangeMyPassword`, and `ShowMyPasswordForm` pages.
2. Introduces 12 MDL activity types / widget patterns not yet demonstrated anywhere in the helpdesk file.
3. Follows the "necessary and sufficient" principle — every new element earns its place by demonstrating a distinct, previously-uncovered pattern.

---

## Requirements Traceability

| # | Original Request | New MDL Pattern | Implementation |
|---|-----------------|-----------------|---------------|
| **R1** | `Administration.ManageMyAccount` | `extends System.User` · `split type $Var case Module.Entity` · `cast $Typed` · DataView with microflow datasource | `HD.UserProfile` entity + `HD.DS_GetMyProfile` microflow + `HD.ManageMyAccount` page |
| **R2** | `Administration.ChangeMyPassword` | `create java action` · `call java action` · `raise error` · `show home page` | `HD.JA_HashPassword` java action + `HD.ACT_ChangePassword` microflow + `HD.ChangeMyPassword` page |
| **R3** | `ShowMyPasswordForm` | No new activities — popup variant of R2 showing the same microflow triggered from a different entry point | `HD.ShowMyPasswordForm` popup page |
| **R4** | Account_Overview — column filter components | `textfilter` · `dropdownfilter` on a boolean attribute column | `HD.Account_Overview` datagrid columns |
| **R5** | Account_Overview — action button components | `ShowContentAs: customContent` column with inline `actionbutton` | `HD.Account_Overview` Actions columns |
| **R6** | Account_Overview — tabs component | `tabcontainer` · `tabpage` (caption) | `HD.Account_Overview` page structure |
| **R7** | Account_New — DataView nested in DataView | Outer `dvCustomer (datasource: $Customer)` + inner `dvFirstTicket (datasource: $Customer/HD.Ticket_Customer/HD.Ticket)` | `HD.Customer_New` page |
| **R8** | Account_Edit — conditional visibility (Agent) | `visible: [$Agent/IsActive = false]` on `textbox tbDeactivationReason` | `HD.Agent_Edit` page |
| **R9** | Account_Edit — conditional visibility (Customer) | `visible: [$Customer/Company != empty]` on `dynamictext` blocks | `HD.Customer_Edit` page |

---

## New MDL Activities Summary

Activities and patterns not present anywhere in the helpdesk file before this change:

| Activity | Requirement | MDL Syntax |
|----------|------------|-----------|
| Entity generalization | R1 | `create or modify persistent entity HD.UserProfile extends System.User (...)` |
| split type | R1 | `split type $Me case HD.UserProfile ... else ... end split` |
| cast | R1 | `cast $Profile;` (inside split type case block) |
| DataView + microflow datasource | R1 | `dataview dvProfile (datasource: microflow HD.DS_GetMyProfile)` |
| create java action | R2 | `create or modify java action HD.JA_HashPassword (PlainText: string) returns string` |
| call java action | R2 | `$Hashed = call java action HD.JA_HashPassword (PlainText = $Form/NewPassword)` |
| raise error | R2 | `raise error;` |
| show home page | R2 | `show home page;` |
| tabcontainer / tabpage | R6 | `tabcontainer tc { tabpage tp (caption: '...') { ... } }` |
| customContent column + actionbutton | R5 | `ShowContentAs: customContent` + `actionbutton` inside column |
| nested DataView | R7 | Inner `dataview (datasource: $Customer/HD.Ticket_Customer/HD.Ticket)` |
| conditional visibility | R8/R9 | `visible: [XPath]` property on widget |

---

## Architecture

### Insertion Point

A new `-- MARK: Account Management` section is inserted between the existing `-- MARK: Pages — HelpDesk` section and `-- MARK: Security — Module Roles`. This keeps all new content isolated and does not modify any existing page, microflow, or entity (except adding `DeactivationReason` to `HD.Agent`).

### Layer 1: Entity Extensions

**`HD.UserProfile extends System.User`** (new entity)
- Enables the `split type` + `cast` pattern in `HD.DS_GetMyProfile`.
- `DisplayName: string(200)` — one field is sufficient to demonstrate the pattern.

**`HD.PasswordForm`** (new non-persistent entity)
- Holds `NewPassword: string` and `ConfirmPassword: string` for the password change form.
- Non-persistent: no database write, pure form holder.

**`HD.Agent`** (modified — add one field)
- Add `DeactivationReason: string(500)` to support R8 conditional visibility.

### Layer 2: Microflows

**`HD.DS_GetMyProfile()`** — R1
- Retrieves `$Me from System.User where [id = '[%CurrentUser%]'] limit 1`.
- `split type $Me case HD.UserProfile: cast $Profile; return $Profile; else: return empty;`
- Returns `HD.UserProfile`. Used as DataView datasource in `HD.ManageMyAccount`.

**`HD.JA_HashPassword(PlainText: string) returns string`** — R2 (java action definition)
- Declares the Java interface only. Java implementation loaded at runtime.
- No source block needed for the pattern demonstration.

**`HD.ACT_ChangePassword($Form: HD.PasswordForm) returns boolean`** — R2
1. Validate `NewPassword` not empty → `validation feedback` + `return false`.
2. Passwords differ → `raise error` (terminates to ErrorEvent).
3. `$Hashed = call java action HD.JA_HashPassword (PlainText = $Form/NewPassword)`.
4. `set $Success = true`.
5. `show home page`.
6. `return $Success`.

**`HD.ACT_Agent_Deactivate($Agent: HD.Agent)`** — R5 support
- `change $Agent (IsActive = false); commit $Agent;`

**`HD.ACT_Agent_Reactivate($Agent: HD.Agent)`** — R5 support
- `change $Agent (IsActive = true, DeactivationReason = empty); commit $Agent;`

### Layer 3: Pages

**`HD.ManageMyAccount`** — R1
- No page parameter. DataView datasource: `microflow HD.DS_GetMyProfile`.
- First demonstration of a DataView (not DataGrid) with microflow datasource.
- Contains `btnChangePwd` action button opening `HD.ShowMyPasswordForm`.

**`HD.ChangeMyPassword($Form: HD.PasswordForm)`** — R2
- Full-page variant. DataView datasource: `$Form`.
- Submit button: `action: microflow HD.ACT_ChangePassword (Form: $currentObject)`.

**`HD.ShowMyPasswordForm($Form: HD.PasswordForm)`** — R3
- `layout: Atlas_Core.PopupLayout`. Same form as R2, different entry point.
- Submit button: `action: microflow HD.ACT_ChangePassword (Form: $currentObject) close_page`.

**`HD.Account_Overview`** — R4 + R5 + R6
- `tabcontainer tcAccounts` with two tabpages: `tabActive` and `tabInactive`. (R6)
- **Active tab**: `dgAgents` (textfilter on Name/Email, dropdownfilter on IsActive, customContent Actions column with Edit + Deactivate buttons) + `dgCustomers` (textfilter on Name/Email/Company, customContent Actions column with New Ticket + Edit buttons). (R4 + R5)
- **Inactive tab**: `dgInactiveAgents` (textfilter, dropdownfilter, DeactivationReason column, Reactivate action button). (R4 + R5)

**`HD.Customer_New($Customer: HD.Customer)`** — R7
- Outer `dvCustomer (datasource: $Customer)` contains textboxes for Name/Email/Company.
- Inner `dvFirstTicket (datasource: $Customer/HD.Ticket_Customer/HD.Ticket)` shows Subject + Status of associated ticket.

**`HD.Agent_Edit($Agent: HD.Agent)`** — R8
- `textbox tbDeactivationReason` has `visible: [$Agent/IsActive = false]`.

**`HD.Customer_Edit($Customer: HD.Customer)`** — R9
- `dynamictext txtContractTitle` and `txtContractInfo` both have `visible: [$Customer/Company != empty]`.

### Layer 4: Navigation, Security, Move

**Navigation** — extend existing `create or replace navigation Responsive` menu block:
```
menu '账户管理' (
  menu item '账户总览'  page HD.Account_Overview;
  menu item '我的账户'  page HD.ManageMyAccount;
  menu item '修改密码'  page HD.ChangeMyPassword;
);
```

**Security grants** — new pages grant to appropriate roles:
- `HD.ManageMyAccount`, `HD.ChangeMyPassword`, `HD.ShowMyPasswordForm`: all three roles (CustomerRole, AgentRole, ManagerRole).
- `HD.Account_Overview`: ManagerRole only.
- `HD.Customer_New`, `HD.Agent_Edit`, `HD.Customer_Edit`: AgentRole + ManagerRole.
- `HD.DS_GetMyProfile`, `HD.ACT_ChangePassword`: all three roles.
- `HD.ACT_Agent_Deactivate`, `HD.ACT_Agent_Reactivate`: ManagerRole only.

**MOVE commands** — all new microflows and pages moved to folder `'Account'` via `move` statements in the existing MARK: Folder Organization section.

---

## Design Decisions

**Why `HD.DS_GetMyProfile` as DataView datasource (not page parameter)?**  
`HD.ManageMyAccount` should be openable directly from a menu item without a preceding microflow call. A microflow datasource on the DataView achieves this cleanly and simultaneously demonstrates the previously-uncovered pattern of DataView (not DataGrid) with microflow datasource.

**Why `raise error` in `HD.ACT_ChangePassword`?**  
Password mismatch is a hard failure (not a validation feedback case) — the user should see the Mendix error dialog. This is consistent with Administration module behavior and correctly demonstrates `raise error` (ErrorEvent path in the flow) as distinct from `validation feedback` (which sets field-level UI errors and continues).

**Why `show home page` after password change?**  
Password change is a security-sensitive operation. Redirecting to the home page clears the form and resets navigation state. This is standard Mendix Administration behavior and introduces `show home page` (the only remaining client-navigation activity type not yet demonstrated).

**Why `dropdownfilter` on `IsActive` (boolean) instead of a `booleanfilter`?**  
`booleanfilter` is not confirmed in the MDL grammar. `dropdownfilter` on a boolean column is the documented approach and matches existing `HD.Ticket_Overview` filter patterns.

**Why `HD.ShowMyPasswordForm` has no new activities?**  
R3 ("ShowMyPasswordForm") exists as a distinct UI entry point (popup vs. full page), not as a distinct microflow pattern. Including it with no new activities is correct — it demonstrates that the same ACT_ChangePassword microflow is reusable from multiple page contexts.

---

## What Is NOT Changed

- Existing pages (`HD.Ticket_Overview`, `KB.Article_Overview`, etc.) — no modifications.
- Existing microflows — no modifications.
- Existing navigation entries — only additive (new menu group appended).
- Existing security grants — only additive.
- Existing `HD.Agent` entity fields — `DeactivationReason` is added only; existing fields unchanged.

---

## File Insertion Order

Within `helpdesk-app.mdl`, the new content is inserted as a single `-- MARK: Account Management` block in this order:

1. Entity additions: `HD.UserProfile`, `HD.PasswordForm`, modified `HD.Agent`
2. Microflows: `HD.DS_GetMyProfile`, `HD.JA_HashPassword`, `HD.ACT_ChangePassword`, `HD.ACT_Agent_Deactivate`, `HD.ACT_Agent_Reactivate`
3. Pages (R1→R3): `HD.ManageMyAccount`, `HD.ChangeMyPassword`, `HD.ShowMyPasswordForm`
4. Pages (R4–R6): `HD.Account_Overview`
5. Pages (R7): `HD.Customer_New`
6. Pages (R8–R9): `HD.Agent_Edit`, `HD.Customer_Edit`
7. Security grants (appended to existing MARK: Security — User Roles section)
8. MOVE commands (appended to existing MARK: Folder Organization section)
9. Navigation update (merged into existing `create or replace navigation Responsive`)
