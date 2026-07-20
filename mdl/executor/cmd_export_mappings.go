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
)

// listExportMappingsFn handles SHOW EXPORT MAPPINGS with HandlerDeps.
func listExportMappingsFn(ctx context.Context, inModule string, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	all, err := deps.MapperReader.ListExportMappings()
	if err != nil {
		return mdlerrors.NewBackend("list export mappings", err)
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

	for _, em := range all {
		modID := h.FindModuleID(em.ContainerID)
		moduleName := h.GetModuleName(modID)
		if inModule != "" && !strings.EqualFold(moduleName, inModule) {
			continue
		}
		qn := moduleName + "." + em.Name
		src := em.JsonStructure
		if src == "" {
			src = em.XmlSchema
		}
		if src == "" {
			src = em.MessageDefinition
		}
		if src == "" {
			src = "(none)"
		}
		rows = append(rows, row{qualifiedName: qn, name: em.Name, schemaSource: src, elementCount: len(em.Elements)})
	}

	if len(rows) == 0 {
		if inModule != "" {
			fmt.Fprintf(deps.Output, "No export mappings found in module %s\n", inModule)
		} else {
			fmt.Fprintln(deps.Output, "No export mappings found")
		}
		return nil
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].qualifiedName < rows[j].qualifiedName })

	result := &TableResult{
		Columns: []string{"Export Mapping", "Name", "Schema Source", "Elements"},
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.name, r.schemaSource, r.elementCount})
	}
	return writeResultTo(deps.Output, deps.Format, result)
}

// listExportMappings prints a table of all export mapping documents.
func listExportMappings(ctx *ExecContext, inModule string) error {
	return listExportMappingsFn(ctx, inModule, ctx.Deps)
}

// describeExportMappingFn handles DESCRIBE EXPORT MAPPING with HandlerDeps.
func describeExportMappingFn(ctx context.Context, name ast.QualifiedName, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	em, err := deps.MapperReader.GetExportMappingByQualifiedName(name.Module, name.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return mdlerrors.NewNotFound("export mapping", name.String())
		}
		return mdlerrors.NewBackend("get export mapping", err)
	}
	if em == nil {
		return mdlerrors.NewNotFound("export mapping", name.String())
	}

	if em.Documentation != "" {
		fmt.Fprintf(deps.Output, "/**\n * %s\n */\n", strings.ReplaceAll(em.Documentation, "\n", "\n * "))
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}
	modID := h.FindModuleID(em.ContainerID)
	moduleName := h.GetModuleName(modID)

	fmt.Fprintf(deps.Output, "create export mapping %s.%s\n", moduleName, em.Name)

	if em.JsonStructure != "" {
		fmt.Fprintf(deps.Output, "  with json structure %s\n", em.JsonStructure)
	} else if em.XmlSchema != "" {
		fmt.Fprintf(deps.Output, "  with xml schema %s\n", em.XmlSchema)
	}

	if em.NullValueOption != "" && em.NullValueOption != "LeaveOutElement" {
		fmt.Fprintf(deps.Output, "  null values %s\n", em.NullValueOption)
	}

	if len(em.Elements) > 0 {
		fmt.Fprintln(deps.Output, "{")
		for _, elem := range em.Elements {
			printExportMappingElement(deps.Output, elem, 1, true)
			fmt.Fprintln(deps.Output)
		}
		fmt.Fprintln(deps.Output, "};")
	}
	return nil
}

// describeExportMapping prints the MDL representation of an export mapping.
func describeExportMapping(ctx *ExecContext, name ast.QualifiedName) error {
	return describeExportMappingFn(ctx, name, ctx.Deps)
}

func printExportMappingElement(w io.Writer, elem *model.ExportMappingElement, depth int, isRoot bool) {
	indent := strings.Repeat("  ", depth)
	if elem.Kind == "Object" {
		if isRoot {
			// Root: Module.Entity { — use "." if entity is empty (parameter mapping)
			entity := elem.Entity
			if entity == "" {
				entity = "."
			}
			fmt.Fprintf(w, "%s%s {\n", indent, entity)
		} else {
			// Nested object element. Several cases:
			//   Assoc/Entity AS jsonKey  — normal association path
			//   ./Entity AS jsonKey      — self-reference (no association, entity set)
			//   . AS jsonKey             — structural grouping (no association, no entity)
			assoc := elem.Association
			entity := elem.Entity
			if assoc == "" && entity == "" {
				fmt.Fprintf(w, "%s. as %s", indent, elem.ExposedName)
			} else if assoc == "" {
				fmt.Fprintf(w, "%s./%s as %s", indent, entity, elem.ExposedName)
			} else {
				fmt.Fprintf(w, "%s%s/%s as %s", indent, assoc, entity, elem.ExposedName)
			}
			if len(elem.Children) > 0 {
				fmt.Fprintln(w, " {")
			}
		}
		if len(elem.Children) > 0 {
			for i, child := range elem.Children {
				printExportMappingElement(w, child, depth+1, false)
				if i < len(elem.Children)-1 {
					fmt.Fprintln(w, ",")
				} else {
					fmt.Fprintln(w)
				}
			}
			fmt.Fprintf(w, "%s}", indent)
		}
	} else {
		// Value mapping: jsonField = Attr
		attrName := elem.Attribute
		// Strip module prefix if present (Module.Entity.Attr → Attr)
		if parts := strings.Split(attrName, "."); len(parts) == 3 {
			attrName = parts[2]
		}
		fmt.Fprintf(w, "%s%s = %s", indent, elem.ExposedName, attrName)
	}
}

// execCreateExportMappingFn creates a new export mapping with HandlerDeps.
func execCreateExportMappingFn(ctx context.Context, s *ast.CreateExportMappingStmt, deps *HandlerDeps) error {
	if !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	existing, _ := deps.MapperReader.GetExportMappingByQualifiedName(s.Name.Module, s.Name.Name)
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExists("export mapping", s.Name.String())
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

	em := &model.ExportMapping{
		ContainerID:     containerID,
		Name:            s.Name.Name,
		ExportLevel:     "Hidden",
		NullValueOption: s.NullValueOption,
	}
	if em.NullValueOption == "" {
		em.NullValueOption = "LeaveOutElement"
	}

	// Set schema source reference
	switch s.SchemaKind {
	case "JSON_STRUCTURE":
		em.JsonStructure = s.SchemaRef.String()
	case "XML_SCHEMA":
		em.XmlSchema = s.SchemaRef.String()
	}

	// Build a path→element info map from the JSON structure for schema alignment.
	jsElems := map[string]*types.JsonElement{}
	if s.SchemaKind == "JSON_STRUCTURE" && s.SchemaRef.Module != "" {
		if js, err2 := deps.MapperReader.GetJsonStructureByQualifiedName(s.SchemaRef.Module, s.SchemaRef.Name); err2 == nil && js != nil {
			buildJsonElementPathMap(js.Elements, jsElems)
		}
	}

	// Build element tree from the AST definition, cloning JSON structure properties
	if s.RootElement != nil {
		root := buildExportMappingElementModel(s.Name.Module, s.RootElement, "", "(Object)", jsElems, deps.DomainModelReader, true)
		em.Elements = append(em.Elements, root)
	}

	if existing != nil {
		em.ID = existing.ID
		if err := deps.MapperWriter.UpdateExportMapping(em); err != nil {
			return mdlerrors.NewBackend("update export mapping", err)
		}
		if !deps.Quiet {
			fmt.Fprintf(deps.Output, "Modified export mapping %s.%s\n", s.Name.Module, s.Name.Name)
		}
		return nil
	}

	if err := deps.MapperWriter.CreateExportMapping(em); err != nil {
		return mdlerrors.NewBackend("create export mapping", err)
	}

	if !deps.Quiet {
		fmt.Fprintf(deps.Output, "Created export mapping %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}

// execCreateExportMapping creates a new export mapping.

// buildExportMappingElementModel converts an AST element definition to a model element.
// It clones properties from the matching JSON structure element and adds mapping bindings.
func buildExportMappingElementModel(moduleName string, def *ast.ExportMappingElementDef, parentEntity, parentPath string, jsElems map[string]*types.JsonElement, b backend.DomainModelReader, isRoot bool) *model.ExportMappingElement {
	elem := &model.ExportMappingElement{
		BaseElement: model.BaseElement{
			ID: model.ID(types.GenerateID()),
		},
	}

	// Determine lookup path
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
		elem.MaxOccurs = jsElem.MaxOccurs
	} else {
		elem.ExposedName = def.JsonName
		elem.JsonPath = lookupPath
	}

	if def.Entity != "" {
		// Object/Array mapping — bind to entity
		elem.Kind = "Object"
		elem.TypeName = "ExportMappings$ObjectMappingElement"

		entity := def.Entity
		if !strings.Contains(entity, ".") {
			entity = moduleName + "." + entity
		}

		assoc := def.Association
		if assoc != "" && !strings.Contains(assoc, ".") {
			assoc = moduleName + "." + assoc
		}

		handling := "Parameter"
		if !isRoot {
			handling = "Find"
		}

		// Check if this is an array element in the JSON structure
		if jsElem, ok := jsElems[lookupPath]; ok && jsElem.ElementType == "Array" {
			// Export arrays have two levels:
			// 1. Array container: Kind=Array, entity=container entity, assoc to parent
			// 2. Item object: Kind=Object, entity=item entity, assoc to container
			//
			// MDL syntax: Assoc/Entity AS items { ItemAssoc/ItemEntity AS ItemsItem { values } }
			// The outer Assoc/Entity is for the container, the nested child provides the item.
			elem.Kind = "Array"
			elem.Association = assoc
			elem.ObjectHandling = handling
			elem.Entity = entity

			itemPath := lookupPath + "|(Object)"

			// The first (and typically only) child of the array in the MDL is the item definition.
			// Its children become the item element's value children.
			if len(def.Children) == 1 && def.Children[0].Entity != "" {
				itemDef := def.Children[0]
				itemEntity := itemDef.Entity
				if !strings.Contains(itemEntity, ".") {
					itemEntity = moduleName + "." + itemEntity
				}
				itemAssoc := itemDef.Association
				if itemAssoc != "" && !strings.Contains(itemAssoc, ".") {
					itemAssoc = moduleName + "." + itemAssoc
				}

				itemElem := &model.ExportMappingElement{
					BaseElement: model.BaseElement{
						ID:       model.ID(types.GenerateID()),
						TypeName: "ExportMappings$ObjectMappingElement",
					},
					Kind:           "Object",
					Entity:         itemEntity,
					Association:    itemAssoc,
					ObjectHandling: "Find",
				}
				if jsItem, ok2 := jsElems[itemPath]; ok2 {
					itemElem.ExposedName = jsItem.ExposedName
					itemElem.JsonPath = jsItem.Path
					itemElem.MaxOccurs = jsItem.MaxOccurs
				} else {
					itemElem.ExposedName = elem.ExposedName + "Item"
					itemElem.JsonPath = itemPath
					itemElem.MaxOccurs = -1
				}
				// Item's children are the value elements
				for _, valChild := range itemDef.Children {
					itemElem.Children = append(itemElem.Children, buildExportMappingElementModel(moduleName, valChild, itemEntity, itemPath, jsElems, b, false))
				}
				elem.Children = append(elem.Children, itemElem)
			} else {
				// Fallback: treat children as direct item children (no intermediate entity)
				for _, child := range def.Children {
					elem.Children = append(elem.Children, buildExportMappingElementModel(moduleName, child, entity, itemPath, jsElems, b, false))
				}
			}
		} else {
			// Regular object element
			elem.Entity = entity
			elem.Association = assoc
			elem.ObjectHandling = handling
			for _, child := range def.Children {
				elem.Children = append(elem.Children, buildExportMappingElementModel(moduleName, child, entity, lookupPath, jsElems, b, false))
			}
		}
	} else {
		// Value mapping — bind to attribute
		elem.Kind = "Value"
		elem.TypeName = "ExportMappings$ValueMappingElement"
		elem.DataType = resolveAttributeType(parentEntity, def.Attribute, b)
		attr := def.Attribute
		if parentEntity != "" && !strings.Contains(attr, ".") {
			attr = parentEntity + "." + attr
		}
		elem.Attribute = attr
		// JsonPath already set from JSON structure clone above
	}

	return elem
}

// execDropExportMappingFn deletes an export mapping with HandlerDeps.
func execDropExportMappingFn(ctx context.Context, s *ast.DropExportMappingStmt, deps *HandlerDeps) error {
	if !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}

	em, err := deps.MapperReader.GetExportMappingByQualifiedName(s.Name.Module, s.Name.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return mdlerrors.NewNotFound("export mapping", s.Name.String())
		}
		return mdlerrors.NewBackend("get export mapping", err)
	}
	if em == nil {
		return mdlerrors.NewNotFound("export mapping", s.Name.String())
	}

	if err := deps.MapperWriter.DeleteExportMapping(em.ID); err != nil {
		return mdlerrors.NewBackend("drop export mapping", err)
	}

	if !deps.Quiet {
		fmt.Fprintf(deps.Output, "Dropped export mapping %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}

// execDropExportMapping deletes an export mapping.

func listExportMappingsDeps(ctx context.Context, deps *HandlerDeps, inModule string) error {
	return listExportMappingsFn(ctx, inModule, deps)
}


func describeExportMappingDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeExportMappingFn(ctx, name, deps)
}

// ── Widgets ──


