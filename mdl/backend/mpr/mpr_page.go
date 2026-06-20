// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"context"
	"sync"

	"github.com/mendixlabs/mxcli/model"
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// pageBackend implements the read-only gen-typed Page / Layout / Snippet
// surface by wrapping modelsdk-native repos.
type pageBackend struct {
	writer       *mmpr.Writer
	pageCache    *pageListCache
	layoutCache  *layoutListCache
	snippetCache *snippetListCache
}

func newPageBackend(writer *mmpr.Writer) *pageBackend {
	return &pageBackend{
		writer:       writer,
		pageCache:    newPageListCache(),
		layoutCache:  newLayoutListCache(),
		snippetCache: newSnippetListCache(),
	}
}

func (b *pageBackend) ListPagesGen() ([]*genPg.Page, error) {
	return b.pageCache.get(context.Background(), func() ([]*genPg.Page, error) {
		return mprrepos.NewPageRepository(b.writer).ListAll()
	})
}

func (b *pageBackend) GetPageGen(id model.ID) (*genPg.Page, error) {
	return mprrepos.NewPageRepository(b.writer).Get(id)
}

func (b *pageBackend) ListLayoutsGen() ([]*genPg.Layout, error) {
	return b.layoutCache.get(context.Background(), func() ([]*genPg.Layout, error) {
		return mprrepos.NewLayoutRepository(b.writer).ListAll()
	})
}

func (b *pageBackend) GetLayoutGen(id model.ID) (*genPg.Layout, error) {
	return mprrepos.NewLayoutRepository(b.writer).Get(id)
}

func (b *pageBackend) ListSnippetsGen() ([]*genPg.Snippet, error) {
	return b.snippetCache.get(context.Background(), func() ([]*genPg.Snippet, error) {
		return mprrepos.NewSnippetRepository(b.writer).ListAll()
	})
}

func (b *pageBackend) GetSnippetGen(id model.ID) (*genPg.Snippet, error) {
	return mprrepos.NewSnippetRepository(b.writer).Get(id)
}

func (b *pageBackend) GetPageContainerUUID(id model.ID) (model.ID, error) {
	return mprrepos.NewPageRepository(b.writer).GetContainerUUID(id)
}

func (b *pageBackend) InvalidateCache() {
	b.pageCache.invalidate()
	b.layoutCache.invalidate()
	b.snippetCache.invalidate()
}

// ── typed caches ───────────────────────────────────────────────────────

type pageListCache struct {
	mu    sync.RWMutex
	items []*genPg.Page
	valid bool
}

func newPageListCache() *pageListCache { return &pageListCache{} }

func (c *pageListCache) get(ctx context.Context, load func() ([]*genPg.Page, error)) ([]*genPg.Page, error) {
	c.mu.RLock()
	if c.valid {
		defer c.mu.RUnlock()
		return c.items, nil
	}
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid {
		return c.items, nil
	}
	items, err := load()
	if err != nil {
		return nil, err
	}
	c.items = items
	c.valid = true
	return c.items, nil
}

func (c *pageListCache) invalidate() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
}

type layoutListCache struct {
	mu    sync.RWMutex
	items []*genPg.Layout
	valid bool
}

func newLayoutListCache() *layoutListCache { return &layoutListCache{} }

func (c *layoutListCache) get(ctx context.Context, load func() ([]*genPg.Layout, error)) ([]*genPg.Layout, error) {
	c.mu.RLock()
	if c.valid {
		defer c.mu.RUnlock()
		return c.items, nil
	}
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid {
		return c.items, nil
	}
	items, err := load()
	if err != nil {
		return nil, err
	}
	c.items = items
	c.valid = true
	return c.items, nil
}

func (c *layoutListCache) invalidate() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
}

type snippetListCache struct {
	mu    sync.RWMutex
	items []*genPg.Snippet
	valid bool
}

func newSnippetListCache() *snippetListCache { return &snippetListCache{} }

func (c *snippetListCache) get(ctx context.Context, load func() ([]*genPg.Snippet, error)) ([]*genPg.Snippet, error) {
	c.mu.RLock()
	if c.valid {
		defer c.mu.RUnlock()
		return c.items, nil
	}
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid {
		return c.items, nil
	}
	items, err := load()
	if err != nil {
		return nil, err
	}
	c.items = items
	c.valid = true
	return c.items, nil
}

func (c *snippetListCache) invalidate() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
}
