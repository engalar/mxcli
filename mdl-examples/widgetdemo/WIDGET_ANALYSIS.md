# WidgetDemo_Showcase Widget Analysis

Analysis of all widget instances in `WidgetDemo.WidgetDemo_Showcase`, classified by Mendix widget type.

## Summary

| Category | Count | Description |
|----------|-------|-------------|
| Native | 62 | Built-in Mendix platform widgets (`Forms$*`, `Pages$*`) |
| Pluggable (dedicated) | 9 | Pluggable widgets with specialized DESCRIBE output |
| Pluggable (generic) | 22 | Pluggable widgets using generic `PLUGGABLEWIDGET` output |
| Pluggable (misclassified) | 2 | Pluggable widgets rendered with native-style names |
| **Total** | **95** | |

## Native Widgets (62)

Standard Mendix platform widgets serialized as `Forms$*` / `Pages$*` BSON types.

| Widget Type | Count | Instances |
|-------------|-------|-----------|
| DYNAMICTEXT | 26 | tS1, txtH1–H6, txtPar, tS2, ctnL, ctnD, gbL, tS3, tS4, tTitle, tCode, tAmount, tS5, tS6, lvT, lvD, tS7, nHt, nLt, nSt, tS8, tS8d |
| CONTAINER | 9 | hdrS1–S8, ctnDemo |
| ACTIONBUTTON | 9 | btnDef, btnPri, btnSuc, btnWar, btnDng, btnNewDG, btnSv, btnCn |
| LAYOUTGRID | 3 | gridMain, gForm, gPlug |
| DATAVIEW | 2 | dvDetail, dvPluggable |
| TEXTBOX | 2 | txtTitle, txtCode |
| DATEPICKER | 2 | dpCreated, dpDue |
| GROUPBOX | 1 | gbDemo |
| TABCONTAINER | 1 | tabDemo (rendered as `Forms$TabControl` comment) |
| CHECKBOX | 1 | cbActive |
| TEXTAREA | 1 | taNotes |
| RADIOBUTTONS | 1 | rbStatus |
| DATAGRID | 1 | dgItems |
| LISTVIEW | 1 | lvItems |
| NAVIGATIONLIST | 1 | navQ |
| SNIPPETCALL | 1 | snCall |

Not counted as widgets (structural elements): ROW (24), COLUMN (varied), FOOTER (1), CONTROLBAR (1), FILTER (1), TEMPLATE (1), ITEM (3), DATAGRID COLUMN (5).

## Pluggable Widgets — Dedicated DESCRIBE (9)

Pluggable widgets recognized by `isKnownCustomWidgetType()` with specialized output format.

| DESCRIBE Name | Widget ID | Count | Instances |
|---------------|-----------|-------|-----------|
| GALLERY | `com.mendix.widget.web.gallery.Gallery` | 1 | galItems |
| COMBOBOX | `com.mendix.widget.web.combobox.Combobox` | 2 | cbPri, cbCat |
| TEXTFILTER | `com.mendix.widget.web.datagridtextfilter.DatagridTextFilter` | 1 | tfTitle |
| DROPDOWNFILTER | `com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter` | 1 | dfCat |
| NUMBERFILTER | `com.mendix.widget.web.datagridnumberfilter.DatagridNumberFilter` | 1 | nfAmt |
| DATEFILTER | `com.mendix.widget.web.datagriddatefilter.DatagridDateFilter` | 1 | dtfDate |
| DROPDOWNSORT | `com.mendix.widget.web.dropdownsort.DropdownSort` | 1 | ddSort1 |
| IMAGE | `com.mendix.widget.web.image.Image` | 1 | myImage1 |

## Pluggable Widgets — Generic PLUGGABLEWIDGET (22)

Pluggable widgets rendered with full widget ID via `extractExplicitProperties()`.

| Widget ID | Short Name | Count | Instances |
|-----------|------------|-------|-----------|
| `com.mendix.widget.custom.switch.Switch` | Switch | 1 | switch1 |
| `com.mendix.widget.custom.starrating.StarRating` | StarRating | 1 | rating1 |
| `com.mendix.widget.custom.slider.Slider` | Slider | 1 | slider1 |
| `com.mendix.widget.custom.RangeSlider.RangeSlider` | RangeSlider | 1 | rangeSlider1 |
| `com.mendix.widget.web.barcodescanner.BarcodeScanner` | BarcodeScanner | 1 | scanner1 |
| `com.mendix.widget.web.htmlelement.HTMLElement` | HTMLElement | 1 | htmlElem1 |
| `com.mendix.widget.custom.badge.Badge` | Badge | 1 | badge1 |
| `com.mendix.widget.custom.progressbar.ProgressBar` | ProgressBar | 1 | progBar1 |
| `com.mendix.widget.custom.progresscircle.ProgressCircle` | ProgressCircle | 1 | progCirc1 |
| `com.mendix.widget.web.accordion.Accordion` | Accordion | 1 | accordion1 |
| `com.mendix.widget.web.tooltip.Tooltip` | Tooltip | 1 | myTooltip1 |
| `com.mendix.widget.web.timeline.Timeline` | Timeline | 2 | timelineText, timelineCustom |
| `com.mendix.widget.web.videoplayer.VideoPlayer` | VideoPlayer | 1 | videoPlayer1 |
| `com.mendix.widget.web.popupmenu.PopupMenu` | PopupMenu | 1 | myPopupMenu1 |
| `com.mendix.widget.web.treenode.TreeNode` | TreeNode | 3 | treeText, treeNested, treeCustom |
| `com.mendix.widget.web.selectionhelper.SelectionHelper` | SelectionHelper | 1 | selHelper1 |
| `com.mendix.widget.web.languageselector.LanguageSelector` | LanguageSelector | 1 | langSel1 |
| `com.mendix.widget.web.accessibilityhelper.AccessibilityHelper` | AccessibilityHelper | 1 | accHelper1 |
| `com.mendix.widget.custom.Maps.Maps` | Maps | 1 | maps1 |
| `com.mendix.widget.web.areachart.AreaChart` | AreaChart | 1 | areaChart1 |

## Pluggable Widgets — Misclassified as Native (2)

These are pluggable widgets (`CustomWidgets$CustomWidget` in BSON) but DESCRIBE renders them with native-style short names instead of `PLUGGABLEWIDGET 'widget.id'` syntax. This is a DESCRIBE bug — the widget type mapping in `customWidgetTypeShortNames` produces a native-looking name.

| DESCRIBE Name | Widget ID | Count | Instances |
|---------------|-----------|-------|-----------|
| BADGEBUTTON | `com.mendix.widget.custom.badgebutton.BadgeButton` | 1 | badgeBtn1 |
| FIELDSET | `com.mendix.widget.web.fieldset.Fieldset` | 1 | fieldset1 |

## Widget Feature Coverage

| Feature | Widgets Demonstrating It |
|---------|--------------------------|
| DataSource (database) | galItems, dgItems, lvItems, timelineText, timelineCustom, treeText, treeNested, treeCustom, langSel1 |
| DataSource (parameter) | dvDetail, dvPluggable |
| Attribute binding | txtTitle, txtCode, dpCreated, dpDue, cbPri, cbCat, cbActive, taNotes, rbStatus, switch1, rating1, slider1, rangeSlider1, scanner1 |
| TextTemplate `{Param}` | tTitle, tCode, tAmount, lvT, lvD |
| Child widget slots | htmlElem1, fieldset1, myTooltip1, timelineCustom, myPopupMenu1, treeNested, treeCustom, accHelper1 |
| Explicit properties | All 22 generic PLUGGABLEWIDGET instances |
| Action binding | btnDef–btnDng, btnNewDG, btnSv, btnCn |
| Filter widgets | tfTitle, dfCat, nfAmt, dtfDate, ddSort1 |
| Selection | galItems (Single) |
| DesignProperties/Class/Style | hdrS1–S8, ctnDemo |
