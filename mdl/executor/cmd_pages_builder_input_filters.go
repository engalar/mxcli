// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"

	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// DataGrid2 filter widget IDs — pluggable widget identifiers used when building
// DataGrid column header filter widgets.
const (
	widgetIDDataGridTextFilter     = "com.mendix.widget.web.datagridtextfilter.DatagridTextFilter"
	widgetIDDataGridDateFilter     = "com.mendix.widget.web.datagriddatefilter.DatagridDateFilter"
	widgetIDDataGridDropdownFilter = "com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter"
	widgetIDDataGridNumberFilter   = "com.mendix.widget.web.datagridnumberfilter.DatagridNumberFilter"
)

func (pb *pageBuilder) getFilterWidgetIDForAttribute(attrPath string) string {
	attrType := pb.findAttributeType(attrPath)
	if attrType == nil {
		return widgetIDDataGridTextFilter
	}

	switch attrType.(type) {
	case *genDm.StringAttributeType:
		return widgetIDDataGridTextFilter
	case *genDm.IntegerAttributeType, *genDm.LongAttributeType,
		*genDm.DecimalAttributeType, *genDm.AutoNumberAttributeType:
		return widgetIDDataGridNumberFilter
	case *genDm.DateTimeAttributeType:
		return widgetIDDataGridDateFilter
	case *genDm.BooleanAttributeType, *genDm.EnumerationAttributeType:
		return widgetIDDataGridDropdownFilter
	default:
		return widgetIDDataGridTextFilter
	}
}

// findAttributeType returns the gen-typed attribute type element for the given
// attribute path, or nil if not found. Returns element.Element for type-switching
// on gen attribute type structs (Stage 3.3.4.C7 migration from sdk/domainmodel).
func (pb *pageBuilder) findAttributeType(attrPath string) element.Element {
	if attrPath == "" {
		return nil
	}

	// Parse the attribute path
	parts := strings.Split(attrPath, ".")
	var entityName, attrName string

	if len(parts) >= 3 {
		// Format: Module.Entity.Attribute
		entityName = parts[0] + "." + parts[1]
		attrName = parts[len(parts)-1]
	} else if len(parts) == 2 {
		// Could be Entity.Attribute or Module.Entity - use context
		if pb.entityContext != "" {
			entityName = pb.entityContext
			attrName = parts[len(parts)-1]
		} else {
			// Assume Module.Entity format without attribute
			return nil
		}
	} else {
		// Just attribute name, use entity context
		if pb.entityContext != "" {
			entityName = pb.entityContext
			attrName = parts[0]
		} else {
			return nil
		}
	}

	// Find the entity and attribute
	pairs, err := pb.getDomainModelsWithContainer()
	if err != nil {
		return nil
	}

	h, err := pb.getHierarchy()
	if err != nil {
		return nil
	}

	// Parse entity qualified name
	entityParts := strings.Split(entityName, ".")
	if len(entityParts) < 2 {
		return nil
	}
	moduleName := entityParts[0]
	entityShortName := entityParts[1]

	// Find the entity
	for _, pair := range pairs {
		modName := h.GetModuleName(pair.ContainerID)
		if modName != moduleName {
			continue
		}
		for _, entityElem := range pair.DM.EntitiesItems() {
			entity, ok := entityElem.(*genDm.Entity)
			if !ok {
				continue
			}
			if entity.Name() != entityShortName {
				continue
			}
			for _, attrElem := range entity.AttributesItems() {
				attr, ok := attrElem.(*genDm.Attribute)
				if !ok {
					continue
				}
				if attr.Name() == attrName {
					return attr.Type()
				}
			}
			return nil
		}
	}

	return nil
}
