// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.g tests for the gen-typed Workflow / Misc family
// formatters. All tests are direct-construction — the
// `expr-checker/minimal.mpr` fixture has no workflow content, so no
// fixture-driven coverage is added here.

package executor

import (
	"testing"

	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// ────────────────────────────────────────────────────────
// GetWorkflowDataAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_GetWorkflowDataAction(t *testing.T) {
	t.Run("with output variable", func(t *testing.T) {
		a := genMf.NewGetWorkflowDataAction()
		a.SetOutputVariableName("Data")
		a.SetWorkflowVariable("Wf")
		a.SetWorkflowQualifiedName("Mod.OnboardingWorkflow")
		got := formatActionGen(nil, a)
		want := "$Data = get workflow data $Wf as Mod.OnboardingWorkflow;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("without output variable", func(t *testing.T) {
		a := genMf.NewGetWorkflowDataAction()
		a.SetWorkflowVariable("Wf")
		a.SetWorkflowQualifiedName("Mod.OnboardingWorkflow")
		got := formatActionGen(nil, a)
		want := "get workflow data $Wf as Mod.OnboardingWorkflow;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty workflow QN still renders", func(t *testing.T) {
		a := genMf.NewGetWorkflowDataAction()
		a.SetWorkflowVariable("W")
		got := formatActionGen(nil, a)
		want := "get workflow data $W as ;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// WorkflowCallAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_WorkflowCallAction(t *testing.T) {
	t.Run("with return variable and output", func(t *testing.T) {
		a := genMf.NewWorkflowCallAction()
		a.SetWorkflowQualifiedName("Mod.Process")
		a.SetWorkflowContextVariable("Order")
		a.SetUseReturnVariable(true)
		a.SetOutputVariableName("Wf")
		got := formatActionGen(nil, a)
		// Grammar requires callArgument: (VARIABLE | parameterName) EQUALS expression.
		// Write path stores only the variable name; use it as both param name and value.
		want := "$Wf = call workflow Mod.Process (Order = $Order);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("UseReturnVariable false renders bare form", func(t *testing.T) {
		a := genMf.NewWorkflowCallAction()
		a.SetWorkflowQualifiedName("Mod.Process")
		a.SetWorkflowContextVariable("Order")
		a.SetUseReturnVariable(false)
		a.SetOutputVariableName("Wf") // ignored when UseReturnVariable is false
		got := formatActionGen(nil, a)
		want := "call workflow Mod.Process (Order = $Order);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("UseReturnVariable true but no output var falls back", func(t *testing.T) {
		a := genMf.NewWorkflowCallAction()
		a.SetWorkflowQualifiedName("Mod.Process")
		a.SetWorkflowContextVariable("Order")
		a.SetUseReturnVariable(true)
		got := formatActionGen(nil, a)
		want := "call workflow Mod.Process (Order = $Order);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty context variable emits empty arg list", func(t *testing.T) {
		a := genMf.NewWorkflowCallAction()
		a.SetWorkflowQualifiedName("Mod.Process")
		got := formatActionGen(nil, a)
		want := "call workflow Mod.Process ();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// GetWorkflowsAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_GetWorkflowsAction(t *testing.T) {
	t.Run("with output variable", func(t *testing.T) {
		a := genMf.NewGetWorkflowsAction()
		a.SetOutputVariableName("WfList")
		a.SetWorkflowContextVariableName("Order")
		got := formatActionGen(nil, a)
		want := "$WfList = get workflows for $Order;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("without output variable", func(t *testing.T) {
		a := genMf.NewGetWorkflowsAction()
		a.SetWorkflowContextVariableName("Order")
		got := formatActionGen(nil, a)
		want := "get workflows for $Order;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// GetWorkflowActivityRecordsAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_GetWorkflowActivityRecordsAction(t *testing.T) {
	t.Run("with output variable", func(t *testing.T) {
		a := genMf.NewGetWorkflowActivityRecordsAction()
		a.SetOutputVariableName("Records")
		a.SetWorkflowVariable("Wf")
		got := formatActionGen(nil, a)
		want := "$Records = get workflow activity records $Wf;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("without output variable", func(t *testing.T) {
		a := genMf.NewGetWorkflowActivityRecordsAction()
		a.SetWorkflowVariable("Wf")
		got := formatActionGen(nil, a)
		want := "get workflow activity records $Wf;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// WorkflowOperationAction (with sub-op dispatch)
// ────────────────────────────────────────────────────────

func TestFormatActionGen_WorkflowOperationAction_Abort(t *testing.T) {
	t.Run("abort with reason", func(t *testing.T) {
		// The write path uses exprToString which already wraps the expression in quotes
		// (e.g. 'payment failed'), so StringTemplate.Text stores the full expression string.
		// The describe code must emit it as-is without additional quoting.
		a := genMf.NewWorkflowOperationAction()
		op := genMf.NewAbortOperation()
		op.SetWorkflowVariable("Wf")
		tmpl := genMf.NewStringTemplate()
		tmpl.SetText("'payment failed'") // expression string as stored by the write path
		op.SetReason(tmpl)
		a.SetOperation(op)
		got := formatActionGen(nil, a)
		want := "workflow operation abort $Wf reason 'payment failed';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("abort without reason (nil Reason element)", func(t *testing.T) {
		a := genMf.NewWorkflowOperationAction()
		op := genMf.NewAbortOperation()
		op.SetWorkflowVariable("Wf")
		a.SetOperation(op)
		got := formatActionGen(nil, a)
		want := "workflow operation abort $Wf;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("abort with empty reason text suppresses clause", func(t *testing.T) {
		a := genMf.NewWorkflowOperationAction()
		op := genMf.NewAbortOperation()
		op.SetWorkflowVariable("Wf")
		tmpl := genMf.NewStringTemplate()
		tmpl.SetText("   ") // only whitespace
		op.SetReason(tmpl)
		a.SetOperation(op)
		got := formatActionGen(nil, a)
		want := "workflow operation abort $Wf;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("abort with reason containing single quote escapes", func(t *testing.T) {
		// exprToString("user's request") → 'user''s request' (escaped expression string)
		a := genMf.NewWorkflowOperationAction()
		op := genMf.NewAbortOperation()
		op.SetWorkflowVariable("Wf")
		tmpl := genMf.NewStringTemplate()
		tmpl.SetText("'user''s request'") // expression string as stored by the write path
		op.SetReason(tmpl)
		a.SetOperation(op)
		got := formatActionGen(nil, a)
		want := "workflow operation abort $Wf reason 'user''s request';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// TestFormatActionGen_WorkflowOperationAction_OtherOps_Direct exercises
// each non-abort sub-op via direct typed construction. Each sub-op
// shares the same `workflow operation <verb> $Wf;` shape.
func TestFormatActionGen_WorkflowOperationAction_OtherOps_Direct(t *testing.T) {
	mkAction := func(setOp func(a *genMf.WorkflowOperationAction)) *genMf.WorkflowOperationAction {
		a := genMf.NewWorkflowOperationAction()
		setOp(a)
		return a
	}

	t.Run("continue", func(t *testing.T) {
		a := mkAction(func(a *genMf.WorkflowOperationAction) {
			op := genMf.NewContinueOperation()
			op.SetWorkflowVariable("Wf")
			a.SetOperation(op)
		})
		got := formatActionGen(nil, a)
		if want := "workflow operation continue $Wf;"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("pause", func(t *testing.T) {
		a := mkAction(func(a *genMf.WorkflowOperationAction) {
			op := genMf.NewPauseOperation()
			op.SetWorkflowVariable("Wf")
			a.SetOperation(op)
		})
		got := formatActionGen(nil, a)
		if want := "workflow operation pause $Wf;"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("restart", func(t *testing.T) {
		a := mkAction(func(a *genMf.WorkflowOperationAction) {
			op := genMf.NewRestartOperation()
			op.SetWorkflowVariable("Wf")
			a.SetOperation(op)
		})
		got := formatActionGen(nil, a)
		if want := "workflow operation restart $Wf;"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("retry", func(t *testing.T) {
		a := mkAction(func(a *genMf.WorkflowOperationAction) {
			op := genMf.NewRetryOperation()
			op.SetWorkflowVariable("Wf")
			a.SetOperation(op)
		})
		got := formatActionGen(nil, a)
		if want := "workflow operation retry $Wf;"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("unpause", func(t *testing.T) {
		a := mkAction(func(a *genMf.WorkflowOperationAction) {
			op := genMf.NewUnpauseOperation()
			op.SetWorkflowVariable("Wf")
			a.SetOperation(op)
		})
		got := formatActionGen(nil, a)
		if want := "workflow operation unpause $Wf;"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("resume renders as unpause for legacy parity", func(t *testing.T) {
		a := mkAction(func(a *genMf.WorkflowOperationAction) {
			op := genMf.NewResumeOperation()
			op.SetWorkflowVariable("Wf")
			a.SetOperation(op)
		})
		got := formatActionGen(nil, a)
		if want := "workflow operation unpause $Wf;"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_WorkflowOperationAction_NilOperation(t *testing.T) {
	a := genMf.NewWorkflowOperationAction()
	got := formatActionGen(nil, a)
	want := "workflow operation ...;"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ────────────────────────────────────────────────────────
// SetTaskOutcomeAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_SetTaskOutcomeAction(t *testing.T) {
	t.Run("simple outcome", func(t *testing.T) {
		a := genMf.NewSetTaskOutcomeAction()
		a.SetWorkflowTaskVariable("Task")
		a.SetOutcomeValue("Approved")
		got := formatActionGen(nil, a)
		want := "set task outcome $Task 'Approved';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("outcome with single quotes is escaped", func(t *testing.T) {
		a := genMf.NewSetTaskOutcomeAction()
		a.SetWorkflowTaskVariable("Task")
		a.SetOutcomeValue("won't fix")
		got := formatActionGen(nil, a)
		want := "set task outcome $Task 'won''t fix';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty outcome", func(t *testing.T) {
		a := genMf.NewSetTaskOutcomeAction()
		a.SetWorkflowTaskVariable("Task")
		got := formatActionGen(nil, a)
		want := "set task outcome $Task '';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// OpenUserTaskAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_OpenUserTaskAction(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		a := genMf.NewOpenUserTaskAction()
		a.SetUserTaskVariable("Task")
		got := formatActionGen(nil, a)
		want := "open user task $Task;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("AssignOnOpen / OpenWhenAssigned ignored for legacy parity", func(t *testing.T) {
		a := genMf.NewOpenUserTaskAction()
		a.SetUserTaskVariable("Task")
		a.SetAssignOnOpen(true)
		a.SetOpenWhenAssigned(true)
		got := formatActionGen(nil, a)
		want := "open user task $Task;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// NotifyWorkflowAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_NotifyWorkflowAction(t *testing.T) {
	t.Run("with output variable", func(t *testing.T) {
		a := genMf.NewNotifyWorkflowAction()
		a.SetOutputVariableName("Result")
		a.SetWorkflowVariable("Wf")
		got := formatActionGen(nil, a)
		want := "$Result = notify workflow $Wf;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("without output variable", func(t *testing.T) {
		a := genMf.NewNotifyWorkflowAction()
		a.SetWorkflowVariable("Wf")
		got := formatActionGen(nil, a)
		want := "notify workflow $Wf;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with activity qualified name", func(t *testing.T) {
		a := genMf.NewNotifyWorkflowAction()
		a.SetOutputVariableName("IsReceived")
		a.SetWorkflowVariable("Wf")
		a.SetActivityQualifiedName("HD.WF_TicketEscalation.WaitForManagerAvailable")
		got := formatActionGen(nil, a)
		want := "$IsReceived = notify workflow $Wf activity HD.WF_TicketEscalation.WaitForManagerAvailable;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// OpenWorkflowAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_OpenWorkflowAction(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		a := genMf.NewOpenWorkflowAction()
		a.SetWorkflowVariable("Wf")
		got := formatActionGen(nil, a)
		want := "open workflow $Wf;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// LockWorkflowAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_LockWorkflowAction(t *testing.T) {
	t.Run("PauseAllWorkflows renders 'all'", func(t *testing.T) {
		a := genMf.NewLockWorkflowAction()
		a.SetPauseAllWorkflows(true)
		// even when other fields are set, 'all' takes precedence.
		a.SetWorkflowQualifiedName("Mod.Ignored")
		got := formatActionGen(nil, a)
		want := "lock workflow all;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with NameSelection renders qualified name", func(t *testing.T) {
		a := genMf.NewLockWorkflowAction()
		sel := genWf.NewWorkflowDefinitionNameSelection()
		sel.SetWorkflowQualifiedName("Mod.MyWorkflow")
		a.SetWorkflowSelection(sel)
		got := formatActionGen(nil, a)
		want := "lock workflow Mod.MyWorkflow;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with ObjectSelection renders variable", func(t *testing.T) {
		a := genMf.NewLockWorkflowAction()
		sel := genWf.NewWorkflowDefinitionObjectSelection()
		sel.SetWorkflowDefinitionVariable("WfDef")
		a.SetWorkflowSelection(sel)
		got := formatActionGen(nil, a)
		want := "lock workflow $WfDef;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to top-level WorkflowQualifiedName when no Selection", func(t *testing.T) {
		a := genMf.NewLockWorkflowAction()
		a.SetWorkflowQualifiedName("Mod.OtherWorkflow")
		got := formatActionGen(nil, a)
		want := "lock workflow Mod.OtherWorkflow;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// UnlockWorkflowAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_UnlockWorkflowAction(t *testing.T) {
	t.Run("ResumeAllPausedWorkflows renders 'all'", func(t *testing.T) {
		a := genMf.NewUnlockWorkflowAction()
		a.SetResumeAllPausedWorkflows(true)
		a.SetWorkflowQualifiedName("Mod.Ignored")
		got := formatActionGen(nil, a)
		want := "unlock workflow all;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with NameSelection renders qualified name", func(t *testing.T) {
		a := genMf.NewUnlockWorkflowAction()
		sel := genWf.NewWorkflowDefinitionNameSelection()
		sel.SetWorkflowQualifiedName("Mod.MyWorkflow")
		a.SetWorkflowSelection(sel)
		got := formatActionGen(nil, a)
		want := "unlock workflow Mod.MyWorkflow;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with ObjectSelection renders variable", func(t *testing.T) {
		a := genMf.NewUnlockWorkflowAction()
		sel := genWf.NewWorkflowDefinitionObjectSelection()
		sel.SetWorkflowDefinitionVariable("WfDef")
		a.SetWorkflowSelection(sel)
		got := formatActionGen(nil, a)
		want := "unlock workflow $WfDef;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("falls back to top-level WorkflowQualifiedName", func(t *testing.T) {
		a := genMf.NewUnlockWorkflowAction()
		a.SetWorkflowQualifiedName("Mod.OtherWorkflow")
		got := formatActionGen(nil, a)
		want := "unlock workflow Mod.OtherWorkflow;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// GenerateJumpToOptionsAction
// ────────────────────────────────────────────────────────

func TestFormatGenerateJumpToOptionsActionGen(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*genMf.GenerateJumpToOptionsAction)
		expected string
	}{
		{
			name: "with output variable",
			setup: func(a *genMf.GenerateJumpToOptionsAction) {
				a.SetWorkflowVariable("Workflow")
				a.SetWorkflowQualifiedName("HD.WF_TicketEscalation")
				a.SetOutputVariableName("JumpOptions")
			},
			expected: "$JumpOptions = generate jump to options for $Workflow as HD.WF_TicketEscalation;",
		},
		{
			name: "without output variable",
			setup: func(a *genMf.GenerateJumpToOptionsAction) {
				a.SetWorkflowVariable("Workflow")
				a.SetWorkflowQualifiedName("HD.WF_TicketEscalation")
			},
			expected: "generate jump to options for $Workflow as HD.WF_TicketEscalation;",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := genMf.NewGenerateJumpToOptionsAction()
			tt.setup(a)
			got := formatGenerateJumpToOptionsActionGen(a)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

// ────────────────────────────────────────────────────────
// ApplyJumpToOptionAction
// ────────────────────────────────────────────────────────

func TestFormatApplyJumpToOptionActionGen(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*genMf.ApplyJumpToOptionAction)
		expected string
	}{
		{
			name: "with output variable",
			setup: func(a *genMf.ApplyJumpToOptionAction) {
				a.SetWorkflowJumpToDetailsVariable("JumpOptions")
				a.SetOutputVariableName("Result")
			},
			expected: "$Result = apply jump to option $JumpOptions;",
		},
		{
			name: "without output variable",
			setup: func(a *genMf.ApplyJumpToOptionAction) {
				a.SetWorkflowJumpToDetailsVariable("JumpOptions")
			},
			expected: "apply jump to option $JumpOptions;",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := genMf.NewApplyJumpToOptionAction()
			tt.setup(a)
			got := formatApplyJumpToOptionActionGen(a)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
