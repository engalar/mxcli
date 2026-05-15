// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genRest "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
)

// buildExternalEntities populates the external_entities catalog table
// from domain model entities that have an OData remote entity source.
func (b *Builder) buildExternalEntities() error {
	domainModels, err := b.cachedDomainModelsGen()
	if err != nil {
		return err
	}

	stmt, err := b.tx.Prepare(`
		INSERT INTO external_entities (Id, Name, QualifiedName, ModuleName,
			ServiceName, EntitySet, RemoteName,
			Countable, Creatable, Deletable, Updatable, AttributeCount,
			ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	projectID, projectName, snapshotID, snapshotDate, snapshotSource, _, _, _ := b.snapshotMeta()

	count := 0
	for _, dm := range domainModels {
		if dm == nil {
			continue
		}
		moduleID := b.hierarchy.findModuleID(model.ID(dm.ID()))
		moduleName := b.hierarchy.getModuleName(moduleID)

		for _, entityElem := range dm.EntitiesItems() {
			entity, ok := entityElem.(*genDm.Entity)
			if !ok {
				continue
			}
			source, ok := entity.Source().(*genRest.ODataRemoteEntitySource)
			if !ok || source == nil {
				continue
			}

			qualifiedName := moduleName + "." + entity.Name()

			boolToInt := func(b bool) int {
				if b {
					return 1
				}
				return 0
			}

			_, err := stmt.Exec(
				string(entity.ID()),
				entity.Name(),
				qualifiedName,
				moduleName,
				source.SourceDocumentQualifiedName(),
				source.EntitySet(),
				source.RemoteName(),
				boolToInt(source.Countable()),
				boolToInt(source.Creatable()),
				boolToInt(source.Deletable()),
				0, // Updatable is not exposed on ODataRemoteEntitySource in gen.
				len(entity.AttributesItems()),
				projectID, projectName, snapshotID, snapshotDate, snapshotSource,
			)
			if err != nil {
				return err
			}
			count++
		}
	}

	b.report("External Entities", count)
	return nil
}

// buildExternalActions populates the external_actions catalog table
// by scanning all microflows and nanoflows for CallExternalAction activities.
func (b *Builder) buildExternalActions() error {
	mfs, err := b.cachedMicroflows()
	if err != nil {
		return err
	}
	nfs, err := b.cachedNanoflows()
	if err != nil {
		return err
	}

	stmt, err := b.tx.Prepare(`
		INSERT INTO external_actions (Id, ServiceName, ActionName, ModuleName,
			UsageCount, CallerNames, ParameterNames,
			ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	projectID, projectName, snapshotID, snapshotDate, snapshotSource, _, _, _ := b.snapshotMeta()

	// Collect unique actions: key = service + "|" + action name
	type actionInfo struct {
		service    string
		actionName string
		module     string
		params     []string
		callers    []string
		count      int
	}
	actionMap := make(map[string]*actionInfo)

	extractActions := func(oc *genMf.MicroflowObjectCollection, flowModule, flowName string) {
		if oc == nil {
			return
		}
		for _, obj := range oc.ObjectsItems() {
			act, ok := obj.(*genMf.ActionActivity)
			if !ok {
				continue
			}
			inner := act.Action()
			if inner == nil {
				continue
			}
			cea, ok := inner.(*genMf.CallExternalAction)
			if !ok {
				continue
			}

			service := cea.ConsumedODataServiceQualifiedName()
			actionName := cea.Name()
			paramItems := cea.ParameterMappingsItems()

			key := service + "|" + actionName
			info, exists := actionMap[key]
			if !exists {
				var params []string
				for _, pm := range paramItems {
					params = append(params, externalParamName(pm))
				}
				info = &actionInfo{
					service:    service,
					actionName: actionName,
					module:     flowModule,
					params:     params,
				}
				actionMap[key] = info
			}
			info.count++
			caller := flowModule + "." + flowName
			// Avoid duplicate caller entries
			found := false
			for _, c := range info.callers {
				if c == caller {
					found = true
					break
				}
			}
			if !found {
				info.callers = append(info.callers, caller)
			}
			// Merge parameter names from different call sites
			if len(paramItems) > len(info.params) {
				info.params = nil
				for _, pm := range paramItems {
					info.params = append(info.params, externalParamName(pm))
				}
			}
		}
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modID := b.hierarchy.findModuleID(model.ID(mf.ID()))
		modName := b.hierarchy.getModuleName(modID)
		extractActions(flowObjectCollection(mf.ObjectCollection()), modName, mf.Name())
	}
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modID := b.hierarchy.findModuleID(model.ID(nf.ID()))
		modName := b.hierarchy.getModuleName(modID)
		extractActions(flowObjectCollection(nf.ObjectCollection()), modName, nf.Name())
	}

	for _, info := range actionMap {
		syntheticID := fmt.Sprintf("%x", sha256.Sum256([]byte(info.service+"|"+info.actionName)))[:32]

		_, err := stmt.Exec(
			syntheticID,
			info.service,
			info.actionName,
			info.module,
			info.count,
			strings.Join(info.callers, ", "),
			strings.Join(info.params, ", "),
			projectID, projectName, snapshotID, snapshotDate, snapshotSource,
		)
		if err != nil {
			return err
		}
	}

	b.report("External Actions", len(actionMap))
	return nil
}

// externalParamName extracts the parameter-name string from a gen
// ExternalActionParameterMapping element. Returns "" for unexpected
// element types so the caller's collected []string keeps positional
// alignment with ParameterMappingsItems().
func externalParamName(e any) string {
	pm, ok := e.(*genMf.ExternalActionParameterMapping)
	if !ok || pm == nil {
		return ""
	}
	return pm.ParameterName()
}
