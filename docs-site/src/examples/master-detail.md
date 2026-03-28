# Master-Detail Page

A single page with a list on the left and a detail form on the right. Selecting an item in the list updates the form via the `SELECTION` data source binding -- no microflow needed.

```sql
CREATE PAGE CRM.Customer_MasterDetail (
  Title: 'Customers',
  Layout: Atlas_Core.Atlas_Default
) {
  LAYOUTGRID mainGrid {
    ROW row1 {
      -- Master list (left panel)
      COLUMN colMaster (DesktopWidth: 4) {
        DYNAMICTEXT heading (Content: 'Customers', RenderMode: H3)
        GALLERY customerList (DataSource: DATABASE CRM.Customer, Selection: Single) {
          TEMPLATE template1 {
            DYNAMICTEXT name (
              Content: '{1}',
              ContentParams: [{1} = Name],
              RenderMode: H4
            )
            DYNAMICTEXT email (
              Content: '{1}',
              ContentParams: [{1} = Email]
            )
          }
        }
      }

      -- Detail form (right panel, bound to gallery selection)
      COLUMN colDetail (DesktopWidth: 8) {
        DYNAMICTEXT detailHeading (Content: 'Details', RenderMode: H3)
        DATAVIEW customerDetail (DataSource: SELECTION customerList) {
          TEXTBOX txtName (Label: 'Name', Attribute: Name)
          TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
          TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
          TEXTAREA txtNotes (Label: 'Notes', Attribute: Notes)
          FOOTER footer1 {
            ACTIONBUTTON btnSave (
              Caption: 'Save',
              Action: SAVE_CHANGES,
              ButtonStyle: Success
            )
            ACTIONBUTTON btnCancel (Caption: 'Cancel', Action: CANCEL_CHANGES)
          }
        }
      }
    }
  }
};
/
```

The key pattern is `DataSource: SELECTION customerList` -- the data view automatically displays whichever item is selected in the gallery. No event handlers, no microflow calls, no state management.
