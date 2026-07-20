// SPDX-License-Identifier: Apache-2.0

// Package completion provides project-aware shell completions for mxcli.
package completion

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"

	// element.ID type is used in item.ID() return values
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// Completer provides project-aware shell completions.
// Lazy-connects to the MPR on first use; thread-safe.
type Completer struct {
	mu     sync.Mutex
	be     *mprbackend.MprBackend
	ready  bool
	modMap map[string]string // module ID → module name (cached)
}

// EnsureConnected opens the MPR if not already connected.
func (c *Completer) EnsureConnected(mprPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ready {
		return nil
	}
	be := mprbackend.New()
	if err := be.Connect(mprPath); err != nil {
		return fmt.Errorf("connect for completion: %w", err)
	}
	c.be = be
	c.ready = true
	c.buildModMap()
	return nil
}

// Close disconnects and releases resources.
func (c *Completer) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.be != nil {
		_ = c.be.Disconnect()
		c.be = nil
		c.ready = false
		c.modMap = nil
	}
}

// ModuleSuggestions returns matching module names.
func (c *Completer) ModuleSuggestions(prefix string) []string {
	if !c.ready {
		return nil
	}
	modules, err := c.be.ModuleLister().ListModules()
	if err != nil {
		return nil
	}
	var out []string
	for _, m := range modules {
		if prefix == "" || strings.HasPrefix(m.Name, prefix) {
			out = append(out, m.Name)
		}
	}
	sort.Strings(out)
	return out
}

// EntitySuggestions returns "Module.EntityName" strings matching prefix.
func (c *Completer) EntitySuggestions(prefix string) []string {
	if !c.ready || c.modMap == nil {
		return nil
	}
	dms, err := c.be.DomainModelReader().ListDomainModelsGen()
	if err != nil {
		return nil
	}
	var out []string
	for _, dm := range dms {
		modName := c.modMap[string(dm.ID())]
		if modName == "" {
			continue
		}
		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			qn := modName + "." + ent.Name()
			if prefix == "" || strings.HasPrefix(qn, prefix) {
				out = append(out, qn)
			}
		}
	}
	sort.Strings(out)
	return out
}

// MicroflowSuggestions returns "Module.MicroflowName" strings matching prefix.
func (c *Completer) MicroflowSuggestions(prefix string) []string {
	if !c.ready {
		return nil
	}
	list, err := c.be.ListMicroflowsGen()
	if err != nil {
		return nil
	}
	return c.qualifyAndFilter(list, prefix)
}

// NanoflowSuggestions returns "Module.NanoflowName" strings matching prefix.
func (c *Completer) NanoflowSuggestions(prefix string) []string {
	if !c.ready {
		return nil
	}
	list, err := c.be.ListNanoflowsGen()
	if err != nil {
		return nil
	}
	return c.qualifyAndFilterNanoflows(list, prefix)
}

// PageSuggestions returns "Module.PageName" strings matching prefix.
func (c *Completer) PageSuggestions(prefix string) []string {
	if !c.ready {
		return nil
	}
	list, err := c.be.PageReader().ListPagesGen()
	if err != nil {
		return nil
	}
	return c.qualifyAndFilterPages(list, prefix)
}

// LayoutSuggestions returns "Module.LayoutName" strings matching prefix.
func (c *Completer) LayoutSuggestions(prefix string) []string {
	if !c.ready {
		return nil
	}
	list, err := c.be.PageReader().ListLayoutsGen()
	if err != nil {
		return nil
	}
	var out []string
	for _, item := range list {
		modName := c.resolveModuleName(item.ID())
		if modName == "" {
			continue
		}
		qn := modName + "." + item.Name()
		if prefix == "" || strings.HasPrefix(qn, prefix) {
			out = append(out, qn)
		}
	}
	sort.Strings(out)
	return out
}

// ── helpers ────────────────────────────────────────────────────────────

func (c *Completer) buildModMap() {
	modules, err := c.be.ModuleLister().ListModules()
	if err != nil {
		return
	}
	c.modMap = make(map[string]string, len(modules))
	for _, m := range modules {
		c.modMap[string(m.ID)] = m.Name
		// Also try to resolve the domain model ID for this module
		if dm, err := c.be.DomainModelReader().GetDomainModelGen(m.ID); err == nil && dm != nil {
			c.modMap[string(dm.ID())] = m.Name
		}
	}
}

func (c *Completer) qualifyAndFilter(list []*genMf.Microflow, prefix string) []string {
	var out []string
	for _, item := range list {
		modName := c.resolveModuleName(item.ID())
		if modName == "" {
			continue
		}
		qn := modName + "." + item.Name()
		if prefix == "" || strings.HasPrefix(qn, prefix) {
			out = append(out, qn)
		}
	}
	sort.Strings(out)
	return out
}

func (c *Completer) qualifyAndFilterNanoflows(list []*genMf.Nanoflow, prefix string) []string {
	var out []string
	for _, item := range list {
		modName := c.resolveModuleName(item.ID())
		if modName == "" {
			continue
		}
		qn := modName + "." + item.Name()
		if prefix == "" || strings.HasPrefix(qn, prefix) {
			out = append(out, qn)
		}
	}
	sort.Strings(out)
	return out
}

func (c *Completer) qualifyAndFilterPages(list []*genPg.Page, prefix string) []string {
	var out []string
	for _, item := range list {
		modName := c.resolveModuleName(item.ID())
		if modName == "" {
			continue
		}
		qn := modName + "." + item.Name()
		if prefix == "" || strings.HasPrefix(qn, prefix) {
			out = append(out, qn)
		}
	}
	sort.Strings(out)
	return out
}

func (c *Completer) resolveModuleName(id element.ID) string {
	key := string(id)
	if n, ok := c.modMap[key]; ok {
		return n
	}
	return ""
}
