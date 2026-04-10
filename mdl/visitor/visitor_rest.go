// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// ============================================================================
// REST Client CREATE Statements
// ============================================================================

// ExitCreateRestClientStatement handles CREATE REST CLIENT Module.Name BASE URL '...' AUTHENTICATION ... BEGIN ... END.
func (b *Builder) ExitCreateRestClientStatement(ctx *parser.CreateRestClientStatementContext) {
	stmt := &ast.CreateRestClientStmt{
		Name: buildQualifiedName(ctx.QualifiedName()),
	}

	// Parse BASE URL
	if baseUrlCtx := ctx.RestClientBaseUrl(); baseUrlCtx != nil {
		bc := baseUrlCtx.(*parser.RestClientBaseUrlContext)
		if sl := bc.STRING_LITERAL(); sl != nil {
			stmt.BaseUrl = unquoteString(sl.GetText())
		}
	}

	// Parse AUTHENTICATION
	if authCtx := ctx.RestClientAuthentication(); authCtx != nil {
		ac := authCtx.(*parser.RestClientAuthenticationContext)
		if ac.BASIC() != nil {
			authDef := &ast.RestAuthDef{Scheme: "BASIC"}
			authValues := ac.AllRestAuthValue()
			if len(authValues) >= 1 {
				authDef.Username = restAuthValueText(authValues[0])
			}
			if len(authValues) >= 2 {
				authDef.Password = restAuthValueText(authValues[1])
			}
			stmt.Authentication = authDef
		}
		// NONE: leave Authentication nil
	}

	// Parse operations
	for _, opCtx := range ctx.AllRestOperationDef() {
		oc := opCtx.(*parser.RestOperationDefContext)
		opDef := parseRestOperationDef(oc)
		stmt.Operations = append(stmt.Operations, opDef)
	}

	// Check for CREATE OR MODIFY
	createStmt := findParentCreateStatement(ctx)
	if createStmt != nil {
		if createStmt.OR() != nil && (createStmt.MODIFY() != nil || createStmt.REPLACE() != nil) {
			stmt.CreateOrModify = true
		}
	}
	stmt.Documentation = findDocCommentText(ctx)

	b.statements = append(b.statements, stmt)
}

// parseRestOperationDef converts a RestOperationDefContext into an ast.RestOperationDef.
func parseRestOperationDef(ctx *parser.RestOperationDefContext) *ast.RestOperationDef {
	op := &ast.RestOperationDef{}

	// Operation name: identifierOrKeyword or STRING_LITERAL
	if iok := ctx.IdentifierOrKeyword(); iok != nil {
		op.Name = identifierOrKeywordText(iok)
	} else if sl := ctx.AllSTRING_LITERAL(); len(sl) > 0 {
		// First STRING_LITERAL is the operation name (single-quoted)
		op.Name = unquoteString(sl[0].GetText())
	}

	// Method
	if methodCtx := ctx.RestHttpMethod(); methodCtx != nil {
		op.Method = strings.ToUpper(methodCtx.GetText())
	}

	// PATH — find the STRING_LITERAL after the PATH keyword
	// If identifierOrKeyword is set, first STRING_LITERAL is PATH
	// If identifierOrKeyword is nil, first is name, second is PATH
	allStrLits := ctx.AllSTRING_LITERAL()
	if ctx.IdentifierOrKeyword() != nil {
		// Name is identifierOrKeyword, first STRING_LITERAL is PATH
		if len(allStrLits) >= 1 {
			op.Path = unquoteString(allStrLits[0].GetText())
		}
	} else {
		// Name is STRING_LITERAL(0), PATH is STRING_LITERAL(1)
		if len(allStrLits) >= 2 {
			op.Path = unquoteString(allStrLits[1].GetText())
		}
	}

	// Parse operation clauses
	for _, clauseCtx := range ctx.AllRestOperationClause() {
		cc := clauseCtx.(*parser.RestOperationClauseContext)
		parseRestOperationClause(cc, op)
	}

	// Parse RESPONSE spec
	if respCtx := ctx.RestResponseSpec(); respCtx != nil {
		rc := respCtx.(*parser.RestResponseSpecContext)
		parseRestResponseSpec(rc, op)
	}

	// Documentation (from doc comment on the operation)
	if docCtx := ctx.DocComment(); docCtx != nil {
		op.Documentation = extractDocCommentText(docCtx)
	}

	return op
}

// parseRestOperationClause handles individual clauses within an operation.
func parseRestOperationClause(ctx *parser.RestOperationClauseContext, op *ast.RestOperationDef) {
	if ctx.PARAMETER() != nil {
		// PARAMETER $name: Type
		param := ast.RestParamDef{}
		if v := ctx.VARIABLE(); v != nil {
			param.Name = v.GetText()
		}
		if dt := ctx.DataType(); dt != nil {
			param.DataType = buildDataType(dt).Kind.String()
		}
		op.Parameters = append(op.Parameters, param)
	} else if ctx.QUERY() != nil {
		// QUERY $name: Type
		param := ast.RestParamDef{}
		if v := ctx.VARIABLE(); v != nil {
			param.Name = v.GetText()
		}
		if dt := ctx.DataType(); dt != nil {
			param.DataType = buildDataType(dt).Kind.String()
		}
		op.QueryParameters = append(op.QueryParameters, param)
	} else if ctx.HEADER() != nil {
		// HEADER 'name' = value
		header := ast.RestHeaderDef{}
		if sl := ctx.STRING_LITERAL(); sl != nil {
			header.Name = unquoteString(sl.GetText())
		}
		if hvCtx := ctx.RestHeaderValue(); hvCtx != nil {
			hv := hvCtx.(*parser.RestHeaderValueContext)
			if hv.PLUS() != nil {
				// 'prefix' + $Variable
				if sl := hv.STRING_LITERAL(); sl != nil {
					header.Prefix = unquoteString(sl.GetText())
				}
				if v := hv.VARIABLE(); v != nil {
					header.Variable = v.GetText()
				}
			} else if hv.VARIABLE() != nil {
				// $Variable only
				header.Variable = hv.VARIABLE().GetText()
			} else if hv.STRING_LITERAL() != nil {
				// static literal
				header.Value = unquoteString(hv.STRING_LITERAL().GetText())
			}
		}
		op.Headers = append(op.Headers, header)
	} else if ctx.BODY() != nil {
		// BODY JSON FROM $var or BODY FILE FROM $var
		if ctx.JSON() != nil {
			op.BodyType = "JSON"
		} else if ctx.FILE_KW() != nil {
			op.BodyType = "FILE"
		}
		if v := ctx.VARIABLE(); v != nil {
			op.BodyVariable = v.GetText()
		}
	} else if ctx.TIMEOUT() != nil {
		// TIMEOUT number
		if nl := ctx.NUMBER_LITERAL(); nl != nil {
			if val, err := strconv.Atoi(nl.GetText()); err == nil {
				op.Timeout = val
			}
		}
	}
}

// parseRestResponseSpec handles the RESPONSE clause of an operation.
func parseRestResponseSpec(ctx *parser.RestResponseSpecContext, op *ast.RestOperationDef) {
	if ctx.NONE() != nil {
		op.ResponseType = "NONE"
	} else if ctx.JSON() != nil {
		op.ResponseType = "JSON"
		if v := ctx.VARIABLE(); v != nil {
			op.ResponseVariable = v.GetText()
		}
	} else if ctx.STRING_TYPE() != nil {
		op.ResponseType = "STRING"
		if v := ctx.VARIABLE(); v != nil {
			op.ResponseVariable = v.GetText()
		}
	} else if ctx.FILE_KW() != nil {
		op.ResponseType = "FILE"
		if v := ctx.VARIABLE(); v != nil {
			op.ResponseVariable = v.GetText()
		}
	} else if ctx.STATUS() != nil {
		op.ResponseType = "STATUS"
		if v := ctx.VARIABLE(); v != nil {
			op.ResponseVariable = v.GetText()
		}
	}
}

// restAuthValueText extracts the text from a RestAuthValue context.
func restAuthValueText(ctx parser.IRestAuthValueContext) string {
	if ctx == nil {
		return ""
	}
	ac := ctx.(*parser.RestAuthValueContext)
	if sl := ac.STRING_LITERAL(); sl != nil {
		return unquoteString(sl.GetText())
	}
	if v := ac.VARIABLE(); v != nil {
		return v.GetText()
	}
	return ""
}

// extractDocCommentText extracts documentation text from a DocComment context.
func extractDocCommentText(ctx parser.IDocCommentContext) string {
	if ctx == nil {
		return ""
	}
	text := ctx.GetText()
	// Remove /** and */ markers
	text = strings.TrimPrefix(text, "/**")
	text = strings.TrimSuffix(text, "*/")

	// Process lines: trim whitespace and leading *
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// ExitCreatePublishedRestServiceStatement handles CREATE PUBLISHED REST SERVICE.
func (b *Builder) ExitCreatePublishedRestServiceStatement(ctx *parser.CreatePublishedRestServiceStatementContext) {
	stmt := &ast.CreatePublishedRestServiceStmt{
		Name: buildQualifiedName(ctx.QualifiedName()),
	}

	// Check for CREATE OR REPLACE
	createStmt := findParentCreateStatement(ctx)
	if createStmt != nil {
		if createStmt.OR() != nil && (createStmt.REPLACE() != nil || createStmt.MODIFY() != nil) {
			stmt.CreateOrReplace = true
		}
	}

	// Parse properties (Path, Version, ServiceName)
	for _, propCtx := range ctx.AllPublishedRestProperty() {
		pc := propCtx.(*parser.PublishedRestPropertyContext)
		key := identifierOrKeywordText(pc.IdentifierOrKeyword().(*parser.IdentifierOrKeywordContext))
		val := unquoteString(pc.STRING_LITERAL().GetText())
		switch strings.ToLower(key) {
		case "path":
			stmt.Path = val
		case "version":
			stmt.Version = val
		case "servicename":
			stmt.ServiceName = val
		case "folder":
			stmt.Folder = val
		}
	}

	// Parse resources
	for _, resCtx := range ctx.AllPublishedRestResource() {
		rc := resCtx.(*parser.PublishedRestResourceContext)
		resDef := &ast.PublishedRestResourceDef{
			Name: unquoteString(rc.STRING_LITERAL().GetText()),
		}

		// Parse operations
		for _, opCtx := range rc.AllPublishedRestOperation() {
			oc := opCtx.(*parser.PublishedRestOperationContext)
			opDef := &ast.PublishedRestOperationDef{}

			// HTTP method
			if mCtx := oc.RestHttpMethod(); mCtx != nil {
				opDef.HTTPMethod = strings.ToUpper(mCtx.GetText())
			}

			// Operation path — strip leading/trailing slashes (CE6550/CE6551)
			if pCtx := oc.PublishedRestOpPath(); pCtx != nil {
				pc := pCtx.(*parser.PublishedRestOpPathContext)
				if pc.STRING_LITERAL() != nil {
					opDef.Path = strings.Trim(unquoteString(pc.STRING_LITERAL().GetText()), "/")
				}
			}

			// Microflow reference
			allQN := oc.AllQualifiedName()
			if len(allQN) >= 1 {
				opDef.Microflow = buildQualifiedName(allQN[0])
			}

			// Optional modifiers
			if oc.DEPRECATED() != nil {
				opDef.Deprecated = true
			}

			// Import/Export mapping (qualifiedName after IMPORT/EXPORT MAPPING)
			if oc.IMPORT() != nil && len(allQN) >= 2 {
				opDef.ImportMapping = allQN[1].GetText()
			}
			if oc.EXPORT() != nil {
				idx := 1
				if oc.IMPORT() != nil {
					idx = 2
				}
				if len(allQN) > idx {
					opDef.ExportMapping = allQN[idx].GetText()
				}
			}

			// Commit
			if oc.COMMIT() != nil {
				if idCtx := oc.IdentifierOrKeyword(); idCtx != nil {
					opDef.Commit = identifierOrKeywordText(idCtx.(*parser.IdentifierOrKeywordContext))
				}
			}

			resDef.Operations = append(resDef.Operations, opDef)
		}

		stmt.Resources = append(stmt.Resources, resDef)
	}

	b.statements = append(b.statements, stmt)
}
