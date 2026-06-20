// SPDX-License-Identifier: Apache-2.0

package ast

// DefineFragmentStmt represents: DEFINE FRAGMENT Name AS { widgets }
type DefineFragmentStmt struct {
	Name    string
	Widgets []*WidgetV3
}

func (s *DefineFragmentStmt) isStatement() {}
func (s *DefineFragmentStmt) TypeName() string { return "DefineFragment" }

// DescribeFragmentFromStmt represents DESCRIBE FRAGMENT FROM PAGE/SNIPPET ... WIDGET ...
type DescribeFragmentFromStmt struct {
	ContainerType string        // "PAGE" or "SNIPPET"
	ContainerName QualifiedName // Module.PageName or Module.SnippetName
	WidgetName    string        // Target widget name
}

func (s *DescribeFragmentFromStmt) isStatement() {}
func (s *DescribeFragmentFromStmt) TypeName() string { return "DescribeFragmentFrom" }
