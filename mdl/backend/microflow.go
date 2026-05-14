// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/sdk/microflows"
)

// MicroflowBackend provides microflow and nanoflow operations.
//
// The interface currently exposes two parallel surfaces while the
// modelsdk-native (gen-typed) migration is in flight:
//
//   - sdk-typed methods (ListMicroflows / GetMicroflow / ...) — the legacy
//     surface, still consumed by ~49 production call sites in mdl/executor.
//     New code should NOT call these; they are scheduled for removal
//     after Followup E migrates the production callers to ctx.Microflows.
//   - gen-typed methods (ListMicroflowsGen / ListNanoflowsGen) — the
//     forward-looking surface backed by the modelsdk repos. Tests and new
//     code should use these so the sdk/microflows import can be retired
//     from mdl/executor.
type MicroflowBackend interface {
	// Deprecated: use ListMicroflowsGen for new code. Retained for legacy
	// production callers (cmd_security, cmd_modules, cmd_diff, cmd_move,
	// cmd_rename, cmd_workflows_write, helpers, autocomplete, ...) until
	// Followup E migrates them to ctx.Microflows.
	ListMicroflows() ([]*microflows.Microflow, error)
	// Deprecated: use ctx.Microflows for new code. See ListMicroflows.
	GetMicroflow(id model.ID) (*microflows.Microflow, error)
	// Deprecated: use ctx.Microflows for new code. See ListMicroflows.
	CreateMicroflow(mf *microflows.Microflow) error
	// Deprecated: use ctx.Microflows for new code. See ListMicroflows.
	UpdateMicroflow(mf *microflows.Microflow) error
	DeleteMicroflow(id model.ID) error
	// Deprecated: use ctx.Microflows for new code. See ListMicroflows.
	MoveMicroflow(mf *microflows.Microflow) error

	// ParseMicroflowFromRaw builds a Microflow from an already-unmarshalled
	// map. Used by diff-local and other callers that have raw map data.
	ParseMicroflowFromRaw(raw map[string]any, unitID, containerID model.ID) *microflows.Microflow

	// ParseMicroflowBSON parses raw microflow BSON bytes into a Microflow.
	// Used by the executor to inspect microflows it has not necessarily
	// loaded via ListMicroflows (e.g. to resolve a CALL MICROFLOW's return
	// type from its raw unit).
	ParseMicroflowBSON(contents []byte, unitID, containerID model.ID) (*microflows.Microflow, error)

	// Deprecated: use ListNanoflowsGen for new code. See ListMicroflows.
	ListNanoflows() ([]*microflows.Nanoflow, error)
	// Deprecated: use ctx.Nanoflows for new code. See ListMicroflows.
	GetNanoflow(id model.ID) (*microflows.Nanoflow, error)
	// Deprecated: use ctx.Nanoflows for new code. See ListMicroflows.
	CreateNanoflow(nf *microflows.Nanoflow) error
	// Deprecated: use ctx.Nanoflows for new code. See ListMicroflows.
	UpdateNanoflow(nf *microflows.Nanoflow) error
	DeleteNanoflow(id model.ID) error
	// Deprecated: use ctx.Nanoflows for new code. See ListMicroflows.
	MoveNanoflow(nf *microflows.Nanoflow) error

	// IsRule reports whether the given qualified name refers to a rule
	// (Microflows$Rule) rather than a microflow. The flow builder uses this
	// to decide whether an IF condition that looks like a function call
	// (Module.Name(...)) should be serialized as a RuleSplitCondition.
	IsRule(qualifiedName string) (bool, error)

	// ListMicroflowsGen returns every microflow in the project as
	// modelsdk-native gen objects. New code (and migrated tests) should
	// prefer this over the deprecated sdk-typed ListMicroflows.
	//
	// Note: the returned *genMf.Microflow values do NOT carry container
	// identity (codec roundtrip drops Container linkage). Callers that
	// need to resolve owning module/folder must use a separate lookup
	// such as ctx.Microflows.GetContainerUUID.
	ListMicroflowsGen() ([]*genMf.Microflow, error)

	// ListNanoflowsGen returns every nanoflow in the project as
	// modelsdk-native gen objects. See ListMicroflowsGen for caveats.
	ListNanoflowsGen() ([]*genMf.Nanoflow, error)
}
