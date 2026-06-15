# Module 11: Theming — Business Requirements

## Business Context

TechCorp's brand color is dark blue (`#1565C0`), but Mendix Atlas UI defaults to a purple tone.
The IT department wants the Helpdesk app to be visually consistent with the company's other systems — at the very least, the primary color should match.

In addition, the button corners are too rounded (Atlas defaults to 8px), and TechCorp's UI guidelines require a smaller radius (4px).

---

## User Stories

- As a product owner, I want the Helpdesk's primary color to be the company brand blue, so that it is consistent in style with the other systems
- As a UI designer, I want to make the button corner radius smaller, so that the interface looks more professional
- As a developer, I don't want to modify the Atlas source; instead I want to override CSS variables, so that there are no conflicts when upgrading Atlas

---

## Brand Requirements

| Element | Default Atlas | TechCorp Requirement |
|---------|--------------|----------------------|
| Primary color (buttons/links/highlights) | purple `#264AE5` | brand blue `#1565C0` |
| Primary color hover | `#1E3AB8` | `#0D47A1` |
| Button corner radius | `8px` | `4px` |
| Font | Atlas default | unchanged |

---

## Acceptance Criteria

- [ ] Primary action buttons appear in brand blue
- [ ] The navigation bar's primary color is brand blue
- [ ] The button corner radius is 4px
- [ ] The theme change does not affect mx check (pure CSS, no MDL changes)
