// SPDX-License-Identifier: Apache-2.0

package golden

import (
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// idSetter is an element that supports setting its UUID.
type idSetter interface{ SetID(element.ID) }

func assignFreshID(e idSetter) {
	e.SetID(element.ID(mpr.GenerateID()))
}

// BuildNanoflow constructs a Nanoflow element tree that matches
// testdata/MyFirstModule.Nanoflow.mxunit.
func BuildNanoflow() element.Element {
	nf := genMf.NewNanoflow()
	assignFreshID(nf)
	nf.SetName("Nanoflow")
	nf.SetDocumentation("")
	nf.SetExcluded(false)
	nf.SetExportLevel("Hidden")
	nf.SetMarkAsUsed(false)
	nf.SetReturnVariableName("")
	nf.SetUseListParameterByReference(true)

	nf.SetMicroflowReturnType(buildListType("Administration.Account"))

	oc := genMf.NewMicroflowObjectCollection()
	assignFreshID(oc)
	nf.SetObjectCollection(oc)
	populateObjects(oc)

	addFlows(nf)

	return nf
}

func buildListType(entityQN string) *genDt.ListType {
	lt := genDt.NewListType()
	assignFreshID(lt)
	lt.SetEntityQualifiedName(entityQN)
	return lt
}

func populateObjects(oc *genMf.MicroflowObjectCollection) {
	addStartEvent(oc)
	addEndEvent(oc)
	addSynchronizeAction(oc, "All", "")
	addSynchronizeAction(oc, "Unsynchronized", "")
	addSynchronizeAction(oc, "Specific", "NewAccount")
	addCreateChangeAction(oc)
	addAssociationRetrieveAction(oc)
	addDatabaseRetrieveAction(oc)
}

func addStartEvent(oc *genMf.MicroflowObjectCollection) {
	e := genMf.NewStartEvent()
	assignFreshID(e)
	e.SetRelativeMiddlePoint("-202;200")
	e.SetSize("20;20")
	oc.AddObjects(e)
}

func addEndEvent(oc *genMf.MicroflowObjectCollection) {
	e := genMf.NewEndEvent()
	assignFreshID(e)
	e.SetDocumentation("")
	e.SetRelativeMiddlePoint("1028;200")
	e.SetReturnValue("$AccountList")
	e.SetSize("20;20")
	oc.AddObjects(e)
}

func addSynchronizeAction(oc *genMf.MicroflowObjectCollection, syncType, variable string) {
	act := wrapAction(genMf.NewActionActivity(), func(a *genMf.ActionActivity) {
		sa := genMf.NewSynchronizeAction()
		assignFreshID(sa)
		sa.SetErrorHandlingType("Abort")
		sa.SetType(syncType)
		sa.SetVariableNames(variable)
		a.SetAction(sa)
	})
	switch syncType {
	case "All":
		act.SetRelativeMiddlePoint("128;200")
	case "Unsynchronized":
		act.SetRelativeMiddlePoint("318;200")
	case "Specific":
		act.SetRelativeMiddlePoint("508;200")
	}
	act.SetSize("120;60")
	oc.AddObjects(act)
}

func addCreateChangeAction(oc *genMf.MicroflowObjectCollection) {
	act := wrapAction(genMf.NewActionActivity(), func(a *genMf.ActionActivity) {
		ca := genMf.NewCreateObjectAction()
		assignFreshID(ca)
		ca.SetCommit("No")
		ca.SetEntityQualifiedName("Administration.Account")
		ca.SetErrorHandlingType("Abort")
		ca.SetRefreshInClient(true)
		ca.SetOutputVariableName("NewAccount")
		a.SetAction(ca)
	})
	act.SetRelativeMiddlePoint("-62;200")
	act.SetSize("120;60")
	oc.AddObjects(act)
}

func addAssociationRetrieveAction(oc *genMf.MicroflowObjectCollection) {
	act := wrapAction(genMf.NewActionActivity(), func(a *genMf.ActionActivity) {
		ra := genMf.NewRetrieveAction()
		assignFreshID(ra)
		ra.SetErrorHandlingType("Abort")
		ra.SetOutputVariableName("AccountPasswordDataList")
		ar := genMf.NewAssociationRetrieveSource()
		assignFreshID(ar)
		ar.SetAssociationQualifiedName("Administration.AccountPasswordData_Account")
		ar.SetStartVariableName("NewAccount")
		ra.SetRetrieveSource(ar)
		a.SetAction(ra)
	})
	act.SetRelativeMiddlePoint("698;200")
	act.SetSize("120;60")
	oc.AddObjects(act)
}

func addDatabaseRetrieveAction(oc *genMf.MicroflowObjectCollection) {
	act := wrapAction(genMf.NewActionActivity(), func(a *genMf.ActionActivity) {
		ra := genMf.NewRetrieveAction()
		assignFreshID(ra)
		ra.SetErrorHandlingType("Abort")
		ra.SetOutputVariableName("AccountList")
		rs := genMf.NewDatabaseRetrieveSource()
		assignFreshID(rs)
		rs.SetEntityQualifiedName("Administration.Account")
		rs.SetSortItemList(buildSortItemList())
		rs.SetRange(buildConstantRange(false))
		rs.SetXPathConstraint("[\n  (\n    FullName = empty\n    and System.owner = '[%CurrentUser%]'\n  )\n]")
		ra.SetRetrieveSource(rs)
		a.SetAction(ra)
	})
	act.SetRelativeMiddlePoint("888;200")
	act.SetSize("120;60")
	oc.AddObjects(act)
}

func buildSortItemList() *genMf.SortItemList {
	sl := genMf.NewSortItemList()
	assignFreshID(sl)
	si := genMf.NewSortItem()
	assignFreshID(si)
	si.SetSortOrder("Descending")
	ar := genDm.NewAttributeRef()
	assignFreshID(ar)
	ar.SetAttributeQualifiedName("Administration.Account.FullName")
	si.SetAttributeRef(ar)
	sl.AddItems(si)
	return sl
}

func buildConstantRange(singleObject bool) *genMf.ConstantRange {
	cr := genMf.NewConstantRange()
	assignFreshID(cr)
	cr.SetSingleObject(singleObject)
	return cr
}

func wrapAction(activity *genMf.ActionActivity, fn func(*genMf.ActionActivity)) *genMf.ActionActivity {
	assignFreshID(activity)
	activity.SetDisabled(false)
	activity.SetCaption("Activity")
	activity.SetBackgroundColor("Default")
	activity.SetDocumentation("")
	activity.SetAutoGenerateCaption(true)
	fn(activity)
	return activity
}

func addFlows(nf *genMf.Nanoflow) {
	ocEl := nf.ObjectCollection()
	if ocEl == nil {
		return
	}
	objects, ok := ocEl.(*genMf.MicroflowObjectCollection)
	if !ok {
		return
	}
	objItems := objects.ObjectsItems()

	// Build flows matching the golden's flow graph.
	// The golden has 8 SequenceFlows with specific origin/destination
	// pairs and bezier control vectors.

	// Define flow connections: [originIdx, destIdx, originVec, destVec]
	type flowDef struct {
		origin int
		dest   int
		oVec   string
		dVec   string
	}
	flows := []flowDef{
		{0, 5, "15;0", "-30;0"},  // StartEvent → CreateChange
		{5, 1, "30;0", "-30;0"},  // CreateChange → Synchronize(All)
		{1, 2, "30;0", "-30;0"},  // Synchronize(All) → Synchronize(Unsynchronized)
		{2, 3, "30;0", "-30;0"},  // Synchronize(Unsynchronized) → Synchronize(Specific)
		{3, 6, "30;0", "-30;0"},  // Synchronize(Specific) → AssocRetrieve
		{6, 7, "30;0", "-30;0"},  // AssocRetrieve → DBRetrieve
		{7, 4, "0;0", "0;0"},     // DBRetrieve → EndEvent
	}

	// Map object creation order indices to golden object positions.
	// Our creation order: StartEvent(0), EndEvent(4), SyncAll(1), SyncUnsync(2), SyncSpecific(3), CreateChange(5), AssocRetrieve(6), DBRetrieve(7)
	// Golden object indices: StartEvent(0), EndEvent(4), SyncAll(1), SyncUnsync(2), SyncSpecific(3), CreateChange(5), AssocRetrieve(6), DBRetrieve(7)
	// Map from our creation order index → golden index (which matches our order exactly)

	for _, fd := range flows {
		if fd.origin >= len(objItems) || fd.dest >= len(objItems) {
			continue
		}
		flow := genMf.NewSequenceFlow()
		assignFreshID(flow)
		flow.SetOriginID(objItems[fd.origin].ID())
		flow.SetDestinationID(objItems[fd.dest].ID())
		flow.SetOriginConnectionIndex(int32(1))
		flow.SetDestinationConnectionIndex(int32(3))
		flow.SetIsErrorHandler(false)

		curve := genMf.NewBezierCurve()
		assignFreshID(curve)
		curve.SetOriginControlVector(fd.oVec)
		curve.SetDestinationControlVector(fd.dVec)
		flow.SetLine(curve)

		nc := genMf.NewNoCase()
		assignFreshID(nc)
		flow.AddCaseValues(nc)
		nf.AddFlows(flow)
	}
}
