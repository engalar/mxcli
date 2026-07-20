// SPDX-License-Identifier: Apache-2.0

package executor

import "sync"

// CacheDomain identifies a domain for cache invalidation.
type CacheDomain int

const (
	CacheDomainModules       CacheDomain = iota
	CacheDomainEntities
	CacheDomainMicroflows
	CacheDomainNanoflows
	CacheDomainPages
	CacheDomainSnippets
	CacheDomainLayouts
	CacheDomainWorkflows
	CacheDomainEnumerations
	CacheDomainConstants
	CacheDomainJavaActions
	CacheDomainJavaScriptActions
	CacheDomainSettings
	CacheDomainNavigation
)

// domainCache provides lazy-load + invalidate for a single domain's data.
type domainCache[T any] struct {
	mu     sync.RWMutex
	data   []T
	loaded bool
	loadFn func() ([]T, error)
}

func newDomainCache[T any](loadFn func() ([]T, error)) *domainCache[T] {
	return &domainCache[T]{loadFn: loadFn}
}

func (c *domainCache[T]) Get() ([]T, error) {
	c.mu.RLock()
	if c.loaded {
		defer c.mu.RUnlock()
		return c.data, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loaded {
		return c.data, nil
	}
	var err error
	c.data, err = c.loadFn()
	c.loaded = true
	return c.data, err
}

func (c *domainCache[T]) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loaded = false
	c.data = nil
}
