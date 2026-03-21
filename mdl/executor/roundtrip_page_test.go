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

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
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

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'DataGrid Columns Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		LAYOUTGRID mainGrid {
			ROW row1 {
				COLUMN col1 (DesktopWidth: 12) {
					DATAGRID dg (DataSource: DATABASE ` + entityName + `) {
						COLUMN colFirst (Attribute: FirstName, Caption: 'First Name')
						COLUMN colLast (Attribute: LastName, Caption: 'Last Name')
						COLUMN colEmail (Attribute: Email, Caption: 'Email Address')
					}
				}
			}
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify that the user-defined columns are present (not template columns)
	expectedColumns := []string{"FirstName", "LastName", "Email"}
	unexpectedColumns := []string{"Code", "Price", "Stock", "IsActive"} // Template columns (excluding Name which might be in entity)

	for _, col := range expectedColumns {
		if !strings.Contains(output, col) {
			t.Errorf("Expected column '%s' not found in DESCRIBE output.\nOutput:\n%s", col, output)
		}
	}

	for _, col := range unexpectedColumns {
		if strings.Contains(output, col) {
			t.Errorf("Unexpected template column '%s' found in DESCRIBE output - user columns may have been ignored.\nOutput:\n%s", col, output)
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

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid without explicit columns
	pageName := testModule + ".V3DataGridNoColumnsPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'DataGrid No Columns Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		LAYOUTGRID mainGrid {
			ROW row1 {
				COLUMN col1 (DesktopWidth: 12) {
					DATAGRID dg (DataSource: DATABASE ` + entityName + `)
				}
			}
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid (no columns): %v", err)
	}

	// Describe the page - should have template columns
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// When no columns are specified, template columns should be used
	if !strings.Contains(output, "DATA GRID") && !strings.Contains(output, "DATAGRID") {
		t.Error("Expected DATAGRID in output")
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

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
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

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'DataGrid OrderBy Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		DATAGRID dg (DataSource: DATABASE FROM ` + entityName + ` SORT BY Name ASC) {
			COLUMN colName (Attribute: Name, Caption: 'Product Name')
			COLUMN colPrice (Attribute: Price, Caption: 'Price')
			COLUMN colStock (Attribute: Stock, Caption: 'In Stock')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid OrderBy: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
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

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100),
		IsActive: Boolean DEFAULT true,
		Stock: Integer
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid with WHERE filter
	pageName := testModule + ".V3DataGridFilterPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'DataGrid Filter Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		DATAGRID dg (
			DataSource: DATABASE FROM ` + entityName + ` WHERE [IsActive = true] SORT BY Name ASC
		) {
			COLUMN colName (Attribute: Name, Caption: 'Name')
			COLUMN colStock (Attribute: Stock, Caption: 'Stock')
			COLUMN colActive (Attribute: IsActive, Caption: 'Active')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid filter: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
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

	// Create entity
	entityName := testModule + ".V3GridControlEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100),
		Code: String(50)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create a simple edit page for the control bar action
	editPageName := testModule + ".V3GridControlEditPage"
	env.registerCleanup("page", editPageName)

	createEditPageMDL := `CREATE PAGE ` + editPageName + ` (
		Params: { $Item: ` + entityName + ` },
		Title: 'Edit Item',
		Layout: Atlas_Core.PopupLayout
	) {
		DATAVIEW dv (DataSource: $Item) {
			TEXTBOX txtName (Label: 'Name', Attribute: Name)
		}
	}`

	if err := env.executeMDL(createEditPageMDL); err != nil {
		t.Fatalf("Failed to create edit page: %v", err)
	}

	// Create page with DataGrid with CONTROLBAR
	pageName := testModule + ".V3DataGridControlBarPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'DataGrid ControlBar Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		LAYOUTGRID mainGrid {
			ROW row1 {
				COLUMN col1 (DesktopWidth: 12) {
					DATAGRID dg (DataSource: DATABASE ` + entityName + `) {
						CONTROLBAR controlBar1 {
							ACTIONBUTTON btnNew (
								Caption: 'New Item',
								Action: CREATE_OBJECT ` + entityName + ` THEN SHOW_PAGE ` + editPageName + `,
								ButtonStyle: Primary
							)
						}
						COLUMN colName (Attribute: Name, Caption: 'Name')
						COLUMN colCode (Attribute: Code, Caption: 'Code')
					}
				}
			}
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with V3 DataGrid ControlBar: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
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

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Field1: String(100),
		Field2: String(100),
		Field3: String(100),
		Field4: Integer,
		Field5: Decimal,
		Field6: Boolean DEFAULT false,
		Field7: DateTime
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid with many columns
	pageName := testModule + ".V3DataGridManyColumnsPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'DataGrid Many Columns Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		DATAGRID dg (DataSource: DATABASE ` + entityName + `) {
			COLUMN col1 (Attribute: Field1, Caption: 'First Field')
			COLUMN col2 (Attribute: Field2, Caption: 'Second Field')
			COLUMN col3 (Attribute: Field3, Caption: 'Third Field')
			COLUMN col4 (Attribute: Field4, Caption: 'Integer Field')
			COLUMN col5 (Attribute: Field5, Caption: 'Decimal Field')
			COLUMN col6 (Attribute: Field6, Caption: 'Boolean Field')
			COLUMN col7 (Attribute: Field7, Caption: 'DateTime Field')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with many DataGrid columns: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
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
	columnCount := strings.Count(output, "COLUMN ")
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

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100),
		Code: String(50),
		Price: Decimal,
		Stock: Integer,
		IsActive: Boolean DEFAULT true
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create page with DataGrid with complex WHERE and multiple OrderBy
	pageName := testModule + ".V3DataGridComplexFilterPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'Complex Filter Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		DATAGRID dg (
			DataSource: DATABASE FROM ` + entityName + ` WHERE [IsActive = true AND Price > 10] OR [Stock < 5] SORT BY Name ASC, Price DESC
		) {
			COLUMN colName (Attribute: Name, Caption: 'Product')
			COLUMN colCode (Attribute: Code, Caption: 'SKU')
			COLUMN colPrice (Attribute: Price, Caption: 'Price')
			COLUMN colStock (Attribute: Stock, Caption: 'Available')
			COLUMN colActive (Attribute: IsActive, Caption: 'Active')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with complex filter: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
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

	// First create an entity to use as the page parameter and microflow parameter
	entityName := testModule + ".MfButtonEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100),
		Status: String(50)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create a microflow that accepts the entity as parameter
	mfName := testModule + ".MfButtonProcess"
	env.registerCleanup("microflow", mfName)

	createMfMDL := `CREATE MICROFLOW ` + mfName + ` (
		$Product: ` + entityName + `
	)
	RETURNS Boolean
	BEGIN
		LOG INFO NODE 'Test' 'Processing item';
		RETURN true;
	END;`

	if err := env.executeMDL(createMfMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	// Create page with action button that calls microflow with parameter
	pageName := testModule + ".MfButtonPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Params: { $Product: ` + entityName + ` },
		Title: 'Microflow Button Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		DATAVIEW dv (DataSource: $Product) {
			TEXTBOX txt (Label: 'Name', Attribute: Name)
			ACTIONBUTTON btnProcess (Caption: 'Process', Action: MICROFLOW ` + mfName + `(Product: $Product))
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with microflow button: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify that the microflow call with parameter is in the output
	if !strings.Contains(output, "CALL_MICROFLOW") {
		t.Errorf("Expected CALL_MICROFLOW in DESCRIBE output.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, mfName) {
		t.Errorf("Expected microflow name '%s' in DESCRIBE output.\nOutput:\n%s", mfName, output)
	}
	if !strings.Contains(output, "Product = $Product") {
		t.Errorf("Expected 'Product = $Product' parameter mapping in DESCRIBE output.\nOutput:\n%s", output)
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

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100)
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Create a microflow that accepts the entity as parameter
	mfName := testModule + ".MfCurrentObjProcess"
	env.registerCleanup("microflow", mfName)

	createMfMDL := `CREATE MICROFLOW ` + mfName + ` (
		$Target: ` + entityName + `
	)
	BEGIN
		LOG INFO NODE 'Test' 'Processing: ' + $Target/Name;
	END;`

	if err := env.executeMDL(createMfMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	// Create page with DataGrid containing button that passes $currentObject
	pageName := testModule + ".MfCurrentObjPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'CurrentObject Button Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		DATAGRID dg (DataSource: DATABASE ` + entityName + `) {
			COLUMN colName (Attribute: Name, Caption: 'Name')
			COLUMN colActions (Attribute: Name, Caption: 'Actions', ShowContentAs: customContent) {
				ACTIONBUTTON btnProcess (Caption: 'Process', Action: MICROFLOW ` + mfName + `(Target: $currentObject))
			}
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page with currentObject microflow button: %v", err)
	}

	// Describe the page
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	// Verify that the microflow call with $currentObject parameter is in the output
	if !strings.Contains(output, "CALL_MICROFLOW") {
		t.Errorf("Expected CALL_MICROFLOW in DESCRIBE output.\nOutput:\n%s", output)
	}
	if !strings.Contains(output, mfName) {
		t.Errorf("Expected microflow name '%s' in DESCRIBE output.\nOutput:\n%s", mfName, output)
	}
	if !strings.Contains(output, "Target = $currentObject") {
		t.Errorf("Expected 'Target = $currentObject' parameter mapping in DESCRIBE output.\nOutput:\n%s", output)
	}

	t.Logf("Microflow button with $currentObject roundtrip successful:\n%s", output)
}

// TestRoundtripPage_DataViewAttributeShortNames tests that DESCRIBE outputs short attribute names
// (not fully qualified Module.Entity.Attribute) for widgets inside a DATAVIEW.
func TestRoundtripPage_DataViewAttributeShortNames(t *testing.T) {
	env := setupTestEnv(t)
	defer env.teardown()

	entityName := testModule + ".AttrShortNameEntity"
	env.registerCleanup("entity", entityName)

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		FirstName: String(100),
		Email: String(200),
		IsActive: Boolean DEFAULT false,
		BirthDate: DateTime
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	pageName := testModule + ".AttrShortNamePage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Params: { $Item: ` + entityName + ` },
		Title: 'Short Attribute Names Test',
		Layout: Atlas_Core.PopupLayout
	) {
		DATAVIEW dv (DataSource: $Item) {
			TEXTBOX txtFirst (Label: 'First Name', Attribute: FirstName)
			TEXTBOX txtEmail (Label: 'Email', Attribute: Email)
			CHECKBOX cbActive (Label: 'Active', Attribute: IsActive)
			DATEPICKER dpBirth (Label: 'Birthday', Attribute: BirthDate)
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
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

	createEntityMDL := `CREATE OR MODIFY PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100),
		Code: String(50),
		Price: Decimal
	);`

	if err := env.executeMDL(createEntityMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	pageName := testModule + ".GridAttrPage"
	env.registerCleanup("page", pageName)

	createPageMDL := `CREATE PAGE ` + pageName + ` (
		Title: 'Grid Attribute Names Test',
		Layout: Atlas_Core.Atlas_Default
	) {
		DATAGRID dg (DataSource: DATABASE ` + entityName + `) {
			COLUMN colName (Attribute: Name, Caption: 'Name')
			COLUMN colCode (Attribute: Code, Caption: 'Code')
			COLUMN colPrice (Attribute: Price, Caption: 'Price')
		}
	}`

	if err := env.executeMDL(createPageMDL); err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
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
