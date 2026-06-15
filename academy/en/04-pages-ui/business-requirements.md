# Module 04: Pages and UI — Business Requirements

## Business Context

We have the data and the logic; now we need to let users see and act on it. A good interface multiplies an agent's efficiency, while a bad one drives people crazy. TechCorp's management requires the interface to be intuitive and clean, with important information visible at a glance.

---

## User Stories

### Ticket List (All Tickets)
- As an agent, I want to see a list of all tickets, filterable by status and priority, so that I can quickly find the tickets I need to handle
- As an agent, I want to see each ticket's status in the list (shown as a colored badge), so that I know the progress without opening the ticket
- As an agent, I want to click a ticket to open its details directly, rather than jumping around in new windows

### My Tickets (Customer View)
- As a customer, I want to see only the tickets that belong to me, not other people's issues
- As a customer, I want to create a new ticket directly on the list page, quick and convenient

### Ticket Details
- As anyone, I want to see all the important information on the ticket detail page (title, description, status, priority, due time)
- As an agent, I want to perform actions directly on the ticket detail page (submit, assign, resolve, reopen)
- As anyone, I want to see all comment records for the ticket
- As an agent, I want to add a comment on the detail page (internal note or external reply)

### New / Edit Ticket
- As a customer, I want to fill in the ticket title and description, and have the system set the default status automatically
- As an agent, I want to adjust the priority while editing

---

## Acceptance Criteria

- [ ] Ticket list: shows title, status (colored badge), priority, SLA due time, action buttons
- [ ] Ticket list: has a status dropdown filter and a title text search filter
- [ ] Ticket list: a "New Ticket" button leading to the new ticket form
- [ ] Ticket details: 2-column layout (info on the left, action buttons on the right)
- [ ] Ticket details: a comment list shown at the bottom
- [ ] Ticket details: an "Add Comment" button that opens a comment input box
- [ ] New form: title (text box), description (multi-line text), priority (dropdown)
- [ ] My Tickets: shows only tickets belonging to the current user, with a "New Ticket" button
