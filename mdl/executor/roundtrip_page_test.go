// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"
)

// --- V3 Page Syntax Tests ---

// TestRoundtripPage_V3DataGridColumns tests that DataGrid columns are correctly created with V3 syntax.
// This is a regression test for the bug where user-defined columns were ignored and template columns were used.
func TestRoundtripPage_V3DataGridColumns(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// First create an entity to use with the DataGrid
	entityName := testModule + ".V3GridTestEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		FirstName: String(100),
		LastName: String(100),
		Email: String(200)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid using V3 syntax and custom columns
	pageName := testModule + ".V3DataGridColumnsPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'DataGrid Columns Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		layoutgrid mainGrid {
			row row1 {
				column col1 (DesktopWidth: 12) {
					datagrid dg (DataSource: database ` + entityName + `) {
						column colFirst (Attribute: FirstName, Caption: 'First Name')
						column colLast (Attribute: LastName, Caption: 'Last Name')
						column colEmail (Attribute: Email, Caption: 'Email Address')
					}
				}
			}
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify that the user-defined columns are present (not template columns)
	expectedColumns := []string{"FirstName", "LastName", "Email"}
	unexpectedColumns := []string{"Code", "Price", "Stock", "IsActive"} // Template columns (excluding Name which might be in entity)

	for _, col := range expectedColumns {
		if !strings.Contains(output, col) {
			t.Errorf("Expected column '%s' not found in describe output.\nOutput:\n%s", col, output)
		}
	}

	for _, col := range unexpectedColumns {
		if strings.Contains(output, col) {
			t.Errorf("Unexpected template column '%s' found in describe output - user columns may have been ignored.\nOutput:\n%s", col, output)
		}
	}

	// Also verify column captions
	if !strings.Contains(output, "First Name") {
		t.Error("Expected caption 'First Name' not found in output")
	}
	if !strings.Contains(output, "Last Name") {
		t.Error("Expected caption 'Last Name' not found in output")
	}
	if !strings.Contains(output, "Email Address") {
		t.Error("Expected caption 'Email Address' not found in output")
	}

	t.Logf("V3 DataGrid columns roundtrip successful:\n%s", output)
}

// TestRoundtripPage_V3DataGridNoColumns tests DataGrid without columns uses template defaults.
func TestRoundtripPage_V3DataGridNoColumns(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// First create an entity
	entityName := testModule + ".V3GridNoColEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Name: String(100)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid without explicit columns
	pageName := testModule + ".V3DataGridNoColumnsPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'DataGrid No Columns Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		layoutgrid mainGrid {
			row row1 {
				column col1 (DesktopWidth: 12) {
					datagrid dg (DataSource: database ` + entityName + `)
				}
			}
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid (no columns): %v", err)
	}

	// Describe the page - should have template columns
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// When no columns are specified, template columns should be used
	if !strings.Contains(output, "data GRID") && !strings.Contains(output, "datagrid") {
		t.Error("Expected datagrid in output")
	}

	t.Logf("V3 DataGrid (no columns) roundtrip successful:\n%s", output)
}

// TestRoundtripPage_V3DataGridWithOrderBy tests DataGrid with ORDER BY clause.
func TestRoundtripPage_V3DataGridWithOrderBy(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create entity
	entityName := testModule + ".V3GridOrderEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Name: String(100),
		Price: Decimal,
		Stock: Integer
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid with ORDER BY
	pageName := testModule + ".V3DataGridOrderByPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'DataGrid OrderBy Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		datagrid dg (DataSource: database from ` + entityName + ` sort by Name asc) {
			column colName (Attribute: Name, Caption: 'Product Name')
			column colPrice (Attribute: Price, Caption: 'Price')
			column colStock (Attribute: Stock, Caption: 'In Stock')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid OrderBy: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify columns are present
	if !strings.Contains(output, "Name") {
		t.Error("Expected column 'Name' in output")
	}
	if !strings.Contains(output, "Price") {
		t.Error("Expected column 'Price' in output")
	}
	if !strings.Contains(output, "Stock") {
		t.Error("Expected column 'Stock' in output")
	}

	t.Logf("V3 DataGrid with OrderBy roundtrip successful:\n%s", output)
}

// TestRoundtripPage_V3DataGridWithFilter tests DataGrid with WHERE filter.
func TestRoundtripPage_V3DataGridWithFilter(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create entity
	entityName := testModule + ".V3GridFilterEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Name: String(100),
		IsActive: Boolean default true,
		Stock: Integer
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid with WHERE filter
	pageName := testModule + ".V3DataGridFilterPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'DataGrid Filter Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		datagrid dg (
			DataSource: database from ` + entityName + ` where [IsActive = true] sort by Name asc
		) {
			column colName (Attribute: Name, Caption: 'Name')
			column colStock (Attribute: Stock, Caption: 'Stock')
			column colActive (Attribute: IsActive, Caption: 'Active')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid filter: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify columns are present
	expectedColumns := []string{"Name", "Stock", "IsActive"}
	for _, col := range expectedColumns {
		if !strings.Contains(output, col) {
			t.Errorf("Expected column '%s' in output", col)
		}
	}

	t.Logf("V3 DataGrid with filter roundtrip successful:\n%s", output)
}

// TestRoundtripPage_V3DataGridWithControlBar tests DataGrid with CONTROLBAR.
func TestRoundtripPage_V3DataGridWithControlBar(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()
	env.requireMinVersion(t, 11, 0) // CREATE PAGE with parameters requires 11.0+

	// Create entity
	entityName := testModule + ".V3GridControlEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Name: String(100),
		Code: String(50)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create a simple edit page for the control bar action
	editPageName := testModule + ".V3GridControlEditPage"
	env.registerCleanup("page", editPageName)

	createEditPageMDL := `create page ` + editPageName + ` (
		Params: { $Item: ` + entityName + ` },
		Title: 'Edit Item',
		Layout: Atlas_Core.PopupLayout
	) {
		dataview dv (DataSource: $Item) {
			textbox txtName (Label: 'Name', Attribute: Name)
		}
	}`

	if err := env.executeMDL(createEditPageMDL); err != nil {
		t.Fatalf("Failed to create edit page: %v", err)
	}

	// Create page with DataGrid with CONTROLBAR
	pageName := testModule + ".V3DataGridControlBarPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'DataGrid ControlBar Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		layoutgrid mainGrid {
			row row1 {
				column col1 (DesktopWidth: 12) {
					datagrid dg (DataSource: database ` + entityName + `) {
						controlbar controlBar1 {
							actionbutton btnNew (
								Caption: 'New Item',
								Action: create_object ` + entityName + ` then show_page ` + editPageName + `,
								ButtonStyle: Primary
							)
						}
						column colName (Attribute: Name, Caption: 'Name')
						column colCode (Attribute: Code, Caption: 'Code')
					}
				}
			}
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid ControlBar: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify columns are present (not template columns)
	if !strings.Contains(output, "Name") {
		t.Error("Expected column 'Name' in output")
	}
	if !strings.Contains(output, "Code") {
		t.Error("Expected column 'Code' in output")
	}

	// Verify template columns are NOT present
	if strings.Contains(output, "Price") || strings.Contains(output, "Stock") {
		t.Error("Unexpected template columns found - user columns may have been ignored")
	}

	t.Logf("V3 DataGrid with ControlBar roundtrip successful:\n%s", output)
}

// TestRoundtripPage_V3DataGridManyColumns tests DataGrid with many columns.
func TestRoundtripPage_V3DataGridManyColumns(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create entity with many attributes
	entityName := testModule + ".V3GridManyColEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Field1: String(100),
		Field2: String(100),
		Field3: String(100),
		Field4: Integer,
		Field5: Decimal,
		Field6: Boolean default false,
		Field7: DateTime
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid with many columns
	pageName := testModule + ".V3DataGridManyColumnsPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'DataGrid Many Columns Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		datagrid dg (DataSource: database ` + entityName + `) {
			column col1 (Attribute: Field1, Caption: 'First Field')
			column col2 (Attribute: Field2, Caption: 'Second Field')
			column col3 (Attribute: Field3, Caption: 'Third Field')
			column col4 (Attribute: Field4, Caption: 'Integer Field')
			column col5 (Attribute: Field5, Caption: 'Decimal Field')
			column col6 (Attribute: Field6, Caption: 'Boolean Field')
			column col7 (Attribute: Field7, Caption: 'DateTime Field')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with many DataGrid columns: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify all 7 columns are present
	expectedFields := []string{"Field1", "Field2", "Field3", "Field4", "Field5", "Field6", "Field7"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected column '%s' not found in output", field)
		}
	}

	// Verify template columns are NOT present
	templateColumns := []string{"Code", "Price", "Stock", "IsActive"}
	for _, col := range templateColumns {
		if strings.Contains(output, col) {
			t.Errorf("Unexpected template column '%s' found - user columns may have been ignored", col)
		}
	}

	// Count columns in output (rough check)
	columnCount := strings.Count(output, "column ")
	if columnCount < 7 {
		t.Errorf("Expected at least 7 columns, found %d", columnCount)
	}

	t.Logf("V3 DataGrid with many columns roundtrip successful:\n%s", output)
}

// TestRoundtripPage_V3DataGridComplexFilter tests DataGrid with complex WHERE clause.
func TestRoundtripPage_V3DataGridComplexFilter(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create entity
	entityName := testModule + ".V3GridComplexEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Name: String(100),
		Code: String(50),
		Price: Decimal,
		Stock: Integer,
		IsActive: Boolean default true
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid with complex WHERE and multiple OrderBy
	pageName := testModule + ".V3DataGridComplexFilterPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'Complex Filter Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		datagrid dg (
			DataSource: database from ` + entityName + ` where [IsActive = true and Price > 10] or [Stock < 5] sort by Name asc, Price desc
		) {
			column colName (Attribute: Name, Caption: 'Product')
			column colCode (Attribute: Code, Caption: 'SKU')
			column colPrice (Attribute: Price, Caption: 'Price')
			column colStock (Attribute: Stock, Caption: 'Available')
			column colActive (Attribute: IsActive, Caption: 'Active')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with complex filter: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify all columns are present
	expectedColumns := []string{"Name", "Code", "Price", "Stock", "IsActive"}
	for _, col := range expectedColumns {
		if !strings.Contains(output, col) {
			t.Errorf("Expected column '%s' not found in output", col)
		}
	}

	t.Logf("V3 DataGrid with complex filter roundtrip successful:\n%s", output)
}

// TestRoundtripPage_MicroflowButtonWithParams tests that action buttons with microflow calls
// and parameters are correctly round-tripped.
func TestRoundtripPage_MicroflowButtonWithParams(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()
	env.requireMinVersion(t, 11, 0) // CREATE PAGE with parameters requires 11.0+

	// First create an entity to use as the page parameter and microflow parameter
	entityName := testModule + ".MfButtonEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Name: String(100),
		Status: String(50)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create a microflow that accepts the entity as parameter
	mfName := testModule + ".MfButtonProcess"
	env.registerCleanup("microflow", mfName)

	createMfMDL := `create microflow ` + mfName + ` (
		$Product: ` + entityName + `
	)
	returns Boolean
	begin
		log info node 'Test' 'Processing item';
		return true;
	end;`

	if err := env.executeMDL(createMfMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	// Create page with action button that calls microflow with parameter
	pageName := testModule + ".MfButtonPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Params: { $Product: ` + entityName + ` },
		Title: 'Microflow Button Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		dataview dv (DataSource: $Product) {
			textbox txt (Label: 'Name', Attribute: Name)
			actionbutton btnProcess (Caption: 'Process', Action: microflow ` + mfName + `(Product: $Product))
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with microflow button: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify that the microflow call with parameter is in the output.
	// DESCRIBE uses `microflow` keyword (not `call_microflow`) and colon param syntax.
	if !strings.Contains(output, "microflow "+mfName) {
		t.Errorf("Expected `microflow %s` in describe output.\nOutput:\n%s", mfName, output)
	}
	if !strings.Contains(output, "Product: $Product") {
		t.Errorf("Expected 'Product: $Product' parameter mapping in describe output.\nOutput:\n%s", output)
	}

	t.Logf("Microflow button with params roundtrip successful:\n%s", output)
}

// TestRoundtripPage_MicroflowButtonWithCurrentObject tests that action buttons in list widgets
// with $currentObject parameter are correctly round-tripped.
func TestRoundtripPage_MicroflowButtonWithCurrentObject(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	// Create an entity for the DataGrid
	entityName := testModule + ".MfCurrentObjEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Name: String(100)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create a microflow that accepts the entity as parameter
	mfName := testModule + ".MfCurrentObjProcess"
	env.registerCleanup("microflow", mfName)

	createMfMDL := `create microflow ` + mfName + ` (
		$Target: ` + entityName + `
	)
	begin
		log info node 'Test' 'Processing: ' + $Target/Name;
	end;`

	if err := env.executeMDL(createMfMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	// Create page with DataGrid containing button that passes $currentObject
	pageName := testModule + ".MfCurrentObjPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'CurrentObject Button Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		datagrid dg (DataSource: database ` + entityName + `) {
			column colName (Attribute: Name, Caption: 'Name')
			column colActions (Attribute: Name, Caption: 'Actions', ShowContentAs: customContent) {
				actionbutton btnProcess (Caption: 'Process', Action: microflow ` + mfName + `(Target: $currentObject))
			}
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with currentObject microflow button: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify that the microflow call with $currentObject parameter is in the output.
	// DESCRIBE uses `microflow` keyword (matches actionExprV3 grammar), not `call_microflow`.
	if !strings.Contains(output, "microflow") {
		t.Errorf("Expected microflow keyword in describe output.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, mfName) {
		t.Errorf("Expected microflow name '%s' in describe output.\nOutput:\n%s", mfName, output)
	}
	if !strings.Contains(output, "Target = $currentObject") {
		t.Errorf("Expected 'Target = $currentObject' parameter mapping in describe output.\nOutput:\n%s", output)
	}

	t.Logf("Microflow button with $currentObject roundtrip successful:\n%s", output)
}

// TestRoundtripPage_DataViewAttributeShortNames tests that DESCRIBE outputs short attribute names
// (not fully qualified Module.Entity.Attribute) for widgets inside a DATAVIEW.
func TestRoundtripPage_DataViewAttributeShortNames(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()
	env.requireMinVersion(t, 11, 0) // CREATE PAGE with parameters requires 11.0+

	entityName := testModule + ".AttrShortNameEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		FirstName: String(100),
		Email: String(200),
		IsActive: Boolean default false,
		BirthDate: DateTime
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	pageName := testModule + ".AttrShortNamePage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Params: { $Item: ` + entityName + ` },
		Title: 'Short Attribute Names Test',
		Layout: Atlas_Core.PopupLayout
	) {
		dataview dv (DataSource: $Item) {
			textbox txtFirst (Label: 'First Name', Attribute: FirstName)
			textbox txtEmail (Label: 'Email', Attribute: Email)
			checkbox cbActive (Label: 'Active', Attribute: IsActive)
			datepicker dpBirth (Label: 'Birthday', Attribute: BirthDate)
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify attributes are short names, not qualified
	shortNames := []string{"Attribute: FirstName", "Attribute: Email", "Attribute: IsActive", "Attribute: BirthDate"}
	for _, expected := range shortNames {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected short attribute '%s' not found in output.\nOutput:\n%s", expected, output)
		}
	}

	// Verify qualified names are NOT present
	qualifiedPrefix := testModule + ".AttrShortNameEntity."
	if strings.Contains(output, qualifiedPrefix) {
		t.Errorf("Found qualified attribute prefix '%s' in output — should use short names.\nOutput:\n%s", qualifiedPrefix, output)
	}

	t.Logf("DataView short attribute names roundtrip successful:\n%s", output)
}

// TestRoundtripPage_DataGridAttributeShortNames tests that DataGrid column attributes
// use short names in DESCRIBE output.
func TestRoundtripPage_DataGridAttributeShortNames(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	entityName := testModule + ".GridAttrEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `create or modify persistent entity ` + entityName + ` (
		Name: String(100),
		Code: String(50),
		Price: Decimal
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	pageName := testModule + ".GridAttrPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `create page ` + pageName + ` (
		Title: 'Grid Attribute Names Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		datagrid dg (DataSource: database ` + entityName + `) {
			column colName (Attribute: Name, Caption: 'Name')
			column colCode (Attribute: Code, Caption: 'Code')
			column colPrice (Attribute: Price, Caption: 'Price')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	output, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify column attributes are short names
	shortNames := []string{"Attribute: Name", "Attribute: Code", "Attribute: Price"}
	for _, expected := range shortNames {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected short attribute '%s' not found in output.\nOutput:\n%s", expected, output)
		}
	}

	// Verify qualified names are NOT present
	qualifiedPrefix := testModule + ".GridAttrEntity."
	if strings.Contains(output, qualifiedPrefix) {
		t.Errorf("Found qualified attribute prefix '%s' in output — should use short names.\nOutput:\n%s", qualifiedPrefix, output)
	}

	t.Logf("DataGrid short attribute names roundtrip successful:\n%s", output)
}

// TestRoundtripPage_V3DataGridColumnFilter tests DataGrid columns with per-column
// filter widgets (textfilter, numberfilter, datefilter, dropdownfilter) inside columns.
// Regression coverage for Gap 2: filtertype property forwarding.
func TestRoundtripPage_V3DataGridColumnFilter(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	entityName := testModule + ".V3FilterEntity"
	env.registerCleanup("entity", entityName)

	if err := env.executeMDL(`create or modify persistent entity ` + entityName + ` (
		Name: String(100),
		Price: Decimal,
		OrderDate: DateTime,
		Status: Boolean default true
	);`); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	pageName := testModule + ".V3ColumnFilterPage"
	env.registerCleanup("page", pageName)

	if err := env.executeMDL(`create page ` + pageName + ` (
		Title: 'Filter Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		datagrid dg1 (DataSource: database ` + entityName + `) {
			column colName   (Attribute: Name,      Caption: 'Name')   { textfilter     fName   (FilterType: startsWith) }
			column colPrice  (Attribute: Price,     Caption: 'Price')  { numberfilter   fPrice  (FilterType: greater) }
			column colDate   (Attribute: OrderDate, Caption: 'Date')   { datefilter     fDate   (FilterType: between) }
			column colStatus (Attribute: Status,    Caption: 'Active') { dropdownfilter fStatus }
		}
	}`); err != nil {
		t.Fatalf("create page: %v", err)
	}

	out, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	for _, col := range []string{"Name", "Price", "Date", "Active"} {
		if !strings.Contains(out, col) {
			t.Errorf("expected column %q in describe output", col)
		}
	}
	// FilterType must roundtrip back into describe output (Gap 2).
	for _, ft := range []string{"FilterType: startsWith", "FilterType: greater", "FilterType: between"} {
		if !strings.Contains(out, ft) {
			t.Errorf("expected %q in describe output (FilterType forwarding broken)", ft)
		}
	}
	t.Logf("column filter roundtrip:\n%s", out)
}

// TestRoundtripPage_V3DataGridControlBarFilter tests DataGrid controlbar with filter
// widgets (textfilter, dropdownfilter). Regression coverage for Gap 1: buildWidgetBSON()
// must emit real filter widgets, not DivContainer placeholders.
func TestRoundtripPage_V3DataGridControlBarFilter(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	entityName := testModule + ".V3CtrlBarFilterEntity"
	env.registerCleanup("entity", entityName)

	if err := env.executeMDL(`create or modify persistent entity ` + entityName + ` (
		Name: String(100),
		Status: Boolean default true
	);`); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	pageName := testModule + ".V3CtrlBarFilterPage"
	env.registerCleanup("page", pageName)

	if err := env.executeMDL(`create page ` + pageName + ` (
		Title: 'ControlBar Filter Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		datagrid dg1 (DataSource: database ` + entityName + `) {
			controlbar cb1 {
				textfilter     fName   (Attributes: [` + entityName + `.Name])
				dropdownfilter fStatus (Attributes: [` + entityName + `.Status])
			}
			column colName   (Attribute: Name,   Caption: 'Name')
			column colStatus (Attribute: Status, Caption: 'Active')
		}
	}`); err != nil {
		t.Fatalf("create page: %v", err)
	}

	out, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("describe: %v", err)
	}
	if !strings.Contains(out, "Name") {
		t.Errorf("expected column Name in describe output")
	}
	// Gap 1: controlbar must emit real filter widgets, not container placeholders.
	if !strings.Contains(out, "textfilter") {
		t.Errorf("expected real textfilter widget in controlbar (Gap 1: buildWidgetBSON emits DivContainer)")
	}
	if !strings.Contains(out, "dropdownfilter") {
		t.Errorf("expected real dropdownfilter widget in controlbar (Gap 1: buildWidgetBSON emits DivContainer)")
	}
	if strings.Contains(out, "container fName") || strings.Contains(out, "container fStatus") {
		t.Errorf("controlbar widgets serialized as plain container — Gap 1 not fixed")
	}
	t.Logf("controlbar filter roundtrip:\n%s", out)
}

// TestRoundtripPage_GalleryDropdownFilterAttrBinding verifies that a dropdownfilter
// inside a gallery filter bar correctly binds the Attributes list to the widget's
// attr property (attrChoice = "linked"). Regression for: dropdownfilter.def.json used
// wrong attrChoice value ("custom") and non-existent property key ("attributes").
func TestRoundtripPage_GalleryDropdownFilterAttrBinding(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	enumName := testModule + ".DropFilterStatus"
	env.registerCleanup("enumeration", enumName)

	if err := env.executeMDL(`create or modify enumeration ` + enumName + ` (
		Active,
		Inactive
	);`); err != nil {
		t.Fatalf("create enumeration: %v", err)
	}

	entityName := testModule + ".GalleryDropFilterEntity"
	env.registerCleanup("entity", entityName)

	if err := env.executeMDL(`create or modify persistent entity ` + entityName + ` (
		Title: String(100),
		Status: ` + enumName + `
	);`); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	pageName := testModule + ".GalleryDropFilterPage"
	env.registerCleanup("page", pageName)

	if err := env.executeMDL(`create page ` + pageName + ` (
		Title: 'Gallery Drop Filter Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		gallery gal1 (DataSource: database ` + entityName + `, Selection: single) {
			filter filterBar {
				dropdownfilter fStatus (Attributes: [` + entityName + `.Status])
			}
			template tmpl1 {
				dynamictext txtTitle (content: '{1}', contentparams: [{1} = Title])
			}
		}
	}`); err != nil {
		t.Fatalf("create page: %v", err)
	}

	out, err := env.describeMDL(`describe page ` + pageName + `;`)
	if err != nil {
		t.Fatalf("describe: %v", err)
	}

	// The dropdownfilter must appear as a real filter widget, not a DivContainer.
	if !strings.Contains(out, "dropdownfilter") {
		t.Errorf("expected dropdownfilter in describe output, got:\n%s", out)
	}
	if strings.Contains(out, "container fStatus") {
		t.Errorf("dropdownfilter was serialized as plain container instead of a filter widget")
	}

	// The Attributes binding must be preserved in the BSON roundtrip.
	// describe should emit the attribute path (Attributes: [...]) so the filter is wired.
	// Bug: dropdownfilter.def.json had wrong propertyKey ("attributes" vs "attr") and
	// wrong attrChoice value ("custom" vs "linked"), causing the binding to be silently dropped.
	if !strings.Contains(out, "Attributes:") && !strings.Contains(out, "attributes:") {
		t.Errorf("Attributes binding lost in roundtrip — dropdownfilter.def.json has wrong propertyKey or attrChoice; got:\n%s", out)
	}
	t.Logf("gallery dropdown filter roundtrip:\n%s", out)
}
