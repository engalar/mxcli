# Workflows

## When to Use This Skill

Use this skill when the user wants to:
- Explore existing workflows in a Mendix project
- Create new workflows with user tasks, decisions, and microflow calls
- Understand workflow structure and activity types
- Navigate workflow cross-references (callers, callees, impact)

## Exploring Workflows

### List All Workflows

```sql
SHOW WORKFLOWS;
SHOW WORKFLOWS IN MyModule;
```

### Describe a Workflow

```sql
DESCRIBE WORKFLOW Module.WorkflowName;
```

Shows the full workflow definition: parameter entity, activities, outcomes, user targeting, boundary events.

### Catalog Queries

```sql
REFRESH CATALOG FULL;

-- All workflows
SELECT * FROM CATALOG.WORKFLOWS;

-- Workflows with user tasks
SELECT QualifiedName, ActivityCount, UserTaskCount, DecisionCount
  FROM CATALOG.WORKFLOWS WHERE UserTaskCount > 0;

-- Workflows in a specific module
SELECT QualifiedName, ParameterEntity
  FROM CATALOG.WORKFLOWS WHERE ModuleName = 'MyModule';
```

### Code Navigation

```sql
-- What calls this workflow
SHOW CALLERS OF Module.WorkflowName;

-- All references to this workflow
SHOW REFERENCES TO Module.WorkflowName;

-- Impact analysis (what would break if changed)
SHOW IMPACT OF Module.WorkflowName;

-- Full context for analysis
SHOW CONTEXT OF Module.WorkflowName;
```

## Creating Workflows

### Basic Structure

```sql
CREATE WORKFLOW Module.ApprovalFlow
  PARAMETER $WorkflowContext: Module.Request
  OVERVIEW PAGE Module.ApprovalOverview
  DUE DATE 'addDays([%CurrentDateTime%], 7)'
BEGIN
  <activities>
END WORKFLOW;
```

Use `CREATE OR REPLACE` to overwrite an existing workflow.

### Activity Types

#### User Task

Assigns work to a user with a task page and outcomes.

```sql
USER TASK ReviewTask 'Review the request'
  PAGE Module.ReviewPage
  TARGETING MICROFLOW Module.DS_GetReviewers
  ENTITY Module.ReviewTask
  DUE DATE 'addDays([%CurrentDateTime%], 3)'
  OUTCOMES
    'Approve' { }
    'Reject' { };
```

- `PAGE` — the task page shown to the assigned user
- `TARGETING MICROFLOW` — microflow that returns the list of candidate users
- `TARGETING XPATH` — XPath constraint to select candidate users
- `ENTITY` — optional task-specific entity (for task-level data)
- `OUTCOMES` — each outcome can contain nested activities in `{ }`

For multi-user tasks (all assignees must complete):

```sql
MULTI USER TASK ParallelReview 'All reviewers must approve'
  PAGE Module.ReviewPage
  TARGETING MICROFLOW Module.DS_GetAllReviewers
  OUTCOMES
    'Approve' { }
    'Reject' { };
```

#### Call Microflow

Executes a microflow as an automated step. Parameters are auto-bound to `$WorkflowContext`.

```sql
CALL MICROFLOW Module.ACT_ProcessRequest;
```

With explicit parameter mappings:

```sql
CALL MICROFLOW Module.ACT_SendNotification
  WITH (Module.ACT_SendNotification.Recipient = '$WorkflowContext/Module.Request_Recipient');
```

With conditional outcomes (when the microflow returns Boolean or Enumeration):

```sql
CALL MICROFLOW Module.ACT_ValidateRequest
  OUTCOMES
    'True' { }
    'False' { };
```

#### Call Workflow (Sub-workflow)

Calls another workflow. The `$WorkflowContext` parameter is auto-bound.

```sql
CALL WORKFLOW Module.SubApproval;
```

#### Decision (Exclusive Split)

Branch based on a Boolean or Enumeration expression.

```sql
-- Boolean decision
DECISION '$WorkflowContext/Amount > 10000'
  OUTCOMES
    'True' {
      USER TASK ManagerApproval 'Manager approval required'
        PAGE Module.ApprovalPage
        OUTCOMES 'Approve' { } 'Reject' { };
    }
    'False' { };

-- Enumeration decision
DECISION '$WorkflowContext/Priority'
  OUTCOMES
    'Module.Priority.High' {
      CALL MICROFLOW Module.ACT_EscalateRequest;
    }
    'Module.Priority.Low' { }
    'Default' { };
```

#### Parallel Split

Execute multiple paths concurrently. All paths must complete before continuing.

```sql
PARALLEL SPLIT
  PATH 1 {
    USER TASK LegalReview 'Legal review'
      PAGE Module.LegalReviewPage
      OUTCOMES 'Approve' { } 'Reject' { };
  }
  PATH 2 {
    USER TASK FinanceReview 'Finance review'
      PAGE Module.FinanceReviewPage
      OUTCOMES 'Approve' { } 'Reject' { };
  };
```

#### Wait Activities

```sql
-- Wait for a timer expression
WAIT FOR TIMER 'addDays([%CurrentDateTime%], 1)';

-- Wait for an external notification
WAIT FOR NOTIFICATION;
```

#### Boundary Events

Attach timer-based interrupts to user tasks, call activities, or wait activities.

```sql
USER TASK ReviewTask 'Review request'
  PAGE Module.ReviewPage
  OUTCOMES 'Approve' { } 'Reject' { }
  BOUNDARY EVENT INTERRUPTING TIMER 'addDays([%CurrentDateTime%], 3)' {
    CALL MICROFLOW Module.ACT_EscalateOverdue;
  };
```

Event types: `INTERRUPTING TIMER`, `NON INTERRUPTING TIMER`.

#### Jump To

Jump back to a named activity (for loops/retries).

```sql
JUMP TO ReviewTask;
```

#### End

Explicitly end a path (implicit at workflow end, but useful in branches).

```sql
END;
```

#### Annotation

Add a sticky-note comment to the workflow canvas.

```sql
ANNOTATION 'This section handles escalation logic';
```

### Drop a Workflow

```sql
DROP WORKFLOW Module.ApprovalFlow;
```

## Complete Example

```sql
CREATE WORKFLOW HR.OnboardingWorkflow
  PARAMETER $WorkflowContext: HR.OnboardingRequest
  OVERVIEW PAGE HR.OnboardingOverview
  DUE DATE 'addDays([%CurrentDateTime%], 30)'
BEGIN
  CALL MICROFLOW HR.ACT_PrepareOnboarding;

  PARALLEL SPLIT
    PATH 1 {
      USER TASK ITSetup 'Set up IT accounts'
        PAGE HR.ITSetupPage
        TARGETING MICROFLOW HR.DS_GetITTeam
        OUTCOMES 'Done' { };
    }
    PATH 2 {
      USER TASK HRPaperwork 'Complete HR paperwork'
        PAGE HR.PaperworkPage
        TARGETING MICROFLOW HR.DS_GetHRTeam
        OUTCOMES 'Done' { };
    };

  DECISION '$WorkflowContext/RequiresTraining'
    OUTCOMES
      'True' {
        USER TASK ScheduleTraining 'Schedule training'
          PAGE HR.TrainingPage
          OUTCOMES 'Scheduled' { };
      }
      'False' { };

  CALL MICROFLOW HR.ACT_CompleteOnboarding;
END WORKFLOW;
```

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Duplicate activity names | Names are auto-deduplicated (CE0495), but use unique names for clarity |
| Missing parameter entity | Always specify `PARAMETER $WorkflowContext: Module.Entity` |
| Bare entity in TARGETING | Use qualified name: `TARGETING MICROFLOW Module.MicroflowName` |
| Forgetting outcomes on user tasks | Every user task needs at least one outcome |
| Using `'Approve'`/`'Reject'` as decision outcomes | Decision outcomes must be `'True'`/`'False'` for Boolean, or enum values |

## Checklist

- [ ] Parameter entity exists in the domain model
- [ ] Overview page exists (if specified)
- [ ] All referenced microflows exist
- [ ] All referenced pages exist
- [ ] User task pages exist and are configured for the task entity
- [ ] Targeting microflows return a list of `System.User`
- [ ] Each user task has at least one outcome
- [ ] Decision expressions match the outcome type (Boolean → True/False, Enum → values)
- [ ] Use `DESCRIBE WORKFLOW` to verify the created workflow
