// SPDX-License-Identifier: Apache-2.0

// Gen-native CREATE path for service document types.
// Uses ServiceRepository.Create instead of sdk/mpr.Serialize* + InsertUnit.
// Only types where the gen encoder safely produces the correct BSON are here;
// complex serializers (DataTransformer, ImportMapping, OData, REST) stay in
// create_services_modelsdk.go until field-mapping is verified against Studio Pro.

package mprbackend

import (
	"fmt"

	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genBE "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genDB "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genJS "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// ── JsonStructure ─────────────────────────────────────────────────────────

func (b *MprBackend) createJsonStructureGen(js *types.JsonStructure) error {
	if js.ID == "" {
		js.ID = model.ID(modelsdkmpr.GenerateID())
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	g := genJS.NewJsonStructure()
	g.SetID(element.ID(string(js.ID)))
	g.SetName(js.Name)
	g.SetDocumentation(js.Documentation)
	g.SetExcluded(js.Excluded)
	if js.ExportLevel != "" {
		g.SetExportLevel(js.ExportLevel)
	}
	g.SetJsonSnippet(js.JsonSnippet)
	return mprrepos.NewServiceRepository(w).Create(string(js.ContainerID), "Documents", g)
}

// ── DatabaseConnection ────────────────────────────────────────────────────

func (b *MprBackend) createDatabaseConnectionGen(conn *model.DatabaseConnection) error {
	if conn.ID == "" {
		conn.ID = model.ID(modelsdkmpr.GenerateID())
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	g := genDB.NewDatabaseConnection()
	g.SetID(element.ID(string(conn.ID)))
	g.SetName(conn.Name)
	g.SetDocumentation(conn.Documentation)
	g.SetExcluded(conn.Excluded)
	g.SetExportLevel(conn.ExportLevel)
	g.SetDatabaseType(conn.DatabaseType)
	g.SetConnectionStringQualifiedName(conn.ConnectionString)
	g.SetUserNameQualifiedName(conn.UserName)
	g.SetPasswordQualifiedName(conn.Password)

	// ConnectionInput holds the literal JDBC URL for Studio Pro dev mode.
	connStr := genDB.NewConnectionString()
	connStr.SetValue(conn.ConnectionInputValue)
	g.SetConnectionInput(connStr)

	return mprrepos.NewServiceRepository(w).Create(string(conn.ContainerID), "Documents", g)
}

// ── BusinessEventService ──────────────────────────────────────────────────

func (b *MprBackend) createBusinessEventServiceGen(svc *model.BusinessEventService) error {
	if svc.ID == "" {
		svc.ID = model.ID(modelsdkmpr.GenerateID())
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	g := genBE.NewBusinessEventService()
	g.SetID(element.ID(string(svc.ID)))
	g.SetName(svc.Name)
	g.SetDocumentation(svc.Documentation)
	g.SetExcluded(svc.Excluded)
	g.SetExportLevel(svc.ExportLevel)
	g.SetDocument(svc.Document)
	// Definition (Part) and OperationImplementations (PartList) start empty;
	// the executor sets them via dedicated mutator operations after create.
	return mprrepos.NewServiceRepository(w).Create(string(svc.ContainerID), "Documents", g)
}
