// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// SerializeImageCollection returns BSON bytes for an image collection unit.
// Ported from sdk/mpr/writer_imagecollection.go — same logic, no Writer dependency.
func SerializeImageCollection(ic *types.ImageCollection) ([]byte, error) {
	if ic.ID == "" {
		ic.ID = model.ID(generateUUID())
	}
	if ic.ExportLevel == "" {
		ic.ExportLevel = "Hidden"
	}

	// Images array always starts with the array marker int32(3).
	images := bson.A{int32(3)}
	for i := range ic.Images {
		img := &ic.Images[i]
		if img.ID == "" {
			img.ID = model.ID(generateUUID())
		}
		images = append(images, bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(img.ID))},
			{Key: "$Type", Value: "Images$Image"},
			{Key: "Image", Value: bson.Binary{Subtype: 0, Data: img.Data}},
			{Key: "ImageFormat", Value: img.Format},
			{Key: "Name", Value: img.Name},
		})
	}

	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(ic.ID))},
		{Key: "$Type", Value: "Images$ImageCollection"},
		{Key: "Documentation", Value: ic.Documentation},
		{Key: "Excluded", Value: false},
		{Key: "ExportLevel", Value: ic.ExportLevel},
		{Key: "Images", Value: images},
		{Key: "Name", Value: ic.Name},
	}

	return bson.Marshal(doc)
}

// SerializeDataTransformer returns BSON bytes for a data transformer unit.
// Ported from sdk/mpr/writer_datatransformer.go — same logic, no Writer dependency.
func SerializeDataTransformer(dt *model.DataTransformer) ([]byte, error) {
	if dt.ID == "" {
		dt.ID = model.ID(generateUUID())
	}

	// Root element
	rootElemID := generateUUID()
	rootElement := bson.D{
		{Key: "$ID", Value: idToBsonBinary(rootElemID)},
		{Key: "$Type", Value: "DataTransformers$StructureObject"},
		{Key: "Attributes", Value: bson.A{int32(2)}},
	}

	// Source
	var source bson.D
	switch strings.ToUpper(dt.SourceType) {
	case "XML":
		source = bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTransformers$XmlSource"},
			{Key: "Content", Value: dt.SourceJSON},
		}
	default: // JSON
		source = bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTransformers$JsonSource"},
			{Key: "Content", Value: dt.SourceJSON},
		}
	}

	// Steps (versioned array prefix int32(2))
	steps := bson.A{int32(2)}
	for _, step := range dt.Steps {
		var action bson.D
		switch strings.ToUpper(step.Technology) {
		case "JSLT":
			action = bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "DataTransformers$JsltAction"},
				{Key: "Jslt", Value: step.Expression},
			}
		case "XSLT":
			action = bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "DataTransformers$XsltAction"},
				{Key: "Xslt", Value: step.Expression},
			}
		default:
			action = bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "DataTransformers$JsltAction"},
				{Key: "Jslt", Value: step.Expression},
			}
		}

		steps = append(steps, bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTransformers$Step"},
			{Key: "Action", Value: action},
			{Key: "InputElementPointer", Value: idToBsonBinary(rootElemID)},
			{Key: "OutputElementPointer", Value: idToBsonBinary(rootElemID)},
		})
	}

	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(dt.ID))},
		{Key: "$Type", Value: "DataTransformers$DataTransformer"},
		{Key: "Name", Value: dt.Name},
		{Key: "Documentation", Value: ""},
		{Key: "Excluded", Value: dt.Excluded},
		{Key: "ExportLevel", Value: "Hidden"},
		{Key: "Source", Value: source},
		{Key: "Elements", Value: bson.A{int32(2), rootElement}},
		{Key: "RootElementPointer", Value: idToBsonBinary(rootElemID)},
		{Key: "Steps", Value: steps},
	}

	return bson.Marshal(doc)
}

// SerializeProjectSettings returns BSON bytes for the project settings unit.
// Ported from sdk/mpr/writer_settings.go — same logic, no Writer dependency.
func SerializeProjectSettings(ps *model.ProjectSettings) ([]byte, error) {
	// Rebuild the Settings array from RawParts, overwriting modified parts.
	settings := bson.A{int32(2)} // versioned array prefix

	for _, rawPart := range ps.RawParts {
		typeName, _ := rawPart["$Type"].(string)
		switch typeName {
		case "Settings$ModelSettings":
			if ps.Model != nil {
				settings = append(settings, mapToOrderedBSOND(serPSModelSettings(ps.Model, rawPart)))
			} else {
				settings = append(settings, mapToOrderedBSOND(rawPart))
			}
		case "Settings$ConfigurationSettings":
			if ps.Configuration != nil {
				settings = append(settings, mapToOrderedBSOND(serPSConfigurationSettings(ps.Configuration, rawPart)))
			} else {
				settings = append(settings, mapToOrderedBSOND(rawPart))
			}
		case "Settings$LanguageSettings":
			if ps.Language != nil {
				settings = append(settings, mapToOrderedBSOND(serPSLanguageSettings(ps.Language, rawPart)))
			} else {
				settings = append(settings, mapToOrderedBSOND(rawPart))
			}
		case "Settings$WorkflowsProjectSettingsPart":
			if ps.Workflows != nil {
				settings = append(settings, mapToOrderedBSOND(serPSWorkflowsSettings(ps.Workflows, rawPart)))
			} else {
				settings = append(settings, mapToOrderedBSOND(rawPart))
			}
		default:
			// Preserve raw part as-is (WebUI, Integration, Certificate, JarDeployment, Distribution, Convention)
			settings = append(settings, mapToOrderedBSOND(rawPart))
		}
	}

	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(ps.ID))},
		{Key: "$Type", Value: "Settings$ProjectSettings"},
		{Key: "Settings", Value: settings},
	}
	return bson.Marshal(doc)
}

// mapToOrderedBSOND converts a map to bson.D with $ID and $Type first.
func mapToOrderedBSOND(m map[string]any) bson.D {
	if m == nil {
		return nil
	}
	out := make(bson.D, 0, len(m))
	if id, ok := m["$ID"]; ok {
		out = append(out, bson.E{Key: "$ID", Value: id})
	}
	if typ, ok := m["$Type"]; ok {
		out = append(out, bson.E{Key: "$Type", Value: typ})
	}
	for k, v := range m {
		if k != "$ID" && k != "$Type" {
			out = append(out, bson.E{Key: k, Value: v})
		}
	}
	return out
}

// ── Private helpers (prefixed serPS* to avoid naming conflicts) ───────────────

func serPSInt64(v int) int64 { return int64(v) }

func serPSModelSettings(ms *model.ModelSettings, raw map[string]any) map[string]any {
	raw["AfterStartupMicroflow"] = ms.AfterStartupMicroflow
	raw["BeforeShutdownMicroflow"] = ms.BeforeShutdownMicroflow
	raw["HealthCheckMicroflow"] = ms.HealthCheckMicroflow
	raw["AllowUserMultipleSessions"] = ms.AllowUserMultipleSessions
	raw["HashAlgorithm"] = ms.HashAlgorithm
	raw["BcryptCost"] = serPSInt64(ms.BcryptCost)
	raw["JavaVersion"] = ms.JavaVersion
	raw["RoundingMode"] = ms.RoundingMode
	raw["ScheduledEventTimeZoneCode"] = ms.ScheduledEventTimeZoneCode
	raw["FirstDayOfWeek"] = ms.FirstDayOfWeek
	raw["DecimalScale"] = serPSInt64(ms.DecimalScale)
	raw["EnableDataStorageOptimisticLocking"] = ms.EnableDataStorageOptimisticLocking
	raw["UseDatabaseForeignKeyConstraints"] = ms.UseDatabaseForeignKeyConstraints
	return raw
}

func serPSConfigurationSettings(cs *model.ConfigurationSettings, raw map[string]any) map[string]any {
	configs := bson.A{int32(2)} // versioned array prefix
	for _, cfg := range cs.Configurations {
		configs = append(configs, serPSServerConfiguration(cfg))
	}
	raw["Configurations"] = configs
	return raw
}

func serPSServerConfiguration(cfg *model.ServerConfiguration) bson.D {
	id := idToBsonBinary(generateUUID())
	if cfg.ID != "" {
		id = idToBsonBinary(string(cfg.ID))
	}
	cfgDoc := bson.D{
		{Key: "$ID", Value: id},
		{Key: "$Type", Value: "Settings$ServerConfiguration"},
		{Key: "Name", Value: cfg.Name},
		{Key: "DatabaseType", Value: cfg.DatabaseType},
		{Key: "DatabaseUrl", Value: cfg.DatabaseUrl},
		{Key: "DatabaseName", Value: cfg.DatabaseName},
		{Key: "DatabaseUserName", Value: cfg.DatabaseUserName},
		{Key: "DatabasePassword", Value: cfg.DatabasePassword},
		{Key: "DatabaseUseIntegratedSecurity", Value: cfg.DatabaseUseIntegratedSecurity},
		{Key: "HttpPortNumber", Value: serPSInt64(cfg.HttpPortNumber)},
		{Key: "ServerPortNumber", Value: serPSInt64(cfg.ServerPortNumber)},
		{Key: "ApplicationRootUrl", Value: cfg.ApplicationRootUrl},
		{Key: "MaxJavaHeapSize", Value: serPSInt64(cfg.MaxJavaHeapSize)},
		{Key: "ExtraJvmParameters", Value: cfg.ExtraJvmParameters},
		{Key: "OpenAdminPort", Value: cfg.OpenAdminPort},
		{Key: "OpenHttpPort", Value: cfg.OpenHttpPort},
		{Key: "CustomSettings", Value: bson.A{int32(2)}},
	}

	if cfg.Tracing != nil {
		tracingDoc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Settings$TracingConfiguration"},
			{Key: "Enabled", Value: cfg.Tracing.Enabled},
		}
		if cfg.Tracing.Endpoint != "" {
			tracingDoc = append(tracingDoc, bson.E{Key: "Endpoint", Value: cfg.Tracing.Endpoint})
		}
		if cfg.Tracing.ServiceName != "" {
			tracingDoc = append(tracingDoc, bson.E{Key: "ServiceName", Value: cfg.Tracing.ServiceName})
		}
		cfgDoc = append(cfgDoc, bson.E{Key: "Tracing", Value: tracingDoc})
	}

	if len(cfg.ConstantValues) > 0 {
		cvArr := bson.A{int32(2)}
		for _, cv := range cfg.ConstantValues {
			cvArr = append(cvArr, serPSConstantValue(cv))
		}
		cfgDoc = append(cfgDoc, bson.E{Key: "ConstantValues", Value: cvArr})
	}

	return cfgDoc
}

func serPSConstantValue(cv *model.ConstantValue) bson.D {
	id := idToBsonBinary(generateUUID())
	if cv.ID != "" {
		id = idToBsonBinary(string(cv.ID))
	}
	return bson.D{
		{Key: "$ID", Value: id},
		{Key: "$Type", Value: "Settings$ConstantValue"},
		{Key: "ConstantId", Value: cv.ConstantId},
		{Key: "Value", Value: cv.Value},
	}
}

func serPSLanguageSettings(ls *model.LanguageSettings, raw map[string]any) map[string]any {
	raw["DefaultLanguageCode"] = ls.DefaultLanguageCode

	// Rebuild the Languages array from the model (the source of truth after
	// ALTER SETTINGS LANGUAGE ADD/DROP). Preserve each existing entry's $ID by
	// matching on Code so unchanged languages keep their identity; mint a fresh
	// $ID for newly added languages.
	prev := extractBsonArrayWithMarker(raw["Languages"])
	existingIDs := map[string]any{}
	for _, item := range prev.Items {
		lm, ok := item.(map[string]any)
		if !ok {
			if d, ok := item.(bson.M); ok {
				lm = map[string]any(d)
			} else {
				continue
			}
		}
		code, _ := lm["Code"].(string)
		if id, ok := lm["$ID"]; ok && code != "" {
			existingIDs[code] = id
		}
	}

	marker := prev.Marker
	if marker == 0 {
		marker = int32(3)
	}
	langArr := bson.A{marker}
	for _, lang := range ls.Languages {
		id, ok := existingIDs[lang.Code]
		if !ok {
			id = idToBsonBinary(generateUUID())
		}
		langDoc := bson.D{
			{Key: "$ID", Value: id},
			{Key: "$Type", Value: "Texts$Language"},
			{Key: "Code", Value: lang.Code},
			{Key: "CheckCompleteness", Value: lang.CheckCompleteness},
			{Key: "CustomDateFormat", Value: lang.CustomDateFormat},
			{Key: "CustomDateTimeFormat", Value: lang.CustomDateTimeFormat},
			{Key: "CustomTimeFormat", Value: lang.CustomTimeFormat},
		}
		langArr = append(langArr, langDoc)
	}
	raw["Languages"] = langArr
	return raw
}

func serPSWorkflowsSettings(ws *model.WorkflowsSettings, raw map[string]any) map[string]any {
	raw["UserEntity"] = ws.UserEntity
	raw["DefaultTaskParallelism"] = serPSInt64(ws.DefaultTaskParallelism)
	raw["WorkflowEngineParallelism"] = serPSInt64(ws.WorkflowEngineParallelism)
	return raw
}
