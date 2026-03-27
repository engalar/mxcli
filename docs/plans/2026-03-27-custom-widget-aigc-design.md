# Custom Widget AIGC Design

## Goal

Provide a skill file that teaches Claude to autonomously create Mendix pluggable widgets from natural language descriptions, producing a ready-to-use `.mpk` file.

## Flow

```
User describes widget → Claude generates source files → npm install → npm run build → .mpk output
```

## Key Decisions

1. **No Yeoman generator** — direct file generation is more controllable for AI
2. **TypeScript + Function Components** — modern, type-safe, matches Mendix best practices
3. **`@mendix/pluggable-widgets-tools ^11.6`** — official build toolchain
4. **Skill-only deliverable** — no Go code changes needed, just `.claude/skills/mendix/create-custom-widget.md`

## Generated Project Structure

```
<widget-name>/
├── package.json
├── tsconfig.json
├── src/
│   ├── package.xml
│   ├── <Name>.xml           # widget property definitions
│   ├── <Name>.tsx            # entry point (Mendix API → React)
│   ├── <Name>.editorConfig.ts
│   ├── components/
│   │   └── <Component>.tsx   # actual React UI
│   └── ui/
│       └── <Name>.css
```

## Skill Content Outline

1. **Prerequisites** — Node.js >= 16, npm
2. **Property Type Reference** — all XML property types with examples
3. **File Templates** — exact content for each generated file
4. **Common Widget Patterns** — KPI card, chart wrapper, custom input, layout component
5. **Build & Integration** — npm install, npm run build, copy .mpk to widgets/
6. **Naming Conventions** — widget ID format, package path conventions
7. **Checklist** — pre-build validation steps

## Integration with mxcli

The skill is synced to user projects via `mxcli init` (already handled by the existing skill sync system in `reference/mendix-repl/templates/.claude/skills/`).
