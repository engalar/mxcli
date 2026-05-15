// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func (b *Builder) buildAssociations() error {
	domainModels, err := b.cachedDomainModelsGen()
	if err != nil {
		return err
	}

	// Build entity ID -> qualified name lookup (reuse already-parsed domain models).
	moduleNames := make(map[model.ID]string)
	entityNames := make(map[model.ID]string)
	for _, dm := range domainModels {
		if dm == nil {
			continue
		}
		modID := b.hierarchy.findModuleID(model.ID(dm.ID()))
		modName := b.hierarchy.getModuleName(modID)
		moduleNames[model.ID(dm.ID())] = modName
		for _, entityElem := range dm.EntitiesItems() {
			entity, ok := entityElem.(*genDm.Entity)
			if !ok {
				continue
			}
			entityNames[model.ID(entity.ID())] = modName + "." + entity.Name()
		}
	}

	stmt, err := b.tx.Prepare(`
		INSERT INTO associations (Id, Name, QualifiedName, ModuleName,
			FromEntity, ToEntity, AssociationType, Owner, StorageFormat, Description,
			ProjectId, ProjectName, SnapshotId, SnapshotDate, SnapshotSource,
			SourceId, SourceBranch, SourceRevision)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	projectID, projectName, snapshotID, snapshotDate, snapshotSource, sourceID, sourceBranch, sourceRevision := b.snapshotMeta()

	count := 0
	for _, dm := range domainModels {
		if dm == nil {
			continue
		}
		modName := moduleNames[model.ID(dm.ID())]

		for _, assocElem := range dm.AssociationsItems() {
			assoc, ok := assocElem.(*genDm.Association)
			if !ok {
				continue
			}
			from := entityNames[model.ID(assoc.ParentRefID())]
			if from == "" {
				from = string(assoc.ParentRefID())
			}
			to := entityNames[model.ID(assoc.ChildRefID())]
			if to == "" {
				to = string(assoc.ChildRefID())
			}
			_, err := stmt.Exec(
				string(assoc.ID()),
				assoc.Name(),
				modName+"."+assoc.Name(),
				modName,
				from,
				to,
				assoc.Type(),
				assoc.Owner(),
				assoc.StorageFormat(),
				assoc.Documentation(),
				projectID, projectName, snapshotID, snapshotDate, snapshotSource,
				sourceID, sourceBranch, sourceRevision,
			)
			if err != nil {
				return err
			}
			count++
		}

		for _, crossElem := range dm.CrossAssociationsItems() {
			ca, ok := crossElem.(*genDm.CrossAssociation)
			if !ok {
				continue
			}
			from := entityNames[model.ID(ca.ParentRefID())]
			if from == "" {
				from = string(ca.ParentRefID())
			}
			_, err := stmt.Exec(
				string(ca.ID()),
				ca.Name(),
				modName+"."+ca.Name(),
				modName,
				from,
				ca.ChildQualifiedName(),
				ca.Type(),
				ca.Owner(),
				ca.StorageFormat(),
				ca.Documentation(),
				projectID, projectName, snapshotID, snapshotDate, snapshotSource,
				sourceID, sourceBranch, sourceRevision,
			)
			if err != nil {
				return err
			}
			count++
		}
	}

	b.report("Associations", count)
	return nil
}
