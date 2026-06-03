// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// microflowActionTypeAliases maps user-facing MDL action keywords to the BSON
// $Type stored in the microflow. The MDL grammar accepts the short keyword; the
// resolved $Type is what setMicroflowActionTranslationBSON matches against.
var microflowActionTypeAliases = map[string]string{
	"ShowMessage":       "Microflows$ShowMessageAction",
	"LogMessage":        "Microflows$LogMessageAction",
	"ValidationMessage": "Microflows$ValidationMessageAction",
}

// resolveMicroflowActionType returns the BSON $Type for a user-facing action
// keyword. If the keyword already looks like a full $Type ("Microflows$..."),
// it is returned unchanged so callers can pass either form.
func resolveMicroflowActionType(keyword string) string {
	if t, ok := microflowActionTypeAliases[keyword]; ok {
		return t
	}
	return keyword
}

// microflowActionTextField maps an action $Type + property to the dotted path of
// the Texts$Text document inside the action. Microflow message actions store
// their text in a TextTemplate whose field name varies by action type.
func microflowActionTextField(actionType, property string) (templateKey string, ok bool) {
	if property != "message" && property != "Message" {
		return "", false
	}
	switch actionType {
	case "Microflows$ShowMessageAction", "Microflows$ValidationMessageAction":
		return "Template", true
	case "Microflows$LogMessageAction":
		return "MessageTemplate", true
	default:
		return "", false
	}
}

// setMicroflowActionTranslationBSON finds the index-th action of actionType
// inside the microflow document and sets the langCode translation on the
// Texts$Text reached via its TextTemplate. Returns false if no matching action
// exists at that index or it has no translatable text for the property.
func setMicroflowActionTranslationBSON(mfDoc bson.D, actionType string, index int, property, langCode, text string) bool {
	templateKey, ok := microflowActionTextField(actionType, property)
	if !ok {
		return false
	}

	coll := dGetDoc(mfDoc, "ObjectCollection")
	if coll == nil {
		return false
	}

	seen := 0
	for _, obj := range dGetArrayElements(dGet(coll, "Objects")) {
		activity, ok := obj.(bson.D)
		if !ok {
			continue
		}
		action := dGetDoc(activity, "Action")
		if action == nil {
			continue
		}
		if dGetString(action, "$Type") != actionType {
			continue
		}
		if seen != index {
			seen++
			continue
		}
		// Found the target action. Locate its Texts$Text via the template.
		template := dGetDoc(action, templateKey)
		if template == nil {
			return false
		}
		textDoc := dGetDoc(template, "Text")
		if textDoc == nil {
			return false
		}
		setTranslationForLang(textDoc, langCode, text)
		return true
	}
	return false
}

// SetMicroflowActionTranslation loads the microflow unit, sets the langCode
// translation on the addressed action's text property, and saves the unit.
func (b *MprBackend) SetMicroflowActionTranslation(docQN, actionType string, index int, property, langCode, text string) error {
	info, err := b.GetRawUnitByName("microflow", docQN)
	if err != nil {
		return fmt.Errorf("locate microflow %q: %w", docQN, err)
	}
	if info == nil {
		return mdlerrors.NewValidationf("microflow %q not found", docQN)
	}

	rawBytes, err := b.msdkReader.GetRawUnitBytes(info.ID)
	if err != nil {
		return fmt.Errorf("read microflow unit: %w", err)
	}
	var mfDoc bson.D
	if err := bson.Unmarshal(rawBytes, &mfDoc); err != nil {
		return fmt.Errorf("unmarshal microflow BSON: %w", err)
	}

	resolved := resolveMicroflowActionType(actionType)
	if !setMicroflowActionTranslationBSON(mfDoc, resolved, index, property, langCode, text) {
		return mdlerrors.NewValidationf(
			"no translatable %s[%d].%s found in microflow %q "+
				"(check the action type, index, and that the action has a message)",
			actionType, index, property, docQN)
	}

	contents, err := bson.Marshal(mfDoc)
	if err != nil {
		return fmt.Errorf("marshal microflow BSON: %w", err)
	}
	return b.writeUnitContents(model.ID(info.ID), contents)
}
