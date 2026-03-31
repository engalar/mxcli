// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"github.com/mendixlabs/mxcli/sdk/pages"

	"go.mongodb.org/mongo-driver/bson"
)

// ============================================================================
// Client Action Serialization
// ============================================================================

// SerializeClientAction serializes a ClientAction to BSON.
// This is the exported version for use by the pluggable widget engine.
func SerializeClientAction(action pages.ClientAction) bson.D {
	return serializeClientAction(action)
}

// serializeClientAction serializes a ClientAction.
func serializeClientAction(action pages.ClientAction) bson.D {
	if action == nil {
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Forms$NoAction"},
			{Key: "DisabledDuringExecution", Value: true},
		}
	}

	switch a := action.(type) {
	case *pages.SaveChangesClientAction:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$SaveChangesClientAction"},
			{Key: "ClosePage", Value: a.ClosePage},
			{Key: "SyncAutomatically", Value: true},
		}
	case *pages.CancelChangesClientAction:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$CancelChangesClientAction"},
			{Key: "ClosePage", Value: a.ClosePage},
		}
	case *pages.ClosePageClientAction:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$ClosePageClientAction"},
		}
	case *pages.DeleteClientAction:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$DeleteClientAction"},
			{Key: "ClosePage", Value: a.ClosePage},
		}
	case *pages.CreateObjectClientAction:
		// Build EntityRef if entity is specified
		var entityRef any
		if a.EntityName != "" {
			entityRef = bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "DomainModels$DirectEntityRef"},
				{Key: "Entity", Value: a.EntityName},
			}
		}
		// Build PageSettings (Forms$FormSettings) - always required, even if no page specified
		pageSettings := bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Forms$FormSettings"},
			{Key: "Form", Value: a.PageName}, // BY_NAME_REFERENCE - qualified name, or empty string if no page
		}
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$CreateObjectClientAction"},
			{Key: "DisabledDuringExecution", Value: true},
			{Key: "EntityRef", Value: entityRef},
			{Key: "NumberOfPagesToClose2", Value: ""},
			{Key: "PageSettings", Value: pageSettings},
		}
	case *pages.PageClientAction:
		// Build ParameterMappings if any
		paramMappings := bson.A{int32(len(a.ParameterMappings))}
		for _, pm := range a.ParameterMappings {
			// Parameter is BY_NAME_REFERENCE: PageName.ParameterName
			paramRef := a.PageName + "." + pm.ParameterName

			// Determine the argument value
			var argument any
			if pm.Variable != "" {
				argument = pm.Variable // e.g., "$Customer"
			} else if pm.Expression != "" {
				argument = pm.Expression
			}

			mapping := bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "Forms$PageParameterMapping"},
				{Key: "Argument", Value: argument},
				{Key: "Parameter", Value: paramRef}, // BY_NAME_REFERENCE
				{Key: "Variable", Value: nil},
			}
			paramMappings = append(paramMappings, mapping)
		}

		// Use FormSettings with Form (by-name reference) like Studio Pro does
		formSettings := bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Forms$FormSettings"},
			{Key: "Form", Value: a.PageName}, // BY_NAME_REFERENCE - qualified name
			{Key: "ParameterMappings", Value: paramMappings},
		}

		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$FormAction"},
			{Key: "DisabledDuringExecution", Value: true},
			{Key: "FormSettings", Value: formSettings},
		}
	case *pages.MicroflowClientAction:
		// Build ParameterMappings if any
		paramMappings := bson.A{int32(len(a.ParameterMappings))}
		for _, pm := range a.ParameterMappings {
			// Parameter is BY_NAME_REFERENCE: MicroflowName.ParameterName
			paramRef := a.MicroflowName + "." + pm.ParameterName

			// Determine the expression value
			var expression string
			if pm.Variable != "" {
				expression = pm.Variable // e.g., "$Customer"
			} else if pm.Expression != "" {
				expression = pm.Expression
			}

			mapping := bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "Forms$MicroflowParameterMapping"},
				{Key: "Expression", Value: expression},
				{Key: "Parameter", Value: paramRef}, // BY_NAME_REFERENCE
				{Key: "Variable", Value: nil},
			}
			paramMappings = append(paramMappings, mapping)
		}

		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$MicroflowAction"},
			{Key: "MicroflowSettings", Value: bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "Forms$MicroflowSettings"},
				{Key: "Microflow", Value: a.MicroflowName},
				{Key: "ParameterMappings", Value: paramMappings},
				{Key: "ProgressBar", Value: "None"},
				{Key: "ProgressMessage", Value: bson.D{
					{Key: "$ID", Value: idToBsonBinary(generateUUID())},
					{Key: "$Type", Value: "Texts$Text"},
					{Key: "Items", Value: bson.A{int32(3)}},
				}},
				{Key: "Asynchronous", Value: false},
				{Key: "FormValidations", Value: "All"},
				{Key: "ConfirmationInfo", Value: nil},
			}},
			{Key: "DisabledDuringExecution", Value: true},
		}
	case *pages.NanoflowClientAction:
		// Build ParameterMappings if any
		nfParamMappings := bson.A{int32(len(a.ParameterMappings))}
		for _, pm := range a.ParameterMappings {
			// Parameter is BY_NAME_REFERENCE: NanoflowName.ParameterName
			paramRef := a.NanoflowName + "." + pm.ParameterName

			// Determine the expression value
			var expression string
			if pm.Variable != "" {
				expression = pm.Variable // e.g., "$Customer"
			} else if pm.Expression != "" {
				expression = pm.Expression
			}

			mapping := bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "Forms$NanoflowParameterMapping"},
				{Key: "Expression", Value: expression},
				{Key: "Parameter", Value: paramRef}, // BY_NAME_REFERENCE
				{Key: "Variable", Value: nil},
			}
			nfParamMappings = append(nfParamMappings, mapping)
		}

		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$CallNanoflowClientAction"},
			{Key: "Nanoflow", Value: a.NanoflowName},
			{Key: "ParameterMappings", Value: nfParamMappings},
			{Key: "ProgressBar", Value: "None"},
			{Key: "ProgressMessage", Value: bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "Texts$Text"},
				{Key: "Items", Value: bson.A{int32(3)}},
			}},
			{Key: "ConfirmationInfo", Value: nil},
			{Key: "DisabledDuringExecution", Value: true},
		}
	case *pages.NoClientAction:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(a.ID))},
			{Key: "$Type", Value: "Forms$NoAction"},
		}
	default:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Forms$NoAction"},
		}
	}
}
