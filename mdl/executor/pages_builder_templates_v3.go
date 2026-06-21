// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"

	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ============================================================================
// Template Attribute Resolution (V3 builder)
// ============================================================================

func (pb *pageBuilder) resolveAttributePathForEntity(attrName string, entityName string) string {
	oldContext := pb.entityContext
	pb.entityContext = entityName
	defer func() { pb.entityContext = oldContext }()
	return pb.resolveAttributePath(attrName)
}

// resolveTemplateAttributePath resolves template parameter values like $widgetName.Attribute
// to fully qualified entity paths like Module.Entity.Attribute.
func (pb *pageBuilder) resolveTemplateAttributePath(attrRef string) string {
	if attrRef == "" {
		return ""
	}

	if after, ok := strings.CutPrefix(attrRef, "$"); ok {
		withoutDollar := after
		parts := strings.SplitN(withoutDollar, ".", 2)
		if len(parts) == 2 {
			widgetName := parts[0]
			attrName := parts[1]

			if entityName, ok := pb.paramEntityNames[widgetName]; ok {
				return entityName + "." + attrName
			}
			if entityName, ok := pb.paramEntityNames["$"+widgetName]; ok {
				return entityName + "." + attrName
			}
			if pb.entityContext != "" {
				return pb.entityContext + "." + attrName
			}
			return attrRef
		}
	}

	return pb.resolveAttributePath(attrRef)
}

// resolveTemplateAttributePathFull resolves a template parameter reference and populates
// the gen ClientTemplateParameter with AttributePath, Expression, or SourceVariable.
func (pb *pageBuilder) resolveTemplateAttributePathFull(attrRef string, param *genPg.ClientTemplateParameter) {
	if attrRef == "" {
		return
	}

	if after, ok := strings.CutPrefix(attrRef, "$"); ok {
		withoutDollar := after
		parts := strings.SplitN(withoutDollar, ".", 2)
		if len(parts) == 2 {
			paramName := parts[0]
			attrName := parts[1]

			if entityName, ok := pb.paramEntityNames[paramName]; ok {
				fullPath := entityName + "." + attrName
				if pb.isNonStringAttribute(fullPath) {
					param.SetExpression("toString($" + paramName + "/" + attrName + ")")
					return
				}
				sv := genPg.NewPageVariable()
				assignFreshID(sv)
				sv.SetPageParameterQualifiedName(paramName)
				param.SetSourceVariable(sv)
				param.SetAttributePath(fullPath)
				return
			}
			if entityName, ok := pb.paramEntityNames["$"+paramName]; ok {
				fullPath := entityName + "." + attrName
				if pb.isNonStringAttribute(fullPath) {
					param.SetExpression("toString($" + paramName + "/" + attrName + ")")
					return
				}
				sv := genPg.NewPageVariable()
				assignFreshID(sv)
				sv.SetPageParameterQualifiedName(paramName)
				param.SetSourceVariable(sv)
				param.SetAttributePath(fullPath)
				return
			}
		}
	}

	resolved := pb.resolveTemplateAttributePath(attrRef)
	if !strings.HasPrefix(attrRef, "$") {
		// If the attrRef is already a full expression (contains function call or embedded $
		// variable), use it directly rather than prepending $currentObject/.
		if strings.Contains(attrRef, "(") || strings.Contains(attrRef, "$") {
			param.SetExpression(attrRef)
			return
		}
		if pb.isNonStringAttribute(resolved) {
			// Non-string attributes (enum, datetime, etc.) need explicit toString().
			param.SetExpression("toString($currentObject/" + attrRef + ")")
		} else {
			// String attributes: use expression format so the describe code and
			// Mendix validator can read the binding. AttributePath alone is not
			// checked by the describe and causes CE0402 "No value specified".
			param.SetExpression("$currentObject/" + attrRef)
		}
		return
	}
	// $var/attr is already in Mendix expression form; use SetExpression so the
	// describe code can read it back (describe reads Expression, not AttributePath).
	if strings.Contains(attrRef, "/") {
		param.SetExpression(attrRef)
		return
	}
	param.SetAttributePath(resolved)
}

// isNonStringAttribute checks if an attribute path refers to a non-String type.
func (pb *pageBuilder) isNonStringAttribute(attrPath string) bool {
	attrType := pb.findAttributeType(attrPath)
	if attrType == nil {
		return false
	}
	_, isString := attrType.(*genDm.StringAttributeType)
	return !isString
}
