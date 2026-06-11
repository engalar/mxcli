// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	msdksecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// msdkWrite reads a unit, applies mutateFn, re-encodes, and writes back.
// It avoids sdk/mpr's updateTransactionID() which fails on hard-linked MPR files
// (SQLITE_READONLY_DBMOVED 1544).
func (b *MprBackend) msdkWrite(unitID model.ID, mutateFn func(elem element.Element) error) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return fmt.Errorf("read unit: %w", err)
	}
	elem, err := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(rawBytes))
	if err != nil {
		return fmt.Errorf("decode unit: %w", err)
	}
	if err := mutateFn(elem); err != nil {
		return err
	}
	newBytes, err := (&codec.Encoder{}).Encode(elem)
	if err != nil {
		return fmt.Errorf("encode unit: %w", err)
	}
	// writeUnitContents routes through scriptBuf (EXECUTE SCRIPT) or unitBuf
	// (import session) when either is active, keeping the ScriptOverlay / import
	// overlay current for subsequent reads in the same block and ensuring all
	// mutations are included in the atomic BatchWrite at Commit time.
	//
	// Previously this used UpdateRawUnit (which only checks the import-mode
	// sessionBuf, not scriptBuf). During EXECUTE SCRIPT that caused two bugs:
	//   1. ScriptOverlay was never updated → each subsequent msdkWrite read
	//      the same stale pre-mutation bytes, so consecutive GRANTs overwrote
	//      each other.
	//   2. commitScriptBuffer BatchWrite overwrote every UpdateRawUnit write
	//      with the pre-mutation domain model stored in the ScriptBuffer,
	//      wiping all security grants applied inside the script block.
	if err := b.writeUnitContents(unitID, newBytes); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}
	return nil
}

func (b *MprBackend) setProjectDemoUsersEnabledViaModelsdk(unitID model.ID, enabled bool) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ps, ok := elem.(*msdksecurity.ProjectSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ProjectSecurity)", elem)
		}
		ps.SetEnableDemoUsers(enabled)
		return nil
	})
}

func (b *MprBackend) addUserRoleViaModelsdk(unitID model.ID, name string, moduleRoles []string, manageAllRoles bool) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ps, ok := elem.(*msdksecurity.ProjectSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ProjectSecurity)", elem)
		}
		ur := msdksecurity.NewUserRole()
		id := element.ID(modelsdkmpr.GenerateID())
		ur.SetID(id)
		// GUID must be a stable binary UUID so mxbuild emits a consistent
		// role UUID in metadata.json across builds. Without it mxbuild
		// generates a random UUID each build, causing the runtime's
		// System.UserRole table to diverge from the model on each rebuild
		// and silently preventing demo-user role assignment at startup.
		ur.SetGuid(string(id))
		ur.SetName(name)
		ur.SetManageAllRoles(manageAllRoles)
		ur.SetManageUsersWithoutRoles(false)
		for _, mr := range moduleRoles {
			ur.AddModuleRoles(mr)
		}
		ps.AddUserRoles(ur)
		return nil
	})
}

func (b *MprBackend) removeUserRoleViaModelsdk(unitID model.ID, name string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ps, ok := elem.(*msdksecurity.ProjectSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ProjectSecurity)", elem)
		}
		for i, ur := range ps.UserRolesItems() {
			typed, ok := ur.(*msdksecurity.UserRole)
			if ok && typed.Name() == name {
				ps.RemoveUserRoles(i)
				return nil
			}
		}
		return fmt.Errorf("user role %q not found", name)
	})
}

func (b *MprBackend) alterUserRoleModuleRolesViaModelsdk(unitID model.ID, userRoleName string, add bool, moduleRoles []string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ps, ok := elem.(*msdksecurity.ProjectSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ProjectSecurity)", elem)
		}
		for _, ur := range ps.UserRolesItems() {
			typed, ok := ur.(*msdksecurity.UserRole)
			if !ok || typed.Name() != userRoleName {
				continue
			}
			// Backfill missing GUID: roles created by old mxcli lack a binary
			// "GUID" field, causing mxbuild to assign a random UUID each build.
			// On any alter operation, write a stable GUID (= $ID) if absent.
			if typed.Guid() == "" {
				typed.SetGuid(string(typed.ID()))
			}
			if add {
				for _, mr := range moduleRoles {
					typed.AddModuleRoles(mr)
				}
			} else {
				existing := typed.ModuleRolesQualifiedNames()
				remove := make(map[string]bool, len(moduleRoles))
				for _, mr := range moduleRoles {
					remove[mr] = true
				}
				filtered := make([]string, 0, len(existing))
				for _, mr := range existing {
					if !remove[mr] {
						filtered = append(filtered, mr)
					}
				}
				typed.SetModuleRolesQualifiedNames(filtered)
			}
			return nil
		}
		return fmt.Errorf("user role %q not found", userRoleName)
	})
}

func (b *MprBackend) removeModuleRoleFromAllUserRolesViaModelsdk(unitID model.ID, qualifiedRole string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ps, ok := elem.(*msdksecurity.ProjectSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ProjectSecurity)", elem)
		}
		for _, ur := range ps.UserRolesItems() {
			typed, ok := ur.(*msdksecurity.UserRole)
			if !ok {
				continue
			}
			existing := typed.ModuleRolesQualifiedNames()
			filtered := make([]string, 0, len(existing))
			for _, mr := range existing {
				if mr != qualifiedRole {
					filtered = append(filtered, mr)
				}
			}
			if len(filtered) != len(existing) {
				typed.SetModuleRolesQualifiedNames(filtered)
			}
		}
		return nil
	})
}

func (b *MprBackend) addDemoUserViaModelsdk(unitID model.ID, userName, password, entity string, userRoles []string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ps, ok := elem.(*msdksecurity.ProjectSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ProjectSecurity)", elem)
		}
		du := msdksecurity.NewDemoUser()
		du.SetID(element.ID(modelsdkmpr.GenerateID()))
		du.SetUserName(userName)
		du.SetPassword(password)
		du.SetEntityQualifiedName(entity)
		du.SetUserRolesQualifiedNames(userRoles)
		ps.AddDemoUsers(du)
		return nil
	})
}

func (b *MprBackend) removeDemoUserViaModelsdk(unitID model.ID, userName string) error {
	return b.msdkWrite(unitID, func(elem element.Element) error {
		ps, ok := elem.(*msdksecurity.ProjectSecurity)
		if !ok {
			return fmt.Errorf("unexpected type %T (want *ProjectSecurity)", elem)
		}
		for i, du := range ps.DemoUsersItems() {
			typed, ok := du.(*msdksecurity.DemoUser)
			if ok && typed.UserName() == userName {
				ps.RemoveDemoUsers(i)
				return nil
			}
		}
		return fmt.Errorf("demo user %q not found", userName)
	})
}
