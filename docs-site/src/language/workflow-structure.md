# Workflow Structure

A workflow definition consists of a context parameter, an optional overview page, and a sequence of activities. Activities execute in order unless control flow elements (decisions, parallel splits, jumps) alter the sequence.

## CREATE WORKFLOW

```sql
CREATE [OR MODIFY] WORKFLOW <Module>.<Name>
  PARAMETER $<Name>: <Module>.<Entity>
  [OVERVIEW PAGE <Module>.<Page>]
BEGIN
  <activities>
END WORKFLOW;
```

| Element | Description |
|---------|-------------|
| `PARAMETER` | The workflow context object, passed when the workflow is started |
| `OVERVIEW PAGE` | Optional page shown in the workflow admin dashboard |
| Activities | A sequence of user tasks, microflow calls, decisions, etc. |

## Full Example

```sql
CREATE WORKFLOW HR.OnboardEmployee
  PARAMETER $Context: HR.OnboardingRequest
  OVERVIEW PAGE HR.OnboardingOverview
BEGIN
  -- Parallel: IT setup and HR paperwork happen simultaneously
  PARALLEL SPLIT
    PATH 1 {
      USER TASK SetupLaptop 'Set up laptop and accounts'
        PAGE HR.IT_SetupPage
        OUTCOMES 'Done' { };
    }
    PATH 2 {
      USER TASK SignDocuments 'Sign employment documents'
        PAGE HR.DocumentSignPage
        OUTCOMES 'Signed' { };
    };

  -- After both paths complete, manager reviews
  DECISION 'Manager approval required?'
    OUTCOMES 'Yes' {
      USER TASK ManagerReview 'Review onboarding'
        PAGE HR.ManagerReviewPage
        OUTCOMES 'Approve' { } 'Reject' {
          CALL MICROFLOW HR.ACT_RejectOnboarding;
          END;
        };
    } 'No' { };

  CALL MICROFLOW HR.ACT_CompleteOnboarding;
END WORKFLOW;
```

## DROP WORKFLOW

```sql
DROP WORKFLOW HR.OnboardEmployee;
```

## Workflow Access

Grant or revoke execute access to control who can start a workflow:

```sql
GRANT EXECUTE ON WORKFLOW HR.OnboardEmployee TO HR.Manager, HR.Admin;
REVOKE EXECUTE ON WORKFLOW HR.OnboardEmployee FROM HR.Manager;
```

## See Also

- [Workflows](./workflows.md) -- overview and when to use workflows
- [Activity Types](./workflow-activities.md) -- details on each activity type
- [Workflow vs Microflow](./workflow-vs-microflow.md) -- choosing the right construct
