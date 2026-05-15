// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

// PageBackend provides page, layout, and snippet operations.
type PageBackend interface {
	// Pages — legacy sdk-typed surface, retired in Stage 3.3.5.E1.
	ListPages() ([]*pages.Page, error)
	GetPage(id model.ID) (*pages.Page, error)
	CreatePage(page *pages.Page) error
	UpdatePage(page *pages.Page) error
	DeletePage(id model.ID) error
	MovePage(page *pages.Page) error

	// Layouts — legacy sdk-typed surface, retired in Stage 3.3.5.E1.
	ListLayouts() ([]*pages.Layout, error)
	GetLayout(id model.ID) (*pages.Layout, error)
	CreateLayout(layout *pages.Layout) error
	UpdateLayout(layout *pages.Layout) error
	DeleteLayout(id model.ID) error

	// Snippets — legacy sdk-typed surface, retired in Stage 3.3.5.E1.
	// No GetSnippet: snippets are resolved by qualified name via ListSnippets.
	ListSnippets() ([]*pages.Snippet, error)
	CreateSnippet(snippet *pages.Snippet) error
	UpdateSnippet(snippet *pages.Snippet) error
	DeleteSnippet(id model.ID) error
	MoveSnippet(snippet *pages.Snippet) error

	// Stage 3.3.5.C1 gen-typed Page/Layout/Snippet surface — additive
	// alongside the legacy sdk-typed methods. Production wiring in
	// mdl/backend/mpr/backend.go; mock stubs in mdl/backend/mock/
	// return descriptive errors per the MockBackend audit rule.
	ListPagesGen() ([]*genPg.Page, error)
	GetPageGen(id model.ID) (*genPg.Page, error)
	CreatePageGen(parentUUID, containmentName string, page *genPg.Page) error
	UpdatePageGen(page *genPg.Page) error

	ListLayoutsGen() ([]*genPg.Layout, error)
	GetLayoutGen(id model.ID) (*genPg.Layout, error)
	CreateLayoutGen(parentUUID, containmentName string, layout *genPg.Layout) error
	UpdateLayoutGen(layout *genPg.Layout) error

	ListSnippetsGen() ([]*genPg.Snippet, error)
	GetSnippetGen(id model.ID) (*genPg.Snippet, error)
	CreateSnippetGen(parentUUID, containmentName string, snippet *genPg.Snippet) error
	UpdateSnippetGen(snippet *genPg.Snippet) error

	// GetPageContainerUUID resolves the parent container UUID (folder
	// or module ID) of a Page unit. Gen objects do not carry container
	// IDs; this helper bridges Page-level lint rules and other
	// consumers that need to build qualified names from gen-typed
	// Page listings.
	GetPageContainerUUID(id model.ID) (model.ID, error)

	// Stage 3.3.5.D5.c gen-typed delete + move: thin wrappers over the
	// modelsdk writer's DeleteUnit / UpdateUnitContainer that keep the
	// executor off the sdk-typed Page/Layout/Snippet structs entirely.
	// The MovePageGen / MoveSnippetGen / MoveLayoutGen signatures take
	// unit ID + new container ID directly; the executor is responsible
	// for resolving the new container before calling.
	DeletePageGen(id model.ID) error
	MovePageGen(id, containerID model.ID) error

	DeleteLayoutGen(id model.ID) error
	MoveLayoutGen(id, containerID model.ID) error

	DeleteSnippetGen(id model.ID) error
	MoveSnippetGen(id, containerID model.ID) error

	// Building blocks and page templates (read-only)
	ListBuildingBlocks() ([]*pages.BuildingBlock, error)
	ListPageTemplates() ([]*pages.PageTemplate, error)
}
