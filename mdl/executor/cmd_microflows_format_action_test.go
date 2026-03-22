// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/microflows"
)

// =============================================================================
// formatAction — CRUD actions
// =============================================================================

func TestFormatAction_CreateObject_Simple(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CreateObjectAction{
		EntityQualifiedName: "MyModule.Customer",
		OutputVariable:      "NewCustomer",
	}
	got := e.formatAction(action, nil, nil)
	if got != "$NewCustomer = CREATE MyModule.Customer;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_CreateObject_WithMembers(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CreateObjectAction{
		EntityQualifiedName: "MyModule.Customer",
		OutputVariable:      "NewCustomer",
		InitialMembers: []*microflows.MemberChange{
			{AttributeQualifiedName: "MyModule.Customer.Name", Value: "'John'"},
			{AttributeQualifiedName: "MyModule.Customer.Age", Value: "25"},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "$NewCustomer = CREATE MyModule.Customer (Name = 'John', Age = 25);"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_CreateObject_WithAssociationMember(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CreateObjectAction{
		EntityQualifiedName: "MyModule.Order",
		OutputVariable:      "NewOrder",
		InitialMembers: []*microflows.MemberChange{
			{AssociationQualifiedName: "MyModule.Order_Customer", Value: "$Customer"},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "$NewOrder = CREATE MyModule.Order (Order_Customer = $Customer);"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_CreateObject_FallbackEntityID(t *testing.T) {
	e := newTestExecutor()
	entityNames := map[model.ID]string{mkID("e1"): "MyModule.Product"}
	action := &microflows.CreateObjectAction{
		EntityID:       mkID("e1"),
		OutputVariable: "NewProduct",
	}
	got := e.formatAction(action, entityNames, nil)
	if got != "$NewProduct = CREATE MyModule.Product;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeObject(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeObjectAction{
		ChangeVariable: "Customer",
		Changes: []*microflows.MemberChange{
			{AttributeQualifiedName: "MyModule.Customer.Name", Value: "'Jane'"},
		},
	}
	got := e.formatAction(action, nil, nil)
	if got != "CHANGE $Customer (Name = 'Jane');" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeObject_NoChanges(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeObjectAction{ChangeVariable: "Obj"}
	got := e.formatAction(action, nil, nil)
	if got != "CHANGE $Obj;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_DeleteObject(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.DeleteObjectAction{DeleteVariable: "Customer"}
	got := e.formatAction(action, nil, nil)
	if got != "DELETE $Customer;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_CommitObjects_WithEvents(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CommitObjectsAction{CommitVariable: "Order", WithEvents: true}
	got := e.formatAction(action, nil, nil)
	if got != "COMMIT $Order WITH EVENTS;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_CommitObjects_WithoutEvents(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CommitObjectsAction{CommitVariable: "Order"}
	got := e.formatAction(action, nil, nil)
	if got != "COMMIT $Order;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_CommitObjects_Refresh(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CommitObjectsAction{CommitVariable: "Order", RefreshInClient: true}
	got := e.formatAction(action, nil, nil)
	if got != "COMMIT $Order REFRESH;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_CommitObjects_WithEventsAndRefresh(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CommitObjectsAction{CommitVariable: "Order", WithEvents: true, RefreshInClient: true}
	got := e.formatAction(action, nil, nil)
	if got != "COMMIT $Order WITH EVENTS REFRESH;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_RollbackObject(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.RollbackObjectAction{RollbackVariable: "Order"}
	got := e.formatAction(action, nil, nil)
	if got != "ROLLBACK $Order;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_RollbackObject_Refresh(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.RollbackObjectAction{RollbackVariable: "Order", RefreshInClient: true}
	got := e.formatAction(action, nil, nil)
	if got != "ROLLBACK $Order REFRESH;" {
		t.Errorf("got %q", got)
	}
}

// =============================================================================
// formatAction — Variable actions
// =============================================================================

func TestFormatAction_CreateVariable(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CreateVariableAction{
		VariableName: "Counter",
		DataType:     &microflows.IntegerType{},
		InitialValue: "0",
	}
	got := e.formatAction(action, nil, nil)
	if got != "DECLARE $Counter Integer = 0;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_CreateVariable_NoInitial(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CreateVariableAction{
		VariableName: "Name",
		DataType:     &microflows.StringType{},
	}
	got := e.formatAction(action, nil, nil)
	if got != "DECLARE $Name String = empty;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeVariable_Simple(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeVariableAction{
		VariableName: "Counter",
		Value:        "$Counter + 1",
	}
	got := e.formatAction(action, nil, nil)
	if got != "SET $Counter = $Counter + 1;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeVariable_WithDollarPrefix(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeVariableAction{
		VariableName: "$Counter",
		Value:        "42",
	}
	got := e.formatAction(action, nil, nil)
	if got != "SET $Counter = 42;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeVariable_XPath(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeVariableAction{
		VariableName: "$Product/Price",
		Value:        "9.99",
	}
	got := e.formatAction(action, nil, nil)
	if got != "CHANGE $Product (Price = 9.99);" {
		t.Errorf("got %q", got)
	}
}

// =============================================================================
// formatAction — List actions
// =============================================================================

func TestFormatAction_CreateList(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CreateListAction{
		EntityQualifiedName: "MyModule.Order",
		OutputVariable:      "OrderList",
	}
	got := e.formatAction(action, nil, nil)
	if got != "$OrderList = CREATE LIST of MyModule.Order;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeList_Add(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeListAction{
		ChangeVariable: "OrderList",
		Type:           microflows.ChangeListTypeAdd,
		Value:          "$NewOrder",
	}
	got := e.formatAction(action, nil, nil)
	if got != "ADD $NewOrder TO $OrderList;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeList_Remove(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeListAction{
		ChangeVariable: "OrderList",
		Type:           microflows.ChangeListTypeRemove,
		Value:          "$OldOrder",
	}
	got := e.formatAction(action, nil, nil)
	if got != "REMOVE $OldOrder FROM $OrderList;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeList_Clear(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeListAction{
		ChangeVariable: "OrderList",
		Type:           microflows.ChangeListTypeClear,
	}
	got := e.formatAction(action, nil, nil)
	if got != "CLEAR $OrderList;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ChangeList_Set(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ChangeListAction{
		ChangeVariable: "OrderList",
		Type:           microflows.ChangeListTypeSet,
		Value:          "$OtherList",
	}
	got := e.formatAction(action, nil, nil)
	if got != "SET $OrderList = $OtherList;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_AggregateList_Count(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.AggregateListAction{
		InputVariable:  "Orders",
		OutputVariable: "Total",
		Function:       microflows.AggregateFunctionCount,
	}
	got := e.formatAction(action, nil, nil)
	if got != "$Total = COUNT($Orders);" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_AggregateList_Sum(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.AggregateListAction{
		InputVariable:          "Orders",
		OutputVariable:         "TotalAmount",
		Function:               microflows.AggregateFunctionSum,
		AttributeQualifiedName: "MyModule.Order.Amount",
	}
	got := e.formatAction(action, nil, nil)
	if got != "$TotalAmount = SUM($Orders.Amount);" {
		t.Errorf("got %q", got)
	}
}

// =============================================================================
// formatAction — Call actions
// =============================================================================

func TestFormatAction_MicroflowCall_WithResult(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.MicroflowCallAction{
		ResultVariableName: "Result",
		MicroflowCall: &microflows.MicroflowCall{
			Microflow: "MyModule.ProcessOrder",
			ParameterMappings: []*microflows.MicroflowCallParameterMapping{
				{Parameter: "MyModule.ProcessOrder.Order", Argument: "$Order"},
			},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "$Result = CALL MICROFLOW MyModule.ProcessOrder(Order = $Order);"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_MicroflowCall_NoResult(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.MicroflowCallAction{
		MicroflowCall: &microflows.MicroflowCall{
			Microflow: "MyModule.DoSomething",
		},
	}
	got := e.formatAction(action, nil, nil)
	if got != "CALL MICROFLOW MyModule.DoSomething();" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_JavaActionCall(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.JavaActionCallAction{
		JavaAction:         "MyModule.SendEmail",
		ResultVariableName: "Success",
		ParameterMappings: []*microflows.JavaActionParameterMapping{
			{
				Parameter: "MyModule.SendEmail.To",
				Value: &microflows.ExpressionBasedCodeActionParameterValue{
					Expression: "$Customer/Email",
				},
			},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "$Success = CALL JAVA ACTION MyModule.SendEmail(To = $Customer/Email);"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_CallExternal(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.CallExternalAction{
		ConsumedODataService: "MyModule.OrderService",
		Name:                 "GetOrders",
		ResultVariableName:   "Orders",
	}
	got := e.formatAction(action, nil, nil)
	want := "$Orders = CALL EXTERNAL ACTION MyModule.OrderService.GetOrders();"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// =============================================================================
// formatAction — UI actions
// =============================================================================

func TestFormatAction_ShowPage(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ShowPageAction{
		PageName: "MyModule.CustomerEdit",
	}
	got := e.formatAction(action, nil, nil)
	if got != "SHOW PAGE MyModule.CustomerEdit;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ShowPage_WithParams(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ShowPageAction{
		PageName: "MyModule.OrderDetail",
		PageParameterMappings: []*microflows.PageParameterMapping{
			{Parameter: "MyModule.OrderDetail.Order", Argument: "$Order"},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "SHOW PAGE MyModule.OrderDetail($Order = $Order);"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_ClosePage(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ClosePageAction{NumberOfPages: 1}
	got := e.formatAction(action, nil, nil)
	if got != "CLOSE PAGE;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ClosePage_Multiple(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ClosePageAction{NumberOfPages: 3}
	got := e.formatAction(action, nil, nil)
	if got != "CLOSE PAGE 3;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ShowMessage(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ShowMessageAction{
		Type: microflows.MessageTypeWarning,
		Template: &model.Text{
			Translations: map[string]string{"en_US": "Order saved"},
		},
	}
	got := e.formatAction(action, nil, nil)
	if got != "SHOW MESSAGE 'Order saved' TYPE Warning;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ShowMessage_EscapesQuotes(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ShowMessageAction{
		Type: microflows.MessageTypeInformation,
		Template: &model.Text{
			Translations: map[string]string{"en_US": "It's done"},
		},
	}
	got := e.formatAction(action, nil, nil)
	if got != "SHOW MESSAGE 'It''s done' TYPE Information;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_ValidationFeedback(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.ValidationFeedbackAction{
		ObjectVariable: "Customer",
		AttributeName:  "MyModule.Customer.Email",
		Template: &model.Text{
			Translations: map[string]string{"en_US": "Email is required"},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "VALIDATION FEEDBACK $Customer/Email MESSAGE 'Email is required';"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_LogMessage(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.LogMessageAction{
		LogLevel:    microflows.LogLevelWarning,
		LogNodeName: "'OrderService'",
		MessageTemplate: &model.Text{
			Translations: map[string]string{"en_US": "Processing order"},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "LOG WARNING NODE 'OrderService' 'Processing order';"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_LogMessage_WithTemplateParams(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.LogMessageAction{
		LogLevel:    microflows.LogLevelInfo,
		LogNodeName: "'App'",
		MessageTemplate: &model.Text{
			Translations: map[string]string{"en_US": "Order {1} for {2}"},
		},
		TemplateParameters: []string{"$OrderNumber", "$CustomerName"},
	}
	got := e.formatAction(action, nil, nil)
	want := "LOG INFO NODE 'App' 'Order {1} for {2}' WITH ({1} = $OrderNumber, {2} = $CustomerName);"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_UnknownAction(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.UnknownAction{TypeName: "SomeNewAction"}
	got := e.formatAction(action, nil, nil)
	if got != "-- Unsupported action type: SomeNewAction" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_Nil(t *testing.T) {
	e := newTestExecutor()
	got := e.formatAction(nil, nil, nil)
	if got != "-- Empty action" {
		t.Errorf("got %q", got)
	}
}

// =============================================================================
// formatAction — Retrieve actions
// =============================================================================

func TestFormatAction_Retrieve_Database(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.RetrieveAction{
		OutputVariable: "Customers",
		Source: &microflows.DatabaseRetrieveSource{
			EntityQualifiedName: "MyModule.Customer",
		},
	}
	got := e.formatAction(action, nil, nil)
	if got != "RETRIEVE $Customers FROM MyModule.Customer;" {
		t.Errorf("got %q", got)
	}
}

func TestFormatAction_Retrieve_WithXPath(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.RetrieveAction{
		OutputVariable: "ActiveCustomers",
		Source: &microflows.DatabaseRetrieveSource{
			EntityQualifiedName: "MyModule.Customer",
			XPathConstraint:     "[IsActive = true()]",
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "RETRIEVE $ActiveCustomers FROM MyModule.Customer\n    WHERE IsActive = true();"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_Retrieve_WithLimit(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.RetrieveAction{
		OutputVariable: "First",
		Source: &microflows.DatabaseRetrieveSource{
			EntityQualifiedName: "MyModule.Customer",
			Range:               &microflows.Range{RangeType: microflows.RangeTypeFirst},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "RETRIEVE $First FROM MyModule.Customer\n    LIMIT 1;"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_Retrieve_WithSorting(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.RetrieveAction{
		OutputVariable: "Sorted",
		Source: &microflows.DatabaseRetrieveSource{
			EntityQualifiedName: "MyModule.Customer",
			Sorting: []*microflows.SortItem{
				{AttributeQualifiedName: "MyModule.Customer.Name", Direction: microflows.SortDirectionAscending},
				{AttributeQualifiedName: "MyModule.Customer.Age", Direction: microflows.SortDirectionDescending},
			},
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "RETRIEVE $Sorted FROM MyModule.Customer\n    SORT BY MyModule.Customer.Name ASC, MyModule.Customer.Age DESC;"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatAction_Retrieve_Association(t *testing.T) {
	e := newTestExecutor()
	action := &microflows.RetrieveAction{
		OutputVariable: "Address",
		Source: &microflows.AssociationRetrieveSource{
			StartVariable:            "Customer",
			AssociationQualifiedName: "MyModule.Customer_Address",
		},
	}
	got := e.formatAction(action, nil, nil)
	want := "RETRIEVE $Address FROM $Customer/MyModule.Customer_Address;"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
