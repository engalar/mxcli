// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

func (m *MockBackend) ListPages() ([]*pages.Page, error) {
	if m.ListPagesFunc != nil {
		return m.ListPagesFunc()
	}
	return nil, nil
}

func (m *MockBackend) GetPage(id model.ID) (*pages.Page, error) {
	if m.GetPageFunc != nil {
		return m.GetPageFunc(id)
	}
	return nil, nil
}

func (m *MockBackend) CreatePage(page *pages.Page) error {
	if m.CreatePageFunc != nil {
		return m.CreatePageFunc(page)
	}
	return nil
}

func (m *MockBackend) UpdatePage(page *pages.Page) error {
	if m.UpdatePageFunc != nil {
		return m.UpdatePageFunc(page)
	}
	return nil
}

func (m *MockBackend) DeletePage(id model.ID) error {
	if m.DeletePageFunc != nil {
		return m.DeletePageFunc(id)
	}
	return nil
}

func (m *MockBackend) MovePage(page *pages.Page) error {
	if m.MovePageFunc != nil {
		return m.MovePageFunc(page)
	}
	return nil
}

func (m *MockBackend) ListLayouts() ([]*pages.Layout, error) {
	if m.ListLayoutsFunc != nil {
		return m.ListLayoutsFunc()
	}
	return nil, nil
}

func (m *MockBackend) GetLayout(id model.ID) (*pages.Layout, error) {
	if m.GetLayoutFunc != nil {
		return m.GetLayoutFunc(id)
	}
	return nil, nil
}

func (m *MockBackend) CreateLayout(layout *pages.Layout) error {
	if m.CreateLayoutFunc != nil {
		return m.CreateLayoutFunc(layout)
	}
	return nil
}

func (m *MockBackend) UpdateLayout(layout *pages.Layout) error {
	if m.UpdateLayoutFunc != nil {
		return m.UpdateLayoutFunc(layout)
	}
	return nil
}

func (m *MockBackend) DeleteLayout(id model.ID) error {
	if m.DeleteLayoutFunc != nil {
		return m.DeleteLayoutFunc(id)
	}
	return nil
}

func (m *MockBackend) ListSnippets() ([]*pages.Snippet, error) {
	if m.ListSnippetsFunc != nil {
		return m.ListSnippetsFunc()
	}
	return nil, nil
}

func (m *MockBackend) CreateSnippet(snippet *pages.Snippet) error {
	if m.CreateSnippetFunc != nil {
		return m.CreateSnippetFunc(snippet)
	}
	return nil
}

func (m *MockBackend) UpdateSnippet(snippet *pages.Snippet) error {
	if m.UpdateSnippetFunc != nil {
		return m.UpdateSnippetFunc(snippet)
	}
	return nil
}

func (m *MockBackend) DeleteSnippet(id model.ID) error {
	if m.DeleteSnippetFunc != nil {
		return m.DeleteSnippetFunc(id)
	}
	return nil
}

func (m *MockBackend) MoveSnippet(snippet *pages.Snippet) error {
	if m.MoveSnippetFunc != nil {
		return m.MoveSnippetFunc(snippet)
	}
	return nil
}

func (m *MockBackend) ListBuildingBlocks() ([]*pages.BuildingBlock, error) {
	if m.ListBuildingBlocksFunc != nil {
		return m.ListBuildingBlocksFunc()
	}
	return nil, nil
}

func (m *MockBackend) ListPageTemplates() ([]*pages.PageTemplate, error) {
	if m.ListPageTemplatesFunc != nil {
		return m.ListPageTemplatesFunc()
	}
	return nil, nil
}

// Stage 3.3.5.C1 gen-typed Page/Layout/Snippet stubs.
//
// Per the MockBackend audit rule (see CLAUDE.md "Backend abstraction
// compliance"), every gen-typed stub returns a descriptive error when
// the matching Func field is nil. Tests that exercise gen-typed page
// flows MUST wire a Func or assert on the error string — silent
// `nil, nil` returns mask test gaps.

func (m *MockBackend) ListPagesGen() ([]*genPg.Page, error) {
	if m.ListPagesGenFunc != nil {
		return m.ListPagesGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListPagesGen not configured")
}

func (m *MockBackend) GetPageGen(id model.ID) (*genPg.Page, error) {
	if m.GetPageGenFunc != nil {
		return m.GetPageGenFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetPageGen not configured")
}

func (m *MockBackend) CreatePageGen(parentUUID, containmentName string, page *genPg.Page) error {
	if m.CreatePageGenFunc != nil {
		return m.CreatePageGenFunc(parentUUID, containmentName, page)
	}
	return fmt.Errorf("MockBackend.CreatePageGen not configured")
}

func (m *MockBackend) UpdatePageGen(page *genPg.Page) error {
	if m.UpdatePageGenFunc != nil {
		return m.UpdatePageGenFunc(page)
	}
	return fmt.Errorf("MockBackend.UpdatePageGen not configured")
}

func (m *MockBackend) ListLayoutsGen() ([]*genPg.Layout, error) {
	if m.ListLayoutsGenFunc != nil {
		return m.ListLayoutsGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListLayoutsGen not configured")
}

func (m *MockBackend) GetLayoutGen(id model.ID) (*genPg.Layout, error) {
	if m.GetLayoutGenFunc != nil {
		return m.GetLayoutGenFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetLayoutGen not configured")
}

func (m *MockBackend) CreateLayoutGen(parentUUID, containmentName string, layout *genPg.Layout) error {
	if m.CreateLayoutGenFunc != nil {
		return m.CreateLayoutGenFunc(parentUUID, containmentName, layout)
	}
	return fmt.Errorf("MockBackend.CreateLayoutGen not configured")
}

func (m *MockBackend) UpdateLayoutGen(layout *genPg.Layout) error {
	if m.UpdateLayoutGenFunc != nil {
		return m.UpdateLayoutGenFunc(layout)
	}
	return fmt.Errorf("MockBackend.UpdateLayoutGen not configured")
}

func (m *MockBackend) ListSnippetsGen() ([]*genPg.Snippet, error) {
	if m.ListSnippetsGenFunc != nil {
		return m.ListSnippetsGenFunc()
	}
	return nil, fmt.Errorf("MockBackend.ListSnippetsGen not configured")
}

func (m *MockBackend) GetSnippetGen(id model.ID) (*genPg.Snippet, error) {
	if m.GetSnippetGenFunc != nil {
		return m.GetSnippetGenFunc(id)
	}
	return nil, fmt.Errorf("MockBackend.GetSnippetGen not configured")
}

func (m *MockBackend) CreateSnippetGen(parentUUID, containmentName string, snippet *genPg.Snippet) error {
	if m.CreateSnippetGenFunc != nil {
		return m.CreateSnippetGenFunc(parentUUID, containmentName, snippet)
	}
	return fmt.Errorf("MockBackend.CreateSnippetGen not configured")
}

func (m *MockBackend) UpdateSnippetGen(snippet *genPg.Snippet) error {
	if m.UpdateSnippetGenFunc != nil {
		return m.UpdateSnippetGenFunc(snippet)
	}
	return fmt.Errorf("MockBackend.UpdateSnippetGen not configured")
}

func (m *MockBackend) GetPageContainerUUID(id model.ID) (model.ID, error) {
	if m.GetPageContainerUUIDFunc != nil {
		return m.GetPageContainerUUIDFunc(id)
	}
	return "", fmt.Errorf("MockBackend.GetPageContainerUUID not configured")
}

// Stage 3.3.5.D5.c gen-typed delete + move stubs.

func (m *MockBackend) DeletePageGen(id model.ID) error {
	if m.DeletePageGenFunc != nil {
		return m.DeletePageGenFunc(id)
	}
	return fmt.Errorf("MockBackend.DeletePageGen not configured")
}

func (m *MockBackend) MovePageGen(id, containerID model.ID) error {
	if m.MovePageGenFunc != nil {
		return m.MovePageGenFunc(id, containerID)
	}
	return fmt.Errorf("MockBackend.MovePageGen not configured")
}

func (m *MockBackend) DeleteLayoutGen(id model.ID) error {
	if m.DeleteLayoutGenFunc != nil {
		return m.DeleteLayoutGenFunc(id)
	}
	return fmt.Errorf("MockBackend.DeleteLayoutGen not configured")
}

func (m *MockBackend) MoveLayoutGen(id, containerID model.ID) error {
	if m.MoveLayoutGenFunc != nil {
		return m.MoveLayoutGenFunc(id, containerID)
	}
	return fmt.Errorf("MockBackend.MoveLayoutGen not configured")
}

func (m *MockBackend) DeleteSnippetGen(id model.ID) error {
	if m.DeleteSnippetGenFunc != nil {
		return m.DeleteSnippetGenFunc(id)
	}
	return fmt.Errorf("MockBackend.DeleteSnippetGen not configured")
}

func (m *MockBackend) MoveSnippetGen(id, containerID model.ID) error {
	if m.MoveSnippetGenFunc != nil {
		return m.MoveSnippetGenFunc(id, containerID)
	}
	return fmt.Errorf("MockBackend.MoveSnippetGen not configured")
}
