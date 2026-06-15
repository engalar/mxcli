# Module 07: AI Collaboration Guide — Escalation Workflow

## Design Choice for This Module

**This module implements escalation approval using a microflow state machine, not the Mendix Workflow Engine.**

Reason: the Workflow Engine's MDL syntax is complex and has a steep learning curve. A microflow state machine clearly demonstrates the approval logic, is the most common implementation you will see in Mendix projects, and is also the foundation for understanding the Workflow Engine.

For the Mendix Workflow Engine MDL implementation, see the advanced documentation (link to be added).

## Steps to Collaborate with Claude

### Step 1: Escalation Request Entity

```
Help me implement the HD.EscalationRequest entity in MDL:
- Reason: string, not empty, the escalation reason
- RequestedAt: DateTime, the request time
- ApprovalStatus: enumeration (Pending/Approved/Rejected), default Pending
- RejectionReason: string, nullable

Association: HD.EscalationRequest_Ticket (from EscalationRequest to Ticket)
```

### Step 2: Three Approval Microflows

```
Help me implement three microflows:
1. HD.ACT_StartEscalation($Ticket, $Reason):
   - Precondition: Ticket.Status = InProgress
   - Create an EscalationRequest, set Reason and RequestedAt

2. HD.ACT_Escalation_Approve($EscalationRequest):
   - Set ApprovalStatus = Approved
   - Change the associated ticket's Priority to Critical
   - Commit both objects

3. HD.ACT_Escalation_Reject($EscalationRequest, $Reason):
   - Set ApprovalStatus = Rejected, RejectionReason = $Reason
   - Commit
```

### Step 3: The Correct Way to Write Popup Pages (Key)

A Mendix dataview **cannot** use the `datasource: new HD.Entity` syntax.
The correct approach is: first create the object with a microflow, then `show page` passing the object as a parameter, and the page binds it with `datasource: $parameter`.

Both popups in this module follow this pattern:

| Popup | Microflow that opens it | Page parameter | Save action |
|-------|------------------------|----------------|-------------|
| EscalationStart_Form | `HD.ACT_OpenEscalationForm($Ticket)` | `$EscalationRequest` | `microflow HD.ACT_StartEscalation_FromObject` |
| EscalationReject_Form | `HD.ACT_OpenRejectForm($EscalationRequest)` | `$EscalationRequest` | `microflow HD.ACT_Escalation_Reject_FromObject` |

`ACT_OpenEscalationForm` validates the ticket status (must be InProgress) **before** creating the object, so the precondition check happens at the step of opening the form rather than at the save step — a better user experience.

### Step 4: Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| `datasource: new` syntax is invalid | Use the three-part pattern "object-creating microflow + show page + datasource: $parameter" |
| Modifying an associated object's attributes | Retrieve along the association path (the association name is the path, with no trailing entity segment and no limit): `retrieve $Ticket from $EscalationRequest/HD.EscalationRequest_Ticket` |
| Commit order | Commit EscalationRequest first, then commit Ticket |
| How to collect the rejection reason | Directly edit the RejectionReason field that EscalationRequest already has; no extra non-persistent entity is needed |
