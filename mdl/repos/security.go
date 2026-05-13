// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// Security is project-scoped: ProjectSecurity is a singleton root, so
// Reader.Get takes no ID. ModuleRole / ModuleSecurity / DemoUser are
// addressed by ID through the ModuleRole / DemoUser methods.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interface
// (GRANT/REVOKE module-role / user-role manipulation) and produce an
// MPR implementation.
type SecurityReader interface {
	Get() (*genSec.ProjectSecurity, error)
	GetModuleSecurity(moduleID model.ID) (*genSec.ModuleSecurity, error)
}

type SecurityWriter interface {
	Update(s *genSec.ProjectSecurity) error
	UpdateModuleSecurity(moduleID model.ID, s *genSec.ModuleSecurity) error
}

type SecurityRepository interface {
	SecurityReader
	SecurityWriter
}
