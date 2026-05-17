// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genBE "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genDBC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatransformers"
	gendomainmodels "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genExpMap "github.com/mendixlabs/mxcli/modelsdk/gen/exportmappings"
	genImg "github.com/mendixlabs/mxcli/modelsdk/gen/images"
	genImpMap "github.com/mendixlabs/mxcli/modelsdk/gen/importmappings"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
	genJson "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	genODataPub "github.com/mendixlabs/mxcli/modelsdk/gen/odatapublish"
	genRest "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
	gensecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
	"github.com/spf13/cobra"
)

// TreeNode represents a node in the project tree JSON output.
type TreeNode struct {
	Label         string      `json:"label"`
	Type          string      `json:"type"`
	QualifiedName string      `json:"qualifiedName,omitempty"`
	Children      []*TreeNode `json:"children,omitempty"`
}

// treeElement holds a name, type, and container ID for building the tree hierarchy.
type treeElement struct {
	Name        string
	Type        string
	ContainerID model.ID
	Children    []*TreeNode // optional pre-built children (for expandable documents)
}

var projectTreeCmd = &cobra.Command{
	Use:   "project-tree",
	Short: "Output the project structure as JSON",
	Long: `Output the full Mendix project structure as a JSON tree.

Each module contains categories (Domain Model, Microflows, Pages, etc.)
with their elements organized into folder hierarchies.

This command is designed for use by IDE integrations (e.g., VS Code TreeView).

Example:
  mxcli project-tree -p app.mpr
  mxcli project-tree -p app.mpr | python3 -m json.tool
`,
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, _ := cmd.Flags().GetString("project")
		if projectPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --project (-p) is required")
			os.Exit(1)
		}

		tree, err := buildProjectTree(projectPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(tree); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	},
}

func buildProjectTree(projectPath string) ([]*TreeNode, error) {
	// project-tree consumes ~30 sdk/mpr.Reader methods (ListEnumerations,
	// ListConstants, GetProjectSettings, ListBusinessEventServices, ...) that
	// have no modelsdk/mpr.Reader equivalent today. Read-path migration is
	// deferred to Phase 4; keep the sdk/mpr reader here until then.
	reader, err := sdkmpr.Open(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open project: %w", err)
	}
	defer reader.Close()

	// Open a parallel modelsdk/mpr reader for microflow/nanoflow listing.
	// Both readers open the same .mpr in read-only mode; their SQLite
	// connections are independent. mprread.List* returns gen-typed
	// elements that callers can later iterate using modelsdk semantics
	// (Stage 4 Task A migration off sdk/mpr microflow APIs).
	mreader, err := mmpr.Open(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open project (modelsdk reader): %w", err)
	}
	defer mreader.Close()

	h, err := executor.NewContainerHierarchy(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to build hierarchy: %w", err)
	}

	modules, err := reader.ListModules()
	if err != nil {
		return nil, fmt.Errorf("failed to list modules: %w", err)
	}

	// Build per-module data
	// Domain model items (entities, associations, enumerations) go into Domain Model container
	// Other documents are organized by folder
	type moduleData struct {
		// Domain Model items (no folder hierarchy)
		entities     []treeElement
		associations []treeElement
		enumerations []treeElement
		// Security items
		moduleRoles []treeElement
		// Documents with folder hierarchy
		documents []treeElement
	}

	modData := make(map[model.ID]*moduleData)
	for _, m := range modules {
		modData[m.ID] = &moduleData{}
	}

	// Collect entities and associations from domain models (gen-typed path).
	// gen-typed DomainModel loses Container() linkage on codec roundtrip,
	// so resolve ContainerID by joining against the raw unit index.
	dmRefs, _ := mreader.ListUnitsByType("DomainModels$DomainModel")
	dmContainerByID := make(map[string]model.ID, len(dmRefs))
	for _, ref := range dmRefs {
		dmContainerByID[ref.ID] = model.ID(ref.ContainerID)
	}
	dms, _ := mprread.ListDomainModels(mreader)
	for _, dm := range dms {
		containerID := dmContainerByID[string(dm.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		for _, entElem := range dm.EntitiesItems() {
			ent, ok := entElem.(*gendomainmodels.Entity)
			if !ok {
				continue
			}
			md.entities = append(md.entities, treeElement{Name: ent.Name(), ContainerID: containerID})
		}
		for _, assocElem := range dm.AssociationsItems() {
			assoc, ok := assocElem.(*gendomainmodels.Association)
			if !ok {
				continue
			}
			md.associations = append(md.associations, treeElement{Name: assoc.Name(), ContainerID: containerID})
		}
	}

	// Collect enumerations (part of Domain Model)
	enumUnits, _ := mprread.ListUnitsWithContainer[*genEnum.Enumeration](mreader)
	for _, u := range enumUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.enumerations = append(md.enumerations, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID})
	}

	// Collect module security (module roles).
	// gen-typed ModuleSecurity has no ContainerID field; resolve it from
	// the raw unit index which carries the SQLite ContainerID column.
	msRefs, _ := mreader.ListUnitsByType("Security$ModuleSecurity")
	msContainerByID := make(map[string]model.ID, len(msRefs))
	for _, ref := range msRefs {
		msContainerByID[ref.ID] = model.ID(ref.ContainerID)
	}
	allMS, _ := reader.ListModuleSecurity()
	for _, ms := range allMS {
		containerID := msContainerByID[string(ms.ID())]
		md, ok := modData[containerID]
		if !ok {
			continue
		}
		for _, item := range ms.ModuleRolesItems() {
			mr, ok := item.(*gensecurity.ModuleRole)
			if !ok {
				continue
			}
			md.moduleRoles = append(md.moduleRoles, treeElement{Name: mr.Name(), ContainerID: containerID})
		}
	}

	// Collect microflows (modelsdk-native gen path).
	// gen-typed Microflow loses Container() linkage on codec roundtrip,
	// so resolve ContainerID by joining mprread results against the
	// raw UnitRef list keyed on unit ID.
	mfRefs, _ := mreader.ListUnitsByType("Microflows$Microflow")
	mfContainerByID := make(map[string]model.ID, len(mfRefs))
	for _, ref := range mfRefs {
		if ref.Type == "Microflows$Microflow" {
			mfContainerByID[ref.ID] = model.ID(ref.ContainerID)
		}
	}
	mfs, _ := mprread.ListMicroflows(mreader)
	for _, mf := range mfs {
		containerID := mfContainerByID[string(mf.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: mf.Name(), ContainerID: containerID, Type: "microflow"})
	}

	// Collect nanoflows (modelsdk-native gen path).
	nfRefs, _ := mreader.ListUnitsByType("Microflows$Nanoflow")
	nfContainerByID := make(map[string]model.ID, len(nfRefs))
	for _, ref := range nfRefs {
		if ref.Type == "Microflows$Nanoflow" {
			nfContainerByID[ref.ID] = model.ID(ref.ContainerID)
		}
	}
	nfs, _ := mprread.ListNanoflows(mreader)
	for _, nf := range nfs {
		containerID := nfContainerByID[string(nf.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: nf.Name(), ContainerID: containerID, Type: "nanoflow"})
	}

	// Collect pages (gen-typed path; ContainerID resolved via unit index join).
	pgRefs, _ := mreader.ListUnitsByType("Forms$Page")
	pgContainerByID := make(map[string]model.ID, len(pgRefs))
	for _, ref := range pgRefs {
		if ref.Type == "Forms$Page" {
			pgContainerByID[ref.ID] = model.ID(ref.ContainerID)
		}
	}
	pgs, _ := mprread.ListPages(mreader)
	for _, pg := range pgs {
		containerID := pgContainerByID[string(pg.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: pg.Name(), ContainerID: containerID, Type: "page"})
	}

	// Collect snippets (gen-typed path; ContainerID resolved via unit index join).
	snRefs, _ := mreader.ListUnitsByType("Forms$Snippet")
	snContainerByID := make(map[string]model.ID, len(snRefs))
	for _, ref := range snRefs {
		snContainerByID[ref.ID] = model.ID(ref.ContainerID)
	}
	sns, _ := mprread.ListSnippets(mreader)
	for _, sn := range sns {
		containerID := snContainerByID[string(sn.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: sn.Name(), ContainerID: containerID, Type: "snippet"})
	}

	// Collect layouts (gen-typed path; ContainerID resolved via unit index join).
	lyRefs, _ := mreader.ListUnitsByType("Forms$Layout")
	lyContainerByID := make(map[string]model.ID, len(lyRefs))
	for _, ref := range lyRefs {
		lyContainerByID[ref.ID] = model.ID(ref.ContainerID)
	}
	lys, _ := mprread.ListLayouts(mreader)
	for _, ly := range lys {
		containerID := lyContainerByID[string(ly.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: ly.Name(), ContainerID: containerID, Type: "layout"})
	}

	// Collect constants
	constUnits, _ := mprread.ListUnitsWithContainer[*genConst.Constant](mreader)
	for _, u := range constUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "constant"})
	}

	// Collect workflows (gen-typed path; ContainerID resolved via unit index join).
	wfRefs, _ := mreader.ListUnitsByType("Workflows$Workflow")
	wfContainerByID := make(map[string]model.ID, len(wfRefs))
	for _, ref := range wfRefs {
		if ref.Type == "Workflows$Workflow" {
			wfContainerByID[ref.ID] = model.ID(ref.ContainerID)
		}
	}
	wfs, _ := mprread.ListWorkflows(mreader)
	for _, wf := range wfs {
		containerID := wfContainerByID[string(wf.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: wf.Name(), ContainerID: containerID, Type: "workflow"})
	}

	// Collect java actions (gen-typed path; ContainerID resolved via unit index join).
	jaRefs, _ := mreader.ListUnitsByType("JavaActions$JavaAction")
	jaContainerByID := make(map[string]model.ID, len(jaRefs))
	for _, ref := range jaRefs {
		if ref.Type == "JavaActions$JavaAction" {
			jaContainerByID[ref.ID] = model.ID(ref.ContainerID)
		}
	}
	jas, _ := mprread.ListJavaActions(mreader)
	for _, ja := range jas {
		containerID := jaContainerByID[string(ja.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: ja.Name(), ContainerID: containerID, Type: "javaaction"})
	}

	// Collect scheduled events
	seUnits, _ := mprread.ListUnitsWithContainer[*genSched.ScheduledEvent](mreader)
	for _, u := range seUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "scheduledevent"})
	}

	// Collect JavaScript actions
	jsaUnits, _ := mprread.ListUnitsWithContainer[*genJSA.JavaScriptAction](mreader)
	for _, u := range jsaUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "javascriptaction"})
	}

	// Collect building blocks (gen-typed path; ContainerID resolved via unit index join).
	bbRefs, _ := mreader.ListUnitsByType("Forms$BuildingBlock")
	bbContainerByID := make(map[string]model.ID, len(bbRefs))
	for _, ref := range bbRefs {
		bbContainerByID[ref.ID] = model.ID(ref.ContainerID)
	}
	bbs, _ := mprread.ListBuildingBlocks(mreader)
	for _, bb := range bbs {
		containerID := bbContainerByID[string(bb.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: bb.Name(), ContainerID: containerID, Type: "buildingblock"})
	}

	// Collect page templates (gen-typed path; ContainerID resolved via unit index join).
	ptRefs, _ := mreader.ListUnitsByType("Forms$PageTemplate")
	ptContainerByID := make(map[string]model.ID, len(ptRefs))
	for _, ref := range ptRefs {
		ptContainerByID[ref.ID] = model.ID(ref.ContainerID)
	}
	pts, _ := mprread.ListPageTemplates(mreader)
	for _, pt := range pts {
		containerID := ptContainerByID[string(pt.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: pt.Name(), ContainerID: containerID, Type: "pagetemplate"})
	}

	// Collect image collections
	icUnits, _ := mprread.ListUnitsWithContainer[*genImg.ImageCollection](mreader)
	for _, u := range icUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "imagecollection"})
	}

	// Collect consumed OData services (clients)
	odataClientUnits, _ := mprread.ListUnitsWithContainer[*genRest.ConsumedODataService](mreader)
	for _, u := range odataClientUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "odataclient"})
	}

	// Collect published OData services (with entity sets as children)
	odataServiceUnits, _ := mprread.ListUnitsWithContainer[*genODataPub.PublishedODataService2](mreader)
	for _, u := range odataServiceUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		children := buildPublishedODataChildren(u.Element)
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "odataservice", Children: children})
	}

	// Collect published REST services (with resources/operations as children)
	restServiceUnits, _ := mprread.ListUnitsWithContainer[*genRest.PublishedRestService](mreader)
	for _, u := range restServiceUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		children := buildPublishedRestChildren(u.Element)
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "publishedrestservice", Children: children})
	}

	// Collect business event services (with channels/messages as children)
	besUnits, _ := mprread.ListUnitsWithContainer[*genBE.BusinessEventService](mreader)
	for _, u := range besUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		children := buildBusinessEventChildren(u.Element)
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "businesseventservice", Children: children})
	}

	// Collect JSON structures
	jsUnits, _ := mprread.ListUnitsWithContainer[*genJson.JsonStructure](mreader)
	for _, u := range jsUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "jsonstructure"})
	}

	// Collect import mappings
	imUnits, _ := mprread.ListUnitsWithContainer[*genImpMap.ImportMapping](mreader)
	for _, u := range imUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "importmapping"})
	}

	// Collect export mappings
	emUnits, _ := mprread.ListUnitsWithContainer[*genExpMap.ExportMapping](mreader)
	for _, u := range emUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "exportmapping"})
	}

	// Collect consumed REST services (with operations as children)
	restClientUnits, _ := mprread.ListUnitsWithContainer[*genRest.ConsumedRestService](mreader)
	for _, u := range restClientUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		children := buildConsumedRestChildren(u.Element)
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "restclient", Children: children})
	}

	// Collect database connections (with queries as children)
	dbcUnits, _ := mprread.ListUnitsWithContainer[*genDBC.DatabaseConnection](mreader)
	for _, u := range dbcUnits {
		dbc := u.Element
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		children := buildDatabaseConnectionChildren(dbc)
		md.documents = append(md.documents, treeElement{Name: dbc.Name(), ContainerID: u.ContainerID, Type: "databaseconnection", Children: children})
	}

	// Collect data transformers
	dtUnits, _ := mprread.ListUnitsWithContainer[*genDT.DataTransformer](mreader)
	for _, u := range dtUnits {
		modID := h.FindModuleID(u.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: u.Element.Name(), ContainerID: u.ContainerID, Type: "datatransformer"})
	}

	// Collect agent-editor documents (Mendix 11.9+; empty on older projects)
	agentModels, _ := mprread.ListAgentEditorModels(mreader)
	for _, m := range agentModels {
		modID := h.FindModuleID(m.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: m.Name, ContainerID: m.ContainerID, Type: "aimodel"})
	}

	kbs, _ := mprread.ListAgentEditorKnowledgeBases(mreader)
	for _, kb := range kbs {
		modID := h.FindModuleID(kb.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: kb.Name, ContainerID: kb.ContainerID, Type: "knowledgebase"})
	}

	mcpServices, _ := mprread.ListAgentEditorConsumedMCPServices(mreader)
	for _, svc := range mcpServices {
		modID := h.FindModuleID(svc.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: svc.Name, ContainerID: svc.ContainerID, Type: "consumedmcpservice"})
	}

	agents, _ := mprread.ListAgentEditorAgents(mreader)
	for _, a := range agents {
		modID := h.FindModuleID(a.ContainerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		md.documents = append(md.documents, treeElement{Name: a.Name, ContainerID: a.ContainerID, Type: "agent"})
	}

	// Mark external entities in domain model.
	// External entities have a source of type Rest$ODataRemoteEntitySource.
	for _, dm := range dms {
		containerID := dmContainerByID[string(dm.ID())]
		modID := h.FindModuleID(containerID)
		md, ok := modData[modID]
		if !ok {
			continue
		}
		for _, entElem := range dm.EntitiesItems() {
			ent, ok := entElem.(*gendomainmodels.Entity)
			if !ok {
				continue
			}
			// Source() returns the raw BSON source object; check type via IsRemote
			// or inspect the source part's type name. We use the entity's Name()
			// to locate the tree element and set it to externalentity.
			// The gen Entity has a source part — check via Source() != nil
			// and check the source element's TypeName.
			src := ent.Source()
			if src == nil {
				continue
			}
			if src.TypeName() == "Rest$ODataRemoteEntitySource" {
				for j := range md.entities {
					if md.entities[j].Name == ent.Name() {
						md.entities[j].Type = "externalentity"
						break
					}
				}
			}
		}
	}

	// Build tree nodes
	var tree []*TreeNode
	for _, m := range modules {
		md := modData[m.ID]
		modNode := &TreeNode{
			Label:         m.Name,
			Type:          "module",
			QualifiedName: m.Name,
		}

		// Domain Model container (entities, associations, enumerations)
		if len(md.entities) > 0 || len(md.associations) > 0 || len(md.enumerations) > 0 {
			dmNode := &TreeNode{
				Label:         "Domain Model",
				Type:          "domainmodel",
				QualifiedName: m.Name,
			}

			// Add entities (regular or external)
			for _, ent := range md.entities {
				entType := "entity"
				if ent.Type == "externalentity" {
					entType = "externalentity"
				}
				dmNode.Children = append(dmNode.Children, &TreeNode{
					Label:         ent.Name,
					Type:          entType,
					QualifiedName: m.Name + "." + ent.Name,
				})
			}

			// Add associations
			for _, assoc := range md.associations {
				dmNode.Children = append(dmNode.Children, &TreeNode{
					Label:         assoc.Name,
					Type:          "association",
					QualifiedName: m.Name + "." + assoc.Name,
				})
			}

			// Add enumerations
			for _, en := range md.enumerations {
				dmNode.Children = append(dmNode.Children, &TreeNode{
					Label:         en.Name,
					Type:          "enumeration",
					QualifiedName: m.Name + "." + en.Name,
				})
			}

			// Sort domain model children alphabetically
			sort.Slice(dmNode.Children, func(i, j int) bool {
				return dmNode.Children[i].Label < dmNode.Children[j].Label
			})

			modNode.Children = append(modNode.Children, dmNode)
		}

		// Security container (module roles)
		if len(md.moduleRoles) > 0 {
			secNode := &TreeNode{Label: "Security", Type: "security"}
			for _, mr := range md.moduleRoles {
				secNode.Children = append(secNode.Children, &TreeNode{
					Label:         mr.Name,
					Type:          "modulerole",
					QualifiedName: m.Name + "." + mr.Name,
				})
			}
			sort.Slice(secNode.Children, func(i, j int) bool {
				return secNode.Children[i].Label < secNode.Children[j].Label
			})
			modNode.Children = append(modNode.Children, secNode)
		}

		// Build folder hierarchy for documents (microflows, pages, etc.)
		if len(md.documents) > 0 {
			docChildren := buildDocumentHierarchy(h, md.documents, m.Name)
			modNode.Children = append(modNode.Children, docChildren...)
		}

		tree = append(tree, modNode)
	}

	// Project Security top-level node
	ps, err := reader.GetProjectSecurity()
	if err == nil {
		psNode := &TreeNode{Label: "Project Security", Type: "projectsecurity", QualifiedName: "ProjectSecurity"}

		// User Roles category
		if urItems := ps.UserRolesItems(); len(urItems) > 0 {
			urNode := &TreeNode{Label: "User Roles", Type: "category"}
			for _, item := range urItems {
				ur, ok := item.(*gensecurity.UserRole)
				if !ok {
					continue
				}
				urNode.Children = append(urNode.Children, &TreeNode{
					Label:         ur.Name(),
					Type:          "userrole",
					QualifiedName: ur.Name(),
				})
			}
			sort.Slice(urNode.Children, func(i, j int) bool {
				return urNode.Children[i].Label < urNode.Children[j].Label
			})
			psNode.Children = append(psNode.Children, urNode)
		}

		// Demo Users category
		if ps.EnableDemoUsers() {
			if duItems := ps.DemoUsersItems(); len(duItems) > 0 {
				duNode := &TreeNode{Label: "Demo Users", Type: "category"}
				for _, item := range duItems {
					du, ok := item.(*gensecurity.DemoUser)
					if !ok {
						continue
					}
					duNode.Children = append(duNode.Children, &TreeNode{
						Label:         du.UserName(),
						Type:          "demouser",
						QualifiedName: du.UserName(),
					})
				}
				sort.Slice(duNode.Children, func(i, j int) bool {
					return duNode.Children[i].Label < duNode.Children[j].Label
				})
				psNode.Children = append(psNode.Children, duNode)
			}
		}

		tree = append([]*TreeNode{psNode}, tree...)
	}

	// Project Settings top-level node
	settings, settingsErr := reader.GetProjectSettings()
	if settingsErr == nil {
		settingsNode := &TreeNode{Label: "Settings", Type: "settings", QualifiedName: "Settings"}
		if settings.Model != nil {
			modelNode := &TreeNode{Label: "Model", Type: "settingscategory"}
			if settings.Model.AfterStartupMicroflow != "" {
				modelNode.Children = append(modelNode.Children, &TreeNode{
					Label: "After Startup: " + settings.Model.AfterStartupMicroflow,
					Type:  "settingsitem",
				})
			}
			if settings.Model.BeforeShutdownMicroflow != "" {
				modelNode.Children = append(modelNode.Children, &TreeNode{
					Label: "Before Shutdown: " + settings.Model.BeforeShutdownMicroflow,
					Type:  "settingsitem",
				})
			}
			if len(modelNode.Children) > 0 {
				settingsNode.Children = append(settingsNode.Children, modelNode)
			}
		}
		if settings.Language != nil && settings.Language.DefaultLanguageCode != "" {
			settingsNode.Children = append(settingsNode.Children, &TreeNode{
				Label: "Default Language: " + settings.Language.DefaultLanguageCode,
				Type:  "settingsitem",
			})
		}
		tree = append([]*TreeNode{settingsNode}, tree...)
	}

	// Navigation top-level node
	nav, navErr := reader.GetNavigation()
	if navErr == nil && len(nav.Profiles) > 0 {
		navNode := &TreeNode{Label: "Navigation", Type: "navigation", QualifiedName: "Navigation"}
		for _, profile := range nav.Profiles {
			profileNode := &TreeNode{
				Label:         profile.Kind,
				Type:          "navprofile",
				QualifiedName: profile.Kind,
			}

			// Home page
			if profile.HomePage != nil {
				target := profile.HomePage.Page
				if target == "" {
					target = profile.HomePage.Microflow
				}
				if target != "" {
					profileNode.Children = append(profileNode.Children, &TreeNode{
						Label: "Home: " + target,
						Type:  "navhome",
					})
				}
			}

			// Role-based home pages
			for _, rbh := range profile.RoleBasedHomePages {
				target := rbh.Page
				if target == "" {
					target = rbh.Microflow
				}
				if target != "" {
					profileNode.Children = append(profileNode.Children, &TreeNode{
						Label: "Home (" + rbh.UserRole + "): " + target,
						Type:  "navhome",
					})
				}
			}

			// Login page
			if profile.LoginPage != "" {
				profileNode.Children = append(profileNode.Children, &TreeNode{
					Label: "Login: " + profile.LoginPage,
					Type:  "navlogin",
				})
			}

			// Menu items
			if len(profile.MenuItems) > 0 {
				menuNode := &TreeNode{Label: "Menu", Type: "navmenu"}
				buildMenuTreeNodes(menuNode, profile.MenuItems)
				profileNode.Children = append(profileNode.Children, menuNode)
			}

			navNode.Children = append(navNode.Children, profileNode)
		}
		tree = append([]*TreeNode{navNode}, tree...)
	}

	// Add System Overview node at the top of the tree
	overviewNode := &TreeNode{
		Label:         "System Overview",
		Type:          "systemoverview",
		QualifiedName: "SystemOverview",
	}
	tree = append([]*TreeNode{overviewNode}, tree...)

	// Sort modules alphabetically (skip non-module nodes at front)
	startIdx := 0
	for startIdx < len(tree) && (tree[startIdx].Type == "systemoverview" || tree[startIdx].Type == "projectsecurity" || tree[startIdx].Type == "navigation") {
		startIdx++
	}
	moduleSlice := tree[startIdx:]
	sort.Slice(moduleSlice, func(i, j int) bool {
		return moduleSlice[i].Label < moduleSlice[j].Label
	})

	return tree, nil
}

// buildDocumentHierarchy organizes documents into folder trees based on their container hierarchy.
// Documents without folders appear at the top level (returned directly).
// Folders contain their documents regardless of document type.
func buildDocumentHierarchy(h *executor.ContainerHierarchy, elements []treeElement, moduleName string) []*TreeNode {
	// Group elements by folder path
	type pathElement struct {
		folderPath string
		name       string
		elemType   string
		children   []*TreeNode
	}

	var items []pathElement
	for _, el := range elements {
		fp := h.BuildFolderPath(el.ContainerID)
		items = append(items, pathElement{folderPath: fp, name: el.Name, elemType: el.Type, children: el.Children})
	}

	// Sort by folder path then name
	sort.Slice(items, func(i, j int) bool {
		if items[i].folderPath != items[j].folderPath {
			return items[i].folderPath < items[j].folderPath
		}
		return items[i].name < items[j].name
	})

	// Build folder tree
	root := &TreeNode{Type: "root"}
	folderNodes := make(map[string]*TreeNode)

	for _, item := range items {
		parent := root
		if item.folderPath != "" {
			parent = getOrCreateFolder(root, folderNodes, item.folderPath)
		}
		leaf := &TreeNode{
			Label:         item.name,
			Type:          item.elemType,
			QualifiedName: moduleName + "." + item.name,
			Children:      item.children,
		}
		parent.Children = append(parent.Children, leaf)
	}

	// Sort folders before documents, then alphabetically
	sortChildren(root)
	for _, folder := range folderNodes {
		sortChildren(folder)
	}

	return root.Children
}

// sortChildren sorts a node's children: folders first, then documents, alphabetically within each group.
func sortChildren(node *TreeNode) {
	if len(node.Children) == 0 {
		return
	}
	sort.Slice(node.Children, func(i, j int) bool {
		iIsFolder := node.Children[i].Type == "folder"
		jIsFolder := node.Children[j].Type == "folder"
		if iIsFolder != jIsFolder {
			return iIsFolder // folders come first
		}
		return node.Children[i].Label < node.Children[j].Label
	})
}

// getOrCreateFolder finds or creates a folder node hierarchy for the given path.
func getOrCreateFolder(root *TreeNode, cache map[string]*TreeNode, path string) *TreeNode {
	if node, ok := cache[path]; ok {
		return node
	}

	// Split path into parts
	parts := splitFolderPath(path)
	current := root
	builtPath := ""

	for _, part := range parts {
		if builtPath != "" {
			builtPath += "/"
		}
		builtPath += part

		if node, ok := cache[builtPath]; ok {
			current = node
			continue
		}

		folderNode := &TreeNode{
			Label: part,
			Type:  "folder",
		}
		current.Children = append(current.Children, folderNode)
		cache[builtPath] = folderNode
		current = folderNode
	}

	return current
}

// buildPublishedODataChildren builds child tree nodes for a published OData service.
// Shows entity sets with their exposed entities.
func buildPublishedODataChildren(svc *genODataPub.PublishedODataService2) []*TreeNode {
	var children []*TreeNode

	// Index entity types by their element ID for O(1) lookup from EntitySet.EntityTypeRefID().
	etByID := map[element.ID]*genODataPub.EntityType{}
	var etItems []*genODataPub.EntityType
	for _, item := range svc.EntityTypesItems() {
		if et, ok := item.(*genODataPub.EntityType); ok {
			etByID[et.ID()] = et
			etItems = append(etItems, et)
		}
	}

	esItems := svc.EntitySetsItems()
	for _, esItem := range esItems {
		es, ok := esItem.(*genODataPub.EntitySet)
		if !ok {
			continue
		}
		label := es.ExposedName()
		var et *genODataPub.EntityType
		if refID := es.EntityTypeRefID(); refID != "" {
			et = etByID[refID]
			if et != nil && et.EntityQualifiedName() != "" {
				label += " → " + et.EntityQualifiedName()
			}
		}
		esNode := &TreeNode{
			Label: label,
			Type:  "odataentityset",
		}
		if et != nil {
			for _, memItem := range et.ChildMembersItems() {
				memLabel, kind := publishedMemberLabel(memItem)
				if memLabel == "" {
					continue
				}
				if kind != "" {
					memLabel += " (" + kind + ")"
				}
				esNode.Children = append(esNode.Children, &TreeNode{
					Label: memLabel,
					Type:  "odatamember",
				})
			}
		}
		children = append(children, esNode)
	}

	// If no entity sets but entity types exist, show entity types directly
	if len(esItems) == 0 {
		for _, et := range etItems {
			label := et.ExposedName()
			if et.EntityQualifiedName() != "" {
				label += " → " + et.EntityQualifiedName()
			}
			children = append(children, &TreeNode{
				Label: label,
				Type:  "odataentityset",
			})
		}
	}

	return children
}

// publishedMemberLabel extracts the exposed name and kind from a published OData
// member. The "kind" string mirrors what sdk/mpr.parser inferred from the BSON
// $Type: "attribute", "association", "id", or "microflow".
func publishedMemberLabel(item element.Element) (label, kind string) {
	switch m := item.(type) {
	case *genODataPub.PublishedAttribute:
		return m.ExposedName(), "attribute"
	case *genODataPub.PublishedAssociationEnd:
		return m.ExposedName(), "association"
	case *genODataPub.PublishedId:
		return m.ExposedName(), "id"
	case *genODataPub.PublishedMicroflow:
		return m.ExposedName(), "microflow"
	case *genODataPub.PublishedMember:
		return m.ExposedName(), ""
	}
	return "", ""
}

// buildPublishedRestChildren builds child tree nodes for a published REST service.
// Shows resources with their operations (like an OpenAPI contract).
func buildPublishedRestChildren(svc *genRest.PublishedRestService) []*TreeNode {
	var children []*TreeNode
	for _, resItem := range svc.ResourcesItems() {
		res, ok := resItem.(*genRest.PublishedRestServiceResource)
		if !ok {
			continue
		}
		resNode := &TreeNode{
			Label: res.Name(),
			Type:  "restresource",
		}
		for _, opItem := range res.OperationsItems() {
			op, ok := opItem.(*genRest.PublishedRestServiceOperation)
			if !ok {
				continue
			}
			method := op.HttpMethod()
			if method == "" {
				method = "GET"
			}
			label := method
			if op.Path() != "" {
				label += " " + op.Path()
			}
			if op.Summary() != "" {
				label += " — " + op.Summary()
			}
			resNode.Children = append(resNode.Children, &TreeNode{
				Label: label,
				Type:  "restoperation",
			})
		}
		children = append(children, resNode)
	}
	return children
}

// buildConsumedRestChildren builds child tree nodes for a consumed REST service
// (REST client). Shows each operation with method and path.
func buildConsumedRestChildren(svc *genRest.ConsumedRestService) []*TreeNode {
	var children []*TreeNode
	for _, opItem := range svc.OperationsItems() {
		op, ok := opItem.(*genRest.RestOperation)
		if !ok {
			continue
		}
		method := restOperationMethod(op.Method())
		if method == "" {
			method = "GET"
		}
		label := method
		if path := restOperationPath(op.Path()); path != "" {
			label += " " + path
		}
		children = append(children, &TreeNode{
			Label: label,
			Type:  "restoperation",
		})
	}
	return children
}

// restOperationMethod extracts the HTTP method string from any of the
// RestOperationMethod variants (with/without body).
func restOperationMethod(m element.Element) string {
	type httpMethoder interface {
		HttpMethod() string
	}
	if hm, ok := m.(httpMethoder); ok {
		return hm.HttpMethod()
	}
	return ""
}

// restOperationPath extracts the URL path string from a RestOperation.Path()
// element. The most common variant is ValueTemplate (literal path with
// {placeholders}). ConstantValue points to a Constant document — its name is
// stored in ValueQualifiedName but resolving to the runtime string would
// require a Constants$Constant lookup, so we surface the reference inline.
func restOperationPath(p element.Element) string {
	switch v := p.(type) {
	case *genRest.ValueTemplate:
		return v.Value()
	case *genRest.ConstantValue:
		if name := v.ValueQualifiedName(); name != "" {
			return "$" + name
		}
		return ""
	}
	return ""
}

// buildBusinessEventChildren builds child tree nodes for a business event service.
// Shows channels with their messages.
func buildBusinessEventChildren(svc *genBE.BusinessEventService) []*TreeNode {
	var children []*TreeNode
	defItem := svc.Definition()
	if defItem == nil {
		return children
	}
	def, ok := defItem.(*genBE.BusinessEventDefinition)
	if !ok {
		return children
	}
	for _, chItem := range def.ChannelsItems() {
		ch, ok := chItem.(*genBE.Channel)
		if !ok {
			continue
		}
		chNode := &TreeNode{
			Label: ch.ChannelName(),
			Type:  "bechannel",
		}
		for _, msgItem := range ch.MessagesItems() {
			msg, ok := msgItem.(*genBE.Message)
			if !ok {
				continue
			}
			direction := ""
			if msg.CanPublish() && msg.CanSubscribe() {
				direction = " (pub/sub)"
			} else if msg.CanPublish() {
				direction = " (publish)"
			} else if msg.CanSubscribe() {
				direction = " (subscribe)"
			}
			msgNode := &TreeNode{
				Label: msg.MessageName() + direction,
				Type:  "bemessage",
			}
			for _, attrItem := range msg.AttributesItems() {
				attr, ok := attrItem.(*genBE.MessageAttribute)
				if !ok {
					continue
				}
				attrLabel := attr.AttributeName()
				if typeName := attributeTypeName(attr.AttributeType()); typeName != "" {
					attrLabel += " : " + typeName
				}
				msgNode.Children = append(msgNode.Children, &TreeNode{
					Label: attrLabel,
					Type:  "beattribute",
				})
			}
			chNode.Children = append(chNode.Children, msgNode)
		}
		children = append(children, chNode)
	}
	return children
}

// attributeTypeName resolves a BusinessEvent message attribute's nested type
// element to its short BSON-type name suffix (e.g. "String" from
// "DataTypes$StringType"). Returns "" if the element is nil.
func attributeTypeName(t element.Element) string {
	if t == nil {
		return ""
	}
	tn := t.TypeName()
	// e.g. "DataTypes$StringType" → "StringType"
	if i := indexByte(tn, '$'); i >= 0 {
		return tn[i+1:]
	}
	return tn
}

// indexByte is a tiny inline strings.IndexByte (avoid importing strings just
// for this single helper). Returns -1 if not found.
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// buildDatabaseConnectionChildren builds child tree nodes for a database connection.
// Shows queries with their parameters and table mappings.
func buildDatabaseConnectionChildren(dbc *genDBC.DatabaseConnection) []*TreeNode {
	var children []*TreeNode
	for _, qItem := range dbc.QueriesItems() {
		q, ok := qItem.(*genDBC.DatabaseQuery)
		if !ok {
			continue
		}
		qNode := &TreeNode{
			Label: q.Name(),
			Type:  "dbquery",
		}
		// Show parameters
		for _, pItem := range q.ParametersItems() {
			p, ok := pItem.(*genDBC.QueryParameter)
			if !ok {
				continue
			}
			pLabel := p.ParameterName()
			if typeName := sqlDataTypeName(p.SqlDataType()); typeName != "" {
				pLabel += " : " + typeName
			}
			qNode.Children = append(qNode.Children, &TreeNode{
				Label: pLabel,
				Type:  "dbqueryparam",
			})
		}
		// Show table mappings
		for _, tmItem := range q.TableMappingsItems() {
			tm, ok := tmItem.(*genDBC.TableMapping)
			if !ok {
				continue
			}
			tmLabel := tm.TableName()
			if tm.EntityQualifiedName() != "" {
				tmLabel += " → " + tm.EntityQualifiedName()
			}
			tmNode := &TreeNode{
				Label: tmLabel,
				Type:  "dbtablemapping",
			}
			for _, colItem := range tm.ColumnsItems() {
				col, ok := colItem.(*genDBC.ColumnMapping)
				if !ok {
					continue
				}
				colLabel := col.ColumnName()
				if col.AttributeQualifiedName() != "" {
					colLabel += " → " + col.AttributeQualifiedName()
				}
				tmNode.Children = append(tmNode.Children, &TreeNode{
					Label: colLabel,
					Type:  "dbcolumnmapping",
				})
			}
			qNode.Children = append(qNode.Children, tmNode)
		}
		children = append(children, qNode)
	}
	return children
}

// sqlDataTypeName extracts the SQL data type name (e.g. "VARCHAR", "INTEGER")
// from any of the SqlDataType variants.
func sqlDataTypeName(t element.Element) string {
	type dataTypeNamer interface {
		DataTypeName() string
	}
	if dn, ok := t.(dataTypeNamer); ok {
		return dn.DataTypeName()
	}
	return ""
}

// buildMenuTreeNodes recursively builds tree nodes from navigation menu items.
func buildMenuTreeNodes(parent *TreeNode, items []*types.NavMenuItem) {
	for _, item := range items {
		label := item.Caption
		if label == "" {
			label = "(unnamed)"
		}
		if item.Page != "" {
			label += " → " + item.Page
		} else if item.Microflow != "" {
			label += " → " + item.Microflow
		}

		node := &TreeNode{
			Label: label,
			Type:  "navmenuitem",
		}

		if len(item.Items) > 0 {
			buildMenuTreeNodes(node, item.Items)
		}

		parent.Children = append(parent.Children, node)
	}
}

// splitFolderPath splits a folder path like "Parent/Child" into parts.
func splitFolderPath(path string) []string {
	if path == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}
