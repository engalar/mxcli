// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// PageReader provides read-only access to page, layout, and snippet documents.
// Satisfied by any backend; also embedded in LintReader and CatalogReader
// to eliminate duplicate method declarations.
type PageReader interface {
	ListPagesGen() ([]*genPg.Page, error)
	GetPageGen(id model.ID) (*genPg.Page, error)

	ListLayoutsGen() ([]*genPg.Layout, error)
	GetLayoutGen(id model.ID) (*genPg.Layout, error)

	ListSnippetsGen() ([]*genPg.Snippet, error)
	GetSnippetGen(id model.ID) (*genPg.Snippet, error)

	// GetPageContainerUUID resolves the parent container UUID (folder or
	// module ID) of a Page unit. Gen objects do not carry container IDs;
	// this helper bridges Page-level lint rules and other consumers that
	// need to build qualified names from gen-typed Page listings.
	GetPageContainerUUID(id model.ID) (model.ID, error)
}

// PageWriter provides write access to page, layout, and snippet documents.
// Only executors and migration commands should depend on this interface.
//
// The Delete*/Move* methods are thin wrappers over the modelsdk writer's
// DeleteUnit / UpdateUnitContainer that keep the executor off the sdk-typed
// Page/Layout/Snippet structs entirely. The Move* signatures take unit ID +
// new container ID directly; the caller resolves the new container first.
type PageWriter interface {
	CreatePageGen(parentUUID, containmentName string, page *genPg.Page) error
	UpdatePageGen(page *genPg.Page) error
	DeletePageGen(id model.ID) error
	MovePageGen(id, containerID model.ID) error

	CreateLayoutGen(parentUUID, containmentName string, layout *genPg.Layout) error
	UpdateLayoutGen(layout *genPg.Layout) error
	DeleteLayoutGen(id model.ID) error
	MoveLayoutGen(id, containerID model.ID) error

	// GetContainerID resolves a module ID and optional folder path to a BSON
	// container UUID. When folder is empty, moduleID itself is returned.
	// Folder path segments are separated by "/"; missing folders are created.
	GetContainerID(moduleID model.ID, folder string) (model.ID, error)

	CreateSnippetGen(parentUUID, containmentName string, snippet *genPg.Snippet) error
	UpdateSnippetGen(snippet *genPg.Snippet) error
	DeleteSnippetGen(id model.ID) error
	MoveSnippetGen(id, containerID model.ID) error
}

// PageBackend composes PageReader and PageWriter, providing the full
// gen-typed Page/Layout/Snippet read/write surface — the sole supported API
// after the Stage 3.3.5.E1 cutover. Production wiring lives in
// mdl/backend/mpr/backend.go; mock stubs in mdl/backend/mock/ return
// descriptive errors per the MockBackend audit rule. FullBackend embeds this
// interface. Consumers that only read pages should declare PageReader;
// consumers that only write should declare PageWriter.
type PageBackend interface {
	PageReader
	PageWriter
}
