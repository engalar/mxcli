// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// buildMicroflowBody converts microflow body context to MicroflowStatement slice.
func buildMicroflowBody(ctx parser.IMicroflowBodyContext) []ast.MicroflowStatement {
	if ctx == nil {
		return nil
	}
	bodyCtx := ctx.(*parser.MicroflowBodyContext)
	var stmts []ast.MicroflowStatement

	for _, stmtCtx := range bodyCtx.AllMicroflowStatement() {
		stmt := buildMicroflowStatement(stmtCtx)
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	return stmts
}

// buildMicroflowStatement converts a microflow statement context to an AST node.
func buildMicroflowStatement(ctx parser.IMicroflowStatementContext) ast.MicroflowStatement {
	if ctx == nil {
		return nil
	}
	mfCtx := ctx.(*parser.MicroflowStatementContext)

	// Extract annotations from the statement context
	ann := extractMicroflowAnnotations(mfCtx.AllAnnotation())

	var stmt ast.MicroflowStatement

	// Check each statement type
	if decl := mfCtx.DeclareStatement(); decl != nil {
		stmt = buildDeclareStatement(decl)
	} else if set := mfCtx.SetStatement(); set != nil {
		stmt = buildSetStatement(set)
	} else if createList := mfCtx.CreateListStatement(); createList != nil {
		// Check createListStatement before createObjectStatement to properly match "CREATE LIST OF"
		stmt = buildCreateListStatement(createList)
	} else if create := mfCtx.CreateObjectStatement(); create != nil {
		stmt = buildCreateObjectStatement(create)
	} else if change := mfCtx.ChangeObjectStatement(); change != nil {
		stmt = buildChangeObjectStatement(change)
	} else if commit := mfCtx.CommitStatement(); commit != nil {
		stmt = buildCommitStatement(commit)
	} else if del := mfCtx.DeleteObjectStatement(); del != nil {
		stmt = buildDeleteObjectStatement(del)
	} else if rollback := mfCtx.RollbackStatement(); rollback != nil {
		stmt = buildRollbackStatement(rollback)
	} else if retr := mfCtx.RetrieveStatement(); retr != nil {
		stmt = buildRetrieveStatement(retr)
	} else if ifStmt := mfCtx.IfStatement(); ifStmt != nil {
		stmt = buildIfStatement(ifStmt)
	} else if loop := mfCtx.LoopStatement(); loop != nil {
		stmt = buildLoopStatement(loop)
	} else if ws := mfCtx.WhileStatement(); ws != nil {
		stmt = buildWhileStatement(ws)
	} else if ret := mfCtx.ReturnStatement(); ret != nil {
		stmt = buildReturnStatement(ret)
	} else if mfCtx.RaiseErrorStatement() != nil {
		stmt = &ast.RaiseErrorStmt{}
	} else if log := mfCtx.LogStatement(); log != nil {
		stmt = buildLogStatement(log)
	} else if call := mfCtx.CallMicroflowStatement(); call != nil {
		stmt = buildCallMicroflowStatement(call)
	} else if call := mfCtx.CallJavaActionStatement(); call != nil {
		stmt = buildCallJavaActionStatement(call)
	} else if call := mfCtx.ExecuteDatabaseQueryStatement(); call != nil {
		stmt = buildExecuteDatabaseQueryStatement(call)
	} else if call := mfCtx.CallExternalActionStatement(); call != nil {
		stmt = buildCallExternalActionStatement(call)
	} else if mfCtx.BreakStatement() != nil {
		stmt = &ast.BreakStmt{}
	} else if mfCtx.ContinueStatement() != nil {
		stmt = &ast.ContinueStmt{}
	} else if listOp := mfCtx.ListOperationStatement(); listOp != nil {
		stmt = buildListOperationStatement(listOp)
	} else if aggr := mfCtx.AggregateListStatement(); aggr != nil {
		stmt = buildAggregateListStatement(aggr)
	} else if addTo := mfCtx.AddToListStatement(); addTo != nil {
		stmt = buildAddToListStatement(addTo)
	} else if removeFrom := mfCtx.RemoveFromListStatement(); removeFrom != nil {
		stmt = buildRemoveFromListStatement(removeFrom)
	} else if showPage := mfCtx.ShowPageStatement(); showPage != nil {
		stmt = buildShowPageStatement(showPage)
	} else if mfCtx.ClosePageStatement() != nil {
		stmt = &ast.ClosePageStmt{NumberOfPages: 1}
	} else if mfCtx.ShowHomePageStatement() != nil {
		stmt = &ast.ShowHomePageStmt{}
	} else if showMsg := mfCtx.ShowMessageStatement(); showMsg != nil {
		stmt = buildShowMessageStatement(showMsg)
	} else if valFeedback := mfCtx.ValidationFeedbackStatement(); valFeedback != nil {
		stmt = buildValidationFeedbackStatement(valFeedback)
	} else if restCall := mfCtx.RestCallStatement(); restCall != nil {
		stmt = buildRestCallStatement(restCall)
	} else if sendRest := mfCtx.SendRestRequestStatement(); sendRest != nil {
		stmt = buildSendRestRequestStatement(sendRest)
	}

	// Attach annotations to the statement
	if stmt != nil && ann != nil {
		setStatementAnnotations(stmt, ann)
	}

	return stmt
}

// extractMicroflowAnnotations extracts activity annotations from annotation contexts.
// Handles @position(x, y), @caption 'text', @color Green, @annotation 'text'.
func extractMicroflowAnnotations(annotations []parser.IAnnotationContext) *ast.ActivityAnnotations {
	if len(annotations) == 0 {
		return nil
	}

	result := &ast.ActivityAnnotations{}
	hasAny := false

	for _, annCtx := range annotations {
		ann := annCtx.(*parser.AnnotationContext)
		annName := strings.ToLower(ann.AnnotationName().GetText())

		switch annName {
		case "position":
			// @position(x, y) — uses parenthesized params
			if params := ann.AnnotationParams(); params != nil {
				paramsCtx := params.(*parser.AnnotationParamsContext)
				allParams := paramsCtx.AllAnnotationParam()
				if len(allParams) >= 2 {
					x := parseAnnotationParamInt(allParams[0])
					y := parseAnnotationParamInt(allParams[1])
					result.Position = &ast.Position{X: x, Y: y}
					hasAny = true
				}
			}

		case "caption":
			// @caption 'text' — bare annotationValue
			if valCtx := ann.AnnotationValue(); valCtx != nil {
				text := extractAnnotationValueString(valCtx)
				if text != "" {
					result.Caption = text
					hasAny = true
				}
			}

		case "color":
			// @color Green — bare annotationValue (identifier)
			if valCtx := ann.AnnotationValue(); valCtx != nil {
				text := extractAnnotationValueIdentifier(valCtx)
				if text != "" {
					result.Color = text
					hasAny = true
				}
			}

		case "annotation":
			// @annotation 'text' — bare annotationValue
			if valCtx := ann.AnnotationValue(); valCtx != nil {
				text := extractAnnotationValueString(valCtx)
				if text != "" {
					result.AnnotationText = text
					hasAny = true
				}
			}
		}
	}

	if !hasAny {
		return nil
	}
	return result
}

// extractAnnotationValueString extracts a string value from an annotationValue context.
func extractAnnotationValueString(ctx parser.IAnnotationValueContext) string {
	valCtx := ctx.(*parser.AnnotationValueContext)
	if lit := valCtx.Literal(); lit != nil {
		litCtx := lit.(*parser.LiteralContext)
		if litCtx.STRING_LITERAL() != nil {
			return unquoteString(litCtx.STRING_LITERAL().GetText())
		}
	}
	// Also try expression — it might be a string literal parsed as expression
	if expr := valCtx.Expression(); expr != nil {
		text := expr.GetText()
		if len(text) >= 2 && text[0] == '\'' && text[len(text)-1] == '\'' {
			return unquoteString(text)
		}
	}
	return ""
}

// extractAnnotationValueIdentifier extracts an identifier value from an annotationValue context.
func extractAnnotationValueIdentifier(ctx parser.IAnnotationValueContext) string {
	valCtx := ctx.(*parser.AnnotationValueContext)
	// Try qualifiedName first (handles plain identifiers like "Green")
	if qn := valCtx.QualifiedName(); qn != nil {
		return qn.GetText()
	}
	// Try expression (might be a plain identifier)
	if expr := valCtx.Expression(); expr != nil {
		return expr.GetText()
	}
	// Try literal
	if lit := valCtx.Literal(); lit != nil {
		return lit.GetText()
	}
	return ""
}

// setStatementAnnotations sets the Annotations field on a microflow statement via type switch.
func setStatementAnnotations(stmt ast.MicroflowStatement, ann *ast.ActivityAnnotations) {
	switch s := stmt.(type) {
	case *ast.DeclareStmt:
		s.Annotations = ann
	case *ast.MfSetStmt:
		s.Annotations = ann
	case *ast.ReturnStmt:
		s.Annotations = ann
	case *ast.RaiseErrorStmt:
		s.Annotations = ann
	case *ast.CreateObjectStmt:
		s.Annotations = ann
	case *ast.ChangeObjectStmt:
		s.Annotations = ann
	case *ast.MfCommitStmt:
		s.Annotations = ann
	case *ast.DeleteObjectStmt:
		s.Annotations = ann
	case *ast.RollbackStmt:
		s.Annotations = ann
	case *ast.RetrieveStmt:
		s.Annotations = ann
	case *ast.IfStmt:
		s.Annotations = ann
	case *ast.LoopStmt:
		s.Annotations = ann
	case *ast.WhileStmt:
		s.Annotations = ann
	case *ast.LogStmt:
		s.Annotations = ann
	case *ast.CallMicroflowStmt:
		s.Annotations = ann
	case *ast.CallJavaActionStmt:
		s.Annotations = ann
	case *ast.ExecuteDatabaseQueryStmt:
		s.Annotations = ann
	case *ast.CallExternalActionStmt:
		s.Annotations = ann
	case *ast.BreakStmt:
		s.Annotations = ann
	case *ast.ContinueStmt:
		s.Annotations = ann
	case *ast.ListOperationStmt:
		s.Annotations = ann
	case *ast.AggregateListStmt:
		s.Annotations = ann
	case *ast.CreateListStmt:
		s.Annotations = ann
	case *ast.AddToListStmt:
		s.Annotations = ann
	case *ast.RemoveFromListStmt:
		s.Annotations = ann
	case *ast.ShowPageStmt:
		s.Annotations = ann
	case *ast.ClosePageStmt:
		s.Annotations = ann
	case *ast.ShowHomePageStmt:
		s.Annotations = ann
	case *ast.ShowMessageStmt:
		s.Annotations = ann
	case *ast.ValidationFeedbackStmt:
		s.Annotations = ann
	case *ast.RestCallStmt:
		s.Annotations = ann
	case *ast.SendRestRequestStmt:
		s.Annotations = ann
	}
}

// buildOnErrorClause converts an OnErrorClauseContext to an ErrorHandlingClause.
func buildOnErrorClause(ctx parser.IOnErrorClauseContext) *ast.ErrorHandlingClause {
	if ctx == nil {
		return nil
	}
	errCtx := ctx.(*parser.OnErrorClauseContext)

	if errCtx.CONTINUE() != nil {
		return &ast.ErrorHandlingClause{Type: ast.ErrorHandlingContinue}
	}
	if errCtx.ROLLBACK() != nil && errCtx.LBRACE() == nil {
		return &ast.ErrorHandlingClause{Type: ast.ErrorHandlingRollback}
	}
	if errCtx.LBRACE() != nil {
		body := buildMicroflowBody(errCtx.MicroflowBody())
		if errCtx.WITHOUT() != nil {
			return &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustomWithoutRollback, Body: body}
		}
		return &ast.ErrorHandlingClause{Type: ast.ErrorHandlingCustom, Body: body}
	}
	return nil
}

// buildDeclareStatement converts DECLARE statement context to DeclareStmt.
func buildDeclareStatement(ctx parser.IDeclareStatementContext) *ast.DeclareStmt {
	if ctx == nil {
		return nil
	}
	declCtx := ctx.(*parser.DeclareStatementContext)

	stmt := &ast.DeclareStmt{}

	// Get variable name
	if v := declCtx.VARIABLE(); v != nil {
		stmt.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get type
	if dt := declCtx.DataType(); dt != nil {
		stmt.Type = buildDataType(dt)
	}

	// Get optional initial value
	if expr := declCtx.Expression(); expr != nil {
		stmt.InitialValue = buildExpression(expr)
	}

	return stmt
}

// buildSetStatement converts SET statement context to MfSetStmt or specialized statement types.
// When the expression is a list operation (HEAD, TAIL, etc.) or aggregate (COUNT, SUM, etc.),
// this returns the specialized statement type instead of MfSetStmt.
func buildSetStatement(ctx parser.ISetStatementContext) ast.MicroflowStatement {
	if ctx == nil {
		return nil
	}
	setCtx := ctx.(*parser.SetStatementContext)

	// Get target variable name
	var targetVar string
	if v := setCtx.VARIABLE(); v != nil {
		targetVar = strings.TrimPrefix(v.GetText(), "$")
	} else if ap := setCtx.AttributePath(); ap != nil {
		targetVar = ap.GetText()
	}

	// Get value expression
	var valueExpr ast.Expression
	if expr := setCtx.Expression(); expr != nil {
		valueExpr = buildExpression(expr)
	}

	// Check if the expression is a list operation or aggregate function
	if funcCall, ok := valueExpr.(*ast.FunctionCallExpr); ok {
		funcName := strings.ToUpper(funcCall.Name)

		// Check for list operations: HEAD, TAIL, FIND, FILTER, SORT, UNION, INTERSECT, SUBTRACT, CONTAINS, EQUALS
		switch funcName {
		case "HEAD":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpHead,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
			}
		case "TAIL":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpTail,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
			}
		case "FIND":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpFind,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
				Condition:      getArgumentExpression(funcCall.Arguments, 1),
			}
		case "FILTER":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpFilter,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
				Condition:      getArgumentExpression(funcCall.Arguments, 1),
			}
		case "SORT":
			stmt := &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpSort,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
			}
			// Parse sort specifications from remaining arguments
			stmt.SortSpecs = extractSortSpecs(funcCall.Arguments[1:])
			return stmt
		case "UNION":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpUnion,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
				SecondVariable: extractVariableName(funcCall.Arguments, 1),
			}
		case "INTERSECT":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpIntersect,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
				SecondVariable: extractVariableName(funcCall.Arguments, 1),
			}
		case "SUBTRACT":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpSubtract,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
				SecondVariable: extractVariableName(funcCall.Arguments, 1),
			}
		case "CONTAINS":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpContains,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
				SecondVariable: extractVariableName(funcCall.Arguments, 1),
			}
		case "EQUALS":
			return &ast.ListOperationStmt{
				OutputVariable: targetVar,
				Operation:      ast.ListOpEquals,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
				SecondVariable: extractVariableName(funcCall.Arguments, 1),
			}
		// Check for aggregate operations: COUNT, SUM, AVERAGE, MINIMUM, MAXIMUM
		case "COUNT":
			return &ast.AggregateListStmt{
				OutputVariable: targetVar,
				Operation:      ast.AggregateCount,
				InputVariable:  extractVariableName(funcCall.Arguments, 0),
			}
		case "SUM":
			inputVar, attr := extractVariableAndAttribute(funcCall.Arguments, 0)
			return &ast.AggregateListStmt{
				OutputVariable: targetVar,
				Operation:      ast.AggregateSum,
				InputVariable:  inputVar,
				Attribute:      attr,
			}
		case "AVERAGE":
			inputVar, attr := extractVariableAndAttribute(funcCall.Arguments, 0)
			return &ast.AggregateListStmt{
				OutputVariable: targetVar,
				Operation:      ast.AggregateAverage,
				InputVariable:  inputVar,
				Attribute:      attr,
			}
		case "MINIMUM":
			inputVar, attr := extractVariableAndAttribute(funcCall.Arguments, 0)
			return &ast.AggregateListStmt{
				OutputVariable: targetVar,
				Operation:      ast.AggregateMinimum,
				InputVariable:  inputVar,
				Attribute:      attr,
			}
		case "MAXIMUM":
			inputVar, attr := extractVariableAndAttribute(funcCall.Arguments, 0)
			return &ast.AggregateListStmt{
				OutputVariable: targetVar,
				Operation:      ast.AggregateMaximum,
				InputVariable:  inputVar,
				Attribute:      attr,
			}
		}
	}

	// Default: regular SET statement
	return &ast.MfSetStmt{
		Target: targetVar,
		Value:  valueExpr,
	}
}

// extractVariableName extracts a variable name from an argument at the given index.
func extractVariableName(args []ast.Expression, index int) string {
	if index >= len(args) {
		return ""
	}
	if varExpr, ok := args[index].(*ast.VariableExpr); ok {
		return varExpr.Name
	}
	// If it's an identifier (unquoted), treat it as a variable name
	if identExpr, ok := args[index].(*ast.IdentifierExpr); ok {
		return identExpr.Name
	}
	return ""
}

// getArgumentExpression returns the expression at the given index, or nil if not present.
func getArgumentExpression(args []ast.Expression, index int) ast.Expression {
	if index >= len(args) {
		return nil
	}
	return args[index]
}

// extractVariableAndAttribute extracts variable and attribute from $Var/Attr or $Var, Attr.
func extractVariableAndAttribute(args []ast.Expression, index int) (varName string, attrName string) {
	if index >= len(args) {
		return "", ""
	}
	// Check for attribute path like $Var/Attr
	if pathExpr, ok := args[index].(*ast.AttributePathExpr); ok {
		varName = pathExpr.Variable
		if len(pathExpr.Path) > 0 {
			attrName = pathExpr.Path[len(pathExpr.Path)-1]
		}
		return
	}
	// Check for simple variable
	if varExpr, ok := args[index].(*ast.VariableExpr); ok {
		varName = varExpr.Name
		// Look for attribute in next argument
		if index+1 < len(args) {
			if identExpr, ok := args[index+1].(*ast.IdentifierExpr); ok {
				attrName = identExpr.Name
			}
		}
		return
	}
	return "", ""
}

// extractSortSpecs extracts sort specifications from function arguments.
// Expected format: Attr ASC, Attr2 DESC or just Attr (defaults to ASC)
func extractSortSpecs(args []ast.Expression) []ast.SortSpec {
	var specs []ast.SortSpec
	for _, arg := range args {
		// Try to parse as "Attr ASC" or "Attr DESC" or just "Attr"
		if identExpr, ok := arg.(*ast.IdentifierExpr); ok {
			// Parse "Name ASC" or "Name DESC" format from expression visitor
			name := identExpr.Name
			ascending := true
			if before, ok0 := strings.CutSuffix(name, " DESC"); ok0 {
				name = before
				ascending = false
			} else if before, ok0 := strings.CutSuffix(name, " ASC"); ok0 {
				name = before
			}
			specs = append(specs, ast.SortSpec{
				Attribute: name,
				Ascending: ascending,
			})
		}
		// For more complex expressions, extract what we can
		if binExpr, ok := arg.(*ast.BinaryExpr); ok {
			// Handle "Attr ASC" parsed as binary expression
			if leftIdent, ok := binExpr.Left.(*ast.IdentifierExpr); ok {
				ascending := true
				if strings.ToUpper(binExpr.Operator) == "DESC" {
					ascending = false
				}
				specs = append(specs, ast.SortSpec{
					Attribute: leftIdent.Name,
					Ascending: ascending,
				})
			}
		}
	}
	return specs
}

// buildCreateObjectStatement converts CREATE OBJECT statement context to CreateObjectStmt.
// Grammar: (VARIABLE EQUALS)? CREATE nonListDataType (LPAREN memberAssignmentList? RPAREN)?
// Example: $NewProduct = CREATE MfTest.Product (Name = $Name, Code = $Code);
func buildCreateObjectStatement(ctx parser.ICreateObjectStatementContext) *ast.CreateObjectStmt {
	if ctx == nil {
		return nil
	}
	createCtx := ctx.(*parser.CreateObjectStatementContext)

	stmt := &ast.CreateObjectStmt{}

	// Get variable name
	if v := createCtx.VARIABLE(); v != nil {
		stmt.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get entity type from nonListDataType - use microflow builder to get entity reference
	if dt := createCtx.NonListDataType(); dt != nil {
		dataType := buildNonListDataType(dt)
		if dataType.EntityRef != nil {
			stmt.EntityType = *dataType.EntityRef
		}
	}

	// Get SET member assignments
	if memberList := createCtx.MemberAssignmentList(); memberList != nil {
		stmt.Changes = buildMemberAssignmentList(memberList)
	}

	// Check for ON ERROR clause
	if errClause := createCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildChangeObjectStatement converts CHANGE statement context to ChangeObjectStmt.
// Grammar: CHANGE VARIABLE (LPAREN memberAssignmentList? RPAREN)?
// Example: CHANGE $Product (Name = $NewName, ModifiedDate = [%CurrentDateTime%]);
func buildChangeObjectStatement(ctx parser.IChangeObjectStatementContext) *ast.ChangeObjectStmt {
	if ctx == nil {
		return nil
	}
	changeCtx := ctx.(*parser.ChangeObjectStatementContext)

	stmt := &ast.ChangeObjectStmt{}

	// Get variable name
	if v := changeCtx.VARIABLE(); v != nil {
		stmt.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get SET member assignments
	if memberList := changeCtx.MemberAssignmentList(); memberList != nil {
		stmt.Changes = buildMemberAssignmentList(memberList)
	}

	return stmt
}

// buildCommitStatement converts COMMIT statement context to MfCommitStmt.
// Grammar: COMMIT VARIABLE (WITH EVENTS)? REFRESH?
func buildCommitStatement(ctx parser.ICommitStatementContext) *ast.MfCommitStmt {
	if ctx == nil {
		return nil
	}
	commitCtx := ctx.(*parser.CommitStatementContext)

	stmt := &ast.MfCommitStmt{}

	// Get variable name
	if v := commitCtx.VARIABLE(); v != nil {
		stmt.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Check for WITH EVENTS
	if commitCtx.EVENTS() != nil {
		stmt.WithEvents = true
	}

	// Check for REFRESH
	if commitCtx.REFRESH() != nil {
		stmt.RefreshInClient = true
	}

	// Check for ON ERROR clause
	if errClause := commitCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildDeleteObjectStatement converts DELETE statement context to DeleteObjectStmt.
func buildDeleteObjectStatement(ctx parser.IDeleteObjectStatementContext) *ast.DeleteObjectStmt {
	if ctx == nil {
		return nil
	}
	delCtx := ctx.(*parser.DeleteObjectStatementContext)

	stmt := &ast.DeleteObjectStmt{}

	// Get variable name
	if v := delCtx.VARIABLE(); v != nil {
		stmt.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Check for ON ERROR clause
	if errClause := delCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildRollbackStatement converts ROLLBACK statement context to RollbackStmt.
func buildRollbackStatement(ctx parser.IRollbackStatementContext) *ast.RollbackStmt {
	if ctx == nil {
		return nil
	}
	rollCtx := ctx.(*parser.RollbackStatementContext)

	stmt := &ast.RollbackStmt{}

	// Get variable name
	if v := rollCtx.VARIABLE(); v != nil {
		stmt.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Check for REFRESH keyword
	stmt.RefreshInClient = rollCtx.REFRESH() != nil

	return stmt
}

// buildRetrieveStatement converts RETRIEVE statement context to RetrieveStmt.
// Grammar: RETRIEVE VARIABLE FROM retrieveSource (WHERE expression)? (SORT_BY sortColumn+)? (OFFSET NUMBER_LITERAL)? (LIMIT NUMBER_LITERAL)?
func buildRetrieveStatement(ctx parser.IRetrieveStatementContext) *ast.RetrieveStmt {
	if ctx == nil {
		return nil
	}
	retrCtx := ctx.(*parser.RetrieveStatementContext)

	stmt := &ast.RetrieveStmt{}

	// Get variable name
	if v := retrCtx.VARIABLE(); v != nil {
		stmt.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get source (database entity or association path)
	if src := retrCtx.RetrieveSource(); src != nil {
		srcCtx := src.(*parser.RetrieveSourceContext)
		if v := srcCtx.VARIABLE(); v != nil {
			// Association retrieve: $Parent/Module.AssociationName
			stmt.StartVariable = strings.TrimPrefix(v.GetText(), "$")
			if qn := srcCtx.QualifiedName(); qn != nil {
				stmt.Source = buildQualifiedName(qn)
			}
		} else if qn := srcCtx.QualifiedName(); qn != nil {
			// Database retrieve: Module.Entity
			stmt.Source = buildQualifiedName(qn)
		}
	}

	// Get WHERE condition (now at RETRIEVE level)
	// Supports both bare expression: WHERE expr
	// and bracket notation: WHERE [expr]
	if retrCtx.WHERE() != nil {
		if xc := retrCtx.XpathConstraint(); xc != nil {
			xcCtx := xc.(*parser.XpathConstraintContext)
			if xpathExpr := xcCtx.XpathExpr(); xpathExpr != nil {
				stmt.Where = buildXPathExpr(xpathExpr)
			}
		} else if expr := retrCtx.Expression(0); expr != nil {
			stmt.Where = buildExpression(expr)
		}
	}

	// Get SORT BY clause with multiple columns
	if retrCtx.SORT_BY() != nil {
		for _, sortColCtx := range retrCtx.AllSortColumn() {
			col := buildSortColumnMicroflow(sortColCtx)
			if col != nil {
				stmt.SortColumns = append(stmt.SortColumns, *col)
			}
		}
	}

	// Get LIMIT and OFFSET expressions
	if limitExpr := retrCtx.GetLimitExpr(); limitExpr != nil {
		stmt.Limit = limitExpr.GetText()
	}
	if offsetExpr := retrCtx.GetOffsetExpr(); offsetExpr != nil {
		stmt.Offset = offsetExpr.GetText()
	}

	// Check for ON ERROR clause
	if errClause := retrCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildSortColumnMicroflow builds a sort column definition from a SortColumnContext.
// This is a duplicate of buildSortColumn in visitor_page_widgets.go but in a different file.
func buildSortColumnMicroflow(ctx parser.ISortColumnContext) *ast.SortColumnDef {
	if ctx == nil {
		return nil
	}
	colCtx := ctx.(*parser.SortColumnContext)

	col := &ast.SortColumnDef{
		Order: "ASC", // Default to ASC
	}

	// Get attribute name from QualifiedName or IDENTIFIER
	if qn := colCtx.QualifiedName(); qn != nil {
		col.Attribute = qn.GetText()
	} else if id := colCtx.IDENTIFIER(); id != nil {
		col.Attribute = id.GetText()
	}

	// Get sort order
	if colCtx.DESC() != nil {
		col.Order = "DESC"
	}

	return col
}

// buildIfStatement converts IF statement context to IfStmt.
func buildIfStatement(ctx parser.IIfStatementContext) *ast.IfStmt {
	if ctx == nil {
		return nil
	}
	ifCtx := ctx.(*parser.IfStatementContext)

	stmt := &ast.IfStmt{}

	// Get all expressions (condition for IF and ELSIFs)
	exprs := ifCtx.AllExpression()
	if len(exprs) > 0 {
		stmt.Condition = buildExpression(exprs[0])
	}

	// Get all microflow bodies (THEN, ELSIF THENs, ELSE)
	bodies := ifCtx.AllMicroflowBody()
	if len(bodies) > 0 {
		stmt.ThenBody = buildMicroflowBody(bodies[0])
	}
	// Last body is ELSE if there's no ELSIF or if there are more bodies than expressions
	if len(bodies) > len(exprs) {
		stmt.ElseBody = buildMicroflowBody(bodies[len(bodies)-1])
	}

	return stmt
}

// buildLoopStatement converts LOOP statement context to LoopStmt.
func buildLoopStatement(ctx parser.ILoopStatementContext) *ast.LoopStmt {
	if ctx == nil {
		return nil
	}
	loopCtx := ctx.(*parser.LoopStatementContext)

	stmt := &ast.LoopStmt{}

	// Get variables (first is loop variable, second is list)
	vars := loopCtx.AllVARIABLE()
	if len(vars) >= 1 {
		stmt.LoopVariable = strings.TrimPrefix(vars[0].GetText(), "$")
	}
	if len(vars) >= 2 {
		stmt.ListVariable = strings.TrimPrefix(vars[1].GetText(), "$")
	}

	// Get body
	if body := loopCtx.MicroflowBody(); body != nil {
		stmt.Body = buildMicroflowBody(body)
	}

	return stmt
}

// buildWhileStatement converts WHILE statement context to WhileStmt.
func buildWhileStatement(ctx parser.IWhileStatementContext) *ast.WhileStmt {
	if ctx == nil {
		return nil
	}
	wsCtx := ctx.(*parser.WhileStatementContext)

	stmt := &ast.WhileStmt{}

	// Get condition expression
	if expr := wsCtx.Expression(); expr != nil {
		stmt.Condition = buildExpression(expr)
	}

	// Get body
	if body := wsCtx.MicroflowBody(); body != nil {
		stmt.Body = buildMicroflowBody(body)
	}

	return stmt
}

// buildReturnStatement converts RETURN statement context to ReturnStmt.
func buildReturnStatement(ctx parser.IReturnStatementContext) *ast.ReturnStmt {
	if ctx == nil {
		return nil
	}
	retCtx := ctx.(*parser.ReturnStatementContext)

	stmt := &ast.ReturnStmt{}

	// Get optional return value
	if expr := retCtx.Expression(); expr != nil {
		stmt.Value = buildExpression(expr)
	}

	return stmt
}
