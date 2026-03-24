// SPDX-License-Identifier: Apache-2.0

// Package executor - Java Action commands (SHOW/DESCRIBE/CREATE JAVA ACTIONS)
package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/javaactions"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// showJavaActions handles SHOW JAVA ACTIONS command.
func (e *Executor) showJavaActions(moduleName string) error {
	// Get hierarchy for module/folder resolution
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Get all Java actions
	javaActions, err := e.reader.ListJavaActions()
	if err != nil {
		return fmt.Errorf("failed to list java actions: %w", err)
	}

	// Collect rows and calculate column widths
	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
	}
	var rows []row
	qnWidth := len("Qualified Name")
	modWidth := len("Module")
	nameWidth := len("Name")
	pathWidth := len("Folder")

	for _, ja := range javaActions {
		modID := h.FindModuleID(ja.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName == "" || modName == moduleName {
			qualifiedName := modName + "." + ja.Name
			folderPath := h.BuildFolderPath(ja.ContainerID)

			rows = append(rows, row{qualifiedName, modName, ja.Name, folderPath})
			if len(qualifiedName) > qnWidth {
				qnWidth = len(qualifiedName)
			}
			if len(modName) > modWidth {
				modWidth = len(modName)
			}
			if len(ja.Name) > nameWidth {
				nameWidth = len(ja.Name)
			}
			if len(folderPath) > pathWidth {
				pathWidth = len(folderPath)
			}
		}
	}

	// Sort by qualified name
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	// Markdown table with aligned columns
	fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s |\n",
		qnWidth, "Qualified Name", modWidth, "Module", nameWidth, "Name", pathWidth, "Folder")
	fmt.Fprintf(e.output, "|-%s-|-%s-|-%s-|-%s-|\n",
		strings.Repeat("-", qnWidth), strings.Repeat("-", modWidth),
		strings.Repeat("-", nameWidth), strings.Repeat("-", pathWidth))
	for _, r := range rows {
		fmt.Fprintf(e.output, "| %-*s | %-*s | %-*s | %-*s |\n",
			qnWidth, r.qualifiedName, modWidth, r.module, nameWidth, r.name, pathWidth, r.folderPath)
	}
	fmt.Fprintf(e.output, "\n(%d java actions)\n", len(rows))
	return nil
}

// describeJavaAction handles DESCRIBE JAVA ACTION command - outputs MDL-style representation.
func (e *Executor) describeJavaAction(name ast.QualifiedName) error {
	qualifiedName := name.Module + "." + name.Name
	ja, err := e.reader.ReadJavaActionByName(qualifiedName)
	if err != nil {
		return fmt.Errorf("java action not found: %s", qualifiedName)
	}

	// Generate MDL-style output for CREATE JAVA ACTION format
	var sb strings.Builder

	// Documentation comment (JavaDoc style) — normalize \r\n to \n
	doc := strings.ReplaceAll(ja.Documentation, "\r\n", "\n")
	doc = strings.ReplaceAll(doc, "\r", "\n")
	if doc != "" {
		sb.WriteString("/**\n")
		for line := range strings.SplitSeq(doc, "\n") {
			sb.WriteString(" * ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString(" */\n")
	}

	// Build CREATE JAVA ACTION statement
	sb.WriteString("CREATE JAVA ACTION ")
	sb.WriteString(qualifiedName)
	sb.WriteString("(")

	// Parameters — one per line when descriptions are present
	hasParamDescriptions := false
	for _, p := range ja.Parameters {
		if p.Description != "" {
			hasParamDescriptions = true
			break
		}
	}

	for i, param := range ja.Parameters {
		if i > 0 {
			sb.WriteString(", ")
		}
		if hasParamDescriptions {
			sb.WriteString("\n    ")
		}
		sb.WriteString(param.Name)
		sb.WriteString(": ")
		if param.ParameterType != nil {
			sb.WriteString(formatJavaActionType(param.ParameterType))
		} else {
			sb.WriteString("Object")
		}
		if param.IsRequired {
			sb.WriteString(" NOT NULL")
		}
		if param.Description != "" {
			paramDoc := strings.ReplaceAll(param.Description, "\r\n", "\n")
			paramDoc = strings.ReplaceAll(paramDoc, "\r", "\n")
			firstLine, _, _ := strings.Cut(paramDoc, "\n")
			sb.WriteString("  -- ")
			sb.WriteString(firstLine)
		}
	}
	if hasParamDescriptions {
		sb.WriteString("\n")
	}
	sb.WriteString(")")

	// Return type
	if ja.ReturnType != nil {
		sb.WriteString(" RETURNS ")
		sb.WriteString(formatJavaActionReturnType(ja.ReturnType))
	}

	// RETURN NAME metadata
	if ja.ActionDefaultReturnName != "" {
		sb.WriteString("\n-- RETURN NAME: '")
		sb.WriteString(ja.ActionDefaultReturnName)
		sb.WriteString("'")
	}

	// EXPOSED AS clause
	if ja.MicroflowActionInfo != nil && ja.MicroflowActionInfo.Caption != "" {
		sb.WriteString("\nEXPOSED AS '")
		sb.WriteString(ja.MicroflowActionInfo.Caption)
		sb.WriteString("' IN '")
		sb.WriteString(ja.MicroflowActionInfo.Category)
		sb.WriteString("'")
		if ja.MicroflowActionInfo.Icon != "" {
			sb.WriteString("\n-- ICON: ")
			sb.WriteString(ja.MicroflowActionInfo.Icon)
		}
	}

	// Try to read and include Java source code
	javaCode := e.readJavaActionUserCode(name.Module, name.Name)
	if javaCode != "" {
		sb.WriteString("\nAS $$\n")
		sb.WriteString(javaCode)
		sb.WriteString("\n$$")
	}

	sb.WriteString(";")

	// Output the complete statement
	fmt.Fprintln(e.output, sb.String())

	// Additional info as comments
	if ja.ExportLevel != "" && ja.ExportLevel != "Hidden" {
		fmt.Fprintf(e.output, "-- EXPORT LEVEL: %s\n", ja.ExportLevel)
	}
	if ja.Excluded {
		fmt.Fprintln(e.output, "-- EXCLUDED: true")
	}

	return nil
}

// readJavaActionUserCode reads the Java source file and extracts the user code section.
func (e *Executor) readJavaActionUserCode(moduleName, actionName string) string {
	if e.mprPath == "" {
		return ""
	}

	// Build path to Java source file
	projectRoot := filepath.Dir(e.mprPath)
	moduleNameLower := strings.ToLower(moduleName)
	javaPath := filepath.Join(projectRoot, "javasource", moduleNameLower, "actions", actionName+".java")

	// Read the file
	content, err := os.ReadFile(javaPath)
	if err != nil {
		return ""
	}

	// Extract user code between BEGIN USER CODE and END USER CODE markers
	source := string(content)
	beginMarker := "// BEGIN USER CODE"
	endMarker := "// END USER CODE"

	beginIdx := strings.Index(source, beginMarker)
	endIdx := strings.Index(source, endMarker)

	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		// No markers found, return empty (or could return raw code)
		return ""
	}

	// Extract code between markers (skip the marker line itself)
	userCode := source[beginIdx+len(beginMarker) : endIdx]
	userCode = strings.TrimPrefix(userCode, "\n")
	userCode = strings.TrimSuffix(userCode, "\n")
	userCode = strings.TrimRight(userCode, " \t")

	return userCode
}

// formatJavaActionType formats a Java action parameter type for MDL output.
func formatJavaActionType(t javaactions.CodeActionParameterType) string {
	if t == nil {
		return "Object"
	}
	// EntityTypeParameterType → ENTITY <name> syntax
	if etp, ok := t.(*javaactions.EntityTypeParameterType); ok {
		if etp.TypeParameterName != "" {
			return "ENTITY <" + etp.TypeParameterName + ">"
		}
		return "ENTITY <>"
	}
	return t.TypeString()
}

// formatJavaActionReturnType formats a Java action return type.
func formatJavaActionReturnType(t javaactions.CodeActionReturnType) string {
	if t == nil {
		return "Void"
	}
	return t.TypeString()
}

// execDropJavaAction handles DROP JAVA ACTION statements.
func (e *Executor) execDropJavaAction(s *ast.DropJavaActionStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project")
	}

	// Get hierarchy for module/folder resolution
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find and delete the Java action
	jas, err := e.reader.ListJavaActions()
	if err != nil {
		return fmt.Errorf("failed to list java actions: %w", err)
	}

	for _, ja := range jas {
		modID := h.FindModuleID(ja.ContainerID)
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && ja.Name == s.Name.Name {
			if err := e.writer.DeleteJavaAction(ja.ID); err != nil {
				return fmt.Errorf("failed to delete java action: %w", err)
			}
			fmt.Fprintf(e.output, "Dropped java action: %s.%s\n", s.Name.Module, s.Name.Name)
			return nil
		}
	}

	return fmt.Errorf("java action not found: %s.%s", s.Name.Module, s.Name.Name)
}

// execCreateJavaAction handles CREATE JAVA ACTION statements.
func (e *Executor) execCreateJavaAction(s *ast.CreateJavaActionStmt) error {
	if e.writer == nil {
		return fmt.Errorf("not connected to a project")
	}

	// Get hierarchy for module/folder resolution
	h, err := e.getHierarchy()
	if err != nil {
		return fmt.Errorf("failed to build hierarchy: %w", err)
	}

	// Find the module
	modules, err := e.reader.ListModules()
	if err != nil {
		return fmt.Errorf("failed to get modules: %w", err)
	}

	var containerID model.ID
	var moduleName string
	for _, mod := range modules {
		if mod.Name == s.Name.Module {
			containerID = mod.ID
			moduleName = mod.Name
			break
		}
	}
	if containerID == "" {
		return fmt.Errorf("module not found: %s", s.Name.Module)
	}

	// Check if Java action already exists
	jas, err := e.reader.ListJavaActions()
	if err != nil {
		return fmt.Errorf("failed to list java actions: %w", err)
	}
	for _, existing := range jas {
		existingModID := h.FindModuleID(existing.ContainerID)
		existingModName := h.GetModuleName(existingModID)
		if existingModName == s.Name.Module && existing.Name == s.Name.Name {
			return fmt.Errorf("java action already exists: %s.%s", s.Name.Module, s.Name.Name)
		}
	}

	// Create the Java action
	ja := &javaactions.JavaAction{
		BaseElement: model.BaseElement{
			ID:       model.ID(mpr.GenerateID()),
			TypeName: "JavaActions$JavaAction",
		},
		ContainerID:   containerID,
		Name:          s.Name.Name,
		Documentation: s.Documentation,
		ExportLevel:   "Public",
	}

	// Build type parameter definitions (with IDs for BY_ID references)
	typeParamNameToID := make(map[string]model.ID)
	for _, tpName := range s.TypeParameters {
		tpDef := &javaactions.TypeParameterDef{
			BaseElement: model.BaseElement{
				ID: model.ID(mpr.GenerateID()),
			},
			Name: tpName,
		}
		ja.TypeParameters = append(ja.TypeParameters, tpDef)
		typeParamNameToID[tpName] = tpDef.ID
	}

	// Build a set of type parameter names for quick lookup
	typeParamNames := make(map[string]bool)
	for _, tpName := range s.TypeParameters {
		typeParamNames[tpName] = true
	}

	// Convert parameters:
	// - TypeEntityTypeParam → EntityTypeParameterType (entity type selector)
	// - Bare name matching a type parameter → TypeParameter (ParameterizedEntityType)
	// - Other types → convert normally
	for _, param := range s.Parameters {
		jaParam := &javaactions.JavaActionParameter{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "JavaActions$JavaActionParameter",
			},
			Name:       param.Name,
			IsRequired: param.IsRequired,
		}
		if param.Type.Kind == ast.TypeEntityTypeParam {
			// Explicit ENTITY <pEntity> → EntityTypeParameterType (entity type selector)
			tpName := param.Type.TypeParamName
			jaParam.ParameterType = &javaactions.EntityTypeParameterType{
				BaseElement:       model.BaseElement{ID: model.ID(mpr.GenerateID())},
				TypeParameterID:   typeParamNameToID[tpName],
				TypeParameterName: tpName,
			}
		} else if isTypeParamRef(param.Type, typeParamNames) {
			// Bare name matching a type parameter → TypeParameter (ParameterizedEntityType)
			tpName := getTypeParamRefName(param.Type)
			jaParam.ParameterType = &javaactions.TypeParameter{
				BaseElement:     model.BaseElement{ID: model.ID(mpr.GenerateID())},
				TypeParameterID: typeParamNameToID[tpName],
				TypeParameter:   tpName,
			}
		} else {
			jaParam.ParameterType = astDataTypeToJavaActionParamType(param.Type)
		}
		ja.Parameters = append(ja.Parameters, jaParam)
	}

	// Convert return type — check if it references a type parameter
	if isTypeParamRef(s.ReturnType, typeParamNames) {
		tpName := getTypeParamRefName(s.ReturnType)
		ja.ReturnType = &javaactions.TypeParameter{
			BaseElement:     model.BaseElement{ID: model.ID(mpr.GenerateID())},
			TypeParameterID: typeParamNameToID[tpName],
			TypeParameter:   tpName,
		}
	} else {
		ja.ReturnType = astDataTypeToJavaActionReturnType(s.ReturnType)
	}

	// Build MicroflowActionInfo if EXPOSED AS clause is present
	if s.ExposedCaption != "" {
		ja.MicroflowActionInfo = &javaactions.MicroflowActionInfo{
			BaseElement: model.BaseElement{ID: model.ID(mpr.GenerateID())},
			Caption:     s.ExposedCaption,
			Category:    s.ExposedCategory,
		}
	}

	// Create in MPR
	if err := e.writer.CreateJavaAction(ja); err != nil {
		return fmt.Errorf("failed to create java action: %w", err)
	}

	// Write Java source file if code is provided
	if s.JavaCode != "" {
		if err := e.writer.WriteJavaSourceFile(moduleName, s.Name.Name, s.JavaCode, ja.Parameters, ja.ReturnType); err != nil {
			return fmt.Errorf("failed to write java source file: %w", err)
		}
	}

	// Clear cache
	e.cache = nil

	fmt.Fprintf(e.output, "Created java action: %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// astDataTypeToJavaActionParamType converts an AST DataType to a Java action parameter type.
func astDataTypeToJavaActionParamType(dt ast.DataType) javaactions.CodeActionParameterType {
	switch dt.Kind {
	case ast.TypeBoolean:
		return &javaactions.BooleanType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$BooleanType",
			},
		}
	case ast.TypeInteger:
		return &javaactions.IntegerType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$IntegerType",
			},
		}
	case ast.TypeLong:
		return &javaactions.LongType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$LongType",
			},
		}
	case ast.TypeDecimal:
		return &javaactions.DecimalType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$DecimalType",
			},
		}
	case ast.TypeString:
		return &javaactions.StringType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$StringType",
			},
		}
	case ast.TypeDateTime, ast.TypeDate:
		return &javaactions.DateTimeType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$DateTimeType",
			},
		}
	case ast.TypeEntity, ast.TypeEnumeration:
		// TypeEnumeration with a qualified name is treated as entity type here,
		// since the visitor can't distinguish entity types from enumeration types
		// for bare qualified names like Module.EntityName.
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		} else if dt.EnumRef != nil {
			entityName = dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
		return &javaactions.EntityType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$EntityType",
			},
			Entity: entityName,
		}
	case ast.TypeListOf:
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		}
		return &javaactions.ListType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$ListType",
			},
			Entity: entityName,
		}
	default:
		// Default to String type for unknown kinds
		return &javaactions.StringType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$StringType",
			},
		}
	}
}

// astDataTypeToJavaActionReturnType converts an AST DataType to a Java action return type.
func astDataTypeToJavaActionReturnType(dt ast.DataType) javaactions.CodeActionReturnType {
	switch dt.Kind {
	case ast.TypeVoid:
		return &javaactions.VoidType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$VoidType",
			},
		}
	case ast.TypeBoolean:
		return &javaactions.BooleanType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$BooleanType",
			},
		}
	case ast.TypeInteger:
		return &javaactions.IntegerType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$IntegerType",
			},
		}
	case ast.TypeLong:
		return &javaactions.LongType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$LongType",
			},
		}
	case ast.TypeDecimal:
		return &javaactions.DecimalType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$DecimalType",
			},
		}
	case ast.TypeString:
		return &javaactions.StringType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$StringType",
			},
		}
	case ast.TypeDateTime, ast.TypeDate:
		return &javaactions.DateTimeType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$DateTimeType",
			},
		}
	case ast.TypeEntity, ast.TypeEnumeration:
		// TypeEnumeration with a qualified name is treated as entity type here,
		// since the visitor can't distinguish entity types from enumeration types.
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		} else if dt.EnumRef != nil {
			entityName = dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
		return &javaactions.EntityType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$EntityType",
			},
			Entity: entityName,
		}
	case ast.TypeListOf:
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		}
		return &javaactions.ListType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$ListType",
			},
			Entity: entityName,
		}
	default:
		// Default to Boolean type (most common for Java actions)
		return &javaactions.BooleanType{
			BaseElement: model.BaseElement{
				ID:       model.ID(mpr.GenerateID()),
				TypeName: "CodeActions$BooleanType",
			},
		}
	}
}

// isTypeParamRef checks if an AST DataType is a bare name that matches a type parameter.
// Type parameter names like "pEntity" parse as either TypeEnumeration (with EnumRef)
// or TypeEntity (with EntityRef) with a single-part name (no module prefix).
func isTypeParamRef(dt ast.DataType, typeParamNames map[string]bool) bool {
	name := getTypeParamRefName(dt)
	return name != "" && typeParamNames[name]
}

// getTypeParamRefName extracts the name from a DataType that could be a type parameter reference.
// Returns empty string if the DataType doesn't look like a type parameter reference.
func getTypeParamRefName(dt ast.DataType) string {
	switch dt.Kind {
	case ast.TypeEnumeration:
		if dt.EnumRef != nil && dt.EnumRef.Module == "" {
			return dt.EnumRef.Name
		}
		if dt.EnumRef != nil {
			return dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
	case ast.TypeEntity:
		if dt.EntityRef != nil && dt.EntityRef.Module == "" {
			return dt.EntityRef.Name
		}
		if dt.EntityRef != nil {
			return dt.EntityRef.Module + "." + dt.EntityRef.Name
		}
	}
	return ""
}
