// SPDX-License-Identifier: Apache-2.0

//go:build linux && integration

package goldenfs

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// step scripts are package-level consts so each declaration stays readable.

const wfStep1aInfra = `
create module role MyFirstModule.Reviewer
  description 'Integration test reviewer role';

create or modify microflow MyFirstModule.ACT_Notify ()
  returns Nothing
  {
    return;
  }

create or modify microflow MyFirstModule.ACT_BoundaryHandler ()
  returns Nothing
  {
    return;
  }
`

const wfStep1bPages = `
create page MyFirstModule.Page_WF_InitialReview
  (title: 'Initial Review', layout: Atlas_Core.Atlas_TopBar,
   params: { $WorkflowUserTask: System.WorkflowUserTask }) { }

create page MyFirstModule.Page_WF_StandardApproval
  (title: 'Standard Approval', layout: Atlas_Core.Atlas_TopBar,
   params: { $WorkflowUserTask: System.WorkflowUserTask }) { }

create page MyFirstModule.Page_WF_SeniorApproval
  (title: 'Senior Approval', layout: Atlas_Core.Atlas_TopBar,
   params: { $WorkflowUserTask: System.WorkflowUserTask }) { }

create page MyFirstModule.Page_WF_FinalSignOff
  (title: 'Final Sign-off', layout: Atlas_Core.Atlas_TopBar,
   params: { $WorkflowUserTask: System.WorkflowUserTask }) { }
`

const wfStep2aEnum = `
create or modify enumeration MyFirstModule.WF_Status (
  Draft    'Draft',
  InReview 'In Review',
  Approved 'Approved',
  Rejected 'Rejected'
);
`

const wfStep2bEntity = `
create or modify entity MyFirstModule.WF_Item (
  Title       : string(200),
  Description : string(unlimited),
  Count       : integer,
  Total       : long,
  Amount      : decimal,
  IsActive    : boolean default true,
  DueDate     : datetime,
  Payload     : binary,
  SeqNo       : autonumber,
  Status      : enumeration(MyFirstModule.WF_Status) default MyFirstModule.WF_Status.Draft
);
`

const wfStep3Grant = `
grant MyFirstModule.Reviewer on MyFirstModule.WF_Item
  (create, delete, read *, write (Title, Description, Count, Total, Amount, IsActive, DueDate, Payload, Status));
`

const wfStep4Workflow = `
create or modify workflow MyFirstModule.WF_ComplexApproval
  parameter $ctx: MyFirstModule.WF_Item
  display 'Complex Approval Flow'
  description 'Comprehensive workflow integration test: user task, multi-user task, decision, parallel split, call microflow, wait for notification, jump to'
{
  user task InitialReview 'Initial Review'
    page MyFirstModule.Page_WF_InitialReview
    targeting users xpath 'System.User[Name != ""]'
    outcomes
      'Submit' {
        decision '$ctx/Amount > 1000'
          outcomes
            true -> {
              multi user task SeniorApproval 'Senior Approval'
                page MyFirstModule.Page_WF_SeniorApproval
                targeting groups xpath 'System.UserRole[Name != ""]'
                outcomes
                  'Approve' { }
                  'Reject'  { };
            }
            false -> {
              user task StandardApproval 'Standard Approval'
                page MyFirstModule.Page_WF_StandardApproval
                targeting users xpath 'System.User[Name != ""]'
                outcomes
                  'Approve' { }
                  'Reject'  { };
            }
          ;
        wait for notification comment 'AwaitExternalSignal';
        parallel split
          path 1 { call microflow MyFirstModule.ACT_Notify; }
          path 2 { }
        ;
        user task FinalSignOff 'Final Sign-off'
          page MyFirstModule.Page_WF_FinalSignOff
          outcomes
            'Sign'              { }
            'ReturnForRevision' { jump to InitialReview; }
          ;
      }
      'Cancel' { };
}
`

const wfStep5Alter = `
alter workflow MyFirstModule.WF_ComplexApproval
  insert boundary event on InitialReview@1
    non interrupting timer 'addHours([%CurrentDateTime%], 24)' {
      call microflow MyFirstModule.ACT_BoundaryHandler;
    };
`

// TestGoldenFS_WorkflowIntegration verifies that a comprehensive Mendix
// Workflow (user task, multi-user task, decision, parallel split, call
// microflow, wait-for-notification, jump-to, boundary event) can be written
// through the golden FUSE overlay without corrupting the project or leaking
// to the base directory.
func TestGoldenFS_WorkflowIntegration(t *testing.T) {
	mxBin := findMxBinaryForTest()
	if mxBin == "" {
		t.Skip("mx binary not available (set MX_BINARY or install mxbuild)")
	}

	// Use helpdesk-clean MPR: a blank project with Atlas Core that passes
	// mx check with 0 errors, so CE1613 from pre-existing project issues
	// does not pollute the result.
	realDir := helpdeskBlankDir(t)
	realMpr := filepath.Join(realDir, "minimal.mpr")
	origStat, err := os.Stat(realMpr)
	if err != nil {
		t.Fatalf("stat real mpr: %v", err)
	}

	snap, err := Open(realDir)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	mprPath := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Each step runs in its own executor + backend so backend list-caches start
	// fresh — this matches real CLI usage and avoids known stale-cache issues
	// when an entity references an enumeration created earlier in the same
	// process. Connect/disconnect cost is negligible for SQLite over FUSE.
	connectStmt := fmt.Sprintf("connect local '%s';", mprPath)

	run := func(label, script string) {
		t.Helper()
		e := executor.New(io.Discard)
		e.SetQuiet(true)
		e.SetBackendFactory(func() backend.ConnectionBackend { return mprbackend.New() })
		defer func() {
			if err := e.Close(); err != nil {
				t.Logf("[%s] executor close warning: %v", label, err)
			}
		}()

		full := connectStmt + "\n" + script
		prog, errs := visitor.Build(full)
		if len(errs) > 0 {
			t.Fatalf("[%s] MDL parse error: %v", label, errs)
		}
		if err := e.ExecuteProgram(prog); err != nil {
			t.Fatalf("[%s] executor error: %v", label, err)
		}
	}

	run("step1a-infra", wfStep1aInfra)
	run("step1b-pages", wfStep1bPages)
	run("step2a-enum", wfStep2aEnum)
	run("step2b-entity", wfStep2bEntity)
	run("step3-grant", wfStep3Grant)
	run("step4-workflow", wfStep4Workflow)
	run("step5-alter", wfStep5Alter)

	cmd := exec.Command(mxBin, "check", mprPath)
	output, err := cmd.CombinedOutput()
	t.Logf("mx check output (%d bytes, err=%v):\n%s", len(output), err, output)

	assertNoFUSECorruption(t, string(output),
		"WF_ComplexApproval",
		"WF_Item",
		"WF_Status",
		"Page_WF_",
		"ACT_Notify",
		"ACT_BoundaryHandler",
	)
	checkFUSEIsolation(t, realMpr, origStat)

	snap.Rollback()
}
