// internal/fkg/concepts/security_patterns.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&SecurityPatternsAdapter{}) }

type SecurityPatternsAdapter struct{}

func (a *SecurityPatternsAdapter) Name() string { return "fkg:security-patterns" }
func (a *SecurityPatternsAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelPattern, LabelImplDetail}}
}
func (a *SecurityPatternsAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *SecurityPatternsAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		// ── Patterns ──────────────────────────────────────────────────────────
		patternNode("xpath-row-filter", "XPath Row-Level Filter Pattern",
			"Multi-tenant data isolation: entity grants with XPath where [System.owner='[%CurrentUser%]']"),
		patternNode("demo-user-setup", "Demo User Setup Pattern",
			"Development test users with create or modify demo user for each user role"),

		// ── ImplDetail nodes ──────────────────────────────────────────────────
		implDetailNode("grant-entity", "Entity Grant", "grant role on entity (create, read *, write *) where '...'"),
		implDetailNode("grant-execute", "Execute Grant", "grant execute on microflow Module.Action to role"),
		implDetailNode("grant-view", "View Grant", "grant view on page Module.Page to role"),
		implDetailNode("xpath-currentuser", "XPath CurrentUser", "where '[System.owner=''[%CurrentUser%]'']' — double single quotes required"),
		implDetailNode("module-role", "Module Role", "create module role HD.CustomerRole"),
		implDetailNode("user-role", "User Role", "create or modify user role Customer (System.User, HD.CustomerRole)"),
		implDetailNode("demo-user", "Demo User", "create or modify demo user 'test@example.com' password 'Demo1234' entity ..."),
		implDetailNode("password-policy", "Password Policy", "alter project security password policy (min_length: 8, require_digit: true)"),
		implDetailNode("security-level", "Security Level", "alter project security level production / off / prototype"),

		// ── Edges: Pattern → concept ──────────────────────────────────────────
		edge("security", "pattern:xpath-row-filter", HasPattern),
		edge("security", "pattern:demo-user-setup", HasPattern),

		// ── Step nodes ─────────────────────────────────────────────────────────
		stepNode("xpath-create-role", "Create module role",
			"CREATE MODULE ROLE for the user type",
			1, "create", "ModuleRole", "HD.CustomerRole",
			"create module role HD.CustomerRole;"),
		stepNode("xpath-grant-entity", "Grant entity access with XPath",
			"GRANT on entity with XPath row-level filter using System.owner",
			2, "grant", "EntityAccess", "HD.Ticket → HD.CustomerRole",
			"grant HD.CustomerRole on HD.Ticket (create, read *, write *) where '[HD.Ticket_Customer/...=''[%CurrentUser%]'']';"),
		stepNode("xpath-grant-execute", "Grant microflow execute",
			"GRANT EXECUTE on relevant microflows to the role",
			3, "grant", "MicroflowAccess", "ACT_*",
			"grant execute on microflow HD.ACT_Ticket_Submit to HD.CustomerRole;"),
		stepNode("xpath-grant-page", "Grant page view",
			"GRANT VIEW on pages to the role",
			4, "grant", "PageAccess", "HD.Ticket_Overview",
			"grant view on page HD.Ticket_Overview to HD.CustomerRole;"),

		stepNode("demo-create-role", "Create user role",
			"Compose module roles into a user role",
			1, "create", "UserRole", "Customer",
			"create or modify user role Customer (System.User, HD.CustomerRole);"),
		stepNode("demo-create-user", "Create demo user",
			"CREATE OR MODIFY DEMO USER for the role",
			2, "create", "DemoUser", "demo@test.com",
			"create or modify demo user 'demo@test.com' password 'Demo1234' entity Administration.Account (Customer);"),
		stepNode("demo-set-level", "Set security level",
			"ALTER PROJECT SECURITY to production level",
			3, "configure", "SecurityLevel", "production",
			"alter project security level production;"),

		// ── Edges: pattern → step ───────────────────────────────────────────
		edge("pattern:xpath-row-filter", "step:xpath-create-role", HasSyntax),
		edge("pattern:xpath-row-filter", "step:xpath-grant-entity", HasSyntax),
		edge("pattern:xpath-row-filter", "step:xpath-grant-execute", HasSyntax),
		edge("pattern:xpath-row-filter", "step:xpath-grant-page", HasSyntax),
		edge("pattern:demo-user-setup", "step:demo-create-role", HasSyntax),
		edge("pattern:demo-user-setup", "step:demo-create-user", HasSyntax),
		edge("pattern:demo-user-setup", "step:demo-set-level", HasSyntax),
	})
}
