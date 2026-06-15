// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/model"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// pageBackend implements the read-only gen-typed Page / Layout / Snippet
// surface by wrapping modelsdk-native repos. Write methods (Create/Update/Delete/
// Move) remain on MprBackend.
type pageBackend struct {
	writer *mmpr.Writer
}

func newPageBackend(writer *mmpr.Writer) *pageBackend {
	return &pageBackend{writer: writer}
}

func (b *pageBackend) ListPagesGen() ([]*genPg.Page, error) {
	return mprrepos.NewPageRepository(b.writer).ListAll()
}

func (b *pageBackend) GetPageGen(id model.ID) (*genPg.Page, error) {
	return mprrepos.NewPageRepository(b.writer).Get(id)
}

func (b *pageBackend) ListLayoutsGen() ([]*genPg.Layout, error) {
	return mprrepos.NewLayoutRepository(b.writer).ListAll()
}

func (b *pageBackend) GetLayoutGen(id model.ID) (*genPg.Layout, error) {
	return mprrepos.NewLayoutRepository(b.writer).Get(id)
}

func (b *pageBackend) ListSnippetsGen() ([]*genPg.Snippet, error) {
	return mprrepos.NewSnippetRepository(b.writer).ListAll()
}

func (b *pageBackend) GetSnippetGen(id model.ID) (*genPg.Snippet, error) {
	return mprrepos.NewSnippetRepository(b.writer).Get(id)
}

func (b *pageBackend) GetPageContainerUUID(id model.ID) (model.ID, error) {
	return mprrepos.NewPageRepository(b.writer).GetContainerUUID(id)
}
