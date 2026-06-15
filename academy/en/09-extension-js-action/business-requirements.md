# Module 09: JS Action Extension — Business Requirements

## Business Context

When agents need to copy a ticket number to a user, they always have to manually select the text and press Ctrl+C, which is awkward.
They also want the browser to pop up a system notification when a new high-priority ticket arrives, instead of refreshing the page every few minutes.

These operations involve **browser APIs** (clipboard, notifications), which a Mendix microflow running on the server cannot call directly.
They need to be completed on the client through a **JavaScript Action**.

---

## User Stories

- As an agent, I want to copy the ticket title to the clipboard by clicking a single "Copy" button, saving me the manual copy
- As an agent, I want to receive a new-ticket notification in the browser background, so I do not have to keep watching the screen and refreshing
- As a system, I want to format a timestamp into a friendly display ("3 minutes ago") instead of the raw ISO format

---

## Acceptance Criteria

- [ ] "Copy ticket number" button: when clicked, the ticket title is copied to the clipboard and a "Copied" message is shown
- [ ] Browser notification: when a ticket's priority is Critical, a desktop notification pops up (requires user authorization)
- [ ] Relative time formatting: `formatRelativeTime(datetime)` returns "just now" / "3 minutes ago" / "2 hours ago", etc.
