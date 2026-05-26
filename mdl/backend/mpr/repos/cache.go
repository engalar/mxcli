// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/mdl/repos"
	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type readerCache struct{ r *mmpr.Reader }

func NewReaderCache(w *mmpr.Writer) repos.ReaderCache {
	return &readerCache{r: w.ConcreteReader()}
}

// mmpr.Reader has no per-unit cache; full invalidation is correct here.
func (c *readerCache) Invalidate()               { c.r.InvalidateCache() }
func (c *readerCache) InvalidateUnit(_ model.ID) { c.r.InvalidateCache() }
