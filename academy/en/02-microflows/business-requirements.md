# Module 02: Microflow Business Logic — Business Requirements

## Business Context

A data structure alone isn't enough — the system needs to know "what can be done under what circumstances." Tickets can't just be submitted any way you like, and the handling process must follow business rules step by step. These rules are enforced by **business logic**.

---

## The Life of a Ticket

```
Draft ──[submit]──► Open ──[assign]──► In Progress ──[resolve]──► Resolved ──[close]──► Closed
                     ▲                                              │
                     └────────────────────[reopen]─────────────────┘
```

---

## User Stories

### Submit a Ticket
- As a customer, I want to submit a ticket and have the system automatically set a reasonable due time, so that I know when to expect a reply
- As the system, I must check that the ticket title is not empty on submission, otherwise show a prompt
- As the system, the due time for a Critical ticket should be 2 hours, High priority is 8 hours, and others are 24 hours

### Assign for Handling
- As an agent, I want to claim an open ticket, changing its status to "In Progress" and showing that I am responsible
- As the system, only tickets in the "Open" status can be assigned

### Resolve a Ticket
- As an agent, when I solve the problem, I want to mark the ticket as "Resolved" and record the resolution time
- As the system, if the resolution time exceeds the promised due time, automatically mark it as "over SLA"

### Reopen a Ticket
- As a customer, if the issue isn't truly resolved, I want to reopen the ticket to request further help
- As the system, only "Resolved" or "Closed" tickets can be reopened

### Close a Ticket
- As the system, after confirmation, formally close a resolved ticket (terminal state, status can no longer change)

---

## Acceptance Criteria

- [ ] Empty title on submit → show error prompt, ticket does not enter Open
- [ ] Submit Critical ticket → SLA due time = current time + 2 hours
- [ ] Submit High ticket → SLA due time = current time + 8 hours
- [ ] Submit Low/Normal ticket → SLA due time = current time + 24 hours
- [ ] Assign: Open → In Progress, record responsible agent
- [ ] Assign a non-Open ticket → show warning prompt
- [ ] Resolve: In Progress → Resolved, record resolution time
- [ ] Resolved over SLA → IsOverSLA automatically set to true
- [ ] Reopen: Resolved/Closed → Open, clear resolution time
- [ ] Close: Resolved → Closed (terminal state)
