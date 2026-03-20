// SPDX-License-Identifier: Apache-2.0

package ast

// CreateWorkflowStmt represents: CREATE WORKFLOW Module.Name ...
type CreateWorkflowStmt struct {
	Name           QualifiedName
	CreateOrModify bool
	Documentation  string

	// Context parameter entity
	ParameterVar    string        // e.g. "$WorkflowContext"
	ParameterEntity QualifiedName // e.g. Module.Entity

	// Optional metadata
	OverviewPage QualifiedName // qualified name of overview page
	DueDate      string        // due date expression

	// Activities
	Activities []WorkflowActivityNode
}

func (s *CreateWorkflowStmt) isStatement() {}

// DropWorkflowStmt represents: DROP WORKFLOW Module.Name
type DropWorkflowStmt struct {
	Name QualifiedName
}

func (s *DropWorkflowStmt) isStatement() {}

// WorkflowActivityNode is the interface for workflow activity AST nodes.
type WorkflowActivityNode interface {
	workflowActivityNode()
}

// WorkflowUserTaskNode represents a USER TASK activity.
type WorkflowUserTaskNode struct {
	Name       string // identifier name
	Caption    string // display caption
	Page       QualifiedName
	Targeting  WorkflowTargetingNode
	Entity     QualifiedName // user task entity
	Outcomes   []WorkflowUserTaskOutcomeNode
}

func (n *WorkflowUserTaskNode) workflowActivityNode() {}

// WorkflowTargetingNode represents user targeting strategy.
type WorkflowTargetingNode struct {
	Kind       string        // "microflow", "xpath", or ""
	Microflow  QualifiedName // for microflow targeting
	XPath      string        // for xpath targeting
}

// WorkflowUserTaskOutcomeNode represents an outcome of a user task.
type WorkflowUserTaskOutcomeNode struct {
	Caption    string
	Activities []WorkflowActivityNode
}

// WorkflowCallMicroflowNode represents a CALL MICROFLOW activity.
type WorkflowCallMicroflowNode struct {
	Microflow QualifiedName
	Caption   string
	Outcomes  []WorkflowConditionOutcomeNode
}

func (n *WorkflowCallMicroflowNode) workflowActivityNode() {}

// WorkflowCallWorkflowNode represents a CALL WORKFLOW activity.
type WorkflowCallWorkflowNode struct {
	Workflow QualifiedName
	Caption  string
}

func (n *WorkflowCallWorkflowNode) workflowActivityNode() {}

// WorkflowDecisionNode represents a DECISION activity.
type WorkflowDecisionNode struct {
	Expression string // decision expression
	Caption    string
	Outcomes   []WorkflowConditionOutcomeNode
}

func (n *WorkflowDecisionNode) workflowActivityNode() {}

// WorkflowConditionOutcomeNode represents an outcome of a decision or call microflow.
type WorkflowConditionOutcomeNode struct {
	Value      string // "True", "False", "Default", or enumeration value
	Activities []WorkflowActivityNode
}

// WorkflowParallelSplitNode represents a PARALLEL SPLIT activity.
type WorkflowParallelSplitNode struct {
	Caption string
	Paths   []WorkflowParallelPathNode
}

func (n *WorkflowParallelSplitNode) workflowActivityNode() {}

// WorkflowParallelPathNode represents a path in a parallel split.
type WorkflowParallelPathNode struct {
	PathNumber int
	Activities []WorkflowActivityNode
}

// WorkflowJumpToNode represents a JUMP TO activity.
type WorkflowJumpToNode struct {
	Target  string // name of target activity
	Caption string
}

func (n *WorkflowJumpToNode) workflowActivityNode() {}

// WorkflowWaitForTimerNode represents a WAIT FOR TIMER activity.
type WorkflowWaitForTimerNode struct {
	DelayExpression string
	Caption         string
}

func (n *WorkflowWaitForTimerNode) workflowActivityNode() {}

// WorkflowWaitForNotificationNode represents a WAIT FOR NOTIFICATION activity.
type WorkflowWaitForNotificationNode struct {
	Caption string
}

func (n *WorkflowWaitForNotificationNode) workflowActivityNode() {}

// WorkflowEndNode represents an END activity.
type WorkflowEndNode struct {
	Caption string
}

func (n *WorkflowEndNode) workflowActivityNode() {}
