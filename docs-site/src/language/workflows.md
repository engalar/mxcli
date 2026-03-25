# Workflows

Workflows model long-running, multi-step business processes such as approval chains, onboarding sequences, and review cycles. Unlike microflows, which execute synchronously in a single transaction, workflows persist their state and can pause indefinitely -- waiting for user input, timers, or external notifications.

## When to Use Workflows

Workflows are appropriate when a process:

- Involves **human tasks** that require user action at specific steps
- Spans **hours, days, or weeks** rather than completing instantly
- Has **branching logic** based on user decisions (approve / reject / escalate)
- Needs a **visual overview** of progress for administrators

For immediate, transactional logic, use [microflows](./microflows.md) instead. See [Workflow vs Microflow](./workflow-vs-microflow.md) for a detailed comparison.

## Inspecting Workflows

```sql
-- List all workflows
SHOW WORKFLOWS;
SHOW WORKFLOWS IN MyModule;

-- View full definition
DESCRIBE WORKFLOW MyModule.ApprovalFlow;
```

## Quick Example

```sql
CREATE WORKFLOW Approval.RequestApproval
  PARAMETER $Context: Approval.Request
  OVERVIEW PAGE Approval.WorkflowOverview
BEGIN
  USER TASK ReviewTask 'Review the request'
    PAGE Approval.ReviewPage
    OUTCOMES 'Approve' {
      CALL MICROFLOW Approval.ACT_Approve;
    } 'Reject' {
      CALL MICROFLOW Approval.ACT_Reject;
    };
END WORKFLOW;
```

## See Also

- [Workflow Structure](./workflow-structure.md) -- full CREATE WORKFLOW syntax
- [Activity Types](./workflow-activities.md) -- all workflow activity types
- [Workflow vs Microflow](./workflow-vs-microflow.md) -- choosing between the two
- [GRANT / REVOKE](./grant-revoke.md) -- granting execute access on workflows
