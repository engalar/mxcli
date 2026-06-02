// SPDX-License-Identifier: Apache-2.0

package archtest_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/archtest"
	"github.com/mendixlabs/mxcli/mdl/canonical"
)

func buildTestRegistry(liftFn func(any) (canonical.Persistable, error), hydrateFn func(any, canonical.HydrateCtx) (canonical.Document, []canonical.Warning, error)) *canonical.DefaultRegistry {
	r := canonical.NewDefaultRegistry()
	r.RegisterGenType("TestType$Foo", canonical.Codec{
		LiftFn:    liftFn,
		HydrateFn: hydrateFn,
	})
	return r
}

func TestCodecComplete_allPresent_passes(t *testing.T) {
	r := buildTestRegistry(
		func(any) (canonical.Persistable, error) { return nil, nil },
		func(any, canonical.HydrateCtx) (canonical.Document, []canonical.Warning, error) { return nil, nil, nil },
	)
	rule := archtest.CodecComplete{
		BuildRegistry: func() *canonical.DefaultRegistry { return r },
		Required:      []string{"TestType$Foo"},
	}
	if v := rule.Check(archtest.Package{}); len(v) != 0 {
		t.Errorf("complete codec should have no violations")
	}
}

func TestCodecComplete_nilLiftFn_fails(t *testing.T) {
	r := buildTestRegistry(
		nil,
		func(any, canonical.HydrateCtx) (canonical.Document, []canonical.Warning, error) { return nil, nil, nil },
	)
	rule := archtest.CodecComplete{
		BuildRegistry: func() *canonical.DefaultRegistry { return r },
		Required:      []string{"TestType$Foo"},
		Hint:          "add LiftFn",
	}
	violations := rule.Check(archtest.Package{})
	if len(violations) != 1 {
		t.Fatalf("want 1 violation for nil LiftFn, got %d", len(violations))
	}
	if violations[0].Hint != "add LiftFn" {
		t.Errorf("hint not propagated")
	}
}

func TestCodecComplete_nilHydrateFn_fails(t *testing.T) {
	r := buildTestRegistry(
		func(any) (canonical.Persistable, error) { return nil, nil },
		nil,
	)
	rule := archtest.CodecComplete{
		BuildRegistry: func() *canonical.DefaultRegistry { return r },
		Required:      []string{"TestType$Foo"},
	}
	if v := rule.Check(archtest.Package{}); len(v) != 1 {
		t.Fatalf("want 1 violation for nil HydrateFn, got %d", len(v))
	}
}

func TestCodecComplete_missingRequiredType_fails(t *testing.T) {
	r := canonical.NewDefaultRegistry() // empty
	rule := archtest.CodecComplete{
		BuildRegistry: func() *canonical.DefaultRegistry { return r },
		Required:      []string{"Missing$Type"},
		Hint:          "register missing type",
	}
	violations := rule.Check(archtest.Package{})
	if len(violations) != 1 {
		t.Fatalf("want 1 violation for unregistered type, got %d", len(violations))
	}
}
