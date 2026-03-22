// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"math"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// safeInt32 converts an int to int32 with clamping to prevent silent overflow.
func safeInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// UpdateProjectSettings updates the project settings document.
// The project settings document always exists, so this only needs update, not create/delete.
func (w *Writer) UpdateProjectSettings(ps *model.ProjectSettings) error {
	contents, err := w.serializeProjectSettings(ps)
	if err != nil {
		return fmt.Errorf("failed to serialize project settings: %w", err)
	}

	return w.updateUnit(string(ps.ID), contents)
}

// serializeProjectSettings converts ProjectSettings to BSON bytes.
// It uses the RawParts for round-trip fidelity, updating only the parts
// that have been parsed and modified.
func (w *Writer) serializeProjectSettings(ps *model.ProjectSettings) ([]byte, error) {
	doc := bson.M{
		"$ID":   idToBsonBinary(string(ps.ID)),
		"$Type": "Settings$ProjectSettings",
	}

	// Rebuild the Settings array from RawParts, overwriting modified parts
	settings := bson.A{int32(2)} // versioned array prefix

	for _, rawPart := range ps.RawParts {
		typeName, _ := rawPart["$Type"].(string)
		switch typeName {
		case "Settings$ModelSettings":
			if ps.Model != nil {
				settings = append(settings, serializeModelSettings(ps.Model, rawPart))
			} else {
				settings = append(settings, rawPart)
			}
		case "Settings$ConfigurationSettings":
			if ps.Configuration != nil {
				settings = append(settings, serializeConfigurationSettings(ps.Configuration, rawPart))
			} else {
				settings = append(settings, rawPart)
			}
		case "Settings$LanguageSettings":
			if ps.Language != nil {
				settings = append(settings, serializeLanguageSettings(ps.Language, rawPart))
			} else {
				settings = append(settings, rawPart)
			}
		case "Settings$WorkflowsProjectSettingsPart":
			if ps.Workflows != nil {
				settings = append(settings, serializeWorkflowsSettings(ps.Workflows, rawPart))
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

// serializeModelSettings updates the raw BSON map with modified model settings fields.
func serializeModelSettings(ms *model.ModelSettings, raw map[string]any) map[string]any {
	raw["AfterStartupMicroflow"] = ms.AfterStartupMicroflow
	raw["BeforeShutdownMicroflow"] = ms.BeforeShutdownMicroflow
	raw["HealthCheckMicroflow"] = ms.HealthCheckMicroflow
	raw["AllowUserMultipleSessions"] = ms.AllowUserMultipleSessions
	raw["HashAlgorithm"] = ms.HashAlgorithm
	raw["BcryptCost"] = safeInt32(ms.BcryptCost)
	raw["JavaVersion"] = ms.JavaVersion
	raw["RoundingMode"] = ms.RoundingMode
	raw["ScheduledEventTimeZoneCode"] = ms.ScheduledEventTimeZoneCode
	raw["FirstDayOfWeek"] = ms.FirstDayOfWeek
	raw["DecimalScale"] = safeInt32(ms.DecimalScale)
	raw["EnableDataStorageOptimisticLocking"] = ms.EnableDataStorageOptimisticLocking
	raw["UseDatabaseForeignKeyConstraints"] = ms.UseDatabaseForeignKeyConstraints
	return raw
}

// serializeConfigurationSettings updates the raw BSON map with modified configuration settings.
func serializeConfigurationSettings(cs *model.ConfigurationSettings, raw map[string]any) map[string]any {
	configs := bson.A{int32(2)} // versioned array prefix
	for _, cfg := range cs.Configurations {
		configs = append(configs, serializeServerConfiguration(cfg))
	}
	raw["Configurations"] = configs
	return raw
}

func serializeServerConfiguration(cfg *model.ServerConfiguration) bson.M {
	cfgDoc := bson.M{
		"$Type":                         "Settings$ServerConfiguration",
		"Name":                          cfg.Name,
		"DatabaseType":                  cfg.DatabaseType,
		"DatabaseUrl":                   cfg.DatabaseUrl,
		"DatabaseName":                  cfg.DatabaseName,
		"DatabaseUserName":              cfg.DatabaseUserName,
		"DatabasePassword":              cfg.DatabasePassword,
		"DatabaseUseIntegratedSecurity": cfg.DatabaseUseIntegratedSecurity,
		"HttpPortNumber":                safeInt32(cfg.HttpPortNumber),
		"ServerPortNumber":              safeInt32(cfg.ServerPortNumber),
		"ApplicationRootUrl":            cfg.ApplicationRootUrl,
		"MaxJavaHeapSize":               safeInt32(cfg.MaxJavaHeapSize),
		"ExtraJvmParameters":            cfg.ExtraJvmParameters,
		"OpenAdminPort":                 cfg.OpenAdminPort,
		"OpenHttpPort":                  cfg.OpenHttpPort,
	}
	if cfg.ID != "" {
		cfgDoc["$ID"] = idToBsonBinary(string(cfg.ID))
	} else {
		cfgDoc["$ID"] = idToBsonBinary(generateUUID())
	}

	// Serialize ConstantValues
	cvArr := bson.A{int32(2)} // versioned array prefix
	for _, cv := range cfg.ConstantValues {
		cvArr = append(cvArr, serializeConstantValue(cv))
	}
	cfgDoc["ConstantValues"] = cvArr

	return cfgDoc
}

func serializeConstantValue(cv *model.ConstantValue) bson.M {
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

// serializeLanguageSettings updates the raw BSON map with modified language settings.
func serializeLanguageSettings(ls *model.LanguageSettings, raw map[string]any) map[string]any {
	raw["DefaultLanguageCode"] = ls.DefaultLanguageCode
	return raw
}

// serializeWorkflowsSettings updates the raw BSON map with modified workflow settings.
func serializeWorkflowsSettings(ws *model.WorkflowsSettings, raw map[string]any) map[string]any {
	raw["UserEntity"] = ws.UserEntity
	raw["DefaultTaskParallelism"] = safeInt32(ws.DefaultTaskParallelism)
	raw["WorkflowEngineParallelism"] = safeInt32(ws.WorkflowEngineParallelism)
	return raw
}
