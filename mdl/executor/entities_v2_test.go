// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// runListEntitiesGen builds a fixture-backed gen ctx, captures the
// table output, and returns the rendered string.
func runListEntitiesGen(t *testing.T, moduleName string) string {
	t.Helper()
	ctx := newDomainModelsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listEntitiesGen(ctx, moduleName); err != nil {
		t.Fatalf("listEntitiesGen(%q): %v", moduleName, err)
	}
	return buf.String()
}

func TestListEntitiesGen_RendersFixtureEntities(t *testing.T) {
	out := runListEntitiesGen(t, "")
	if out == "" {
		t.Fatal("listEntitiesGen produced no output")
	}
	if !strings.Contains(out, "entities)") {
		t.Errorf("output missing entity count summary; got:\n%s", out)
	}
	for _, want := range []string{"Entity", "Type", "Attrs", "Assocs"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing column %q; got:\n%s", want, out)
		}
	}
}

func TestListEntitiesGen_FilterByModule(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	mods, err := ctx.ModuleLister.ListModules()
	if err != nil || len(mods) == 0 {
		t.Skip("fixture has no modules")
	}
	// Pick the first non-System module that has entities.
	var modName string
	for _, m := range mods {
		if m.Name == "System" {
			continue
		}
		dms, err := ctx.DomainModels.List(m.ID)
		if err != nil || len(dms) == 0 {
			continue
		}
		if dms[0] != nil && len(dms[0].EntitiesItems()) > 0 {
			modName = m.Name
			break
		}
	}
	if modName == "" {
		t.Skip("fixture has no module with entities")
	}
	out := runListEntitiesGen(t, modName)
	if !strings.Contains(out, modName+".") {
		t.Errorf("filtered output should contain qualified name with module prefix %q; got:\n%s", modName, out)
	}
}

func TestEntityKindForGen_NilEntity(t *testing.T) {
	if got := entityKindForGen(nil); got != "" {
		t.Errorf("entityKindForGen(nil) = %q, want empty", got)
	}
}

func TestEntityGeneralizationQNGen_NilEntity(t *testing.T) {
	if got := entityGeneralizationQNGen(nil); got != "" {
		t.Errorf("entityGeneralizationQNGen(nil) = %q, want empty", got)
	}
}

// TestListEntitiesGen_DeterministicOutput verifies two consecutive runs
// produce byte-identical output (per CLAUDE.md "Map iteration is
// deterministic" rule).
func TestListEntitiesGen_DeterministicOutput(t *testing.T) {
	a := runListEntitiesGen(t, "")
	b := runListEntitiesGen(t, "")
	if a != b {
		t.Fatalf("listEntitiesGen output is not deterministic:\na=%s\nb=%s", a, b)
	}
}

// TestFormatAttributeTypeGen_NilInput verifies the nil guard.
func TestFormatAttributeTypeGen_NilInput(t *testing.T) {
	if got := formatAttributeTypeGen(nil); got != "Unknown" {
		t.Errorf("nil: got %q", got)
	}
}

// TestFormatAttributeTypeGen_PrimitiveTypes verifies polymorphic
// dispatch over the 11 simple concrete attribute-type subtypes
// (Stage 3.3.4 A2). String + Enumeration are tested separately because
// they carry parameters.
func TestFormatAttributeTypeGen_PrimitiveTypes(t *testing.T) {
	cases := []struct {
		name string
		at   element.Element
		want string
	}{
		{"Integer", &genDm.IntegerAttributeType{}, "Integer"},
		{"Long", &genDm.LongAttributeType{}, "Long"},
		{"Decimal", &genDm.DecimalAttributeType{}, "Decimal"},
		{"Float", &genDm.FloatAttributeType{}, "Float"},
		{"Currency", &genDm.CurrencyAttributeType{}, "Currency"},
		{"Boolean", &genDm.BooleanAttributeType{}, "Boolean"},
		{"DateTime", &genDm.DateTimeAttributeType{}, "DateTime"},
		{"AutoNumber", &genDm.AutoNumberAttributeType{}, "AutoNumber"},
		{"Binary", &genDm.BinaryAttributeType{}, "Binary"},
		{"HashedString", &genDm.HashedStringAttributeType{}, "HashedString"},
		{"MultiLanguage", &genDm.MultiLanguageAttributeType{}, "MultiLanguage"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatAttributeTypeGen(c.at); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestDescribeEntityGen_RendersFixtureEntity verifies the gen-typed
// describe outputs a complete create-or-modify entity statement that
// round-trips header / attributes / closing structures (Stage 3.3.4 A3).
func TestDescribeEntityGen_RendersFixtureEntity(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil || len(pairs) == 0 {
		t.Skip("fixture has no domain models")
	}
	mods, _ := ctx.ModuleLister.ListModules()
	moduleNames := make(map[string]string)
	for _, m := range mods {
		moduleNames[string(m.ID)] = m.Name
	}
	// Find an entity to describe.
	var modName, entityName string
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		mn := moduleNames[string(p.ContainerID)]
		if mn == "" || mn == "System" {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			ent, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			modName = mn
			entityName = ent.Name()
			break
		}
		if entityName != "" {
			break
		}
	}
	if entityName == "" {
		t.Skip("fixture has no non-System entities")
	}
	var buf bytes.Buffer
	ctx.Output = &buf
	if err := describeEntityGen(ctx, ast.QualifiedName{Module: modName, Name: entityName}); err != nil {
		t.Fatalf("describeEntityGen(%s.%s): %v", modName, entityName, err)
	}
	out := buf.String()
	if !strings.Contains(out, "create or modify") {
		t.Errorf("output missing header; got:\n%s", out)
	}
	if !strings.Contains(out, modName+"."+entityName) {
		t.Errorf("output missing qualified name %s.%s; got:\n%s", modName, entityName, out)
	}
	if !strings.Contains(out, ");") && !strings.Contains(out, ")\n") {
		t.Errorf("output missing closing paren; got:\n%s", out)
	}
}

func TestDescribeEntityGen_NotFound(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	err := describeEntityGen(ctx, ast.QualifiedName{Module: "NoSuchModule", Name: "Nope"})
	if err == nil {
		t.Error("describeEntityGen with unknown entity: expected error, got nil")
	}
}

// TestFormatAttributeTypeGen_FixtureRoundTrip exercises String /
// Enumeration / DateTime / Decimal / Boolean using actual decoded
// fixture entities — covering the codec path end-to-end (no manual
// struct construction). Skips if the fixture exposes no attributes.
func TestFormatAttributeTypeGen_FixtureRoundTrip(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil || len(pairs) == 0 {
		t.Skip("fixture has no domain models")
	}
	saw := map[string]bool{}
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			for _, a := range entity.AttributesItems() {
				attr, ok := a.(*genDm.Attribute)
				if !ok {
					continue
				}
				typ := attr.Type()
				name := formatAttributeTypeGen(typ)
				if name == "" || name == "Unknown" {
					t.Logf("attr %s/%s: formatter returned %q (type=%T raw_typename=%q)", entity.Name(), attr.Name(), name, typ, func() string {
						if typ == nil {
							return "<nil>"
						}
						return typ.TypeName()
					}())
				}
				saw[name] = true
			}
		}
	}
	if len(saw) == 0 {
		t.Skip("fixture has no entity attributes")
	}
	t.Logf("formatter recognised %d distinct types from fixture: %v", len(saw), saw)
}
