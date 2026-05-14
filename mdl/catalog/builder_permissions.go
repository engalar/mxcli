// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"database/sql"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// buildPermissions extracts security permissions from all documents.
// This is only run in full mode as it requires parsing all documents.
func (b *Builder) buildPermissions() error {
	if !b.fullMode {
		return nil
	}

	stmt, err := b.tx.Prepare(`
		INSERT INTO permissions (ModuleRoleName, ElementType, ElementName, MemberName, AccessType, XPathConstraint, ModuleName, ProjectId, SnapshotId)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	projectID := b.catalog.projectID
	snapshotID := b.snapshot.ID
	permCount := 0

	permCount += b.buildEntityPermissions(stmt, projectID, snapshotID)
	permCount += b.buildMicroflowPermissions(stmt, projectID, snapshotID)
	permCount += b.buildPagePermissions(stmt, projectID, snapshotID)
	permCount += b.buildODataServicePermissions(stmt, projectID, snapshotID)

	b.report("Permissions", permCount)
	return nil
}

// buildEntityPermissions extracts entity-level and member-level access
// permissions on the gen-typed read path (Stage 3.3.4 C2.b).
func (b *Builder) buildEntityPermissions(stmt *sql.Stmt, projectID, snapshotID string) int {
	count := 0

	dms, err := b.cachedDomainModelsGen()
	if err != nil {
		return 0
	}

	for _, dm := range dms {
		if dm == nil {
			continue
		}
		moduleID := b.hierarchy.findModuleID(model.ID(dm.ID()))
		moduleName := b.hierarchy.getModuleName(moduleID)

		for _, e := range dm.EntitiesItems() {
			ent, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			entityQN := moduleName + "." + ent.Name()

			for _, r := range ent.AccessRulesItems() {
				rule, ok := r.(*genDm.AccessRule)
				if !ok {
					continue
				}
				roleNames := rule.ModuleRolesQualifiedNames()
				if len(roleNames) == 0 {
					continue
				}

				xpath := rule.XPathConstraint()
				hasRead, hasWrite := entityAccessFromMemberRightsGen(rule)

				for _, roleName := range roleNames {
					if rule.AllowCreate() {
						stmt.Exec(roleName, "ENTITY", entityQN, nil, "CREATE", xpath, moduleName, projectID, snapshotID)
						count++
					}
					if hasRead {
						stmt.Exec(roleName, "ENTITY", entityQN, nil, "READ", xpath, moduleName, projectID, snapshotID)
						count++
					}
					if hasWrite {
						stmt.Exec(roleName, "ENTITY", entityQN, nil, "WRITE", xpath, moduleName, projectID, snapshotID)
						count++
					}
					if rule.AllowDelete() {
						stmt.Exec(roleName, "ENTITY", entityQN, nil, "DELETE", xpath, moduleName, projectID, snapshotID)
						count++
					}
					count += b.emitMemberPermissionsGen(stmt, rule, ent, roleName, entityQN, xpath, moduleName, projectID, snapshotID)
				}
			}
		}
	}

	return count
}

// entityAccessFromMemberRights — legacy sdk-typed helper. Kept until
// builder_permissions_test.go migrates to fixtures using gen types
// (Stage 3.3.4 C6 territory).
func entityAccessFromMemberRights(rule *domainmodel.AccessRule) (hasRead, hasWrite bool) {
	if len(rule.MemberAccesses) > 0 {
		for _, ma := range rule.MemberAccesses {
			if ma.AccessRights == domainmodel.MemberAccessRightsReadOnly || ma.AccessRights == domainmodel.MemberAccessRightsReadWrite {
				hasRead = true
			}
			if ma.AccessRights == domainmodel.MemberAccessRightsReadWrite {
				hasWrite = true
			}
		}
	} else {
		dmr := rule.DefaultMemberAccessRights
		if dmr == domainmodel.MemberAccessRightsReadOnly || dmr == domainmodel.MemberAccessRightsReadWrite {
			hasRead = true
		}
		if dmr == domainmodel.MemberAccessRightsReadWrite {
			hasWrite = true
		}
	}
	return
}

// entityAccessFromMemberRightsGen mirrors the legacy helper but reads
// gen accessors (DefaultMemberAccessRights() string, MemberAccess.AccessRights()
// string).
func entityAccessFromMemberRightsGen(rule *genDm.AccessRule) (hasRead, hasWrite bool) {
	mems := rule.MemberAccessesItems()
	if len(mems) > 0 {
		for _, m := range mems {
			ma, ok := m.(*genDm.MemberAccess)
			if !ok {
				continue
			}
			switch ma.AccessRights() {
			case "ReadWrite":
				hasRead = true
				hasWrite = true
			case "ReadOnly":
				hasRead = true
			}
		}
	} else {
		switch rule.DefaultMemberAccessRights() {
		case "ReadWrite":
			hasRead = true
			hasWrite = true
		case "ReadOnly":
			hasRead = true
		}
	}
	return
}

// emitMemberPermissionsGen mirrors emitMemberPermissions for the gen
// AccessRule / Entity (Stage 3.3.4 C2.b). The MemberAccess element
// references attributes / associations by qualified name in gen
// (AttributeQualifiedName / AssociationQualifiedName); we use the
// trailing segment as the member name.
func (b *Builder) emitMemberPermissionsGen(stmt *sql.Stmt, rule *genDm.AccessRule, ent *genDm.Entity,
	roleName, entityQN, xpath, moduleName, projectID, snapshotID string) int {
	count := 0

	mems := rule.MemberAccessesItems()
	if len(mems) > 0 {
		for _, m := range mems {
			ma, ok := m.(*genDm.MemberAccess)
			if !ok {
				continue
			}
			memberName := simpleNameFromQN(ma.AttributeQualifiedName())
			if memberName == "" {
				memberName = simpleNameFromQN(ma.AssociationQualifiedName())
			}
			if memberName == "" {
				continue
			}
			switch ma.AccessRights() {
			case "ReadWrite":
				stmt.Exec(roleName, "ENTITY", entityQN, memberName, "MEMBER_READ", xpath, moduleName, projectID, snapshotID)
				stmt.Exec(roleName, "ENTITY", entityQN, memberName, "MEMBER_WRITE", xpath, moduleName, projectID, snapshotID)
				count += 2
			case "ReadOnly":
				stmt.Exec(roleName, "ENTITY", entityQN, memberName, "MEMBER_READ", xpath, moduleName, projectID, snapshotID)
				count++
			}
		}
	} else {
		def := rule.DefaultMemberAccessRights()
		if def == "" || def == "None" {
			return 0
		}
		for _, a := range ent.AttributesItems() {
			attr, ok := a.(*genDm.Attribute)
			if !ok {
				continue
			}
			switch def {
			case "ReadWrite":
				stmt.Exec(roleName, "ENTITY", entityQN, attr.Name(), "MEMBER_READ", xpath, moduleName, projectID, snapshotID)
				stmt.Exec(roleName, "ENTITY", entityQN, attr.Name(), "MEMBER_WRITE", xpath, moduleName, projectID, snapshotID)
				count += 2
			case "ReadOnly":
				stmt.Exec(roleName, "ENTITY", entityQN, attr.Name(), "MEMBER_READ", xpath, moduleName, projectID, snapshotID)
				count++
			}
		}
	}

	return count
}

// simpleNameFromQN extracts the trailing segment from a dotted QN
// (e.g. "Module.Entity.Attr" → "Attr"). Empty input returns empty.
func simpleNameFromQN(qn string) string {
	if qn == "" {
		return ""
	}
	if i := strings.LastIndex(qn, "."); i >= 0 && i < len(qn)-1 {
		return qn[i+1:]
	}
	return qn
}

// buildMicroflowPermissions extracts microflow execution permissions.
func (b *Builder) buildMicroflowPermissions(stmt *sql.Stmt, projectID, snapshotID string) int {
	count := 0

	mfs, err := b.cachedMicroflows()
	if err != nil {
		return 0
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		roles := mf.AllowedModuleRolesQualifiedNames()
		if len(roles) == 0 {
			continue
		}

		moduleID := b.hierarchy.findModuleID(model.ID(mf.ID()))
		moduleName := b.hierarchy.getModuleName(moduleID)
		mfQN := moduleName + "." + mf.Name()

		for _, roleName := range roles {
			stmt.Exec(roleName, "MICROFLOW", mfQN, nil, "EXECUTE", nil, moduleName, projectID, snapshotID)
			count++
		}
	}

	return count
}

// buildPagePermissions extracts page view permissions.
func (b *Builder) buildPagePermissions(stmt *sql.Stmt, projectID, snapshotID string) int {
	count := 0

	pages, err := b.reader.ListPages()
	if err != nil {
		return 0
	}

	for _, pg := range pages {
		if len(pg.AllowedRoles) == 0 {
			continue
		}

		moduleID := b.hierarchy.findModuleID(pg.ContainerID)
		moduleName := b.hierarchy.getModuleName(moduleID)
		pgQN := moduleName + "." + pg.Name

		for _, roleID := range pg.AllowedRoles {
			roleName := string(roleID)
			stmt.Exec(roleName, "PAGE", pgQN, nil, "VIEW", nil, moduleName, projectID, snapshotID)
			count++
		}
	}

	return count
}

// buildODataServicePermissions extracts published OData service access permissions.
func (b *Builder) buildODataServicePermissions(stmt *sql.Stmt, projectID, snapshotID string) int {
	count := 0

	services, err := b.reader.ListPublishedODataServices()
	if err != nil {
		return 0
	}

	for _, svc := range services {
		if len(svc.AllowedModuleRoles) == 0 {
			continue
		}

		moduleID := b.hierarchy.findModuleID(svc.ContainerID)
		moduleName := b.hierarchy.getModuleName(moduleID)
		svcQN := moduleName + "." + svc.Name

		for _, roleName := range svc.AllowedModuleRoles {
			stmt.Exec(roleName, "ODATA_SERVICE", svcQN, nil, "ACCESS", nil, moduleName, projectID, snapshotID)
			count++
		}
	}

	return count
}
