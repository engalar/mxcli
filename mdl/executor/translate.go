// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// translateDocument dispatches a TRANSLATE statement to the doc-type handler.
func translateDocument(ctx *ExecContext, stmt *ast.TranslateStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if stmt.Lang == "" {
		return mdlerrors.NewValidation("TRANSLATE requires a target language (IN langCode)")
	}

	if err := requireRegisteredLanguage(ctx, stmt.Lang); err != nil {
		return err
	}

	switch strings.ToUpper(stmt.DocType) {
	case "PAGE":
		return translatePage(ctx, stmt, "page")
	case "SNIPPET":
		return translatePage(ctx, stmt, "snippet")
	case "ENUMERATION":
		return translateEnumeration(ctx, stmt)
	case "WORKFLOW":
		return mdlerrors.NewUnsupported(
			"TRANSLATE WORKFLOW is not supported: workflow activity text (TaskName/TaskDescription) " +
				"uses Microflows$StringTemplate in Mendix 11.x, which does not support multilingual translation. " +
				"Use page or enumeration translation instead.")
	case "NAVIGATION":
		return translateNavigation(ctx, stmt)
	case "MICROFLOW":
		return mdlerrors.NewUnsupported("TRANSLATE MICROFLOW not yet implemented")
	default:
		return mdlerrors.NewUnsupported(fmt.Sprintf("TRANSLATE %s not supported", stmt.DocType))
	}
}

// requireRegisteredLanguage returns an actionable error if lang is not in the
// project's registered languages. This guides the AI/user to add the language
// before translating, rather than silently writing a translation for a language
// Studio Pro will not display.
func requireRegisteredLanguage(ctx *ExecContext, lang string) error {
	ps, err := ctx.SettingsReader.GetProjectSettings()
	if err != nil {
		return mdlerrors.NewBackend("read project settings", err)
	}
	if ps != nil && ps.Language != nil {
		for _, l := range ps.Language.Languages {
			if l.Code == lang {
				return nil
			}
		}
	}
	return mdlerrors.NewValidationf(
		"language '%s' is not registered in this project. Run ALTER SETTINGS LANGUAGE ADD '%s' first.",
		lang, lang)
}

// translatePage applies translation SET operations to a page or snippet.
func translatePage(ctx *ExecContext, stmt *ast.TranslateStmt, containerType string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	var unitID model.ID
	if containerType == "snippet" {
		unitID, err = findSnippetIDGen(ctx, stmt.QName, h)
	} else {
		unitID, err = findPageIDGen(ctx, stmt.QName, h)
	}
	if err != nil {
		return err
	}

	mutator, err := ctx.PageMutationOperator.OpenPageForMutation(unitID)
	if err != nil {
		return mdlerrors.NewBackend("open "+containerType+" for mutation", err)
	}

	for _, op := range stmt.Ops {
		if err := applyTranslateOp(mutator, op, stmt.Lang); err != nil {
			return mdlerrors.NewBackend("translate", err)
		}
	}

	if err := mutator.Save(); err != nil {
		return mdlerrors.NewBackend("save translated "+containerType, err)
	}

	fmt.Fprintf(ctx.Output, "Translated %s %s in %s (%d fields)\n",
		containerType, stmt.QName.String(), stmt.Lang, len(stmt.Ops))
	return nil
}

// translateEnumeration applies translation SET operations to an enumeration.
// Each op path is "ValueName.caption"; the text is the caption translation for
// stmt.Lang.
func translateEnumeration(ctx *ExecContext, stmt *ast.TranslateStmt) error {
	for _, op := range stmt.Ops {
		valueName, prop, hasDot := strings.Cut(op.Path, ".")
		if !hasDot || !strings.EqualFold(prop, "caption") {
			return mdlerrors.NewValidationf(
				"invalid enumeration path %q: expected ValueName.caption", op.Path)
		}
		if err := ctx.SettingsWriter.SetEnumerationTranslation(
			stmt.QName.String(), valueName, stmt.Lang, op.Text); err != nil {
			return mdlerrors.NewBackend("translate enumeration value "+valueName, err)
		}
	}
	fmt.Fprintf(ctx.Output, "Translated enumeration %s in %s (%d values)\n",
		stmt.QName.String(), stmt.Lang, len(stmt.Ops))
	return nil
}

// translateMicroflowStmt applies TRANSLATE MICROFLOW operations. Microflow
// activities are unnamed, so each op addresses an action by its type and the
// 0-based ordinal index among same-typed actions (Type-Index addressing).
func translateMicroflowStmt(ctx *ExecContext, stmt *ast.TranslateMicroflowStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	if stmt.Lang == "" {
		return mdlerrors.NewValidation("TRANSLATE requires a target language (IN langCode)")
	}
	if err := requireRegisteredLanguage(ctx, stmt.Lang); err != nil {
		return err
	}

	docQN := stmt.QName.String()
	for _, op := range stmt.Ops {
		if err := ctx.SettingsWriter.SetMicroflowActionTranslation(
			docQN, op.ActionType, op.Index, op.Property, stmt.Lang, op.Text); err != nil {
			return mdlerrors.NewBackend(
				fmt.Sprintf("translate %s[%d].%s", op.ActionType, op.Index, op.Property), err)
		}
	}
	fmt.Fprintf(ctx.Output, "TRANSLATED MICROFLOW %s IN %s (%d actions)\n",
		docQN, stmt.Lang, len(stmt.Ops))
	return nil
}

// applyTranslateOp routes a single SET path = text op to the right mutator call.
// A path of "title" (no dot) targets the page/snippet title; "Widget.property"
// targets a translatable text property of a widget.
func applyTranslateOp(mutator backend.PageMutator, op ast.TranslateSetOp, langCode string) error {
	widget, prop, hasDot := strings.Cut(op.Path, ".")
	if !hasDot {
		// Bare path — page-level title.
		if strings.EqualFold(widget, "title") {
			return mutator.SetPageTitleTranslation(langCode, op.Text)
		}
		return fmt.Errorf("unsupported translate path %q (expected Widget.property or title)", op.Path)
	}
	return mutator.SetWidgetTranslation(widget, prop, langCode, op.Text)
}

// translateNavigation applies translation SET operations to navigation menu
// captions. The SET path is the hierarchy of STRING_LITERAL captions joined by
// dots, ending with ".caption" (the property name), e.g.:
//
//	translate navigation Responsive in zh_CN
//	  set 'My Tickets'.caption = '我的工单',
//	      'Ticket Management'.'All Tickets'.caption = '所有工单';
func translateNavigation(ctx *ExecContext, stmt *ast.TranslateStmt) error {
	profileName := stmt.QName.String()
	for _, op := range stmt.Ops {
		// Path format: "MenuHierarchy.caption". The last dot-separated segment
		// is the property ("caption"), everything before is the menu hierarchy.
		lastDot := strings.LastIndex(op.Path, ".")
		if lastDot < 0 {
			return mdlerrors.NewValidationf(
				"invalid navigation translate path %q: expected 'Menu.Caption' or 'Parent.Child.Caption'", op.Path)
		}
		property := op.Path[lastDot+1:]
		if !strings.EqualFold(property, "caption") {
			return mdlerrors.NewValidationf(
				"invalid navigation property %q: only 'caption' is supported", property)
		}
		menuPath := strings.Split(op.Path[:lastDot], ".")
		if err := ctx.SettingsWriter.SetNavigationCaptionTranslation(profileName, menuPath, stmt.Lang, op.Text); err != nil {
			return mdlerrors.NewBackend("translate navigation", err)
		}
	}
	fmt.Fprintf(ctx.Output, "Translated navigation %s in %s (%d fields)\n",
		profileName, stmt.Lang, len(stmt.Ops))
	return nil
}
