// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// fakeElem is a minimal test stub satisfying element.Element.
type fakeElem struct{ id element.ID }

func (f *fakeElem) ID() element.ID                 { return f.id }
func (f *fakeElem) TypeName() string               { return "Test$Fake" }
func (f *fakeElem) Container() element.Element     { return nil }
func (f *fakeElem) SetContainer(_ element.Element) {}
func (f *fakeElem) Unit() element.Unit             { return nil }
func (f *fakeElem) Raw() bson.Raw                  { return nil }
func (f *fakeElem) IsDirty() bool                  { return false }
func (f *fakeElem) Properties() []element.Property { return nil }

func TestListUnitsWithContainerGen_CacheHitReturnsCached(t *testing.T) {
	prebuilt := []ContainerWithGen[*fakeElem]{
		{Elem: &fakeElem{id: "aaa"}, ContainerID: "container-1"},
	}
	listCalled := false
	list := func() ([]*fakeElem, error) {
		listCalled = true
		return nil, nil
	}
	resolve := func(_ element.ID) (element.ID, error) { return "", nil }
	cacheGet := func() ([]ContainerWithGen[*fakeElem], bool) { return prebuilt, true }
	cachePut := func(_ []ContainerWithGen[*fakeElem]) {}

	got, err := listUnitsWithContainerGen(list, resolve, cacheGet, cachePut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listCalled {
		t.Error("list should not be called on cache hit")
	}
	if len(got) != 1 || got[0].ContainerID != "container-1" {
		t.Errorf("expected cached result, got %v", got)
	}
}

func TestListUnitsWithContainerGen_CacheMissBuildsAndStores(t *testing.T) {
	elems := []*fakeElem{
		{id: "e1"},
		{id: "e2"},
	}
	listCalls := 0
	resolveCalls := 0

	list := func() ([]*fakeElem, error) {
		listCalls++
		return elems, nil
	}
	resolve := func(id element.ID) (element.ID, error) {
		resolveCalls++
		return element.ID("c-" + string(id)), nil
	}
	var stored []ContainerWithGen[*fakeElem]
	cacheGet := func() ([]ContainerWithGen[*fakeElem], bool) { return nil, false }
	cachePut := func(v []ContainerWithGen[*fakeElem]) { stored = v }

	got, err := listUnitsWithContainerGen(list, resolve, cacheGet, cachePut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listCalls != 1 {
		t.Errorf("expected list called once, got %d", listCalls)
	}
	if resolveCalls != 2 {
		t.Errorf("expected resolveContainer called twice, got %d", resolveCalls)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].ContainerID != "c-e1" || got[1].ContainerID != "c-e2" {
		t.Errorf("unexpected container IDs: %v", got)
	}
	if len(stored) != 2 {
		t.Errorf("cachePut not called with full slice: len=%d", len(stored))
	}
}

func TestListUnitsWithContainerGen_ResolveContainerError_KeepsZeroID(t *testing.T) {
	elems := []*fakeElem{
		{id: "good"},
		{id: "bad"},
	}
	list := func() ([]*fakeElem, error) { return elems, nil }
	resolve := func(id element.ID) (element.ID, error) {
		if id == "bad" {
			return "", errors.New("not found")
		}
		return "c-good", nil
	}
	cacheGet := func() ([]ContainerWithGen[*fakeElem], bool) { return nil, false }
	cachePut := func(_ []ContainerWithGen[*fakeElem]) {}

	got, err := listUnitsWithContainerGen(list, resolve, cacheGet, cachePut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].ContainerID != "c-good" {
		t.Errorf("expected c-good, got %q", got[0].ContainerID)
	}
	if got[1].ContainerID != "" {
		t.Errorf("expected zero ID for error case, got %q", got[1].ContainerID)
	}
}

func TestListUnitsWithContainerGen_ListError_Propagates(t *testing.T) {
	listErr := errors.New("db error")
	list := func() ([]*fakeElem, error) { return nil, listErr }
	resolve := func(_ element.ID) (element.ID, error) { return "", nil }
	cacheGet := func() ([]ContainerWithGen[*fakeElem], bool) { return nil, false }
	putCalled := false
	cachePut := func(_ []ContainerWithGen[*fakeElem]) { putCalled = true }

	_, err := listUnitsWithContainerGen(list, resolve, cacheGet, cachePut)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, listErr) {
		t.Errorf("expected listErr, got %v", err)
	}
	if putCalled {
		t.Error("cachePut should not be called on list error")
	}
}
