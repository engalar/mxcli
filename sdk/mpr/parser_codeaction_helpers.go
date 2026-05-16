// SPDX-License-Identifier: Apache-2.0

// Package mpr - Shared helpers for Java/JavaScript action BSON parsing.
// These were previously in parser_javaactions.go; extracted so that
// parseJavaScriptAction (parser_misc.go) can use them after the Java action
// read path was retired.
package mpr

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// parseCodeActionReturnType parses a code action return type from raw BSON.
func parseCodeActionReturnType(raw map[string]any) types.CodeActionReturnType {
	if raw == nil {
		return nil
	}

	typeName := extractString(raw["$Type"])
	switch typeName {
	case "CodeActions$VoidType":
		return &types.VoidType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$BooleanType":
		return &types.BooleanType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$IntegerType":
		return &types.IntegerType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$LongType":
		return &types.LongType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$DecimalType":
		return &types.DecimalType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$StringType":
		return &types.StringType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$DateTimeType":
		return &types.DateTimeType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$EntityType", "CodeActions$ConcreteEntityType":
		et := &types.EntityType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		et.Entity = extractString(raw["Entity"])
		return et
	case "CodeActions$ListType":
		lt := &types.ListType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		// ListType can have Entity directly or Parameter containing ConcreteEntityType
		if entity := extractString(raw["Entity"]); entity != "" {
			lt.Entity = entity
		} else if param := toMap(raw["Parameter"]); param != nil {
			lt.Entity = extractString(param["Entity"])
		}
		return lt
	case "CodeActions$FileDocumentType":
		return &types.FileDocumentType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$EnumerationType":
		et := &types.EnumerationType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		et.Enumeration = extractString(raw["Enumeration"])
		return et
	case "CodeActions$TypeParameter":
		tp := &types.TypeParameter{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		tp.TypeParameter = extractString(raw["TypeParameter"])
		return tp
	case "CodeActions$ParameterizedEntityType":
		// Return type referencing a type parameter (e.g., returns the entity passed as type param)
		tp := &types.TypeParameter{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		// ParameterizedEntityType stores the type parameter as a binary ID pointer
		id := extractBsonID(raw["TypeParameterPointer"])
		if id == "" {
			id = extractBsonID(raw["TypeParameter"])
		}
		tp.TypeParameterID = model.ID(id)
		return tp
	}

	// Unknown type - return nil
	return nil
}

// parseJavaActionParameter parses a Java/JavaScript action parameter from raw BSON.
func parseJavaActionParameter(raw map[string]any) *types.JavaActionParameter {
	if raw == nil {
		return nil
	}

	// Skip array markers (items without $ID)
	if raw["$ID"] == nil {
		return nil
	}

	param := &types.JavaActionParameter{}
	param.ID = model.ID(extractBsonID(raw["$ID"]))
	param.TypeName = extractString(raw["$Type"])
	param.Name = extractString(raw["Name"])
	param.Description = extractString(raw["Description"])
	param.Category = extractString(raw["Category"])
	param.IsRequired = extractBool(raw["IsRequired"], false)

	// Parse parameter type - handle both map and primitive.D
	switch pt := raw["ParameterType"].(type) {
	case map[string]any:
		param.ParameterType = parseCodeActionParameterType(pt)
	case primitive.D:
		param.ParameterType = parseCodeActionParameterType(primitiveToMap(pt))
	}

	return param
}

// parseCodeActionParameterType parses a code action parameter type from raw BSON.
func parseCodeActionParameterType(raw map[string]any) types.CodeActionParameterType {
	if raw == nil {
		return nil
	}

	typeName := extractString(raw["$Type"])
	switch typeName {
	case "CodeActions$BasicParameterType":
		// BasicParameterType wraps the actual type in a "Type" property
		innerType := toMap(raw["Type"])
		if innerType != nil {
			return parseInnerParameterType(innerType)
		}
		return nil
	case "CodeActions$BooleanType":
		return &types.BooleanType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$IntegerType":
		return &types.IntegerType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$LongType":
		return &types.LongType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$DecimalType":
		return &types.DecimalType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$StringType":
		return &types.StringType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$DateTimeType":
		return &types.DateTimeType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$EntityType", "CodeActions$ConcreteEntityType":
		et := &types.EntityType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		et.Entity = extractString(raw["Entity"])
		return et
	case "CodeActions$ListType":
		lt := &types.ListType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		// ListType can have Entity directly or Parameter containing ConcreteEntityType
		if entity := extractString(raw["Entity"]); entity != "" {
			lt.Entity = entity
		} else if param := toMap(raw["Parameter"]); param != nil {
			lt.Entity = extractString(param["Entity"])
		}
		return lt
	case "CodeActions$StringTemplateParameterType":
		st := &types.StringTemplateParameterType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		st.Grammar = extractString(raw["Grammar"])
		return st
	case "CodeActions$FileDocumentType":
		return &types.FileDocumentType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$EnumerationType":
		et := &types.EnumerationType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		et.Enumeration = extractString(raw["Enumeration"])
		return et
	case "CodeActions$MicroflowType", "JavaActions$MicroflowJavaActionParameterType":
		return &types.MicroflowType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$TypeParameter":
		tp := &types.TypeParameter{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		tp.TypeParameter = extractString(raw["TypeParameter"])
		return tp
	case "CodeActions$EntityTypeParameterType":
		etpt := &types.EntityTypeParameterType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		// Studio Pro uses "TypeParameterPointer"; fall back to "TypeParameter" for backward compat
		id := extractBsonID(raw["TypeParameterPointer"])
		if id == "" {
			id = extractBsonID(raw["TypeParameter"])
		}
		etpt.TypeParameterID = model.ID(id)
		return etpt
	case "JavaScriptActions$NanoflowJavaScriptActionParameterType":
		return &types.NanoflowType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	}

	// Unknown type - return nil
	return nil
}

// parseInnerParameterType parses the inner type from BasicParameterType.
func parseInnerParameterType(raw map[string]any) types.CodeActionParameterType {
	if raw == nil {
		return nil
	}

	typeName := extractString(raw["$Type"])
	switch typeName {
	case "CodeActions$BooleanType":
		return &types.BooleanType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$IntegerType":
		return &types.IntegerType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$DecimalType":
		return &types.DecimalType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$StringType":
		return &types.StringType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$DateTimeType":
		return &types.DateTimeType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$MicroflowType", "JavaActions$MicroflowJavaActionParameterType":
		return &types.MicroflowType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
	case "CodeActions$ConcreteEntityType", "CodeActions$EntityType":
		et := &types.EntityType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		et.Entity = extractString(raw["Entity"])
		return et
	case "CodeActions$ListType":
		lt := &types.ListType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		// ListType contains Parameter with ConcreteEntityType
		if param := toMap(raw["Parameter"]); param != nil {
			lt.Entity = extractString(param["Entity"])
		} else if entity := extractString(raw["Entity"]); entity != "" {
			lt.Entity = entity
		}
		return lt
	case "CodeActions$EntityTypeParameterType":
		etpt := &types.EntityTypeParameterType{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		// Studio Pro uses "TypeParameterPointer"; fall back to "TypeParameter" for backward compat
		id := extractBsonID(raw["TypeParameterPointer"])
		if id == "" {
			id = extractBsonID(raw["TypeParameter"])
		}
		etpt.TypeParameterID = model.ID(id)
		return etpt
	case "CodeActions$ParameterizedEntityType":
		tp := &types.TypeParameter{
			BaseElement: model.BaseElement{ID: model.ID(extractBsonID(raw["$ID"]))},
		}
		// ParameterizedEntityType stores the type parameter as a binary ID pointer
		id := extractBsonID(raw["TypeParameterPointer"])
		if id == "" {
			id = extractBsonID(raw["TypeParameter"])
		}
		tp.TypeParameterID = model.ID(id)
		return tp
	}

	return nil
}

// toMap converts various BSON types to map[string]interface{}.
func toMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	switch m := v.(type) {
	case map[string]any:
		return m
	case primitive.D:
		return primitiveToMap(m)
	default:
		return nil
	}
}

// primitiveToMap converts primitive.D to map[string]interface{}.
func primitiveToMap(d primitive.D) map[string]any {
	result := make(map[string]any)
	for _, e := range d {
		result[e.Key] = e.Value
	}
	return result
}
