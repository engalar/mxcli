# Snippets

Snippets are reusable page fragments that can be embedded in multiple pages. They allow you to define a widget tree once and include it wherever needed, promoting consistency and reducing duplication.

## CREATE SNIPPET

```sql
CREATE [OR REPLACE] SNIPPET <Module>.<Name>
(
  [Params: { $Param: Module.Entity [, ...] },]
  [Folder: '<path>']
)
{
  <widget-tree>
}
```

### Basic Snippet

A snippet without parameters:

```sql
CREATE SNIPPET MyModule.Footer
(
  Folder: 'Snippets'
)
{
  CONTAINER cFooter (Class: 'app-footer') {
    DYNAMICTEXT txtCopyright (Content: '2024 My Company. All rights reserved.')
  }
}
```

### Snippet with Parameters

Snippets can accept entity parameters, similar to pages:

```sql
CREATE SNIPPET MyModule.CustomerCard
(
  Params: { $Customer: MyModule.Customer }
)
{
  CONTAINER cCard (Class: 'card') {
    DATAVIEW dvCustomer (DataSource: $Customer) {
      DYNAMICTEXT txtName (Content: '{1}', Attribute: Name)
      DYNAMICTEXT txtEmail (Content: '{1}', Attribute: Email)
      ACTIONBUTTON btnEdit (
        Caption: 'Edit',
        Action: PAGE MyModule.Customer_Edit,
        ButtonStyle: Primary
      )
    }
  }
}
```

## Using Snippets in Pages

Embed a snippet in a page using the `SNIPPETCALL` widget:

```sql
CREATE PAGE MyModule.Home
(
  Title: 'Home',
  Layout: Atlas_Core.Atlas_Default
)
{
  CONTAINER cMain {
    SNIPPETCALL scFooter (Snippet: MyModule.Footer)
  }
}
```

## Inspecting Snippets

```sql
-- List all snippets in a module
SHOW SNIPPETS IN MyModule;

-- View full MDL definition
DESCRIBE SNIPPET MyModule.CustomerCard;
```

## DROP SNIPPET

```sql
DROP SNIPPET MyModule.Footer;
```

## ALTER SNIPPET

Snippets support the same in-place modification operations as pages. See [ALTER PAGE / ALTER SNIPPET](./alter-page.md):

```sql
ALTER SNIPPET MyModule.CustomerCard {
  SET Caption = 'View Details' ON btnEdit;
  INSERT AFTER txtEmail {
    DYNAMICTEXT txtPhone (Content: '{1}', Attribute: Phone)
  }
};
```

## See Also

- [Pages](./pages.md) -- page overview
- [Widget Types](./widget-types.md) -- available widgets including SNIPPETCALL
- [ALTER PAGE](./alter-page.md) -- modifying snippets and pages in-place
- [Common Patterns](./page-patterns.md) -- patterns that use snippets
