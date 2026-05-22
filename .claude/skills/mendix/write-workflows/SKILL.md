# Writing Mendix Workflows in MDL

Workflows automate multi-step processes with human user tasks and microflow calls.
Mendix 11.2+ uses the `Workflows$` BSON namespace.

## Create a Workflow

```mdl
create workflow Module.WorkflowName
  parameter $WorkflowContext: Module.ParameterEntity
  display 'Human-readable title'
  export level Hidden           -- Hidden | Public
begin
  -- activities here
end workflow
```

## Activity Types

### Call Microflow

```mdl
call microflow Module.MicroflowName with (Param = '$WorkflowContext')
  outcomes
    default -> { };
```

### User Task

```mdl
user task taskName 'Display Title'
  page Module.UserTaskPage
  due date 'addDays([%CurrentDateTime%], 5)'
  targeting xpath '[System.UserRoles/Name = ''Module.RoleName'']'
  outcomes
    'Approved' { }
    'Rejected' {
      -- nested activities
    };
```

## User Task Targeting

Targeting controls which users can claim and complete a user task.

| MDL Syntax | BSON Type | Behaviour |
|------------|-----------|-----------|
| *(omitted)* | `Workflows$NoUserTargeting` | Any user can claim (default) |
| `targeting xpath '...'` | `Workflows$XPathUserTargeting` | Filter users by XPath |
| `targeting microflow Module.MF` | `Workflows$MicroflowUserTargeting` | MF returns eligible users |
| `targeting groups xpath '...'` | `Workflows$XPathGroupTargeting` | Filter by user group |
| `targeting groups microflow Module.MF` | `Workflows$MicroflowGroupTargeting` | MF returns eligible groups |

**Important:** Omitting `targeting` defaults to `NoUserTargeting` — valid BSON, no parse error.
Studio Pro will report CE1859 for `NoUserTargeting` but it does not crash `mx check`.

### Common XPath targeting patterns

```mdl
-- Current user only
targeting xpath '[%CurrentUser%]'

-- Users with a specific module role
targeting xpath '[System.UserRoles/Name = ''Module.RoleName'']'

-- Users linked to the workflow context object
targeting xpath '[Module.Entity_Account = $WorkflowContext/Module.Entity_Owner]'
```

## Jump (Loop Back)

```mdl
jump to taskName comment 'Reason for returning';
```

## Decisions (Route by Outcome)

```mdl
call microflow Module.RouteDecision with (Entity = '$WorkflowContext')
  outcomes
    'Module.Enum.ValueA' -> {
      user task taskA 'Handle A'
        page Module.PageA
        outcomes
          'Done' { };
    }
    'Module.Enum.ValueB' -> {
      user task taskB 'Handle B'
        page Module.PageB
        outcomes
          'Done' { };
    };
```

## Describe → Exec Roundtrip

`describe workflow` outputs fully re-executable MDL:

```bash
mxcli -p app.mpr -c "describe workflow Module.WF_Name"
# Copy output and exec against another project
```

**Known limitation:** `NoUserTargeting` is not emitted by describe (it's the default).
Re-executing the describe output produces identical BSON.

## User Task Page Requirement

The page assigned to a user task **must** accept a `System.WorkflowUserTask` parameter.
Without it, Studio Pro reports CE7410.

```mdl
create or modify page Module.UserTaskPage (
  Title: 'Review',
  Layout: Atlas_Core.Atlas_Default,
  Params: { $WorkflowUserTask: System.WorkflowUserTask }
) { }
```

## Full Example

```mdl
create workflow Module.WF_ApprovalFlow
  parameter $WorkflowContext: Module.Request
  display 'Approval Flow'
  export level Hidden
begin
  call microflow Module.SUB_OnStart with (Request = '$WorkflowContext')
    outcomes
      default -> { };

  user task reviewTask 'Review Request'
    page Module.UserTask_Review
    targeting xpath '[System.UserRoles/Name = ''Module.Manager'']'
    due date 'addDays([%CurrentDateTime%], 3)'
    outcomes
      'Approved' {
        call microflow Module.SUB_OnApproved with (Request = '$WorkflowContext')
          outcomes
            default -> { };
      }
      'Rejected' {
        call microflow Module.SUB_OnRejected with (Request = '$WorkflowContext')
          outcomes
            default -> { };
      };
end workflow
```

## ALTER Workflow

```mdl
-- Change due date on a user task
alter workflow Module.WF_Name
  set reviewTask due date 'addDays([%CurrentDateTime%], 7)';

-- Change targeting
alter workflow Module.WF_Name
  set reviewTask targeting xpath '[System.UserRoles/Name = ''Module.Admin'']';
```

## expr Checker Coverage

`mxcli expr` scans the following workflow expression fields:
- `Workflows$MicroflowCallParameterMapping.Expression` — call parameter arguments
- `Workflows$SingleUserTaskActivity.DueDate` — due date expressions
- `Workflows$XPathUserTargeting.XPathConstraint` — user XPath targeting
- `Workflows$XPathGroupTargeting.XPathConstraint` — group XPath targeting
