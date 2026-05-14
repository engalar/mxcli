// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.3.E1 — MockWorkflowMutator surface is fully gen-typed; all
// stubs return descriptive errors for unset Funcs per CLAUDE.md
// MockBackend audit rule.

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

var _ backend.WorkflowMutator = (*MockWorkflowMutator)(nil)

// MockWorkflowMutator implements backend.WorkflowMutator. Every interface
// method is backed by a public function field.
type MockWorkflowMutator struct {
	SetPropertyFunc           func(prop string, value string) error
	SetPropertyWithEntityFunc func(prop string, value string, entity string) error
	SetActivityPropertyFunc   func(activityRef string, atPos int, prop string, value string) error
	DropActivityFunc          func(activityRef string, atPos int) error
	DropOutcomeFunc           func(activityRef string, atPos int, outcomeName string) error
	DropPathFunc              func(activityRef string, atPos int, pathCaption string) error
	DropBranchFunc            func(activityRef string, atPos int, branchName string) error
	DropBoundaryEventFunc     func(activityRef string, atPos int) error
	SaveFunc                  func() error

	// Gen-typed activity insertion / replacement.
	InsertAfterActivityGenFunc func(activityRef string, atPos int, activities []element.Element) error
	ReplaceActivityGenFunc     func(activityRef string, atPos int, activities []element.Element) error
	InsertOutcomeGenFunc       func(activityRef string, atPos int, outcomeName string, activities []element.Element) error
	InsertPathGenFunc          func(activityRef string, atPos int, pathCaption string, activities []element.Element) error
	InsertBranchGenFunc        func(activityRef string, atPos int, condition string, activities []element.Element) error
	InsertBoundaryEventGenFunc func(activityRef string, atPos int, eventType string, delay string, activities []element.Element) error
}

func (m *MockWorkflowMutator) SetProperty(prop string, value string) error {
	if m.SetPropertyFunc != nil {
		return m.SetPropertyFunc(prop, value)
	}
	return fmt.Errorf("MockWorkflowMutator.SetProperty not configured")
}

func (m *MockWorkflowMutator) SetPropertyWithEntity(prop string, value string, entity string) error {
	if m.SetPropertyWithEntityFunc != nil {
		return m.SetPropertyWithEntityFunc(prop, value, entity)
	}
	return fmt.Errorf("MockWorkflowMutator.SetPropertyWithEntity not configured")
}

func (m *MockWorkflowMutator) SetActivityProperty(activityRef string, atPos int, prop string, value string) error {
	if m.SetActivityPropertyFunc != nil {
		return m.SetActivityPropertyFunc(activityRef, atPos, prop, value)
	}
	return fmt.Errorf("MockWorkflowMutator.SetActivityProperty not configured")
}

func (m *MockWorkflowMutator) DropActivity(activityRef string, atPos int) error {
	if m.DropActivityFunc != nil {
		return m.DropActivityFunc(activityRef, atPos)
	}
	return fmt.Errorf("MockWorkflowMutator.DropActivity not configured")
}

func (m *MockWorkflowMutator) DropOutcome(activityRef string, atPos int, outcomeName string) error {
	if m.DropOutcomeFunc != nil {
		return m.DropOutcomeFunc(activityRef, atPos, outcomeName)
	}
	return fmt.Errorf("MockWorkflowMutator.DropOutcome not configured")
}

func (m *MockWorkflowMutator) DropPath(activityRef string, atPos int, pathCaption string) error {
	if m.DropPathFunc != nil {
		return m.DropPathFunc(activityRef, atPos, pathCaption)
	}
	return fmt.Errorf("MockWorkflowMutator.DropPath not configured")
}

func (m *MockWorkflowMutator) DropBranch(activityRef string, atPos int, branchName string) error {
	if m.DropBranchFunc != nil {
		return m.DropBranchFunc(activityRef, atPos, branchName)
	}
	return fmt.Errorf("MockWorkflowMutator.DropBranch not configured")
}

func (m *MockWorkflowMutator) DropBoundaryEvent(activityRef string, atPos int) error {
	if m.DropBoundaryEventFunc != nil {
		return m.DropBoundaryEventFunc(activityRef, atPos)
	}
	return fmt.Errorf("MockWorkflowMutator.DropBoundaryEvent not configured")
}

func (m *MockWorkflowMutator) Save() error {
	if m.SaveFunc != nil {
		return m.SaveFunc()
	}
	return fmt.Errorf("MockWorkflowMutator.Save not configured")
}

// ── Gen-typed activity insertion / replacement ────────────────────────

func (m *MockWorkflowMutator) InsertAfterActivityGen(activityRef string, atPos int, activities []element.Element) error {
	if m.InsertAfterActivityGenFunc != nil {
		return m.InsertAfterActivityGenFunc(activityRef, atPos, activities)
	}
	return fmt.Errorf("MockWorkflowMutator.InsertAfterActivityGen not configured")
}

func (m *MockWorkflowMutator) ReplaceActivityGen(activityRef string, atPos int, activities []element.Element) error {
	if m.ReplaceActivityGenFunc != nil {
		return m.ReplaceActivityGenFunc(activityRef, atPos, activities)
	}
	return fmt.Errorf("MockWorkflowMutator.ReplaceActivityGen not configured")
}

func (m *MockWorkflowMutator) InsertOutcomeGen(activityRef string, atPos int, outcomeName string, activities []element.Element) error {
	if m.InsertOutcomeGenFunc != nil {
		return m.InsertOutcomeGenFunc(activityRef, atPos, outcomeName, activities)
	}
	return fmt.Errorf("MockWorkflowMutator.InsertOutcomeGen not configured")
}

func (m *MockWorkflowMutator) InsertPathGen(activityRef string, atPos int, pathCaption string, activities []element.Element) error {
	if m.InsertPathGenFunc != nil {
		return m.InsertPathGenFunc(activityRef, atPos, pathCaption, activities)
	}
	return fmt.Errorf("MockWorkflowMutator.InsertPathGen not configured")
}

func (m *MockWorkflowMutator) InsertBranchGen(activityRef string, atPos int, condition string, activities []element.Element) error {
	if m.InsertBranchGenFunc != nil {
		return m.InsertBranchGenFunc(activityRef, atPos, condition, activities)
	}
	return fmt.Errorf("MockWorkflowMutator.InsertBranchGen not configured")
}

func (m *MockWorkflowMutator) InsertBoundaryEventGen(activityRef string, atPos int, eventType string, delay string, activities []element.Element) error {
	if m.InsertBoundaryEventGenFunc != nil {
		return m.InsertBoundaryEventGenFunc(activityRef, atPos, eventType, delay, activities)
	}
	return fmt.Errorf("MockWorkflowMutator.InsertBoundaryEventGen not configured")
}
