# Module 05: Security & Permissions — Business Requirements

## Business Context

TechCorp's IT support system has strict data isolation requirements: customers can only see their own issues, not anyone else's; agents can manage all tickets; managers have the highest level of access.

This is not just about "hiding menus" — it is isolation at the database level. Even if someone guesses a page URL, they cannot see data that does not belong to them.

---

## User Stories

### Customer Permissions
- As a customer, I can only see the tickets I submitted (the system filters at the database level, not by hiding the UI)
- As a customer, I can create tickets and modify ticket titles and descriptions
- As a customer, I cannot see comments marked as "internal" (agent-only viewing)
- As a customer, I cannot perform "assign" or "resolve" actions (only agents can do this)

### Agent Permissions
- As an agent, I can view and manage all customers' tickets
- As an agent, I can add internal comments (visible to other agents, not to customers)
- As an agent, I can assign tickets and mark them resolved
- As an agent, I cannot see the administrative permission settings pages

### Manager Permissions
- As a manager, I have all agent permissions plus the ability to delete tickets
- As a manager, I can access user management features
- As a manager, I can initialize demo data

---

## Acceptance Criteria

- [ ] Logging in as demo_customer, I can only see my own tickets
- [ ] Logging in as demo_agent, I can see all tickets
- [ ] Logging in as demo_customer, I cannot see comments marked IsInternal=true
- [ ] Role-specific home pages: Customer sees My Tickets, Agent sees All Tickets, Manager sees... (same as Agent; extension left for advanced work)
- [ ] The navigation menu automatically hides menu items that are inaccessible to the current role
- [ ] Three demo accounts can log in directly: demo_customer / demo_agent / demo_manager, all with password Demo12345678
