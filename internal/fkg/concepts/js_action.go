// internal/fkg/concepts/js_action.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&JavaScriptActionAdapter{}) }

type JavaScriptActionAdapter struct{}

func (a *JavaScriptActionAdapter) Name() string { return "fkg:js-action" }
func (a *JavaScriptActionAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelCodeExtension, LabelSyntaxFeature, LabelSkill}}
}
func (a *JavaScriptActionAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *JavaScriptActionAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("js-action", "JavaScript Action", "Client-side JavaScript extension called from nanoflows"),

		extNode("js-action", "JavaScript Action", "create or modify javascript action with platform 'Web'"),
		extNode("js-action.clipboard", "Clipboard API", "navigator.clipboard.writeText for copy-to-clipboard"),
		extNode("js-action.notification", "Notification API", "Browser Notification API for desktop alerts"),
		extNode("js-action.relative-time", "Relative Time", "Format Mendix DateTime as relative time string"),

		syntaxNode("js-action.create", "CREATE OR MODIFY JAVASCRIPT ACTION name (params) returns type platform 'Web' { imports $$ ... $$ code $$ ... $$ }"),
		syntaxNode("js-action.call", "CALL JAVASCRIPT ACTION Module.ActionName(params) from nanoflow"),

		skillNode("extend-with-javascript", "JS Action creation, browser API integration, platform declaration"),

		// ── Pattern ─────────────────────────────────────────────────────────────
		patternNode("extend-with-js", "JS Action Extension Pattern",
			"Create JS Action with platform 'Web' → implement browser API code → call from nanoflow"),

		// ── Step nodes ─────────────────────────────────────────────────────────
		stepNode("js-create", "Create JavaScript Action",
			"CREATE OR MODIFY JAVASCRIPT ACTION with platform 'Web' and parameter types",
			1, "create", "JavaScriptAction", "HD.JSA_CopyToClipboard",
			"create or modify javascript action HD.JSA_CopyToClipboard (text: string not null) returns void platform 'Web' { imports $$ ... $$ code $$ ... $$ }"),
		stepNode("js-implement", "Implement browser API call",
			"Write JavaScript code using browser APIs (Clipboard, Notification, Date formatting)",
			2, "configure", "JavaScriptCode", "implementation",
			"imports $$ import \\\"mx-global\\\"; $$ code $$ navigator.clipboard.writeText(text); $$"),
		stepNode("js-call-from-nf", "Call from nanoflow",
			"Use CALL JAVASCRIPT ACTION in nanoflow with parameters and return value",
			3, "wire", "Nanoflow", "NF_CopyToClipboard",
			"call javascript action HD.JSA_CopyToClipboard(Text = $Ticket/Subject);"),

		// ── Edges ──────────────────────────────────────────────────────────────
		edge("js-action", "pattern:extend-with-js", HasPattern),

		edge("js-action", "ext:js-action", HasExt),
		edge("js-action", "ext:js-action.clipboard", HasExt),
		edge("js-action", "ext:js-action.notification", HasExt),
		edge("js-action", "ext:js-action.relative-time", HasExt),
		edge("js-action", "syntax:js-action.create", HasSyntax),
		edge("js-action", "syntax:js-action.call", HasSyntax),
		edge("js-action", "skill:extend-with-javascript", HasSkill),
		edge("js-action", "nanoflow", RelatedTo),
		edge("js-action", "page", RelatedTo),

		edge("pattern:extend-with-js", "step:js-create", HasSyntax),
		edge("pattern:extend-with-js", "step:js-implement", HasSyntax),
		edge("pattern:extend-with-js", "step:js-call-from-nf", HasSyntax),
	})
}
