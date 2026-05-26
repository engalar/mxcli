// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.a tests for the gen-typed Object Actions family.
// Stage 3.2.2.b tests for the gen-typed Form Actions family.
// Stage 3.2.2.c tests for the gen-typed List operations family.
//
// Two test surfaces:
//  1. Fixture-driven (whatever the testdata MPR happens to contain)
//     covers CreateObjectAction / ChangeObjectAction / DeleteAction /
//     ShowPageAction end-to-end through DescribeMicroflowGenToString.
//  2. Direct-construction unit tests build minimal gen objects in
//     memory to cover every action kind including ones the fixture
//     doesn't carry (CommitAction, RollbackAction, AggregateListAction,
//     CloseFormAction, ShowHomePageAction, ShowMessageAction, plus all
//     of the List operations family — the fixture has no list ops at
//     all so direct construction is the only coverage path).
//
// String comparison is exact — the formatter is intentionally 1:1 with
// the legacy `cmd_microflows_format_action.go` so the migrated output
// can be diffed against the SDK path during the cutover.

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDM "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTx "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// ────────────────────────────────────────────────────────
// Direct-construction unit tests (one per formatter)
// ────────────────────────────────────────────────────────

func TestFormatActionGen_DeleteAction(t *testing.T) {
	a := newGenAction(t, "Microflows$DeleteAction").(*genMf.DeleteAction)
	a.SetDeleteVariableName("Account")

	got := formatActionGen(nil, a)
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
			if got := formatActionGen(nil, a); got != tc.want {
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
			if got := formatActionGen(nil, a); got != tc.want {
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
		got := formatActionGen(nil, a)
		want := "$NewOrder = create Sales.Order;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("default output var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
		a.SetEntityQualifiedName("Sales.Order")
		got := formatActionGen(nil, a)
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

		got := formatActionGen(nil, a)
		want := "$NewOrder = create Sales.Order (Total = 0, Order_Customer = $Customer, Logistics.Order_Carrier = $Carrier);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty entity qualified name falls back to 'Entity'", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateObjectAction").(*genMf.CreateObjectAction)
		a.SetOutputVariableName("X")
		got := formatActionGen(nil, a)
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
		got := formatActionGen(nil, a)
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

		got := formatActionGen(nil, a)
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

		got := formatActionGen(nil, a)
		want := "change $Order (Sales.Order_Customer = $Customer);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("default change variable", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeObjectAction").(*genMf.ChangeObjectAction)
		got := formatActionGen(nil, a)
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
		got := formatActionGen(nil, a)
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
		got := formatActionGen(nil, a)
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
		got := formatActionGen(nil, a)
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
		got := formatActionGen(nil, a)
		want := "$N = count($Orders);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("default output var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$AggregateAction").(*genMf.AggregateListAction)
		a.SetInputListVariableName("Orders")
		got := formatActionGen(nil, a)
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
	if got := formatActionGen(nil, nil); got != "-- Empty action" {
		t.Errorf("nil action: got %q, want %q", got, "-- Empty action")
	}

	// A valid gen element of a kind we don't handle yet. CastAction
	// landed in Stage 3.2.2.e; AppServiceCallAction is held back for
	// the external-integration family in Stage 3.2.2.f.
	unsupported := newGenAction(t, "Microflows$AppServiceCallAction")
	if got := formatActionGen(nil, unsupported); got != "" {
		t.Errorf("unsupported AppServiceCallAction: got %q, want empty", got)
	}
}

// ────────────────────────────────────────────────────────
// Stage 3.2.2.b — Form Actions family (direct construction)
// ────────────────────────────────────────────────────────

func TestFormatActionGen_ShowPageAction(t *testing.T) {
	t.Run("bare page reference (no parameters)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ShowFormAction").(*genMf.ShowPageAction)
		ps := newGenAction(t, "Pages$PageSettings").(*genPg.PageSettings)
		ps.SetPageQualifiedName("Sales.OrderOverview")
		a.SetPageSettings(ps)

		got := formatActionGen(nil, a)
		want := "show page Sales.OrderOverview;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with parameter mapping", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ShowFormAction").(*genMf.ShowPageAction)
		ps := newGenAction(t, "Pages$PageSettings").(*genPg.PageSettings)
		ps.SetPageQualifiedName("Administration.ChangePasswordForm")

		ppm := newGenAction(t, "Pages$PageParameterMapping").(*genPg.PageParameterMapping)
		ppm.SetParameterQualifiedName("Administration.ChangePasswordForm.AccountPasswordData")
		ppm.SetArgument("$AccountPasswordData")
		ps.AddParameterMappings(ppm)

		a.SetPageSettings(ps)

		got := formatActionGen(nil, a)
		want := "show page Administration.ChangePasswordForm($AccountPasswordData = $AccountPasswordData);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("multiple parameter mappings", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ShowFormAction").(*genMf.ShowPageAction)
		ps := newGenAction(t, "Pages$PageSettings").(*genPg.PageSettings)
		ps.SetPageQualifiedName("Sales.OrderEdit")

		for _, pair := range []struct{ qn, arg string }{
			{"Sales.OrderEdit.Order", "$Order"},
			{"Sales.OrderEdit.Customer", "$Customer"},
		} {
			ppm := newGenAction(t, "Pages$PageParameterMapping").(*genPg.PageParameterMapping)
			ppm.SetParameterQualifiedName(pair.qn)
			ppm.SetArgument(pair.arg)
			ps.AddParameterMappings(ppm)
		}
		a.SetPageSettings(ps)

		got := formatActionGen(nil, a)
		want := "show page Sales.OrderEdit($Order = $Order, $Customer = $Customer);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("missing PageSettings falls back to UnknownPage", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ShowFormAction").(*genMf.ShowPageAction)
		got := formatActionGen(nil, a)
		want := "show page UnknownPage;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("PageSettings with empty page name falls back to UnknownPage", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ShowFormAction").(*genMf.ShowPageAction)
		ps := newGenAction(t, "Pages$PageSettings").(*genPg.PageSettings)
		// Leave PageQualifiedName empty.
		a.SetPageSettings(ps)
		got := formatActionGen(nil, a)
		want := "show page UnknownPage;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_CloseFormAction(t *testing.T) {
	cases := []struct {
		name          string
		numberOfPages int32
		want          string
	}{
		{"default (zero)", 0, "close page;"},
		{"single page", 1, "close page;"},
		{"two pages", 2, "close page 2;"},
		{"five pages", 5, "close page 5;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newGenAction(t, "Microflows$CloseFormAction").(*genMf.CloseFormAction)
			a.SetNumberOfPages(tc.numberOfPages)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_ShowHomePageAction(t *testing.T) {
	t.Run("constant output regardless of state", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ShowHomePageAction").(*genMf.ShowHomePageAction)
		got := formatActionGen(nil, a)
		want := "show home page;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("error-handling field set should not affect output", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ShowHomePageAction").(*genMf.ShowHomePageAction)
		a.SetErrorHandlingType("Custom")
		got := formatActionGen(nil, a)
		want := "show home page;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_ShowMessageAction(t *testing.T) {
	// Helper: build a ShowMessageAction with a TextTemplate whose nested
	// Texts$Text carries the supplied {languageCode -> text} translations
	// and the supplied parameter expressions.
	build := func(t *testing.T, msgType string, translations map[string]string, params ...string) *genMf.ShowMessageAction {
		t.Helper()
		a := newGenAction(t, "Microflows$ShowMessageAction").(*genMf.ShowMessageAction)
		if msgType != "" {
			a.SetType(msgType)
		}
		if translations != nil || len(params) > 0 {
			tmpl := newGenAction(t, "Microflows$TextTemplate").(*genMf.TextTemplate)
			if translations != nil {
				text := newGenAction(t, "Texts$Text").(*genTx.Text)
				for lang, body := range translations {
					tr := newGenAction(t, "Texts$Translation").(*genTx.Translation)
					tr.SetLanguageCode(lang)
					tr.SetText(body)
					text.AddTranslations(tr)
				}
				tmpl.SetText(text)
			}
			for _, expr := range params {
				ta := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
				ta.SetExpression(expr)
				tmpl.AddArguments(ta)
			}
			a.SetTemplate(tmpl)
		}
		return a
	}

	t.Run("no template defaults to ellipsis literal and Information type", func(t *testing.T) {
		a := build(t, "", nil)
		got := formatActionGen(nil, a)
		want := "show message '...' type Information;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("explicit Warning type with single en_US translation", func(t *testing.T) {
		a := build(t, genMf.ShowMessageTypeWarning, map[string]string{
			"en_US": "Order saved",
		})
		got := formatActionGen(nil, a)
		want := "show message 'Order saved' type Warning;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("non-English translation falls back when en_US missing", func(t *testing.T) {
		a := build(t, genMf.ShowMessageTypeError, map[string]string{
			"nl_NL": "Bestelling opgeslagen",
		})
		got := formatActionGen(nil, a)
		want := "show message 'Bestelling opgeslagen' type Error;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("en_US wins over other languages", func(t *testing.T) {
		a := build(t, genMf.ShowMessageTypeInformation, map[string]string{
			"nl_NL": "Hallo",
			"en_US": "Hello",
			"de_DE": "Hallo",
		})
		got := formatActionGen(nil, a)
		want := "show message 'Hello' type Information;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("template parameters render as objects clause", func(t *testing.T) {
		a := build(t, genMf.ShowMessageTypeInformation,
			map[string]string{"en_US": "Saved {1} of {2}"},
			"$Order/Number", "$Order/Total",
		)
		got := formatActionGen(nil, a)
		want := "show message 'Saved {1} of {2}' type Information objects [$Order/Number, $Order/Total];"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("template parameters with no translation keep the placeholder text", func(t *testing.T) {
		// No Text element on the TextTemplate, but Arguments are
		// present — the legacy formatter still emits the objects clause
		// alongside the unquoted placeholder text.
		a := build(t, genMf.ShowMessageTypeWarning, nil, "$x")
		got := formatActionGen(nil, a)
		want := "show message '...' type Warning objects [$x];"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("text containing single quotes is escaped", func(t *testing.T) {
		a := build(t, genMf.ShowMessageTypeInformation, map[string]string{
			"en_US": "It's done",
		})
		got := formatActionGen(nil, a)
		want := "show message 'It''s done' type Information;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// Stage 3.2.2.c — List operations family (direct construction)
// ────────────────────────────────────────────────────────

func TestFormatActionGen_CreateListAction(t *testing.T) {
	cases := []struct {
		name      string
		entityQN  string
		outputVar string
		want      string
	}{
		{"basic", "Sales.Order", "Orders", "$Orders = create list of Sales.Order;"},
		{"empty entity falls back to 'Entity'", "", "L", "$L = create list of Entity;"},
		{"cross-module entity uses qualified name", "Logistics.Carrier", "Carriers", "$Carriers = create list of Logistics.Carrier;"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newGenAction(t, "Microflows$CreateListAction").(*genMf.CreateListAction)
			a.SetEntityQualifiedName(tc.entityQN)
			a.SetOutputVariableName(tc.outputVar)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_ChangeListAction(t *testing.T) {
	cases := []struct {
		name    string
		typ     string
		varName string
		value   string
		want    string
	}{
		{"add object", genMf.ChangeListActionTypeAdd, "Orders", "$NewOrder", "add $NewOrder to $Orders;"},
		{"remove object", genMf.ChangeListActionTypeRemove, "Orders", "$OldOrder", "remove $OldOrder from $Orders;"},
		{"clear ignores value", genMf.ChangeListActionTypeClear, "Orders", "$Ignored", "clear $Orders;"},
		{"set assigns expression", genMf.ChangeListActionTypeSet, "Orders", "$NewList", "set $Orders = $NewList;"},
		{"unknown type falls through to legacy form", "Frobnicate", "Orders", "$X", "change list $Orders (Frobnicate);"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newGenAction(t, "Microflows$ChangeListAction").(*genMf.ChangeListAction)
			a.SetChangeVariableName(tc.varName)
			a.SetType(tc.typ)
			a.SetValue(tc.value)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_ListOperationAction_DefaultsAndNil(t *testing.T) {
	t.Run("nil operation gives placeholder with default Result", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ListOperationsAction").(*genMf.ListOperationAction)
		got := formatActionGen(nil, a)
		want := "$Result = list operation ...;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("nil operation respects custom output var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ListOperationsAction").(*genMf.ListOperationAction)
		a.SetOutputVariableName("Custom")
		got := formatActionGen(nil, a)
		want := "$Custom = list operation ...;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty output var defaults to Result for non-nil operation", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ListOperationsAction").(*genMf.ListOperationAction)
		head := newGenAction(t, "Microflows$Head").(*genMf.Head)
		head.SetListVariableName("Orders")
		a.SetOperation(head)
		got := formatActionGen(nil, a)
		want := "$Result = head($Orders);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// newListOpAction is a small helper that builds a ListOperationAction
// wrapping the supplied inner operation with `outputVar`.
func newListOpAction(t *testing.T, outputVar string, op element.Element) *genMf.ListOperationAction {
	t.Helper()
	a := newGenAction(t, "Microflows$ListOperationsAction").(*genMf.ListOperationAction)
	a.SetOutputVariableName(outputVar)
	a.SetOperation(op)
	return a
}

func TestFormatActionGen_HeadOperation(t *testing.T) {
	cases := []struct {
		name    string
		listVar string
		out     string
		want    string
	}{
		{"basic", "Orders", "First", "$First = head($Orders);"},
		{"empty list var still emits dollar prefix", "", "First", "$First = head($);"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newGenAction(t, "Microflows$Head").(*genMf.Head)
			h.SetListVariableName(tc.listVar)
			a := newListOpAction(t, tc.out, h)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_TailOperation(t *testing.T) {
	cases := []struct {
		name    string
		listVar string
		out     string
		want    string
	}{
		{"basic", "Orders", "Rest", "$Rest = tail($Orders);"},
		{"different list", "Customers", "Tail", "$Tail = tail($Customers);"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tl := newGenAction(t, "Microflows$Tail").(*genMf.Tail)
			tl.SetListVariableName(tc.listVar)
			a := newListOpAction(t, tc.out, tl)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_FindByExpression(t *testing.T) {
	cases := []struct {
		name    string
		listVar string
		expr    string
		out     string
		want    string
	}{
		{"basic predicate", "Orders", "$currentObject/Status = 'Open'", "Open", "$Open = find($Orders, $currentObject/Status = 'Open');"},
		{"complex expression", "Items", "$currentObject/Qty > 0 and $currentObject/Active", "Active", "$Active = find($Items, $currentObject/Qty > 0 and $currentObject/Active);"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newGenAction(t, "Microflows$FindByExpression").(*genMf.FindByExpression)
			f.SetListVariableName(tc.listVar)
			f.SetExpression(tc.expr)
			a := newListOpAction(t, tc.out, f)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_FilterByExpression(t *testing.T) {
	cases := []struct {
		name    string
		listVar string
		expr    string
		out     string
		want    string
	}{
		{"basic predicate", "Orders", "$currentObject/Total > 100", "Big", "$Big = filter($Orders, $currentObject/Total > 100);"},
		{"alphanumeric var", "Customers", "$currentObject/Name != empty", "Named", "$Named = filter($Customers, $currentObject/Name != empty);"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newGenAction(t, "Microflows$FilterByExpression").(*genMf.FilterByExpression)
			f.SetListVariableName(tc.listVar)
			f.SetExpression(tc.expr)
			a := newListOpAction(t, tc.out, f)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFormatActionGen_FindByAttribute exercises the gen `Find` element
// (legacy `FindByAttributeOperation`): qualified attribute or
// association reference combined with an expression.
func TestFormatActionGen_FindByAttribute(t *testing.T) {
	t.Run("attribute + expression renders equality form", func(t *testing.T) {
		f := newGenAction(t, "Microflows$Find").(*genMf.Find)
		f.SetListVariableName("Orders")
		f.SetAttributeQualifiedName("Sales.Order.Status")
		f.SetExpression("'Open'")
		a := newListOpAction(t, "Open", f)
		got := formatActionGen(nil, a)
		want := "$Open = find($Orders, Status = 'Open');"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("association + expression strips qualifier", func(t *testing.T) {
		f := newGenAction(t, "Microflows$Find").(*genMf.Find)
		f.SetListVariableName("Orders")
		f.SetAssociationQualifiedName("Sales.Order_Customer")
		f.SetExpression("$Customer")
		a := newListOpAction(t, "Found", f)
		got := formatActionGen(nil, a)
		want := "$Found = find($Orders, Order_Customer = $Customer);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("expression only (no attribute or association)", func(t *testing.T) {
		f := newGenAction(t, "Microflows$Find").(*genMf.Find)
		f.SetListVariableName("Orders")
		f.SetExpression("$currentObject/Total > 0")
		a := newListOpAction(t, "Pos", f)
		got := formatActionGen(nil, a)
		want := "$Pos = find($Orders, $currentObject/Total > 0);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("missing both attribute and expression yields commented placeholder", func(t *testing.T) {
		f := newGenAction(t, "Microflows$Find").(*genMf.Find)
		f.SetListVariableName("Orders")
		a := newListOpAction(t, "X", f)
		got := formatActionGen(nil, a)
		want := "-- $X = find($Orders) — missing attribute/expression"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_FilterByAttribute(t *testing.T) {
	t.Run("attribute + expression renders equality form", func(t *testing.T) {
		f := newGenAction(t, "Microflows$Filter").(*genMf.Filter)
		f.SetListVariableName("Orders")
		f.SetAttributeQualifiedName("Sales.Order.Status")
		f.SetExpression("'Open'")
		a := newListOpAction(t, "Open", f)
		got := formatActionGen(nil, a)
		want := "$Open = filter($Orders, Status = 'Open');"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("expression only", func(t *testing.T) {
		f := newGenAction(t, "Microflows$Filter").(*genMf.Filter)
		f.SetListVariableName("Orders")
		f.SetExpression("$currentObject/Total > 0")
		a := newListOpAction(t, "Pos", f)
		got := formatActionGen(nil, a)
		want := "$Pos = filter($Orders, $currentObject/Total > 0);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("missing both yields commented placeholder", func(t *testing.T) {
		f := newGenAction(t, "Microflows$Filter").(*genMf.Filter)
		f.SetListVariableName("Orders")
		a := newListOpAction(t, "X", f)
		got := formatActionGen(nil, a)
		want := "-- $X = filter($Orders) — missing attribute/expression"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_SortOperation(t *testing.T) {
	t.Run("no sort items renders bare sort()", func(t *testing.T) {
		s := newGenAction(t, "Microflows$Sort").(*genMf.Sort)
		s.SetListVariableName("Orders")
		a := newListOpAction(t, "Sorted", s)
		got := formatActionGen(nil, a)
		want := "$Sorted = sort($Orders);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("single ascending column via AttributePath fallback", func(t *testing.T) {
		s := newGenAction(t, "Microflows$Sort").(*genMf.Sort)
		s.SetListVariableName("Orders")

		list := newGenAction(t, "Microflows$SortItemList").(*genMf.SortItemList)
		si := newGenAction(t, "Microflows$SortItem").(*genMf.SortItem)
		si.SetAttributePath("Sales.Order.OrderDate")
		si.SetSortOrder(genMf.SortOrderEnumAscending)
		list.AddItems(si)
		s.SetSortItemList(list)

		a := newListOpAction(t, "Sorted", s)
		got := formatActionGen(nil, a)
		want := "$Sorted = sort($Orders, OrderDate asc);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("AttributeRef wins over AttributePath", func(t *testing.T) {
		s := newGenAction(t, "Microflows$Sort").(*genMf.Sort)
		s.SetListVariableName("Orders")

		list := newGenAction(t, "Microflows$SortItemList").(*genMf.SortItemList)
		si := newGenAction(t, "Microflows$SortItem").(*genMf.SortItem)
		ar := newGenAction(t, "DomainModels$AttributeRef").(*genDM.AttributeRef)
		ar.SetAttributeQualifiedName("Sales.Order.Total")
		si.SetAttributeRef(ar)
		// AttributePath would lose the race even if set.
		si.SetAttributePath("ignored.Path.Loser")
		si.SetSortOrder(genMf.SortOrderEnumDescending)
		list.AddItems(si)
		s.SetSortItemList(list)

		a := newListOpAction(t, "Sorted", s)
		got := formatActionGen(nil, a)
		want := "$Sorted = sort($Orders, Total desc);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("multiple columns with mixed direction", func(t *testing.T) {
		s := newGenAction(t, "Microflows$Sort").(*genMf.Sort)
		s.SetListVariableName("Orders")

		list := newGenAction(t, "Microflows$SortItemList").(*genMf.SortItemList)
		for _, p := range []struct {
			path, dir string
		}{
			{"Sales.Order.Customer", genMf.SortOrderEnumAscending},
			{"Sales.Order.Total", genMf.SortOrderEnumDescending},
		} {
			si := newGenAction(t, "Microflows$SortItem").(*genMf.SortItem)
			si.SetAttributePath(p.path)
			si.SetSortOrder(p.dir)
			list.AddItems(si)
		}
		s.SetSortItemList(list)

		a := newListOpAction(t, "Sorted", s)
		got := formatActionGen(nil, a)
		want := "$Sorted = sort($Orders, Customer asc, Total desc);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("missing attribute substitutes ellipsis placeholder", func(t *testing.T) {
		s := newGenAction(t, "Microflows$Sort").(*genMf.Sort)
		s.SetListVariableName("Orders")

		list := newGenAction(t, "Microflows$SortItemList").(*genMf.SortItemList)
		si := newGenAction(t, "Microflows$SortItem").(*genMf.SortItem)
		// Leave AttributePath/AttributeRef empty.
		si.SetSortOrder(genMf.SortOrderEnumAscending)
		list.AddItems(si)
		s.SetSortItemList(list)

		a := newListOpAction(t, "Sorted", s)
		got := formatActionGen(nil, a)
		want := "$Sorted = sort($Orders, ... asc);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_SetOperations(t *testing.T) {
	cases := []struct {
		name     string
		typeName string
		want     string
	}{
		{"union", "Microflows$Union", "$Combined = union($A, $B);"},
		{"intersect", "Microflows$Intersect", "$Combined = intersect($A, $B);"},
		{"subtract", "Microflows$Subtract", "$Combined = subtract($A, $B);"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			el := newGenAction(t, tc.typeName)
			switch o := el.(type) {
			case *genMf.Union:
				o.SetListVariableName("A")
				o.SetSecondListOrObjectVariableName("B")
			case *genMf.Intersect:
				o.SetListVariableName("A")
				o.SetSecondListOrObjectVariableName("B")
			case *genMf.Subtract:
				o.SetListVariableName("A")
				o.SetSecondListOrObjectVariableName("B")
			default:
				t.Fatalf("unexpected element type %T", el)
			}
			a := newListOpAction(t, "Combined", el)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatActionGen_ContainsAndEquals(t *testing.T) {
	t.Run("contains", func(t *testing.T) {
		c := newGenAction(t, "Microflows$Contains").(*genMf.Contains)
		c.SetListVariableName("Orders")
		c.SetSecondListOrObjectVariableName("Order")
		a := newListOpAction(t, "Has", c)
		got := formatActionGen(nil, a)
		want := "$Has = contains($Orders, $Order);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("equals (gen ListEquals)", func(t *testing.T) {
		e := newGenAction(t, "Microflows$ListEquals").(*genMf.ListEquals)
		e.SetListVariableName("A")
		e.SetSecondListOrObjectVariableName("B")
		a := newListOpAction(t, "Eq", e)
		got := formatActionGen(nil, a)
		want := "$Eq = equals($A, $B);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFormatActionGen_ListRangeOperation(t *testing.T) {
	cases := []struct {
		name   string
		offset string
		limit  string
		want   string
	}{
		{"no custom range collapses to bare range()", "", "", "$Page = range($Orders);"},
		{"offset only", "10", "", "$Page = range($Orders, 10);"},
		{"limit only forces zero offset", "", "20", "$Page = range($Orders, 0, 20);"},
		{"offset and limit", "10", "20", "$Page = range($Orders, 10, 20);"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newGenAction(t, "Microflows$ListRange").(*genMf.ListRange)
			r.SetListVariableName("Orders")
			if tc.offset != "" || tc.limit != "" {
				cr := newGenAction(t, "Microflows$CustomRange").(*genMf.CustomRange)
				cr.SetOffsetExpression(tc.offset)
				cr.SetLimitExpression(tc.limit)
				r.SetCustomRange(cr)
			}
			a := newListOpAction(t, "Page", r)
			if got := formatActionGen(nil, a); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFormatListOperationGen_DirectDispatch exercises the helper
// directly to guard the nil-op placeholder path and a handful of
// representative dispatch cases without going through the wrapping
// ListOperationAction. This is the entry point the renderer uses for
// any future caller that already has the inner operation in hand.
func TestFormatListOperationGen_DirectDispatch(t *testing.T) {
	t.Run("nil op placeholder", func(t *testing.T) {
		got := formatListOperationGen(nil, "X")
		want := "$X = list operation ...;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("unsupported op type renders %T placeholder", func(t *testing.T) {
		// Reach for an element kind that's a valid gen type but is
		// not a list operation primitive — Annotation works.
		ann := genMf.NewAnnotation()
		got := formatListOperationGen(ann, "X")
		if !strings.Contains(got, "$X = list operation ") {
			t.Errorf("got %q, want containing %q", got, "$X = list operation ")
		}
		if !strings.Contains(got, "Annotation") {
			t.Errorf("got %q, want containing %q", got, "Annotation")
		}
	})
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

// TestDescribeMicroflowGenToString_FormActions_Fixture exercises the
// gen-typed body renderer end-to-end for the Form Actions family. The
// fixture's only Form Action surface is ShowPage; CloseForm /
// ShowHomePage / ShowMessage are covered by the direct-construction
// tests above (the fixture does not carry those action kinds).
//
// Each case asserts the formatter's full statement appears in the
// rendered microflow body — enough to confirm the activity-level dispatch
// reaches `formatShowPageActionGen` through `formatActivityGen`.
func TestDescribeMicroflowGenToString_FormActions_Fixture(t *testing.T) {
	w := openMprWriterForTest(t)
	cases := []struct {
		qn   string
		want []string
	}{
		{
			"Administration.NewAccount",
			[]string{
				"show page Administration.Account_New($AccountPasswordData = $AccountPasswordData);",
			},
		},
		{
			"Administration.ShowMyPasswordForm",
			[]string{
				"show page Administration.ChangeMyPasswordForm($AccountPasswordData = $AccountPasswordData);",
			},
		},
		{
			"Administration.ShowPasswordForm",
			[]string{
				"show page Administration.ChangePasswordForm($AccountPasswordData = $AccountPasswordData);",
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
	// Stage 3.2.2.b — Form Actions family.
	case "Microflows$ShowFormAction": // gen Go type: ShowPageAction
		return genMf.NewShowPageAction()
	case "Microflows$CloseFormAction":
		return genMf.NewCloseFormAction()
	case "Microflows$ShowHomePageAction":
		return genMf.NewShowHomePageAction()
	case "Microflows$ShowMessageAction":
		return genMf.NewShowMessageAction()
	case "Microflows$TextTemplate":
		return genMf.NewTextTemplate()
	case "Microflows$TemplateArgument":
		return genMf.NewTemplateArgument()
	case "Pages$PageSettings":
		return genPg.NewPageSettings()
	case "Pages$PageParameterMapping":
		return genPg.NewPageParameterMapping()
	case "Texts$Text":
		return genTx.NewText()
	case "Texts$Translation":
		return genTx.NewTranslation()
	// Stage 3.2.2.c — List operations family.
	case "Microflows$CreateListAction":
		return genMf.NewCreateListAction()
	case "Microflows$ChangeListAction":
		return genMf.NewChangeListAction()
	case "Microflows$ListOperationsAction": // gen Go type: ListOperationAction
		return genMf.NewListOperationAction()
	case "Microflows$Head":
		return genMf.NewHead()
	case "Microflows$Tail":
		return genMf.NewTail()
	case "Microflows$Find":
		return genMf.NewFind()
	case "Microflows$FindByExpression":
		return genMf.NewFindByExpression()
	case "Microflows$Filter":
		return genMf.NewFilter()
	case "Microflows$FilterByExpression":
		return genMf.NewFilterByExpression()
	case "Microflows$Sort":
		return genMf.NewSort()
	case "Microflows$SortItemList":
		return genMf.NewSortItemList()
	case "Microflows$SortItem":
		return genMf.NewSortItem()
	case "Microflows$Union":
		return genMf.NewUnion()
	case "Microflows$Intersect":
		return genMf.NewIntersect()
	case "Microflows$Subtract":
		return genMf.NewSubtract()
	case "Microflows$Contains":
		return genMf.NewContains()
	case "Microflows$ListEquals":
		return genMf.NewListEquals()
	case "Microflows$ListRange":
		return genMf.NewListRange()
	case "Microflows$CustomRange":
		return genMf.NewCustomRange()
	case "DomainModels$AttributeRef":
		return genDM.NewAttributeRef()
	// Stage 3.2.2.d — Microflow/Java/JavaScript call action family.
	case "Microflows$MicroflowCallAction":
		return genMf.NewMicroflowCallAction()
	case "Microflows$MicroflowCall":
		return genMf.NewMicroflowCall()
	case "Microflows$MicroflowCallParameterMapping":
		return genMf.NewMicroflowCallParameterMapping()
	case "Microflows$NanoflowCallAction":
		return genMf.NewNanoflowCallAction()
	case "Microflows$NanoflowCall":
		return genMf.NewNanoflowCall()
	case "Microflows$NanoflowCallParameterMapping":
		return genMf.NewNanoflowCallParameterMapping()
	case "Microflows$JavaActionCallAction":
		return genMf.NewJavaActionCallAction()
	case "Microflows$JavaActionParameterMapping":
		return genMf.NewJavaActionParameterMapping()
	case "Microflows$JavaScriptActionCallAction":
		return genMf.NewJavaScriptActionCallAction()
	case "Microflows$JavaScriptActionParameterMapping":
		return genMf.NewJavaScriptActionParameterMapping()
	case "Microflows$StringTemplateParameterValue":
		return genMf.NewStringTemplateParameterValue()
	case "Microflows$BasicCodeActionParameterValue":
		return genMf.NewBasicCodeActionParameterValue()
	case "Microflows$MicroflowParameterValue":
		return genMf.NewMicroflowParameterValue()
	case "Microflows$EntityTypeCodeActionParameterValue":
		return genMf.NewEntityTypeCodeActionParameterValue()
	case "Microflows$TypedTemplate":
		return genMf.NewTypedTemplate()
	// Stage 3.2.2.e — Variable / Expression / Data family.
	case "Microflows$CreateVariableAction":
		return genMf.NewCreateVariableAction()
	case "Microflows$ChangeVariableAction":
		return genMf.NewChangeVariableAction()
	case "Microflows$RetrieveAction":
		return genMf.NewRetrieveAction()
	case "Microflows$DatabaseRetrieveSource":
		return genMf.NewDatabaseRetrieveSource()
	case "Microflows$AssociationRetrieveSource":
		return genMf.NewAssociationRetrieveSource()
	case "Microflows$ConstantRange":
		return genMf.NewConstantRange()
	case "Microflows$LogMessageAction":
		return genMf.NewLogMessageAction()
	case "Microflows$StringTemplate":
		return genMf.NewStringTemplate()
	case "Microflows$DownloadFileAction":
		return genMf.NewDownloadFileAction()
	case "Microflows$ValidationFeedbackAction":
		return genMf.NewValidationFeedbackAction()
	// Sentinel for the "still unsupported" guard test in
	// TestFormatActionGen_NilAndUnsupported. Stage 3.2.2.f will
	// pick this up.
	case "Microflows$AppServiceCallAction":
		return genMf.NewAppServiceCallAction()
	default:
		t.Fatalf("newGenAction: unknown type %q", typeName)
		return nil
	}
}
