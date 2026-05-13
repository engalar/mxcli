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
