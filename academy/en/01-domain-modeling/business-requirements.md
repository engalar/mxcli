# Module 01: Domain Modeling — Business Requirements

## Business Context

TechCorp's Helpdesk system needs to track three categories of core information: **who submitted the issue** (customer), **who is handling the issue** (agent), and **what exactly the issue is** (ticket).

Without this foundational information, the system has nothing to run on — much like a library with no book catalog.

---

## User Stories

### Customer Information Management
- As an IT agent, I want to know which employee submitted the ticket, so that I can reach out to them
- As an IT agent, I want to know which department the employee is in, so that I can assess the scope of impact
- As a manager, I want to view ticket distribution by company/department, so that I can allocate resources sensibly

### Ticket Information
- As an agent, I want to know the issue's title and detailed description, so that I can quickly understand what needs to be resolved
- As an agent, I want to know the issue's urgency, so that I can prioritize the most important things
- As an agent, I want to see the ticket's current status (Draft / Open / In Progress / Resolved / Closed), so that I know how much work remains
- As a customer, I want to see when my ticket must be resolved (SLA due time), so that I can set reasonable expectations

### Agent Information
- As a manager, I want to know which agents are active, so that I can assign tickets sensibly
- As the system, I need to record which agent a ticket was assigned to, so that responsibility can be tracked

---

## Acceptance Criteria

- [ ] The system can record a customer's name, email, and company
- [ ] The system can record an agent's name, email, and whether they are active
- [ ] Every ticket has a title, description, urgency, current status, and SLA due time
- [ ] A ticket must be linked to one customer (who submitted it)
- [ ] A ticket can be linked to one agent (who is handling it)
- [ ] Urgency has four levels: Low / Normal / High / Critical
- [ ] Ticket status has five values: Draft / Open / In Progress / Resolved / Closed
