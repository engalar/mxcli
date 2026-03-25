# Workflow vs Microflow

Workflows and microflows are both used to express business logic, but they serve different purposes and have different execution models.

## Comparison

| Aspect | Microflow | Workflow |
|--------|-----------|----------|
| **Duration** | Milliseconds to seconds | Hours, days, or weeks |
| **Execution** | Synchronous, single transaction | Asynchronous, persisted state |
| **Trigger** | Button click, page load, event | Explicitly started, resumes on events |
| **Human tasks** | Not supported (shows pages but does not wait) | Built-in user task with outcomes |
| **Branching** | IF / ELSE within a single execution | Decisions with separate outcome paths |
| **Parallelism** | Not supported | Parallel splits with multiple concurrent paths |
| **Persistence** | No state between calls | State persisted across restarts |
| **Monitoring** | Log messages only | Admin overview page, task inbox |
| **Error handling** | ON ERROR blocks | Outcome-based routing |

## When to Use a Microflow

Use a microflow when:

- The operation completes immediately (create, update, delete, calculate)
- No human decision points are needed
- The logic runs within a single database transaction
- You need fine-grained error handling with rollback

```sql
CREATE MICROFLOW Shop.ACT_ProcessPayment
BEGIN
  DECLARE $Order Shop.Order;
  RETRIEVE $Order FROM Shop.Order WHERE [OrderId = $OrderId] LIMIT 1;
  CHANGE $Order (Status = 'Paid', PaidDate = [%CurrentDateTime%]);
  COMMIT $Order;
  RETURN true;
END;
```

## When to Use a Workflow

Use a workflow when:

- The process involves **human approval or review steps**
- The process spans **multiple days or longer**
- You need a **visual overview** of where each case stands
- Multiple steps should happen **in parallel**
- The process may **wait for external events**

```sql
CREATE WORKFLOW Shop.OrderApproval
  PARAMETER $Context: Shop.Order
  OVERVIEW PAGE Shop.OrderWorkflowOverview
BEGIN
  USER TASK ReviewOrder 'Review the order'
    PAGE Shop.OrderReviewPage
    OUTCOMES 'Approve' {
      CALL MICROFLOW Shop.ACT_ApproveOrder;
    } 'Reject' {
      CALL MICROFLOW Shop.ACT_RejectOrder;
      END;
    };
  CALL MICROFLOW Shop.ACT_FulfillOrder;
END WORKFLOW;
```

## Combining Both

Workflows and microflows work together. A workflow orchestrates the process while microflows handle the actual logic at each step:

1. **Workflow** defines the process: who needs to act, in what order, with what outcomes
2. **Microflows** execute within workflow steps: create objects, send emails, update status
3. **Pages** provide the UI for user tasks

A common pattern is:

```sql
-- Microflow does the work
CREATE MICROFLOW Approval.ACT_Approve
BEGIN
  DECLARE $Request Approval.Request;
  RETRIEVE $Request FROM Approval.Request WHERE [id = $RequestId] LIMIT 1;
  CHANGE $Request (Status = 'Approved', ApprovedBy = '[%CurrentUser%]');
  COMMIT $Request;
END;

-- Workflow orchestrates the process
CREATE WORKFLOW Approval.ApprovalProcess
  PARAMETER $Context: Approval.Request
BEGIN
  USER TASK Review 'Review request'
    PAGE Approval.ReviewPage
    OUTCOMES 'Approve' {
      CALL MICROFLOW Approval.ACT_Approve;
    } 'Reject' {
      CALL MICROFLOW Approval.ACT_Reject;
      END;
    };
END WORKFLOW;
```

## See Also

- [Workflows](./workflows.md) -- workflow overview
- [Workflow Structure](./workflow-structure.md) -- CREATE WORKFLOW syntax
- [Activity Types](./workflow-activities.md) -- workflow activity reference
- [Microflows and Nanoflows](./microflows.md) -- microflow overview
