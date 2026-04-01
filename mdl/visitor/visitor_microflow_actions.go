// SPDX-License-Identifier: Apache-2.0

package visitor

import (
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// buildLogStatement converts LOG statement context to LogStmt.
// Grammar: LOG logLevel? (NODE STRING_LITERAL)? expression logTemplateParams?
func buildLogStatement(ctx parser.ILogStatementContext) *ast.LogStmt {
	if ctx == nil {
		return nil
	}
	logCtx := ctx.(*parser.LogStatementContext)

	stmt := &ast.LogStmt{
		Level: ast.LogInfo, // Default level
	}

	// Get log level
	if level := logCtx.LogLevel(); level != nil {
		levelCtx := level.(*parser.LogLevelContext)
		if levelCtx.INFO() != nil {
			stmt.Level = ast.LogInfo
		} else if levelCtx.WARNING() != nil {
			stmt.Level = ast.LogWarning
		} else if levelCtx.ERROR() != nil {
			stmt.Level = ast.LogError
		} else if levelCtx.DEBUG() != nil {
			stmt.Level = ast.LogDebug
		} else if levelCtx.TRACE() != nil {
			stmt.Level = ast.LogTrace
		} else if levelCtx.CRITICAL() != nil {
			stmt.Level = ast.LogCritical
		}
	}

	// Get optional node name
	if logCtx.NODE() != nil {
		if str := logCtx.STRING_LITERAL(); str != nil {
			stmt.Node = unquoteString(str.GetText())
		}
	}

	// Get message expression
	if expr := logCtx.Expression(); expr != nil {
		stmt.Message = buildExpression(expr)
	}

	// Parse template parameters: WITH ({1} = expr, {2} = expr, ...)
	if params := logCtx.LogTemplateParams(); params != nil {
		logParamsCtx := params.(*parser.LogTemplateParamsContext)
		if tplParams := logParamsCtx.TemplateParams(); tplParams != nil {
			stmt.Template = buildTemplateParams(tplParams)
		}
	}

	return stmt
}

// buildTemplateParams converts templateParams context to []ast.TemplateParam.
// Handles both WITH ({1} = expr) syntax and deprecated PARAMETERS [array] syntax.
func buildTemplateParams(ctx parser.ITemplateParamsContext) []ast.TemplateParam {
	if ctx == nil {
		return nil
	}
	paramsCtx := ctx.(*parser.TemplateParamsContext)

	var result []ast.TemplateParam

	// Handle WITH ({1} = expr, {2} = expr, ...) syntax
	for _, param := range paramsCtx.AllTemplateParam() {
		paramCtx := param.(*parser.TemplateParamContext)
		indexStr := paramCtx.NUMBER_LITERAL().GetText()
		index, _ := strconv.Atoi(indexStr)

		var tp ast.TemplateParam
		tp.Index = index

		// Parse the expression and check for data source attribute reference
		if exprCtx := paramCtx.Expression(); exprCtx != nil {
			expr := buildExpression(exprCtx)
			tp.Value = expr

			// Check if this is a $Widget.Attr pattern (AttributePathExpr with Path)
			if pathExpr, ok := expr.(*ast.AttributePathExpr); ok && len(pathExpr.Path) > 0 {
				// This is a data source attribute reference
				tp.DataSourceName = pathExpr.Variable
				tp.AttributeName = pathExpr.Path[len(pathExpr.Path)-1]
			}
		}

		result = append(result, tp)
	}

	// Handle deprecated PARAMETERS [array] syntax
	if arr := paramsCtx.ArrayLiteral(); arr != nil {
		arrCtx := arr.(*parser.ArrayLiteralContext)
		for i, lit := range arrCtx.AllLiteral() {
			var tp ast.TemplateParam
			tp.Index = i + 1 // 1-based index
			tp.Value = buildLiteralExpression(lit)
			result = append(result, tp)
		}
	}

	return result
}

// buildCallMicroflowStatement converts CALL MICROFLOW statement context to CallMicroflowStmt.
// Grammar: (VARIABLE EQUALS)? CALL MICROFLOW qualifiedName LPAREN callArgumentList? RPAREN
func buildCallMicroflowStatement(ctx parser.ICallMicroflowStatementContext) *ast.CallMicroflowStmt {
	if ctx == nil {
		return nil
	}
	callCtx := ctx.(*parser.CallMicroflowStatementContext)

	stmt := &ast.CallMicroflowStmt{}

	// Get result variable if present
	if v := callCtx.VARIABLE(); v != nil {
		stmt.OutputVariable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get microflow name
	if qn := callCtx.QualifiedName(); qn != nil {
		stmt.MicroflowName = buildQualifiedName(qn)
	}

	// Get arguments from callArgumentList
	if argList := callCtx.CallArgumentList(); argList != nil {
		stmt.Arguments = buildCallArgumentList(argList)
	}

	// Check for ON ERROR clause
	if errClause := callCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildCallJavaActionStatement converts CALL JAVA ACTION statement context to CallJavaActionStmt.
// Grammar: (VARIABLE EQUALS)? CALL JAVA ACTION qualifiedName LPAREN callArgumentList? RPAREN
func buildCallJavaActionStatement(ctx parser.ICallJavaActionStatementContext) *ast.CallJavaActionStmt {
	if ctx == nil {
		return nil
	}
	callCtx := ctx.(*parser.CallJavaActionStatementContext)

	stmt := &ast.CallJavaActionStmt{}

	// Get result variable if present
	if v := callCtx.VARIABLE(); v != nil {
		stmt.OutputVariable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get java action name
	if qn := callCtx.QualifiedName(); qn != nil {
		stmt.ActionName = buildQualifiedName(qn)
	}

	// Get arguments from callArgumentList
	if argList := callCtx.CallArgumentList(); argList != nil {
		stmt.Arguments = buildCallArgumentList(argList)
	}

	// Check for ON ERROR clause
	if errClause := callCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildExecuteDatabaseQueryStatement converts EXECUTE DATABASE QUERY context to ExecuteDatabaseQueryStmt.
func buildExecuteDatabaseQueryStatement(ctx parser.IExecuteDatabaseQueryStatementContext) *ast.ExecuteDatabaseQueryStmt {
	if ctx == nil {
		return nil
	}
	execCtx := ctx.(*parser.ExecuteDatabaseQueryStatementContext)

	stmt := &ast.ExecuteDatabaseQueryStmt{}

	// Get result variable if present
	if v := execCtx.VARIABLE(); v != nil {
		stmt.OutputVariable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get query name (Module.Connection.QueryName — 3-part identifier)
	if qn := execCtx.QualifiedName(); qn != nil {
		stmt.QueryName = getQualifiedNameText(qn)
	}

	// Get dynamic query if present
	if execCtx.DYNAMIC() != nil {
		if sl := execCtx.STRING_LITERAL(); sl != nil {
			stmt.DynamicQuery = unquoteString(sl.GetText())
		} else if ds := execCtx.DOLLAR_STRING(); ds != nil {
			stmt.DynamicQuery = unquoteDollarString(ds.GetText())
		} else if expr := execCtx.Expression(); expr != nil {
			stmt.DynamicQuery = expr.GetText()
		}
	}

	// Get query parameter arguments from first callArgumentList
	if argList := execCtx.CallArgumentList(0); argList != nil {
		stmt.Arguments = buildCallArgumentList(argList)
	}

	// Get connection parameter arguments from second callArgumentList (after CONNECTION keyword)
	if execCtx.CONNECTION() != nil {
		if argList := execCtx.CallArgumentList(1); argList != nil {
			stmt.ConnectionArguments = buildCallArgumentList(argList)
		}
	}

	// Check for ON ERROR clause
	if errClause := execCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildCallExternalActionStatement converts CALL EXTERNAL ACTION context to CallExternalActionStmt.
// Grammar: (VARIABLE EQUALS)? CALL EXTERNAL ACTION qualifiedName DOT IDENTIFIER LPAREN callArgumentList? RPAREN
func buildCallExternalActionStatement(ctx parser.ICallExternalActionStatementContext) *ast.CallExternalActionStmt {
	if ctx == nil {
		return nil
	}
	callCtx := ctx.(*parser.CallExternalActionStatementContext)

	stmt := &ast.CallExternalActionStmt{}

	// Get result variable if present
	if v := callCtx.VARIABLE(); v != nil {
		stmt.OutputVariable = strings.TrimPrefix(v.GetText(), "$")
	}

	// qualifiedName matches Module.ServiceName.ActionName (3+ parts)
	// Split off the last segment as the action name, rest is service qualified name
	if qn := callCtx.QualifiedName(); qn != nil {
		fullText := getQualifiedNameText(qn) // e.g. "Module.ServiceName.ActionName"
		if lastDot := strings.LastIndex(fullText, "."); lastDot >= 0 {
			servicePart := fullText[:lastDot]      // "Module.ServiceName"
			stmt.ActionName = fullText[lastDot+1:] // "ActionName"
			// Split service part into Module.Name
			if before, after, ok := strings.Cut(servicePart, "."); ok {
				stmt.ServiceName = ast.QualifiedName{
					Module: before,
					Name:   after,
				}
			} else {
				stmt.ServiceName = ast.QualifiedName{Name: servicePart}
			}
		} else {
			// Single identifier — treat as action name
			stmt.ActionName = fullText
		}
	}

	// Get arguments from callArgumentList
	if argList := callCtx.CallArgumentList(); argList != nil {
		stmt.Arguments = buildCallArgumentList(argList)
	}

	// Check for ON ERROR clause
	if errClause := callCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildCallArgumentList converts callArgumentList context to CallArgument slice.
func buildCallArgumentList(ctx parser.ICallArgumentListContext) []ast.CallArgument {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.CallArgumentListContext)
	var args []ast.CallArgument

	for _, argCtx := range listCtx.AllCallArgument() {
		arg := argCtx.(*parser.CallArgumentContext)
		ca := ast.CallArgument{}

		// Name can be VARIABLE or parameterName
		if v := arg.VARIABLE(); v != nil {
			ca.Name = strings.TrimPrefix(v.GetText(), "$")
		} else if pn := arg.ParameterName(); pn != nil {
			ca.Name = parameterNameText(pn)
		}
		if expr := arg.Expression(); expr != nil {
			ca.Value = buildExpression(expr)
		}

		args = append(args, ca)
	}

	return args
}

// buildMemberAssignmentList converts memberAssignmentList context to ChangeItem slice.
func buildMemberAssignmentList(ctx parser.IMemberAssignmentListContext) []ast.ChangeItem {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.MemberAssignmentListContext)
	var items []ast.ChangeItem

	for _, assignCtx := range listCtx.AllMemberAssignment() {
		assign := assignCtx.(*parser.MemberAssignmentContext)
		ci := ast.ChangeItem{}

		// Get attribute name (can be IDENTIFIER, keyword, or quoted identifier)
		if name := assign.MemberAttributeName(); name != nil {
			ci.Attribute = memberAttributeNameText(name)
		}
		if expr := assign.Expression(); expr != nil {
			ci.Value = buildExpression(expr)
		}

		items = append(items, ci)
	}

	return items
}

// buildChangeList converts changeList context to ChangeItem slice.
func buildChangeList(ctx parser.IChangeListContext) []ast.ChangeItem {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.ChangeListContext)
	var items []ast.ChangeItem

	for _, itemCtx := range listCtx.AllChangeItem() {
		item := itemCtx.(*parser.ChangeItemContext)
		ci := ast.ChangeItem{}

		if id := item.IDENTIFIER(); id != nil {
			ci.Attribute = id.GetText()
		}
		if expr := item.Expression(); expr != nil {
			ci.Value = buildExpression(expr)
		}

		items = append(items, ci)
	}

	return items
}

// ============================================================================
// List Operation Statements
// ============================================================================

// buildListOperationStatement converts list operation statement context to ListOperationStmt.
// Grammar: VARIABLE EQUALS listOperation
func buildListOperationStatement(ctx parser.IListOperationStatementContext) *ast.ListOperationStmt {
	if ctx == nil {
		return nil
	}
	listOpCtx := ctx.(*parser.ListOperationStatementContext)

	stmt := &ast.ListOperationStmt{}

	// Get output variable
	if v := listOpCtx.VARIABLE(); v != nil {
		stmt.OutputVariable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get the list operation
	if opCtx := listOpCtx.ListOperation(); opCtx != nil {
		op := opCtx.(*parser.ListOperationContext)

		// Get all variables from the operation
		vars := op.AllVARIABLE()

		// Determine operation type based on which token is present
		if op.HEAD() != nil {
			stmt.Operation = ast.ListOpHead
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
		} else if op.TAIL() != nil {
			stmt.Operation = ast.ListOpTail
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
		} else if op.FIND() != nil {
			stmt.Operation = ast.ListOpFind
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if expr := op.Expression(); expr != nil {
				stmt.Condition = buildExpression(expr)
			}
		} else if op.FILTER() != nil {
			stmt.Operation = ast.ListOpFilter
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if expr := op.Expression(); expr != nil {
				stmt.Condition = buildExpression(expr)
			}
		} else if op.SORT() != nil {
			stmt.Operation = ast.ListOpSort
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if sortList := op.SortSpecList(); sortList != nil {
				stmt.SortSpecs = buildSortSpecList(sortList)
			}
		} else if op.UNION() != nil {
			stmt.Operation = ast.ListOpUnion
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if len(vars) >= 2 {
				stmt.SecondVariable = strings.TrimPrefix(vars[1].GetText(), "$")
			}
		} else if op.INTERSECT() != nil {
			stmt.Operation = ast.ListOpIntersect
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if len(vars) >= 2 {
				stmt.SecondVariable = strings.TrimPrefix(vars[1].GetText(), "$")
			}
		} else if op.SUBTRACT() != nil {
			stmt.Operation = ast.ListOpSubtract
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if len(vars) >= 2 {
				stmt.SecondVariable = strings.TrimPrefix(vars[1].GetText(), "$")
			}
		} else if op.CONTAINS() != nil {
			stmt.Operation = ast.ListOpContains
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if len(vars) >= 2 {
				stmt.SecondVariable = strings.TrimPrefix(vars[1].GetText(), "$")
			}
		} else if op.EQUALS_OP() != nil {
			stmt.Operation = ast.ListOpEquals
			if len(vars) >= 1 {
				stmt.InputVariable = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if len(vars) >= 2 {
				stmt.SecondVariable = strings.TrimPrefix(vars[1].GetText(), "$")
			}
		}
	}

	return stmt
}

// buildSortSpecList converts sortSpecList context to SortSpec slice.
func buildSortSpecList(ctx parser.ISortSpecListContext) []ast.SortSpec {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.SortSpecListContext)
	var specs []ast.SortSpec

	for _, specCtx := range listCtx.AllSortSpec() {
		spec := specCtx.(*parser.SortSpecContext)
		ss := ast.SortSpec{
			Ascending: true, // Default to ascending
		}

		if id := spec.IDENTIFIER(); id != nil {
			ss.Attribute = id.GetText()
		}
		if spec.DESC() != nil {
			ss.Ascending = false
		}

		specs = append(specs, ss)
	}

	return specs
}

// buildAggregateListStatement converts aggregate list statement context to AggregateListStmt.
// Grammar: VARIABLE EQUALS listAggregateOperation
func buildAggregateListStatement(ctx parser.IAggregateListStatementContext) *ast.AggregateListStmt {
	if ctx == nil {
		return nil
	}
	aggrCtx := ctx.(*parser.AggregateListStatementContext)

	stmt := &ast.AggregateListStmt{}

	// Get output variable
	if v := aggrCtx.VARIABLE(); v != nil {
		stmt.OutputVariable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get the aggregate operation
	if opCtx := aggrCtx.ListAggregateOperation(); opCtx != nil {
		op := opCtx.(*parser.ListAggregateOperationContext)

		// Determine operation type
		if op.COUNT() != nil {
			stmt.Operation = ast.AggregateCount
			if v := op.VARIABLE(); v != nil {
				stmt.InputVariable = strings.TrimPrefix(v.GetText(), "$")
			}
		} else if op.SUM() != nil {
			stmt.Operation = ast.AggregateSum
			if path := op.AttributePath(); path != nil {
				stmt.InputVariable, stmt.Attribute = parseAttributePath(path.GetText())
			}
		} else if op.AVERAGE() != nil {
			stmt.Operation = ast.AggregateAverage
			if path := op.AttributePath(); path != nil {
				stmt.InputVariable, stmt.Attribute = parseAttributePath(path.GetText())
			}
		} else if op.MINIMUM() != nil {
			stmt.Operation = ast.AggregateMinimum
			if path := op.AttributePath(); path != nil {
				stmt.InputVariable, stmt.Attribute = parseAttributePath(path.GetText())
			}
		} else if op.MAXIMUM() != nil {
			stmt.Operation = ast.AggregateMaximum
			if path := op.AttributePath(); path != nil {
				stmt.InputVariable, stmt.Attribute = parseAttributePath(path.GetText())
			}
		}
	}

	return stmt
}

// parseAttributePath parses an attribute path like "$Products/Price" or "$Products.Price" into variable and attribute.
func parseAttributePath(path string) (variable string, attribute string) {
	// Remove $ prefix if present
	path = strings.TrimPrefix(path, "$")

	// Try splitting on / first (XPath style), then on . (object property style)
	var parts []string
	if strings.Contains(path, "/") {
		parts = strings.Split(path, "/")
	} else if strings.Contains(path, ".") {
		parts = strings.Split(path, ".")
	} else {
		// No separator found, entire path is variable name
		return path, ""
	}

	if len(parts) >= 1 {
		variable = parts[0]
	}
	if len(parts) >= 2 {
		attribute = parts[len(parts)-1]
	}

	return variable, attribute
}

// buildCreateListStatement converts create list statement context to CreateListStmt.
// Grammar: VARIABLE EQUALS CREATE LIST OF qualifiedName
func buildCreateListStatement(ctx parser.ICreateListStatementContext) *ast.CreateListStmt {
	if ctx == nil {
		return nil
	}
	createCtx := ctx.(*parser.CreateListStatementContext)

	stmt := &ast.CreateListStmt{}

	// Get variable name
	if v := createCtx.VARIABLE(); v != nil {
		stmt.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get entity type
	if qn := createCtx.QualifiedName(); qn != nil {
		stmt.EntityType = buildQualifiedName(qn)
	}

	return stmt
}

// buildAddToListStatement converts add to list statement context to AddToListStmt.
// Grammar: ADD VARIABLE TO VARIABLE
func buildAddToListStatement(ctx parser.IAddToListStatementContext) *ast.AddToListStmt {
	if ctx == nil {
		return nil
	}
	addCtx := ctx.(*parser.AddToListStatementContext)

	stmt := &ast.AddToListStmt{}

	// Get both variables
	vars := addCtx.AllVARIABLE()
	if len(vars) >= 1 {
		stmt.Item = strings.TrimPrefix(vars[0].GetText(), "$")
	}
	if len(vars) >= 2 {
		stmt.List = strings.TrimPrefix(vars[1].GetText(), "$")
	}

	return stmt
}

// buildRemoveFromListStatement converts remove from list statement context to RemoveFromListStmt.
// Grammar: REMOVE VARIABLE FROM VARIABLE
func buildRemoveFromListStatement(ctx parser.IRemoveFromListStatementContext) *ast.RemoveFromListStmt {
	if ctx == nil {
		return nil
	}
	removeCtx := ctx.(*parser.RemoveFromListStatementContext)

	stmt := &ast.RemoveFromListStmt{}

	// Get both variables
	vars := removeCtx.AllVARIABLE()
	if len(vars) >= 1 {
		stmt.Item = strings.TrimPrefix(vars[0].GetText(), "$")
	}
	if len(vars) >= 2 {
		stmt.List = strings.TrimPrefix(vars[1].GetText(), "$")
	}

	return stmt
}

// ============================================================================
// Page Actions
// ============================================================================

// buildShowPageStatement converts show page statement context to ShowPageStmt.
// Grammar: SHOW PAGE qualifiedName (LPAREN showPageArgList? RPAREN)? (FOR VARIABLE)? (WITH memberAssignmentList)?
func buildShowPageStatement(ctx parser.IShowPageStatementContext) *ast.ShowPageStmt {
	if ctx == nil {
		return nil
	}
	showCtx := ctx.(*parser.ShowPageStatementContext)

	stmt := &ast.ShowPageStmt{
		Location: "Content", // Default location
	}

	// Get page name
	if qn := showCtx.QualifiedName(); qn != nil {
		stmt.PageName = buildQualifiedName(qn)
	}

	// Get page arguments
	if argList := showCtx.ShowPageArgList(); argList != nil {
		stmt.Arguments = buildShowPageArgList(argList)
	}

	// Get FOR variable (for data grid selection, etc.)
	if showCtx.FOR() != nil {
		if v := showCtx.VARIABLE(); v != nil {
			stmt.ForObject = strings.TrimPrefix(v.GetText(), "$")
		}
	}

	// Get WITH settings (title override, location, etc.)
	if showCtx.WITH() != nil {
		if memberList := showCtx.MemberAssignmentList(); memberList != nil {
			changes := buildMemberAssignmentList(memberList)
			for _, change := range changes {
				switch strings.ToLower(change.Attribute) {
				case "title":
					if litExpr, ok := change.Value.(*ast.LiteralExpr); ok && litExpr.Kind == ast.LiteralString {
						if s, ok := litExpr.Value.(string); ok {
							stmt.Title = s
						}
					}
				case "location":
					if litExpr, ok := change.Value.(*ast.LiteralExpr); ok && litExpr.Kind == ast.LiteralString {
						if s, ok := litExpr.Value.(string); ok {
							stmt.Location = s
						}
					} else if identExpr, ok := change.Value.(*ast.IdentifierExpr); ok {
						stmt.Location = identExpr.Name
					}
				case "modal", "modalform":
					if litExpr, ok := change.Value.(*ast.LiteralExpr); ok && litExpr.Kind == ast.LiteralBoolean {
						if b, ok := litExpr.Value.(bool); ok {
							stmt.ModalForm = b
						}
					}
				}
			}
		}
	}

	return stmt
}

// buildShowPageArgList converts showPageArgList context to ShowPageArg slice.
// Grammar: showPageArg (COMMA showPageArg)*
// showPageArg: VARIABLE EQUALS (VARIABLE | expression) | identifierOrKeyword COLON expression
func buildShowPageArgList(ctx parser.IShowPageArgListContext) []ast.ShowPageArg {
	if ctx == nil {
		return nil
	}
	listCtx := ctx.(*parser.ShowPageArgListContext)
	var args []ast.ShowPageArg

	for _, argCtx := range listCtx.AllShowPageArg() {
		arg := argCtx.(*parser.ShowPageArgContext)
		spa := ast.ShowPageArg{}

		if iok := arg.IdentifierOrKeyword(); iok != nil {
			// Widget-style: Param: $value
			spa.ParamName = identifierOrKeywordText(iok)
			if expr := arg.Expression(); expr != nil {
				spa.Value = buildExpression(expr)
			}
		} else {
			// Canonical: $Param = $value
			vars := arg.AllVARIABLE()
			if len(vars) >= 1 {
				spa.ParamName = strings.TrimPrefix(vars[0].GetText(), "$")
			}
			if len(vars) >= 2 {
				spa.Value = &ast.VariableExpr{Name: strings.TrimPrefix(vars[1].GetText(), "$")}
			} else if expr := arg.Expression(); expr != nil {
				spa.Value = buildExpression(expr)
			}
		}

		args = append(args, spa)
	}

	return args
}

// buildShowMessageStatement converts showMessageStatement context to ShowMessageStmt.
// Grammar: SHOW MESSAGE expression (TYPE identifierOrKeyword)? (OBJECTS LBRACKET expressionList RBRACKET)?
func buildShowMessageStatement(ctx parser.IShowMessageStatementContext) *ast.ShowMessageStmt {
	if ctx == nil {
		return nil
	}
	smCtx := ctx.(*parser.ShowMessageStatementContext)

	stmt := &ast.ShowMessageStmt{
		Type: "Information", // Default message type
	}

	if expr := smCtx.Expression(); expr != nil {
		stmt.Message = buildExpression(expr)
	}

	if id := smCtx.IdentifierOrKeyword(); id != nil {
		stmt.Type = id.GetText()
	}

	// Build template arguments (optional)
	if exprList := smCtx.ExpressionList(); exprList != nil {
		listCtx := exprList.(*parser.ExpressionListContext)
		for _, expr := range listCtx.AllExpression() {
			stmt.TemplateArgs = append(stmt.TemplateArgs, buildExpression(expr))
		}
	}

	return stmt
}

// buildValidationFeedbackStatement converts validationFeedbackStatement context to ValidationFeedbackStmt.
// Grammar: VALIDATION FEEDBACK attributePath MESSAGE expression (OBJECTS LBRACKET expressionList RBRACKET)?
func buildValidationFeedbackStatement(ctx parser.IValidationFeedbackStatementContext) *ast.ValidationFeedbackStmt {
	if ctx == nil {
		return nil
	}
	vfCtx := ctx.(*parser.ValidationFeedbackStatementContext)

	stmt := &ast.ValidationFeedbackStmt{}

	// Build attribute path
	if attrPath := vfCtx.AttributePath(); attrPath != nil {
		stmt.AttributePath = buildAttributePathFromContext(attrPath)
	}

	// Build message expression
	if msgExpr := vfCtx.Expression(); msgExpr != nil {
		stmt.Message = buildExpression(msgExpr)
	}

	// Build template arguments (optional)
	if exprList := vfCtx.ExpressionList(); exprList != nil {
		listCtx := exprList.(*parser.ExpressionListContext)
		for _, expr := range listCtx.AllExpression() {
			stmt.TemplateArgs = append(stmt.TemplateArgs, buildExpression(expr))
		}
	}

	return stmt
}

// buildAttributePathFromContext builds an AttributePathExpr from attributePath context.
// Grammar: VARIABLE ((SLASH | DOT) (IDENTIFIER | qualifiedName))+
// Iterates children in order to preserve the separator (/ vs .) for each segment.
func buildAttributePathFromContext(ctx parser.IAttributePathContext) *ast.AttributePathExpr {
	if ctx == nil {
		return nil
	}
	apCtx := ctx.(*parser.AttributePathContext)

	result := &ast.AttributePathExpr{}

	// Get variable name (first element)
	if v := apCtx.VARIABLE(); v != nil {
		result.Variable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Iterate children in order to capture separator-segment pairs.
	// Pattern: VARIABLE (separator segment)+ where separator is SLASH or DOT
	lastSep := "/"
	for _, child := range apCtx.GetChildren() {
		if tn, ok := child.(antlr.TerminalNode); ok {
			switch tn.GetSymbol().GetTokenType() {
			case parser.MDLParserSLASH:
				lastSep = "/"
			case parser.MDLParserDOT:
				lastSep = "."
			case parser.MDLParserIDENTIFIER:
				name := tn.GetText()
				result.Path = append(result.Path, name)
				result.Segments = append(result.Segments, ast.PathSegment{Name: name, Separator: lastSep})
			}
		} else if qn, ok := child.(parser.IQualifiedNameContext); ok {
			name := qn.GetText()
			result.Path = append(result.Path, name)
			result.Segments = append(result.Segments, ast.PathSegment{Name: name, Separator: lastSep})
		}
	}

	return result
}

// ============================================================================
// REST Call Statements
// ============================================================================

// buildRestCallStatement converts REST CALL statement context to RestCallStmt.
// Grammar: (VARIABLE EQUALS)? REST CALL httpMethod restCallUrl restCallUrlParams?
//
//	restCallHeaderClause* restCallAuthClause? restCallBodyClause?
//	restCallTimeoutClause? restCallReturnsClause onErrorClause?
func buildRestCallStatement(ctx parser.IRestCallStatementContext) *ast.RestCallStmt {
	if ctx == nil {
		return nil
	}
	restCtx := ctx.(*parser.RestCallStatementContext)

	stmt := &ast.RestCallStmt{}

	// Get output variable if present
	if v := restCtx.VARIABLE(); v != nil {
		stmt.OutputVariable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Get HTTP method
	if method := restCtx.HttpMethod(); method != nil {
		methodCtx := method.(*parser.HttpMethodContext)
		if methodCtx.GET() != nil {
			stmt.Method = ast.HttpMethodGet
		} else if methodCtx.POST() != nil {
			stmt.Method = ast.HttpMethodPost
		} else if methodCtx.PUT() != nil {
			stmt.Method = ast.HttpMethodPut
		} else if methodCtx.PATCH() != nil {
			stmt.Method = ast.HttpMethodPatch
		} else if methodCtx.DELETE() != nil {
			stmt.Method = ast.HttpMethodDelete
		}
	}

	// Get URL
	if urlCtx := restCtx.RestCallUrl(); urlCtx != nil {
		urlC := urlCtx.(*parser.RestCallUrlContext)
		if strLit := urlC.STRING_LITERAL(); strLit != nil {
			stmt.URL = &ast.LiteralExpr{
				Kind:  ast.LiteralString,
				Value: unquoteString(strLit.GetText()),
			}
		} else if expr := urlC.Expression(); expr != nil {
			stmt.URL = buildExpression(expr)
		}
	}

	// Get URL template parameters
	if urlParams := restCtx.RestCallUrlParams(); urlParams != nil {
		paramsCtx := urlParams.(*parser.RestCallUrlParamsContext)
		if tplParams := paramsCtx.TemplateParams(); tplParams != nil {
			stmt.URLParams = buildTemplateParams(tplParams)
		}
	}

	// Get headers
	for _, headerClause := range restCtx.AllRestCallHeaderClause() {
		hdrCtx := headerClause.(*parser.RestCallHeaderClauseContext)
		header := ast.RestHeader{}
		if id := hdrCtx.IDENTIFIER(); id != nil {
			header.Name = id.GetText()
		} else if strLit := hdrCtx.STRING_LITERAL(); strLit != nil {
			// Handle quoted header names like 'Content-Type'
			header.Name = unquoteString(strLit.GetText())
		}
		if expr := hdrCtx.Expression(); expr != nil {
			header.Value = buildExpression(expr)
		}
		stmt.Headers = append(stmt.Headers, header)
	}

	// Get auth clause
	if authClause := restCtx.RestCallAuthClause(); authClause != nil {
		authCtx := authClause.(*parser.RestCallAuthClauseContext)
		exprs := authCtx.AllExpression()
		if len(exprs) >= 2 {
			stmt.Auth = &ast.RestAuth{
				Username: buildExpression(exprs[0]),
				Password: buildExpression(exprs[1]),
			}
		}
	}

	// Get body clause
	if bodyClause := restCtx.RestCallBodyClause(); bodyClause != nil {
		bodyCtx := bodyClause.(*parser.RestCallBodyClauseContext)
		body := &ast.RestBody{}

		if bodyCtx.MAPPING() != nil {
			// Export mapping: BODY MAPPING QualifiedName FROM $Variable
			body.Type = ast.RestBodyMapping
			if qn := bodyCtx.QualifiedName(); qn != nil {
				body.MappingName = buildQualifiedName(qn)
			}
			if v := bodyCtx.VARIABLE(); v != nil {
				body.SourceVariable = strings.TrimPrefix(v.GetText(), "$")
			}
		} else {
			// Custom body template
			body.Type = ast.RestBodyCustom
			if strLit := bodyCtx.STRING_LITERAL(); strLit != nil {
				body.Template = &ast.LiteralExpr{
					Kind:  ast.LiteralString,
					Value: unquoteString(strLit.GetText()),
				}
			} else if expr := bodyCtx.Expression(); expr != nil {
				body.Template = buildExpression(expr)
			}
			// Get template parameters
			if tplParams := bodyCtx.TemplateParams(); tplParams != nil {
				body.TemplateParams = buildTemplateParams(tplParams)
			}
		}

		stmt.Body = body
	}

	// Get timeout clause
	if timeoutClause := restCtx.RestCallTimeoutClause(); timeoutClause != nil {
		timeoutCtx := timeoutClause.(*parser.RestCallTimeoutClauseContext)
		if expr := timeoutCtx.Expression(); expr != nil {
			stmt.Timeout = buildExpression(expr)
		}
	}

	// Get returns clause
	if returnsClause := restCtx.RestCallReturnsClause(); returnsClause != nil {
		returnsCtx := returnsClause.(*parser.RestCallReturnsClauseContext)
		result := ast.RestResult{}

		if returnsCtx.STRING_TYPE() != nil {
			result.Type = ast.RestResultString
		} else if returnsCtx.RESPONSE() != nil {
			result.Type = ast.RestResultResponse
		} else if returnsCtx.MAPPING() != nil {
			result.Type = ast.RestResultMapping
			qns := returnsCtx.AllQualifiedName()
			if len(qns) >= 1 {
				result.MappingName = buildQualifiedName(qns[0])
			}
			if len(qns) >= 2 {
				result.ResultEntity = buildQualifiedName(qns[1])
			}
		} else if returnsCtx.NONE() != nil || returnsCtx.NOTHING() != nil {
			result.Type = ast.RestResultNone
		}

		stmt.Result = result
	}

	// Get error handling clause
	if errClause := restCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}

// buildSendRestRequestStatement builds a SendRestRequestStmt from the parser context.
// Syntax: [$Var =] SEND REST REQUEST Module.Service.Operation [BODY $var] [ON ERROR ...]
func buildSendRestRequestStatement(ctx parser.ISendRestRequestStatementContext) *ast.SendRestRequestStmt {
	if ctx == nil {
		return nil
	}
	sendCtx := ctx.(*parser.SendRestRequestStatementContext)

	stmt := &ast.SendRestRequestStmt{}

	// Output variable
	if v := sendCtx.VARIABLE(); v != nil {
		stmt.OutputVariable = strings.TrimPrefix(v.GetText(), "$")
	}

	// Operation qualified name (Module.Service.Operation)
	if qn := sendCtx.QualifiedName(); qn != nil {
		stmt.Operation = buildQualifiedName(qn)
	}

	// Body clause
	if bodyClause := sendCtx.SendRestRequestBodyClause(); bodyClause != nil {
		bc := bodyClause.(*parser.SendRestRequestBodyClauseContext)
		if v := bc.VARIABLE(); v != nil {
			stmt.BodyVariable = strings.TrimPrefix(v.GetText(), "$")
		}
	}

	// Error handling
	if errClause := sendCtx.OnErrorClause(); errClause != nil {
		stmt.ErrorHandling = buildOnErrorClause(errClause)
	}

	return stmt
}
