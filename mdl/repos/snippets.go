// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// SnippetReader / SnippetWriter / SnippetRepository — signatures
// intentionally minimal until Stage 3 cutover. Snippets share the
// pages gen package and are persisted under "Pages$Snippet" units.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// and produce an MPR implementation.
type SnippetReader interface {
	Get(id model.ID) (*genPg.Snippet, error)
	List(moduleID model.ID) ([]*genPg.Snippet, error)
	ListAll() ([]*genPg.Snippet, error)
	FindByQualifiedName(qn string) (*genPg.Snippet, error)

	// GetContainerUUID returns the parent container UUID (folder or
	// module ID) of a snippet unit. Codec-decoded gen objects do not
	// carry container linkage, so callers retrieve it from the MPR Unit
	// table by UnitID. Returns "" with a non-nil error if the unit is
	// not found.
	GetContainerUUID(id model.ID) (model.ID, error)
}

type SnippetWriter interface {
	Create(parentUUID string, containmentName string, snippet *genPg.Snippet) error
	Update(snippet *genPg.Snippet) error
	Delete(id model.ID) error
	Move(id model.ID, newParentUUID string) error
}

type SnippetRepository interface {
	SnippetReader
	SnippetWriter
}
