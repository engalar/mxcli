// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// MicroflowBackend provides microflow and nanoflow operations.
//
// After Followups E (Stage 3.2 closeout) and F (catalog + linter
// migration), the executor / catalog / linter all consume
// modelsdk-native gen objects everywhere. The remaining surface here
// is intentionally small:
//
//   - DeleteMicroflow / DeleteNanoflow — fallback path used by
//     repo_extract.go's deleteMicroflowViaRepoOrBackend when
//     ctx.Microflows is nil (mock-only test contexts).
//   - IsRule — fallback for isRuleViaRepoOrBackend, same rationale.
//   - ListMicroflowsGen / ListNanoflowsGen / GetMicroflowGen — the
//     forward-looking gen-typed surface; catalog + linter consumers
//     route through these.
//
// Followup F3 retired the sdk-typed ListMicroflows / GetMicroflow /
// ListNanoflows methods. All other CRUD methods (Get / Create /
// Update / Move / Parse) were retired earlier in Followup E6.
// Production routes through ctx.Microflows / ctx.Nanoflows
// (modelsdk repos) directly.
type MicroflowBackend interface {
	// DeleteMicroflow removes a microflow by ID. Retained as the
	// fallback for deleteMicroflowViaRepoOrBackend in mock-only test
	// contexts that do not wire ctx.Microflows.
	DeleteMicroflow(id model.ID) error

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
