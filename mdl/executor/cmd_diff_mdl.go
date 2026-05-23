// SPDX-License-Identifier: Apache-2.0

// Package executor - MDL generation functions for diff (statement→text and project→text converters)
package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	entityModel "github.com/mendixlabs/mxcli/mdl/model/entity"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// ============================================================================
// Statement to MDL Converters
// ============================================================================

// entityStmtToMDL converts a CreateEntityStmt to MDL text via the canonical
// EntityModel pipeline (Lift → ToMDL). Both proposed (stmt) and current (gen)
// renderings share the same serializer so diff output is byte-stable.
func entityStmtToMDL(_ *ExecContext, s *ast.CreateEntityStmt) string {
	m, err := entityModel.Lift(s)
	if err != nil {
		return fmt.Sprintf("/* entity lift error: %v */", err)
	}
	return m.ToMDL() + ";\n/"
}

// viewEntityStmtToMDL converts a CreateViewEntityStmt to MDL text
func viewEntityStmtToMDL(ctx *ExecContext, s *ast.CreateViewEntityStmt) string {
	var lines []string

	if s.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+s.Documentation)
		lines = append(lines, " */")
	}

	if s.Position != nil {
		lines = append(lines, fmt.Sprintf("@Position(%d, %d)", s.Position.X, s.Position.Y))
	}

	lines = append(lines, fmt.Sprintf("create view entity %s (", s.Name))

	for i, attr := range s.Attributes {
		typeStr := dataTypeToString(ctx, attr.Type)
		comma := ","
		if i == len(s.Attributes)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  %s: %s%s", attr.Name, typeStr, comma))
	}

	lines = append(lines, ") as (")
	// Indent OQL query
	for line := range strings.SplitSeq(s.Query.RawQuery, "\n") {
		lines = append(lines, "  "+line)
	}
	lines = append(lines, ");")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// enumerationStmtToMDL converts a CreateEnumerationStmt to MDL text
func enumerationStmtToMDL(ctx *ExecContext, s *ast.CreateEnumerationStmt) string {
	var lines []string

	if s.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+s.Documentation)
		lines = append(lines, " */")
	}

	lines = append(lines, fmt.Sprintf("create enumeration %s (", s.Name))

	for i, v := range s.Values {
		comma := ","
		if i == len(s.Values)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  %s '%s'%s", v.Name, v.Caption, comma))
	}

	lines = append(lines, ");")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// associationStmtToMDL converts a CreateAssociationStmt to MDL text
func associationStmtToMDL(ctx *ExecContext, s *ast.CreateAssociationStmt) string {
	var lines []string

	if s.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+s.Documentation)
		lines = append(lines, " */")
	}

	lines = append(lines, fmt.Sprintf("create association %s", s.Name))
	lines = append(lines, fmt.Sprintf("from %s to %s", s.Parent, s.Child))

	assocType := "Reference"
	if s.Type == ast.AssocReferenceSet {
		assocType = "ReferenceSet"
	}
	lines = append(lines, fmt.Sprintf("type %s", assocType))

	owner := "Default"
	if s.Owner == ast.OwnerBoth {
		owner = "Both"
	}
	lines = append(lines, fmt.Sprintf("owner %s", owner))

	deleteBehavior := "DELETE_BUT_KEEP_REFERENCES"
	switch s.DeleteBehavior {
	case ast.DeleteCascade:
		deleteBehavior = "DELETE_CASCADE"
	case ast.DeleteIfNoReferences:
		deleteBehavior = "DELETE_IF_NO_REFERENCES"
	}
	lines = append(lines, fmt.Sprintf("delete_behavior %s;", deleteBehavior))
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// microflowStmtToMDL converts a CreateMicroflowStmt to MDL text
func microflowStmtToMDL(ctx *ExecContext, s *ast.CreateMicroflowStmt) string {
	var lines []string

	// Annotations
	if s.Excluded {
		lines = append(lines, "@excluded")
	}

	// Documentation
	if s.Documentation != "" {
		lines = append(lines, "/**")
		for docLine := range strings.SplitSeq(s.Documentation, "\n") {
			lines = append(lines, " * "+docLine)
		}
		lines = append(lines, " */")
	}

	// CREATE [OR MODIFY] MICROFLOW header with parameters
	header := "create"
	if s.CreateOrModify {
		header = "create or modify"
	}
	if len(s.Parameters) > 0 {
		lines = append(lines, fmt.Sprintf("%s microflow %s (", header, s.Name))
		for i, param := range s.Parameters {
			paramType := dataTypeToString(ctx, param.Type)
			comma := ","
			if i == len(s.Parameters)-1 {
				comma = ""
			}
			lines = append(lines, fmt.Sprintf("  $%s: %s%s", param.Name, paramType, comma))
		}
		lines = append(lines, ")")
	} else {
		lines = append(lines, fmt.Sprintf("%s microflow %s ()", header, s.Name))
	}

	// Folder
	if s.Folder != "" {
		lines = append(lines, fmt.Sprintf("folder '%s'", s.Folder))
	}

	// Comment
	if s.Comment != "" {
		lines = append(lines, fmt.Sprintf("comment '%s'", s.Comment))
	}

	// Return type
	if s.ReturnType != nil {
		returnType := dataTypeToString(ctx, s.ReturnType.Type)
		if returnType != "Void" && returnType != "" {
			returnLine := fmt.Sprintf("returns %s", returnType)
			if s.ReturnType.Variable != "" {
				returnLine += fmt.Sprintf(" as $%s", s.ReturnType.Variable)
			}
			lines = append(lines, returnLine)
		}
	}

	// BEGIN block
	lines = append(lines, "begin")

	// Body statements
	for _, stmt := range s.Body {
		stmtLines := microflowStatementToMDL(ctx, stmt, 1)
		lines = append(lines, stmtLines...)
	}

	lines = append(lines, "end;")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// nanoflowStmtToMDL converts a CreateNanoflowStmt to MDL text
func nanoflowStmtToMDL(ctx *ExecContext, s *ast.CreateNanoflowStmt) string {
	var lines []string

	// Annotations
	if s.Excluded {
		lines = append(lines, "@excluded")
	}

	// Documentation
	if s.Documentation != "" {
		lines = append(lines, "/**")
		for docLine := range strings.SplitSeq(s.Documentation, "\n") {
			lines = append(lines, " * "+docLine)
		}
		lines = append(lines, " */")
	}

	// CREATE [OR MODIFY] NANOFLOW header with parameters
	header := "create"
	if s.CreateOrModify {
		header = "create or modify"
	}
	if len(s.Parameters) > 0 {
		lines = append(lines, fmt.Sprintf("%s nanoflow %s (", header, s.Name))
		for i, param := range s.Parameters {
			paramType := dataTypeToString(ctx, param.Type)
			comma := ","
			if i == len(s.Parameters)-1 {
				comma = ""
			}
			lines = append(lines, fmt.Sprintf("  $%s: %s%s", param.Name, paramType, comma))
		}
		lines = append(lines, ")")
	} else {
		lines = append(lines, fmt.Sprintf("%s nanoflow %s ()", header, s.Name))
	}

	// Folder
	if s.Folder != "" {
		lines = append(lines, fmt.Sprintf("folder '%s'", s.Folder))
	}

	// Comment
	if s.Comment != "" {
		lines = append(lines, fmt.Sprintf("comment '%s'", s.Comment))
	}

	// Return type
	if s.ReturnType != nil {
		returnType := dataTypeToString(ctx, s.ReturnType.Type)
		if returnType != "Void" && returnType != "" {
			returnLine := fmt.Sprintf("returns %s", returnType)
			if s.ReturnType.Variable != "" {
				returnLine += fmt.Sprintf(" as $%s", s.ReturnType.Variable)
			}
			lines = append(lines, returnLine)
		}
	}

	// BEGIN block
	lines = append(lines, "begin")

	// Body statements
	for _, stmt := range s.Body {
		stmtLines := microflowStatementToMDL(ctx, stmt, 1)
		lines = append(lines, stmtLines...)
	}

	lines = append(lines, "end;")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// microflowStatementToMDL converts a microflow statement to MDL lines
func microflowStatementToMDL(ctx *ExecContext, stmt ast.MicroflowStatement, indent int) []string {
	indentStr := strings.Repeat("  ", indent)
	var lines []string

	switch s := stmt.(type) {
	case *ast.DeclareStmt:
		typeStr := dataTypeToString(ctx, s.Type)
		initVal := "empty"
		if s.InitialValue != nil {
			initVal = diffExpressionToString(ctx, s.InitialValue)
		}
		lines = append(lines, fmt.Sprintf("%sdeclare $%s %s = %s;", indentStr, s.Variable, typeStr, initVal))

	case *ast.MfSetStmt:
		lines = append(lines, fmt.Sprintf("%sset $%s = %s;", indentStr, s.Target, diffExpressionToString(ctx, s.Value)))

	case *ast.ReturnStmt:
		if s.Value != nil {
			lines = append(lines, fmt.Sprintf("%sreturn %s;", indentStr, diffExpressionToString(ctx, s.Value)))
		} else {
			lines = append(lines, fmt.Sprintf("%sreturn;", indentStr))
		}

	case *ast.CreateObjectStmt:
		if len(s.Changes) > 0 {
			var members []string
			for _, c := range s.Changes {
				members = append(members, fmt.Sprintf("%s = %s", c.Attribute, diffExpressionToString(ctx, c.Value)))
			}
			lines = append(lines, fmt.Sprintf("%s$%s = create %s (%s);", indentStr, s.Variable, s.EntityType, strings.Join(members, ", ")))
		} else {
			lines = append(lines, fmt.Sprintf("%s$%s = create %s;", indentStr, s.Variable, s.EntityType))
		}

	case *ast.ChangeObjectStmt:
		if len(s.Changes) > 0 {
			var members []string
			for _, c := range s.Changes {
				members = append(members, fmt.Sprintf("%s = %s", c.Attribute, diffExpressionToString(ctx, c.Value)))
			}
			lines = append(lines, fmt.Sprintf("%schange $%s (%s);", indentStr, s.Variable, strings.Join(members, ", ")))
		} else {
			lines = append(lines, fmt.Sprintf("%schange $%s;", indentStr, s.Variable))
		}

	case *ast.MfCommitStmt:
		suffix := ""
		if s.WithEvents {
			suffix += " with events"
		}
		if s.RefreshInClient {
			suffix += " refresh"
		}
		lines = append(lines, fmt.Sprintf("%scommit $%s%s;", indentStr, s.Variable, suffix))

	case *ast.DeleteObjectStmt:
		lines = append(lines, fmt.Sprintf("%sdelete $%s;", indentStr, s.Variable))

	case *ast.RetrieveStmt:
		var stmt string
		if s.StartVariable != "" {
			stmt = fmt.Sprintf("%sretrieve $%s from $%s/%s", indentStr, s.Variable, s.StartVariable, s.Source)
		} else {
			stmt = fmt.Sprintf("%sretrieve $%s from %s", indentStr, s.Variable, s.Source)
		}
		if s.Where != nil {
			stmt += fmt.Sprintf("\n%s    where %s", indentStr, diffExpressionToString(ctx, s.Where))
		}
		if s.Limit != "" {
			stmt += fmt.Sprintf("\n%s    limit %s", indentStr, s.Limit)
		}
		lines = append(lines, stmt+";")

	case *ast.IfStmt:
		lines = append(lines, fmt.Sprintf("%sif %s then", indentStr, diffExpressionToString(ctx, s.Condition)))
		for _, thenStmt := range s.ThenBody {
			lines = append(lines, microflowStatementToMDL(ctx, thenStmt, indent+1)...)
		}
		if s.HasElse || len(s.ElseBody) > 0 {
			lines = append(lines, indentStr+"else")
			for _, elseStmt := range s.ElseBody {
				lines = append(lines, microflowStatementToMDL(ctx, elseStmt, indent+1)...)
			}
		}
		lines = append(lines, indentStr+"end if;")

	case *ast.EnumSplitStmt:
		lines = append(lines, fmt.Sprintf("%scase $%s", indentStr, s.Variable))
		for _, c := range s.Cases {
			lines = append(lines, fmt.Sprintf("%s  when %s then", indentStr, formatEnumSplitCaseValues(enumSplitCaseValues(c))))
			for _, caseStmt := range c.Body {
				lines = append(lines, microflowStatementToMDL(ctx, caseStmt, indent+1)...)
			}
		}
		if len(s.ElseBody) > 0 {
			lines = append(lines, indentStr+"  else")
			for _, elseStmt := range s.ElseBody {
				lines = append(lines, microflowStatementToMDL(ctx, elseStmt, indent+1)...)
			}
		}
		lines = append(lines, indentStr+"end case;")

	case *ast.InheritanceSplitStmt:
		lines = append(lines, fmt.Sprintf("%ssplit type $%s", indentStr, s.Variable))
		for _, c := range s.Cases {
			lines = append(lines, fmt.Sprintf("%scase %s", indentStr, c.Entity.String()))
			for _, caseStmt := range c.Body {
				lines = append(lines, microflowStatementToMDL(ctx, caseStmt, indent+1)...)
			}
		}
		if len(s.ElseBody) > 0 {
			lines = append(lines, indentStr+"else")
			for _, elseStmt := range s.ElseBody {
				lines = append(lines, microflowStatementToMDL(ctx, elseStmt, indent+1)...)
			}
		}
		lines = append(lines, indentStr+"end split;")

	case *ast.CastObjectStmt:
		if s.ObjectVariable == "" {
			lines = append(lines, fmt.Sprintf("%scast $%s;", indentStr, s.OutputVariable))
		} else {
			lines = append(lines, fmt.Sprintf("%s$%s = cast $%s;", indentStr, s.OutputVariable, s.ObjectVariable))
		}

	case *ast.LoopStmt:
		lines = append(lines, fmt.Sprintf("%sloop $%s in $%s", indentStr, s.LoopVariable, s.ListVariable))
		for _, bodyStmt := range s.Body {
			lines = append(lines, microflowStatementToMDL(ctx, bodyStmt, indent+1)...)
		}
		lines = append(lines, indentStr+"end loop;")

	case *ast.LogStmt:
		nodeStr := defaultLogNodeExpression
		if s.Node != nil {
			nodeStr = diffExpressionToString(ctx, s.Node)
		}
		msgStr := diffExpressionToString(ctx, s.Message)
		stmt := fmt.Sprintf("%slog %s node %s %s", indentStr, strings.ToLower(s.Level.String()), nodeStr, msgStr)
		if len(s.Template) > 0 {
			var params []string
			for _, p := range s.Template {
				params = append(params, fmt.Sprintf("{%d} = %s", p.Index, diffExpressionToString(ctx, p.Value)))
			}
			stmt += fmt.Sprintf(" with (%s)", strings.Join(params, ", "))
		}
		lines = append(lines, stmt+";")

	case *ast.CallMicroflowStmt:
		var params []string
		for _, arg := range s.Arguments {
			params = append(params, fmt.Sprintf("%s = %s", arg.Name, diffExpressionToString(ctx, arg.Value)))
		}
		paramStr := strings.Join(params, ", ")
		if s.OutputVariable != "" {
			lines = append(lines, fmt.Sprintf("%s$%s = call microflow %s(%s);", indentStr, s.OutputVariable, s.MicroflowName, paramStr))
		} else {
			lines = append(lines, fmt.Sprintf("%scall microflow %s(%s);", indentStr, s.MicroflowName, paramStr))
		}

	case *ast.CallNanoflowStmt:
		var params []string
		for _, arg := range s.Arguments {
			params = append(params, fmt.Sprintf("%s = %s", arg.Name, diffExpressionToString(ctx, arg.Value)))
		}
		paramStr := strings.Join(params, ", ")
		if s.OutputVariable != "" {
			lines = append(lines, fmt.Sprintf("%s$%s = call nanoflow %s(%s);", indentStr, s.OutputVariable, s.NanoflowName, paramStr))
		} else {
			lines = append(lines, fmt.Sprintf("%scall nanoflow %s(%s);", indentStr, s.NanoflowName, paramStr))
		}

	case *ast.BreakStmt:
		lines = append(lines, indentStr+"break;")

	case *ast.ContinueStmt:
		lines = append(lines, indentStr+"continue;")
	}

	return lines
}

// ============================================================================
// Project to MDL Converters
// ============================================================================

// entityToMDLGen converts a gen-typed project entity to MDL text via the
// canonical EntityModel pipeline (Hydrate → ToMDL).
func entityToMDLGen(_ *ExecContext, moduleName string, entity *genDm.Entity) string {
	m, _, err := entityModel.Hydrate(moduleName, entity)
	if err != nil {
		return fmt.Sprintf("/* entity hydrate error: %v */", err)
	}
	return m.ToMDL() + ";\n/"
}

// viewEntityFromProjectToMDLGen converts a gen-typed view entity to MDL.
func viewEntityFromProjectToMDLGen(ctx *ExecContext, moduleName string, entity *genDm.Entity) string {
	var lines []string

	if entity.Documentation() != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+entity.Documentation())
		lines = append(lines, " */")
	}

	if loc := entity.Location(); loc != "" {
		if x, y, ok := parseLocationBSON(loc); ok {
			lines = append(lines, fmt.Sprintf("@Position(%d, %d)", x, y))
		}
	}
	lines = append(lines, fmt.Sprintf("create view entity %s.%s (", moduleName, entity.Name()))

	attrs := entity.AttributesItems()
	for i, item := range attrs {
		attr, ok := item.(*genDm.Attribute)
		if !ok {
			continue
		}
		typeStr := formatAttributeTypeGen(attr.Type())
		comma := ","
		if i == len(attrs)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  %s: %s%s", attr.Name(), typeStr, comma))
	}

	lines = append(lines, ") as (")
	if src, ok := entity.Source().(*genDm.OqlViewEntitySource); ok && src.Oql() != "" {
		for line := range strings.SplitSeq(src.Oql(), "\n") {
			lines = append(lines, "  "+line)
		}
	}
	lines = append(lines, ");")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// enumerationToMDL converts a project enumeration to MDL text
func enumerationToMDL(ctx *ExecContext, moduleName string, enum *model.Enumeration) string {
	var lines []string

	if enum.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+enum.Documentation)
		lines = append(lines, " */")
	}

	lines = append(lines, fmt.Sprintf("create enumeration %s.%s (", moduleName, enum.Name))

	for i, v := range enum.Values {
		comma := ","
		if i == len(enum.Values)-1 {
			comma = ""
		}
		caption := ""
		if v.Caption != nil {
			caption = v.Caption.GetTranslation("en_US")
		}
		lines = append(lines, fmt.Sprintf("  %s '%s'%s", v.Name, caption, comma))
	}

	lines = append(lines, ");")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// associationToMDLGen converts a gen-typed project association to MDL.
func associationToMDLGen(ctx *ExecContext, moduleName string, assoc *genDm.Association, dm *genDm.DomainModel) string {
	var lines []string

	// Build entity name map
	entityNames := make(map[model.ID]string)
	for _, item := range dm.EntitiesItems() {
		entity, ok := item.(*genDm.Entity)
		if !ok {
			continue
		}
		entityNames[model.ID(entity.ID())] = entity.Name()
	}

	if assoc.Documentation() != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+assoc.Documentation())
		lines = append(lines, " */")
	}

	fromEntity := entityNames[model.ID(assoc.ParentRefID())]
	toEntity := entityNames[model.ID(assoc.ChildRefID())]

	lines = append(lines, fmt.Sprintf("create association %s.%s", moduleName, assoc.Name()))
	lines = append(lines, fmt.Sprintf("from %s.%s to %s.%s", moduleName, fromEntity, moduleName, toEntity))

	assocType := "Reference"
	if assoc.Type() == "ReferenceSet" {
		assocType = "ReferenceSet"
	}
	lines = append(lines, fmt.Sprintf("type %s", assocType))

	owner := "Default"
	if assoc.Owner() == "Both" {
		owner = "Both"
	}
	lines = append(lines, fmt.Sprintf("owner %s", owner))

	deleteBehavior := "DELETE_BUT_KEEP_REFERENCES"
	if dbe, ok := assoc.DeleteBehavior().(*genDm.AssociationDeleteBehavior); ok && dbe != nil {
		switch dbe.ChildDeleteBehavior() {
		case "DeleteMeAndReferences":
			deleteBehavior = "DELETE_CASCADE"
		case "DeleteMeIfNoReferences":
			deleteBehavior = "DELETE_IF_NO_REFERENCES"
		}
	}
	lines = append(lines, fmt.Sprintf("delete_behavior %s;", deleteBehavior))
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// crossAssociationToMDLGen converts a gen-typed cross-association to MDL.
func crossAssociationToMDLGen(ctx *ExecContext, moduleName string, assoc *genDm.CrossAssociation, pairs []DomainModelGenWithContainer) string {
	var lines []string

	entityNames := make(map[model.ID]string)
	moduleNames := make(map[model.ID]string)
	if mods, err := ctx.Backend.ListModules(); err == nil {
		for _, m := range mods {
			moduleNames[m.ID] = m.Name
		}
	}
	for _, pair := range pairs {
		if pair.DM == nil {
			continue
		}
		modName := moduleNames[pair.ContainerID]
		for _, item := range pair.DM.EntitiesItems() {
			entity, ok := item.(*genDm.Entity)
			if !ok {
				continue
			}
			entityNames[model.ID(entity.ID())] = modName + "." + entity.Name()
		}
	}

	if assoc.Documentation() != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+assoc.Documentation())
		lines = append(lines, " */")
	}

	fromEntity := entityNames[model.ID(assoc.ParentRefID())]
	if fromEntity == "" {
		fromEntity = string(assoc.ParentRefID())
	}
	toEntity := assoc.ChildQualifiedName()

	lines = append(lines, fmt.Sprintf("create association %s.%s", moduleName, assoc.Name()))
	lines = append(lines, fmt.Sprintf("from %s to %s", fromEntity, toEntity))

	assocType := "Reference"
	if assoc.Type() == "ReferenceSet" {
		assocType = "ReferenceSet"
	}
	lines = append(lines, fmt.Sprintf("type %s", assocType))

	owner := "Default"
	if assoc.Owner() == "Both" {
		owner = "Both"
	}
	lines = append(lines, fmt.Sprintf("owner %s", owner))

	deleteBehavior := "DELETE_BUT_KEEP_REFERENCES"
	if dbe, ok := assoc.DeleteBehavior().(*genDm.AssociationDeleteBehavior); ok && dbe != nil {
		switch dbe.ChildDeleteBehavior() {
		case "DeleteMeAndReferences":
			deleteBehavior = "DELETE_CASCADE"
		case "DeleteMeIfNoReferences":
			deleteBehavior = "DELETE_IF_NO_REFERENCES"
		}
	}
	lines = append(lines, fmt.Sprintf("delete_behavior %s;", deleteBehavior))
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// ============================================================================
// Helper Functions
// ============================================================================

// dataTypeToString converts a DataType to its string representation
func dataTypeToString(_ *ExecContext, dt ast.DataType) string {
	switch dt.Kind {
	case ast.TypeString:
		if dt.Length > 0 {
			return fmt.Sprintf("String(%d)", dt.Length)
		}
		return "String"
	case ast.TypeInteger:
		return "Integer"
	case ast.TypeLong:
		return "Long"
	case ast.TypeDecimal:
		return "Decimal"
	case ast.TypeBoolean:
		return "Boolean"
	case ast.TypeDateTime:
		return "DateTime"
	case ast.TypeDate:
		return "Date"
	case ast.TypeAutoNumber:
		return "AutoNumber"
	case ast.TypeBinary:
		return "Binary"
	case ast.TypeEnumeration:
		if dt.EnumRef != nil {
			return fmt.Sprintf("Enumeration(%s)", dt.EnumRef.String())
		}
		return "Enumeration"
	case ast.TypeEntity:
		if dt.EntityRef != nil {
			return dt.EntityRef.String()
		}
		return "Object"
	case ast.TypeListOf:
		if dt.EntityRef != nil {
			return fmt.Sprintf("List of %s", dt.EntityRef.String())
		}
		return "List"
	case ast.TypeVoid:
		return "Void"
	default:
		return "Unknown"
	}
}

// diffExpressionToString converts an expression to its string representation for diff output
func diffExpressionToString(ctx *ExecContext, expr ast.Expression) string {
	if expr == nil {
		return "empty"
	}

	switch ex := expr.(type) {
	case *ast.LiteralExpr:
		if ex.Kind == ast.LiteralString {
			return fmt.Sprintf("'%v'", ex.Value)
		}
		if ex.Kind == ast.LiteralEmpty {
			return "empty"
		}
		if ex.Kind == ast.LiteralNull {
			return "null"
		}
		return fmt.Sprintf("%v", ex.Value)
	case *ast.VariableExpr:
		return "$" + ex.Name
	case *ast.AttributePathExpr:
		return "$" + ex.Variable + "/" + strings.Join(ex.Path, "/")
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", diffExpressionToString(ctx, ex.Left), ex.Operator, diffExpressionToString(ctx, ex.Right))
	case *ast.UnaryExpr:
		return fmt.Sprintf("%s%s", ex.Operator, diffExpressionToString(ctx, ex.Operand))
	case *ast.FunctionCallExpr:
		var args []string
		for _, arg := range ex.Arguments {
			args = append(args, diffExpressionToString(ctx, arg))
		}
		return fmt.Sprintf("%s(%s)", ex.Name, strings.Join(args, ", "))
	case *ast.TokenExpr:
		return fmt.Sprintf("[%%%s%%]", ex.Token)
	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", diffExpressionToString(ctx, ex.Inner))
	case *ast.QualifiedNameExpr:
		return ex.QualifiedName.String()
	case *ast.ConstantRefExpr:
		return "@" + ex.QualifiedName.String()
	default:
		return fmt.Sprintf("%v", expr)
	}
}
