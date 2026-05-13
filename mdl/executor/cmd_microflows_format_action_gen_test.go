// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.a tests for the gen-typed Object Actions family.
//
// Two test surfaces:
//  1. Fixture-driven (whatever the testdata MPR happens to contain)
//     covers CreateObjectAction / ChangeObjectAction / DeleteAction
//     end-to-end through DescribeMicroflowGenToString.
//  2. Direct-construction unit tests build minimal gen objects in
//     memory to cover every action kind including ones the fixture
//     doesn't carry (CommitAction, RollbackAction, AggregateListAction).
//
// String comparison is exact — the formatter is intentionally 1:1 with
// the legacy `cmd_microflows_format_action.go` so the migrated output
// can be diffed against the SDK path during the cutover.

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ────────────────────────────────────────────────────────
// Direct-construction unit tests (one per formatter)
// ────────────────────────────────────────────────────────

func TestFormatActionGen_DeleteAction(t *testing.T) {
	a := newGenAction(t, "Microflows$DeleteAction").(*genMf.DeleteAction)
	a.SetDeleteVariableName("Account")

	got := formatActionGen(a)
	want := "delete $Account;"
	if got != want {
		t.Errorf("DeleteAction:\n got: %q\nwant: %q", got, want)
	}
}

func TestFormatActionGen_CommitAction(t *testing.T) {
	cases := []struct {
		name       string
		varName    string
		withEvents bool
		refresh    bool
		want       string
	}{
		{"bare", "Order", false, false, "commit $Order;"},
		{"with events", "Order", true, false, "commit $Order with events;"},
		{"refresh", "Order", false, true, "commit $Order refresh;"},
		{"both", "Order", true, true, "commit $Order with events refresh;"},
		{"empty var defaults to Object", "", false, false, "commit $Object;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newGenAction(t, "Microflows$CommitAction").(*genMf.CommitAction)
			a.SetCommitVariableName(tc.varName)
			a.SetWithEvents(tc.withEvents)
			a.SetRefreshInClient(tc.refresh)
			if got := formatActionGen(a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_RollbackAction(t *testing.T) {
	cases := []struct {
		name    string
		varName string
		refresh bool
		want    string
	}{
		{"bare", "Order", false, "rollback $Order;"},
		{"refresh", "Order", true, "rollback $Order refresh;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newGenAction(t, "Microflows$RollbackAction").(*genMf.RollbackAction)
			a.SetRollbackVariableName(tc.varName)
			a.SetRefreshInClient(tc.refresh)
			if got := formatActionGen(a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_CreateObjectAction(t *testing.T) {
	t.Run("bare entity", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
		a.SetEntityQualifiedName("Sales.Order")
		a.SetOutputVariableName("NewOrder")
		got := formatActionGen(a)
		want := "$NewOrder = create Sales.Order;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("default output var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
		a.SetEntityQualifiedName("Sales.Order")
		got := formatActionGen(a)
		want := "$NewObject = create Sales.Order;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with attribute and same-module association initializers", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
		a.SetEntityQualifiedName("Sales.Order")
		a.SetOutputVariableName("NewOrder")

		// Attribute member: only the leaf "Total" should be emitted.
		mAttr := newGenAction(t, "Microflows$MemberChange").(*genMf.MemberChange)
		mAttr.SetAttributeQualifiedName("Sales.Order.Total")
		mAttr.SetValue("0")
		a.AddItems(mAttr)

		// Same-module association: prefix is stripped.
		mAssoc := newGenAction(t, "Microflows$MemberChange").(*genMf.MemberChange)
		mAssoc.SetAssociationQualifiedName("Sales.Order_Customer")
		mAssoc.SetValue("$Customer")
		a.AddItems(mAssoc)

		// Cross-module association: full qualified name is preserved.
		mCross := newGenAction(t, "Microflows$MemberChange").(*genMf.MemberChange)
		mCross.SetAssociationQualifiedName("Logistics.Order_Carrier")
		mCross.SetValue("$Carrier")
		a.AddItems(mCross)

		got := formatActionGen(a)
		want := "$NewOrder = create Sales.Order (Total = 0, Order_Customer = $Customer, Logistics.Order_Carrier = $Carrier);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty entity qualified name falls back to 'Entity'", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
		a.SetOutputVariableName("X")
		got := formatActionGen(a)
		want := "$X = create Entity;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_ChangeObjectAction(t *testing.T) {
	t.Run("bare", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeObjectAction").(*genMf.ChangeObjectAction)
		a.SetChangeVariableName("Account")
		got := formatActionGen(a)
		want := "change $Account;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with members and refresh", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeObjectAction").(*genMf.ChangeObjectAction)
		a.SetChangeVariableName("Account")
		a.SetRefreshInClient(true)

		m := newGenAction(t, "Microflows$MemberChange").(*genMf.MemberChange)
		m.SetAttributeQualifiedName("Administration.Account.Password")
		m.SetValue("$AccountPasswordData/NewPassword")
		a.AddItems(m)

		got := formatActionGen(a)
		want := "change $Account (Password = $AccountPasswordData/NewPassword) refresh;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("association keeps qualified name (no prefix strip in change path)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeObjectAction").(*genMf.ChangeObjectAction)
		a.SetChangeVariableName("Order")

		m := newGenAction(t, "Microflows$MemberChange").(*genMf.MemberChange)
		m.SetAssociationQualifiedName("Sales.Order_Customer")
		m.SetValue("$Customer")
		a.AddItems(m)

		got := formatActionGen(a)
		want := "change $Order (Sales.Order_Customer = $Customer);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("default change variable", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeObjectAction").(*genMf.ChangeObjectAction)
		got := formatActionGen(a)
		want := "change $Object;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_AggregateListAction(t *testing.T) {
	t.Run("count (default function)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$AggregateAction").(*genMf.AggregateListAction)
		a.SetInputListVariableName("Orders")
		a.SetOutputVariableName("OrderCount")
		// Empty AggregateFunction defaults to Count.
		got := formatActionGen(a)
		want := "$OrderCount = count($Orders);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("attribute-based sum", func(t *testing.T) {
		a := newGenAction(t, "Microflows$AggregateAction").(*genMf.AggregateListAction)
		a.SetInputListVariableName("Orders")
		a.SetOutputVariableName("Total")
		a.SetAggregateFunction(genMf.AggregateFunctionEnumSum)
		a.SetAttributeQualifiedName("Sales.Order.Amount")
		got := formatActionGen(a)
		want := "$Total = sum($Orders.Amount);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("expression-based average", func(t *testing.T) {
		a := newGenAction(t, "Microflows$AggregateAction").(*genMf.AggregateListAction)
		a.SetInputListVariableName("Orders")
		a.SetOutputVariableName("Avg")
		a.SetAggregateFunction(genMf.AggregateFunctionEnumAverage)
		a.SetUseExpression(true)
		a.SetExpression("$currentObject/Amount + 1")
		got := formatActionGen(a)
		want := "$Avg = average($Orders, $currentObject/Amount + 1);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("count with attribute still uses bare form (count is special)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$AggregateAction").(*genMf.AggregateListAction)
		a.SetInputListVariableName("Orders")
		a.SetOutputVariableName("N")
		a.SetAggregateFunction(genMf.AggregateFunctionEnumCount)
		a.SetAttributeQualifiedName("Sales.Order.Amount")
		got := formatActionGen(a)
		want := "$N = count($Orders);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("default output var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$AggregateAction").(*genMf.AggregateListAction)
		a.SetInputListVariableName("Orders")
		got := formatActionGen(a)
		want := "$Result = count($Orders);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// TestFormatActionGen_NilAndUnsupported guards two corner cases the
// renderer relies on: nil action gives a stable comment marker; an
// unsupported action returns "" so the caller falls back to its
// existing TODO placeholder line.
func TestFormatActionGen_NilAndUnsupported(t *testing.T) {
	if got := formatActionGen(nil); got != "-- Empty action" {
		t.Errorf("nil action: got %q, want %q", got, "-- Empty action")
	}

	// A valid gen element of a kind we don't handle. CastAction is
	// not in the Stage 3.2.2.a scope.
	cast := newGenAction(t, "Microflows$CastAction")
	if got := formatActionGen(cast); got != "" {
		t.Errorf("unsupported CastAction: got %q, want empty", got)
	}
}

// TestFormatActivityGen_NonAction verifies the wrapper returns "" for
// non-ActionActivity nodes (start/end events, annotations, etc.) so
// the caller's structural framing is not bypassed.
func TestFormatActivityGen_NonAction(t *testing.T) {
	start := newGenAction(t, "Microflows$StartEvent")
	if got := formatActivityGen(nil, start); got != "" {
		t.Errorf("StartEvent: got %q, want empty", got)
	}
	end := newGenAction(t, "Microflows$EndEvent")
	if got := formatActivityGen(nil, end); got != "" {
		t.Errorf("EndEvent: got %q, want empty", got)
	}
}

// ────────────────────────────────────────────────────────
// Fixture-driven integration tests
// ────────────────────────────────────────────────────────

// TestDescribeMicroflowGenToString_ObjectActions_Fixture exercises the
// gen-typed body renderer end-to-end against testdata/expr-checker/
// minimal.mpr. Three actions in this family (CreateObject, ChangeObject,
// Delete) appear in the fixture; CommitAction / RollbackAction /
// AggregateListAction are covered by the direct-construction tests
// above.
func TestDescribeMicroflowGenToString_ObjectActions_Fixture(t *testing.T) {
	w := openMprWriterForTest(t)
	cases := []struct {
		qn   string
		want []string
	}{
		{
			"Administration.NewAccount",
			[]string{
				"$NewAccount = create Administration.Account;",
				"$AccountPasswordData = create Administration.AccountPasswordData (AccountPasswordData_Account = $NewAccount);",
			},
		},
		{
			"Administration.SaveNewAccount",
			[]string{
				"change $Account (Password = $AccountPasswordData/NewPassword) refresh;",
				"delete $AccountPasswordData;",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.qn, func(t *testing.T) {
			mf := findMicroflowByQN(t, w, tc.qn)
			out, err := DescribeMicroflowGenToString(newGenDescribeContext(t, w), mf)
			if err != nil {
				t.Fatalf("DescribeMicroflowGenToString: %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("missing expected line %q in output:\n%s", want, out)
				}
			}
		})
	}
}

// ────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────

// newGenAction constructs a freshly-initialised gen element by BSON
// type name. The switch keeps the test surface small — adding a new
// action kind to the formatter only requires extending this map.
func newGenAction(t *testing.T, typeName string) element.Element {
	t.Helper()
	switch typeName {
	case "Microflows$DeleteAction":
		return genMf.NewDeleteAction()
	case "Microflows$CommitAction":
		return genMf.NewCommitAction()
	case "Microflows$RollbackAction":
		return genMf.NewRollbackAction()
	case "Microflows$CreateObjectAction":
		return genMf.NewCreateObjectAction()
	case "Microflows$ChangeObjectAction":
		return genMf.NewChangeObjectAction()
	case "Microflows$AggregateAction":
		return genMf.NewAggregateListAction()
	case "Microflows$MemberChange":
		return genMf.NewMemberChange()
	case "Microflows$CastAction":
		return genMf.NewCastAction()
	case "Microflows$StartEvent":
		return genMf.NewStartEvent()
	case "Microflows$EndEvent":
		return genMf.NewEndEvent()
	default:
		t.Fatalf("newGenAction: unknown type %q", typeName)
		return nil
	}
}
