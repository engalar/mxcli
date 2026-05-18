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
