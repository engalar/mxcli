// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"context"
	"sync"

	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// microflowBackend implements the gen-typed Microflow/Nanoflow surface
// with per-query caching via microflowCache.
type microflowBackend struct {
	writer  *mmpr.Writer
	mfCache *microflowCache
	nfCache *microflowNanoflowCache
}

func newMicroflowBackend(writer *mmpr.Writer) *microflowBackend {
	return &microflowBackend{
		writer:  writer,
		mfCache: newMicroflowCache(),
		nfCache: newMicroflowNanoflowCache(),
	}
}

func (b *microflowBackend) ListMicroflowsGen() ([]*genMf.Microflow, error) {
	repo := mprrepos.NewMicroflowRepository(b.writer)
	return b.mfCache.get(context.Background(), repo.ListAll)
}

func (b *microflowBackend) ListNanoflowsGen() ([]*genMf.Nanoflow, error) {
	repo := mprrepos.NewNanoflowRepository(b.writer)
	return b.nfCache.get(context.Background(), func() ([]*genMf.Nanoflow, error) {
		return repo.List("")
	})
}

func (b *microflowBackend) GetMicroflowGen(id model.ID) (*genMf.Microflow, error) {
	return mprrepos.NewMicroflowRepository(b.writer).Get(id)
}

func (b *microflowBackend) IsRule(qualifiedName string) (bool, error) {
	return mprrepos.NewMicroflowRepository(b.writer).IsRule(qualifiedName)
}

// InvalidateCache clears the cached listings so the next call re-fetches.
func (b *microflowBackend) InvalidateCache() {
	b.mfCache.invalidate()
	b.nfCache.invalidate()
}

// ── microflowCache ─────────────────────────────────────────────────────

type microflowCache struct {
	mu    sync.RWMutex
	items []*genMf.Microflow
	valid bool
}

func newMicroflowCache() *microflowCache { return &microflowCache{} }

func (c *microflowCache) get(ctx context.Context, load func() ([]*genMf.Microflow, error)) ([]*genMf.Microflow, error) {
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

func (c *microflowCache) invalidate() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
}

// ── microflowNanoflowCache ─────────────────────────────────────────────

type microflowNanoflowCache struct {
	mu    sync.RWMutex
	items []*genMf.Nanoflow
	valid bool
}

func newMicroflowNanoflowCache() *microflowNanoflowCache { return &microflowNanoflowCache{} }

func (c *microflowNanoflowCache) get(ctx context.Context, load func() ([]*genMf.Nanoflow, error)) ([]*genMf.Nanoflow, error) {
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

func (c *microflowNanoflowCache) invalidate() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
}
