// SPDX-License-Identifier: Apache-2.0
package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadJavaActionSource_AllSections(t *testing.T) {
	tmpDir := t.TempDir()
	mprPath := filepath.Join(tmpDir, "test.mpr")
	if err := os.WriteFile(mprPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	javaDir := filepath.Join(tmpDir, "javasource", "mymod", "actions")
	if err := os.MkdirAll(javaDir, 0755); err != nil {
		t.Fatal(err)
	}
	javaContent := "package mymod.actions;\n" +
		"import com.mendix.systemwideinterfaces.core.IContext;\n" +
		"import java.util.List;\n" +
		"public class MyAction extends UserAction<java.lang.String> {\n" +
		"\tpublic java.lang.String executeAction() throws Exception {\n" +
		"\t\t// BEGIN USER CODE\n" +
		"\t\treturn this.Input.trim();\n" +
		"\t\t// END USER CODE\n" +
		"\t}\n" +
		"\t// BEGIN EXTRA CODE\n" +
		"\tprivate String helper() { return \"\"; }\n" +
		"\t// END EXTRA CODE\n" +
		"}\n"
	if err := os.WriteFile(filepath.Join(javaDir, "MyAction.java"), []byte(javaContent), 0644); err != nil {
		t.Fatal(err)
	}

	userCode, imports, extraCode := readJavaActionSource(mprPath, "mymod", "MyAction")

	if !strings.Contains(userCode, "return this.Input.trim();") {
		t.Errorf("userCode = %q", userCode)
	}
	if !strings.Contains(extraCode, "helper") {
		t.Errorf("extraCode = %q", extraCode)
	}
	hasListImport := false
	for _, imp := range imports {
		if strings.Contains(imp, "java.util.List") {
			hasListImport = true
		}
	}
	if !hasListImport {
		t.Errorf("imports = %v, want java.util.List", imports)
	}
}
