// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

func TestDescribeTranslations_TextOutput_ShowsMissing(t *testing.T) {
	mb := describeTranslationsMockBackend()
	ctx, buf := newMockCtx(t, withBackend(mb))
	stmt := &ast.DescribeTranslationsStmt{
		QName: ast.QualifiedName{Module: "MyModule", Name: "Home"},
		Lang:  "zh_CN",
	}
	if err := describeTranslations(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Button_Submit") {
		t.Errorf("expected Button_Submit in output: %s", out)
	}
	if !strings.Contains(out, "(missing)") {
		t.Errorf("expected '(missing)' for untranslated field: %s", out)
	}
	if !strings.Contains(out, "translate page") {
		t.Errorf("expected translate template in output: %s", out)
	}
	if !strings.Contains(out, "'?'") {
		t.Errorf("expected '?' placeholder in template: %s", out)
	}
}

func TestDescribeTranslations_JSONOutput(t *testing.T) {
	mb := describeTranslationsMockBackend()
	ctx, buf := newMockCtx(t, withBackend(mb), withFormat(FormatJSON))
	stmt := &ast.DescribeTranslationsStmt{
		QName: ast.QualifiedName{Module: "MyModule", Name: "Home"},
		Lang:  "zh_CN",
	}
	if err := describeTranslations(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result struct {
		Document          string           `json:"document"`
		TargetLanguage    string           `json:"target_language"`
		Missing           []map[string]any `json:"missing"`
		Translated        []map[string]any `json:"translated"`
		TranslateTemplate string           `json:"translate_template"`
	}
	if err := json.Unmarshal([]byte(buf.String()), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if len(result.Missing) != 1 {
		t.Errorf("expected 1 missing field, got %d", len(result.Missing))
	}
	if result.Missing[0]["source"] == "" {
		t.Error("missing[].source must not be empty")
	}
	if len(result.Translated) != 1 {
		t.Errorf("expected 1 translated field, got %d", len(result.Translated))
	}
	if !strings.Contains(result.TranslateTemplate, "translate page") {
		t.Errorf("translate_template missing: %s", result.TranslateTemplate)
	}
	if !strings.Contains(result.TranslateTemplate, "'?'") {
		t.Errorf("translate_template must contain '?' placeholders: %s", result.TranslateTemplate)
	}
}

func describeTranslationsMockBackend() *mock.MockBackend {
	return &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					Languages: []model.Language{{Code: "en_US"}, {Code: "zh_CN"}},
				},
			}, nil
		},
		ListTranslationNodesFunc: func(docQN, docType string) ([]model.TranslationNode, error) {
			return []model.TranslationNode{
				{
					Path:     "Button_Submit.caption",
					Property: "caption",
					DocType:  "PAGE",
					Texts:    map[string]string{"en_US": "Submit", "zh_CN": "提交"},
				},
				{
					Path:     "TextBox_Email.placeholder",
					Property: "placeholder",
					DocType:  "PAGE",
					Texts:    map[string]string{"en_US": "Enter email"},
				},
			}, nil
		},
	}
}
