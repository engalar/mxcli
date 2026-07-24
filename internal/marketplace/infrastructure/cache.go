// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Cache struct {
	baseDir string
}

type catalogCache struct {
	Timestamp time.Time `json:"timestamp"`
	Data      []byte    `json:"data"`
}

const catalogTTL = 24 * time.Hour

func NewCache(baseDir string) *Cache {
	return &Cache{baseDir: baseDir}
}

func (c *Cache) MPKPath(contentID int, version string) string {
	return filepath.Join(c.baseDir, fmt.Sprintf("%d", contentID), version, "module.mpk")
}

func (c *Cache) IsCached(contentID int, version string) bool {
	_, err := os.Stat(c.MPKPath(contentID, version))
	return err == nil
}

func (c *Cache) CatalogPath(profile string) string {
	return filepath.Join(c.baseDir, fmt.Sprintf("catalog-%s.json", profile))
}

func (c *Cache) ReadCatalog(profile string) ([]byte, bool) {
	path := c.CatalogPath(profile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entry catalogCache
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	if time.Since(entry.Timestamp) > catalogTTL {
		return nil, false
	}
	return entry.Data, true
}

func (c *Cache) WriteCatalog(profile string, data []byte) error {
	path := c.CatalogPath(profile)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	entry := catalogCache{Timestamp: time.Now(), Data: data}
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}
