# Module 07: Escalation Workflow — Business Requirements

## Business Context

Some ticket issues exceed an ordinary agent's ability to handle — for example, finance-related payment problems, or failures that require coordination with an external vendor.
In these cases an **escalation approval** mechanism is needed: the agent submits a request, and the manager decides whether to raise the ticket's priority.

This is a very common scenario in IT support; support teams of any size have a similar process.

---

## User Stories

### Agent Initiates Escalation
- As an agent, when I encounter a problem I cannot handle, I want to submit an escalation request to the manager, explaining the reason
- As a system, an escalation request can only target a ticket in the "In Progress" status
- As a system, a ticket can only have one pending escalation request at a time

### Manager Approval
- As a manager, I want to see all pending escalation requests, sorted by priority
- As a manager, I can approve a request (the ticket priority is automatically raised to "Critical")
- As a manager, I can reject a request (entering a rejection reason; the ticket status stays unchanged)

### Escalation Outcome
- As an agent, after my escalation request is approved, I can see that the ticket priority has changed to "Critical"
- As an agent, after my escalation request is rejected, I can see the rejection reason

---

## Acceptance Criteria

- [ ] An escalation request contains: reason text, request time (automatic), approval status (Pending/Approved/Rejected)
- [ ] Only an "In Progress" ticket can have an escalation request initiated
- [ ] After approval: the ticket Priority becomes Critical
- [ ] After rejection: RejectionReason is recorded, the ticket is unchanged
- [ ] Escalation overview page: the manager can see all pending requests
- [ ] Approval form: contains a ticket information summary + approve/reject buttons + rejection reason input field
