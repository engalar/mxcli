# Module 10: Widget Development — Business Requirements

## Business Context

In the ticket list, the status (Open / In Progress / Resolved) is currently shown as plain text.
Support agents report that it is hard to quickly distinguish different statuses — they need **colored badges** to grasp ticket priority at a glance.

The built-in components of Mendix Atlas UI do not provide conditional color badges, so this requires developing a **custom Widget**.

---

## User Stories

- As a support agent, I want to see colored status badges in the ticket list (green = Resolved, blue = In Progress, yellow = Open, gray = Draft/Closed), so that I can tell a ticket's status at a glance
- As a developer, I want to use this Widget with a simple attribute binding, so that I don't have to write CSS class names

---

## Widget Specification

**Widget name:** TicketStatusBadge

**Properties:**
- `statusValue` (Enumeration: HD.TicketStatus): the ticket status

**Color mapping:**
- Draft → gray `#9E9E9E`
- Open → orange `#FF9800` (pending)
- InProgress → blue `#2196F3`
- Resolved → green `#4CAF50`
- Closed → blue-gray `#607D8B`

---

## Acceptance Criteria

- [ ] The Widget is packaged as a `.mpk` file that can be imported into Studio Pro
- [ ] Binding the Status property on a page displays a colored circular badge + text
- [ ] The Widget renders correctly in Mendix 11.x with no CE0463 error
