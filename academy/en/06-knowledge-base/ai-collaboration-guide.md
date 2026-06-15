# Module 06: AI Collaboration Guide — Knowledge Base Module

## Prerequisites

First run the reference implementation for modules 01–05 (or run files 01–05 of the capstone reference implementation).

## New MDL Concepts in This Module

| Concept | Syntax Example |
|---------|----------------|
| Self-referencing association (category tree) | `from KB.Category to KB.Category` |
| Many-to-many junction table | Create an empty entity KB.ArticleTag, with two associations pointing to each side |
| unique constraint | `Name: string(100) not null unique` |
| integer attribute | `ViewCount: integer default 0` |

## Steps to Collaborate with Claude

### Step 1: Have Claude Design the KB Domain Model

```
Read academy/en/06-knowledge-base/business-requirements.md and help me implement the knowledge base domain model in MDL:
- KB module
- KB.ArticleStatus enumeration (Draft/Published/Archived)
- KB.Category entity (Name, Description, self-referencing parent-category association)
- KB.Tag entity (Name, unique)
- KB.Article entity (Title, Content, Status, PublishedAt, ViewCount)
  - association KB.Article → KB.Category
- KB.ArticleTag junction table (many-to-many: Article ↔ Tag)
```

### Step 2: Publish and Archive Microflows

```
Help me implement two microflows:
- KB.ACT_Article_Publish: validate Content is non-empty, Status Draft → Published, record PublishedAt
- KB.ACT_Article_Archive: verify the current status is Published, Status → Archived
```

### Step 3: Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| Self-referencing association naming | Use KB.Category_Parent (not KB.Category_Category) to avoid ambiguity |
| Many-to-many junction table | The KB.ArticleTag entity itself has no attributes, only two associations |
| XPath string filtering | `where '[Status = ''Published'']'` (single quotes must be doubled inside XPath) |
| PublishedAt is empty before publishing | The publish microflow assigns it with `PublishedAt = '[%CurrentDateTime%]'` |
| module role does not support or modify | Use `create module role KB.Reader;` (you cannot write `create or modify module role`) |
