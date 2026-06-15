# Module 04: AI Collaboration Guide — Pages and UI

## Prerequisite

First run the reference implementations of modules 01–03.

## Atlas UI Component Quick Reference

| Need | Atlas Component | MDL Keyword |
|------|-----------------|-------------|
| List page | DataGrid | `datagrid` |
| Detail / form | DataView | `dataview` |
| Popup page | PopupLayout | `layout: Atlas_Core.PopupLayout` |
| 2-column layout | LayoutGrid | `layoutgrid ... row ... column (desktopwidth: 8)` |
| Button | ActionButton | `actionbutton ... buttonstyle: primary` |
| Text display | DynamicText | `dynamictext` |
| Text input | TextBox | `textbox` |
| Multi-line input | TextArea | `textarea` |

## Steps for Collaborating with Claude

```
Help me create the main Helpdesk pages in MDL:
1. HD.Ticket_Overview: ticket list (all tickets), using the Atlas_Default layout, displayed with a DataGrid,
   columns include Subject/Status/Priority/SLADueAt, with a text filter and a dropdown filter,
   with New Ticket and Open (open details) buttons
2. HD.Ticket_Detail: ticket details, using a 2-column layout (left 8, right 4), left DataView showing fields,
   action buttons on the right (Submit/Assign/Resolve/Reopen/Close), with a comment DataGrid at the bottom
3. HD.Ticket_NewEdit: new/edit form, Subject textbox + Description textarea + Save/Cancel
4. HD.MyTickets_Overview: my tickets list, datasource: microflow HD.DS_MyTickets,
   with New Ticket and Open buttons
5. HD.AddComment_Form: popup, input Content (textarea) and IsInternal (checkbox), Save button
```

## Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| `action: show_page` needs page params | Syntax: `show_page HD.Ticket_Detail (Ticket: $currentObject)` |
| Popup pages need PopupLayout | `layout: Atlas_Core.PopupLayout` |
| Page params format | `params: { $Ticket: HD.Ticket }` |
| DataView binding to a page param | `datasource: $Ticket` |
| Buttons inside the footer | `footer ftrActions { actionbutton ... }` |
