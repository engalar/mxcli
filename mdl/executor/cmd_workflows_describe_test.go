// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// formatSingleActivityGen is a test helper that wraps a single gen-typed
// activity in a Flow and runs formatWorkflowActivitiesGen to get the
// DESCRIBE output. Stage 3.3.3.E0 — replaces the legacy sdk-typed
// formatSingleActivity helper.
func formatSingleActivityGen(act element.Element, indent string) []string {
	flow := genWf.NewFlow()
	flow.AddActivities(act)
	return formatWorkflowActivitiesGen(flow, indent)
}

// --- P0: strip Module.Microflow prefix from parameter names ---

func TestFormatCallMicroflowTask_ParameterNameStripping(t *testing.T) {
	tests := []struct {
		name          string
		paramName     string
		wantParamName string
	}{
		{
			name:          "three-segment name strips prefix",
			paramName:     "WorkflowBaseline.CallMF.Entity",
			wantParamName: "Entity",
		},
		{
			name:          "two-segment name strips prefix",
			paramName:     "CallMF.Entity",
			wantParamName: "Entity",
		},
		{
			name:          "single-segment name unchanged",
			paramName:     "Entity",
			wantParamName: "Entity",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := genWf.NewCallMicroflowTask()
			task.SetMicroflowQualifiedName("Module.SomeMicroflow")
			task.SetName("callMfTask")
			task.SetCaption("Call MF")
			pm := genWf.NewMicroflowCallParameterMapping()
			pm.SetParameterQualifiedName(tc.paramName)
			pm.SetExpression("$WorkflowContext")
			task.AddParameterMappings(pm)

			lines := formatCallMicroflowGen(task, "  ")
			output := strings.Join(lines, "\n")

			wantFragment := tc.wantParamName + " = "
			if !strings.Contains(output, wantFragment) {
				t.Errorf("expected output to contain %q, got:\n%s", wantFragment, output)
			}

			// Ensure the full qualified prefix is NOT in the parameter position
			if tc.paramName != tc.wantParamName && strings.Contains(output, tc.paramName+" = ") {
				t.Errorf("output should not contain full qualified name %q as parameter, got:\n%s", tc.paramName, output)
			}
		})
	}
}

// --- P2a: DESCRIBE outputs Caption with COMMENT 'caption' format ---

func TestFormatJumpTo_CaptionCommentFormat(t *testing.T) {
	tests := []struct {
		name    string
		caption string
		actName string
		want    string
	}{
		{
			name:    "caption used over name",
			caption: "Go Back to Review",
			actName: "jumpAct1",
			want:    "jump to target1 comment 'Go Back to Review'",
		},
		{
			name:    "name fallback when caption empty",
			caption: "",
			actName: "jumpAct1",
			want:    "jump to target1 comment 'jumpAct1'",
		},
		{
			name:    "caption with single quote escaped",
			caption: "it's done",
			actName: "jumpAct1",
			want:    "jump to target1 comment 'it''s done'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			activity := genWf.NewJumpToActivity()
			activity.SetTargetActivityQualifiedName("target1")
			activity.SetName(tc.actName)
			activity.SetCaption(tc.caption)

			lines := formatSingleActivityGen(activity, "")
			output := strings.Join(lines, "\n")

			if !strings.Contains(output, tc.want) {
				t.Errorf("expected output to contain %q, got:\n%s", tc.want, output)
			}
		})
	}
}

func TestFormatWaitForTimer_CaptionCommentFormat(t *testing.T) {
	tests := []struct {
		name    string
		caption string
		actName string
		delay   string
		want    string
	}{
		{
			name:    "caption with delay",
			caption: "Wait 2 Hours",
			actName: "waitAct1",
			delay:   "${PT2H}",
			want:    "wait for timer '${PT2H}' comment 'Wait 2 Hours'",
		},
		{
			name:    "name fallback no delay",
			caption: "",
			actName: "waitAct1",
			delay:   "",
			want:    "wait for timer comment 'waitAct1'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			activity := genWf.NewWaitForTimerActivity()
			activity.SetDelay(tc.delay)
			activity.SetName(tc.actName)
			activity.SetCaption(tc.caption)

			lines := formatSingleActivityGen(activity, "")
			output := strings.Join(lines, "\n")

			if !strings.Contains(output, tc.want) {
				t.Errorf("expected output to contain %q, got:\n%s", tc.want, output)
			}
		})
	}
}

func TestFormatCallWorkflowActivity_CaptionCommentFormat(t *testing.T) {
	tests := []struct {
		name    string
		caption string
		actName string
		want    string
	}{
		{
			name:    "caption used",
			caption: "Run Sub-Workflow",
			actName: "callWf1",
			want:    "call workflow Module.SubFlow comment 'Run Sub-Workflow'",
		},
		{
			name:    "name fallback",
			caption: "",
			actName: "callWf1",
			want:    "call workflow Module.SubFlow comment 'callWf1'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			activity := genWf.NewCallWorkflowActivity()
			activity.SetWorkflowQualifiedName("Module.SubFlow")
			activity.SetName(tc.actName)
			activity.SetCaption(tc.caption)

			lines := formatCallWorkflowGen(activity, "")
			output := strings.Join(lines, "\n")

			if !strings.Contains(output, tc.want) {
				t.Errorf("expected output to contain %q, got:\n%s", tc.want, output)
			}
		})
	}
}
