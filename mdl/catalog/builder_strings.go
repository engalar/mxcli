// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)

// buildStrings extracts string literals from documents into the FTS5 strings table.
// Only runs in full mode.
func (b *Builder) buildStrings() error {
	if !b.fullMode {
		return nil
	}

	stmt, err := b.tx.Prepare(`
		INSERT INTO strings (QualifiedName, ObjectType, StringValue, StringContext, Language, ElementId, ModuleName)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	count := 0
	insert := func(qn, objType, value, ctx, lang, elementID, module string) {
		if value == "" {
			return
		}
		stmt.Exec(qn, objType, value, ctx, lang, elementID, module)
		count++
	}

	// Extract from pages (title, URL) — using cached gen list. Container
	// linkage is resolved via the unit hierarchy (Stage 3.3.5.C7e).
	pageGenList, err := b.cachedPagesGen()
	if err == nil {
		for _, pg := range pageGenList {
			if pg == nil {
				continue
			}
			pgID := model.ID(pg.ID())
			containerID := b.hierarchy.containerParent[pgID]
			moduleID := b.hierarchy.findModuleID(containerID)
			moduleName := b.hierarchy.getModuleName(moduleID)
			qn := moduleName + "." + pg.Name()

			pageIDStr := string(pgID)

			// Page title translations (with language code). gen Page
			// stores Title as element.Element decoded into *texts.Text.
			for _, item := range textTranslations(pg.Title()) {
				insert(qn, "PAGE", item.text, "page_title", item.lang, pageIDStr, moduleName)
			}

			// Page URL (no language)
			if url := pg.Url(); url != "" {
				insert(qn, "PAGE", url, "page_url", "", pageIDStr, moduleName)
			}
		}
	}

	// Extract from microflows — using cached list
	mfList, err := b.cachedMicroflows()
	if err == nil {
		for _, mf := range mfList {
			if mf == nil {
				continue
			}
			moduleID := b.hierarchy.findModuleID(model.ID(mf.ID()))
			moduleName := b.hierarchy.getModuleName(moduleID)
			qn := moduleName + "." + mf.Name()

			mfID := string(mf.ID())

			// Documentation (no language)
			if doc := mf.Documentation(); doc != "" {
				insert(qn, "MICROFLOW", doc, "documentation", "", mfID, moduleName)
			}

			// Extract strings from activities
			extractActivityStrings(flowObjectCollection(mf.ObjectCollection()), qn, "MICROFLOW", moduleName, insert)
		}
	}

	// Extract from enumerations (value captions) — using cached list
	enums, err := b.cachedEnumerations()
	if err == nil {
		for _, enum := range enums {
			moduleID := b.hierarchy.findModuleID(enum.ContainerID)
			moduleName := b.hierarchy.getModuleName(moduleID)
			qn := moduleName + "." + enum.Name

			enumID := string(enum.ID)
			for _, val := range enum.Values {
				if val.Caption != nil && val.Caption.Translations != nil {
					valID := string(val.ID)
					if valID == "" {
						valID = enumID
					}
					for lang, t := range val.Caption.Translations {
						insert(qn, "ENUMERATION", t, "enum_caption", lang, valID, moduleName)
					}
				}
			}
		}
	}

	// Extract from workflows — using cached gen list
	wfList, err := b.cachedWorkflows()
	if err == nil {
		for _, wf := range wfList {
			if wf == nil {
				continue
			}
			moduleID := b.hierarchy.findModuleID(model.ID(wf.ID()))
			moduleName := b.hierarchy.getModuleName(moduleID)
			qn := moduleName + "." + wf.Name()

			wfID := string(wf.ID())
			if name := readWorkflowTextElement(wf.WorkflowName()); name != "" {
				insert(qn, "WORKFLOW", name, "workflow_name", "", wfID, moduleName)
			} else if title := wf.Title(); title != "" {
				insert(qn, "WORKFLOW", title, "workflow_name", "", wfID, moduleName)
			}
			if desc := readWorkflowTextElement(wf.WorkflowDescription()); desc != "" {
				insert(qn, "WORKFLOW", desc, "workflow_description", "", wfID, moduleName)
			}
			if doc := wf.Documentation(); doc != "" {
				insert(qn, "WORKFLOW", doc, "documentation", "", wfID, moduleName)
			}
			if flow, ok := wf.Flow().(*genWf.Flow); ok && flow != nil {
				extractWorkflowFlowStringsGen(flow, qn, moduleName, insert)
			}
		}
	}

	// Extract from published REST services
	prsServices, err := b.reader.ListPublishedRestServices()
	if err == nil {
		for _, svc := range prsServices {
			moduleID := b.hierarchy.findModuleID(svc.ContainerID)
			moduleName := b.hierarchy.getModuleName(moduleID)
			qn := moduleName + "." + svc.Name
			svcID := string(svc.ID)

			insert(qn, "PUBLISHED_REST_SERVICE", svc.Path, "rest_path", "", svcID, moduleName)
			insert(qn, "PUBLISHED_REST_SERVICE", svc.ServiceName, "service_name", "", svcID, moduleName)
			insert(qn, "PUBLISHED_REST_SERVICE", svc.Version, "version", "", svcID, moduleName)

			for _, res := range svc.Resources {
				insert(qn, "PUBLISHED_REST_SERVICE", res.Name, "resource_name", "", svcID, moduleName)
				for _, op := range res.Operations {
					if op.Path != "" {
						insert(qn, "PUBLISHED_REST_SERVICE", op.Path, "operation_path", "", svcID, moduleName)
					}
					if op.Summary != "" {
						insert(qn, "PUBLISHED_REST_SERVICE", op.Summary, "operation_summary", "", svcID, moduleName)
					}
				}
			}
		}
	}

	b.report("strings", count)
	return nil
}

// readWorkflowTextElement extracts the inner Text scalar from a Texts$Text
// wrapper element.
func readWorkflowTextElement(elem element.Element) string {
	if elem == nil {
		return ""
	}
	for _, field := range []string{"Text", "Translation", "Value"} {
		if v, _ := codec.ReadBSONFieldString(elem.Raw(), field); v != "" {
			return v
		}
	}
	return ""
}

// activityCaptionGen extracts the Caption field across the heterogenous
// gen activity types via a small reflection-free type switch.
func activityCaptionGen(act element.Element) string {
	switch v := act.(type) {
	case *genWf.UserTask:
		return v.Caption()
	case *genWf.SingleUserTaskActivity:
		return v.Caption()
	case *genWf.MultiUserTaskActivity:
		return v.Caption()
	case *genWf.CallMicroflowActivity:
		return v.Caption()
	case *genWf.CallMicroflowTask:
		return v.Caption()
	case *genWf.CallWorkflowActivity:
		return v.Caption()
	case *genWf.ExclusiveSplitActivity:
		return v.Caption()
	case *genWf.ParallelSplitActivity:
		return v.Caption()
	case *genWf.JumpToActivity:
		return v.Caption()
	case *genWf.WaitForTimerActivity:
		return v.Caption()
	case *genWf.WaitForNotificationActivity:
		return v.Caption()
	}
	v, _ := codec.ReadBSONFieldString(act.Raw(), "Caption")
	return v
}

// extractWorkflowFlowStringsGen extracts strings from workflow activities
// recursively. Mirrors the legacy extractWorkflowFlowStrings semantics.
func extractWorkflowFlowStringsGen(flow *genWf.Flow, qn, moduleName string, insert func(string, string, string, string, string, string, string)) {
	if flow == nil {
		return
	}
	for _, act := range flow.ActivitiesItems() {
		if act == nil {
			continue
		}
		actID := string(act.ID())
		if cap := activityCaptionGen(act); cap != "" {
			insert(qn, "WORKFLOW", cap, "activity_caption", "", actID, moduleName)
		}

		// User-task family: TaskName + TaskDescription + outcome captions
		switch v := act.(type) {
		case *genWf.UserTask:
			if name := readWorkflowTextElement(v.TaskName()); name != "" {
				insert(qn, "WORKFLOW", name, "task_name", "", actID, moduleName)
			}
			if desc := readWorkflowTextElement(v.TaskDescription()); desc != "" {
				insert(qn, "WORKFLOW", desc, "task_description", "", actID, moduleName)
			}
			extractUserTaskOutcomeStringsGen(v.OutcomesItems(), qn, moduleName, actID, insert)
		case *genWf.SingleUserTaskActivity:
			if name := readWorkflowTextElement(v.TaskName()); name != "" {
				insert(qn, "WORKFLOW", name, "task_name", "", actID, moduleName)
			}
			if desc := readWorkflowTextElement(v.TaskDescription()); desc != "" {
				insert(qn, "WORKFLOW", desc, "task_description", "", actID, moduleName)
			}
			extractUserTaskOutcomeStringsGen(v.OutcomesItems(), qn, moduleName, actID, insert)
		case *genWf.MultiUserTaskActivity:
			if name := readWorkflowTextElement(v.TaskName()); name != "" {
				insert(qn, "WORKFLOW", name, "task_name", "", actID, moduleName)
			}
			if desc := readWorkflowTextElement(v.TaskDescription()); desc != "" {
				insert(qn, "WORKFLOW", desc, "task_description", "", actID, moduleName)
			}
			extractUserTaskOutcomeStringsGen(v.OutcomesItems(), qn, moduleName, actID, insert)
		case *genWf.CallMicroflowActivity:
			extractConditionOutcomeStringsGen(v.OutcomesItems(), qn, moduleName, insert)
		case *genWf.CallMicroflowTask:
			extractConditionOutcomeStringsGen(v.OutcomesItems(), qn, moduleName, insert)
		case *genWf.ExclusiveSplitActivity:
			extractConditionOutcomeStringsGen(v.OutcomesItems(), qn, moduleName, insert)
		case *genWf.ParallelSplitActivity:
			for _, oc := range v.OutcomesItems() {
				if pso, ok := oc.(*genWf.ParallelSplitOutcome); ok {
					if f, ok := pso.Flow().(*genWf.Flow); ok {
						extractWorkflowFlowStringsGen(f, qn, moduleName, insert)
					}
				}
			}
		}
	}
}

func extractUserTaskOutcomeStringsGen(outcomes []element.Element, qn, moduleName, actID string, insert func(string, string, string, string, string, string, string)) {
	for _, oc := range outcomes {
		if utc, ok := oc.(*genWf.UserTaskOutcome); ok {
			if cap := utc.Caption(); cap != "" {
				insert(qn, "WORKFLOW", cap, "outcome_caption", "", actID, moduleName)
			}
			if f, ok := utc.Flow().(*genWf.Flow); ok && f != nil {
				extractWorkflowFlowStringsGen(f, qn, moduleName, insert)
			}
		}
	}
}

func extractConditionOutcomeStringsGen(outcomes []element.Element, qn, moduleName string, insert func(string, string, string, string, string, string, string)) {
	for _, oc := range outcomes {
		var f *genWf.Flow
		switch v := oc.(type) {
		case *genWf.BooleanConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.EnumerationValueConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.VoidConditionOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		case *genWf.ExclusiveSplitOutcome:
			f, _ = v.Flow().(*genWf.Flow)
		}
		if f != nil {
			extractWorkflowFlowStringsGen(f, qn, moduleName, insert)
		}
	}
}

// extractActivityStrings extracts string literals from microflow/nanoflow activities.
// Walks gen-typed ObjectCollections; mirrors the legacy sdk-typed
// switch but reads through TextTranslations on the gen Text element.
func extractActivityStrings(oc *genMf.MicroflowObjectCollection, qn, objType, moduleName string, insert func(string, string, string, string, string, string, string)) {
	if oc == nil {
		return
	}

	for _, obj := range oc.ObjectsItems() {
		act, ok := obj.(*genMf.ActionActivity)
		if !ok {
			continue
		}
		inner := act.Action()
		if inner == nil {
			continue
		}

		actID := string(act.ID())

		switch a := inner.(type) {
		case *genMf.LogMessageAction:
			emitTextTranslations(a.MessageTemplate(), qn, objType, "log_message", actID, moduleName, insert)
			if node := a.Node(); node != "" {
				insert(qn, objType, node, "log_node", "", actID, moduleName)
			}
		case *genMf.ShowMessageAction:
			emitTextTranslations(a.Template(), qn, objType, "show_message", actID, moduleName, insert)
		case *genMf.ValidationFeedbackAction:
			emitTextTranslations(a.FeedbackTemplate(), qn, objType, "validation_message", actID, moduleName, insert)
		}
	}
}

// emitTextTranslations walks a gen Text element's translations and
// emits one insert call per (language, text) pair. No-op when the
// element is nil or not a *texts.Text.
func emitTextTranslations(text textElement, qn, objType, ctxKey, actID, moduleName string, insert func(string, string, string, string, string, string, string)) {
	if text == nil {
		return
	}
	for _, item := range textTranslations(text) {
		insert(qn, objType, item.text, ctxKey, item.lang, actID, moduleName)
	}
}

// textElement is the abstract surface used by emitTextTranslations.
// Both gen Text (texts.Text) and any future text-bearing element with
// the same TranslationsItems() shape will satisfy it.
type textElement = element.Element

// translationPair is a (language code, text) tuple extracted from a
// gen Text element's Translations list.
type translationPair struct {
	lang string
	text string
}

// textTranslations extracts the Translations list of a gen
// texts.Text element. Returns nil for unknown element shapes (the
// caller will simply not emit anything).
func textTranslations(e element.Element) []translationPair {
	if e == nil {
		return nil
	}
	t, ok := e.(*texts.Text)
	if !ok {
		return nil
	}
	items := t.TranslationsItems()
	out := make([]translationPair, 0, len(items))
	for _, child := range items {
		tr, ok := child.(*texts.Translation)
		if !ok || tr == nil {
			continue
		}
		out = append(out, translationPair{lang: tr.LanguageCode(), text: tr.Text()})
	}
	return out
}
