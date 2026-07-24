// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCache_MPKPath(t *testing.T) {
	base := t.TempDir()
	c := NewCache(base)
	path := c.MPKPath(2888, "7.0.3")
	want := filepath.Join(base, "2888", "7.0.3", "module.mpk")
	if path != want {
		t.Errorf("got %q, want %q", path, want)
	}
}

func TestCache_IsCached(t *testing.T) {
	base := t.TempDir()
	c := NewCache(base)
	if c.IsCached(2888, "7.0.3") {
		t.Error("should not be cached initially")
	}
	mpkDir := filepath.Dir(c.MPKPath(2888, "7.0.3"))
	os.MkdirAll(mpkDir, 0755)
	os.WriteFile(c.MPKPath(2888, "7.0.3"), []byte("test"), 0644)
	if !c.IsCached(2888, "7.0.3") {
		t.Error("should be cached after file creation")
	}
}

func TestCache_CatalogRoundTrip(t *testing.T) {
	base := t.TempDir()
	c := NewCache(base)

	data, ok := c.ReadCatalog("default")
	if ok {
		t.Error("should not have cached catalog initially")
	}

	if err := c.WriteCatalog("default", []byte(`{"hello":"world"}`)); err != nil {
		t.Fatal(err)
	}

	data, ok = c.ReadCatalog("default")
	if !ok {
		t.Fatal("should have cached catalog after write")
	}
	if string(data) != `{"hello":"world"}` {
		t.Errorf("got %q, want json data", string(data))
	}
}

func TestCache_CatalogExpiry(t *testing.T) {
	base := t.TempDir()
	c := &Cache{baseDir: base}

	if err := c.WriteCatalog("default", []byte(`"test"`)); err != nil {
		t.Fatal(err)
	}

	data, ok := c.ReadCatalog("default")
	if !ok {
		t.Fatal("catalog should be fresh immediately")
	}
	if string(data) != `"test"` {
		t.Errorf("got %q", string(data))
	}
}
