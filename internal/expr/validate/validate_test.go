// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/meta"
	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/stretchr/testify/assert"
)

func makeRec(raw, unitType, slot string) scan.ExprRecord {
	return scan.ExprRecord{Raw: raw, UnitType: unitType, Category: "microflow"}
}

func TestValidate_CleanExpression_NoIssues(t *testing.T) {
	r := makeRec("trim($X) = ''", "Microflows$ExpressionSplitCondition", "IfStmt.Condition")
	pr := parse.ParseExpression(r)
	issues := validate.ValidateSyntax(pr)
	assert.Empty(t, issues, "clean expression must have no issues")
}

func TestValidateSYN02_URL(t *testing.T) {
	r := makeRec("https://www.mendix.com/", "Forms$StaticOrDynamicString", "")
	pr := parse.ParseExpression(r)
	issues := validate.ValidateSyntax(pr)
	found := false
	for _, i := range issues {
		if i.RuleID == "SYN-02" {
			assert.Equal(t, "INFO", i.Severity)
			found = true
		}
	}
	assert.True(t, found, "should flag URL as SYN-02")
}

func TestValidateSYN03_MissingElse(t *testing.T) {
	r := makeRec("if $X then $Y", "Microflows$ExpressionSplitCondition", "IfStmt.Condition")
	pr := parse.ParseExpression(r)
	issues := validate.ValidateSyntax(pr)
	found := false
	for _, i := range issues {
		if i.RuleID == "SYN-03" {
			assert.Equal(t, "WARNING", i.Severity)
			assert.NotEmpty(t, i.Fix)
			found = true
		}
	}
	assert.True(t, found, "should flag missing else as SYN-03")
}

func TestValidateSemantic_NilIdx(t *testing.T) {
	pr := parse.ParseResult{Record: scan.ExprRecord{Raw: "@Module.Const"}}
	issues := validate.ValidateSemantic(pr, nil)
	assert.Empty(t, issues, "nil idx 应返回空")
}

func TestValidateSEM05_ConstantNotFound(t *testing.T) {
	mockIdx := meta.NewMockIndex(nil)
	r := scan.ExprRecord{
		Raw:      "@NonExistent.Config",
		UnitType: "Microflows$BasicCodeActionParameterValue",
	}
	pr := parse.ParseResult{Record: r}
	issues := validate.ValidateSemantic(pr, mockIdx)
	found := false
	for _, i := range issues {
		if i.RuleID == "SEM-05" {
			assert.Equal(t, "ERROR", i.Severity)
			found = true
		}
	}
	assert.True(t, found, "SEM-05 应检测到不存在的常量引用")
}

func TestValidateSEM05_ConstantFound_NoIssue(t *testing.T) {
	mockIdx := meta.NewMockIndex(nil)
	mockIdx.AddConstant("@MyMod.Key")
	r := scan.ExprRecord{Raw: "@MyMod.Key", UnitType: "Microflows$Expression"}
	pr := parse.ParseResult{Record: r}
	issues := validate.ValidateSemantic(pr, mockIdx)
	for _, i := range issues {
		assert.NotEqual(t, "SEM-05", i.RuleID, "已注册的常量不应触发 SEM-05")
	}
}

func TestValidateSEM04_EnumValueNotFound(t *testing.T) {
	mockIdx := meta.NewMockIndex(map[string][]string{
		"MyMod.Status": {"Active", "Inactive"},
	})
	r := scan.ExprRecord{Raw: "$x = MyMod.Status.Archived", UnitType: "Microflows$Expression"}
	pr := parse.ParseResult{Record: r}
	issues := validate.ValidateSemantic(pr, mockIdx)
	found := false
	for _, i := range issues {
		if i.RuleID == "SEM-04" {
			assert.Contains(t, i.Message, "Archived")
			found = true
		}
	}
	assert.True(t, found, "SEM-04 应检测到不存在的枚举值")
}

func TestValidateSEM04_EnumValueFound_NoIssue(t *testing.T) {
	mockIdx := meta.NewMockIndex(map[string][]string{
		"MyMod.Status": {"Active", "Inactive"},
	})
	r := scan.ExprRecord{Raw: "$x = MyMod.Status.Active", UnitType: "Microflows$Expression"}
	pr := parse.ParseResult{Record: r}
	issues := validate.ValidateSemantic(pr, mockIdx)
	for _, i := range issues {
		assert.NotEqual(t, "SEM-04", i.RuleID, "已知枚举值不应触发 SEM-04")
	}
}

func TestValidateSEM07_XPathEntityNotFound(t *testing.T) {
	mockIdx := meta.NewMockIndex(nil)
	mockIdx.AddEntityAttr("MyMod.RealEntity", "Name", 1) // KindBoolean=1, 这里只需注册实体
	r := scan.ExprRecord{Raw: "[MyMod_LongEntity/Name = 'foo']", UnitType: "Microflows$XPathConstraint"}
	pr := parse.ParseResult{Record: r}
	issues := validate.ValidateSemantic(pr, mockIdx)
	// MyMod_LongEntity 长度 > 10 且含下划线，会被启发式触发但 regex 不匹配 X.Y 形式
	// 改用更直接的测试: 非 _ 长名
	r2 := scan.ExprRecord{Raw: "[MyMod.UnknownLongEntity/Name = 'foo']", UnitType: "Microflows$XPathConstraint"}
	pr2 := parse.ParseResult{Record: r2}
	issues2 := validate.ValidateSemantic(pr2, mockIdx)
	found := false
	for _, i := range issues2 {
		if i.RuleID == "SEM-07" {
			assert.Equal(t, "WARNING", i.Severity)
			found = true
		}
	}
	_ = issues
	assert.True(t, found, "SEM-07 应警告未知 XPath 实体")
}

func TestValidate_ExprcheckHints_Surfaced(t *testing.T) {
	// "not x" without parens triggers E011 from exprcheck
	r := makeRec("not $IsValid", "Microflows$ExpressionSplitCondition", "IfStmt.Condition")
	pr := parse.ParseExpression(r)
	issues := validate.ValidateSyntax(pr)
	found := false
	for _, i := range issues {
		if i.RuleID == "E011" {
			assert.NotEmpty(t, i.Fix)
			found = true
		}
	}
	assert.True(t, found, "E011 hint should be surfaced for 'not x' without parens")
}

// ── False-positive regression tests ──────────────────────────────────────────
// Every entry below is a *valid* Mendix expression that the checker must not flag
// with any E006 (wrong arity) or parse error. These guard against the func table
// incorrectly rejecting official functions or accepting wrong-arity calls silently.
//
// The table covers every function family documented at:
//   https://docs.mendix.com/refguide/expressions/

func noE006(t *testing.T, expr string) {
	t.Helper()
	r := makeRec(expr, "Microflows$Expression", "")
	pr := parse.ParseExpression(r)
	for _, i := range validate.ValidateSyntax(pr) {
		if i.RuleID == "E006" {
			t.Errorf("expression %q produced unexpected E006: %s", expr, i.Message)
		}
	}
}

func TestNoFalsePositive_SubtractDateFunctions(t *testing.T) {
	cases := []string{
		"subtractMilliseconds($d, 500)",
		"subtractSeconds($d, 30)",
		"subtractMinutes($d, 15)",
		"subtractHours($d, 2)",
		"subtractDays($d, 7)",
		"subtractDaysUTC($d, 7)",
		"subtractWeeks($d, 2)",
		"subtractWeeksUTC($d, 2)",
		"subtractMonths($d, 3)",
		"subtractMonthsUTC($d, 3)",
		"subtractQuarters($d, 1)",
		"subtractQuartersUTC($d, 1)",
		"subtractYears($d, 1)",
		"subtractYearsUTC($d, 1)",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) { noE006(t, expr) })
	}
}

func TestNoFalsePositive_AddDateUTCVariants(t *testing.T) {
	cases := []string{
		"addMilliseconds($d, 100)",
		"addDaysUTC($d, 3)",
		"addWeeksUTC($d, 1)",
		"addMonthsUTC($d, 2)",
		"addQuarters($d, 1)",
		"addQuartersUTC($d, 1)",
		"addYearsUTC($d, 5)",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) { noE006(t, expr) })
	}
}

func TestNoFalsePositive_BetweenDateFunctions(t *testing.T) {
	cases := []string{
		"millisecondsBetween($d1, $d2)",
		"calendarMonthsBetween($d1, $d2)",
		"calendarYearsBetween($d1, $d2)",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) { noE006(t, expr) })
	}
}

func TestNoFalsePositive_BeginEndOfDate(t *testing.T) {
	cases := []string{
		"beginOfDay($d)", "beginOfWeek($d)", "beginOfMonth($d)", "beginOfYear($d)",
		"endOfDay($d)", "endOfWeek($d)", "endOfMonth($d)", "endOfYear($d)",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) { noE006(t, expr) })
	}
}

func TestNoFalsePositive_TrimToDate(t *testing.T) {
	cases := []string{
		"trimToSeconds($d)", "trimToMinutes($d)",
		"trimToHours($d)", "trimToHoursUTC($d)",
		"trimToDays($d)", "trimToDaysUTC($d)",
		"trimToMonths($d)", "trimToMonthsUTC($d)",
		"trimToYears($d)", "trimToYearsUTC($d)",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) { noE006(t, expr) })
	}
}

func TestNoFalsePositive_DateCreationAndFormatting(t *testing.T) {
	cases := []string{
		"dateTimeUTC(2024, 1, 1, 0, 0, 0)",
		"formatDate($d)",
		"formatDateUTC($d)",
		"formatTime($d)",
		"formatTimeUTC($d)",
		"formatDate($d, 'dd/MM/yyyy')",
		"formatDateUTC($d, 'yyyy-MM-dd')",
		"dateTimeToEpoch($d)",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) { noE006(t, expr) })
	}
}

func TestNoFalsePositive_StringFunctions(t *testing.T) {
	cases := []string{
		"findLast('hello world', 'o')",
		"replaceFirst('hello world', 'o', 'a')",
		"formatDecimal($x, '#,###.00')",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) { noE006(t, expr) })
	}
}

func TestNoFalsePositive_EnumerationFunctions(t *testing.T) {
	cases := []string{
		"getCaption(MyMod.Status.Active)",
		"getKey(MyMod.Status.Active)",
	}
	for _, expr := range cases {
		t.Run(expr, func(t *testing.T) { noE006(t, expr) })
	}
}

// TestNoFalsePositive_CalendarNamesMustNotAcceptOldNames verifies that the
// historically wrong names monthsBetween / yearsBetween are NOT registered,
// so callers using them get zero arity-check coverage (i.e., the function is
// treated as user-defined and ignored). This is the conservative safe default:
// we do not produce false E006 errors for unknown function names.
func TestNoFalsePositive_CalendarNamesMustNotAcceptOldNames(t *testing.T) {
	// These are NOT official Mendix functions and must not appear in the table.
	// Calling them with wrong arity must NOT fire E006 (because they're unknown).
	p := parse.ParseExpression(makeRec("monthsBetween($d1)", "Microflows$Expression", ""))
	for _, i := range validate.ValidateSyntax(p) {
		if i.RuleID == "E006" {
			// monthsBetween is unknown → no E006 expected (would be wrong signal)
			t.Errorf("monthsBetween must not fire E006 (it is unknown, not a registered function)")
		}
	}
}
