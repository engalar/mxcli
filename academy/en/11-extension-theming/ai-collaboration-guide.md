# Module 11: AI Collaboration Guide — Theming

## Approach: CSS Variable Overrides (no Studio Pro required)

Atlas UI uses CSS custom properties (CSS Variables) to control colors, corner radii, and other visual parameters.
By overriding these variables in a custom CSS file, you can achieve theming without touching the Atlas source.

## Three Implementation Approaches

### Approach A: CSS variable override (recommended, simplest)

1. In Studio Pro: App → Styling → find the custom CSS location (usually `theme/web/custom-variables.css`)
2. Paste the contents of `theme/helpdesk-theme.css` into it
3. Re-run the project

### Approach B: SCSS variables (requires the Atlas SCSS toolchain)

```bash
# In Studio Pro: App → Styling → export the Atlas SCSS source
# Modify the color variables in _variables.scss
# Recompile: npm run build (Studio Pro triggers this automatically)
```

### Approach C: Generate CSS in collaboration with Claude

```
Help me generate theme override CSS for Mendix Atlas UI:
- Primary color: #1565C0 (brand blue)
- Primary color hover: #0D47A1
- Button corner radius: 4px
- Based on the CSS variable naming conventions of Atlas UI
```

## Placement

| Studio Pro Version | File Location |
|--------------------|---------------|
| Mendix 9.x     | `theme/web/custom-variables.css` |
| Mendix 10/11.x | `theme/web/main.css` (append at the end of the file) |

## Validation

No mx check needed — CSS changes do not affect the MPR structure.
Run the app in a browser and verify the button color and corner radius are correct.
