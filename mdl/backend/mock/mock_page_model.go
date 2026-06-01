// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

func (m *MockBackend) GetPageModel(id model.ID) (*types.PageModel, error) {
	if m.GetPageModelFunc != nil {
		return m.GetPageModelFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetPageModel not configured")
}

func (m *MockBackend) GetSnippetModel(id model.ID) (*types.PageModel, error) {
	if m.GetSnippetModelFunc != nil {
		return m.GetSnippetModelFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetSnippetModel not configured")
}

func (m *MockBackend) GetLayoutModel(id model.ID) (*types.PageModel, error) {
	if m.GetLayoutModelFunc != nil {
		return m.GetLayoutModelFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetLayoutModel not configured")
}

func (m *MockBackend) WritePageModel(id model.ID, pm *types.PageModel) error {
	if m.WritePageModelFunc != nil {
		return m.WritePageModelFunc(id, pm)
	}
	return fmt.Errorf("MockBackend.WritePageModel not configured")
}

func (m *MockBackend) WriteSnippetModel(id model.ID, pm *types.PageModel) error {
	if m.WriteSnippetModelFunc != nil {
		return m.WriteSnippetModelFunc(id, pm)
	}
	return fmt.Errorf("MockBackend.WriteSnippetModel not configured")
}
