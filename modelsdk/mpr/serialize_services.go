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
	rootElement := bson.M{
		"$ID":        idToBsonBinary(rootElemID),
		"$Type":      "DataTransformers$StructureObject",
		"Attributes": bson.A{int32(2)},
	}

	// Source
	var source bson.M
	switch strings.ToUpper(dt.SourceType) {
	case "XML":
		source = bson.M{
			"$ID":     idToBsonBinary(generateUUID()),
			"$Type":   "DataTransformers$XmlSource",
			"Content": dt.SourceJSON,
		}
	default: // JSON
		source = bson.M{
			"$ID":     idToBsonBinary(generateUUID()),
			"$Type":   "DataTransformers$JsonSource",
			"Content": dt.SourceJSON,
		}
	}

	// Steps (versioned array prefix int32(2))
	steps := bson.A{int32(2)}
	for _, step := range dt.Steps {
		var action bson.M
		switch strings.ToUpper(step.Technology) {
		case "JSLT":
			action = bson.M{
				"$ID":   idToBsonBinary(generateUUID()),
				"$Type": "DataTransformers$JsltAction",
				"Jslt":  step.Expression,
			}
		case "XSLT":
			action = bson.M{
				"$ID":   idToBsonBinary(generateUUID()),
				"$Type": "DataTransformers$XsltAction",
				"Xslt":  step.Expression,
			}
		default:
			action = bson.M{
				"$ID":   idToBsonBinary(generateUUID()),
				"$Type": "DataTransformers$JsltAction",
				"Jslt":  step.Expression,
			}
		}

		steps = append(steps, bson.M{
			"$ID":                  idToBsonBinary(generateUUID()),
			"$Type":                "DataTransformers$Step",
			"Action":               action,
			"InputElementPointer":  idToBsonBinary(rootElemID),
			"OutputElementPointer": idToBsonBinary(rootElemID),
		})
	}

	doc := bson.M{
		"$ID":                idToBsonBinary(string(dt.ID)),
		"$Type":              "DataTransformers$DataTransformer",
		"Name":               dt.Name,
		"Documentation":      "",
		"Excluded":           dt.Excluded,
		"ExportLevel":        "Hidden",
		"Source":             source,
		"Elements":           bson.A{int32(2), rootElement},
		"RootElementPointer": idToBsonBinary(rootElemID),
		"Steps":              steps,
	}

	return bson.Marshal(doc)
}

// SerializeProjectSettings returns BSON bytes for the project settings unit.
// Ported from sdk/mpr/writer_settings.go — same logic, no Writer dependency.
func SerializeProjectSettings(ps *model.ProjectSettings) ([]byte, error) {
	doc := bson.M{
		"$ID":   idToBsonBinary(string(ps.ID)),
		"$Type": "Settings$ProjectSettings",
	}

	// Rebuild the Settings array from RawParts, overwriting modified parts.
	settings := bson.A{int32(2)} // versioned array prefix

	for _, rawPart := range ps.RawParts {
		typeName, _ := rawPart["$Type"].(string)
		switch typeName {
		case "Settings$ModelSettings":
			if ps.Model != nil {
				settings = append(settings, serPSModelSettings(ps.Model, rawPart))
			} else {
				settings = append(settings, rawPart)
			}
		case "Settings$ConfigurationSettings":
			if ps.Configuration != nil {
				settings = append(settings, serPSConfigurationSettings(ps.Configuration, rawPart))
			} else {
				settings = append(settings, rawPart)
			}
		case "Settings$LanguageSettings":
			if ps.Language != nil {
				settings = append(settings, serPSLanguageSettings(ps.Language, rawPart))
			} else {
				settings = append(settings, rawPart)
			}
		case "Settings$WorkflowsProjectSettingsPart":
			if ps.Workflows != nil {
				settings = append(settings, serPSWorkflowsSettings(ps.Workflows, rawPart))
			} else {
				settings = append(settings, rawPart)
			}
		default:
			// Preserve raw part as-is (WebUI, Integration, Certificate, JarDeployment, Distribution, Convention)
			settings = append(settings, rawPart)
		}
	}

	doc["Settings"] = settings
	return bson.Marshal(doc)
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

func serPSServerConfiguration(cfg *model.ServerConfiguration) bson.M {
	cfgDoc := bson.M{
		"$Type":                         "Settings$ServerConfiguration",
		"Name":                          cfg.Name,
		"DatabaseType":                  cfg.DatabaseType,
		"DatabaseUrl":                   cfg.DatabaseUrl,
		"DatabaseName":                  cfg.DatabaseName,
		"DatabaseUserName":              cfg.DatabaseUserName,
		"DatabasePassword":              cfg.DatabasePassword,
		"DatabaseUseIntegratedSecurity": cfg.DatabaseUseIntegratedSecurity,
		"HttpPortNumber":                serPSInt64(cfg.HttpPortNumber),
		"ServerPortNumber":              serPSInt64(cfg.ServerPortNumber),
		"ApplicationRootUrl":            cfg.ApplicationRootUrl,
		"MaxJavaHeapSize":               serPSInt64(cfg.MaxJavaHeapSize),
		"ExtraJvmParameters":            cfg.ExtraJvmParameters,
		"OpenAdminPort":                 cfg.OpenAdminPort,
		"OpenHttpPort":                  cfg.OpenHttpPort,
		"CustomSettings":                bson.A{int32(2)},
		"Tracing":                       nil,
	}
	if cfg.ID != "" {
		cfgDoc["$ID"] = idToBsonBinary(string(cfg.ID))
	} else {
		cfgDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// Serialize ConstantValues (versioned array prefix)
	cvArr := bson.A{int32(2)}
	for _, cv := range cfg.ConstantValues {
		cvArr = append(cvArr, serPSConstantValue(cv))
	}
	cfgDoc["ConstantValues"] = cvArr

	return cfgDoc
}

func serPSConstantValue(cv *model.ConstantValue) bson.M {
	cvDoc := bson.M{
		"$Type":      "Settings$ConstantValue",
		"ConstantId": cv.ConstantId,
		"Value":      cv.Value,
	}
	if cv.ID != "" {
		cvDoc["$ID"] = idToBsonBinary(string(cv.ID))
	} else {
		cvDoc["$ID"] = idToBsonBinary(generateUUID())
	}
	return cvDoc
}

func serPSLanguageSettings(ls *model.LanguageSettings, raw map[string]any) map[string]any {
	raw["DefaultLanguageCode"] = ls.DefaultLanguageCode
	return raw
}

func serPSWorkflowsSettings(ws *model.WorkflowsSettings, raw map[string]any) map[string]any {
	raw["UserEntity"] = ws.UserEntity
	raw["DefaultTaskParallelism"] = serPSInt64(ws.DefaultTaskParallelism)
	raw["WorkflowEngineParallelism"] = serPSInt64(ws.WorkflowEngineParallelism)
	return raw
}
