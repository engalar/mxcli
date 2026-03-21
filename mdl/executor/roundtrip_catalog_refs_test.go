// SPDX-License-Identifier: Apache-2.0

//go:build integration

// Tests for catalog reference extraction and SHOW CALLERS/CALLEES/REFERENCES/IMPACT commands.
// These verify that cross-references between microflows, entities, pages, and associations
// are correctly registered in the catalog refs table.
package executor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
)

// --- Helpers ---

// buildCatalogFull triggers a full catalog rebuild with references.
func buildCatalogFull(t *testing.T, env *testEnv) {
	t.Helper()
	if err := env.executor.Execute(&ast.RefreshCatalogStmt{Full: true, Force: true}); err != nil {
		t.Fatalf("REFRESH CATALOG FULL FORCE failed: %v", err)
	}
}

// countRefs returns the number of refs matching the given source, target, and kind.
// Pass empty string for any parameter to skip that filter.
func countRefs(t *testing.T, env *testEnv, sourceName, targetName, refKind string) int {
	t.Helper()
	if env.executor.catalog == nil {
		t.Fatal("catalog not built")
	}

	conditions := []string{"1=1"}
	if sourceName != "" {
		conditions = append(conditions, fmt.Sprintf("SourceName = '%s'", sourceName))
	}
	if targetName != "" {
		conditions = append(conditions, fmt.Sprintf("TargetName = '%s'", targetName))
	}
	if refKind != "" {
		conditions = append(conditions, fmt.Sprintf("RefKind = '%s'", refKind))
	}

	query := "SELECT COUNT(*) AS cnt FROM refs WHERE " + strings.Join(conditions, " AND ")
	result, err := env.executor.catalog.Query(query)
	if err != nil {
		t.Fatalf("refs query failed: %v", err)
	}
	if result.Count == 0 || len(result.Rows) == 0 {
		return 0
	}
	// Parse count from first row
	cntStr := fmt.Sprintf("%v", result.Rows[0][0])
	var cnt int
	fmt.Sscanf(cntStr, "%d", &cnt)
	return cnt
}

// assertRefExists verifies that at least one ref row exists matching the criteria.
func assertRefExists(t *testing.T, env *testEnv, sourceName, targetName, refKind string) {
	t.Helper()
	cnt := countRefs(t, env, sourceName, targetName, refKind)
	if cnt == 0 {
		t.Errorf("expected ref (source=%q, target=%q, kind=%q) but found none", sourceName, targetName, refKind)
	}
}

// assertNoRef verifies that no ref row exists matching the criteria.
func assertNoRef(t *testing.T, env *testEnv, sourceName, targetName, refKind string) {
	t.Helper()
	cnt := countRefs(t, env, sourceName, targetName, refKind)
	if cnt > 0 {
		t.Errorf("expected no ref (source=%q, target=%q, kind=%q) but found %d", sourceName, targetName, refKind, cnt)
	}
}

// --- Tier 1: Direct refs table verification ---

func TestCatalogRefs_MicroflowCallsMicroflow(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	// Create target microflow
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.TargetMf () RETURNS Boolean
BEGIN
  RETURN true;
END;`, mod)); err != nil {
		t.Fatalf("Failed to create target microflow: %v", err)
	}

	// Create caller microflow
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CallerMf () RETURNS Boolean
BEGIN
  $Result = CALL MICROFLOW %s.TargetMf ();
  RETURN $Result;
END;`, mod, mod)); err != nil {
		t.Fatalf("Failed to create caller microflow: %v", err)
	}

	buildCatalogFull(t, env)
	assertRefExists(t, env, mod+".CallerMf", mod+".TargetMf", "call")
}

func TestCatalogRefs_MicroflowCreatesEntity(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.RefCustomer (Name: String(100));`, mod)); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CreatorMf () RETURNS Boolean
BEGIN
  $Obj = CREATE %s.RefCustomer;
  RETURN true;
END;`, mod, mod)); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	buildCatalogFull(t, env)
	assertRefExists(t, env, mod+".CreatorMf", mod+".RefCustomer", "create")
}

func TestCatalogRefs_MicroflowRetrievesEntity(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.RefProduct (Code: String(50));`, mod)); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.RetrieverMf () RETURNS Boolean
BEGIN
  RETRIEVE $Items FROM %s.RefProduct;
  RETURN true;
END;`, mod, mod)); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	buildCatalogFull(t, env)
	assertRefExists(t, env, mod+".RetrieverMf", mod+".RefProduct", "retrieve")
}

func TestCatalogRefs_Association(t *testing.T) {
	t.Skip("TODO: association references not yet extracted into refs table (RefKindAssociate defined but unused)")

	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.RefParent (Name: String(100));`, mod)); err != nil {
		t.Fatal(err)
	}
	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.RefChild (Label: String(100));`, mod)); err != nil {
		t.Fatal(err)
	}
	if err := env.executeMDL(fmt.Sprintf(`CREATE ASSOCIATION %s.RefChild_RefParent FROM %s.RefChild TO %s.RefParent;`, mod, mod, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)
	assertRefExists(t, env, mod+".RefChild_RefParent", mod+".RefParent", "associate")
}

func TestCatalogRefs_MultipleRefKindsToSameTarget(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.RefOrder (OrderNum: String(50));`, mod)); err != nil {
		t.Fatal(err)
	}

	// Microflow that both creates and retrieves the same entity
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.MultiRefMf () RETURNS Boolean
BEGIN
  $Obj = CREATE %s.RefOrder;
  RETRIEVE $List FROM %s.RefOrder;
  RETURN true;
END;`, mod, mod, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)
	assertRefExists(t, env, mod+".MultiRefMf", mod+".RefOrder", "create")
	assertRefExists(t, env, mod+".MultiRefMf", mod+".RefOrder", "retrieve")
}

func TestCatalogRefs_NoReferences(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.Orphan (Name: String(100));`, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)

	// No microflow or page references this entity
	cnt := countRefs(t, env, "", mod+".Orphan", "")
	if cnt > 0 {
		t.Errorf("expected no references to orphan entity, found %d", cnt)
	}
}

// --- Tier 2: SHOW command output verification ---

func TestCatalogRefs_ShowCallersOf(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	// Create target
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CalleeA () RETURNS Boolean
BEGIN
  RETURN true;
END;`, mod)); err != nil {
		t.Fatal(err)
	}

	// Create caller
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CallerA () RETURNS Boolean
BEGIN
  $R = CALL MICROFLOW %s.CalleeA ();
  RETURN $R;
END;`, mod, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)

	// Execute SHOW CALLERS OF
	env.output.Reset()
	err := env.executor.Execute(&ast.ShowStmt{
		ObjectType: ast.ShowCallers,
		Name:       parseQualifiedName(mod + ".CalleeA"),
	})
	if err != nil {
		t.Fatalf("SHOW CALLERS failed: %v", err)
	}

	output := env.output.String()
	if !strings.Contains(output, mod+".CallerA") {
		t.Errorf("expected output to contain %s.CallerA, got:\n%s", mod, output)
	}
}

func TestCatalogRefs_ShowCalleesOf(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	// Create callee
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CalleeB () RETURNS Boolean
BEGIN
  RETURN true;
END;`, mod)); err != nil {
		t.Fatal(err)
	}

	// Create caller
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CallerB () RETURNS Boolean
BEGIN
  $R = CALL MICROFLOW %s.CalleeB ();
  RETURN $R;
END;`, mod, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)

	env.output.Reset()
	err := env.executor.Execute(&ast.ShowStmt{
		ObjectType: ast.ShowCallees,
		Name:       parseQualifiedName(mod + ".CallerB"),
	})
	if err != nil {
		t.Fatalf("SHOW CALLEES failed: %v", err)
	}

	output := env.output.String()
	if !strings.Contains(output, mod+".CalleeB") {
		t.Errorf("expected output to contain %s.CalleeB, got:\n%s", mod, output)
	}
}

func TestCatalogRefs_ShowCallersTransitive(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	// Create chain: CallerC1 -> CallerC2 -> CalleeC
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CalleeC () RETURNS Boolean
BEGIN
  RETURN true;
END;`, mod)); err != nil {
		t.Fatal(err)
	}

	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CallerC2 () RETURNS Boolean
BEGIN
  $R = CALL MICROFLOW %s.CalleeC ();
  RETURN $R;
END;`, mod, mod)); err != nil {
		t.Fatal(err)
	}

	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.CallerC1 () RETURNS Boolean
BEGIN
  $R = CALL MICROFLOW %s.CallerC2 ();
  RETURN $R;
END;`, mod, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)

	// Non-transitive: only direct callers
	env.output.Reset()
	err := env.executor.Execute(&ast.ShowStmt{
		ObjectType: ast.ShowCallers,
		Name:       parseQualifiedName(mod + ".CalleeC"),
	})
	if err != nil {
		t.Fatalf("SHOW CALLERS failed: %v", err)
	}
	output := env.output.String()
	if !strings.Contains(output, mod+".CallerC2") {
		t.Errorf("expected direct caller CallerC2 in output:\n%s", output)
	}

	// Transitive: should include CallerC1 too
	env.output.Reset()
	err = env.executor.Execute(&ast.ShowStmt{
		ObjectType: ast.ShowCallers,
		Name:       parseQualifiedName(mod + ".CalleeC"),
		Transitive: true,
	})
	if err != nil {
		t.Fatalf("SHOW CALLERS TRANSITIVE failed: %v", err)
	}
	output = env.output.String()
	if !strings.Contains(output, mod+".CallerC2") {
		t.Errorf("expected CallerC2 in transitive output:\n%s", output)
	}
	if !strings.Contains(output, mod+".CallerC1") {
		t.Errorf("expected CallerC1 in transitive output:\n%s", output)
	}
}

func TestCatalogRefs_ShowReferencesTo(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.RefTarget (Name: String(100));`, mod)); err != nil {
		t.Fatal(err)
	}

	// Microflow creates the entity
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.RefCreator () RETURNS Boolean
BEGIN
  $Obj = CREATE %s.RefTarget;
  RETURN true;
END;`, mod, mod)); err != nil {
		t.Fatal(err)
	}

	// Another microflow retrieves it
	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.RefRetriever () RETURNS Boolean
BEGIN
  RETRIEVE $List FROM %s.RefTarget;
  RETURN true;
END;`, mod, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)

	env.output.Reset()
	err := env.executor.Execute(&ast.ShowStmt{
		ObjectType: ast.ShowReferences,
		Name:       parseQualifiedName(mod + ".RefTarget"),
	})
	if err != nil {
		t.Fatalf("SHOW REFERENCES TO failed: %v", err)
	}

	output := env.output.String()
	if !strings.Contains(output, mod+".RefCreator") {
		t.Errorf("expected RefCreator in references output:\n%s", output)
	}
	if !strings.Contains(output, mod+".RefRetriever") {
		t.Errorf("expected RefRetriever in references output:\n%s", output)
	}
}

func TestCatalogRefs_ShowReferencesNoResults(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.Unreferenced (Name: String(100));`, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)

	env.output.Reset()
	err := env.executor.Execute(&ast.ShowStmt{
		ObjectType: ast.ShowReferences,
		Name:       parseQualifiedName(mod + ".Unreferenced"),
	})
	if err != nil {
		t.Fatalf("SHOW REFERENCES TO failed: %v", err)
	}

	output := env.output.String()
	if !strings.Contains(output, "no references found") {
		t.Errorf("expected 'no references found' in output:\n%s", output)
	}
}

func TestCatalogRefs_ShowImpactOf(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	mod := testModule

	if err := env.executeMDL(fmt.Sprintf(`CREATE OR MODIFY PERSISTENT ENTITY %s.ImpactEntity (Name: String(100));`, mod)); err != nil {
		t.Fatal(err)
	}

	if err := env.executeMDL(fmt.Sprintf(`CREATE MICROFLOW %s.ImpactMf () RETURNS Boolean
BEGIN
  $Obj = CREATE %s.ImpactEntity;
  RETURN true;
END;`, mod, mod)); err != nil {
		t.Fatal(err)
	}

	buildCatalogFull(t, env)

	env.output.Reset()
	err := env.executor.Execute(&ast.ShowStmt{
		ObjectType: ast.ShowImpact,
		Name:       parseQualifiedName(mod + ".ImpactEntity"),
	})
	if err != nil {
		t.Fatalf("SHOW IMPACT failed: %v", err)
	}

	output := env.output.String()
	if !strings.Contains(output, mod+".ImpactMf") {
		t.Errorf("expected ImpactMf in impact output:\n%s", output)
	}
}
