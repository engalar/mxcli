// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/catalog"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"

	_ "modernc.org/sqlite"
)

// mpr015Reader implements linter.LintReader for MPR015 tests.
// Embeds the interface so that unused methods panic rather than silently return zero values.
type mpr015Reader struct {
	linter.LintReader // nil-panic default for methods we don't configure
	mfs               map[model.ID]*genMf.Microflow
}

func (r *mpr015Reader) GetMicroflowGen(id model.ID) (*genMf.Microflow, error) {
	return r.mfs[id], nil
}

// makeMFWithParam builds a Microflow that has one parameter named paramName.
// If targetQN and paramRef are non-empty, it also adds a MicroflowCallAction
// that calls targetQN with a single ParameterMapping referencing paramRef.
func makeMFWithParam(mfName, paramName, targetQN, paramRef string) *genMf.Microflow {
	mf := genMf.NewMicroflow()
	mf.SetName(mfName)

	oc := genMf.NewMicroflowObjectCollection()

	// Add a parameter
	param := genMf.NewMicroflowParameter()
	param.SetName(paramName)
	oc.AddObjects(param)

	// Optionally add a call action with a parameter mapping
	if targetQN != "" {
		callAction := genMf.NewMicroflowCallAction()

		call := genMf.NewMicroflowCall()
		call.SetMicroflowQualifiedName(targetQN)

		if paramRef != "" {
			mapping := genMf.NewMicroflowCallParameterMapping()
			mapping.SetParameterQualifiedName(paramRef)
			mapping.SetArgument("$x")
			call.AddParameterMappings(mapping)
		}

		callAction.SetMicroflowCall(call)
		oc.AddObjects(callAction)
	}

	mf.SetObjectCollection(oc)
	return mf
}

// setupMPR015DB creates an in-memory DB with microflows + modules rows.
func setupMPR015DB(t *testing.T, rows []struct{ id, qn, mod string }) catalog.CatalogDB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE microflows (
		Id TEXT, Name TEXT, QualifiedName TEXT, ModuleName TEXT,
		Folder TEXT, MicroflowType TEXT, Description TEXT, ReturnType TEXT,
		ParameterCount INTEGER, ActivityCount INTEGER, Complexity INTEGER
	)`); err != nil {
		t.Fatalf("create microflows table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE modules (Name TEXT PRIMARY KEY, Source TEXT)`); err != nil {
		t.Fatalf("create modules table: %v", err)
	}
	for _, r := range rows {
		parts := strings.SplitN(r.qn, ".", 2)
		name := r.qn
		if len(parts) == 2 {
			name = parts[1]
		}
		if _, err := db.Exec(
			`INSERT INTO microflows VALUES (?, ?, ?, ?, '', 'Microflow', '', 'Void', 0, 1, 1)`,
			r.id, name, r.qn, r.mod,
		); err != nil {
			t.Fatalf("insert mf row: %v", err)
		}
		if _, err := db.Exec(`INSERT OR IGNORE INTO modules (Name, Source) VALUES (?, '')`, r.mod); err != nil {
			t.Fatalf("insert module row: %v", err)
		}
	}
	return catalog.WrapSqlDB(db)
}

// makeID creates a deterministic element.ID from a string (good enough for tests).
func makeID(s string) element.ID { return element.ID(s) }

// TestMPR015_FlagsBrokenParameterRef verifies that a caller referencing a
// non-existent parameter ("OldParam") of a target microflow is flagged.
func TestMPR015_FlagsBrokenParameterRef(t *testing.T) {
	const targetID = "target-001"
	const callerID = "caller-001"

	// Target has param "MessageId"; caller erroneously references "OldParam".
	targetMF := makeMFWithParam("GET_Message_ById", "MessageId", "", "")
	targetMF.SetID(makeID(targetID))

	callerMF := makeMFWithParam(
		"Caller_MF", "SomeParam",
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.OldParam", // broken — param doesn't exist
	)
	callerMF.SetID(makeID(callerID))

	reader := &mpr015Reader{mfs: map[model.ID]*genMf.Microflow{
		model.ID(targetID): targetMF,
		model.ID(callerID): callerMF,
	}}
	db := setupMPR015DB(t, []struct{ id, qn, mod string }{
		{targetID, "Common_Utils.GET_Message_ById", "Common_Utils"},
		{callerID, "ContractorReg.Caller_MF", "ContractorReg"},
	})
	defer db.Close()

	ctx := linter.NewLintContextFromDBAndReader(db, reader)
	violations := NewBrokenMFParamRefRule().Check(ctx)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}
	v := violations[0]
	if v.RuleID != "MPR015" {
		t.Errorf("RuleID = %q, want MPR015", v.RuleID)
	}
	if !strings.Contains(v.Message, "OldParam") {
		t.Errorf("message should mention OldParam: %q", v.Message)
	}
	if v.Location.Module != "ContractorReg" {
		t.Errorf("Location.Module = %q, want ContractorReg", v.Location.Module)
	}
	if v.Location.DocumentName != "Caller_MF" {
		t.Errorf("Location.DocumentName = %q, want Caller_MF", v.Location.DocumentName)
	}
}

// TestMPR015_NoViolationWhenParamExists verifies no violation when the
// referenced parameter actually exists.
func TestMPR015_NoViolationWhenParamExists(t *testing.T) {
	const targetID = "target-002"
	const callerID = "caller-002"

	targetMF := makeMFWithParam("GET_Message_ById", "MessageId", "", "")
	targetMF.SetID(makeID(targetID))

	callerMF := makeMFWithParam(
		"Caller_OK", "SomeParam",
		"Common_Utils.GET_Message_ById",
		"Common_Utils.GET_Message_ById.MessageId", // valid
	)
	callerMF.SetID(makeID(callerID))

	reader := &mpr015Reader{mfs: map[model.ID]*genMf.Microflow{
		model.ID(targetID): targetMF,
		model.ID(callerID): callerMF,
	}}
	db := setupMPR015DB(t, []struct{ id, qn, mod string }{
		{targetID, "Common_Utils.GET_Message_ById", "Common_Utils"},
		{callerID, "Common_Utils.Caller_OK", "Common_Utils"},
	})
	defer db.Close()

	ctx := linter.NewLintContextFromDBAndReader(db, reader)
	violations := NewBrokenMFParamRefRule().Check(ctx)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d: %v", len(violations), violations)
	}
}

// TestMPR015_UnknownTargetSkipped verifies that calls to microflows not in the
// catalog (e.g. marketplace modules) are silently skipped.
func TestMPR015_UnknownTargetSkipped(t *testing.T) {
	const callerID = "caller-003"

	callerMF := makeMFWithParam(
		"Caller_Ext", "SomeParam",
		"External.SomeMF",
		"External.SomeMF.Gone",
	)
	callerMF.SetID(makeID(callerID))

	reader := &mpr015Reader{mfs: map[model.ID]*genMf.Microflow{
		model.ID(callerID): callerMF,
	}}
	// Only the caller is in the catalog; External.SomeMF is absent.
	db := setupMPR015DB(t, []struct{ id, qn, mod string }{
		{callerID, "MyModule.Caller_Ext", "MyModule"},
	})
	defer db.Close()

	ctx := linter.NewLintContextFromDBAndReader(db, reader)
	violations := NewBrokenMFParamRefRule().Check(ctx)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations for unknown target, got %d: %v", len(violations), violations)
	}
}

// TestMPR015_BrokenRefInsideLoop verifies detection inside a LoopedActivity body.
func TestMPR015_BrokenRefInsideLoop(t *testing.T) {
	const targetID = "target-004"
	const callerID = "caller-004"

	targetMF := makeMFWithParam("SUB_Process", "ItemId", "", "")
	targetMF.SetID(makeID(targetID))

	// Build caller with broken call inside a loop body
	callerMF := genMf.NewMicroflow()
	callerMF.SetName("Caller_Loop")
	callerMF.SetID(makeID(callerID))

	outerOC := genMf.NewMicroflowObjectCollection()
	loopBody := genMf.NewMicroflowObjectCollection()

	callAction := genMf.NewMicroflowCallAction()
	call := genMf.NewMicroflowCall()
	call.SetMicroflowQualifiedName("Common.SUB_Process")
	mapping := genMf.NewMicroflowCallParameterMapping()
	mapping.SetParameterQualifiedName("Common.SUB_Process.DeadParam") // broken
	mapping.SetArgument("$item")
	call.AddParameterMappings(mapping)
	callAction.SetMicroflowCall(call)
	loopBody.AddObjects(callAction)

	loop := genMf.NewLoopedActivity()
	loop.SetObjectCollection(loopBody)
	outerOC.AddObjects(loop)
	callerMF.SetObjectCollection(outerOC)

	reader := &mpr015Reader{mfs: map[model.ID]*genMf.Microflow{
		model.ID(targetID): targetMF,
		model.ID(callerID): callerMF,
	}}
	db := setupMPR015DB(t, []struct{ id, qn, mod string }{
		{targetID, "Common.SUB_Process", "Common"},
		{callerID, "MyModule.Caller_Loop", "MyModule"},
	})
	defer db.Close()

	ctx := linter.NewLintContextFromDBAndReader(db, reader)
	violations := NewBrokenMFParamRefRule().Check(ctx)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation inside loop, got %d: %v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "DeadParam") {
		t.Errorf("message should mention DeadParam: %q", violations[0].Message)
	}
}

// TestMPR015_ReaderNil_ReturnsEmpty verifies graceful no-op when reader is absent.
func TestMPR015_ReaderNil_ReturnsEmpty(t *testing.T) {
	ctx := linter.NewLintContextFromDB(nil)
	violations := NewBrokenMFParamRefRule().Check(ctx)
	if len(violations) != 0 {
		t.Fatalf("expected 0 violations without reader, got %d", len(violations))
	}
}

// TestMPR015_Metadata verifies rule metadata fields.
func TestMPR015_Metadata(t *testing.T) {
	r := NewBrokenMFParamRefRule()
	if r.ID() != "MPR015" {
		t.Errorf("ID = %q, want MPR015", r.ID())
	}
	if r.Category() != "MPR" {
		t.Errorf("Category = %q, want MPR", r.Category())
	}
	if r.DefaultSeverity() != linter.SeverityError {
		t.Errorf("DefaultSeverity = %v, want Error", r.DefaultSeverity())
	}
}
