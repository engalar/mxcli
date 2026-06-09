// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/model"
)

func TestTranslateMicroflow_CallsSetMicroflowActionTranslation(t *testing.T) {
	var gotQN, gotActionType, gotProp, gotLang, gotText string
	var gotIndex int
	mb := &mock.MockBackend{
		IsConnectedFunc:        func() bool { return true },
		GetProjectSettingsFunc: translateTestSettings,
		SetMicroflowActionTranslationFunc: func(docQN, actionType string, index int, property, lang, text string) error {
			gotQN, gotActionType, gotIndex, gotProp, gotLang, gotText = docQN, actionType, index, property, lang, text
			return nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateMicroflowStmt{
		QName: ast.QualifiedName{Module: "MyModule", Name: "ACT_Save"},
		Lang:  "zh_CN",
		Ops: []ast.TranslateMicroflowSetOp{
			{ActionType: "ShowMessage", Index: 0, Property: "message", Text: "已保存"},
		},
	}
	if err := translateMicroflowStmt(ctx, stmt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQN != "MyModule.ACT_Save" {
		t.Errorf("docQN = %q", gotQN)
	}
	if gotActionType != "ShowMessage" || gotIndex != 0 || gotProp != "message" {
		t.Errorf("action addressing wrong: type=%q idx=%d prop=%q", gotActionType, gotIndex, gotProp)
	}
	if gotLang != "zh_CN" || gotText != "已保存" {
		t.Errorf("lang/text wrong: %q/%q", gotLang, gotText)
	}
	if !strings.Contains(buf.String(), "TRANSLATED MICROFLOW") {
		t.Errorf("expected confirmation output: %s", buf.String())
	}
}

func TestTranslateMicroflow_LangNotRegistered(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		GetProjectSettingsFunc: func() (*model.ProjectSettings, error) {
			return &model.ProjectSettings{
				Language: &model.LanguageSettings{
					Languages: []model.Language{{Code: "en_US"}},
				},
			}, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.TranslateMicroflowStmt{
		QName: ast.QualifiedName{Module: "MyModule", Name: "ACT_Save"},
		Lang:  "zh_CN",
		Ops:   []ast.TranslateMicroflowSetOp{{ActionType: "ShowMessage", Index: 0, Property: "message", Text: "x"}},
	}
	err := translateMicroflowStmt(ctx, stmt)
	if err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected 'not registered' error, got: %v", err)
	}
}
