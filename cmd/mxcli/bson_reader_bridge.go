// SPDX-License-Identifier: Apache-2.0

// bsonReader abstracts the sdk/mpr.Reader for BSON tool commands.
// Consolidates the sdk/mpr dependency to this single file so that
// cmd_bson_*.go do not import sdk/mpr directly.
package main

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDM "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	gensecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	genWorkflows "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// bsonReader exposes the BSON inspection methods needed by cmd_bson_*.go.
// Satisfied by *sdkmpr.Reader; future implementations can use modelsdk/mpr.
type bsonReader interface {
	GetRawUnitByName(objectType, qualifiedName string) (*types.RawUnitInfo, error)
	ListRawUnits(objectType string) ([]*types.RawUnitInfo, error)
	Close() error
}

// openBSONReader opens a project at path and returns a bsonReader.
func openBSONReader(path string) (bsonReader, error) {
	return sdkmpr.Open(path)
}

// widgetReader exposes the widget inspection methods needed by cmd_extract_templates.go.
// Satisfied by *sdkmpr.Reader.
type widgetReader interface {
	GetMendixVersion() (string, error)
	ListAllCustomWidgetTypes() ([]*types.RawCustomWidgetType, error)
	Close() error
}

// openWidgetReader opens a project at path and returns a widgetReader.
func openWidgetReader(path string) (widgetReader, error) {
	return sdkmpr.Open(path)
}

// projectTreeReader abstracts sdk/mpr.Reader for the project tree command.
// Satisfied by *sdkmpr.Reader.
type projectTreeReader interface {
	Close() error
	GetNavigation() (*types.NavigationDocument, error)
	GetProjectSecurity() (*gensecurity.ProjectSecurity, error)
	GetProjectSettings() (*model.ProjectSettings, error)
	ListAgentEditorAgents() ([]*types.Agent, error)
	ListAgentEditorConsumedMCPServices() ([]*types.ConsumedMCPService, error)
	ListAgentEditorKnowledgeBases() ([]*types.KnowledgeBase, error)
	ListAgentEditorModels() ([]*types.Model, error)
	ListBuildingBlocksGen() ([]*genPg.BuildingBlock, error)
	ListBusinessEventServices() ([]*model.BusinessEventService, error)
	ListConstants() ([]*model.Constant, error)
	ListConsumedODataServices() ([]*model.ConsumedODataService, error)
	ListConsumedRestServices() ([]*model.ConsumedRestService, error)
	ListDataTransformers() ([]*model.DataTransformer, error)
	ListDatabaseConnections() ([]*model.DatabaseConnection, error)
	ListDomainModelsGen() ([]*genDM.DomainModel, error)
	ListEnumerations() ([]*model.Enumeration, error)
	ListExportMappings() ([]*model.ExportMapping, error)
	ListFolders() ([]*types.FolderInfo, error)
	ListImageCollections() ([]*types.ImageCollection, error)
	ListImportMappings() ([]*model.ImportMapping, error)
	ListJavaActionsGen() ([]*genJA.JavaAction, error)
	ListJavaScriptActions() ([]*types.JavaScriptAction, error)
	ListJsonStructures() ([]*types.JsonStructure, error)
	ListLayoutsGen() ([]*genPg.Layout, error)
	ListModuleSecurity() ([]*gensecurity.ModuleSecurity, error)
	ListModules() ([]*model.Module, error)
	ListUnits() ([]*types.UnitInfo, error)
	ListPageTemplatesGen() ([]*genPg.PageTemplate, error)
	ListPagesGen() ([]*genPg.Page, error)
	ListPublishedODataServices() ([]*model.PublishedODataService, error)
	ListPublishedRestServices() ([]*model.PublishedRestService, error)
	ListScheduledEvents() ([]*model.ScheduledEvent, error)
	ListSnippetsGen() ([]*genPg.Snippet, error)
	ListWorkflowsGen() ([]*genWorkflows.Workflow, error)
}

// openProjectTreeReader opens a project at path and returns a projectTreeReader.
func openProjectTreeReader(path string) (projectTreeReader, error) {
	return sdkmpr.Open(path)
}
