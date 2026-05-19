// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func runListAssociationsGen(t *testing.T, moduleName string) string {
	t.Helper()
	ctx := newDomainModelsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	ctx.Format = FormatTable
	if err := listAssociations(ctx, moduleName); err != nil {
		t.Fatalf("listAssociations(%q): %v", moduleName, err)
	}
	return buf.String()
}

func TestListAssociationsGen_RendersFixture(t *testing.T) {
	out := runListAssociationsGen(t, "")
	if out == "" {
		t.Fatal("listAssociations produced no output")
	}
	for _, want := range []string{"Qualified Name", "Type", "Owner", "Storage", "associations)"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}
}

func TestListAssociationsGen_DeterministicOutput(t *testing.T) {
	a := runListAssociationsGen(t, "")
	b := runListAssociationsGen(t, "")
	if a != b {
		t.Fatalf("listAssociations output is not deterministic")
	}
}

// TestDescribeAssociationGen_FixtureAssoc finds any association in the
// fixture and exercises describeAssociation on it.
func TestDescribeAssociationGen_FixtureAssoc(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil || len(pairs) == 0 {
		t.Skip("fixture has no domain models")
	}
	mods, _ := ctx.Backend.ListModules()
	moduleNames := map[string]string{}
	for _, m := range mods {
		moduleNames[string(m.ID)] = m.Name
	}
	var modName, assocName string
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		mn := moduleNames[string(p.ContainerID)]
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok {
				continue
			}
			modName = mn
			assocName = assoc.Name()
			break
		}
		if assocName != "" {
			break
		}
	}
	if assocName == "" {
		t.Skip("fixture has no intra-module associations")
	}
	var buf bytes.Buffer
	ctx.Output = &buf
	if err := describeAssociation(ctx, ast.QualifiedName{Module: modName, Name: assocName}); err != nil {
		t.Fatalf("describeAssociation(%s.%s): %v", modName, assocName, err)
	}
	out := buf.String()
	for _, want := range []string{"create association", modName + "." + assocName, "from ", " to ", "type ", "owner ", "delete_behavior "} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}
}

func TestDescribeAssociationGen_NotFound(t *testing.T) {
	ctx := newDomainModelsTestContext(t)
	var buf bytes.Buffer
	ctx.Output = &buf
	err := describeAssociation(ctx, ast.QualifiedName{Module: "NoMod", Name: "NoAssoc"})
	if err == nil {
		t.Error("describeAssociation with unknown association: expected error, got nil")
	}
}
