# Capstone: Complete Helpdesk System — Business Requirements

> This is your final goal. From the very first module, you have been building this system bit by bit.
> When you complete all the modules, your app should satisfy every requirement described here.

---

## Company Background

TechCorp is a tech company with 500 employees. The IT support team (8 agents) handles technical issues from across the company every day: laptops, system access, network connectivity, software installation…

**Problem:** Everything is currently handled by email and instant messaging. Issues frequently get lost, customers don't know the progress, and managers can't gauge the team's workload.

**Goal:** Build a self-service ticketing system that makes the entire support process transparent, traceable, and SLA-backed.

---

## User Types and Their Day

### Alice — Marketing Manager (Customer)

Alice's laptop suddenly can't connect to the company VPN, and she has an important meeting in the afternoon.

She opens the Helpdesk, logs in, and sees the "My Tickets" page.
- She clicks "New Ticket", fills in the title "Cannot connect to VPN", describes the symptoms, marks the priority as "High", and submits.
- The system tells her the deadline is 8 hours from now.
- Two hours later, she sees the ticket status change to "In Progress", with an agent's note "Investigating, please stay connected to the network".
- After the issue is resolved, she receives a "Resolved" notification, confirms, and closes the ticket.

### Bob — IT Agent (Agent)

The first thing Bob does at work: open "All Tickets", sort by SLA deadline, and quickly find the most urgent tickets.
- He finds an "Urgent" priority payment-issue ticket and claims it (assigns it to himself).
- He opens the ticket detail, reads the description, and adds an internal note: "Contacted Finance, awaiting confirmation".
- After the issue is resolved, he marks it as "Resolved" and adds an external comment explaining the solution.
- In his free time, he writes a knowledge base article "Troubleshooting Common VPN Issues" to help customers solve similar problems themselves.

### Carol — IT Supervisor (Manager)

Carol reviews the overall situation every morning:
- She opens "All Tickets" and checks how many tickets have breached SLA (IsOverSLA=true).
- She finds an overdue ticket and reassigns it to another agent.
- She approves an escalation request (an Agent has requested raising a ticket's priority to "Urgent").

---

## Functional Requirements

### Ticket Management

| Feature | Who Can Do It | Notes |
|---------|---------------|-------|
| Submit a ticket | Customer / Agent | Title is required, SLA is calculated automatically |
| View own tickets | Customer | Only their own, not others' |
| View all tickets | Agent / Manager | Everything, filterable by status/priority |
| Assign an agent | Agent / Manager | Ticket status changes to "In Progress" |
| Resolve a ticket | Agent / Manager | Records the resolution time, computes whether overdue |
| Reopen a ticket | Agent / Manager | Returns from Resolved/Closed to Open |
| Close a ticket | Agent / Manager | Terminal state, status can no longer be changed |
| Add a comment | Everyone | Customers only see non-internal comments |

### Knowledge Base

| Feature | Who Can Do It | Notes |
|---------|---------------|-------|
| Browse articles | Everyone (logged in) | Only "Published" articles |
| Search articles | Everyone | Search by title keyword |
| Write articles | Agent / Manager | Has Draft and Published states |
| Publish articles | Agent / Manager | Draft → Published |

### Users and Permissions

| Role | Login Account | Password | Home Page After Login |
|------|--------------|----------|----------------------|
| Customer | demo_customer@helpdesk.test | Demo12345678 | My Tickets |
| Agent | demo_agent@helpdesk.test | Demo12345678 | All Tickets |
| Manager | demo_manager@helpdesk.test | Demo12345678 | All Tickets |

---

## UI Requirements

### Visual Standards

- **Ticket list**: the status column uses colored badges to distinguish (green = Resolved, yellow = In Progress, red = Overdue)
- **Ticket detail**: 2-column layout (information on the left, action buttons on the right)
- **Comment section**: shown at the bottom of the detail page, internal comments have a clear marker
- **Navigation**: automatically shows different menu items based on role

### Demo Acceptance

The following demo paths must run end to end with no error popups:

**Path 1 (Customer's view):**
1. Log in as demo_customer
2. Create a "Normal" priority ticket
3. View the ticket, add a comment
4. Log out

**Path 2 (Agent's view):**
1. Log in as demo_agent
2. Open "All Tickets", find the ticket the customer created above
3. Assign it to yourself
4. Mark it as "Resolved", add a resolution-note comment
5. Log out

**Path 3 (Manager's view):**
1. Log in as demo_manager
2. View all tickets
3. Log out

---

## Technical Acceptance Criteria

```bash
# 1. Execute the full app MDL (modules 01-05 in order, or the capstone reference implementation)
mxcli exec academy/zh/01-领域建模/参考实现/domain-model.mdl -p MyProject.mpr
mxcli exec academy/zh/02-微流业务逻辑/参考实现/microflows.mdl  -p MyProject.mpr
mxcli exec academy/zh/03-纳流与客户端/参考实现/nanoflows.mdl   -p MyProject.mpr
mxcli exec academy/zh/04-页面与UI/参考实现/pages.mdl           -p MyProject.mpr
mxcli exec academy/zh/05-安全与权限/参考实现/security.mdl      -p MyProject.mpr

# 2. Mendix platform validation (must be 0 StorageLoadException)
~/.mxcli/mxbuild/*/modeler/mx check MyProject.mpr \
  2>&1 | grep -c "StorageLoadException"
# Expected output: 0
```
