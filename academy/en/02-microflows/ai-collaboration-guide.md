# Module 02: AI Collaboration Guide — Microflow Business Logic

## Prerequisite

First run the domain model:

```bash
mxcli exec academy/zh/01-领域建模/参考实现/domain-model.mdl -p MyProject.mpr
```

## Steps for Collaborating with Claude

### Step 1: Have Claude Implement the State Machine

```
Read academy/zh/02-微流业务逻辑/业务需求.md,
and help me implement the ticket business logic microflows in MDL:
- HD.ACT_Ticket_Submit (submit)
- HD.ACT_Ticket_Assign (assign)
- HD.ACT_Ticket_Resolve (resolve)
- HD.ACT_Ticket_Reopen (reopen)
- HD.ACT_Ticket_Close (close)
```

### Step 2: Validate SLA Calculation

SLA calculation uses the `addHours('[%CurrentDateTime%]', N)` function, where N can be a constant reference `@HD.SLA_HIGH_HOURS`.

Ask Claude to confirm that the generated microflow contains:

```mdl
change $Ticket (
  Status   = HD.TicketStatus.Open,
  SLADueAt = addHours('[%CurrentDateTime%]', @HD.SLA_CRITICAL_HOURS)
);
```

### Step 3: Validate the State Pre-Check

Every microflow should first check whether the current status is correct, for example:

```mdl
if $Ticket/Status != HD.TicketStatus.Open {
  show message 'Only Open tickets can be assigned.' type warning;
  return false;
}
```

### Step 4: Common Pitfalls

| Pitfall | Cause | Solution |
|---------|-------|----------|
| Constant reference error | Wrote `HD.SLA_HIGH_HOURS` instead of `@HD.SLA_HIGH_HOURS` | Add the `@` prefix |
| SLA calculation compares date only, not time | The over-SLA check missed `and $Ticket/SLADueAt != empty` | Add a null-value guard |
| ResolvedAt not cleared on reopen | Only changed Status | Also `change $Ticket (ResolvedAt = empty)` |

## Reference Implementation

See `参考实现/microflows.mdl`.
