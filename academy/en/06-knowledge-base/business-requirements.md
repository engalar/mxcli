# Module 06: Knowledge Base Module — Business Requirements

## Business Context

The IT support team discovered that 70% of ticket issues are repetitive — the same VPN problem, the same password reset procedure, asked by someone every week.
If these solutions could be organized into articles so customers can find answers themselves, the agents' workload could be cut in half.

This is the value of a **knowledge base**: let knowledge accumulate, and make self-service possible.

---

## User Stories

### Customer (Self-Service Lookup)
- As a customer, I want to search knowledge base articles by keyword, so I can find answers myself before contacting an agent
- As a customer, I can only see officially published articles, not drafts or archived content

### Agent (Knowledge Capture)
- As an agent, I want to create a draft article to capture a solution, then publish it for everyone to see
- As an agent, I want to archive outdated articles to keep the knowledge base tidy
- As an agent, I want to sort articles into different categories (such as "Account Issues", "Network Connectivity") for easier lookup

### Knowledge Base Structure
- As a system, an article must belong to a category
- As a system, a category can have subcategories (such as "Network Issues > VPN > VPN Connection Failure")
- As a system, an article can have multiple tags (such as "howto", "faq") for aggregated queries

---

## Article Lifecycle

```
Draft ──[Publish]──► Published ──[Archive]──► Archived
```

- Publish: content must be non-empty; the system records the publish time
- Archive: only published articles can be archived

---

## Acceptance Criteria

- [ ] The knowledge base has a category tree (a category can have a parent category)
- [ ] An article belongs to one category and can have multiple tags (many-to-many)
- [ ] Customers only see "Published" articles (filtered at the database level)
- [ ] Publishing with empty content → an error message is shown
- [ ] Archiving a non-"Published" article → silent failure (no error, simply returns false)
- [ ] The knowledge base home page lists all published articles with a title search filter
- [ ] The article detail page has "Publish" and "Archive" buttons (visible to agents/admins)
