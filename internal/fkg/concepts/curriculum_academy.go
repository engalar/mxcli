// internal/fkg/concepts/curriculum_academy.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&CurriculumAcademyAdapter{}) }

type CurriculumAcademyAdapter struct{}

func (a *CurriculumAcademyAdapter) Name() string { return "fkg:curriculum-academy" }
func (a *CurriculumAcademyAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{NodeLabels: []mxgraph.Label{LabelCurriculum}}
}
func (a *CurriculumAcademyAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *CurriculumAcademyAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		// ── Module 00: Getting Started ────────────────────────────────────────
		curriculumNode("academy-00-getting-started", "00-入门准备 (Getting Started)",
			"Tool installation, project creation, mxcli check → exec → docker loop"),
		edge("curriculum:academy-00-getting-started", "skill:create-page", Teaches),
		edge("curriculum:academy-00-getting-started", "entity", Teaches),

		// ── Module 01: Domain Modeling ────────────────────────────────────────
		curriculumNode("academy-01-domain-modeling", "01-领域建模 (Domain Modeling)",
			"Entities, attributes, enumerations, associations, constants, indexes"),
		edge("curriculum:academy-01-domain-modeling", "curriculum:academy-00-getting-started", Depends),
		edge("curriculum:academy-01-domain-modeling", "entity", Teaches),
		edge("curriculum:academy-01-domain-modeling", "constant", Teaches),
		edge("curriculum:academy-01-domain-modeling", "skill:generate-domain-model", Teaches),
		edge("curriculum:academy-01-domain-modeling", "skill:mendix/associations", Teaches),
		edge("curriculum:academy-01-domain-modeling", "skill:manage-constants", Teaches),

		// ── Module 02: Microflow Business Logic ────────────────────────────────
		curriculumNode("academy-02-microflows", "02-微流业务逻辑 (Microflow Business Logic)",
			"Microflow syntax, state machine, validation, SLA calculation"),
		edge("curriculum:academy-02-microflows", "curriculum:academy-01-domain-modeling", Depends),
		edge("curriculum:academy-02-microflows", "microflow", Teaches),
		edge("curriculum:academy-02-microflows", "constant", Teaches),
		edge("curriculum:academy-02-microflows", "skill:write-microflows", Teaches),
		edge("curriculum:academy-02-microflows", "skill:patterns-data-processing", Teaches),
		edge("curriculum:academy-02-microflows", "pattern:state-machine-sla", Teaches),
		edge("curriculum:academy-02-microflows", "pattern:validation-feedback", Teaches),

		// ── Module 03: Nanoflows ───────────────────────────────────────────────
		curriculumNode("academy-03-nanoflows", "03-纳流与客户端 (Nanoflows & Client-Side)",
			"Nanoflow syntax, client-side logic, restrictions vs microflows"),
		edge("curriculum:academy-03-nanoflows", "curriculum:academy-02-microflows", Depends),
		edge("curriculum:academy-03-nanoflows", "nanoflow", Teaches),
		edge("curriculum:academy-03-nanoflows", "skill:write-nanoflows", Teaches),

		// ── Module 04: Pages & UI ──────────────────────────────────────────────
		curriculumNode("academy-04-pages", "04-页面与UI (Pages & UI)",
			"Page creation, DataGrid, DataView, LayoutGrid, filters, actions"),
		edge("curriculum:academy-04-pages", "curriculum:academy-01-domain-modeling", Depends),
		edge("curriculum:academy-04-pages", "curriculum:academy-02-microflows", Depends),
		edge("curriculum:academy-04-pages", "page", Teaches),
		edge("curriculum:academy-04-pages", "skill:create-page", Teaches),
		edge("curriculum:academy-04-pages", "skill:alter-page", Teaches),
		edge("curriculum:academy-04-pages", "skill:overview-pages", Teaches),
		edge("curriculum:academy-04-pages", "skill:master-detail-pages", Teaches),
		edge("curriculum:academy-04-pages", "pattern:overview-page", Teaches),
		edge("curriculum:academy-04-pages", "pattern:master-detail", Teaches),
		edge("curriculum:academy-04-pages", "pattern:popup-form", Teaches),

		// ── Module 05: Security & Permissions ──────────────────────────────────
		curriculumNode("academy-05-security", "05-安全与权限 (Security & Permissions)",
			"Module roles, entity grants, XPath row-level filters, demo users, navigation"),
		edge("curriculum:academy-05-security", "curriculum:academy-01-domain-modeling", Depends),
		edge("curriculum:academy-05-security", "curriculum:academy-04-pages", Depends),
		edge("curriculum:academy-05-security", "security", Teaches),
		edge("curriculum:academy-05-security", "navigation", Teaches),
		edge("curriculum:academy-05-security", "skill:manage-security", Teaches),
		edge("curriculum:academy-05-security", "skill:manage-navigation", Teaches),
		edge("curriculum:academy-05-security", "pattern:xpath-row-filter", Teaches),
		edge("curriculum:academy-05-security", "pattern:demo-user-setup", Teaches),

		// ── Module 06: Knowledge Base Module ───────────────────────────────────
		curriculumNode("academy-06-kb", "06-知识库模块 (Knowledge Base Module)",
			"Self-referencing associations, many-to-many via intermediate entity, cross-module security"),
		edge("curriculum:academy-06-kb", "curriculum:academy-01-domain-modeling", Depends),
		edge("curriculum:academy-06-kb", "curriculum:academy-05-security", Depends),
		edge("curriculum:academy-06-kb", "entity", Teaches),
		edge("curriculum:academy-06-kb", "security", Teaches),
		edge("curriculum:academy-06-kb", "skill:manage-security", Teaches),
		edge("curriculum:academy-06-kb", "pattern:self-ref-association", Teaches),
		edge("curriculum:academy-06-kb", "pattern:many-to-many", Teaches),

		// ── Module 07: Escalation & Workflow ───────────────────────────────────
		curriculumNode("academy-07-workflow", "07-审批工作流 (Approval Workflow)",
			"Microflow-based escalation, native Workflow Engine, user tasks, boundary events"),
		edge("curriculum:academy-07-workflow", "curriculum:academy-02-microflows", Depends),
		edge("curriculum:academy-07-workflow", "curriculum:academy-04-pages", Depends),
		edge("curriculum:academy-07-workflow", "workflow", Teaches),
		edge("curriculum:academy-07-workflow", "microflow", Teaches),
		edge("curriculum:academy-07-workflow", "pattern:popup-form", Teaches),

		// ── Module 08: Java Action ─────────────────────────────────────────────
		curriculumNode("academy-08-java-action", "08-扩展-Java-Action (Java Action Extension)",
			"Java Action creation, BCrypt password hashing, external JAR dependencies"),
		edge("curriculum:academy-08-java-action", "curriculum:academy-02-microflows", Depends),
		edge("curriculum:academy-08-java-action", "java-action", Teaches),
		edge("curriculum:academy-08-java-action", "skill:extend-with-java", Teaches),

		// ── Module 09: JavaScript Action ───────────────────────────────────────
		curriculumNode("academy-09-js-action", "09-扩展-JS-Action (JavaScript Action Extension)",
			"JavaScript Action creation, clipboard API, notifications, relative time"),
		edge("curriculum:academy-09-js-action", "curriculum:academy-03-nanoflows", Depends),
		edge("curriculum:academy-09-js-action", "js-action", Teaches),
		edge("curriculum:academy-09-js-action", "skill:extend-with-javascript", Teaches),

		// ── Module 10: Widget Development ──────────────────────────────────────
		curriculumNode("academy-10-widget", "10-扩展-Widget开发 (Widget Development)",
			"Pluggable Widget creation, build pipeline, PLUGGABLEWIDGET MDL syntax"),
		edge("curriculum:academy-10-widget", "curriculum:academy-04-pages", Depends),
		edge("curriculum:academy-10-widget", "widget", Teaches),
		edge("curriculum:academy-10-widget", "skill:mendix/custom-widgets", Teaches),

		// ── Module 11: Theme Customization ─────────────────────────────────────
		curriculumNode("academy-11-theme", "11-扩展-主题定制 (Theme Customization)",
			"Atlas UI CSS variable override, brand colors, button border-radius"),
		edge("curriculum:academy-11-theme", "curriculum:academy-04-pages", Depends),
		edge("curriculum:academy-11-theme", "page", Teaches),
		// Theme is pure CSS — no FKG concept node, listed as extension
		edge("curriculum:academy-11-theme", "ext:css-theme", Teaches),

		// ── Module 12: AI Collaboration ────────────────────────────────────────
		curriculumNode("academy-12-ai-collab", "12-AI协作模式 (AI Collaboration Patterns)",
			"Prompt engineering, verification-driven development, error debugging patterns"),
		edge("curriculum:academy-12-ai-collab", "curriculum:academy-00-getting-started", Depends),

		// ── Capstone ───────────────────────────────────────────────────────────
		curriculumNode("academy-capstone", "Capstone: Helpdesk Full Delivery",
			"Full-stack integration: all modules combined, demo data seeding, end-to-end verification"),
		edge("curriculum:academy-capstone", "curriculum:academy-01-domain-modeling", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-02-microflows", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-03-nanoflows", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-04-pages", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-05-security", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-06-kb", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-07-workflow", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-08-java-action", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-09-js-action", Depends),
		edge("curriculum:academy-capstone", "curriculum:academy-10-widget", Depends),
		edge("curriculum:academy-capstone", "pattern:seed-demo-data", Teaches),
		edge("curriculum:academy-capstone", "skill:create-page", Teaches),
		edge("curriculum:academy-capstone", "skill:write-microflows", Teaches),
	})
}
