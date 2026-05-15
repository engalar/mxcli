// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"github.com/mendixlabs/mxcli/sdk/pages"
)

// Re-export sdk/pages types still used by the read-only BuildingBlock /
// PageTemplate surface. The Page/Layout/Snippet aliases were retired in
// Stage 3.3.5.E1 alongside the legacy ListPages/ListLayouts/ListSnippets
// and Delete* PageBackend methods.
type (
	BuildingBlock = pages.BuildingBlock
	PageTemplate  = pages.PageTemplate
)

// PageBackend provides page, layout, and snippet operations.
type PageBackend interface {
	// Stage 3.3.5.C1 gen-typed Page/Layout/Snippet surface — sole
	// supported read/write API after the Stage 3.3.5.E1 cutover.
	// Production wiring in mdl/backend/mpr/backend.go; mock stubs in
	// mdl/backend/mock/ return descriptive errors per the MockBackend
	// audit rule.
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
	ListBuildingBlocks() ([]*BuildingBlock, error)
	ListPageTemplates() ([]*PageTemplate, error)
}
