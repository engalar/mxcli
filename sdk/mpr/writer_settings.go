// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// safeInt64 converts an int to int64.
func safeInt64(v int) int64 {
	return int64(v)
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
	raw["BcryptCost"] = safeInt64(ms.BcryptCost)
	raw["JavaVersion"] = ms.JavaVersion
	raw["RoundingMode"] = ms.RoundingMode
	raw["ScheduledEventTimeZoneCode"] = ms.ScheduledEventTimeZoneCode
	raw["FirstDayOfWeek"] = ms.FirstDayOfWeek
	raw["DecimalScale"] = safeInt64(ms.DecimalScale)
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
		"HttpPortNumber":                safeInt64(cfg.HttpPortNumber),
		"ServerPortNumber":              safeInt64(cfg.ServerPortNumber),
		"ApplicationRootUrl":            cfg.ApplicationRootUrl,
		"MaxJavaHeapSize":               safeInt64(cfg.MaxJavaHeapSize),
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
	raw["DefaultTaskParallelism"] = safeInt64(ws.DefaultTaskParallelism)
	raw["WorkflowEngineParallelism"] = safeInt64(ws.WorkflowEngineParallelism)
	return raw
}
