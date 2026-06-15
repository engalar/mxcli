# AI-Assisted Development Academy — English Curriculum

Build a complete IT Helpdesk application using Claude Code + mxcli,
mastering AI-assisted Mendix development from domain modeling to production-ready apps.

---

## Course Map

| Module | Topic | Prerequisites |
|--------|-------|---------------|
| [00 Getting Started](00-getting-started/) | Setup & first exec | None |
| [01 Domain Modeling](01-domain-modeling/) | Entities, enums, associations | 00 |
| [02 Microflows](02-microflows/) | Business logic, state machines | 01 |
| [03 Nanoflows](03-nanoflows/) | Client-side logic | 01 |
| [04 Pages & UI](04-pages-ui/) | Atlas layouts, forms, grids | 01–03 |
| [05 Security](05-security/) | Roles, access rules, row-level XPath | 01–04 |
| [06 Knowledge Base](06-knowledge-base/) | Cross-module, many-to-many | 05 |
| [07 Escalation Workflow](07-escalation-workflow/) | Approval state machine | 02 |
| [08 Java Action](08-extension-java-action/) | BCrypt, server extensions | Studio Pro + JDK |
| [09 JS Action](09-extension-js-action/) | Browser API, client extensions | 01 |
| [10 Widget Development](10-extension-widget-dev/) | React pluggable widget | Node.js |
| [11 Theming](11-extension-theming/) | Atlas CSS variables | None |
| [12 AI Collaboration](12-ai-collaboration/) | Prompt design, debugging patterns | All |
| [Capstone](capstone-helpdesk/) | Full app delivery | All modules |

## MDL Reference Files

MDL files are language-neutral and shared with the Chinese curriculum.
See `zh/[module]/参考实现/` for all reference implementations.

## Quick Start

```bash
mxcli new MyHelpdesk --version 11.6.6
claude  # open Claude Code in project directory
# Follow the guide in 00-getting-started/ai-collaboration-guide.md
```
