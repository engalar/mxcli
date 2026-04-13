// SPDX-License-Identifier: Apache-2.0

package syntax

func init() {
	Register(SyntaxFeature{
		Path:    "workflow",
		Summary: "Multi-step business processes with user tasks, decisions, and parallel paths",
		Keywords: []string{
			"workflow", "business process", "approval", "review",
			"user task", "decision", "parallel",
		},
		Syntax: "CREATE WORKFLOW Module.Name\n  PARAMETER $Context: Module.Entity\nBEGIN\n  <activities>\nEND WORKFLOW;",
		Example: "CREATE WORKFLOW HR.LeaveApproval\n  PARAMETER $Context: HR.LeaveRequest\nBEGIN\n  USER TASK Review 'Review request'\n    PAGE HR.ReviewPage\n    OUTCOMES 'Approve' { } 'Reject' { };\nEND WORKFLOW;",
		SeeAlso: []string{"workflow.user-task", "workflow.decision", "workflow.parallel-split"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.show",
		Summary: "List and describe existing workflows",
		Keywords: []string{
			"list workflows", "show workflows", "describe workflow",
		},
		Syntax: "SHOW WORKFLOWS;\nSHOW WORKFLOWS IN <module>;\nDESCRIBE WORKFLOW Module.Name;",
		Example: "SHOW WORKFLOWS IN HR;\nDESCRIBE WORKFLOW HR.LeaveApproval;",
	})

	Register(SyntaxFeature{
		Path:    "workflow.create",
		Summary: "Create a new workflow definition with activities and flow",
		Keywords: []string{
			"create workflow", "new workflow", "define workflow",
			"parameter", "overview page", "due date",
		},
		Syntax: "CREATE [OR MODIFY] WORKFLOW Module.Name\n  PARAMETER $Context: Module.Entity\n  [OVERVIEW PAGE Module.OverviewPage]\n  [DUE DATE '<expression>']\nBEGIN\n  <activities>\nEND WORKFLOW;",
		Example: "CREATE WORKFLOW Module.ApprovalFlow\n  PARAMETER $Context: Module.Request\n  OVERVIEW PAGE Module.WF_Overview\nBEGIN\n  USER TASK ReviewTask 'Review the request'\n    PAGE Module.ReviewPage\n    OUTCOMES 'Approve' { } 'Reject' { };\nEND WORKFLOW;",
		SeeAlso: []string{"workflow.user-task", "workflow.decision", "workflow.drop"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.user-task",
		Summary: "User task activity — assigns work to users with outcomes",
		Keywords: []string{
			"user task", "human task", "assign", "assignee",
			"outcomes", "approve", "reject", "page",
		},
		Syntax: "USER TASK <name> '<caption>'\n  [PAGE Module.Page]\n  [TARGETING MICROFLOW Module.MF | TARGETING XPATH '<xpath>']\n  [ENTITY Module.Entity]\n  OUTCOMES '<outcome1>' { <activities> } '<outcome2>' { <activities> };",
		Example: "USER TASK ReviewTask 'Review the request'\n  PAGE HR.ReviewPage\n  TARGETING XPATH '[Module.Employee/Active = true()]'\n  OUTCOMES 'Approve' { } 'Reject' { };",
		SeeAlso: []string{"workflow.user-task.targeting", "workflow.create"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.user-task.targeting",
		Summary: "Control who can pick up a user task — microflow or XPath based",
		Keywords: []string{
			"targeting", "user targeting", "who can execute",
			"assignee", "candidate", "xpath", "microflow",
			"task assignment", "user filter",
		},
		Syntax: "TARGETING MICROFLOW Module.MF\nTARGETING XPATH '<xpath-expression>'",
		Example: "-- XPath targeting: only active managers\nUSER TASK Approve 'Approve request'\n  TARGETING XPATH '[HR.Employee/Role = \"Manager\" and Active = true()]'\n  OUTCOMES 'Done' { };\n\n-- Microflow targeting: custom logic\nUSER TASK Approve 'Approve request'\n  TARGETING MICROFLOW HR.GetApprovers\n  OUTCOMES 'Done' { };",
		MinVersion: "9.0.0",
		SeeAlso:    []string{"workflow.user-task"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.decision",
		Summary: "Decision activity — conditional branching based on expression outcomes",
		Keywords: []string{
			"decision", "conditional", "branch", "if", "condition",
			"exclusive gateway", "XOR",
		},
		Syntax: "DECISION ['<caption>'] [COMMENT '<text>']\n  OUTCOMES '<outcome>' { <activities> } ...;",
		Example: "DECISION 'Check amount'\n  OUTCOMES 'Under 1000' { } 'Over 1000' {\n    USER TASK ManagerApproval 'Manager must approve'\n      OUTCOMES 'OK' { };\n  };",
		SeeAlso: []string{"workflow.create", "workflow.parallel-split"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.parallel-split",
		Summary: "Parallel split — execute multiple paths concurrently",
		Keywords: []string{
			"parallel", "concurrent", "split", "fork", "join",
			"parallel gateway", "AND",
		},
		Syntax: "PARALLEL SPLIT [COMMENT '<text>']\n  PATH 1 { <activities> }\n  PATH 2 { <activities> };",
		Example: "PARALLEL SPLIT\n  PATH 1 {\n    USER TASK LegalReview 'Legal review'\n      OUTCOMES 'Done' { };\n  }\n  PATH 2 {\n    USER TASK TechReview 'Technical review'\n      OUTCOMES 'Done' { };\n  };",
		SeeAlso: []string{"workflow.decision", "workflow.create"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.call-microflow",
		Summary: "Call a microflow as a workflow activity",
		Keywords: []string{
			"call microflow", "microflow task", "automated step",
			"system task",
		},
		Syntax: "CALL MICROFLOW Module.MF [COMMENT '<text>']\n  [OUTCOMES '<outcome>' { <activities> } ...];",
		Example: "CALL MICROFLOW HR.SendNotification\n  COMMENT 'Notify manager';",
		SeeAlso: []string{"workflow.create", "workflow.call-workflow"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.call-workflow",
		Summary: "Call a sub-workflow from within a workflow",
		Keywords: []string{
			"call workflow", "sub-workflow", "nested workflow",
		},
		Syntax: "CALL WORKFLOW Module.WF [COMMENT '<text>'];",
		Example: "CALL WORKFLOW HR.SubApproval COMMENT 'Delegate to sub-process';",
		SeeAlso: []string{"workflow.create", "workflow.call-microflow"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.drop",
		Summary: "Delete a workflow definition",
		Keywords: []string{
			"drop workflow", "delete workflow", "remove workflow",
		},
		Syntax: "DROP WORKFLOW Module.Name;",
		Example: "DROP WORKFLOW HR.LeaveApproval;",
		SeeAlso: []string{"workflow.create"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.catalog",
		Summary: "Query workflow metadata via catalog tables",
		Keywords: []string{
			"catalog", "query workflows", "workflow metadata",
			"cross-reference", "callers", "callees",
		},
		Syntax: "REFRESH CATALOG FULL;\nSELECT * FROM CATALOG.WORKFLOWS;\nSHOW CALLERS OF Module.WorkflowName;\nSHOW REFERENCES TO Module.WorkflowName;",
		Example: "REFRESH CATALOG FULL;\nSELECT QualifiedName, ActivityCount, UserTaskCount\n  FROM CATALOG.WORKFLOWS WHERE UserTaskCount > 0;",
		SeeAlso: []string{"workflow.show"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.boundary-event",
		Summary: "Attach boundary events (timer) to user tasks for timeouts",
		Keywords: []string{
			"boundary event", "timer", "timeout", "deadline",
			"SLA", "escalation",
		},
		Syntax: "BOUNDARY TIMER ON <task-name> AFTER '<duration>' {\n  <activities>\n}",
		Example: "BOUNDARY TIMER ON ReviewTask AFTER 'P3D' {\n  CALL MICROFLOW Module.Escalate;\n}",
		MinVersion: "10.6.0",
		SeeAlso:    []string{"workflow.user-task"},
	})

	Register(SyntaxFeature{
		Path:    "workflow.alter",
		Summary: "Modify an existing workflow — change properties, add/remove activities",
		Keywords: []string{
			"alter workflow", "modify workflow", "update workflow",
			"add activity", "drop activity", "replace activity",
		},
		Syntax: "ALTER WORKFLOW Module.Name SET <property> = <value>;\nALTER WORKFLOW Module.Name INSERT <activity> [BEFORE|AFTER <name>];\nALTER WORKFLOW Module.Name DROP <activity-name>;\nALTER WORKFLOW Module.Name REPLACE <name> WITH <activity>;",
		Example: "ALTER WORKFLOW HR.LeaveApproval SET DUE DATE = 'addDays([%CurrentDateTime%], 7)';\nALTER WORKFLOW HR.LeaveApproval INSERT\n  CALL MICROFLOW HR.NotifyHR\n  AFTER ReviewTask;",
		SeeAlso: []string{"workflow.create", "workflow.drop"},
	})
}
