// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/microflows"

	"go.mongodb.org/mongo-driver/bson"
)

// CreateMicroflow creates a new microflow.
func (w *Writer) CreateMicroflow(mf *microflows.Microflow) error {
	if mf.ID == "" {
		mf.ID = model.ID(generateUUID())
	}
	mf.TypeName = "Microflows$Microflow"

	contents, err := w.serializeMicroflow(mf)
	if err != nil {
		return fmt.Errorf("failed to serialize microflow: %w", err)
	}

	return w.insertUnit(string(mf.ID), string(mf.ContainerID), "Documents", "Microflows$Microflow", contents)
}

// UpdateMicroflow updates an existing microflow.
func (w *Writer) UpdateMicroflow(mf *microflows.Microflow) error {
	contents, err := w.serializeMicroflow(mf)
	if err != nil {
		return fmt.Errorf("failed to serialize microflow: %w", err)
	}

	return w.updateUnit(string(mf.ID), contents)
}

// DeleteMicroflow deletes a microflow.
func (w *Writer) DeleteMicroflow(id model.ID) error {
	return w.deleteUnit(string(id))
}

// MoveMicroflow moves a microflow to a new container (folder or module).
// Only updates the ContainerID in the database, preserving all BSON content
// (layout positions, flow connections, etc.) as-is.
func (w *Writer) MoveMicroflow(mf *microflows.Microflow) error {
	return w.moveUnitByID(string(mf.ID), string(mf.ContainerID))
}

// CreateNanoflow creates a new nanoflow.
func (w *Writer) CreateNanoflow(nf *microflows.Nanoflow) error {
	if nf.ID == "" {
		nf.ID = model.ID(generateUUID())
	}
	nf.TypeName = "Microflows$Nanoflow"

	contents, err := w.serializeNanoflow(nf)
	if err != nil {
		return fmt.Errorf("failed to serialize nanoflow: %w", err)
	}

	return w.insertUnit(string(nf.ID), string(nf.ContainerID), "Documents", "Microflows$Nanoflow", contents)
}

// UpdateNanoflow updates an existing nanoflow.
func (w *Writer) UpdateNanoflow(nf *microflows.Nanoflow) error {
	contents, err := w.serializeNanoflow(nf)
	if err != nil {
		return fmt.Errorf("failed to serialize nanoflow: %w", err)
	}

	return w.updateUnit(string(nf.ID), contents)
}

// DeleteNanoflow deletes a nanoflow.
func (w *Writer) DeleteNanoflow(id model.ID) error {
	return w.deleteUnit(string(id))
}

// MoveNanoflow moves a nanoflow to a new container (folder or module).
// Only updates the ContainerID in the database, preserving all BSON content as-is.
func (w *Writer) MoveNanoflow(nf *microflows.Nanoflow) error {
	return w.moveUnitByID(string(nf.ID), string(nf.ContainerID))
}

func (w *Writer) serializeMicroflow(mf *microflows.Microflow) ([]byte, error) {
	// Build main document with required fields in correct order
	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(mf.ID))},
		{Key: "$Type", Value: "Microflows$Microflow"},
		{Key: "AllowConcurrentExecution", Value: mf.AllowConcurrentExecution},
		{Key: "AllowedModuleRoles", Value: allowedModuleRolesArray(mf.AllowedModuleRoles)},
		{Key: "ApplyEntityAccess", Value: false},
		{Key: "ConcurrencyErrorMicroflow", Value: ""},
		{Key: "ConcurrenyErrorMessage", Value: bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Texts$Text"},
			{Key: "Items", Value: bson.A{int32(3)}}, // Empty array marker
		}},
		{Key: "Documentation", Value: mf.Documentation},
		{Key: "Excluded", Value: mf.Excluded},
		{Key: "ExportLevel", Value: "Hidden"},
	}

	// Add Flows array (SequenceFlows and AnnotationFlows go here, not in ObjectCollection)
	flows := bson.A{int32(3)} // Start with array type marker
	if mf.ObjectCollection != nil {
		for _, flow := range mf.ObjectCollection.Flows {
			flows = append(flows, serializeSequenceFlow(flow))
		}
		for _, af := range mf.ObjectCollection.AnnotationFlows {
			flows = append(flows, serializeAnnotationFlow(af))
		}
	}
	doc = append(doc, bson.E{Key: "Flows", Value: flows})

	// Add remaining fields
	doc = append(doc, bson.E{Key: "MarkAsUsed", Value: mf.MarkAsUsed})
	doc = append(doc, bson.E{Key: "MicroflowActionInfo", Value: nil})

	// Note: Parameters are NOT stored in MicroflowParameterCollection
	// They go in ObjectCollection.Objects as Microflows$MicroflowParameter entries

	// Add return type
	if mf.ReturnType != nil {
		doc = append(doc, bson.E{Key: "MicroflowReturnType", Value: serializeMicroflowDataType(mf.ReturnType)})
	}

	doc = append(doc, bson.E{Key: "Name", Value: mf.Name})

	// Add object collection (without flows - they're in Flows array)
	// Parameters go in ObjectCollection.Objects, pass them here
	if mf.ObjectCollection != nil {
		doc = append(doc, bson.E{Key: "ObjectCollection", Value: serializeMicroflowObjectCollectionWithoutFlows(mf.ObjectCollection, mf.Parameters)})
	}

	// Add remaining optional fields
	returnVarName := mf.ReturnVariableName
	if returnVarName == "" {
		returnVarName = "Variable" // Default value used by Mendix
	}
	doc = append(doc, bson.E{Key: "ReturnVariableName", Value: returnVarName})
	doc = append(doc, bson.E{Key: "StableId", Value: idToBsonBinary(generateUUID())})
	doc = append(doc, bson.E{Key: "Url", Value: ""})
	doc = append(doc, bson.E{Key: "UrlSearchParameters", Value: bson.A{int32(1)}})
	doc = append(doc, bson.E{Key: "WorkflowActionInfo", Value: nil})

	return bson.Marshal(doc)
}

// serializeSequenceFlow serializes a SequenceFlow to BSON with correct structure.
func serializeSequenceFlow(flow *microflows.SequenceFlow) bson.D {
	// Serialize CaseValues
	caseValues := bson.A{int32(2)} // Default empty array marker
	if flow.CaseValue != nil {
		switch cv := flow.CaseValue.(type) {
		case microflows.EnumerationCase:
			caseValues = bson.A{
				int32(2),
				bson.D{
					{Key: "$ID", Value: idToBsonBinary(string(cv.ID))},
					{Key: "$Type", Value: "Microflows$EnumerationCase"},
					{Key: "Value", Value: cv.Value},
				},
			}
		case *microflows.EnumerationCase:
			caseValues = bson.A{
				int32(2),
				bson.D{
					{Key: "$ID", Value: idToBsonBinary(string(cv.ID))},
					{Key: "$Type", Value: "Microflows$EnumerationCase"},
					{Key: "Value", Value: cv.Value},
				},
			}
		}
	}

	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(flow.ID))},
		{Key: "$Type", Value: "Microflows$SequenceFlow"},
		{Key: "CaseValues", Value: caseValues},
		{Key: "DestinationConnectionIndex", Value: int32(flow.DestinationConnectionIndex)},
		{Key: "DestinationPointer", Value: idToBsonBinary(string(flow.DestinationID))},
		{Key: "IsErrorHandler", Value: flow.IsErrorHandler},
		{Key: "Line", Value: bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Microflows$BezierCurve"},
			{Key: "DestinationControlVector", Value: "0;0"},
			{Key: "OriginControlVector", Value: "0;0"},
		}},
		{Key: "OriginConnectionIndex", Value: int32(flow.OriginConnectionIndex)},
		{Key: "OriginPointer", Value: idToBsonBinary(string(flow.OriginID))},
	}
}

// serializeAnnotationFlow serializes an AnnotationFlow to BSON.
func serializeAnnotationFlow(af *microflows.AnnotationFlow) bson.D {
	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(af.ID))},
		{Key: "$Type", Value: "Microflows$AnnotationFlow"},
		{Key: "DestinationConnectionIndex", Value: int32(0)},
		{Key: "DestinationPointer", Value: idToBsonBinary(string(af.DestinationID))},
		{Key: "Line", Value: bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "Microflows$BezierCurve"},
			{Key: "DestinationControlVector", Value: "0;0"},
			{Key: "OriginControlVector", Value: "0;0"},
		}},
		{Key: "OriginConnectionIndex", Value: int32(0)},
		{Key: "OriginPointer", Value: idToBsonBinary(string(af.OriginID))},
	}
}

// serializeMicroflowParameter serializes a MicroflowParameter to BSON.
// Parameters go in ObjectCollection.Objects, not in a separate collection.
func serializeMicroflowParameter(p *microflows.MicroflowParameter, posX int) bson.D {
	// Calculate position based on index - parameters appear at the top of the microflow
	relativeMiddlePoint := fmt.Sprintf("%d;53", 200+posX*100)

	doc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(p.ID))},
		{Key: "$Type", Value: "Microflows$MicroflowParameter"},
		{Key: "DefaultValue", Value: ""},
		{Key: "Documentation", Value: p.Documentation},
		{Key: "HasVariableNameBeenChanged", Value: true},
		{Key: "IsRequired", Value: true},
		{Key: "Name", Value: p.Name},
		{Key: "RelativeMiddlePoint", Value: relativeMiddlePoint},
		{Key: "Size", Value: "30;30"},
	}
	if p.Type != nil {
		doc = append(doc, bson.E{Key: "VariableType", Value: serializeMicroflowDataType(p.Type)})
	}
	return doc
}

// serializeMicroflowDataType serializes a microflow data type to BSON.
func serializeMicroflowDataType(dt microflows.DataType) bson.D {
	if dt == nil {
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$VoidType"},
		}
	}

	switch t := dt.(type) {
	case *microflows.BooleanType:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$BooleanType"},
		}
	case *microflows.IntegerType:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$IntegerType"},
		}
	case *microflows.LongType:
		// Mendix uses IntegerType for 64-bit integers (Long in Java)
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$IntegerType"},
		}
	case *microflows.DecimalType:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$DecimalType"},
		}
	case *microflows.StringType:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$StringType"},
		}
	case *microflows.DateTimeType:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$DateTimeType"},
		}
	case *microflows.BinaryType:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$BinaryType"},
		}
	case *microflows.VoidType:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$VoidType"},
		}
	case *microflows.ObjectType:
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$ObjectType"},
		}
		// Entity is a BY_NAME_REFERENCE - stored as qualified name string, not binary GUID
		if t.EntityQualifiedName != "" {
			doc = append(doc, bson.E{Key: "Entity", Value: t.EntityQualifiedName})
		}
		return doc
	case *microflows.ListType:
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$ListType"},
		}
		// Entity is a BY_NAME_REFERENCE - stored as qualified name string, not binary GUID
		if t.EntityQualifiedName != "" {
			doc = append(doc, bson.E{Key: "Entity", Value: t.EntityQualifiedName})
		}
		return doc
	case *microflows.EnumerationType:
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$EnumerationType"},
		}
		// Enumeration is a BY_NAME_REFERENCE - stored as qualified name string, not binary GUID
		if t.EnumerationQualifiedName != "" {
			doc = append(doc, bson.E{Key: "Enumeration", Value: t.EnumerationQualifiedName})
		}
		return doc
	default:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(generateUUID())},
			{Key: "$Type", Value: "DataTypes$VoidType"},
		}
	}
}

// serializeMicroflowObjectCollectionWithoutFlows serializes the object collection to BSON (flows are in separate Flows array).
// Parameters are also included in the Objects array.
func serializeMicroflowObjectCollectionWithoutFlows(oc *microflows.MicroflowObjectCollection, params []*microflows.MicroflowParameter) bson.D {
	// Start with array type marker, then serialize objects (NOT flows)
	objects := bson.A{int32(3)} // Array type marker

	// Add parameters first (they appear at the top of the microflow)
	for i, p := range params {
		objects = append(objects, serializeMicroflowParameter(p, i))
	}

	// Add regular microflow objects
	for _, obj := range oc.Objects {
		if objDoc := serializeMicroflowObject(obj); objDoc != nil {
			objects = append(objects, objDoc)
		}
	}

	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(oc.ID))},
		{Key: "$Type", Value: "Microflows$MicroflowObjectCollection"},
		{Key: "Objects", Value: objects},
	}
}

// serializeMicroflowObjectCollection serializes the object collection for nested collections (like in LoopedActivity).
// Note: Flows are NOT included here - in Mendix, all flows are stored at the top-level microflow,
// not inside nested ObjectCollections. SequenceFlow's container must be a Microflow, not a MicroflowObjectCollection.
func serializeMicroflowObjectCollection(oc *microflows.MicroflowObjectCollection) bson.D {
	objects := bson.A{int32(3)} // Array type marker

	for _, obj := range oc.Objects {
		if objDoc := serializeMicroflowObject(obj); objDoc != nil {
			objects = append(objects, objDoc)
		}
	}

	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(string(oc.ID))},
		{Key: "$Type", Value: "Microflows$MicroflowObjectCollection"},
		{Key: "Objects", Value: objects},
	}
}

// serializeMicroflowObject serializes a single microflow object.
func serializeMicroflowObject(obj microflows.MicroflowObject) bson.D {
	switch o := obj.(type) {
	case *microflows.StartEvent:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$StartEvent"},
			{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
			{Key: "Size", Value: sizeToString(o.Size)},
		}

	case *microflows.EndEvent:
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$EndEvent"},
			{Key: "Documentation", Value: ""},
			{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
		}
		if o.ReturnValue != "" {
			doc = append(doc, bson.E{Key: "ReturnValue", Value: o.ReturnValue + "\n"})
		}
		doc = append(doc, bson.E{Key: "Size", Value: sizeToString(o.Size)})
		return doc

	case *microflows.ErrorEvent:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$ErrorEvent"},
			{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
			{Key: "Size", Value: sizeToString(o.Size)},
		}

	case *microflows.ActionActivity:
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$ActionActivity"},
		}
		if o.Action != nil {
			doc = append(doc, bson.E{Key: "Action", Value: serializeMicroflowAction(o.Action)})
		}
		bgColor := o.BackgroundColor
		if bgColor == "" {
			bgColor = "Default"
		}
		doc = append(doc, bson.E{Key: "AutoGenerateCaption", Value: o.AutoGenerateCaption})
		doc = append(doc, bson.E{Key: "BackgroundColor", Value: bgColor})
		doc = append(doc, bson.E{Key: "Caption", Value: o.Caption})
		doc = append(doc, bson.E{Key: "Disabled", Value: false})
		doc = append(doc, bson.E{Key: "Documentation", Value: o.Documentation})
		doc = append(doc, bson.E{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)})
		doc = append(doc, bson.E{Key: "Size", Value: sizeToString(o.Size)})
		return doc

	case *microflows.ExclusiveSplit:
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$ExclusiveSplit"},
			{Key: "Caption", Value: o.Caption},
			{Key: "ErrorHandlingType", Value: string(o.ErrorHandlingType)},
			{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
			{Key: "Size", Value: sizeToString(o.Size)},
		}
		// Serialize SplitCondition
		if o.SplitCondition != nil {
			switch sc := o.SplitCondition.(type) {
			case *microflows.ExpressionSplitCondition:
				doc = append(doc, bson.E{Key: "SplitCondition", Value: bson.D{
					{Key: "$ID", Value: idToBsonBinary(string(sc.ID))},
					{Key: "$Type", Value: "Microflows$ExpressionSplitCondition"},
					{Key: "Expression", Value: sc.Expression},
				}})
			}
		}
		return doc

	case *microflows.ExclusiveMerge:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$ExclusiveMerge"},
			{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
			{Key: "Size", Value: sizeToString(o.Size)},
		}

	case *microflows.LoopedActivity:
		doc := bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$LoopedActivity"},
			{Key: "ErrorHandlingType", Value: string(o.ErrorHandlingType)},
		}
		// Serialize LoopSource (IterableList or WhileLoopCondition)
		switch ls := o.LoopSource.(type) {
		case *microflows.IterableList:
			doc = append(doc, bson.E{Key: "LoopSource", Value: bson.D{
				{Key: "$ID", Value: idToBsonBinary(string(ls.ID))},
				{Key: "$Type", Value: "Microflows$IterableList"},
				{Key: "ListVariableName", Value: ls.ListVariableName},
				{Key: "VariableName", Value: ls.VariableName},
			}})
		case *microflows.WhileLoopCondition:
			doc = append(doc, bson.E{Key: "LoopSource", Value: bson.D{
				{Key: "$ID", Value: idToBsonBinary(string(ls.ID))},
				{Key: "$Type", Value: "Microflows$WhileLoopCondition"},
				{Key: "WhileExpression", Value: ls.WhileExpression},
			}})
		}
		// Serialize nested ObjectCollection
		if o.ObjectCollection != nil {
			doc = append(doc, bson.E{Key: "ObjectCollection", Value: serializeMicroflowObjectCollection(o.ObjectCollection)})
		}
		doc = append(doc,
			bson.E{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
			bson.E{Key: "Size", Value: sizeToString(o.Size)},
		)
		return doc

	case *microflows.BreakEvent:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$BreakEvent"},
			{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
			{Key: "Size", Value: sizeToString(o.Size)},
		}

	case *microflows.ContinueEvent:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$ContinueEvent"},
			{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
			{Key: "Size", Value: sizeToString(o.Size)},
		}

	case *microflows.Annotation:
		return bson.D{
			{Key: "$ID", Value: idToBsonBinary(string(o.ID))},
			{Key: "$Type", Value: "Microflows$Annotation"},
			{Key: "Caption", Value: o.Caption},
			{Key: "RelativeMiddlePoint", Value: pointToString(o.Position)},
			{Key: "Size", Value: sizeToString(o.Size)},
		}

	default:
		return nil
	}
}

// serializePoint serializes a Point to BSON (nested object format).
func serializePoint(pt model.Point) bson.D {
	return bson.D{
		{Key: "$Type", Value: "Common$Point"},
		{Key: "X", Value: int32(pt.X)},
		{Key: "Y", Value: int32(pt.Y)},
	}
}

// serializeSize serializes a Size to BSON (nested object format).
func serializeSize(sz model.Size) bson.D {
	return bson.D{
		{Key: "$Type", Value: "Common$Size"},
		{Key: "Width", Value: int32(sz.Width)},
		{Key: "Height", Value: int32(sz.Height)},
	}
}

// pointToString converts a Point to string format "X;Y" for microflows.
func pointToString(pt model.Point) string {
	return fmt.Sprintf("%d;%d", pt.X, pt.Y)
}

// sizeToString converts a Size to string format "Width;Height" for microflows.
func sizeToString(sz model.Size) string {
	return fmt.Sprintf("%d;%d", sz.Width, sz.Height)
}

// serializeStringTemplate serializes a Text to BSON as a Microflows$StringTemplate.
// This is used for LOG message templates, not Texts$Text.
func serializeStringTemplate(text *model.Text, params []string) bson.D {
	// Get the text from the first translation (usually en_US)
	var textValue string
	for _, value := range text.Translations {
		textValue = value
		break
	}

	// Build parameters array
	var paramsVal any
	if len(params) > 0 {
		paramArr := bson.A{int32(3)} // Array with items marker
		for _, p := range params {
			paramArr = append(paramArr, bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "Microflows$TemplateParameter"},
				{Key: "Expression", Value: p},
			})
		}
		paramsVal = paramArr
	} else {
		paramsVal = bson.A{int32(2)} // Empty array marker
	}

	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "Microflows$StringTemplate"},
		{Key: "Parameters", Value: paramsVal},
		{Key: "Text", Value: textValue},
	}
}

// serializeTextTemplate serializes a Text as a Microflows$TextTemplate with nested Texts$Text.
// This is required for ValidationFeedbackAction.FeedbackTemplate.
func serializeTextTemplate(text *model.Text, params []string) bson.D {
	// Build parameters array
	var paramsVal any
	if len(params) > 0 {
		paramArr := bson.A{int32(3)} // Array with items marker
		for _, p := range params {
			paramArr = append(paramArr, bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "Microflows$TemplateParameter"},
				{Key: "Expression", Value: p},
			})
		}
		paramsVal = paramArr
	} else {
		paramsVal = bson.A{int32(2)} // Empty array marker
	}

	// Build the nested Texts$Text object
	textDoc := bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "Texts$Text"},
	}
	if len(text.Translations) > 0 {
		var transArray bson.A
		transArray = append(transArray, int32(3)) // items marker (3 = has items)
		for lang, value := range text.Translations {
			transArray = append(transArray, bson.D{
				{Key: "$ID", Value: idToBsonBinary(generateUUID())},
				{Key: "$Type", Value: "Texts$Translation"},
				{Key: "LanguageCode", Value: lang},
				{Key: "Text", Value: value},
			})
		}
		textDoc = append(textDoc, bson.E{Key: "Items", Value: transArray})
	} else {
		textDoc = append(textDoc, bson.E{Key: "Items", Value: bson.A{int32(2)}})
	}

	return bson.D{
		{Key: "$ID", Value: idToBsonBinary(generateUUID())},
		{Key: "$Type", Value: "Microflows$TextTemplate"},
		{Key: "Parameters", Value: paramsVal},
		{Key: "Text", Value: textDoc},
	}
}

func (w *Writer) serializeNanoflow(nf *microflows.Nanoflow) ([]byte, error) {
	params := make([]bson.M, 0, len(nf.Parameters))
	for _, p := range nf.Parameters {
		params = append(params, bson.M{
			"$ID":           string(p.ID),
			"$Type":         p.TypeName,
			"Name":          p.Name,
			"Documentation": p.Documentation,
		})
	}

	doc := bson.M{
		"$ID":           string(nf.ID),
		"$Type":         nf.TypeName,
		"Name":          nf.Name,
		"Documentation": nf.Documentation,
		"MarkAsUsed":    nf.MarkAsUsed,
		"Excluded":      nf.Excluded,
		"Parameters":    params,
	}
	return bson.Marshal(doc)
}

// stringOrDefault returns the value if non-empty, otherwise the default.
func stringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
