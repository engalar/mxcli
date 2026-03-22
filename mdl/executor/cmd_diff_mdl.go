// SPDX-License-Identifier: Apache-2.0

// Package executor - MDL generation functions for diff (statement→text and project→text converters)
package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// ============================================================================
// Statement to MDL Converters
// ============================================================================

// entityStmtToMDL converts a CreateEntityStmt to MDL text
func (e *Executor) entityStmtToMDL(s *ast.CreateEntityStmt) string {
	var lines []string

	// Documentation
	if s.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+s.Documentation)
		lines = append(lines, " */")
	}

	// Position annotation
	if s.Position != nil {
		lines = append(lines, fmt.Sprintf("@Position(%d, %d)", s.Position.X, s.Position.Y))
	}

	// Entity type
	entityType := s.Kind.String()
	lines = append(lines, fmt.Sprintf("CREATE %s ENTITY %s (", entityType, s.Name))

	// Attributes
	for i, attr := range s.Attributes {
		// Attribute documentation
		if attr.Documentation != "" {
			lines = append(lines, fmt.Sprintf("  /** %s */", attr.Documentation))
		}

		typeStr := e.dataTypeToString(attr.Type)
		constraints := ""

		if attr.NotNull {
			constraints += " NOT NULL"
			if attr.NotNullError != "" {
				constraints += fmt.Sprintf(" ERROR '%s'", attr.NotNullError)
			}
		}
		if attr.Unique {
			constraints += " UNIQUE"
			if attr.UniqueError != "" {
				constraints += fmt.Sprintf(" ERROR '%s'", attr.UniqueError)
			}
		}
		if attr.HasDefault {
			defaultVal := fmt.Sprintf("%v", attr.DefaultValue)
			if attr.Type.Kind == ast.TypeString {
				defaultVal = fmt.Sprintf("'%s'", attr.DefaultValue)
			}
			constraints += fmt.Sprintf(" DEFAULT %s", defaultVal)
		}

		comma := ","
		if i == len(s.Attributes)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  %s: %s%s%s", attr.Name, typeStr, constraints, comma))
	}

	lines = append(lines, ")")

	// Indexes
	for _, idx := range s.Indexes {
		var cols []string
		for _, col := range idx.Columns {
			colStr := col.Name
			if col.Descending {
				colStr += " DESC"
			}
			cols = append(cols, colStr)
		}
		lines = append(lines, fmt.Sprintf("INDEX (%s)", strings.Join(cols, ", ")))
	}

	lines = append(lines, ";")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// viewEntityStmtToMDL converts a CreateViewEntityStmt to MDL text
func (e *Executor) viewEntityStmtToMDL(s *ast.CreateViewEntityStmt) string {
	var lines []string

	if s.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+s.Documentation)
		lines = append(lines, " */")
	}

	if s.Position != nil {
		lines = append(lines, fmt.Sprintf("@Position(%d, %d)", s.Position.X, s.Position.Y))
	}

	lines = append(lines, fmt.Sprintf("CREATE VIEW ENTITY %s (", s.Name))

	for i, attr := range s.Attributes {
		typeStr := e.dataTypeToString(attr.Type)
		comma := ","
		if i == len(s.Attributes)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  %s: %s%s", attr.Name, typeStr, comma))
	}

	lines = append(lines, ") AS (")
	// Indent OQL query
	for line := range strings.SplitSeq(s.Query.RawQuery, "\n") {
		lines = append(lines, "  "+line)
	}
	lines = append(lines, ");")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// enumerationStmtToMDL converts a CreateEnumerationStmt to MDL text
func (e *Executor) enumerationStmtToMDL(s *ast.CreateEnumerationStmt) string {
	var lines []string

	if s.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+s.Documentation)
		lines = append(lines, " */")
	}

	lines = append(lines, fmt.Sprintf("CREATE ENUMERATION %s (", s.Name))

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
func (e *Executor) associationStmtToMDL(s *ast.CreateAssociationStmt) string {
	var lines []string

	if s.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+s.Documentation)
		lines = append(lines, " */")
	}

	lines = append(lines, fmt.Sprintf("CREATE ASSOCIATION %s", s.Name))
	lines = append(lines, fmt.Sprintf("FROM %s TO %s", s.Parent, s.Child))

	assocType := "Reference"
	if s.Type == ast.AssocReferenceSet {
		assocType = "ReferenceSet"
	}
	lines = append(lines, fmt.Sprintf("TYPE %s", assocType))

	owner := "Default"
	if s.Owner == ast.OwnerBoth {
		owner = "Both"
	}
	lines = append(lines, fmt.Sprintf("OWNER %s", owner))

	deleteBehavior := "DELETE_BUT_KEEP_REFERENCES"
	switch s.DeleteBehavior {
	case ast.DeleteCascade:
		deleteBehavior = "DELETE_CASCADE"
	case ast.DeleteIfNoReferences:
		deleteBehavior = "DELETE_IF_NO_REFERENCES"
	}
	lines = append(lines, fmt.Sprintf("DELETE_BEHAVIOR %s;", deleteBehavior))
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// microflowStmtToMDL converts a CreateMicroflowStmt to MDL text
func (e *Executor) microflowStmtToMDL(s *ast.CreateMicroflowStmt) string {
	var lines []string

	// Documentation
	if s.Documentation != "" {
		lines = append(lines, "/**")
		for docLine := range strings.SplitSeq(s.Documentation, "\n") {
			lines = append(lines, " * "+docLine)
		}
		lines = append(lines, " */")
	}

	// CREATE MICROFLOW header with parameters
	if len(s.Parameters) > 0 {
		lines = append(lines, fmt.Sprintf("CREATE MICROFLOW %s (", s.Name))
		for i, param := range s.Parameters {
			paramType := e.dataTypeToString(param.Type)
			comma := ","
			if i == len(s.Parameters)-1 {
				comma = ""
			}
			lines = append(lines, fmt.Sprintf("  $%s: %s%s", param.Name, paramType, comma))
		}
		lines = append(lines, ")")
	} else {
		lines = append(lines, fmt.Sprintf("CREATE MICROFLOW %s ()", s.Name))
	}

	// Return type
	if s.ReturnType != nil {
		returnType := e.dataTypeToString(s.ReturnType.Type)
		if returnType != "Void" && returnType != "" {
			returnLine := fmt.Sprintf("RETURNS %s", returnType)
			if s.ReturnType.Variable != "" {
				returnLine += fmt.Sprintf(" AS $%s", s.ReturnType.Variable)
			}
			lines = append(lines, returnLine)
		}
	}

	// BEGIN block
	lines = append(lines, "BEGIN")

	// Body statements
	for _, stmt := range s.Body {
		stmtLines := e.microflowStatementToMDL(stmt, 1)
		lines = append(lines, stmtLines...)
	}

	lines = append(lines, "END;")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// microflowStatementToMDL converts a microflow statement to MDL lines
func (e *Executor) microflowStatementToMDL(stmt ast.MicroflowStatement, indent int) []string {
	indentStr := strings.Repeat("  ", indent)
	var lines []string

	switch s := stmt.(type) {
	case *ast.DeclareStmt:
		typeStr := e.dataTypeToString(s.Type)
		initVal := "empty"
		if s.InitialValue != nil {
			initVal = e.expressionToString(s.InitialValue)
		}
		lines = append(lines, fmt.Sprintf("%sDECLARE $%s %s = %s;", indentStr, s.Variable, typeStr, initVal))

	case *ast.MfSetStmt:
		lines = append(lines, fmt.Sprintf("%sSET $%s = %s;", indentStr, s.Target, e.expressionToString(s.Value)))

	case *ast.ReturnStmt:
		if s.Value != nil {
			lines = append(lines, fmt.Sprintf("%sRETURN %s;", indentStr, e.expressionToString(s.Value)))
		} else {
			lines = append(lines, fmt.Sprintf("%sRETURN;", indentStr))
		}

	case *ast.CreateObjectStmt:
		if len(s.Changes) > 0 {
			var members []string
			for _, c := range s.Changes {
				members = append(members, fmt.Sprintf("%s = %s", c.Attribute, e.expressionToString(c.Value)))
			}
			lines = append(lines, fmt.Sprintf("%s$%s = CREATE %s (%s);", indentStr, s.Variable, s.EntityType, strings.Join(members, ", ")))
		} else {
			lines = append(lines, fmt.Sprintf("%s$%s = CREATE %s;", indentStr, s.Variable, s.EntityType))
		}

	case *ast.ChangeObjectStmt:
		if len(s.Changes) > 0 {
			var members []string
			for _, c := range s.Changes {
				members = append(members, fmt.Sprintf("%s = %s", c.Attribute, e.expressionToString(c.Value)))
			}
			lines = append(lines, fmt.Sprintf("%sCHANGE $%s (%s);", indentStr, s.Variable, strings.Join(members, ", ")))
		} else {
			lines = append(lines, fmt.Sprintf("%sCHANGE $%s;", indentStr, s.Variable))
		}

	case *ast.MfCommitStmt:
		suffix := ""
		if s.WithEvents {
			suffix += " WITH EVENTS"
		}
		if s.RefreshInClient {
			suffix += " REFRESH"
		}
		lines = append(lines, fmt.Sprintf("%sCOMMIT $%s%s;", indentStr, s.Variable, suffix))

	case *ast.DeleteObjectStmt:
		lines = append(lines, fmt.Sprintf("%sDELETE $%s;", indentStr, s.Variable))

	case *ast.RetrieveStmt:
		var stmt string
		if s.StartVariable != "" {
			stmt = fmt.Sprintf("%sRETRIEVE $%s FROM $%s/%s", indentStr, s.Variable, s.StartVariable, s.Source)
		} else {
			stmt = fmt.Sprintf("%sRETRIEVE $%s FROM %s", indentStr, s.Variable, s.Source)
		}
		if s.Where != nil {
			stmt += fmt.Sprintf("\n%s    WHERE %s", indentStr, e.expressionToString(s.Where))
		}
		if s.Limit != "" {
			stmt += fmt.Sprintf("\n%s    LIMIT %s", indentStr, s.Limit)
		}
		lines = append(lines, stmt+";")

	case *ast.IfStmt:
		lines = append(lines, fmt.Sprintf("%sIF %s THEN", indentStr, e.expressionToString(s.Condition)))
		for _, thenStmt := range s.ThenBody {
			lines = append(lines, e.microflowStatementToMDL(thenStmt, indent+1)...)
		}
		if len(s.ElseBody) > 0 {
			lines = append(lines, indentStr+"ELSE")
			for _, elseStmt := range s.ElseBody {
				lines = append(lines, e.microflowStatementToMDL(elseStmt, indent+1)...)
			}
		}
		lines = append(lines, indentStr+"END IF;")

	case *ast.LoopStmt:
		lines = append(lines, fmt.Sprintf("%sLOOP $%s IN $%s", indentStr, s.LoopVariable, s.ListVariable))
		for _, bodyStmt := range s.Body {
			lines = append(lines, e.microflowStatementToMDL(bodyStmt, indent+1)...)
		}
		lines = append(lines, indentStr+"END LOOP;")

	case *ast.LogStmt:
		nodeStr := s.Node
		if !strings.HasPrefix(nodeStr, "'") {
			nodeStr = "'" + nodeStr + "'"
		}
		msgStr := e.expressionToString(s.Message)
		stmt := fmt.Sprintf("%sLOG %s NODE %s %s", indentStr, strings.ToUpper(s.Level.String()), nodeStr, msgStr)
		if len(s.Template) > 0 {
			var params []string
			for _, p := range s.Template {
				params = append(params, fmt.Sprintf("{%d} = %s", p.Index, e.expressionToString(p.Value)))
			}
			stmt += fmt.Sprintf(" WITH (%s)", strings.Join(params, ", "))
		}
		lines = append(lines, stmt+";")

	case *ast.CallMicroflowStmt:
		var params []string
		for _, arg := range s.Arguments {
			params = append(params, fmt.Sprintf("%s = %s", arg.Name, e.expressionToString(arg.Value)))
		}
		paramStr := strings.Join(params, ", ")
		if s.OutputVariable != "" {
			lines = append(lines, fmt.Sprintf("%s$%s = CALL MICROFLOW %s(%s);", indentStr, s.OutputVariable, s.MicroflowName, paramStr))
		} else {
			lines = append(lines, fmt.Sprintf("%sCALL MICROFLOW %s(%s);", indentStr, s.MicroflowName, paramStr))
		}

	case *ast.BreakStmt:
		lines = append(lines, indentStr+"BREAK;")

	case *ast.ContinueStmt:
		lines = append(lines, indentStr+"CONTINUE;")
	}

	return lines
}

// ============================================================================
// Project to MDL Converters
// ============================================================================

// entityToMDL converts a project entity to MDL text
func (e *Executor) entityToMDL(moduleName string, entity *domainmodel.Entity, dm *domainmodel.DomainModel) string {
	var lines []string

	// Documentation
	if entity.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+entity.Documentation)
		lines = append(lines, " */")
	}

	// Position
	lines = append(lines, fmt.Sprintf("@Position(%d, %d)", entity.Location.X, entity.Location.Y))

	// Entity type
	entityType := "PERSISTENT"
	if strings.Contains(entity.Source, "OqlView") {
		entityType = "VIEW"
	} else if !entity.Persistable {
		entityType = "NON-PERSISTENT"
	}

	lines = append(lines, fmt.Sprintf("CREATE %s ENTITY %s.%s (", entityType, moduleName, entity.Name))

	// Build validation rules map
	validationsByAttr := make(map[model.ID][]*domainmodel.ValidationRule)
	validationsByName := make(map[string][]*domainmodel.ValidationRule)
	for _, vr := range entity.ValidationRules {
		validationsByAttr[vr.AttributeID] = append(validationsByAttr[vr.AttributeID], vr)
		attrName := extractAttrNameFromQualified(string(vr.AttributeID))
		if attrName != "" {
			validationsByName[attrName] = append(validationsByName[attrName], vr)
		}
	}

	// Attributes
	for i, attr := range entity.Attributes {
		// Documentation
		if attr.Documentation != "" {
			lines = append(lines, fmt.Sprintf("  /** %s */", attr.Documentation))
		}

		typeStr := formatAttributeType(attr.Type)
		var constraints strings.Builder

		// Check for validation rules
		attrValidations := validationsByAttr[attr.ID]
		if len(attrValidations) == 0 {
			attrValidations = validationsByName[attr.Name]
		}
		for _, vr := range attrValidations {
			if vr.Type == "Required" {
				constraints.WriteString(" NOT NULL")
				if vr.ErrorMessage != nil {
					errMsg := vr.ErrorMessage.GetTranslation("en_US")
					if errMsg != "" {
						constraints.WriteString(fmt.Sprintf(" ERROR '%s'", errMsg))
					}
				}
			}
			if vr.Type == "Unique" {
				constraints.WriteString(" UNIQUE")
				if vr.ErrorMessage != nil {
					errMsg := vr.ErrorMessage.GetTranslation("en_US")
					if errMsg != "" {
						constraints.WriteString(fmt.Sprintf(" ERROR '%s'", errMsg))
					}
				}
			}
		}

		// Default value
		if attr.Value != nil && attr.Value.DefaultValue != "" {
			defaultVal := attr.Value.DefaultValue
			if _, ok := attr.Type.(*domainmodel.StringAttributeType); ok {
				defaultVal = fmt.Sprintf("'%s'", defaultVal)
			}
			constraints.WriteString(fmt.Sprintf(" DEFAULT %s", defaultVal))
		}

		comma := ","
		if i == len(entity.Attributes)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  %s: %s%s%s", attr.Name, typeStr, constraints.String(), comma))
	}

	lines = append(lines, ")")

	// Build attr name map for indexes
	attrNames := make(map[model.ID]string)
	for _, attr := range entity.Attributes {
		attrNames[attr.ID] = attr.Name
	}

	// Indexes
	for _, idx := range entity.Indexes {
		var cols []string
		for _, ia := range idx.Attributes {
			colName := attrNames[ia.AttributeID]
			if !ia.Ascending {
				colName += " DESC"
			}
			cols = append(cols, colName)
		}
		if len(cols) > 0 {
			lines = append(lines, fmt.Sprintf("INDEX (%s)", strings.Join(cols, ", ")))
		}
	}

	lines = append(lines, ";")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// viewEntityFromProjectToMDL converts a view entity from project to MDL
func (e *Executor) viewEntityFromProjectToMDL(moduleName string, entity *domainmodel.Entity, dm *domainmodel.DomainModel) string {
	var lines []string

	if entity.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+entity.Documentation)
		lines = append(lines, " */")
	}

	lines = append(lines, fmt.Sprintf("@Position(%d, %d)", entity.Location.X, entity.Location.Y))
	lines = append(lines, fmt.Sprintf("CREATE VIEW ENTITY %s.%s (", moduleName, entity.Name))

	for i, attr := range entity.Attributes {
		typeStr := formatAttributeType(attr.Type)
		comma := ","
		if i == len(entity.Attributes)-1 {
			comma = ""
		}
		lines = append(lines, fmt.Sprintf("  %s: %s%s", attr.Name, typeStr, comma))
	}

	lines = append(lines, ") AS (")
	if entity.OqlQuery != "" {
		for line := range strings.SplitSeq(entity.OqlQuery, "\n") {
			lines = append(lines, "  "+line)
		}
	}
	lines = append(lines, ");")
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// enumerationToMDL converts a project enumeration to MDL text
func (e *Executor) enumerationToMDL(moduleName string, enum *model.Enumeration) string {
	var lines []string

	if enum.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+enum.Documentation)
		lines = append(lines, " */")
	}

	lines = append(lines, fmt.Sprintf("CREATE ENUMERATION %s.%s (", moduleName, enum.Name))

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

// associationToMDL converts a project association to MDL text
func (e *Executor) associationToMDL(moduleName string, assoc *domainmodel.Association, dm *domainmodel.DomainModel) string {
	var lines []string

	// Build entity name map
	entityNames := make(map[model.ID]string)
	for _, entity := range dm.Entities {
		entityNames[entity.ID] = entity.Name
	}

	if assoc.Documentation != "" {
		lines = append(lines, "/**")
		lines = append(lines, " * "+assoc.Documentation)
		lines = append(lines, " */")
	}

	fromEntity := entityNames[assoc.ParentID]
	toEntity := entityNames[assoc.ChildID]

	lines = append(lines, fmt.Sprintf("CREATE ASSOCIATION %s.%s", moduleName, assoc.Name))
	lines = append(lines, fmt.Sprintf("FROM %s.%s TO %s.%s", moduleName, fromEntity, moduleName, toEntity))

	assocType := "Reference"
	if assoc.Type == domainmodel.AssociationTypeReferenceSet {
		assocType = "ReferenceSet"
	}
	lines = append(lines, fmt.Sprintf("TYPE %s", assocType))

	owner := "Default"
	if assoc.Owner == domainmodel.AssociationOwnerBoth {
		owner = "Both"
	}
	lines = append(lines, fmt.Sprintf("OWNER %s", owner))

	deleteBehavior := "DELETE_BUT_KEEP_REFERENCES"
	if assoc.ChildDeleteBehavior != nil {
		switch assoc.ChildDeleteBehavior.Type {
		case domainmodel.DeleteBehaviorTypeDeleteMeAndReferences:
			deleteBehavior = "DELETE_CASCADE"
		case domainmodel.DeleteBehaviorTypeDeleteMeIfNoReferences:
			deleteBehavior = "DELETE_IF_NO_REFERENCES"
		}
	}
	lines = append(lines, fmt.Sprintf("DELETE_BEHAVIOR %s;", deleteBehavior))
	lines = append(lines, "/")

	return strings.Join(lines, "\n")
}

// ============================================================================
// Helper Functions
// ============================================================================

// dataTypeToString converts a DataType to its string representation
func (e *Executor) dataTypeToString(dt ast.DataType) string {
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
		return "DateTime"
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

// expressionToString converts an expression to its string representation
func (e *Executor) expressionToString(expr ast.Expression) string {
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
		return fmt.Sprintf("%s %s %s", e.expressionToString(ex.Left), ex.Operator, e.expressionToString(ex.Right))
	case *ast.UnaryExpr:
		return fmt.Sprintf("%s%s", ex.Operator, e.expressionToString(ex.Operand))
	case *ast.FunctionCallExpr:
		var args []string
		for _, arg := range ex.Arguments {
			args = append(args, e.expressionToString(arg))
		}
		return fmt.Sprintf("%s(%s)", ex.Name, strings.Join(args, ", "))
	case *ast.TokenExpr:
		return fmt.Sprintf("[%%%s%%]", ex.Token)
	case *ast.ParenExpr:
		return fmt.Sprintf("(%s)", e.expressionToString(ex.Inner))
	default:
		return fmt.Sprintf("%v", expr)
	}
}
