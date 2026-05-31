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
