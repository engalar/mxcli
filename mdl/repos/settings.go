// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genSet "github.com/mendixlabs/mxcli/modelsdk/gen/settings"
)

// Settings covers two scopes — project-level and module-level — that the
// legacy SettingsBackend / ModuleSettingsBackend split into separate
// interfaces. Stage 2 mirrors that split.
//
// TODO Stage 3 cutover: flesh out signatures from the legacy interfaces
// and produce MPR implementations.

type ProjectSettingsReader interface {
	Get() (*genSet.ProjectSettings, error)
}

type ProjectSettingsWriter interface {
	Update(s *genSet.ProjectSettings) error
}

type ProjectSettingsRepository interface {
	ProjectSettingsReader
	ProjectSettingsWriter
}

// ModuleSettings: gen/settings has no exported ModuleSettings root type;
// each module owns small per-feature settings elements (e.g. nav,
// security). Stage 2 keys these by element.Element until Stage 3
// inventories the actual unit types.
type ModuleSettingsReader interface {
	Get(moduleID model.ID) (element.Element, error)
}

type ModuleSettingsWriter interface {
	Update(moduleID model.ID, s element.Element) error
}

type ModuleSettingsRepository interface {
	ModuleSettingsReader
	ModuleSettingsWriter
}
