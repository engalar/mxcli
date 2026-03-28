# CRM Module

A complete customer management feature: domain model, validation, CRUD pages, and security -- all in one script.

## Domain Model

```sql
-- Enumerations first (referenced by entities)
CREATE ENUMERATION CRM.CustomerStatus (
  Active 'Active',
  Inactive 'Inactive',
  Suspended 'Suspended'
);

CREATE ENUMERATION CRM.ContactType (
  Email 'Email',
  Phone 'Phone',
  Visit 'Visit'
);

-- Entities with per-attribute documentation
/** Customer master data */
@Position(100, 100)
CREATE PERSISTENT ENTITY CRM.Customer (
  /** Auto-generated unique identifier */
  CustomerId: AutoNumber NOT NULL UNIQUE DEFAULT 1,
  /** Full legal name */
  Name: String(200) NOT NULL ERROR 'Customer name is required',
  /** Primary contact email */
  Email: String(200) UNIQUE ERROR 'Email already exists',
  /** Phone number in international format */
  Phone: String(50),
  /** Current account balance */
  Balance: Decimal DEFAULT 0,
  /** Whether the account is active */
  IsActive: Boolean DEFAULT TRUE,
  /** Current lifecycle status */
  Status: Enumeration(CRM.CustomerStatus) DEFAULT 'Active',
  /** Free-form notes about this customer */
  Notes: String(unlimited)
)
INDEX (Name)
INDEX (Email);
/

/** Record of a customer interaction */
@Position(400, 100)
CREATE PERSISTENT ENTITY CRM.ContactLog (
  /** Date and time of the interaction */
  ContactDate: DateTime NOT NULL,
  /** Type of interaction */
  Type: Enumeration(CRM.ContactType) DEFAULT 'Email',
  /** Summary of what was discussed */
  Summary: String(2000) NOT NULL ERROR 'Summary is required',
  /** Follow-up needed? */
  FollowUpRequired: Boolean DEFAULT FALSE
);
/

-- Associations
CREATE ASSOCIATION CRM.ContactLog_Customer
  FROM CRM.ContactLog TO CRM.Customer
  TYPE Reference OWNER Default;
/
```

## Validation Microflow

The two-microflow pattern: a validation microflow returns field-level feedback, and an action microflow calls it before saving.

```sql
CREATE MICROFLOW CRM.VAL_Customer ($Customer: CRM.Customer)
RETURNS Boolean AS $IsValid
BEGIN
  DECLARE $IsValid Boolean = true;

  IF trim($Customer/Name) = '' THEN
    SET $IsValid = false;
    VALIDATION FEEDBACK $Customer/Name MESSAGE 'Name cannot be empty';
  END IF;

  IF $Customer/Email != empty AND NOT contains($Customer/Email, '@') THEN
    SET $IsValid = false;
    VALIDATION FEEDBACK $Customer/Email MESSAGE 'Enter a valid email address';
  END IF;

  IF $Customer/Balance < 0 THEN
    SET $IsValid = false;
    VALIDATION FEEDBACK $Customer/Balance MESSAGE 'Balance cannot be negative';
  END IF;

  RETURN $IsValid;
END;
/

CREATE MICROFLOW CRM.ACT_Customer_Save ($Customer: CRM.Customer)
RETURNS Boolean AS $IsValid
BEGIN
  $IsValid = CALL MICROFLOW CRM.VAL_Customer($param = $Customer);

  IF $IsValid THEN
    COMMIT $Customer;
    CLOSE PAGE;
  END IF;

  RETURN $IsValid;
END;
/
```

## Pages

```sql
-- Overview page with data grid
CREATE PAGE CRM.Customer_Overview (
  Title: 'Customers',
  Layout: Atlas_Core.Atlas_Default
) {
  DATAGRID2 ON CRM.Customer (
    COLUMN Name { Caption: 'Name' }
    COLUMN Email { Caption: 'Email' }
    COLUMN Phone { Caption: 'Phone' }
    COLUMN Status { Caption: 'Status' }
    COLUMN IsActive { Caption: 'Active' }
    SEARCH ON Name, Email
    BUTTON 'New' CALL CRM.Customer_NewEdit
    BUTTON 'Edit' CALL CRM.Customer_NewEdit
    BUTTON 'Delete' CALL CONFIRM DELETE
  )
};
/

-- NewEdit page with validation
CREATE PAGE CRM.Customer_NewEdit (
  Params: { $Customer: CRM.Customer },
  Title: 'Customer',
  Layout: Atlas_Core.PopupLayout
) {
  LAYOUTGRID mainGrid {
    ROW row1 {
      COLUMN col1 (DesktopWidth: AutoFill) {
        DATAVIEW dataView1 (DataSource: $Customer) {
          TEXTBOX txtName (Label: 'Name', Attribute: Name)
          TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
          TEXTBOX txtPhone (Label: 'Phone', Attribute: Phone)
          TEXTAREA txtNotes (Label: 'Notes', Attribute: Notes)
          FOOTER footer1 {
            ACTIONBUTTON btnSave (
              Caption: 'Save',
              Action: CALL CRM.ACT_Customer_Save,
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

## Security

```sql
-- Module roles
CREATE MODULE ROLE CRM.User;
CREATE MODULE ROLE CRM.Admin DESCRIPTION 'Full customer management access';

-- Entity access
GRANT CRM.Admin ON CRM.Customer (CREATE, DELETE, READ *, WRITE *);
GRANT CRM.User ON CRM.Customer (CREATE, READ *, WRITE *)
  WHERE '[IsActive = true]';

GRANT CRM.Admin ON CRM.ContactLog (CREATE, DELETE, READ *, WRITE *);
GRANT CRM.User ON CRM.ContactLog (CREATE, READ *, WRITE *);

-- Document access
GRANT EXECUTE ON MICROFLOW CRM.ACT_Customer_Save TO CRM.User;
GRANT VIEW ON PAGE CRM.Customer_Overview TO CRM.User;
GRANT VIEW ON PAGE CRM.Customer_NewEdit TO CRM.User;

-- User roles
CREATE OR MODIFY USER ROLE CRMUser (System.User, CRM.User);
CREATE OR MODIFY USER ROLE CRMAdmin (System.User, CRM.Admin);

-- Demo users for testing
CREATE OR MODIFY DEMO USER 'crm_user' PASSWORD 'Password1!' (CRMUser);
CREATE OR MODIFY DEMO USER 'crm_admin' PASSWORD 'Password1!' (CRMAdmin);
ALTER PROJECT SECURITY DEMO USERS ON;
```
