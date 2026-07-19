// SPDX-License-Identifier: Apache-2.0

package golden

import (
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// goldenUUIDs maps element keys to their Studio Pro-generated UUIDs.
// Keys are human-readable labels matching the builder structure.
var goldenUUIDs = map[string]string{
	// Nanoflow root
	"Nanoflow":                   "9315486c-edde-473c-879e-f8f447ff1417",
	"MicroflowReturnType":        "6839e276-e4b5-4cbb-b67a-fc9bb1309914",
	"ObjectCollection":           "c5150bd0-74f5-407e-9768-e58d3fb2b0c9",
	// Objects
	"StartEvent":                  "ec75e6dc-1676-44ac-a7f7-8c84c349ee28",
	"EndEvent":                    "068637c1-8521-49aa-955d-06efd4a98fd4",
	"SyncAll":                     "bd665d06-c493-4096-b2fe-fbc544f76884",
	"SyncAll.Action":              "edd61c27-b843-41b7-805f-4259ba218c2f",
	"SyncUnsync":                  "c8573512-98b7-4ad7-a864-fe5efe2c10c4",
	"SyncUnsync.Action":           "a3a189d7-50ac-4944-bb3b-690d2287b2de",
	"SyncSpecific":                "26dc2c8c-a806-40c4-b0b6-83de9455adb7",
	"SyncSpecific.Action":         "f8732f47-f9ed-40eb-9ac0-135198455fc1",
	"CreateChange":                "2911414d-6662-463a-93a6-77f5e0e3397a",
	"CreateChange.Action":         "e4557269-b6ce-4dce-84e5-998542ad7fae",
	"AssocRetrieve":               "afe7bfe2-9923-4172-af51-69e9024acdc1",
	"AssocRetrieve.Action":        "4fa3270c-2822-47dc-9734-dd71882bc933",
	"AssocRetrieve.RetrieveSource": "4fa99399-4fea-4e35-97ec-bdc0e97bc98d",
	"DBRetrieve":                  "c2896874-7123-41cf-beee-e027738edf30",
	"DBRetrieve.Action":           "049e4893-9bbf-4222-bc21-5bc9b0c82fd5",
	"DBRetrieve.RetrieveSource":   "24a17543-1b21-4d74-8505-817d6a793700",
	"NewSortings":                 "16bc4879-2d26-4413-911f-65a75a22d761",
	"SortItem":                    "a451a804-39be-480b-855d-848a26de295d",
	"AttributeRef":                "2959f97a-c529-4f76-b88a-de13447e7b6d",
	"Range":                       "4174b329-88e0-47fa-a0de-ae8233518e1e",
	// Flows
	"Flow0": "6f045ead-3643-4422-842b-cf21fd9a0700",
	"Flow0.Line": "9fbbc835-0dcd-47c0-9464-079492cd6c03",
	"Flow1": "1fcf57cf-1d19-478e-979e-4d8464e7b074",
	"Flow1.Line": "826ec3a1-fc40-43dc-9a97-d3ff8e8cdc4b",
	"Flow2": "bae2bbae-9450-4163-989f-23a307c76abe",
	"Flow2.Line": "a550e06c-cc1f-41a6-a17f-644034630969",
	"Flow3": "b47073e4-b72d-4cce-b926-12eef3636367",
	"Flow3.Line": "56fe576f-810b-41d6-b0a7-4a8bf3f54309",
	"Flow4": "34b2cb22-fe7c-4e7e-aeaa-7e49ce2dd4a8",
	"Flow4.Line": "3f6e3060-088b-406a-9055-f2f33b9069f9",
	"Flow5": "0a2d24cc-5ca3-44a8-9ea9-f0f440f5442c",
	"Flow5.Line": "b3011821-1b1d-4f9e-b32e-ca7ba47f6f43",
	"Flow6": "2e1d40ae-7240-4cb2-8137-552d2433602d",
	"Flow6.Line": "8c7eaffd-09cb-413a-8bb1-7e36832e6ed8",
}

type idSetter interface{ SetID(element.ID) }

func setID(e idSetter, key string) {
	uid, ok := goldenUUIDs[key]
	if !ok {
		panic("golden UUID not found for key: " + key)
	}
	e.SetID(element.ID(uid))
}

func setIDOpt(e idSetter, key string) {
	if uid, ok := goldenUUIDs[key]; ok {
		e.SetID(element.ID(uid))
	}
}

// BuildNanoflow constructs a Nanoflow element tree exactly matching
// testdata/MyFirstModule.Nanoflow.mxunit — byte-for-byte.
func BuildNanoflow() element.Element {
	nf := genMf.NewNanoflow()
	setID(nf, "Nanoflow")
	nf.SetName("Nanoflow")
	nf.SetDocumentation("")
	nf.SetExcluded(false)
	nf.SetExportLevel("Hidden")
	nf.SetMarkAsUsed(false)
	nf.SetReturnVariableName("")
	nf.SetUseListParameterByReference(true)

	// MicroflowReturnType
	mrt := genDt.NewListType()
	setID(mrt, "MicroflowReturnType")
	mrt.SetEntityQualifiedName("Administration.Account")
	nf.SetMicroflowReturnType(mrt)

	// ObjectCollection
	oc := genMf.NewMicroflowObjectCollection()
	setID(oc, "ObjectCollection")
	nf.SetObjectCollection(oc)
	populateObjects(oc)

	// Flows
	addFlows(nf)

	return nf
}

func addElement(oc *genMf.MicroflowObjectCollection, e element.Element) {
	oc.AddObjects(e)
}

func populateObjects(oc *genMf.MicroflowObjectCollection) {
	// StartEvent
	se := genMf.NewStartEvent()
	setID(se, "StartEvent")
	se.SetRelativeMiddlePoint("-202;200")
	se.SetSize("20;20")
	addElement(oc, se)

	// EndEvent
	ee := genMf.NewEndEvent()
	setID(ee, "EndEvent")
	ee.SetDocumentation("")
	ee.SetRelativeMiddlePoint("1028;200")
	ee.SetReturnValue("$AccountList")
	ee.SetSize("20;20")
	addElement(oc, ee)

	// Sync (All)
	addSyncAction(oc, "SyncAll", "All", "", "128;200")
	// Sync (Unsynchronized)
	addSyncAction(oc, "SyncUnsync", "Unsynchronized", "", "318;200")
	// Sync (Specific, with "NewAccount")
	addSyncAction(oc, "SyncSpecific", "Specific", "NewAccount", "508;200")

	// CreateChangeAction — CreateObjectAction with golden's exact field values
	ccAct := genMf.NewActionActivity()
	setID(ccAct, "CreateChange")
	ccAct.SetDisabled(false)
	ccAct.SetCaption("Activity")
	ccAct.SetBackgroundColor("Default")
	ccAct.SetDocumentation("")
	ccAct.SetAutoGenerateCaption(true)
	ccAct.SetRelativeMiddlePoint("-62;200")
	ccAct.SetSize("120;60")

	ca := genMf.NewCreateObjectAction()
	setID(ca, "CreateChange.Action")
	ca.SetCommit("No")
	ca.SetEntityQualifiedName("Administration.Account")
	ca.SetErrorHandlingType("Abort")
	// Items: empty PartList [marker=2]
	// (CreateObjectAction has Items inherited from ChangeAction base)
	ca.SetRefreshInClient(true)
	ca.SetOutputVariableName("NewAccount")
	ccAct.SetAction(ca)
	addElement(oc, ccAct)

	// AssociationRetrieveAction
	arAct := genMf.NewActionActivity()
	setID(arAct, "AssocRetrieve")
	arAct.SetDisabled(false)
	arAct.SetCaption("Activity")
	arAct.SetBackgroundColor("Default")
	arAct.SetDocumentation("")
	arAct.SetAutoGenerateCaption(true)
	arAct.SetRelativeMiddlePoint("698;200")
	arAct.SetSize("120;60")

	ar := genMf.NewRetrieveAction()
	setID(ar, "AssocRetrieve.Action")
	ar.SetErrorHandlingType("Abort")
	ar.SetOutputVariableName("AccountPasswordDataList")
	ars := genMf.NewAssociationRetrieveSource()
	setID(ars, "AssocRetrieve.RetrieveSource")
	ars.SetAssociationQualifiedName("Administration.AccountPasswordData_Account")
	ars.SetStartVariableName("NewAccount")
	ar.SetRetrieveSource(ars)
	arAct.SetAction(ar)
	addElement(oc, arAct)

	// DatabaseRetrieveAction (with sort)
	dbAct := genMf.NewActionActivity()
	setID(dbAct, "DBRetrieve")
	dbAct.SetDisabled(false)
	dbAct.SetCaption("Activity")
	dbAct.SetBackgroundColor("Default")
	dbAct.SetDocumentation("")
	dbAct.SetAutoGenerateCaption(true)
	dbAct.SetRelativeMiddlePoint("888;200")
	dbAct.SetSize("120;60")

	db := genMf.NewRetrieveAction()
	setID(db, "DBRetrieve.Action")
	db.SetErrorHandlingType("Abort")
	db.SetOutputVariableName("AccountList")

	dbrs := genMf.NewDatabaseRetrieveSource()
	setID(dbrs, "DBRetrieve.RetrieveSource")
	dbrs.SetEntityQualifiedName("Administration.Account")

	// NewSortings
	nsl := genMf.NewSortItemList()
	setID(nsl, "NewSortings")
	si := genMf.NewSortItem()
	setID(si, "SortItem")
	si.SetSortOrder("Descending")
	ar2 := genDm.NewAttributeRef()
	setID(ar2, "AttributeRef")
	ar2.SetAttributeQualifiedName("Administration.Account.FullName")
	si.SetAttributeRef(ar2)
	nsl.AddItems(si)
	dbrs.SetSortItemList(nsl)

	// Range: ConstantRange{SingleObject: false}
	cr := genMf.NewConstantRange()
	setID(cr, "Range")
	cr.SetSingleObject(false)
	dbrs.SetRange(cr)

	dbrs.SetXPathConstraint("[\n  (\n    FullName = empty\n    and System.owner = '[%CurrentUser%]'\n  )\n]")
	db.SetRetrieveSource(dbrs)
	dbAct.SetAction(db)
	addElement(oc, dbAct)
}

func addSyncAction(oc *genMf.MicroflowObjectCollection, key, syncType, variable, pos string) {
	act := genMf.NewActionActivity()
	setID(act, key)
	act.SetDisabled(false)
	act.SetCaption("Activity")
	act.SetBackgroundColor("Default")
	act.SetDocumentation("")
	act.SetAutoGenerateCaption(true)
	act.SetRelativeMiddlePoint(pos)
	act.SetSize("120;60")

	sa := genMf.NewSynchronizeAction()
	setID(sa, key+".Action")
	sa.SetErrorHandlingType("Abort")
	sa.SetType(syncType)
	sa.SetVariableNames(variable)
	act.SetAction(sa)
	addElement(oc, act)
}

func addFlows(nf *genMf.Nanoflow) {
	// Objects in ObjectCollection by index:
	//   [0] marker=3
	//   [1] StartEvent
	//   [2] EndEvent
	//   [3] SyncAll
	//   [4] SyncUnsync
	//   [5] SyncSpecific
	//   [6] CreateChange
	//   [7] AssocRetrieve
	//   [8] DBRetrieve
	//
	// Golden flows connect:
	//   StartEvent(1) → CreateChange(6)    (Flow0)
	//   CreateChange(6) → SyncAll(3)       (Flow1)
	//   SyncAll(3) → SyncUnsync(4)        (Flow2)
	//   SyncUnsync(4) → SyncSpecific(5)   (Flow3)
	//   SyncSpecific(5) → AssocRetrieve(7)(Flow4)
	//   AssocRetrieve(7) → DBRetrieve(8)  (Flow5)
	//   DBRetrieve(8) → EndEvent(2)       (Flow6)

	type flowDef struct {
		key            string
		originIdx, destIdx int
		oVec, dVec     string
	}
	// objItems is 0-indexed without the PartList marker:
	//   [0] StartEvent, [1] EndEvent, [2] SyncAll,
	//   [3] SyncUnsync, [4] SyncSpecific, [5] CreateChange,
	//   [6] AssocRetrieve, [7] DBRetrieve
	//
	// Golden flow graph (matching UUID-based verification):
	//   Start → CreateChange (15;0, -30;0)
	//   SyncAll → SyncUnsync (30;0, -30;0)
	//   SyncUnsync → SyncSpecific (30;0, -30;0)
	//   SyncSpecific → AssocRetrieve (30;0, -30;0)
	//   CreateChange → SyncAll (0;0, 0;0)     ← rejoin
	//   AssocRetrieve → DBRetrieve (30;0, -30;0)
	//   DBRetrieve → EndEvent (0;0, 0;0)
	flows := []flowDef{
		{"Flow0", 0, 5, "15;0", "-30;0"},
		{"Flow1", 2, 3, "30;0", "-30;0"},
		{"Flow2", 3, 4, "30;0", "-30;0"},
		{"Flow3", 4, 6, "30;0", "-30;0"},
		{"Flow4", 5, 2, "0;0", "0;0"},          // CreateChange → SyncAll rejoin
		{"Flow5", 6, 7, "30;0", "-30;0"},
		{"Flow6", 7, 1, "0;0", "0;0"},
	}

	ocEl := nf.ObjectCollection()
	if ocEl == nil {
		return
	}
	objects, ok := ocEl.(*genMf.MicroflowObjectCollection)
	if !ok {
		return
	}
	objItems := objects.ObjectsItems()

	for _, fd := range flows {
		if fd.originIdx >= len(objItems) || fd.destIdx >= len(objItems) {
			continue
		}
		flow := genMf.NewSequenceFlow()
		setIDOpt(flow, fd.key)
		flow.SetOriginID(objItems[fd.originIdx].ID())
		flow.SetDestinationID(objItems[fd.destIdx].ID())
		flow.SetOriginConnectionIndex(int32(1))
		flow.SetDestinationConnectionIndex(int32(3))
		flow.SetIsErrorHandler(false)

		curve := genMf.NewBezierCurve()
		setIDOpt(curve, fd.key+".Line")
		curve.SetOriginControlVector(fd.oVec)
		curve.SetDestinationControlVector(fd.dVec)
		flow.SetLine(curve)

		// CaseValues must be empty [marker=2] — no NoCase element.
		nf.AddFlows(flow)
	}
}
