// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/visitor"
)

// --- Security Roundtrip Tests ---

// securityTestModule uses MyFirstModule which has module security.
const securityTestModule = "MyFirstModule"

// securityTestEnv wraps testEnv with module role setup/teardown.
type securityTestEnv struct {
	*testEnv
	roles []string // qualified role names to clean up
}

// setupSecurityTestEnv creates module roles for security tests.
func setupSecurityTestEnv(t *testing.T) *securityTestEnv {
	t.Helper()
	env := setupTestEnv(t)
	senv := &securityTestEnv{testEnv: env}

	// Create two module roles for testing
	for _, role := range []string{"SecTestAdmin", "SecTestViewer"} {
		mdl := "CREATE MODULE ROLE " + securityTestModule + "." + role + ";"
		if err := env.executeMDL(mdl); err != nil {
			// Ignore if already exists
			if !strings.Contains(err.Error(), "already exists") {
				t.Fatalf("Failed to create module role %s: %v", role, err)
			}
		}
		senv.roles = append(senv.roles, securityTestModule+"."+role)
	}

	return senv
}

// teardown disconnects from the project. No cleanup needed since each test
// uses a fresh project copy that is automatically deleted.
func (s *securityTestEnv) teardown() {
	s.testEnv.teardown()
}

func TestRoundtripSecurity_EntityAccessGrant(t *testing.T) {
	env := setupSecurityTestEnv(t)
	defer env.teardown()

	entityName := securityTestModule + ".SecTestEntity"
	env.registerCleanup("entity", entityName)

	// Create entity
	createMDL := `CREATE PERSISTENT ENTITY ` + entityName + ` (
		Name: String(100),
		Email: String(200)
	);`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Grant access
	grantMDL := `GRANT ` + securityTestModule + `.SecTestAdmin ON ` + entityName + ` (CREATE, DELETE, READ *, WRITE *);`
	if err := env.executeMDL(grantMDL); err != nil {
		t.Fatalf("Failed to grant entity access: %v", err)
	}

	// Describe and verify
	output, err := env.describeMDL(`DESCRIBE ENTITY ` + entityName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe entity: %v", err)
	}

	// Verify GRANT line is present
	if !strings.Contains(output, "GRANT") {
		t.Errorf("Expected GRANT statement in DESCRIBE output.\nActual:\n%s", output)
	}
	if !strings.Contains(output, securityTestModule+".SecTestAdmin") {
		t.Errorf("Expected role %s.SecTestAdmin in GRANT statement.\nActual:\n%s", securityTestModule, output)
	}
	if !strings.Contains(output, "CREATE") || !strings.Contains(output, "DELETE") {
		t.Errorf("Expected CREATE, DELETE in GRANT rights.\nActual:\n%s", output)
	}
	if !strings.Contains(output, "READ *") || !strings.Contains(output, "WRITE *") {
		t.Errorf("Expected READ * and WRITE * in GRANT rights.\nActual:\n%s", output)
	}

	t.Logf("Entity security roundtrip successful:\n%s", output)

	// Round-trip: parse the DESCRIBE output and verify it's valid MDL
	_, parseErrs := visitor.Build(output)
	if len(parseErrs) > 0 {
		t.Errorf("DESCRIBE output is not valid MDL: %v\nOutput:\n%s", parseErrs[0], output)
	}
}

func TestRoundtripSecurity_EntityAccessMultipleRoles(t *testing.T) {
	env := setupSecurityTestEnv(t)
	defer env.teardown()

	entityName := securityTestModule + ".SecTestMultiRole"
	env.registerCleanup("entity", entityName)

	// Create entity
	createMDL := `CREATE PERSISTENT ENTITY ` + entityName + ` (
		DisplayName: String(200)
	);`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Grant full access to Admin
	grantAdmin := `GRANT ` + securityTestModule + `.SecTestAdmin ON ` + entityName + ` (CREATE, DELETE, READ *, WRITE *);`
	if err := env.executeMDL(grantAdmin); err != nil {
		t.Fatalf("Failed to grant Admin access: %v", err)
	}

	// Grant read-only access to Viewer
	grantViewer := `GRANT ` + securityTestModule + `.SecTestViewer ON ` + entityName + ` (READ *);`
	if err := env.executeMDL(grantViewer); err != nil {
		t.Fatalf("Failed to grant Viewer access: %v", err)
	}

	// Describe and verify
	output, err := env.describeMDL(`DESCRIBE ENTITY ` + entityName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe entity: %v", err)
	}

	// Should contain two GRANT statements
	grantCount := strings.Count(output, "GRANT")
	if grantCount < 2 {
		t.Errorf("Expected at least 2 GRANT statements, got %d.\nOutput:\n%s", grantCount, output)
	}
	if !strings.Contains(output, securityTestModule+".SecTestAdmin") {
		t.Errorf("Expected SecTestAdmin role in output.\nActual:\n%s", output)
	}
	if !strings.Contains(output, securityTestModule+".SecTestViewer") {
		t.Errorf("Expected SecTestViewer role in output.\nActual:\n%s", output)
	}

	t.Logf("Entity multi-role security roundtrip successful:\n%s", output)

	// Round-trip parse check
	_, parseErrs := visitor.Build(output)
	if len(parseErrs) > 0 {
		t.Errorf("DESCRIBE output is not valid MDL: %v\nOutput:\n%s", parseErrs[0], output)
	}
}

func TestRoundtripSecurity_EntityAccessWithXPath(t *testing.T) {
	env := setupSecurityTestEnv(t)
	defer env.teardown()

	entityName := securityTestModule + ".SecTestXPath"
	env.registerCleanup("entity", entityName)

	// Create entity
	createMDL := `CREATE PERSISTENT ENTITY ` + entityName + ` (
		Owner: String(100)
	);`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Grant with XPath constraint
	grantMDL := `GRANT ` + securityTestModule + `.SecTestViewer ON ` + entityName + ` (READ *) WHERE '[%CurrentUser%] = Owner';`
	if err := env.executeMDL(grantMDL); err != nil {
		t.Fatalf("Failed to grant entity access with XPath: %v", err)
	}

	// Describe and verify
	output, err := env.describeMDL(`DESCRIBE ENTITY ` + entityName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe entity: %v", err)
	}

	if !strings.Contains(output, "WHERE") {
		t.Errorf("Expected WHERE clause in GRANT statement.\nActual:\n%s", output)
	}
	if !strings.Contains(output, "[%CurrentUser%] = Owner") {
		t.Errorf("Expected XPath constraint in output.\nActual:\n%s", output)
	}

	t.Logf("Entity XPath security roundtrip successful:\n%s", output)

	// Round-trip parse check
	_, parseErrs := visitor.Build(output)
	if len(parseErrs) > 0 {
		t.Errorf("DESCRIBE output is not valid MDL: %v\nOutput:\n%s", parseErrs[0], output)
	}
}

func TestRoundtripSecurity_EntityNoAccess(t *testing.T) {
	env := setupSecurityTestEnv(t)
	defer env.teardown()

	entityName := securityTestModule + ".SecTestNoAccess"
	env.registerCleanup("entity", entityName)

	// Create entity without granting any access
	createMDL := `CREATE PERSISTENT ENTITY ` + entityName + ` (
		Value: Integer
	);`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Describe — should NOT contain any GRANT
	output, err := env.describeMDL(`DESCRIBE ENTITY ` + entityName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe entity: %v", err)
	}

	if strings.Contains(output, "GRANT") {
		t.Errorf("Expected NO GRANT statement for entity without access rules.\nActual:\n%s", output)
	}

	t.Logf("Entity no-access roundtrip successful:\n%s", output)
}

func TestRoundtripSecurity_MicroflowAccess(t *testing.T) {
	env := setupSecurityTestEnv(t)
	defer env.teardown()

	mfName := securityTestModule + ".SecTestMicroflow"
	env.registerCleanup("microflow", mfName)

	// Create microflow
	createMDL := `CREATE MICROFLOW ` + mfName + ` ()
	BEGIN
		LOG INFO 'test';
	END;`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	// Grant execute access to both roles
	grantMDL := `GRANT EXECUTE ON MICROFLOW ` + mfName + ` TO ` + securityTestModule + `.SecTestAdmin, ` + securityTestModule + `.SecTestViewer;`
	if err := env.executeMDL(grantMDL); err != nil {
		t.Fatalf("Failed to grant microflow access: %v", err)
	}

	// Describe and verify
	output, err := env.describeMDL(`DESCRIBE MICROFLOW ` + mfName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe microflow: %v", err)
	}

	if !strings.Contains(output, "GRANT EXECUTE ON MICROFLOW") {
		t.Errorf("Expected GRANT EXECUTE statement.\nActual:\n%s", output)
	}
	if !strings.Contains(output, securityTestModule+".SecTestAdmin") {
		t.Errorf("Expected SecTestAdmin role in output.\nActual:\n%s", output)
	}
	if !strings.Contains(output, securityTestModule+".SecTestViewer") {
		t.Errorf("Expected SecTestViewer role in output.\nActual:\n%s", output)
	}

	t.Logf("Microflow security roundtrip successful:\n%s", output)

	// Round-trip parse check
	_, parseErrs := visitor.Build(output)
	if len(parseErrs) > 0 {
		t.Errorf("DESCRIBE output is not valid MDL: %v\nOutput:\n%s", parseErrs[0], output)
	}
}

func TestRoundtripSecurity_MicroflowNoAccess(t *testing.T) {
	env := setupSecurityTestEnv(t)
	defer env.teardown()

	mfName := securityTestModule + ".SecTestMfNoAccess"
	env.registerCleanup("microflow", mfName)

	// Create microflow without granting access
	createMDL := `CREATE MICROFLOW ` + mfName + ` ()
	BEGIN
		LOG INFO 'test';
	END;`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create microflow: %v", err)
	}

	// Describe — should NOT contain GRANT
	output, err := env.describeMDL(`DESCRIBE MICROFLOW ` + mfName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe microflow: %v", err)
	}

	if strings.Contains(output, "GRANT") {
		t.Errorf("Expected NO GRANT statement for microflow without access.\nActual:\n%s", output)
	}

	t.Logf("Microflow no-access roundtrip successful:\n%s", output)
}

func TestRoundtripSecurity_PageAccess(t *testing.T) {
	env := setupSecurityTestEnv(t)
	defer env.teardown()

	pageName := securityTestModule + ".SecTestPage"
	env.registerCleanup("page", pageName)

	// Create page
	createMDL := `CREATE PAGE ` + pageName + ` (Title: 'Security Test', Layout: Atlas_Core.Atlas_Default) {
		CONTAINER container1 {
			DYNAMICTEXT text1 (Content: 'Hello')
		}
	}`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	// Grant view access
	grantMDL := `GRANT VIEW ON PAGE ` + pageName + ` TO ` + securityTestModule + `.SecTestAdmin;`
	if err := env.executeMDL(grantMDL); err != nil {
		t.Fatalf("Failed to grant page access: %v", err)
	}

	// Describe and verify
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	if !strings.Contains(output, "GRANT VIEW ON PAGE") {
		t.Errorf("Expected GRANT VIEW statement.\nActual:\n%s", output)
	}
	if !strings.Contains(output, securityTestModule+".SecTestAdmin") {
		t.Errorf("Expected SecTestAdmin role in output.\nActual:\n%s", output)
	}

	t.Logf("Page security roundtrip successful:\n%s", output)

	// Round-trip parse check
	_, parseErrs := visitor.Build(output)
	if len(parseErrs) > 0 {
		t.Errorf("DESCRIBE output is not valid MDL: %v\nOutput:\n%s", parseErrs[0], output)
	}
}

func TestRoundtripSecurity_PageNoAccess(t *testing.T) {
	env := setupSecurityTestEnv(t)
	defer env.teardown()

	pageName := securityTestModule + ".SecTestPageNoAccess"
	env.registerCleanup("page", pageName)

	// Create page without granting access
	createMDL := `CREATE PAGE ` + pageName + ` (Title: 'No Access', Layout: Atlas_Core.Atlas_Default) {
		CONTAINER container1 {
			DYNAMICTEXT text1 (Content: 'Hello')
		}
	}`
	if err := env.executeMDL(createMDL); err != nil {
		t.Fatalf("Failed to create page: %v", err)
	}

	// Describe — should NOT contain GRANT
	output, err := env.describeMDL(`DESCRIBE PAGE ` + pageName + `;`)
	if err != nil {
		t.Fatalf("Failed to describe page: %v", err)
	}

	if strings.Contains(output, "GRANT") {
		t.Errorf("Expected NO GRANT statement for page without access.\nActual:\n%s", output)
	}

	t.Logf("Page no-access roundtrip successful:\n%s", output)
}
