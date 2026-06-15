# Module 03: Nanoflows and the Client — Business Requirements

## Business Context

Not every operation needs to hit the server. Some simple operations — like quickly creating a new ticket, or searching and filtering within a list — can be done directly in the user's browser, with faster response and a better experience.

---

## User Stories

### Quick Ticket Creation
- As a customer, I want to quickly submit a ticket title in a clean popup without navigating to another page, so that I can ask for help faster
- As the system, a quickly created ticket defaults to "Draft" status with "Normal" priority

### Ticket Search Filtering
- As an agent, I want to quickly search for tickets containing a keyword on the ticket list page, without refreshing the entire page
- As the system, search results should respond in real time, with no server round-trip

### Priority Label Formatting
- As an agent, I want to see human-friendly priority labels (such as "⚠️ High") in the ticket list, rather than English enum values
- As the system, the formatting logic should run on the client, with no server computation needed

---

## Acceptance Criteria

- [ ] Quick create: enter title → ticket saved to the database → status Draft, priority Normal
- [ ] Search: enter keyword → filter results without refreshing the whole page
- [ ] Priority labels: `Critical` → `🔴 紧急`, `High` → `🟠 高`, `Normal` → `🟡 普通`, `Low` → `🟢 低`
