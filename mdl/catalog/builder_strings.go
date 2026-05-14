// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	"github.com/mendixlabs/mxcli/sdk/workflows"
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

	// Extract from pages (title, URL) — using cached list
	pageList, err := b.cachedPages()
	if err == nil {
		for _, pg := range pageList {
			moduleID := b.hierarchy.findModuleID(pg.ContainerID)
			moduleName := b.hierarchy.getModuleName(moduleID)
			qn := moduleName + "." + pg.Name

			pageID := string(pg.ID)

			// Page title translations (with language code)
			if pg.Title != nil && pg.Title.Translations != nil {
				for lang, t := range pg.Title.Translations {
					insert(qn, "PAGE", t, "page_title", lang, pageID, moduleName)
				}
			}

			// Page URL (no language)
			if pg.URL != "" {
				insert(qn, "PAGE", pg.URL, "page_url", "", pageID, moduleName)
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

	// Extract from workflows — using cached list
	wfList, err := b.cachedWorkflows()
	if err == nil {
		for _, wf := range wfList {
			moduleID := b.hierarchy.findModuleID(wf.ContainerID)
			moduleName := b.hierarchy.getModuleName(moduleID)
			qn := moduleName + "." + wf.Name

			wfID := string(wf.ID)
			if wf.WorkflowName != "" {
				insert(qn, "WORKFLOW", wf.WorkflowName, "workflow_name", "", wfID, moduleName)
			}
			if wf.WorkflowDescription != "" {
				insert(qn, "WORKFLOW", wf.WorkflowDescription, "workflow_description", "", wfID, moduleName)
			}
			if wf.Documentation != "" {
				insert(qn, "WORKFLOW", wf.Documentation, "documentation", "", wfID, moduleName)
			}

			if wf.Flow != nil {
				extractWorkflowFlowStrings(wf.Flow, qn, moduleName, insert)
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

// extractWorkflowFlowStrings extracts strings from workflow activities recursively.
func extractWorkflowFlowStrings(flow *workflows.Flow, qn, moduleName string, insert func(string, string, string, string, string, string, string)) {
	for _, act := range flow.Activities {
		actID := string(act.GetID())
		if act.GetCaption() != "" {
			insert(qn, "WORKFLOW", act.GetCaption(), "activity_caption", "", actID, moduleName)
		}

		switch a := act.(type) {
		case *workflows.UserTask:
			if a.TaskName != "" {
				insert(qn, "WORKFLOW", a.TaskName, "task_name", "", actID, moduleName)
			}
			if a.TaskDescription != "" {
				insert(qn, "WORKFLOW", a.TaskDescription, "task_description", "", actID, moduleName)
			}
			for _, outcome := range a.Outcomes {
				if outcome.Caption != "" {
					insert(qn, "WORKFLOW", outcome.Caption, "outcome_caption", "", actID, moduleName)
				}
				if outcome.Flow != nil {
					extractWorkflowFlowStrings(outcome.Flow, qn, moduleName, insert)
				}
			}
		case *workflows.SystemTask:
			for _, outcome := range a.Outcomes {
				if f := outcome.GetFlow(); f != nil {
					extractWorkflowFlowStrings(f, qn, moduleName, insert)
				}
			}
		case *workflows.CallMicroflowTask:
			for _, outcome := range a.Outcomes {
				if f := outcome.GetFlow(); f != nil {
					extractWorkflowFlowStrings(f, qn, moduleName, insert)
				}
			}
		case *workflows.ExclusiveSplitActivity:
			for _, outcome := range a.Outcomes {
				if f := outcome.GetFlow(); f != nil {
					extractWorkflowFlowStrings(f, qn, moduleName, insert)
				}
			}
		case *workflows.ParallelSplitActivity:
			for _, outcome := range a.Outcomes {
				if outcome.Flow != nil {
					extractWorkflowFlowStrings(outcome.Flow, qn, moduleName, insert)
				}
			}
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
