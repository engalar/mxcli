// internal/fkg/concepts/java_action.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&JavaActionAdapter{}) }

type JavaActionAdapter struct{}

func (a *JavaActionAdapter) Name() string { return "fkg:java-action" }
func (a *JavaActionAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelConcept, LabelCodeExtension, LabelSyntaxFeature, LabelSkill}}
}
func (a *JavaActionAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *JavaActionAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		conceptNode("java-action", "Java Action", "Server-side Java extension called from microflows"),

		extNode("java-action", "Java Action", "create or modify java action with imports and code blocks"),
		extNode("java-action.bcrypt", "BCrypt Integration", "Password hashing with external bcrypt.jar in userlib/"),
		extNode("java-action.sha256", "SHA-256 Hashing", "Pure MDL SHA-256 implementation using java.security"),

		syntaxNode("java-action.create", "CREATE OR MODIFY JAVA ACTION name (params) returns type imports $$ ... $$ code $$ ... $$"),
		syntaxNode("java-action.call", "CALL JAVA ACTION Module.ActionName(params) from microflow"),

		skillNode("extend-with-java", "Java Action creation, parameters, return types, external JAR dependencies"),

		// ── Pattern ─────────────────────────────────────────────────────────────
		patternNode("extend-with-java", "Java Action Extension Pattern",
			"Create Java Action definition → implement Java code → call from microflow"),

		// ── Step nodes ─────────────────────────────────────────────────────────
		stepNode("ja-create", "Create Java Action definition",
			"CREATE OR MODIFY JAVA ACTION with parameters, return type, imports and code blocks",
			1, "create", "JavaAction", "HD.JA_HashPassword",
			"create or modify java action HD.JA_HashPassword (Password: string not null) returns string imports $$ ... $$ code $$ ... $$"),
		stepNode("ja-implement", "Implement Java code",
			"Write imports and code blocks for hashing, encryption, or custom logic",
			2, "configure", "JavaCode", "implementation",
			"imports $$ import java.security.MessageDigest; $$ code $$ MessageDigest digest = MessageDigest.getInstance(\\\"SHA-256\\\"); ... $$"),
		stepNode("ja-call-from-mf", "Call from microflow",
			"Use CALL JAVA ACTION in microflow to invoke the extension",
			3, "wire", "Microflow", "VerifyPassword",
			"call java action HD.JA_VerifyPassword(Password = $Password, HashedPassword = $HashedPassword);"),
		stepNode("ja-deploy-jar", "Deploy external JAR",
			"Place third-party JAR files in project's userlib/ directory for Java Action imports",
			4, "configure", "Dependency", "bcrypt.jar",
			"cp bcrypt-0.9.jar ./userlib/"),

		// ── Edges ──────────────────────────────────────────────────────────────
		edge("java-action", "pattern:extend-with-java", HasPattern),

		edge("java-action", "ext:java-action", HasExt),
		edge("java-action", "ext:java-action.bcrypt", HasExt),
		edge("java-action", "ext:java-action.sha256", HasExt),
		edge("java-action", "syntax:java-action.create", HasSyntax),
		edge("java-action", "syntax:java-action.call", HasSyntax),
		edge("java-action", "skill:extend-with-java", HasSkill),
		edge("java-action", "microflow", RelatedTo),
		edge("java-action", "entity", RelatedTo),

		edge("pattern:extend-with-java", "step:ja-create", HasSyntax),
		edge("pattern:extend-with-java", "step:ja-implement", HasSyntax),
		edge("pattern:extend-with-java", "step:ja-call-from-mf", HasSyntax),
		edge("pattern:extend-with-java", "step:ja-deploy-jar", HasSyntax),
	})
}
