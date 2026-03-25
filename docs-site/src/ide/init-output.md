# What Gets Created

Running `mxcli init` creates the following directory structure in your Mendix project.

## Universal Files (All Tools)

These files are shared by all AI tools:

```
your-mendix-project/
├── AGENTS.md                          # Comprehensive AI assistant guide
├── .ai-context/
│   ├── skills/                        # MDL pattern guides
│   │   ├── write-microflows.md        # Microflow syntax and patterns
│   │   ├── create-page.md            # Page/widget syntax reference
│   │   ├── alter-page.md             # ALTER PAGE in-place modifications
│   │   ├── overview-pages.md         # CRUD page patterns
│   │   ├── master-detail-pages.md    # Master-detail page patterns
│   │   ├── generate-domain-model.md  # Entity/Association syntax
│   │   ├── check-syntax.md           # Pre-flight validation checklist
│   │   ├── organize-project.md       # Folders, MOVE, project structure
│   │   ├── manage-security.md        # Roles, access control, GRANT/REVOKE
│   │   ├── manage-navigation.md      # Navigation profiles, menus
│   │   ├── demo-data.md              # Database/import/demo data
│   │   ├── xpath-constraints.md      # XPath syntax in WHERE clauses
│   │   ├── database-connections.md   # External database connections
│   │   ├── test-microflows.md        # Test annotations and Docker setup
│   │   └── patterns-data-processing.md # Delta merge, batch processing
│   └── examples/                      # Example MDL scripts
├── mxcli                              # CLI executable (copied)
└── .devcontainer/                     # Dev container configuration
    ├── devcontainer.json
    └── Dockerfile
```

## Tool-Specific Files

### Claude Code

```
.claude/
├── settings.json          # Claude Code project settings
├── commands/              # Slash commands for Claude
│   └── mendix/           # Mendix-specific commands
└── lint-rules/            # Starlark lint rules
CLAUDE.md                  # Project context for Claude
```

### Cursor

```
.cursorrules               # Compact MDL reference and command guide
```

### Continue.dev

```
.continue/
└── config.json            # Custom commands and slash commands
```

### Windsurf

```
.windsurfrules             # MDL rules for Codeium's AI
```

### Aider

```
.aider.conf.yml            # YAML configuration for Aider
```

## Dev Container Contents

The `.devcontainer/` provides a sandboxed environment with:

| Component | Purpose |
|-----------|---------|
| **mxcli** | Mendix CLI binary |
| **MxBuild / mx** | Project validation and building (auto-downloaded on first use) |
| **JDK 21** (Adoptium) | Required by MxBuild |
| **Docker-in-Docker** | Running Mendix apps locally with `mxcli docker` |
| **Node.js** | Playwright testing support |
| **PostgreSQL client** | Database connectivity |
| **Claude Code** | AI coding assistant (auto-installed on container creation) |

Key paths inside the container:

```
~/.mxcli/mxbuild/{version}/modeler/mx    # mx check / mx build
~/.mxcli/runtime/{version}/               # Mendix runtime (auto-downloaded)
./mxcli                                    # Project-local mxcli binary
```

## Skills Overview

The skill files in `.ai-context/skills/` teach AI assistants how to generate correct MDL. Each skill covers a specific topic:

| Skill File | What It Teaches |
|-----------|----------------|
| `write-microflows.md` | Microflow syntax, common mistakes, validation checklist |
| `create-page.md` | Page/widget syntax reference |
| `overview-pages.md` | CRUD page patterns (data grids, search, buttons) |
| `generate-domain-model.md` | Entity, attribute, association, enumeration syntax |
| `manage-security.md` | Module roles, user roles, GRANT/REVOKE patterns |
| `demo-data.md` | Mendix ID system, association storage, data insertion |
| `check-syntax.md` | Pre-flight validation checklist before execution |

AI assistants are instructed to read the relevant skill file before generating MDL for that topic.
