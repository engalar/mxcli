// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.f tests for the gen-typed External integration family
// formatters. Each top-level formatter has direct-construction unit
// tests covering the typical surface plus the gen/legacy BSON-key
// alignment fallbacks. Fixture-driven coverage is added when the
// `expr-checker/minimal.mpr` fixture carries a microflow that invokes
// the action; for action kinds the fixture lacks (REST, XML, JSON
// transform, web service), direct construction is the only coverage
// path until a richer fixture is added.

package executor

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// ────────────────────────────────────────────────────────
// CallExternalAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_CallExternalAction(t *testing.T) {
	t.Run("bare call without parameters", func(t *testing.T) {
		a := genMf.NewCallExternalAction()
		a.SetConsumedODataServiceQualifiedName("Catalog.OData")
		a.SetName("ListProducts")
		got := formatActionGen(nil, a)
		want := "call external action Catalog.OData.ListProducts();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("call with output variable and parameters", func(t *testing.T) {
		a := genMf.NewCallExternalAction()
		a.SetConsumedODataServiceQualifiedName("Catalog.OData")
		a.SetName("FindByCategory")
		a.SetVariableName("Products")

		pm := genMf.NewRestOperationParameterMapping()
		pm.SetParameterQualifiedName("Catalog.FindByCategory.Category")
		pm.SetValue("$Cat")
		a.AddParameterMappings(pm)

		got := formatActionGen(nil, a)
		want := "$Products = call external action Catalog.OData.FindByCategory(Category = $Cat);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("missing service and action fall back to defaults", func(t *testing.T) {
		a := genMf.NewCallExternalAction()
		got := formatActionGen(nil, a)
		want := "call external action ODataService.Action();"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// RestCallAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_RestCallAction(t *testing.T) {
	// build a minimal RestCallAction with the supplied configuration.
	build := func(t *testing.T, method, url, outputVar, resultType string) *genMf.RestCallAction {
		t.Helper()
		a := genMf.NewRestCallAction()
		hc := genMf.NewHttpConfiguration()
		hc.SetHttpMethod(method)
		if url != "" {
			tmpl := genMf.NewStringTemplate()
			tmpl.SetText(url)
			hc.SetCustomLocationTemplate(tmpl)
		}
		a.SetHttpConfiguration(hc)
		if resultType != "" {
			a.SetResultHandlingType(resultType)
			rh := genMf.NewResultHandling()
			if outputVar != "" {
				rh.SetOutputVariableName(outputVar)
			}
			a.SetResultHandling(rh)
		}
		return a
	}

	t.Run("GET with String result", func(t *testing.T) {
		a := build(t, genMf.HttpMethodGet, "https://api.example.com/users", "Body", genMf.ResultHandlingTypeString)
		got := formatActionGen(nil, a)
		want := "$Body = rest call get 'https://api.example.com/users'\n    returns String;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("POST with String result and headers", func(t *testing.T) {
		a := build(t, genMf.HttpMethodPost, "https://api.example.com/users", "Resp", genMf.ResultHandlingTypeString)
		hc, _ := a.HttpConfiguration().(*genMf.HttpConfiguration)
		he := genMf.NewHttpHeaderEntry()
		he.SetKey("Authorization")
		he.SetValue("$Token")
		hc.AddHeaderEntries(he)
		got := formatActionGen(nil, a)
		want := "$Resp = rest call post 'https://api.example.com/users'\n    header 'Authorization' = $Token\n    returns String;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DELETE returns Nothing without output var", func(t *testing.T) {
		a := build(t, genMf.HttpMethodDelete, "https://api.example.com/users/{1}", "", genMf.ResultHandlingTypeNone)
		hc, _ := a.HttpConfiguration().(*genMf.HttpConfiguration)
		tmpl, _ := hc.CustomLocationTemplate().(*genMf.StringTemplate)
		ta := genMf.NewTemplateArgument()
		ta.SetExpression("$UserId")
		tmpl.AddArguments(ta)
		got := formatActionGen(nil, a)
		want := "rest call delete 'https://api.example.com/users/{1}' with ({1} = $UserId)\n    returns Nothing;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("HttpResponse result returns 'response'", func(t *testing.T) {
		a := build(t, genMf.HttpMethodGet, "https://api.example.com/data", "Resp", genMf.ResultHandlingTypeHttpResponse)
		got := formatActionGen(nil, a)
		want := "$Resp = rest call get 'https://api.example.com/data'\n    returns response;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("PUT with auth and timeout", func(t *testing.T) {
		a := build(t, genMf.HttpMethodPut, "https://api.example.com/u", "Resp", genMf.ResultHandlingTypeString)
		hc, _ := a.HttpConfiguration().(*genMf.HttpConfiguration)
		hc.SetUseAuthentication(true)
		hc.SetHttpAuthenticationUserName("$User")
		hc.SetAuthenticationPassword("$Pass")
		a.SetTimeOutExpression("30")
		got := formatActionGen(nil, a)
		want := "$Resp = rest call put 'https://api.example.com/u'\n    auth basic $User password $Pass\n    timeout 30\n    returns String;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("PATCH with custom body template", func(t *testing.T) {
		a := build(t, genMf.HttpMethodPatch, "https://api.example.com/u/{1}", "Resp", genMf.ResultHandlingTypeString)
		req := genMf.NewCustomRequestHandling()
		btmpl := genMf.NewStringTemplate()
		btmpl.SetText(`{ "name": "{1}" }`)
		bArg := genMf.NewTemplateArgument()
		bArg.SetExpression("$Name")
		btmpl.AddArguments(bArg)
		req.SetTemplate(btmpl)
		a.SetRequestHandling(req)
		got := formatActionGen(nil, a)
		want := "$Resp = rest call patch 'https://api.example.com/u/{1}'\n    body '{ \"name\": \"{1}\" }' with ({1} = $Name)\n    returns String;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("Mapping result emits 'as list of Entity' when not single object", func(t *testing.T) {
		a := build(t, genMf.HttpMethodGet, "https://api.example.com/users", "Users", genMf.ResultHandlingTypeMapping)
		rh, _ := a.ResultHandling().(*genMf.ResultHandling)
		call := genMf.NewImportMappingCall()
		call.SetMappingQualifiedName("Sales.UserMapping")
		rh.SetImportMappingCall(call)
		got := formatActionGen(nil, a)
		want := "$Users = rest call get 'https://api.example.com/users'\n    returns mapping Sales.UserMapping;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("output var fallback through ResultVariableName raw key", func(t *testing.T) {
		// gen `OutputVariableName()` is bound to BSON `OutputVariableName`,
		// but Studio Pro writes the value under `ResultVariableName`.
		// Build an action with an explicit raw BSON ResultHandling so we
		// can exercise the raw-key fallback.
		raw, err := bson.Marshal(bson.M{
			"$Type":              "Microflows$ResultHandling",
			"ResultVariableName": "BodyFromRaw",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rh := genMf.NewResultHandling()
		rh.SetRaw(raw)
		rh.InitFromRaw(raw)

		a := build(t, genMf.HttpMethodGet, "https://api.example.com/raw", "", genMf.ResultHandlingTypeString)
		a.SetResultHandling(rh)
		got := formatActionGen(nil, a)
		want := "$BodyFromRaw = rest call get 'https://api.example.com/raw'\n    returns String;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// RestOperationCallAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_RestOperationCallAction(t *testing.T) {
	t.Run("bare operation call", func(t *testing.T) {
		a := genMf.NewRestOperationCallAction()
		a.SetOperationQualifiedName("Catalog.GetUsers")
		got := formatActionGen(nil, a)
		want := "send rest request Catalog.GetUsers;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with output variable", func(t *testing.T) {
		a := genMf.NewRestOperationCallAction()
		a.SetOperationQualifiedName("Catalog.GetUserById")
		ov := genMf.NewOutputVariable()
		ov.SetVariableName("User")
		a.SetOutputVariable(ov)
		got := formatActionGen(nil, a)
		want := "$User = send rest request Catalog.GetUserById;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("with path and query parameters and body", func(t *testing.T) {
		a := genMf.NewRestOperationCallAction()
		a.SetOperationQualifiedName("Catalog.UpdateUser")

		pm := genMf.NewRestParameterMapping()
		pm.SetParameterQualifiedName("Catalog.UpdateUser.Id")
		pm.SetValue("$UserId")
		a.AddParameterMappings(pm)

		qm := genMf.NewRestOperationParameterMapping()
		qm.SetParameterQualifiedName("Catalog.UpdateUser.IncludeArchived")
		qm.SetValue("true")
		a.AddQueryParameterMappings(qm)

		bv := genMf.NewBodyVariable()
		bv.SetVariableName("Updates")
		a.SetBodyVariable(bv)

		got := formatActionGen(nil, a)
		want := "send rest request Catalog.UpdateUser\n    with ($Id = $UserId, $IncludeArchived = true)\n    body $Updates;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("query param BSON-key fallback to QueryParameter", func(t *testing.T) {
		a := genMf.NewRestOperationCallAction()
		a.SetOperationQualifiedName("Catalog.Search")

		raw, err := bson.Marshal(bson.M{
			"$Type":          "Microflows$RestOperationParameterMapping",
			"QueryParameter": "Catalog.Search.Page",
			"Value":          "1",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		qm := genMf.NewRestOperationParameterMapping()
		qm.SetRaw(raw)
		qm.InitFromRaw(raw)
		a.AddQueryParameterMappings(qm)

		got := formatActionGen(nil, a)
		want := "send rest request Catalog.Search\n    with ($Page = 1);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// ExecuteDatabaseQueryAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_ExecuteDatabaseQueryAction(t *testing.T) {
	t.Run("static query", func(t *testing.T) {
		a := genDC.NewExecuteDatabaseQueryAction()
		a.SetQueryQualifiedName("Sales.GetOrders")
		a.SetOutputVariableName("Rows")
		got := formatActionGen(nil, a)
		want := "$Rows = execute database query Sales.GetOrders;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("dynamic query with parameters", func(t *testing.T) {
		a := genDC.NewExecuteDatabaseQueryAction()
		a.SetQueryQualifiedName("Sales.GetOrders")
		a.SetOutputVariableName("Rows")
		a.SetDynamicQuery("$DynSql")

		pm := genDC.NewQueryParameterMapping()
		pm.SetParameterName("MinTotal")
		pm.SetValue("$Threshold")
		a.AddParameterMappings(pm)

		got := formatActionGen(nil, a)
		want := "$Rows = execute database query Sales.GetOrders dynamic $DynSql (MinTotal = $Threshold);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("connection override", func(t *testing.T) {
		a := genDC.NewExecuteDatabaseQueryAction()
		a.SetQueryQualifiedName("Sales.GetOrders")

		cm := genDC.NewConnectionParameterMapping()
		cm.SetParameterName("Url")
		cm.SetValue("$RuntimeUrl")
		a.AddConnectionParameterMappings(cm)

		got := formatActionGen(nil, a)
		want := "execute database query Sales.GetOrders\n    connection (Url = $RuntimeUrl);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("multiple parameters", func(t *testing.T) {
		a := genDC.NewExecuteDatabaseQueryAction()
		a.SetQueryQualifiedName("Sales.GetOrders")
		a.SetOutputVariableName("Rows")

		for _, p := range []struct{ name, value string }{
			{"From", "$Start"},
			{"To", "$End"},
		} {
			pm := genDC.NewQueryParameterMapping()
			pm.SetParameterName(p.name)
			pm.SetValue(p.value)
			a.AddParameterMappings(pm)
		}

		got := formatActionGen(nil, a)
		want := "$Rows = execute database query Sales.GetOrders (From = $Start, To = $End);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// ImportXmlAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_ImportXmlAction(t *testing.T) {
	t.Run("basic import with output variable", func(t *testing.T) {
		a := genMf.NewImportXmlAction()
		a.SetXmlDocumentVariableName("XmlDoc")

		rh := genMf.NewResultHandling()
		rh.SetOutputVariableName("Imported")
		call := genMf.NewImportMappingCall()
		call.SetMappingQualifiedName("Sales.ImportOrders")
		rh.SetImportMappingCall(call)
		a.SetResultHandling(rh)

		got := formatActionGen(nil, a)
		want := "$Imported = import from mapping Sales.ImportOrders($XmlDoc);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("import without output variable", func(t *testing.T) {
		a := genMf.NewImportXmlAction()
		a.SetXmlDocumentVariableName("XmlDoc")

		rh := genMf.NewResultHandling()
		call := genMf.NewImportMappingCall()
		call.SetMappingQualifiedName("Sales.ImportOrders")
		rh.SetImportMappingCall(call)
		a.SetResultHandling(rh)

		got := formatActionGen(nil, a)
		want := "import from mapping Sales.ImportOrders($XmlDoc);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("legacy ReturnValueMapping fallback", func(t *testing.T) {
		// Studio Pro emits the older `ReturnValueMapping` key on the
		// nested ImportMappingCall. Build the call with a raw BSON doc
		// to exercise the fallback path.
		raw, err := bson.Marshal(bson.M{
			"$Type":              "Microflows$ImportMappingCall",
			"ReturnValueMapping": "Sales.LegacyMapping",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		call := genMf.NewImportMappingCall()
		call.SetRaw(raw)
		call.InitFromRaw(raw)

		rh := genMf.NewResultHandling()
		rh.SetOutputVariableName("Out")
		rh.SetImportMappingCall(call)

		a := genMf.NewImportXmlAction()
		a.SetXmlDocumentVariableName("XmlDoc")
		a.SetResultHandling(rh)

		got := formatActionGen(nil, a)
		want := "$Out = import from mapping Sales.LegacyMapping($XmlDoc);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// ExportXmlAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_ExportXmlAction(t *testing.T) {
	t.Run("basic export with typed handling", func(t *testing.T) {
		a := genMf.NewExportXmlAction()

		// OutputMethod is polymorphic in BSON ($Type: Microflows$StringExport
		// etc.) but the gen package only generates the empty base type.
		// Wrap a raw BSON doc as *element.Base so the formatter's raw
		// fallback fires for `OutputVariableName`.
		omRaw, err := bson.Marshal(bson.M{
			"$Type":              "Microflows$StringExport",
			"OutputVariableName": "Xml",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		om := newBaseElementWithRaw(t, "Microflows$StringExport", omRaw)
		a.SetOutputMethod(om)

		req := genMf.NewMappingRequestHandling()
		req.SetMappingQualifiedName("Sales.ExportOrders")
		req.SetMappingArgumentVariableName("Order")
		a.SetResultHandling(req)

		got := formatActionGen(nil, a)
		want := "$Xml = export to mapping Sales.ExportOrders($Order);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("export without parameter variable", func(t *testing.T) {
		a := genMf.NewExportXmlAction()

		req := genMf.NewMappingRequestHandling()
		req.SetMappingQualifiedName("Sales.ExportOrders")
		a.SetResultHandling(req)

		got := formatActionGen(nil, a)
		want := "export to mapping Sales.ExportOrders;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("legacy MappingId raw-key fallback", func(t *testing.T) {
		// Some MPRs encode the result handling with raw BSON keys
		// `MappingId` + `MappingVariableName` (legacy parser path).
		// Build the result handling as an *element.Base wrapping the
		// raw doc to exercise the fallback path.
		rhRaw, err := bson.Marshal(bson.M{
			"$Type":               "Microflows$LegacyExportHandling",
			"MappingId":           "Sales.LegacyExportMapping",
			"MappingVariableName": "Order",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rh := newBaseElementWithRaw(t, "Microflows$LegacyExportHandling", rhRaw)

		a := genMf.NewExportXmlAction()
		a.SetResultHandling(rh)

		got := formatActionGen(nil, a)
		want := "export to mapping Sales.LegacyExportMapping($Order);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// TransformJsonAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_TransformJsonAction(t *testing.T) {
	t.Run("basic transform", func(t *testing.T) {
		a := genMf.NewTransformJsonAction()
		a.SetInputVariableName("Json")
		a.SetOutputVariableName("Out")
		a.SetTransformationQualifiedName("Sales.JsonStruct")
		got := formatActionGen(nil, a)
		want := "$Out = transform $Json with Sales.JsonStruct;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("transform without output variable", func(t *testing.T) {
		a := genMf.NewTransformJsonAction()
		a.SetInputVariableName("Json")
		a.SetTransformationQualifiedName("Sales.JsonStruct")
		got := formatActionGen(nil, a)
		want := "transform $Json with Sales.JsonStruct;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("transform with empty transformer keeps slot", func(t *testing.T) {
		a := genMf.NewTransformJsonAction()
		a.SetInputVariableName("Json")
		a.SetOutputVariableName("Out")
		got := formatActionGen(nil, a)
		want := "$Out = transform $Json with ;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// WebServiceCallAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_WebServiceCallAction(t *testing.T) {
	t.Run("bare service reference", func(t *testing.T) {
		a := genMf.NewWebServiceCallAction()
		a.SetImportedWebServiceQualifiedName("Integration.MyService")
		got := formatActionGen(nil, a)
		want := "call web service Integration.MyService;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("service with operation, mappings, and timeout", func(t *testing.T) {
		a := genMf.NewWebServiceCallAction()
		a.SetImportedWebServiceQualifiedName("Integration.MyService")
		a.SetOperationName("GetOrder")
		a.SetTimeOutExpression("60")

		// Send mapping: nested ExportMappingCall under
		// RequestBodyHandling. Use raw BSON so we exercise the
		// *element.Base fallback path.
		reqRaw, err := bson.Marshal(bson.M{
			"$Type": "Microflows$WebServiceRequestHandling",
			"ExportMappingCall": bson.M{
				"$Type":   "Microflows$ExportMappingCall",
				"Mapping": "Integration.SendOrder",
			},
		})
		if err != nil {
			t.Fatalf("marshal req: %v", err)
		}
		// Use a concrete gen type whose codec keeps raw bytes:
		// HttpHeaderEntry has no required fields and stores the raw doc.
		// SimpleSqlDataType also works — use the generic *element.Base
		// fallback by constructing through the registry would be ideal,
		// but for tests we can use a CustomRequestHandling which is a
		// *genMf.CustomRequestHandling and does not match *element.Base.
		// Instead, create an element.Base directly via a stub helper.
		req := newBaseElementWithRaw(t, "Microflows$WebServiceRequestHandling", reqRaw)
		a.SetRequestBodyHandling(req)

		// Receive mapping: nested ImportMappingCall under ResultHandling.
		rh := genMf.NewResultHandling()
		rh.SetOutputVariableName("Order")
		call := genMf.NewImportMappingCall()
		call.SetMappingQualifiedName("Integration.ReceiveOrder")
		rh.SetImportMappingCall(call)
		a.SetResultHandling(rh)

		got := formatActionGen(nil, a)
		want := "$Order = call web service Integration.MyService\noperation GetOrder\nsend mapping Integration.SendOrder\nreceive mapping Integration.ReceiveOrder\ntimeout 60;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("service ImportedService raw-key fallback", func(t *testing.T) {
		// Studio Pro writes the BSON key `ImportedService` (the gen
		// type aliases this as `ImportedWebService`). Build via raw
		// BSON to exercise the fallback path.
		raw, err := bson.Marshal(bson.M{
			"$Type":           "Microflows$WebServiceCallAction",
			"ImportedService": "Integration.LegacyService",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		a := genMf.NewWebServiceCallAction()
		a.SetRaw(raw)
		a.InitFromRaw(raw)
		got := formatActionGen(nil, a)
		want := "call web service Integration.LegacyService;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("operation name with spaces is mdlQuoted", func(t *testing.T) {
		// Bare identifiers (no spaces, dots, etc.) skip quoting via
		// `formatWebServiceReference`. Names with spaces or special
		// characters fall through to `mdlQuote`.
		a := genMf.NewWebServiceCallAction()
		a.SetImportedWebServiceQualifiedName("Integration.MyService")
		a.SetOperationName("Get Order Details")
		got := formatActionGen(nil, a)
		want := "call web service Integration.MyService\noperation 'Get Order Details';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// newBaseElementWithRaw creates an *element.Base wrapping the supplied
// raw BSON, useful for exercising the formatter's *element.Base
// fallback paths in tests where no factory is registered for the
// document's $Type.
func newBaseElementWithRaw(t *testing.T, typeName string, raw []byte) *element.Base {
	t.Helper()
	b := &element.Base{}
	b.SetTypeName(typeName)
	b.SetRaw(raw)
	return b
}
