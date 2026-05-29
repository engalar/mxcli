// SPDX-License-Identifier: Apache-2.0

package types

// Unit type name constants for MPR BSON $Type fields.
// These are the storage names used by Mendix Studio Pro — not the SDK
// qualified names. Use these constants instead of raw string literals
// when switching on UnitInfo.Type or unitCache entries.
const (
	// Core project structure
	UnitTypeModule       = "Projects$ModuleImpl"
	UnitTypeModuleSettings = "Projects$ModuleSettings"
	UnitTypeFolder       = "Projects$Folder"
	UnitTypeProjectSettings = "Settings$ProjectSettings"

	// Domain model
	UnitTypeDomainModel  = "DomainModels$DomainModel"
	UnitTypeEnumeration  = "Enumerations$Enumeration"
	UnitTypeConstant     = "Constants$Constant"

	// Flows
	UnitTypeMicroflow    = "Microflows$Microflow"
	UnitTypeNanoflow     = "Microflows$Nanoflow"
	UnitTypeRule         = "Microflows$Rule"

	// Pages / UI
	UnitTypePage         = "Forms$Page"
	UnitTypeLayout       = "Forms$Layout"
	UnitTypeSnippet      = "Forms$Snippet"

	// Java / JavaScript actions
	UnitTypeJavaAction       = "JavaActions$JavaAction"
	UnitTypeJavaScriptAction = "JavaActions$JavaScriptAction"

	// Workflows
	UnitTypeWorkflow     = "Workflows$Workflow"

	// REST / OData / Business events
	UnitTypePublishedRestService = "Rest$PublishedRestService"
	UnitTypePublishedODataService = "ODataPublish$PublishedODataService"
	UnitTypeConsumedODataService  = "Rest$ConsumedODataService"
	UnitTypeBusinessEvent         = "BusinessEvents$" // prefix — multiple sub-types

	// Database connector
	UnitTypeDatabaseConnection = "DatabaseConnector$DatabaseConnection"

	// Images
	UnitTypeImageCollection = "Images$ImageCollection"
)
