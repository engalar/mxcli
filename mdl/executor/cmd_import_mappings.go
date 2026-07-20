// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// listImportMappingsFn handles SHOW IMPORT MAPPINGS with HandlerDeps.
func listImportMappingsFn(ctx context.Context, inModule string, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	all, err := deps.MapperReader.ListImportMappings()
	if err != nil {
		return mdlerrors.NewBackend("list import mappings", err)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}

	type row struct {
		qualifiedName, name, schemaSource string
		elementCount                      int
	}
	var rows []row

	for _, im := range all {
		modID := h.FindModuleID(im.ContainerID)
		moduleName := h.GetModuleName(modID)
		if inModule != "" && !strings.EqualFold(moduleName, inModule) {
			continue
		}
		qn := moduleName + "." + im.Name
		src := im.JsonStructure
		if src == "" {
			src = im.XmlSchema
		}
		if src == "" {
			src = im.MessageDefinition
		}
		if src == "" {
			src = "(none)"
		}
		rows = append(rows, row{qualifiedName: qn, name: im.Name, schemaSource: src, elementCount: len(im.Elements)})
	}

	if len(rows) == 0 {
		if inModule != "" {
			fmt.Fprintf(deps.Output, "No import mappings found in module %s\n", inModule)
		} else {
			fmt.Fprintln(deps.Output, "No import mappings found")
		}
		return nil
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].qualifiedName < rows[j].qualifiedName })

	result := &TableResult{
		Columns: []string{"Import Mapping", "Name", "Schema Source", "Elements"},
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.name, r.schemaSource, r.elementCount})
	}
	return writeResultTo(deps.Output, deps.Format, result)
}

// listImportMappings prints a table of all import mapping documents.
func listImportMappings(ctx *ExecContext, inModule string) error {
	return listImportMappingsFn(ctx, inModule, ctx.Deps)
}

// describeImportMapping prints the MDL representation of an import mapping.
func describeImportMappingFn(ctx context.Context, name ast.QualifiedName, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	im, err := deps.MapperReader.GetImportMappingByQualifiedName(name.Module, name.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return mdlerrors.NewNotFound("import mapping", name.String())
		}
		return mdlerrors.NewBackend("get import mapping", err)
	}
	if im == nil {
		return mdlerrors.NewNotFound("import mapping", name.String())
	}

	if im.Documentation != "" {
		fmt.Fprintf(deps.Output, "/**\n * %s\n */\n", strings.ReplaceAll(im.Documentation, "\n", "\n * "))
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}
	modID := h.FindModuleID(im.ContainerID)
	moduleName := h.GetModuleName(modID)

	fmt.Fprintf(deps.Output, "create import mapping %s.%s\n", moduleName, im.Name)

	if im.JsonStructure != "" {
		fmt.Fprintf(deps.Output, "  with json structure %s\n", im.JsonStructure)
	} else if im.XmlSchema != "" {
		fmt.Fprintf(deps.Output, "  with xml schema %s\n", im.XmlSchema)
	}

	if len(im.Elements) > 0 {
		fmt.Fprintln(deps.Output, "{")
		for _, elem := range im.Elements {
			printImportMappingElement(deps.Output, elem, 1, true)
			fmt.Fprintln(deps.Output)
		}
		fmt.Fprintln(deps.Output, "};")
	}
	return nil
}

func describeImportMapping(ctx *ExecContext, name ast.QualifiedName) error {
	return describeImportMappingFn(ctx, name, ctx.Deps)
}

// handlingKeyword returns the MDL keyword for a Mendix ObjectHandling value.
func handlingKeyword(handling string) string {
	switch handling {
	case "Find":
		return "find"
	case "FindOrCreate":
		return "find or create"
	default:
		return "create"
	}
}

func printImportMappingElement(w io.Writer, elem *model.ImportMappingElement, depth int, isRoot bool) {
	indent := strings.Repeat("  ", depth)
	if elem.Kind == "Object" {
		handling := handlingKeyword(elem.ObjectHandling)
		if isRoot {
			// Root: CREATE Module.Entity { — use "." if entity is empty
			entity := elem.Entity
			if entity == "" {
				entity = "."
			}
			fmt.Fprintf(w, "%s%s %s {\n", indent, handling, entity)
		} else {
			// Nested object element:
			//   CREATE Assoc/Entity = jsonKey   — normal association path
			//   CREATE ./Entity = jsonKey       — self-reference (no association)
			//   CREATE . = jsonKey              — structural grouping (no association, no entity)
			assoc := elem.Association
			entity := elem.Entity
			if assoc == "" && entity == "" {
				fmt.Fprintf(w, "%s%s . = %s", indent, handling, elem.ExposedName)
			} else if assoc == "" {
				fmt.Fprintf(w, "%s%s ./%s = %s", indent, handling, entity, elem.ExposedName)
			} else {
				fmt.Fprintf(w, "%s%s %s/%s = %s", indent, handling, assoc, entity, elem.ExposedName)
			}
			if len(elem.Children) > 0 {
				fmt.Fprintln(w, " {")
			}
		}
		if len(elem.Children) > 0 {
			for i, child := range elem.Children {
				printImportMappingElement(w, child, depth+1, false)
				if i < len(elem.Children)-1 {
					fmt.Fprintln(w, ",")
				} else {
					fmt.Fprintln(w)
				}
			}
			fmt.Fprintf(w, "%s}", indent)
		}
	} else {
		// Value mapping: Attr = jsonField KEY
		attrName := elem.Attribute
		// Strip module prefix if present (Module.Entity.Attr → Attr)
		if parts := strings.Split(attrName, "."); len(parts) == 3 {
			attrName = parts[2]
		}
		keyStr := ""
		if elem.IsKey {
			keyStr = " key"
		}
		fmt.Fprintf(w, "%s%s = %s%s", indent, attrName, elem.ExposedName, keyStr)
	}
}

// execCreateImportMappingFn creates a new import mapping with HandlerDeps.
func execCreateImportMappingFn(ctx context.Context, s *ast.CreateImportMappingStmt, deps *HandlerDeps) error {
	if !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	existing, _ := deps.MapperReader.GetImportMappingByQualifiedName(s.Name.Module, s.Name.Name)
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExists("import mapping", s.Name.String())
	}

	modules, err := deps.ModuleLister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	var module *model.Module
	for _, m := range modules {
		if m.Name == s.Name.Module {
			module = m
			break
		}
	}
	if module == nil {
		return mdlerrors.NewNotFound("module", s.Name.Module)
	}
	containerID := module.ID

	im := &model.ImportMapping{
		ContainerID: containerID,
		Name:        s.Name.Name,
		ExportLevel: "Hidden",
	}

	// Set schema source reference
	switch s.SchemaKind {
	case "JSON_STRUCTURE":
		im.JsonStructure = s.SchemaRef.String()
	case "XML_SCHEMA":
		im.XmlSchema = s.SchemaRef.String()
	}

	// Build path→JsonElement map from JSON structure — mapping elements clone from this
	jsElementsByPath := map[string]*types.JsonElement{}
	if s.SchemaKind == "JSON_STRUCTURE" && s.SchemaRef.Module != "" {
		if js, err2 := deps.MapperReader.GetJsonStructureByQualifiedName(s.SchemaRef.Module, s.SchemaRef.Name); err2 == nil && js != nil {
			buildJsonElementPathMap(js.Elements, jsElementsByPath)
		}
	}

	// Build element tree from the AST definition, cloning JSON structure properties
	if s.RootElement != nil {
		root := buildImportMappingElementModel(s.Name.Module, s.RootElement, "", "(Object)", deps.DomainModelReader, jsElementsByPath, true)
		im.Elements = append(im.Elements, root)
	}

	if existing != nil {
		im.ID = existing.ID
		if err := deps.MapperWriter.UpdateImportMapping(im); err != nil {
			return mdlerrors.NewBackend("update import mapping", err)
		}
		if !deps.Quiet {
			fmt.Fprintf(deps.Output, "Modified import mapping %s.%s\n", s.Name.Module, s.Name.Name)
		}
		return nil
	}

	if err := deps.MapperWriter.CreateImportMapping(im); err != nil {
		return mdlerrors.NewBackend("create import mapping", err)
	}

	if !deps.Quiet {
		fmt.Fprintf(deps.Output, "Created import mapping %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}

// execCreateImportMapping creates a new import mapping.

// buildImportMappingElementModel converts an AST element definition to a model element.
// It clones properties from the matching JSON structure element (ExposedName, JsonPath,
// MaxOccurs, ElementType, etc.) and adds mapping-specific bindings (Entity, Attribute,
// Association, ObjectHandling).
func buildImportMappingElementModel(moduleName string, def *ast.ImportMappingElementDef, parentEntity, parentPath string, b backend.DomainModelReader, jsElems map[string]*types.JsonElement, isRoot bool) *model.ImportMappingElement {
	elem := &model.ImportMappingElement{
		BaseElement: model.BaseElement{
			ID: model.ID(types.GenerateID()),
		},
	}

	// Determine lookup path in JSON structure
	var lookupPath string
	if isRoot {
		lookupPath = "(Object)"
	} else {
		lookupPath = parentPath + "|" + def.JsonName
	}

	// Clone properties from the matching JSON structure element
	if jsElem, ok := jsElems[lookupPath]; ok {
		elem.ExposedName = jsElem.ExposedName
		elem.JsonPath = jsElem.Path
		elem.MinOccurs = jsElem.MinOccurs
		elem.MaxOccurs = jsElem.MaxOccurs
		elem.Nillable = jsElem.Nillable
		elem.OriginalValue = jsElem.OriginalValue
		elem.FractionDigits = jsElem.FractionDigits
		elem.TotalDigits = jsElem.TotalDigits
		elem.MaxLength = jsElem.MaxLength
	} else {
		elem.ExposedName = def.JsonName
		elem.JsonPath = lookupPath
		elem.Nillable = true
		elem.FractionDigits = -1
		elem.TotalDigits = -1
	}

	if def.Entity != "" {
		// Object/Array mapping — bind to entity
		elem.Kind = "Object"
		elem.TypeName = "ImportMappings$ObjectMappingElement"

		entity := def.Entity
		if !strings.Contains(entity, ".") {
			entity = moduleName + "." + entity
		}

		assoc := def.Association
		if assoc != "" && !strings.Contains(assoc, ".") {
			assoc = moduleName + "." + assoc
		}

		handling := def.ObjectHandling
		if handling == "" {
			handling = "Create"
		}

		elem.Entity = entity
		elem.Association = assoc
		elem.ObjectHandling = handling

		// For arrays: skip the container, use the item path directly.
		// Studio Pro represents arrays as a single ObjectMappingElement at the |(Object) item path.
		childPath := lookupPath
		if jsElem, ok := jsElems[lookupPath]; ok && jsElem.ElementType == "Array" {
			itemPath := lookupPath + "|(Object)"
			if jsItem, ok2 := jsElems[itemPath]; ok2 {
				elem.ExposedName = jsItem.ExposedName
				elem.JsonPath = jsItem.Path
				elem.MinOccurs = jsItem.MinOccurs
				elem.MaxOccurs = jsItem.MaxOccurs
				elem.Nillable = jsItem.Nillable
			}
			childPath = itemPath
		}

		for _, child := range def.Children {
			elem.Children = append(elem.Children, buildImportMappingElementModel(moduleName, child, entity, childPath, b, jsElems, false))
		}
	} else {
		// Value mapping — bind to attribute
		elem.Kind = "Value"
		elem.TypeName = "ImportMappings$ValueMappingElement"
		elem.DataType = resolveAttributeType(parentEntity, def.Attribute, b)
		elem.IsKey = def.IsKey
		attr := def.Attribute
		if parentEntity != "" && !strings.Contains(attr, ".") {
			attr = parentEntity + "." + attr
		}
		elem.Attribute = attr
	}

	return elem
}

// buildJsonElementPathMap recursively builds a map from JSON path → JsonElement.
func buildJsonElementPathMap(elems []*types.JsonElement, m map[string]*types.JsonElement) {
	for _, e := range elems {
		if e == nil {
			continue
		}
		m[e.Path] = e
		buildJsonElementPathMap(e.Children, m)
	}
}

// resolveAttributeType looks up the data type of an entity attribute from the project.
// Returns "String" as default if the attribute cannot be found.
func resolveAttributeType(entityQN, attrName string, b backend.DomainModelReader) string {
	if b == nil || entityQN == "" {
		return "String"
	}
	parts := strings.SplitN(entityQN, ".", 2)
	if len(parts) != 2 {
		return "String"
	}
	dms, err := b.ListDomainModelsGen()
	if err != nil {
		return "String"
	}
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		for _, item := range dm.EntitiesItems() {
			e, ok := item.(*genDm.Entity)
			if !ok || e.Name() != parts[1] {
				continue
			}
			for _, attrItem := range e.AttributesItems() {
				a, ok := attrItem.(*genDm.Attribute)
				if !ok || a.Name() != attrName || a.Type() == nil {
					continue
				}
				return importMappingAttributeTypeNameGen(a.Type())
			}
		}
	}
	return "String"
}

func importMappingAttributeTypeNameGen(t any) string {
	switch t := t.(type) {
	case *genDm.StringAttributeType:
		return "String"
	case *genDm.IntegerAttributeType:
		return "Integer"
	case *genDm.LongAttributeType:
		return "Long"
	case *genDm.DecimalAttributeType:
		return "Decimal"
	case *genDm.FloatAttributeType:
		return "Float"
	case *genDm.CurrencyAttributeType:
		return "Currency"
	case *genDm.BooleanAttributeType:
		return "Boolean"
	case *genDm.DateTimeAttributeType:
		return "DateTime"
	case *genDm.AutoNumberAttributeType:
		return "AutoNumber"
	case *genDm.BinaryAttributeType:
		return "Binary"
	case *genDm.HashedStringAttributeType:
		return "HashedString"
	case *genDm.MultiLanguageAttributeType:
		return "MultiLanguage"
	case *genDm.EnumerationAttributeType:
		return "Enumeration"
	default:
		_ = t
		return "String"
	}
}

// execDropImportMappingFn deletes an import mapping with HandlerDeps.
func execDropImportMappingFn(ctx context.Context, s *ast.DropImportMappingStmt, deps *HandlerDeps) error {
	if !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	im, err := deps.MapperReader.GetImportMappingByQualifiedName(s.Name.Module, s.Name.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return mdlerrors.NewNotFound("import mapping", s.Name.String())
		}
		return mdlerrors.NewBackend("get import mapping", err)
	}

	if err := deps.MapperWriter.DeleteImportMapping(im.ID); err != nil {
		return mdlerrors.NewBackend("drop import mapping", err)
	}

	if !deps.Quiet {
		fmt.Fprintf(deps.Output, "Dropped import mapping %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}

// execDropImportMapping deletes an import mapping.

func listImportMappingsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	return listImportMappingsFn(ctx, inModule, deps)
}


func describeImportMappingDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeImportMappingFn(ctx, name, deps)
}


