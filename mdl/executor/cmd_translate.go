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

	switch strings.ToUpper(stmt.DocType) {
	case "PAGE":
		return translatePage(ctx, stmt, "page")
	case "SNIPPET":
		return translatePage(ctx, stmt, "snippet")
	case "ENUMERATION":
		return mdlerrors.NewUnsupported("TRANSLATE ENUMERATION not yet implemented")
	case "WORKFLOW":
		return mdlerrors.NewUnsupported("TRANSLATE WORKFLOW not yet implemented")
	case "MICROFLOW":
		return mdlerrors.NewUnsupported("TRANSLATE MICROFLOW not yet implemented")
	default:
		return mdlerrors.NewUnsupported(fmt.Sprintf("TRANSLATE %s not supported", stmt.DocType))
	}
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

	mutator, err := ctx.Backend.OpenPageForMutation(unitID)
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
