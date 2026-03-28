# Validation Pattern

Mendix uses a two-microflow pattern for form validation: a validation microflow checks fields and returns feedback, and an action microflow calls it before saving.

## The Validation Microflow

`VALIDATION FEEDBACK` attaches error messages to specific fields. The form highlights the field and displays the message.

```sql
CREATE MICROFLOW Sales.VAL_Order ($Order: Sales.Order)
RETURNS Boolean AS $IsValid
BEGIN
  DECLARE $IsValid Boolean = true;

  -- Required field
  IF $Order/OrderNumber = empty OR trim($Order/OrderNumber) = '' THEN
    SET $IsValid = false;
    VALIDATION FEEDBACK $Order/OrderNumber MESSAGE 'Order number is required';
  END IF;

  -- Date must be in the future
  IF $Order/DeliveryDate != empty AND $Order/DeliveryDate < [%CurrentDateTime%] THEN
    SET $IsValid = false;
    VALIDATION FEEDBACK $Order/DeliveryDate MESSAGE 'Delivery date must be in the future';
  END IF;

  -- Numeric range
  IF $Order/Quantity <= 0 THEN
    SET $IsValid = false;
    VALIDATION FEEDBACK $Order/Quantity MESSAGE 'Quantity must be greater than zero';
  END IF;

  IF $Order/Quantity > 10000 THEN
    SET $IsValid = false;
    VALIDATION FEEDBACK $Order/Quantity MESSAGE 'Maximum quantity is 10,000';
  END IF;

  -- Cross-field validation
  IF $Order/DiscountPercent > 0 AND $Order/ApprovedBy = empty THEN
    SET $IsValid = false;
    VALIDATION FEEDBACK $Order/ApprovedBy MESSAGE 'Discounted orders require approval';
  END IF;

  RETURN $IsValid;
END;
/
```

## The Action Microflow

The action microflow calls validation, and only saves if it passes:

```sql
CREATE MICROFLOW Sales.ACT_Order_Save ($Order: Sales.Order)
RETURNS Boolean AS $IsValid
BEGIN
  $IsValid = CALL MICROFLOW Sales.VAL_Order($param = $Order);

  IF $IsValid THEN
    COMMIT $Order;
    CLOSE PAGE;
  END IF;

  RETURN $IsValid;
END;
/
```

## Wiring It Up

The page's Save button calls the action microflow (not the validation microflow directly):

```sql
CREATE PAGE Sales.Order_Edit (
  Params: { $Order: Sales.Order },
  Title: 'Order',
  Layout: Atlas_Core.PopupLayout
) {
  LAYOUTGRID mainGrid {
    ROW row1 {
      COLUMN col1 (DesktopWidth: AutoFill) {
        DATAVIEW dv (DataSource: $Order) {
          TEXTBOX txtOrderNumber (Label: 'Order Number', Attribute: OrderNumber)
          DATEPICKER dpDelivery (Label: 'Delivery Date', Attribute: DeliveryDate)
          TEXTBOX txtQuantity (Label: 'Quantity', Attribute: Quantity)
          TEXTBOX txtDiscount (Label: 'Discount %', Attribute: DiscountPercent)
          FOOTER footer1 {
            ACTIONBUTTON btnSave (
              Caption: 'Save',
              Action: CALL Sales.ACT_Order_Save,
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
