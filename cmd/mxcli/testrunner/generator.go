// SPDX-License-Identifier: Apache-2.0

package testrunner

import (
	"fmt"
	"regexp"
	"strings"
)

// GenerateTestRunner generates the MDL for a TestRunner microflow from parsed test cases.
// The generated microflow:
// - Wraps each test in LOG+CALL+assertion pattern
// - Outputs structured MXTEST: log lines for result parsing
// - Returns Boolean (required by Mendix for after-startup)
func GenerateTestRunner(suite *TestSuite) string {
	var b strings.Builder

	// Module creation (idempotent — only if MxTest module doesn't exist)
	b.WriteString("CREATE MODULE MxTest;\n\n")

	b.WriteString("CREATE OR REPLACE MICROFLOW MxTest.TestRunner ()\n")
	b.WriteString("RETURNS Boolean AS $AllPassed\n")
	b.WriteString("{\n")
	b.WriteString("  DECLARE $AllPassed Boolean = true;\n")
	b.WriteString("  DECLARE $TestFailed Boolean = false;\n")
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  LOG INFO NODE 'MXTEST' 'MXTEST:START:%s';\n", escapeMDLString(suite.Name)))
	b.WriteString("\n")

	for i, tc := range suite.Tests {
		writeTestBlock(&b, tc, i)
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("  LOG INFO NODE 'MXTEST' 'MXTEST:END:%s';\n", escapeMDLString(suite.Name)))
	b.WriteString("  RETURN $AllPassed;\n")
	b.WriteString("}\n")
	b.WriteString("/\n")

	return b.String()
}

// writeTestBlock writes the MDL for a single test case within the TestRunner.
func writeTestBlock(b *strings.Builder, tc TestCase, index int) {
	suffix := fmt.Sprintf("_%d", index+1)

	b.WriteString(fmt.Sprintf("  -- Test %d: %s\n", index+1, tc.Name))
	b.WriteString(fmt.Sprintf("  LOG INFO NODE 'MXTEST' 'MXTEST:RUN:%s:%s';\n",
		escapeMDLString(tc.ID), escapeMDLString(tc.Name)))
	b.WriteString("  SET $TestFailed = false;\n")

	if tc.Throws != "" {
		writeThrowsTestBlock(b, tc, suffix)
		return
	}

	// Collect variable names from the MDL body and rename them with suffix
	varNames := extractVariableNames(tc.MDL)
	renamedMDL := renameVariables(tc.MDL, varNames, suffix)

	// Rewrite the MDL body: indent it and wrap CALL MICROFLOW with error handling
	lines := strings.Split(renamedMDL, "\n")
	rewritten := rewriteWithErrorHandling(lines, tc.ID)
	for _, line := range rewritten {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Generate assertion checks for @expect (with renamed variables)
	// Use flat IF blocks (no nesting) to avoid Mendix end-event issues
	if len(tc.Expects) > 0 {
		for _, exp := range tc.Expects {
			renamedExp := renameExpect(exp, varNames, suffix)
			writeExpectAssertion(b, tc.ID, renamedExp)
		}
	} else if tc.Throws == "" {
		// No expectations — just check it didn't throw
		b.WriteString("  IF $TestFailed = false THEN\n")
		b.WriteString(fmt.Sprintf("    LOG INFO NODE 'MXTEST' 'MXTEST:PASS:%s';\n", escapeMDLString(tc.ID)))
		b.WriteString("  END IF;\n")
	}
}

// writeThrowsTestBlock generates code for a test that expects an error.
func writeThrowsTestBlock(b *strings.Builder, tc TestCase, suffix string) {
	didThrowVar := "$DidThrow" + suffix
	b.WriteString(fmt.Sprintf("  DECLARE %s Boolean = false;\n", didThrowVar))

	// Rename variables in the body
	varNames := extractVariableNames(tc.MDL)
	renamedMDL := renameVariables(tc.MDL, varNames, suffix)

	lines := strings.Split(renamedMDL, "\n")
	rewritten := rewriteForThrowsTest(lines, tc.ID, didThrowVar)
	for _, line := range rewritten {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Check that it did throw
	b.WriteString(fmt.Sprintf("  IF %s THEN\n", didThrowVar))
	b.WriteString(fmt.Sprintf("    LOG INFO NODE 'MXTEST' 'MXTEST:PASS:%s';\n", escapeMDLString(tc.ID)))
	b.WriteString("  ELSE\n")
	b.WriteString(fmt.Sprintf("    LOG ERROR NODE 'MXTEST' 'MXTEST:FAIL:%s:Expected exception but none was thrown';\n",
		escapeMDLString(tc.ID)))
	b.WriteString("    SET $AllPassed = false;\n")
	b.WriteString("  END IF;\n")
}

// writeExpectAssertion generates an IF/ELSE check for a single @expect assertion.
// Uses compound condition with AND to guard against checking after exception.
// Only uses = operator (not <>) since <> causes Mendix expression errors.
func writeExpectAssertion(b *strings.Builder, testID string, exp Expect) {
	varRef := exp.Variable
	value := exp.Value

	var passCondition string
	if exp.Operator == "=" {
		passCondition = fmt.Sprintf("$TestFailed = false and %s = %s", varRef, value)
	} else {
		// For != assertions, invert: pass when values differ
		passCondition = fmt.Sprintf("$TestFailed = false and %s = %s", varRef, value)
		// Actually this needs to FAIL when equal — swap PASS/FAIL below
	}

	if exp.Operator == "=" {
		b.WriteString(fmt.Sprintf("  IF %s THEN\n", passCondition))
		b.WriteString(fmt.Sprintf("    LOG INFO NODE 'MXTEST' 'MXTEST:PASS:%s';\n", escapeMDLString(testID)))
		b.WriteString("  ELSE\n")
		failMsg := fmt.Sprintf("Expected %s %s %s", varRef, exp.Operator, value)
		b.WriteString(fmt.Sprintf("    LOG ERROR NODE 'MXTEST' 'MXTEST:FAIL:%s:%s';\n",
			escapeMDLString(testID), escapeMDLString(failMsg)))
		b.WriteString("    SET $AllPassed = false;\n")
		b.WriteString("  END IF;\n")
	} else {
		// != operator: pass when NOT equal, fail when equal
		condition := fmt.Sprintf("$TestFailed = false and %s = %s", varRef, value)
		b.WriteString(fmt.Sprintf("  IF %s THEN\n", condition))
		failMsg := fmt.Sprintf("Expected %s %s %s", varRef, exp.Operator, value)
		b.WriteString(fmt.Sprintf("    LOG ERROR NODE 'MXTEST' 'MXTEST:FAIL:%s:%s';\n",
			escapeMDLString(testID), escapeMDLString(failMsg)))
		b.WriteString("    SET $AllPassed = false;\n")
		b.WriteString("  ELSE\n")
		b.WriteString(fmt.Sprintf("    LOG INFO NODE 'MXTEST' 'MXTEST:PASS:%s';\n", escapeMDLString(testID)))
		b.WriteString("  END IF;\n")
	}
}

// varPattern matches $VariableName in MDL ($ followed by word characters).
var varPattern = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)

// assignPattern matches "$var = CALL" or "$var = CREATE" at the start of any
// statement line inside the test body.
var assignPattern = regexp.MustCompile(`(?m)^\s*\$([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?:CALL|CREATE)`)

// extractVariableNames finds all user-defined variable names in the MDL body.
// Returns the set of variable names (without $ prefix).
func extractVariableNames(mdl string) map[string]bool {
	names := make(map[string]bool)

	// Find assignment targets: $var = CALL MICROFLOW ... or $var = CREATE ...
	for _, m := range assignPattern.FindAllStringSubmatch(mdl, -1) {
		names[m[1]] = true
	}

	return names
}

// renameVariables replaces all occurrences of $varName with $varName_suffix
// for each variable in the names set.
func renameVariables(mdl string, names map[string]bool, suffix string) string {
	if len(names) == 0 {
		return mdl
	}

	return varPattern.ReplaceAllStringFunc(mdl, func(match string) string {
		// match is like "$result" — extract the name part
		name := match[1:] // strip $
		if names[name] {
			return "$" + name + suffix
		}
		return match
	})
}

// renameExpect applies variable renaming to an Expect assertion.
func renameExpect(exp Expect, names map[string]bool, suffix string) Expect {
	renamed := exp

	// Rename the variable reference (e.g., "$result" -> "$result_1" or "$product/Name" -> "$product_1/Name")
	renamed.Variable = varPattern.ReplaceAllStringFunc(exp.Variable, func(match string) string {
		name := match[1:]
		if names[name] {
			return "$" + name + suffix
		}
		return match
	})

	return renamed
}

// rewriteWithErrorHandling adds ON ERROR clauses to CALL MICROFLOW statements.
// The error handler ends with RETURN $AllPassed to properly terminate the flow path
// with a return value (required when the microflow returns a type).
func rewriteWithErrorHandling(lines []string, testID string) []string {
	var result []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if containsCallMicroflow(trimmed) && strings.HasSuffix(trimmed, ";") {
			withoutSemicolon := strings.TrimSuffix(trimmed, ";")
			result = append(result, withoutSemicolon+" ON ERROR {")
			result = append(result, fmt.Sprintf(
				"  LOG ERROR NODE 'MXTEST' 'MXTEST:FAIL:%s:Exception during execution';",
				escapeMDLString(testID)))
			result = append(result, "  SET $TestFailed = true;")
			result = append(result, "  SET $AllPassed = false;")
			result = append(result, "  RETURN $AllPassed;")
			result = append(result, "};")
		} else if containsCallMicroflow(trimmed) && !strings.HasSuffix(trimmed, ";") {
			var callLines []string
			callLines = append(callLines, line)
			for i+1 < len(lines) {
				i++
				nextLine := lines[i]
				callLines = append(callLines, nextLine)
				if strings.HasSuffix(strings.TrimSpace(nextLine), ";") {
					break
				}
			}
			joined := strings.Join(callLines, "\n")
			joined = strings.TrimSpace(joined)
			joined = strings.TrimSuffix(joined, ";")
			result = append(result, joined+" ON ERROR {")
			result = append(result, fmt.Sprintf(
				"  LOG ERROR NODE 'MXTEST' 'MXTEST:FAIL:%s:Exception during execution';",
				escapeMDLString(testID)))
			result = append(result, "  SET $TestFailed = true;")
			result = append(result, "  SET $AllPassed = false;")
			result = append(result, "  RETURN $AllPassed;")
			result = append(result, "};")
		} else {
			result = append(result, line)
		}
	}

	return result
}

// rewriteForThrowsTest wraps CALL MICROFLOW with ON ERROR that sets the throw flag.
func rewriteForThrowsTest(lines []string, testID string, didThrowVar string) []string {
	var result []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if containsCallMicroflow(trimmed) && strings.HasSuffix(trimmed, ";") {
			withoutSemicolon := strings.TrimSuffix(trimmed, ";")
			result = append(result, withoutSemicolon+" ON ERROR {")
			result = append(result, fmt.Sprintf("  SET %s = true;", didThrowVar))
			result = append(result, "};")
		} else if containsCallMicroflow(trimmed) && !strings.HasSuffix(trimmed, ";") {
			var callLines []string
			callLines = append(callLines, line)
			for i+1 < len(lines) {
				i++
				nextLine := lines[i]
				callLines = append(callLines, nextLine)
				if strings.HasSuffix(strings.TrimSpace(nextLine), ";") {
					break
				}
			}
			joined := strings.Join(callLines, "\n")
			joined = strings.TrimSpace(joined)
			joined = strings.TrimSuffix(joined, ";")
			result = append(result, joined+" ON ERROR {")
			result = append(result, fmt.Sprintf("  SET %s = true;", didThrowVar))
			result = append(result, "};")
		} else {
			result = append(result, line)
		}
	}

	return result
}

// containsCallMicroflow checks if a line starts a CALL MICROFLOW statement.
func containsCallMicroflow(s string) bool {
	upper := strings.ToUpper(s)
	return strings.Contains(upper, "CALL MICROFLOW") ||
		strings.Contains(upper, "CALL NANOFLOW")
}

// escapeMDLString escapes single quotes for MDL string literals.
func escapeMDLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
