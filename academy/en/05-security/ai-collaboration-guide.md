# Module 05: AI Collaboration Guide — Security & Permissions

## Prerequisites

First run the reference implementation for modules 01–04.

## The Four Layers of Security Configuration

```
1. Module Role     ← defines the role names
2. Entity Grant    ← decides who can read/write/delete which entities
3. Microflow/Page Grants ← decides who can run which logic and pages
4. User Role       ← combines module roles into system user types
```

## Steps to Collaborate with Claude

```
Help me configure a three-role security model for Helpdesk using MDL:
- CustomerRole: read/write only their own tickets (XPath row-level filtering), cannot see Internal comments
- AgentRole: full access to all tickets
- ManagerRole: AgentRole plus delete permission

Also needed:
- User roles: Customer (includes HD.CustomerRole), Agent (includes HD.AgentRole), Manager (includes HD.ManagerRole)
- Demo users: demo_customer / demo_agent / demo_manager, password Demo12345678
- Navigation: Customer home → My Tickets, Agent home → All Tickets, Manager home → All Tickets
```

## XPath Row-Level Filtering

Row-level filtering uses the `where '[xpath]'` syntax:

```mdl
-- See only your own tickets: from Ticket find the associated Customer, then check the Customer's owner
grant HD.CustomerRole on HD.Ticket (create, read *, write *)
  where '[HD.Ticket_Customer/HD.Customer/System.owner=''[%CurrentUser%]'']';

-- See only non-internal comments
grant HD.CustomerRole on HD.TicketComment (create, read *)
  where '[IsInternal = false]';
```

Note: single quotes inside an XPath string must be **doubled** (`''`).

## Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| Demo user password does not meet the password policy | First run `alter project security password policy (min_length: 8, require_digit: true, require_mixed_case: true)` |
| Demo user gets "unknown user" at login | Check whether you first ran `alter project security demo users on` |
| Navigation home page setting has no effect | The `home page ... for Customer` syntax requires an exact match on the User Role name |
