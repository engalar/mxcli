// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.d tests for the gen-typed Microflow/Java/JavaScript call
// action family.
//
// All tests are direct-construction. The expr-checker fixture has no
// MicroflowCallAction / NanoflowCallAction / Java(Script)ActionCallAction
// surface, so the only available coverage path is to build minimal gen
// objects in memory and feed them through `formatActionGen`.
//
// String comparison is exact — the formatter is intentionally 1:1 with
// legacy `cmd_microflows_format_action.go` so the migrated output can
// be diffed against the SDK path during the cutover.

package executor

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ────────────────────────────────────────────────────────
// MicroflowCallAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_MicroflowCallAction(t *testing.T) {
	t.Run("nil MicroflowCall falls back to placeholder name, no return var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$MicroflowCallAction").(*genMf.MicroflowCallAction)
		got := formatActionGen(a)
		want := "call microflow Microflow();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("bare call (no params, no return var)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$MicroflowCallAction").(*genMf.MicroflowCallAction)
		call := newGenAction(t, "Microflows$MicroflowCall").(*genMf.MicroflowCall)
		call.SetMicroflowQualifiedName("Sales.RecalculateTotals")
		a.SetMicroflowCall(call)

		got := formatActionGen(a)
		want := "call microflow Sales.RecalculateTotals();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with return variable", func(t *testing.T) {
		a := newGenAction(t, "Microflows$MicroflowCallAction").(*genMf.MicroflowCallAction)
		call := newGenAction(t, "Microflows$MicroflowCall").(*genMf.MicroflowCall)
		call.SetMicroflowQualifiedName("Sales.GetCustomerTotal")
		a.SetMicroflowCall(call)
		a.SetUseReturnVariable(true)
		a.SetOutputVariableName("Total")

		got := formatActionGen(a)
		want := "$Total = call microflow Sales.GetCustomerTotal();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("UseReturnVariable false suppresses return var even if name set", func(t *testing.T) {
		a := newGenAction(t, "Microflows$MicroflowCallAction").(*genMf.MicroflowCallAction)
		call := newGenAction(t, "Microflows$MicroflowCall").(*genMf.MicroflowCall)
		call.SetMicroflowQualifiedName("Sales.Touch")
		a.SetMicroflowCall(call)
		a.SetOutputVariableName("Discarded")
		// UseReturnVariable left false.

		got := formatActionGen(a)
		want := "call microflow Sales.Touch();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with multiple parameter mappings", func(t *testing.T) {
		a := newGenAction(t, "Microflows$MicroflowCallAction").(*genMf.MicroflowCallAction)
		call := newGenAction(t, "Microflows$MicroflowCall").(*genMf.MicroflowCall)
		call.SetMicroflowQualifiedName("Sales.PlaceOrder")

		for _, pair := range []struct{ qn, arg string }{
			{"Sales.PlaceOrder.Customer", "$Customer"},
			{"Sales.PlaceOrder.Total", "$Total"},
		} {
			pm := newGenAction(t, "Microflows$MicroflowCallParameterMapping").(*genMf.MicroflowCallParameterMapping)
			pm.SetParameterQualifiedName(pair.qn)
			pm.SetArgument(pair.arg)
			call.AddParameterMappings(pm)
		}
		a.SetMicroflowCall(call)
		a.SetUseReturnVariable(true)
		a.SetOutputVariableName("Order")

		got := formatActionGen(a)
		want := "$Order = call microflow Sales.PlaceOrder(Customer = $Customer, Total = $Total);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty MicroflowCall.Microflow falls back to Microflow placeholder", func(t *testing.T) {
		a := newGenAction(t, "Microflows$MicroflowCallAction").(*genMf.MicroflowCallAction)
		call := newGenAction(t, "Microflows$MicroflowCall").(*genMf.MicroflowCall)
		// Leave qualified name empty.
		a.SetMicroflowCall(call)

		got := formatActionGen(a)
		want := "call microflow Microflow();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// NanoflowCallAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_NanoflowCallAction(t *testing.T) {
	t.Run("nil NanoflowCall falls back to placeholder", func(t *testing.T) {
		a := newGenAction(t, "Microflows$NanoflowCallAction").(*genMf.NanoflowCallAction)
		got := formatActionGen(a)
		want := "call nanoflow Nanoflow();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("bare nanoflow call", func(t *testing.T) {
		a := newGenAction(t, "Microflows$NanoflowCallAction").(*genMf.NanoflowCallAction)
		call := newGenAction(t, "Microflows$NanoflowCall").(*genMf.NanoflowCall)
		call.SetNanoflowQualifiedName("Mobile.OnLogin")
		a.SetNanoflowCall(call)

		got := formatActionGen(a)
		want := "call nanoflow Mobile.OnLogin();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with parameter mappings and return var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$NanoflowCallAction").(*genMf.NanoflowCallAction)
		call := newGenAction(t, "Microflows$NanoflowCall").(*genMf.NanoflowCall)
		call.SetNanoflowQualifiedName("Mobile.LoadProfile")

		pm := newGenAction(t, "Microflows$NanoflowCallParameterMapping").(*genMf.NanoflowCallParameterMapping)
		pm.SetParameterQualifiedName("Mobile.LoadProfile.AccountId")
		pm.SetArgument("$Id")
		call.AddParameterMappings(pm)

		a.SetNanoflowCall(call)
		a.SetUseReturnVariable(true)
		a.SetOutputVariableName("Profile")

		got := formatActionGen(a)
		want := "$Profile = call nanoflow Mobile.LoadProfile(AccountId = $Id);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("UseReturnVariable false suppresses return var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$NanoflowCallAction").(*genMf.NanoflowCallAction)
		call := newGenAction(t, "Microflows$NanoflowCall").(*genMf.NanoflowCall)
		call.SetNanoflowQualifiedName("Mobile.Touch")
		a.SetNanoflowCall(call)
		a.SetOutputVariableName("Discarded") // ignored — UseReturnVariable=false

		got := formatActionGen(a)
		want := "call nanoflow Mobile.Touch();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// JavaActionCallAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_JavaActionCallAction(t *testing.T) {
	t.Run("missing action reference falls back to placeholder", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		got := formatActionGen(a)
		want := "call java action JavaAction();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("bare action with no params", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.SendEmail")

		got := formatActionGen(a)
		want := "call java action Toolbox.SendEmail();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with string template parameter and return var", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.RenderTemplate")
		a.SetUseReturnVariable(true)
		a.SetOutputVariableName("Rendered")

		pm := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pm.SetParameterQualifiedName("Toolbox.RenderTemplate.Greeting")
		pv := newGenAction(t, "Microflows$StringTemplateParameterValue").(*genMf.StringTemplateParameterValue)
		tt := newGenAction(t, "Microflows$TypedTemplate").(*genMf.TypedTemplate)
		tt.SetText("Hello {1}!")
		pv.SetTypedTemplate(tt)
		pm.SetValue(pv)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "$Rendered = call java action Toolbox.RenderTemplate(Greeting = 'Hello {1}!');"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("ParameterValue fallback when Value unset", func(t *testing.T) {
		// Some MPRs carry the Value under the newer ParameterValue
		// field instead of the legacy Value field. Verify the
		// formatter prefers Value but falls back to ParameterValue.
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.Echo")

		pm := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pm.SetParameterQualifiedName("Toolbox.Echo.Input")
		pv := newGenAction(t, "Microflows$BasicCodeActionParameterValue").(*genMf.BasicCodeActionParameterValue)
		pv.SetArgument("$Input")
		pm.SetParameterValue(pv) // not SetValue
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "call java action Toolbox.Echo(Input = $Input);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("BasicCode empty argument renders as 'empty'", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.Maybe")

		pm := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pm.SetParameterQualifiedName("Toolbox.Maybe.OptionalArg")
		pv := newGenAction(t, "Microflows$BasicCodeActionParameterValue").(*genMf.BasicCodeActionParameterValue)
		// Argument left empty.
		pm.SetValue(pv)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "call java action Toolbox.Maybe(OptionalArg = empty);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("MicroflowParameterValue renders as quoted QN", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.RegisterCallback")

		pm := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pm.SetParameterQualifiedName("Toolbox.RegisterCallback.Microflow")
		pv := newGenAction(t, "Microflows$MicroflowParameterValue").(*genMf.MicroflowParameterValue)
		pv.SetMicroflowQualifiedName("Sales.OnOrderPlaced")
		pm.SetValue(pv)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "call java action Toolbox.RegisterCallback(Microflow = 'Sales.OnOrderPlaced');"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("MicroflowParameterValue empty QN renders as 'empty'", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.RegisterCallback")

		pm := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pm.SetParameterQualifiedName("Toolbox.RegisterCallback.Microflow")
		pv := newGenAction(t, "Microflows$MicroflowParameterValue").(*genMf.MicroflowParameterValue)
		// Empty QN.
		pm.SetValue(pv)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "call java action Toolbox.RegisterCallback(Microflow = empty);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("EntityTypeCode renders as quoted QN", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.GetMetaInfo")

		pm := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pm.SetParameterQualifiedName("Toolbox.GetMetaInfo.EntityType")
		pv := newGenAction(t, "Microflows$EntityTypeCodeActionParameterValue").(*genMf.EntityTypeCodeActionParameterValue)
		pv.SetEntityQualifiedName("Sales.Order")
		pm.SetValue(pv)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "call java action Toolbox.GetMetaInfo(EntityType = 'Sales.Order');"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("multiple parameter mappings preserve order", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.Combo")

		// First: ExpressionBased via the codec fallback path.
		pm1 := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pm1.SetParameterQualifiedName("Toolbox.Combo.Expr")
		pm1.SetValue(newExpressionBasedParameterValue(t, "$x + 1"))
		a.AddParameterMappings(pm1)

		// Second: BasicCode.
		pm2 := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pm2.SetParameterQualifiedName("Toolbox.Combo.Basic")
		basic := newGenAction(t, "Microflows$BasicCodeActionParameterValue").(*genMf.BasicCodeActionParameterValue)
		basic.SetArgument("$y")
		pm2.SetValue(basic)
		a.AddParameterMappings(pm2)

		got := formatActionGen(a)
		want := "call java action Toolbox.Combo(Expr = $x + 1, Basic = $y);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("mapping with nil value is dropped", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaActionCallAction").(*genMf.JavaActionCallAction)
		a.SetJavaActionQualifiedName("Toolbox.Skip")

		pmDrop := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pmDrop.SetParameterQualifiedName("Toolbox.Skip.NoValue")
		// No SetValue / SetParameterValue.
		a.AddParameterMappings(pmDrop)

		pmKeep := newGenAction(t, "Microflows$JavaActionParameterMapping").(*genMf.JavaActionParameterMapping)
		pmKeep.SetParameterQualifiedName("Toolbox.Skip.Keep")
		basic := newGenAction(t, "Microflows$BasicCodeActionParameterValue").(*genMf.BasicCodeActionParameterValue)
		basic.SetArgument("$kept")
		pmKeep.SetValue(basic)
		a.AddParameterMappings(pmKeep)

		got := formatActionGen(a)
		want := "call java action Toolbox.Skip(Keep = $kept);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// JavaScriptActionCallAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_JavaScriptActionCallAction(t *testing.T) {
	t.Run("missing action ref, no params: literal placeholder", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaScriptActionCallAction").(*genMf.JavaScriptActionCallAction)
		got := formatActionGen(a)
		want := "-- JavaScriptAction: missing action reference"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("missing action ref, 1 param: singular label", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaScriptActionCallAction").(*genMf.JavaScriptActionCallAction)
		pm := newGenAction(t, "Microflows$JavaScriptActionParameterMapping").(*genMf.JavaScriptActionParameterMapping)
		pm.SetParameterQualifiedName("Mobile.Echo.Input")
		basic := newGenAction(t, "Microflows$BasicCodeActionParameterValue").(*genMf.BasicCodeActionParameterValue)
		basic.SetArgument("$x")
		pm.SetParameterValue(basic)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "-- JavaScriptAction: missing action reference (1 param)"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("missing action ref, multiple params: plural label", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaScriptActionCallAction").(*genMf.JavaScriptActionCallAction)
		for i := 0; i < 3; i++ {
			pm := newGenAction(t, "Microflows$JavaScriptActionParameterMapping").(*genMf.JavaScriptActionParameterMapping)
			pm.SetParameterQualifiedName("Mobile.X.P")
			a.AddParameterMappings(pm)
		}
		got := formatActionGen(a)
		want := "-- JavaScriptAction: missing action reference (3 params)"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("bare call (no params, no return var)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaScriptActionCallAction").(*genMf.JavaScriptActionCallAction)
		a.SetJavaScriptActionQualifiedName("Mobile.Beep")

		got := formatActionGen(a)
		want := "call javascript action Mobile.Beep();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with return variable and string template param", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaScriptActionCallAction").(*genMf.JavaScriptActionCallAction)
		a.SetJavaScriptActionQualifiedName("Mobile.RenderTemplate")
		a.SetUseReturnVariable(true)
		a.SetOutputVariableName("Rendered")

		pm := newGenAction(t, "Microflows$JavaScriptActionParameterMapping").(*genMf.JavaScriptActionParameterMapping)
		pm.SetParameterQualifiedName("Mobile.RenderTemplate.Body")
		pv := newGenAction(t, "Microflows$StringTemplateParameterValue").(*genMf.StringTemplateParameterValue)
		tt := newGenAction(t, "Microflows$TypedTemplate").(*genMf.TypedTemplate)
		tt.SetText("Hi {1}")
		pv.SetTypedTemplate(tt)
		pm.SetParameterValue(pv)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "$Rendered = call javascript action Mobile.RenderTemplate(Body = 'Hi {1}');"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("BasicCode empty arg is dropped (JS variant), other params preserved", func(t *testing.T) {
		// JavaScript-side: BasicCode with empty argument renders as
		// "" and the mapping is dropped — different from Java's
		// "empty" placeholder.
		a := newGenAction(t, "Microflows$JavaScriptActionCallAction").(*genMf.JavaScriptActionCallAction)
		a.SetJavaScriptActionQualifiedName("Mobile.Combo")

		pmDrop := newGenAction(t, "Microflows$JavaScriptActionParameterMapping").(*genMf.JavaScriptActionParameterMapping)
		pmDrop.SetParameterQualifiedName("Mobile.Combo.Optional")
		emptyBasic := newGenAction(t, "Microflows$BasicCodeActionParameterValue").(*genMf.BasicCodeActionParameterValue)
		pmDrop.SetParameterValue(emptyBasic)
		a.AddParameterMappings(pmDrop)

		pmKeep := newGenAction(t, "Microflows$JavaScriptActionParameterMapping").(*genMf.JavaScriptActionParameterMapping)
		pmKeep.SetParameterQualifiedName("Mobile.Combo.Required")
		basic := newGenAction(t, "Microflows$BasicCodeActionParameterValue").(*genMf.BasicCodeActionParameterValue)
		basic.SetArgument("$req")
		pmKeep.SetParameterValue(basic)
		a.AddParameterMappings(pmKeep)

		got := formatActionGen(a)
		want := "call javascript action Mobile.Combo(Required = $req);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("EntityTypeCode renders as quoted QN", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaScriptActionCallAction").(*genMf.JavaScriptActionCallAction)
		a.SetJavaScriptActionQualifiedName("Mobile.Inspect")

		pm := newGenAction(t, "Microflows$JavaScriptActionParameterMapping").(*genMf.JavaScriptActionParameterMapping)
		pm.SetParameterQualifiedName("Mobile.Inspect.EntityType")
		pv := newGenAction(t, "Microflows$EntityTypeCodeActionParameterValue").(*genMf.EntityTypeCodeActionParameterValue)
		pv.SetEntityQualifiedName("Sales.Order")
		pm.SetParameterValue(pv)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "call javascript action Mobile.Inspect(EntityType = 'Sales.Order');"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("MicroflowParameterValue is rejected for JS actions (mapping dropped)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$JavaScriptActionCallAction").(*genMf.JavaScriptActionCallAction)
		a.SetJavaScriptActionQualifiedName("Mobile.Reject")

		pm := newGenAction(t, "Microflows$JavaScriptActionParameterMapping").(*genMf.JavaScriptActionParameterMapping)
		pm.SetParameterQualifiedName("Mobile.Reject.NotAllowed")
		pv := newGenAction(t, "Microflows$MicroflowParameterValue").(*genMf.MicroflowParameterValue)
		pv.SetMicroflowQualifiedName("Sales.OnX")
		pm.SetParameterValue(pv)
		a.AddParameterMappings(pm)

		got := formatActionGen(a)
		want := "call javascript action Mobile.Reject();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// formatCodeActionParameterValueGen — direct unit coverage
// ────────────────────────────────────────────────────────

func TestFormatCodeActionParameterValueGen_NilAndUnknown(t *testing.T) {
	if got := formatCodeActionParameterValueGen(nil, true); got != "" {
		t.Errorf("nil: got %q, want empty", got)
	}
	// Pass an unrelated gen element — should fall through to "" so
	// the caller drops the mapping.
	other := newGenAction(t, "Microflows$DeleteAction")
	if got := formatCodeActionParameterValueGen(other, true); got != "" {
		t.Errorf("unrelated element: got %q, want empty", got)
	}
}

// TestFormatCodeActionParameterValueGen_ExpressionBasedFallback exercises
// the codec fall-back path for ExpressionBasedCodeActionParameterValue:
// the gen type is a stub with no Expression getter and no registered
// factory, so a real BSON load lands the doc in *element.Base. We
// simulate that here by hand-building a Base with a raw doc carrying
// `$Type=Microflows$ExpressionBasedCodeActionParameterValue` and an
// `Expression` field, then verifying the formatter pulls the expression.
func TestFormatCodeActionParameterValueGen_ExpressionBasedFallback(t *testing.T) {
	v := newExpressionBasedParameterValue(t, "$x + 1")
	got := formatCodeActionParameterValueGen(v, true)
	want := "$x + 1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatCodeActionParameterValueGen_StringTemplate_NilTypedTemplate(t *testing.T) {
	pv := newGenAction(t, "Microflows$StringTemplateParameterValue").(*genMf.StringTemplateParameterValue)
	// No TypedTemplate set.
	if got := formatCodeActionParameterValueGen(pv, true); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFormatCodeActionParameterValueGen_AllowMicroflowParamGate(t *testing.T) {
	pv := newGenAction(t, "Microflows$MicroflowParameterValue").(*genMf.MicroflowParameterValue)
	pv.SetMicroflowQualifiedName("Sales.OnX")
	if got := formatCodeActionParameterValueGen(pv, true); got != "'Sales.OnX'" {
		t.Errorf("Java host: got %q", got)
	}
	if got := formatCodeActionParameterValueGen(pv, false); got != "" {
		t.Errorf("JS host: got %q, want empty", got)
	}
}

// ────────────────────────────────────────────────────────
// Helpers (Stage 3.2.2.d-only)
// ────────────────────────────────────────────────────────

// newExpressionBasedParameterValue builds an *element.Base whose raw
// BSON is the legacy on-disk shape for
// `Microflows$ExpressionBasedCodeActionParameterValue`. Used by the
// fall-back-path tests because the gen type is a stub today (no
// registered factory, no Expression getter).
func newExpressionBasedParameterValue(t *testing.T, expr string) element.Element {
	t.Helper()
	raw, err := bson.Marshal(bson.D{
		{Key: "$Type", Value: "Microflows$ExpressionBasedCodeActionParameterValue"},
		{Key: "Expression", Value: expr},
	})
	if err != nil {
		t.Fatalf("bson.Marshal: %v", err)
	}
	b := &element.Base{}
	b.SetTypeName("Microflows$ExpressionBasedCodeActionParameterValue")
	b.SetRaw(raw)
	return b
}
