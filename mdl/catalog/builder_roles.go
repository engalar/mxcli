// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"strings"

	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// buildRoleMappings populates the role_mappings table from project security.
// This maps user roles to their assigned module roles.
func (b *Builder) buildRoleMappings() error {
	ps, err := b.reader.GetProjectSecurityGen()
	if err != nil {
		// No project security — skip silently
		return nil
	}
	if ps == nil {
		return nil
	}

	stmt, err := b.tx.Prepare(`
		INSERT INTO role_mappings (UserRoleName, ModuleRoleName, ModuleName, ProjectId, SnapshotId)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	projectID := b.catalog.projectID
	snapshotID := b.snapshot.ID
	count := 0

	for _, item := range ps.UserRolesItems() {
		ur, ok := item.(*genSec.UserRole)
		if !ok {
			continue
		}
		for _, mr := range ur.ModuleRolesQualifiedNames() {
			// Module role is a qualified name like "MyModule.Admin"
			moduleName := ""
			if parts := strings.SplitN(mr, ".", 2); len(parts) == 2 {
				moduleName = parts[0]
			}
			stmt.Exec(ur.Name(), mr, moduleName, projectID, snapshotID)
			count++
		}
	}

	b.report("RoleMappings", count)
	return nil
}
