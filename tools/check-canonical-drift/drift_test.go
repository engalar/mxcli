// tools/check-canonical-drift/drift_test.go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDiff_ChangedLines(t *testing.T) {
	diff := "diff --git a/mdl/executor/cmd_diff_mdl.go b/mdl/executor/cmd_diff_mdl.go\n" +
		"index abc..def 100644\n" +
		"--- a/mdl/executor/cmd_diff_mdl.go\n" +
		"+++ b/mdl/executor/cmd_diff_mdl.go\n" +
		"@@ -93,3 +93,3 @@\n" +
		" unchanged\n" +
		"-old\n" +
		"+new\n"
	changed, _ := parseDiff(diff)
	ranges := changed["mdl/executor/cmd_diff_mdl.go"]
	require.Len(t, ranges, 1)
	assert.Equal(t, 93, ranges[0].start)
	assert.Equal(t, 95, ranges[0].end)
}

func TestParseDiff_MultipleHunks(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_diff_mdl.go\n" +
		"@@ -10,2 +10,2 @@\n" +
		"-old\n" +
		"+new\n" +
		"@@ -50,1 +50,1 @@\n" +
		"-old2\n" +
		"+new2\n"
	changed, _ := parseDiff(diff)
	ranges := changed["mdl/executor/cmd_diff_mdl.go"]
	require.Len(t, ranges, 2)
	assert.Equal(t, 10, ranges[0].start)
	assert.Equal(t, 11, ranges[0].end)
	assert.Equal(t, 50, ranges[1].start)
	assert.Equal(t, 50, ranges[1].end)
}

func TestParseDiff_SkipsNonExecutor(t *testing.T) {
	diff := "+++ b/cmd/mxcli/main.go\n@@ -1 +1 @@\n+line\n"
	changed, _ := parseDiff(diff)
	assert.Empty(t, changed)
}

func TestParseDiff_SkipsTestFiles(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_diff_mdl_test.go\n@@ -1 +1 @@\n+line\n"
	changed, _ := parseDiff(diff)
	assert.Empty(t, changed)
}

func TestParseDiff_ZeroCountHunk(t *testing.T) {
	// @@ -5,3 +5,0 @@ means 3 lines deleted, 0 added — no new-side range
	diff := "+++ b/mdl/executor/cmd_diff_mdl.go\n@@ -5,3 +5,0 @@\n-del1\n-del2\n-del3\n"
	changed, _ := parseDiff(diff)
	assert.Empty(t, changed["mdl/executor/cmd_diff_mdl.go"])
}

func TestParseDiff_NewUnmigratedFunc(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_diff_mdl.go\n" +
		"@@ -0,0 +1,4 @@\n" +
		"+func fooStmtToMDL(ctx *ExecContext) string {\n" +
		"+\treturn \"foo\"\n" +
		"+}\n"
	_, newFuncs := parseDiff(diff)
	require.Len(t, newFuncs, 1)
	assert.Equal(t, "fooStmtToMDL", newFuncs[0])
}

func TestParseDiff_NewMigratedFunc(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_diff_mdl.go\n" +
		"@@ -0,0 +1,4 @@\n" +
		"+func fooStmtToMDL(ctx *ExecContext) string {\n" +
		"+\treturn m.ToMDL()\n" +
		"+}\n"
	_, newFuncs := parseDiff(diff)
	assert.Empty(t, newFuncs)
}

func TestParseDiff_NewToMDLGenFunc(t *testing.T) {
	diff := "+++ b/mdl/executor/cmd_entities_gen.go\n" +
		"@@ -0,0 +1,3 @@\n" +
		"+func barToMDLGen(ctx *ExecContext) string {\n" +
		"+\treturn \"bar\"\n" +
		"+}\n"
	_, newFuncs := parseDiff(diff)
	require.Len(t, newFuncs, 1)
	assert.Equal(t, "barToMDLGen", newFuncs[0])
}

// --- scanSource tests ---

const srcMigrated = `package executor

func entityStmtToMDL(s *ast.CreateEntityStmt) string {
	m, _ := entity.Lift(s)
	return m.ToMDL()
}
`

const srcUnmigrated = `package executor

import "strings"

func associationStmtToMDL(s *ast.CreateAssociationStmt) string {
	var sb strings.Builder
	sb.WriteString("create association ")
	return sb.String()
}
`

const srcToMDLGen = `package executor

func viewEntityToMDLGen(mod string, e *genDm.Entity) string {
	var sb strings.Builder
	sb.WriteString(e.Name())
	return sb.String()
}
`

const srcNonMatching = `package executor

func execCreateEntity(ctx *ExecContext, s *ast.CreateEntityStmt) error {
	return nil
}
`

const srcMultiple = `package executor

import "strings"

func enumerationStmtToMDL(s *ast.CreateEnumerationStmt) string {
	var sb strings.Builder
	sb.WriteString("create enumeration")
	return sb.String()
}

func microflowStmtToMDL(s *ast.CreateMicroflowStmt) string {
	var sb strings.Builder
	sb.WriteString("create microflow")
	return sb.String()
}

func entityStmtToMDL(s *ast.CreateEntityStmt) string {
	m, _ := entity.Lift(s)
	return m.ToMDL()
}
`

func TestScanSource_MigratedNotReported(t *testing.T) {
	fns := scanSource("fake.go", srcMigrated)
	assert.Empty(t, fns)
}

func TestScanSource_UnmigratedReported(t *testing.T) {
	fns := scanSource("fake.go", srcUnmigrated)
	require.Len(t, fns, 1)
	assert.Equal(t, "associationStmtToMDL", fns[0].name)
	assert.Greater(t, fns[0].end, fns[0].start)
}

func TestScanSource_ToMDLGenReported(t *testing.T) {
	fns := scanSource("fake.go", srcToMDLGen)
	require.Len(t, fns, 1)
	assert.Equal(t, "viewEntityToMDLGen", fns[0].name)
}

func TestScanSource_NonMatchingSkipped(t *testing.T) {
	fns := scanSource("fake.go", srcNonMatching)
	assert.Empty(t, fns)
}

func TestScanSource_MultipleOnlyUnmigrated(t *testing.T) {
	fns := scanSource("fake.go", srcMultiple)
	require.Len(t, fns, 2)
	names := []string{fns[0].name, fns[1].name}
	assert.Contains(t, names, "enumerationStmtToMDL")
	assert.Contains(t, names, "microflowStmtToMDL")
}

func TestScanSource_LineRangesPopulated(t *testing.T) {
	fns := scanSource("fake.go", srcUnmigrated)
	require.Len(t, fns, 1)
	assert.Greater(t, fns[0].start, 0)
	assert.GreaterOrEqual(t, fns[0].end, fns[0].start)
}

// --- crossMatch tests ---

func TestCrossMatch_TouchedFunction(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "associationStmtToMDL", start: 93, end: 130,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{100, 105}},
	}
	v := crossMatch(fns, changed)
	require.Len(t, v, 1)
	assert.Equal(t, "associationStmtToMDL", v[0].name)
	assert.Equal(t, "modified", v[0].reason)
}

func TestCrossMatch_UntouchedFunction(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "associationStmtToMDL", start: 93, end: 130,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{200, 210}},
	}
	v := crossMatch(fns, changed)
	assert.Empty(t, v)
}

func TestCrossMatch_DifferentFile(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "associationStmtToMDL", start: 93, end: 130,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_entities_gen.go": {{100, 110}},
	}
	v := crossMatch(fns, changed)
	assert.Empty(t, v)
}

func TestCrossMatch_BoundaryStart(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "microflowStmtToMDL", start: 50, end: 80,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{50, 50}},
	}
	v := crossMatch(fns, changed)
	require.Len(t, v, 1)
}

func TestCrossMatch_BoundaryEnd(t *testing.T) {
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "microflowStmtToMDL", start: 50, end: 80,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{80, 85}},
	}
	v := crossMatch(fns, changed)
	require.Len(t, v, 1)
}

func TestCrossMatch_NoDoubleCounting(t *testing.T) {
	// Two overlapping ranges both touching the same function → one violation only
	fns := []unmgrFunc{{
		file: "mdl/executor/cmd_diff_mdl.go",
		name: "associationStmtToMDL", start: 93, end: 130,
	}}
	changed := map[string][]lineRange{
		"mdl/executor/cmd_diff_mdl.go": {{100, 105}, {110, 115}},
	}
	v := crossMatch(fns, changed)
	require.Len(t, v, 1)
}

func TestScanSource_InvalidSyntaxReturnsNil(t *testing.T) {
	// parse error on completely invalid source → returns nil, no panic
	fns := scanSource("fake.go", "this is not Go code {{{")
	assert.Nil(t, fns)
}
