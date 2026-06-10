// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// ExitCreateImageCollectionStatement is called when exiting the createImageCollectionStatement production.
func (b *Builder) ExitCreateImageCollectionStatement(ctx *parser.CreateImageCollectionStatementContext) {
	stmt := &ast.CreateImageCollectionStmt{
		Name:        buildQualifiedName(ctx.QualifiedName()),
		ExportLevel: "Hidden",
	}

	// Extract /** ... */ doc comment (same as other create statements)
	stmt.Comment = findDocCommentText(ctx)

	if opts := ctx.ImageCollectionOptions(); opts != nil {
		optsCtx := opts.(*parser.ImageCollectionOptionsContext)
		for _, opt := range optsCtx.AllImageCollectionOption() {
			optCtx := opt.(*parser.ImageCollectionOptionContext)
			if optCtx.EXPORT() != nil && optCtx.LEVEL() != nil && optCtx.STRING_LITERAL() != nil {
				stmt.ExportLevel = unquoteString(optCtx.STRING_LITERAL().GetText())
			}
			if optCtx.COMMENT() != nil && optCtx.STRING_LITERAL() != nil {
				stmt.Comment = unquoteString(optCtx.STRING_LITERAL().GetText())
			}
		}
	}

	if body := ctx.ImageCollectionBody(); body != nil {
		bodyCtx := body.(*parser.ImageCollectionBodyContext)
		for _, item := range bodyCtx.AllImageCollectionItem() {
			itemCtx := item.(*parser.ImageCollectionItemContext)
			name := itemCtx.ImageName().GetText()
			// Strip quotes from quoted identifiers ("Name" or `Name`)
			if len(name) >= 2 && (name[0] == '"' || name[0] == '`') {
				name = name[1 : len(name)-1]
			}
			stmt.Images = append(stmt.Images, ast.ImageItem{
				Name:     name,
				FilePath: unquoteString(itemCtx.GetPath().GetText()),
			})
		}
	}

	createStmt := findParentCreateStatement(ctx)
	if createStmt != nil && createStmt.OR() != nil && (createStmt.REPLACE() != nil || createStmt.MODIFY() != nil) {
		stmt.CreateOrModify = true
	}

	b.statements = append(b.statements, stmt)
}

// exitAlterImageCollectionStatement handles ALTER IMAGE COLLECTION statements,
// dispatched from ExitAlterStatement when IMAGE and COLLECTION tokens are present.
func (b *Builder) exitAlterImageCollectionStatement(ctx *parser.AlterStatementContext) {
	qn := ctx.QualifiedName()
	if qn == nil {
		return
	}
	stmt := &ast.AlterImageCollectionStmt{
		Name: buildQualifiedName(qn),
	}
	for _, rawAction := range ctx.AllAlterImageCollectionAction() {
		action := rawAction.(*parser.AlterImageCollectionActionContext)
		names := action.AllImageName()
		switch {
		case action.ADD() != nil:
			stmt.Actions = append(stmt.Actions, &ast.AddImageAction{
				ImageName: imageNameText(names[0]),
				FilePath:  unquoteString(action.STRING_LITERAL().GetText()),
			})
		case action.DROP() != nil:
			stmt.Actions = append(stmt.Actions, &ast.DropImageAction{
				ImageName: imageNameText(names[0]),
			})
		case action.RENAME() != nil:
			stmt.Actions = append(stmt.Actions, &ast.RenameImageAction{
				From: imageNameText(names[0]),
				To:   imageNameText(names[1]),
			})
		case action.SET() != nil:
			stmt.Actions = append(stmt.Actions, &ast.SetImageAction{
				ImageName: imageNameText(names[0]),
				FilePath:  unquoteString(action.STRING_LITERAL().GetText()),
			})
		case action.MOVE() != nil:
			stmt.Actions = append(stmt.Actions, &ast.MoveImageCollectionAction{
				Target: buildQualifiedName(action.QualifiedName()),
			})
		case action.EXPORT() != nil:
			stmt.Actions = append(stmt.Actions, &ast.ExportImageAction{
				ImageName: imageNameText(names[0]),
				FilePath:  unquoteString(action.STRING_LITERAL().GetText()),
			})
		}
	}
	b.statements = append(b.statements, stmt)
}

// imageNameText returns the text of an imageName rule, stripping optional quotes.
func imageNameText(ctx parser.IImageNameContext) string {
	if ctx == nil {
		return ""
	}
	return unquoteIdentifier(ctx.GetText())
}
