// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"

	"go.mongodb.org/mongo-driver/bson"
)

func (r *Reader) parseDomainModel(unitID, containerID string, contents []byte) (*domainmodel.DomainModel, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BSON: %w", err)
	}

	dm := &domainmodel.DomainModel{}
	dm.ID = model.ID(unitID)
	dm.TypeName = "DomainModels$DomainModel"
	dm.ContainerID = model.ID(containerID)

	// Parse entities - use extractBsonArray to handle Mendix array format
	entities := extractBsonArray(raw["Entities"])
	for _, e := range entities {
		if entityMap, ok := e.(map[string]any); ok {
			entity := parseEntity(entityMap)
			dm.Entities = append(dm.Entities, entity)
		}
	}

	// Parse associations
	associations := extractBsonArray(raw["Associations"])
	for _, a := range associations {
		if assocMap, ok := a.(map[string]any); ok {
			assoc := parseAssociation(assocMap)
			dm.Associations = append(dm.Associations, assoc)
		}
	}

	// Parse cross-module associations
	crossAssocs := extractBsonArray(raw["CrossAssociations"])
	for _, ca := range crossAssocs {
		if caMap, ok := ca.(map[string]any); ok {
			crossAssoc := parseCrossAssociation(caMap)
			dm.CrossAssociations = append(dm.CrossAssociations, crossAssoc)
		}
	}

	// Parse annotations
	annotations := extractBsonArray(raw["Annotations"])
	for _, a := range annotations {
		if annotMap, ok := a.(map[string]any); ok {
			annot := parseAnnotation(annotMap)
			dm.Annotations = append(dm.Annotations, annot)
		}
	}

	return dm, nil
}

func parseEntity(raw map[string]any) *domainmodel.Entity {
	entity := &domainmodel.Entity{}

	// Use extractBsonID to handle various ID formats (string, binary, base64)
	entity.ID = model.ID(extractBsonID(raw["$ID"]))
	if typeName, ok := raw["$Type"].(string); ok {
		entity.TypeName = typeName
	}
	if name, ok := raw["Name"].(string); ok {
		entity.Name = name
	}
	if doc, ok := raw["Documentation"].(string); ok {
		entity.Documentation = doc
	}

	// Parse location - handle string "x;y" format or map format
	if locStr, ok := raw["Location"].(string); ok {
		// Parse "x;y" format
		parts := strings.Split(locStr, ";")
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &entity.Location.X)
			fmt.Sscanf(parts[1], "%d", &entity.Location.Y)
		}
	} else if loc, ok := raw["Location"].(map[string]any); ok {
		entity.Location.X = extractInt(loc["x"])
		entity.Location.Y = extractInt(loc["y"])
	}

	// Parse persistable - default to true if not specified
	entity.Persistable = true
	if persistable, ok := raw["Persistable"].(bool); ok {
		entity.Persistable = persistable
	}

	// Parse source (for view/external entities)
	if source, ok := raw["Source"].(map[string]any); ok {
		if sourceType, ok := source["$Type"].(string); ok {
			entity.Source = sourceType
		}
		// For view entities, extract the OQL query directly
		if oqlQuery, ok := source["OqlQuery"].(string); ok {
			entity.OqlQuery = oqlQuery
		}
		// For view entities, extract the source document reference
		if sourceDocRef, ok := source["SourceDocument"].(string); ok {
			entity.SourceDocumentRef = sourceDocRef
		}
		// For external entities (OData remote entity source)
		if entity.Source == "Rest$ODataRemoteEntitySource" {
			entity.RemoteServiceName = extractString(source["SourceDocument"])
			entity.RemoteEntitySet = extractString(source["EntitySet"])
			entity.RemoteEntityName = extractString(source["RemoteName"])
			entity.Countable = extractBool(source["Countable"], false)
			entity.Creatable = extractBool(source["Creatable"], false)
			entity.Deletable = extractBool(source["Deletable"], false)
			entity.Updatable = extractBool(source["Updatable"], false)
		}
	}

	// Parse generalization (parent entity) - field is MaybeGeneralization in newer formats
	genField := raw["Generalization"]
	if genField == nil {
		genField = raw["MaybeGeneralization"]
	}
	if genField != nil {
		if genMap, ok := genField.(map[string]any); ok {
			if genID := extractBsonID(genMap["$ID"]); genID != "" {
				entity.GeneralizationID = model.ID(genID)
			}
			// Handle qualified name reference (e.g., "System.User")
			if genRef, ok := genMap["Generalization"].(string); ok {
				entity.GeneralizationRef = genRef
			}
			// For NoGeneralization, Persistable is stored inside the generalization object
			if genType, ok := genMap["$Type"].(string); ok && genType == "DomainModels$NoGeneralization" {
				if persistable, ok := genMap["Persistable"].(bool); ok {
					entity.Persistable = persistable
				}
			}
		}
	}

	// Parse attributes using extractBsonArray
	attrs := extractBsonArray(raw["Attributes"])
	for _, a := range attrs {
		if attrMap, ok := a.(map[string]any); ok {
			attr := parseAttribute(attrMap)
			entity.Attributes = append(entity.Attributes, attr)
		}
	}

	// Parse indexes
	indexes := extractBsonArray(raw["Indexes"])
	for _, i := range indexes {
		if indexMap, ok := i.(map[string]any); ok {
			index := parseIndex(indexMap)
			entity.Indexes = append(entity.Indexes, index)
		}
	}

	// Parse access rules
	rules := extractBsonArray(raw["AccessRules"])
	for _, r := range rules {
		if ruleMap, ok := r.(map[string]any); ok {
			rule := parseAccessRule(ruleMap)
			entity.AccessRules = append(entity.AccessRules, rule)
		}
	}

	// Parse validation rules
	validations := extractBsonArray(raw["ValidationRules"])
	for _, v := range validations {
		if validMap, ok := v.(map[string]any); ok {
			validation := parseValidationRule(validMap)
			entity.ValidationRules = append(entity.ValidationRules, validation)
		}
	}

	// Parse event handlers
	handlers := extractBsonArray(raw["EventHandlers"])
	for _, h := range handlers {
		if handlerMap, ok := h.(map[string]any); ok {
			handler := parseEventHandler(handlerMap)
			entity.EventHandlers = append(entity.EventHandlers, handler)
		}
	}

	return entity
}

func parseAttribute(raw map[string]any) *domainmodel.Attribute {
	attr := &domainmodel.Attribute{}

	attr.ID = model.ID(extractBsonID(raw["$ID"]))
	attr.TypeName = extractString(raw["$Type"])
	attr.Name = extractString(raw["Name"])
	attr.Documentation = extractString(raw["Documentation"])

	// Parse attribute type - Mendix uses "NewType" field
	if attrType, ok := raw["NewType"].(map[string]any); ok {
		attr.Type = parseAttributeType(attrType)
	} else if attrType, ok := raw["Type"].(map[string]any); ok {
		// Fallback to "Type" for older format
		attr.Type = parseAttributeType(attrType)
	}

	// Parse default value
	if val, ok := raw["Value"].(map[string]any); ok {
		attr.Value = parseAttributeValue(val)
	}

	return attr
}

func parseAttributeValue(raw map[string]any) *domainmodel.AttributeValue {
	typeName := extractString(raw["$Type"])
	defaultValue := extractString(raw["DefaultValue"])
	valueID := model.ID(extractBsonID(raw["$ID"]))

	switch typeName {
	case "DomainModels$StoredValue":
		val := &domainmodel.AttributeValue{
			Type:         "StoredValue",
			DefaultValue: defaultValue,
		}
		val.ID = valueID
		return val
	case "DomainModels$CalculatedValue":
		val := &domainmodel.AttributeValue{
			Type:          "CalculatedValue",
			MicroflowID:   model.ID(extractBsonID(raw["Microflow"])),
			MicroflowName: extractString(raw["Microflow"]),
		}
		val.ID = valueID
		return val
	case "DomainModels$OqlViewValue":
		val := &domainmodel.AttributeValue{
			Type:          "OqlViewValue",
			ViewReference: extractString(raw["Reference"]),
		}
		val.ID = valueID
		return val
	default:
		val := &domainmodel.AttributeValue{
			DefaultValue: defaultValue,
		}
		val.ID = valueID
		return val
	}
}

func parseAttributeType(raw map[string]any) domainmodel.AttributeType {
	typeName, _ := raw["$Type"].(string)
	typeID := model.ID(extractBsonID(raw["$ID"]))

	switch typeName {
	case "DomainModels$StringAttributeType":
		t := &domainmodel.StringAttributeType{}
		t.ID = typeID
		if length, ok := raw["Length"].(int32); ok {
			t.Length = int(length)
		}
		return t
	case "DomainModels$IntegerAttributeType":
		t := &domainmodel.IntegerAttributeType{}
		t.ID = typeID
		return t
	case "DomainModels$LongAttributeType":
		t := &domainmodel.LongAttributeType{}
		t.ID = typeID
		return t
	case "DomainModels$DecimalAttributeType":
		t := &domainmodel.DecimalAttributeType{}
		t.ID = typeID
		return t
	case "DomainModels$BooleanAttributeType":
		t := &domainmodel.BooleanAttributeType{}
		t.ID = typeID
		return t
	case "DomainModels$DateTimeAttributeType":
		t := &domainmodel.DateTimeAttributeType{}
		t.ID = typeID
		if localize, ok := raw["LocalizeDate"].(bool); ok {
			t.LocalizeDate = localize
		}
		return t
	case "DomainModels$EnumerationAttributeType":
		t := &domainmodel.EnumerationAttributeType{}
		t.ID = typeID
		// Enumeration is stored as qualified name string (BY_NAME_REFERENCE)
		if enumRef, ok := raw["Enumeration"].(string); ok {
			t.EnumerationRef = enumRef
			// Also store in EnumerationID for backward compatibility
			t.EnumerationID = model.ID(enumRef)
		}
		return t
	case "DomainModels$AutoNumberAttributeType":
		t := &domainmodel.AutoNumberAttributeType{}
		t.ID = typeID
		return t
	case "DomainModels$BinaryAttributeType":
		t := &domainmodel.BinaryAttributeType{}
		t.ID = typeID
		return t
	case "DomainModels$HashedStringAttributeType":
		t := &domainmodel.HashedStringAttributeType{}
		t.ID = typeID
		return t
	default:
		t := &domainmodel.StringAttributeType{} // Default fallback
		t.ID = typeID
		return t
	}
}

func parseAssociation(raw map[string]any) *domainmodel.Association {
	assoc := &domainmodel.Association{}

	assoc.ID = model.ID(extractBsonID(raw["$ID"]))
	assoc.TypeName = extractString(raw["$Type"])
	assoc.Name = extractString(raw["Name"])
	assoc.Documentation = extractString(raw["Documentation"])
	assoc.ParentID = model.ID(extractBsonID(raw["ParentPointer"]))
	assoc.ChildID = model.ID(extractBsonID(raw["ChildPointer"]))
	assoc.Type = domainmodel.AssociationType(extractString(raw["Type"]))
	assoc.Owner = domainmodel.AssociationOwner(extractString(raw["Owner"]))
	if sf := extractString(raw["StorageFormat"]); sf != "" {
		assoc.StorageFormat = domainmodel.AssociationStorageFormat(sf)
	} else {
		assoc.StorageFormat = domainmodel.StorageFormatTable
	}

	// Parse delete behavior
	if deleteBehaviorRaw, ok := raw["DeleteBehavior"].(map[string]any); ok {
		if parentType := extractString(deleteBehaviorRaw["ParentDeleteBehavior"]); parentType != "" {
			assoc.ParentDeleteBehavior = &domainmodel.DeleteBehavior{
				Type: domainmodel.DeleteBehaviorType(parentType),
			}
		}
		if childType := extractString(deleteBehaviorRaw["ChildDeleteBehavior"]); childType != "" {
			assoc.ChildDeleteBehavior = &domainmodel.DeleteBehavior{
				Type: domainmodel.DeleteBehaviorType(childType),
			}
		}
	}

	return assoc
}

func parseCrossAssociation(raw map[string]any) *domainmodel.CrossModuleAssociation {
	ca := &domainmodel.CrossModuleAssociation{}

	ca.ID = model.ID(extractBsonID(raw["$ID"]))
	ca.TypeName = extractString(raw["$Type"])
	ca.Name = extractString(raw["Name"])
	ca.Documentation = extractString(raw["Documentation"])
	ca.ParentID = model.ID(extractBsonID(raw["ParentPointer"]))
	ca.ChildRef = extractString(raw["Child"])
	ca.Type = domainmodel.AssociationType(extractString(raw["Type"]))
	ca.Owner = domainmodel.AssociationOwner(extractString(raw["Owner"]))
	if sf := extractString(raw["StorageFormat"]); sf != "" {
		ca.StorageFormat = domainmodel.AssociationStorageFormat(sf)
	} else {
		ca.StorageFormat = domainmodel.StorageFormatTable
	}

	// Parse delete behavior
	if deleteBehaviorRaw, ok := raw["DeleteBehavior"].(map[string]any); ok {
		if parentType := extractString(deleteBehaviorRaw["ParentDeleteBehavior"]); parentType != "" {
			ca.ParentDeleteBehavior = &domainmodel.DeleteBehavior{
				Type: domainmodel.DeleteBehaviorType(parentType),
			}
		}
		if childType := extractString(deleteBehaviorRaw["ChildDeleteBehavior"]); childType != "" {
			ca.ChildDeleteBehavior = &domainmodel.DeleteBehavior{
				Type: domainmodel.DeleteBehaviorType(childType),
			}
		}
	}

	return ca
}

func parseAnnotation(raw map[string]any) *domainmodel.Annotation {
	annot := &domainmodel.Annotation{}

	annot.ID = model.ID(extractBsonID(raw["$ID"]))
	annot.TypeName = extractString(raw["$Type"])
	annot.Caption = extractString(raw["Caption"])

	if loc, ok := raw["Location"].(map[string]any); ok {
		annot.Location.X = extractInt(loc["x"])
		annot.Location.Y = extractInt(loc["y"])
	}

	return annot
}

func parseIndex(raw map[string]any) *domainmodel.Index {
	index := &domainmodel.Index{}

	index.ID = model.ID(extractBsonID(raw["$ID"]))
	index.Name = extractString(raw["Name"])

	// Parse index attributes
	attrs := extractBsonArray(raw["Attributes"])
	for _, a := range attrs {
		if attrMap, ok := a.(map[string]any); ok {
			// Try "AttributePointer" first (Mendix format), then "Attribute"
			attrID := extractBsonID(attrMap["AttributePointer"])
			if attrID == "" {
				attrID = extractBsonID(attrMap["Attribute"])
			}
			if attrID != "" {
				// Populate both AttributeIDs and Attributes for compatibility
				index.AttributeIDs = append(index.AttributeIDs, model.ID(attrID))

				// Parse as IndexAttribute with ascending/descending info
				// Default to ascending (true) unless explicitly set to false
				ascending := true
				if asc, ok := attrMap["Ascending"].(bool); ok {
					ascending = asc
				} else if sortOrder := extractString(attrMap["SortOrder"]); sortOrder == "Descending" {
					ascending = false
				}

				indexAttr := &domainmodel.IndexAttribute{
					AttributeID: model.ID(attrID),
					Ascending:   ascending,
				}
				index.Attributes = append(index.Attributes, indexAttr)
			}
		}
	}

	return index
}

func parseAccessRule(raw map[string]any) *domainmodel.AccessRule {
	rule := &domainmodel.AccessRule{}

	rule.ID = model.ID(extractBsonID(raw["$ID"]))
	rule.AllowCreate = extractBool(raw["AllowCreate"], false)
	rule.AllowRead = extractBool(raw["AllowRead"], false)
	rule.AllowWrite = extractBool(raw["AllowWrite"], false)
	rule.AllowDelete = extractBool(raw["AllowDelete"], false)
	rule.XPathConstraint = extractString(raw["XPathConstraint"])

	// Parse default member access rights
	if dmr := extractString(raw["DefaultMemberAccessRights"]); dmr != "" {
		rule.DefaultMemberAccessRights = domainmodel.MemberAccessRights(dmr)
	}

	// Parse module roles - try both field names (AllowedModuleRoles for newer, ModuleRoles for older)
	rolesField := raw["AllowedModuleRoles"]
	if rolesField == nil {
		rolesField = raw["ModuleRoles"]
	}
	roles := extractBsonArray(rolesField)
	for _, r := range roles {
		// Module roles can be BY_NAME (string) or BY_ID (binary)
		if name, ok := r.(string); ok {
			rule.ModuleRoleNames = append(rule.ModuleRoleNames, name)
			rule.ModuleRoles = append(rule.ModuleRoles, model.ID(name))
		} else {
			roleID := extractBsonID(r)
			if roleID != "" {
				rule.ModuleRoles = append(rule.ModuleRoles, model.ID(roleID))
			}
		}
	}

	// Parse member accesses
	memberAccesses := extractBsonArray(raw["MemberAccesses"])
	for _, ma := range memberAccesses {
		maMap := toMap(ma)
		if maMap == nil {
			continue
		}
		access := parseMemberAccess(maMap)
		rule.MemberAccesses = append(rule.MemberAccesses, access)
	}

	return rule
}

func parseMemberAccess(raw map[string]any) *domainmodel.MemberAccess {
	ma := &domainmodel.MemberAccess{}
	ma.ID = model.ID(extractBsonID(raw["$ID"]))

	// Access rights
	if ar := extractString(raw["AccessRights"]); ar != "" {
		ma.AccessRights = domainmodel.MemberAccessRights(ar)
	}

	// Attribute - BY_NAME reference (e.g., "Shop.Customer.FirstName")
	if attr := extractString(raw["Attribute"]); attr != "" {
		ma.AttributeName = attr
		ma.AttributeID = model.ID(attr)
	}

	// Association - BY_NAME reference (e.g., "Shop.Order_Customer")
	if assoc := extractString(raw["Association"]); assoc != "" {
		ma.AssociationName = assoc
		ma.AssociationID = model.ID(assoc)
	}

	return ma
}

func parseValidationRule(raw map[string]any) *domainmodel.ValidationRule {
	rule := &domainmodel.ValidationRule{}

	rule.ID = model.ID(extractBsonID(raw["$ID"]))

	// Attribute can be a qualified name like "DmTest.Cars.CarId" or an ID
	attrRef := raw["Attribute"]
	if attrID := extractBsonID(attrRef); attrID != "" {
		rule.AttributeID = model.ID(attrID)
	} else if attrName, ok := attrRef.(string); ok {
		// Store qualified name as ID - will need to resolve later
		rule.AttributeID = model.ID(attrName)
	}

	// Get rule type from RuleInfo.$Type field
	// e.g., "DomainModels$RequiredRuleInfo" -> "Required"
	if ruleInfo, ok := raw["RuleInfo"].(map[string]any); ok {
		ruleType := extractString(ruleInfo["$Type"])
		rule.Type = normalizeValidationRuleType(ruleType)
	}

	// Parse error message from "Message" field (not "ErrorMessage")
	if errMsg, ok := raw["Message"].(map[string]any); ok {
		rule.ErrorMessage = parseText(errMsg)
	} else if errMsg, ok := raw["ErrorMessage"].(map[string]any); ok {
		// Fallback for older format
		rule.ErrorMessage = parseText(errMsg)
	}

	return rule
}

// normalizeValidationRuleType converts BSON type names to simple rule types.
// e.g., "DomainModels$RequiredRuleInfo" -> "Required"
func normalizeValidationRuleType(fullType string) string {
	// Strip prefix "DomainModels$"
	if idx := strings.Index(fullType, "$"); idx >= 0 {
		fullType = fullType[idx+1:]
	}
	// Strip suffix "RuleInfo"
	if strings.HasSuffix(fullType, "RuleInfo") {
		fullType = fullType[:len(fullType)-8]
	}
	// Strip suffix "Rule" (for backward compatibility)
	if strings.HasSuffix(fullType, "Rule") {
		fullType = fullType[:len(fullType)-4]
	}
	return fullType
}

func parseEventHandler(raw map[string]any) *domainmodel.EventHandler {
	handler := &domainmodel.EventHandler{}

	handler.ID = model.ID(extractBsonID(raw["$ID"]))
	handler.Event = domainmodel.EventType(extractString(raw["Event"]))
	handler.MicroflowID = model.ID(extractBsonID(raw["Microflow"]))
	handler.RaiseErrorOnFalse = extractBool(raw["RaiseErrorOnFalse"], false)

	return handler
}

// parseMicroflow parses microflow contents from BSON.
