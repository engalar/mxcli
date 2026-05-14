// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/sdk/microflows"
)

// MicroflowBackend provides microflow and nanoflow operations.
//
// After Followup E (Stage 3.2 closeout) the executor consumes
// modelsdk-native gen objects everywhere. The remaining surface here
// is intentionally small:
//
//   - ListMicroflows / ListNanoflows — kept as the sdk-typed reads
//     because the catalog builder (mdl/catalog) still consumes
//     *microflows.Microflow / *microflows.Nanoflow through the
//     CatalogReader structural interface. After Followup F1 the
//     catalog also walks gen objects, but these methods are kept on
//     FullBackend until Followup F3 retires them so external
//     CatalogReader implementations have a deprecation window.
//   - DeleteMicroflow / DeleteNanoflow — fallback path used by
//     repo_extract.go's deleteMicroflowViaRepoOrBackend when
//     ctx.Microflows is nil (mock-only test contexts).
//   - IsRule — fallback for isRuleViaRepoOrBackend, same rationale.
//   - ListMicroflowsGen / ListNanoflowsGen / GetMicroflowGen — the
//     forward-looking gen-typed surface; production catalog +
//     linter consumers route through these.
//
// All other CRUD methods (Create / Update / Move / Parse) have
// been retired — production routes through ctx.Microflows /
// ctx.Nanoflows (modelsdk repos) directly.
type MicroflowBackend interface {
	// ListMicroflows returns every microflow as the legacy sdk type.
	// Deprecated by Followup F1 — the catalog now walks gen-typed
	// objects via ListMicroflowsGen. Retained until Followup F3
	// retires the sdk-typed surface.
	ListMicroflows() ([]*microflows.Microflow, error)

	// GetMicroflow fetches a single microflow by ID as the legacy sdk
	// type. Deprecated by Followup F2 — the linter now fetches
	// gen-typed bodies via GetMicroflowGen. Retained until Followup F3
	// retires the sdk-typed surface.
	GetMicroflow(id model.ID) (*microflows.Microflow, error)

	// DeleteMicroflow removes a microflow by ID. Retained as the
	// fallback for deleteMicroflowViaRepoOrBackend in mock-only test
	// contexts that do not wire ctx.Microflows.
	DeleteMicroflow(id model.ID) error

	// ListNanoflows mirrors ListMicroflows. Same catalog-only retention.
	// Deprecated by Followup F1.
	ListNanoflows() ([]*microflows.Nanoflow, error)

	// DeleteNanoflow mirrors DeleteMicroflow. Same fallback retention.
	DeleteNanoflow(id model.ID) error

	// IsRule reports whether the given qualified name refers to a rule
	// (Microflows$Rule) rather than a microflow. The flow builder uses
	// this to decide whether an IF condition that looks like a function
	// call (Module.Name(...)) should be serialized as a
	// RuleSplitCondition. Retained as the fallback for
	// isRuleViaRepoOrBackend.
	IsRule(qualifiedName string) (bool, error)

	// ListMicroflowsGen returns every microflow in the project as
	// modelsdk-native gen objects. Production catalog/linter consumers
	// route through this. Used as the fallback inside listMicroflowsGen
	// for mock-only test contexts that have not wired ctx.Microflows.
	//
	// Note: the returned *genMf.Microflow values do NOT carry container
	// identity (codec roundtrip drops Container linkage). Callers that
	// need to resolve owning module/folder must use a separate lookup
	// such as ctx.Microflows.GetContainerUUID, or the cached
	// listMicroflowsWithContainerGen helper.
	ListMicroflowsGen() ([]*genMf.Microflow, error)

	// ListNanoflowsGen returns every nanoflow in the project as
	// modelsdk-native gen objects. See ListMicroflowsGen for caveats.
	ListNanoflowsGen() ([]*genMf.Nanoflow, error)

	// GetMicroflowGen fetches a single microflow body by ID as a
	// modelsdk-native gen object. Used by lint rules that walk full
	// activity trees (error handling, loop-commit, split-caption,
	// etc) and by any catalog consumer that needs a single flow.
	//
	// Implementations should return (nil, nil) for unknown IDs (not an
	// error) so callers can skip silently.
	GetMicroflowGen(id model.ID) (*genMf.Microflow, error)
}
