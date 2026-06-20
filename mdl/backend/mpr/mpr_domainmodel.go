// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"context"
	"sync"

	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/modelsdk/meta"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// domainModelBackend implements the read-only gen-typed DomainModel surface
// with per-query caching.
type domainModelBackend struct {
	writer      *mmpr.Writer
	domainCache *domainModelListCache
}

func newDomainModelBackend(writer *mmpr.Writer) *domainModelBackend {
	return &domainModelBackend{
		writer:      writer,
		domainCache: newDomainModelListCache(),
	}
}

func (b *domainModelBackend) ListDomainModelsGen() ([]*genDm.DomainModel, error) {
	return b.domainCache.get(context.Background(), func() ([]*genDm.DomainModel, error) {
		dms, err := mprrepos.NewDomainModelRepository(b.writer).List("")
		if err != nil {
			return nil, err
		}
		return append(dms, builtinSystemDomainModel()), nil
	})
}

func (b *domainModelBackend) GetDomainModelGen(moduleID model.ID) (*genDm.DomainModel, error) {
	if string(moduleID) == meta.SystemModuleID {
		return builtinSystemDomainModel(), nil
	}
	dms, err := mprrepos.NewDomainModelRepository(b.writer).List(moduleID)
	if err != nil {
		return nil, err
	}
	if len(dms) == 0 {
		return nil, nil
	}
	return dms[0], nil
}

func (b *domainModelBackend) GetDomainModelByIDGen(id model.ID) (*genDm.DomainModel, error) {
	return mprrepos.NewDomainModelRepository(b.writer).Get(id)
}

func (b *domainModelBackend) InvalidateCache() {
	b.domainCache.invalidate()
}

// ── typed cache ────────────────────────────────────────────────────────

type domainModelListCache struct {
	mu    sync.RWMutex
	items []*genDm.DomainModel
	valid bool
}

func newDomainModelListCache() *domainModelListCache { return &domainModelListCache{} }

func (c *domainModelListCache) get(ctx context.Context, load func() ([]*genDm.DomainModel, error)) ([]*genDm.DomainModel, error) {
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

func (c *domainModelListCache) invalidate() {
	c.mu.Lock()
	c.valid = false
	c.mu.Unlock()
}
