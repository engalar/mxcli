// SPDX-License-Identifier: Apache-2.0
package ast_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// TestAnnotatedInterfaceCoverage 确认所有 MicroflowStatement 实现都有 GetAnnotations。
// 接口方法加上后，编译期就强制每个实现类型提供 GetAnnotations；这个测试在运行期
// 额外确认空值返回不 panic，并作为实现类型清单的活文档。
func TestAnnotatedInterfaceCoverage(t *testing.T) {
	stmts := []ast.MicroflowStatement{
		&ast.DeclareStmt{},
		&ast.MfSetStmt{},
		&ast.ReturnStmt{},
		&ast.RaiseErrorStmt{},
		&ast.CreateObjectStmt{},
		&ast.ChangeObjectStmt{},
		&ast.MfCommitStmt{},
		&ast.DeleteObjectStmt{},
		&ast.RollbackStmt{},
		&ast.RetrieveStmt{},
		&ast.IfStmt{},
		&ast.EnumSplitStmt{},
		&ast.InheritanceSplitStmt{},
		&ast.CastObjectStmt{},
		&ast.LoopStmt{},
		&ast.WhileStmt{},
		&ast.LogStmt{},
		&ast.CallMicroflowStmt{},
		&ast.CallNanoflowStmt{},
		&ast.CallJavaActionStmt{},
		&ast.CallJavaScriptActionStmt{},
		&ast.CallWebServiceStmt{},
		&ast.ExecuteDatabaseQueryStmt{},
		&ast.CallExternalActionStmt{},
		&ast.BreakStmt{},
		&ast.ContinueStmt{},
		&ast.ListOperationStmt{},
		&ast.AggregateListStmt{},
		&ast.CreateListStmt{},
		&ast.AddToListStmt{},
		&ast.RemoveFromListStmt{},
		&ast.ShowPageStmt{},
		&ast.ClosePageStmt{},
		&ast.ShowHomePageStmt{},
		&ast.SynchronizeStmt{},
		&ast.ShowMessageStmt{},
		&ast.DownloadFileStmt{},
		&ast.ValidationFeedbackStmt{},
		&ast.RestCallStmt{},
		&ast.SendRestRequestStmt{},
		&ast.ImportFromMappingStmt{},
		&ast.ExportToMappingStmt{},
		&ast.TransformJsonStmt{},
		// workflow statements
		&ast.CallWorkflowStmt{},
		&ast.GetWorkflowDataStmt{},
		&ast.GetWorkflowsStmt{},
		&ast.GetWorkflowActivityRecordsStmt{},
		&ast.WorkflowOperationStmt{},
		&ast.SetTaskOutcomeStmt{},
		&ast.OpenUserTaskStmt{},
		&ast.NotifyWorkflowStmt{},
		&ast.OpenWorkflowStmt{},
		&ast.LockWorkflowStmt{},
		&ast.UnlockWorkflowStmt{},
		&ast.GenerateJumpToStmt{},
		&ast.ApplyJumpToStmt{},
	}
	for _, s := range stmts {
		// 编译期已保证实现了接口；运行期确认 nil 返回不 panic
		if got := s.GetAnnotations(); got != nil {
			t.Errorf("%T zero value: GetAnnotations() = %v, want nil", s, got)
		}
	}
	t.Logf("all %d MicroflowStatement types implement GetAnnotations()", len(stmts))
}
