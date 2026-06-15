# Capstone Execution Guide

## Quick Start (run the full suite on a clean project)

```bash
# 1. Prepare a clean project
mxcli new MyHelpdesk --version 11.6.6

# 2. Execute the reference implementations in order
mxcli exec academy/zh/capstone-helpdesk/参考实现/01-domain.mdl      -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/02-microflows.mdl  -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/03-nanoflows.mdl   -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/04-pages.mdl       -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/05-security.mdl    -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/06-kb.mdl          -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/07-escalation.mdl  -p MyHelpdesk/MyHelpdesk.mpr
mxcli exec academy/zh/capstone-helpdesk/参考实现/99-seed-data.mdl   -p MyHelpdesk/MyHelpdesk.mpr

# 3. Validate (on Windows, Studio Pro locates mxbuild automatically)
mxcli docker check -p MyHelpdesk/MyHelpdesk.mpr
# Expected: 0 StorageLoadException

# 4. Start the app
mxcli local run -p MyHelpdesk/MyHelpdesk.mpr --admin-password Admin1234

# 5. Log in with a demo account
# URL: http://localhost:8080
# Customer: demo_customer@helpdesk.test / Demo12345678
# Agent:    demo_agent@helpdesk.test    / Demo12345678
# Manager:  demo_manager@helpdesk.test  / Demo12345678
# After logging in as Manager, click "Initialize Demo Data" in the navigation menu to seed the data
```

## File Reference

| File | Contents | Depends On |
|------|----------|------------|
| 01-domain.mdl     | HD entities, enumerations, associations, constants | none |
| 02-microflows.mdl | Ticket state machine microflows, data sources | 01 |
| 03-nanoflows.mdl  | Nanoflows: quick create, search, formatting | 01 |
| 04-pages.mdl      | All HD pages | 01–03 |
| 05-security.mdl   | HD roles, authorization, demo users, navigation | 01–04 |
| 06-kb.mdl         | KB domain model + microflows + pages + security | 05 |
| 07-escalation.mdl | Escalation approval entities + microflows + pages + permissions + navigation | 02, 06 |
| 99-seed-data.mdl  | Demo data microflow (triggered manually after login) | all |

> Note: 07 changes the Manager home page to the escalation approval page, and adds Knowledge Base / Escalations menu items;
> 99 adds an "Admin > Initialize Demo Data" menu item on top of that. Be sure to execute in the order 01→99.
