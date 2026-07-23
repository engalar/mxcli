// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.2.e tests for the gen-typed Variable / Expression / Data
// family.
//
// Direct-construction tests cover every branch of every formatter,
// including the legacy quirks (CastAction's raw-BSON ObjectVariable,
// CreateVariableAction's polymorphic VariableType, RetrieveAction's
// reverse-association detection and ConstantRange-as-CustomRange
// fallback). Fixture-driven integration tests at the bottom exercise
// the action surfaces present in testdata/expr-checker/minimal.mpr
// (retrieve, validation feedback, change variable XPath form).

package executor

import (
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genTx "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// ────────────────────────────────────────────────────────
// CastAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_CastAction(t *testing.T) {
	t.Run("both vars present", func(t *testing.T) {
		raw, err := bson.Marshal(bson.M{
			"$Type":              "Microflows$CastAction",
			"OutputVariableName": "Cast",
			"ObjectVariableName": "Account",
		})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		a := genMf.NewCastAction()
		a.SetRaw(raw)
		a.InitFromRaw(raw)

		got := formatActionGen(nil, a)
		want := "$Cast = cast $Account;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("only object var (cast in-place)", func(t *testing.T) {
		raw, err := bson.Marshal(bson.M{
			"$Type":              "Microflows$CastAction",
			"ObjectVariableName": "Account",
		})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		a := genMf.NewCastAction()
		a.SetRaw(raw)
		a.InitFromRaw(raw)

		got := formatActionGen(nil, a)
		want := "cast $Account;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("only output var via VariableName fallback", func(t *testing.T) {
		raw, err := bson.Marshal(bson.M{
			"$Type":        "Microflows$CastAction",
			"VariableName": "Legacy",
		})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		a := genMf.NewCastAction()
		a.SetRaw(raw)
		a.InitFromRaw(raw)

		got := formatActionGen(nil, a)
		want := "cast $Legacy;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("both empty (defensive: no $-prefix added)", func(t *testing.T) {
		a := genMf.NewCastAction()
		got := formatActionGen(nil, a)
		want := "cast ;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// CreateVariableAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_CreateVariableAction(t *testing.T) {
	t.Run("string with literal initial value", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateVariableAction").(*genMf.CreateVariableAction)
		a.SetVariableName("Greeting")
		a.SetVariableType(genDT.NewStringType())
		a.SetInitialValue("'hello'")

		got := formatActionGen(nil, a)
		want := "declare $Greeting String = 'hello';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("integer with empty initial value defaults to empty", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateVariableAction").(*genMf.CreateVariableAction)
		a.SetVariableName("Count")
		a.SetVariableType(genDT.NewIntegerType())

		got := formatActionGen(nil, a)
		want := "declare $Count Integer = empty;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("trailing newline in initial value is stripped before empty check", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateVariableAction").(*genMf.CreateVariableAction)
		a.SetVariableName("Total")
		a.SetVariableType(genDT.NewDecimalType())
		a.SetInitialValue("$Other + 1\n")

		got := formatActionGen(nil, a)
		want := "declare $Total Decimal = $Other + 1;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("ObjectType with EntityQualifiedName", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateVariableAction").(*genMf.CreateVariableAction)
		a.SetVariableName("Customer")
		obj := genDT.NewObjectType()
		obj.SetEntityQualifiedName("Sales.Customer")
		a.SetVariableType(obj)

		got := formatActionGen(nil, a)
		want := "declare $Customer Sales.Customer = empty;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("ListType with EntityQualifiedName", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateVariableAction").(*genMf.CreateVariableAction)
		a.SetVariableName("Customers")
		lst := genDT.NewListType()
		lst.SetEntityQualifiedName("Sales.Customer")
		a.SetVariableType(lst)

		got := formatActionGen(nil, a)
		want := "declare $Customers List of Sales.Customer = empty;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("EnumerationType renders with enum prefix", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateVariableAction").(*genMf.CreateVariableAction)
		a.SetVariableName("Status")
		en := genDT.NewEnumerationType()
		en.SetEnumerationQualifiedName("Sales.OrderStatus")
		a.SetVariableType(en)

		got := formatActionGen(nil, a)
		want := "declare $Status enum Sales.OrderStatus = empty;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("nil VariableType falls back to Object", func(t *testing.T) {
		a := newGenAction(t, "Microflows$CreateVariableAction").(*genMf.CreateVariableAction)
		a.SetVariableName("X")

		got := formatActionGen(nil, a)
		want := "declare $X Unknown = empty;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// formatGenDataType direct tests — fills in the small primitives that
// the CreateVariableAction tests don't otherwise reach.
func TestFormatGenDataType(t *testing.T) {
	cases := []struct {
		name string
		in   element.Element
		want string
	}{
		{"Boolean", genDT.NewBooleanType(), "Boolean"},
		{"DateTime", genDT.NewDateTimeType(), "DateTime"},
		{"Binary", genDT.NewBinaryType(), "Binary"},
		{"Float", genDT.NewFloatType(), "Float"},
		{"Void", genDT.NewVoidType(), "Void"},
		{"nil", nil, "Unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatGenDataType(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("ObjectType without QN falls back to Object", func(t *testing.T) {
		if got := formatGenDataType(genDT.NewObjectType()); got != "Object" {
			t.Errorf("got %q, want Object", got)
		}
	})
	t.Run("ListType without QN falls back to List", func(t *testing.T) {
		if got := formatGenDataType(genDT.NewListType()); got != "List" {
			t.Errorf("got %q, want List", got)
		}
	})
	t.Run("EnumerationType without QN falls back to Enumeration", func(t *testing.T) {
		if got := formatGenDataType(genDT.NewEnumerationType()); got != "Enumeration" {
			t.Errorf("got %q, want Enumeration", got)
		}
	})
}

// ────────────────────────────────────────────────────────
// ChangeVariableAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_ChangeVariableAction(t *testing.T) {
	t.Run("plain variable name", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeVariableAction").(*genMf.ChangeVariableAction)
		a.SetChangeVariableName("Total")
		a.SetValue("$Total + 1")

		got := formatActionGen(nil, a)
		want := "set $Total = $Total + 1;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("variable name already prefixed with $", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeVariableAction").(*genMf.ChangeVariableAction)
		a.SetChangeVariableName("$IsValid")
		a.SetValue("false")

		got := formatActionGen(nil, a)
		want := "set $IsValid = false;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("XPath attribute change form", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeVariableAction").(*genMf.ChangeVariableAction)
		a.SetChangeVariableName("$Product/Price")
		a.SetValue("100")

		got := formatActionGen(nil, a)
		want := "change $Product (Price = 100);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("XPath form with association navigation", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ChangeVariableAction").(*genMf.ChangeVariableAction)
		a.SetChangeVariableName("Order/Sales.Order_Customer/Name")
		a.SetValue("'ACME'")

		got := formatActionGen(nil, a)
		want := "change $Order (Name = 'ACME');"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// RetrieveAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_RetrieveAction(t *testing.T) {
	t.Run("nil source falls back to placeholder", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Customers")

		got := formatActionGen(nil, a)
		if got != "" {
			t.Errorf("nil source: got %q, want empty string (caller emits TODO)", got)
		}
	})

	t.Run("missing output variable defaults to Result but nil source returns empty", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)

		got := formatActionGen(nil, a)
		if got != "" {
			t.Errorf("nil source: got %q, want empty string", got)
		}
	})

	t.Run("AssociationRetrieveSource bare", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Account")
		src := newGenAction(t, "Microflows$AssociationRetrieveSource").(*genMf.AssociationRetrieveSource)
		src.SetStartVariableName("AccountPasswordData")
		src.SetAssociationQualifiedName("Administration.AccountPasswordData_Account")
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $Account from $AccountPasswordData/Administration.AccountPasswordData_Account;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("AssociationRetrieveSource missing start var defaults to Object", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("X")
		src := newGenAction(t, "Microflows$AssociationRetrieveSource").(*genMf.AssociationRetrieveSource)
		src.SetAssociationQualifiedName("M.A_B")
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $X from $Object/M.A_B;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource entity only (no constraint, no range)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Orders")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $Orders from Sales.Order;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource with single-predicate XPath (brackets stripped)", func(t *testing.T) {
		// Legacy parity: single-predicate XPath has its outer
		// brackets stripped when rendered into the `where` clause —
		// see `cmd_microflows_format_action.go` lines 411-430. Only
		// multi-predicate XPath keeps each `[…]` wrapper.
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Open")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")
		src.SetXPathConstraint("[Status = 'Open']")
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $Open from Sales.Order\n    where Status = 'Open';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource with un-bracketed XPath passed through", func(t *testing.T) {
		// Defensive case: legacy only strips brackets when both ends
		// are present. A malformed constraint without brackets is
		// emitted verbatim.
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("X")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")
		src.SetXPathConstraint("Status = 'Open'")
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $X from Sales.Order\n    where Status = 'Open';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource splits multi-predicate XPath", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Hits")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")
		src.SetXPathConstraint("[Status = 'Open'][Total > 100]")
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $Hits from Sales.Order\n    where [Status = 'Open']\n    [Total > 100];"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource with newline-separated XPath predicates", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Hits")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")
		src.SetXPathConstraint("[Status = 'Open']\n[Total > 100]")
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $Hits from Sales.Order\n    where [Status = 'Open']\n    [Total > 100];"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource with sort by", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Sorted")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")

		list := newGenAction(t, "Microflows$SortItemList").(*genMf.SortItemList)
		si1 := newGenAction(t, "Microflows$SortItem").(*genMf.SortItem)
		si1.SetAttributePath("Sales.Order.Total")
		si1.SetSortOrder(genMf.SortOrderEnumDescending)
		list.AddItems(si1)
		si2 := newGenAction(t, "Microflows$SortItem").(*genMf.SortItem)
		si2.SetAttributePath("Sales.Order.Created")
		// SortOrder left at default (treated as ascending).
		list.AddItems(si2)
		src.SetSortItemList(list)
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $Sorted from Sales.Order\n    sort by Sales.Order.Total desc, Sales.Order.Created asc;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource with first range (limit 1)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("First")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")
		r := newGenAction(t, "Microflows$ConstantRange").(*genMf.ConstantRange)
		r.SetSingleObject(true)
		src.SetRange(r)
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $First from Sales.Order\n    limit 1;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource with custom range", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Page")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")
		r := newGenAction(t, "Microflows$CustomRange").(*genMf.CustomRange)
		r.SetLimitExpression("20")
		r.SetOffsetExpression("40")
		src.SetRange(r)
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $Page from Sales.Order\n    limit 20\n    offset 40;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("ConstantRange with limit/offset (legacy custom-range fallback)", func(t *testing.T) {
		// ConstantRange in BSON can carry LimitExpression/OffsetExpression
		// alongside SingleObject — Studio Pro's older shape. The gen
		// type doesn't expose these, so the formatter reads them from
		// raw BSON.
		raw, err := bson.Marshal(bson.M{
			"$Type":            "Microflows$ConstantRange",
			"SingleObject":     false,
			"LimitExpression":  "5",
			"OffsetExpression": "10",
		})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		r := genMf.NewConstantRange()
		r.SetRaw(raw)
		r.InitFromRaw(raw)

		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("Page")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		src.SetEntityQualifiedName("Sales.Order")
		src.SetRange(r)
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $Page from Sales.Order\n    limit 5\n    offset 10;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("DatabaseRetrieveSource with empty entity qualified name", func(t *testing.T) {
		a := newGenAction(t, "Microflows$RetrieveAction").(*genMf.RetrieveAction)
		a.SetOutputVariableName("X")
		src := newGenAction(t, "Microflows$DatabaseRetrieveSource").(*genMf.DatabaseRetrieveSource)
		// EntityQualifiedName intentionally left blank.
		a.SetRetrieveSource(src)

		got := formatActionGen(nil, a)
		want := "retrieve $X from Entity;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// LogMessageAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_LogMessageAction(t *testing.T) {
	t.Run("bare log defaults: info level, Application node, Message body", func(t *testing.T) {
		a := newGenAction(t, "Microflows$LogMessageAction").(*genMf.LogMessageAction)

		got := formatActionGen(nil, a)
		want := "log info node 'Application' 'Message';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("level lower-cased; node expression preserved", func(t *testing.T) {
		a := newGenAction(t, "Microflows$LogMessageAction").(*genMf.LogMessageAction)
		a.SetLevel(genMf.LogLevelError)
		a.SetNode("getKey(M.Nodes.X)")
		tmpl := newGenAction(t, "Microflows$StringTemplate").(*genMf.StringTemplate)
		tmpl.SetText("Bad happened")
		a.SetMessageTemplate(tmpl)

		got := formatActionGen(nil, a)
		want := "log error node getKey(M.Nodes.X) 'Bad happened';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("template parameters render as positional placeholders", func(t *testing.T) {
		a := newGenAction(t, "Microflows$LogMessageAction").(*genMf.LogMessageAction)
		a.SetLevel(genMf.LogLevelDebug)
		tmpl := newGenAction(t, "Microflows$StringTemplate").(*genMf.StringTemplate)
		tmpl.SetText("Got {1} for user {2}")
		ta1 := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
		ta1.SetExpression("$Order/Total")
		tmpl.AddArguments(ta1)
		ta2 := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
		ta2.SetExpression("$User/Name")
		tmpl.AddArguments(ta2)
		a.SetMessageTemplate(tmpl)

		got := formatActionGen(nil, a)
		want := "log debug node 'Application' 'Got {1} for user {2}' with ({1} = $Order/Total, {2} = $User/Name);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty arguments are skipped (gaps don't shift indices)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$LogMessageAction").(*genMf.LogMessageAction)
		tmpl := newGenAction(t, "Microflows$StringTemplate").(*genMf.StringTemplate)
		tmpl.SetText("only one")
		ta1 := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
		ta1.SetExpression("$X")
		tmpl.AddArguments(ta1)
		// Second argument has no expression — must be dropped.
		taEmpty := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
		tmpl.AddArguments(taEmpty)
		ta2 := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
		ta2.SetExpression("$Y")
		tmpl.AddArguments(ta2)
		a.SetMessageTemplate(tmpl)

		got := formatActionGen(nil, a)
		want := "log info node 'Application' 'only one' with ({1} = $X, {2} = $Y);"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// DownloadFileAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_DownloadFileAction(t *testing.T) {
	t.Run("bare download (no $ prefix in BSON)", func(t *testing.T) {
		a := newGenAction(t, "Microflows$DownloadFileAction").(*genMf.DownloadFileAction)
		a.SetFileDocumentVariableName("File")

		got := formatActionGen(nil, a)
		want := "download file $File;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("show in browser appended", func(t *testing.T) {
		a := newGenAction(t, "Microflows$DownloadFileAction").(*genMf.DownloadFileAction)
		a.SetFileDocumentVariableName("File")
		a.SetShowFileInBrowser(true)

		got := formatActionGen(nil, a)
		want := "download file $File show in browser;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("variable already $-prefixed not double-prefixed", func(t *testing.T) {
		a := newGenAction(t, "Microflows$DownloadFileAction").(*genMf.DownloadFileAction)
		a.SetFileDocumentVariableName("$File")

		got := formatActionGen(nil, a)
		want := "download file $File;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("empty variable name renders bare placeholder", func(t *testing.T) {
		a := newGenAction(t, "Microflows$DownloadFileAction").(*genMf.DownloadFileAction)

		got := formatActionGen(nil, a)
		want := "download file ;"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// ValidationFeedbackAction
// ────────────────────────────────────────────────────────

func TestFormatActionGen_ValidationFeedbackAction(t *testing.T) {
	t.Run("attribute path: last segment appended via slash", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ValidationFeedbackAction").(*genMf.ValidationFeedbackAction)
		a.SetObjectVariableName("Feedback")
		a.SetAttributeQualifiedName("FeedbackModule.Feedback.Subject")
		// FeedbackTemplate left nil — defaults to '...'.

		got := formatActionGen(nil, a)
		want := "validation feedback $Feedback/Subject message '...';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("association path: full QN appended via slash", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ValidationFeedbackAction").(*genMf.ValidationFeedbackAction)
		a.SetObjectVariableName("Order")
		a.SetAssociationQualifiedName("Sales.Order_Customer")

		got := formatActionGen(nil, a)
		want := "validation feedback $Order/Sales.Order_Customer message '...';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("attribute QN with fewer than 3 dots: no slash appended", func(t *testing.T) {
		// Defensive case: legacy guard `len(parts) >= 3` means a malformed
		// qualified name like "Subject" leaves the surface as just $Var.
		a := newGenAction(t, "Microflows$ValidationFeedbackAction").(*genMf.ValidationFeedbackAction)
		a.SetObjectVariableName("Feedback")
		a.SetAttributeQualifiedName("Subject")

		got := formatActionGen(nil, a)
		want := "validation feedback $Feedback message '...';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("FeedbackTemplate with en_US translation", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ValidationFeedbackAction").(*genMf.ValidationFeedbackAction)
		a.SetObjectVariableName("Feedback")
		a.SetAttributeQualifiedName("FeedbackModule.Feedback.Subject")

		tmpl := newGenAction(t, "Microflows$TextTemplate").(*genMf.TextTemplate)
		text := newGenAction(t, "Texts$Text").(*genTx.Text)
		tr := newGenAction(t, "Texts$Translation").(*genTx.Translation)
		tr.SetLanguageCode("en_US")
		tr.SetText("Subject is required")
		text.AddTranslations(tr)
		tmpl.SetText(text)
		a.SetFeedbackTemplate(tmpl)

		got := formatActionGen(nil, a)
		want := "validation feedback $Feedback/Subject message 'Subject is required';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("FeedbackTemplate falls back to first non-en translation", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ValidationFeedbackAction").(*genMf.ValidationFeedbackAction)
		a.SetObjectVariableName("Feedback")
		a.SetAttributeQualifiedName("FeedbackModule.Feedback.Subject")

		tmpl := newGenAction(t, "Microflows$TextTemplate").(*genMf.TextTemplate)
		text := newGenAction(t, "Texts$Text").(*genTx.Text)
		tr := newGenAction(t, "Texts$Translation").(*genTx.Translation)
		tr.SetLanguageCode("nl_NL")
		tr.SetText("Onderwerp is verplicht")
		text.AddTranslations(tr)
		tmpl.SetText(text)
		a.SetFeedbackTemplate(tmpl)

		got := formatActionGen(nil, a)
		want := "validation feedback $Feedback/Subject message 'Onderwerp is verplicht';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("object variable already $-prefixed not double-prefixed", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ValidationFeedbackAction").(*genMf.ValidationFeedbackAction)
		a.SetObjectVariableName("$Feedback")
		a.SetAttributeQualifiedName("M.E.Subject")

		got := formatActionGen(nil, a)
		want := "validation feedback $Feedback/Subject message '...';"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("template args: single objects clause emitted", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ValidationFeedbackAction").(*genMf.ValidationFeedbackAction)
		a.SetObjectVariableName("Ticket")
		a.SetAttributeQualifiedName("HD.Ticket.Subject")

		tmpl := newGenAction(t, "Microflows$TextTemplate").(*genMf.TextTemplate)
		text := newGenAction(t, "Texts$Text").(*genTx.Text)
		tr := newGenAction(t, "Texts$Translation").(*genTx.Translation)
		tr.SetLanguageCode("en_US")
		tr.SetText("Value must be at least {1} characters.")
		text.AddTranslations(tr)
		tmpl.SetText(text)
		arg := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
		arg.SetExpression("3")
		tmpl.AddArguments(arg)
		a.SetFeedbackTemplate(tmpl)

		got := formatActionGen(nil, a)
		want := "validation feedback $Ticket/Subject message 'Value must be at least {1} characters.' objects [3];"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("template args: multiple objects clause emitted", func(t *testing.T) {
		a := newGenAction(t, "Microflows$ValidationFeedbackAction").(*genMf.ValidationFeedbackAction)
		a.SetObjectVariableName("Ticket")
		a.SetAttributeQualifiedName("HD.Ticket.Subject")

		tmpl := newGenAction(t, "Microflows$TextTemplate").(*genMf.TextTemplate)
		text := newGenAction(t, "Texts$Text").(*genTx.Text)
		tr := newGenAction(t, "Texts$Translation").(*genTx.Translation)
		tr.SetLanguageCode("en_US")
		tr.SetText("{1} is not unique in {2}.")
		text.AddTranslations(tr)
		tmpl.SetText(text)
		arg1 := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
		arg1.SetExpression("$Ticket/Subject")
		tmpl.AddArguments(arg1)
		arg2 := newGenAction(t, "Microflows$TemplateArgument").(*genMf.TemplateArgument)
		arg2.SetExpression("$ModuleName")
		tmpl.AddArguments(arg2)
		a.SetFeedbackTemplate(tmpl)

		got := formatActionGen(nil, a)
		want := "validation feedback $Ticket/Subject message '{1} is not unique in {2}.' objects [$Ticket/Subject, $ModuleName];"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

// ────────────────────────────────────────────────────────
// Fixture-driven integration tests
// ────────────────────────────────────────────────────────

// TestDescribeMicroflowGenToString_DataFamily_Fixture exercises the
// end-to-end gen-typed renderer for the Variable / Expression / Data
// family on testdata/expr-checker/minimal.mpr. The fixture's data
// family surface is:
//
//   - Administration.ChangePassword: AssociationRetrieveSource form
//     (`retrieve $Account from $AccountPasswordData/...`).
//   - FeedbackModule.PopulateUserAttributes: DatabaseRetrieveSource
//     with an XPath constraint and limit-1 range.
//   - FeedbackModule.VAL_Feedback: ValidationFeedbackAction with both
//     attribute (last-segment) and full template-text rendering, plus
//     ChangeVariableAction for the `$ValidFeedback = false` guards.
//   - FeedbackModule.SUB_Feedback_PostToAppInsights: LogMessageAction
//     inside the rest-call error handler block + CreateVariableAction
//     (`declare $ServerLocation String = '...'`).
func TestDescribeMicroflowGenToString_DataFamily_Fixture(t *testing.T) {
	w := openMprWriterForTest(t)
	cases := []struct {
		qn   string
		want []string
	}{
		{
			"Administration.ChangePassword",
			[]string{
				"retrieve $Account from $AccountPasswordData/Administration.AccountPasswordData_Account;",
			},
		},
		{
			"FeedbackModule.PopulateUserAttributes",
			[]string{
				"retrieve $CurrentUser from System.User",
				"where id = $currentUser",
				"limit 1;",
			},
		},
		{
			"FeedbackModule.VAL_Feedback",
			[]string{
				"validation feedback $Feedback/Subject message 'Subject length cannot be longer than 200 characters\\n';",
				"validation feedback $Feedback/Subject message 'Subject is required';",
				"set $ValidFeedback = false;",
			},
		},
		{
			"FeedbackModule.SUB_Feedback_PostToAppInsights",
			[]string{
				"declare $ServerLocation String = 'https://feedback-api.mendix.com/v2/feedback-items';",
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
