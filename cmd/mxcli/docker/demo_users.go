// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"fmt"
	"io"
	"os"

	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/model"
	gensecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// EnsureDemoUsers checks whether the project has demo users configured.
// If none exist, it enables demo users and creates a default admin user
// so the application is accessible after startup (required for --security demo).
func EnsureDemoUsers(projectPath string, w io.Writer) error {
	fmt.Fprintln(w, "Checking demo users...")

	be, err := mprbackend.NewFromPath(projectPath)
	if err != nil {
		return fmt.Errorf("opening project: %w", err)
	}
	defer be.Disconnect()

	ps, err := be.GetProjectSecurityGen()
	if err != nil {
		return fmt.Errorf("reading project security: %w", err)
	}

	if len(ps.DemoUsersItems()) > 0 {
		fmt.Fprintf(w, "  Found %d demo user(s), skipping.\n", len(ps.DemoUsersItems()))
		return nil
	}

	fmt.Fprintln(w, "  No demo users found, creating default admin...")

	if !ps.EnableDemoUsers() {
		if err := be.SetProjectDemoUsersEnabled(model.ID(ps.ID()), true); err != nil {
			return fmt.Errorf("enabling demo users: %w", err)
		}
		fmt.Fprintln(w, "  Enabled demo users.")
	}

	roleName := "Administrator"
	if urItems := ps.UserRolesItems(); len(urItems) > 0 {
		if ur, ok := urItems[0].(*gensecurity.UserRole); ok {
			roleName = ur.Name()
		}
		for _, item := range urItems {
			ur, ok := item.(*gensecurity.UserRole)
			if !ok {
				continue
			}
			if ur.Name() == "Administrator" || ur.Name() == "Admin" {
				roleName = ur.Name()
				break
			}
		}
	}

	if err := be.AddDemoUser(model.ID(ps.ID()), "admin", "Admin123!", "", []string{roleName}); err != nil {
		return fmt.Errorf("creating demo user: %w", err)
	}

	fmt.Fprintf(w, "  Created demo user: admin / Admin123! (role: %s)\n", roleName)
	return nil
}

// EnsureDemoUsersIfNeeded calls EnsureDemoUsers only when the output writer
// is a real file (not io.Discard) and the project path exists.
func EnsureDemoUsersIfNeeded(projectPath string, w io.Writer) {
	if projectPath == "" {
		return
	}
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return
	}
	if err := EnsureDemoUsers(projectPath, w); err != nil {
		fmt.Fprintf(w, "  Warning: could not ensure demo users: %v\n", err)
	}
}
