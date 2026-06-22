package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchRollupCP_replacesXmlGlob(t *testing.T) {
	// Build exact unpatched rollup.config.mjs content matching
	// the oldPng/oldXml patterns in patchRollupCP exactly.
	xmlLine := "                cp(join(sourcePath, \"src/**/*.xml\"), outDir);"
	pngBlock := "if (existsSync(`src/${widgetName}.icon.png`) || existsSync(`src/${widgetName}.tile.png`)) {\n                        cp(join(sourcePath, `src/${widgetName}.@(tile|icon)?(.dark).png`), outDir);\n                    }"
	content := "import { widgetName, sourcePath, outDir } from \"./shared.mjs\";\n"
	content += "import { copyLicenseFile, createMpkFile } from \"./helpers/rollup-helper.mjs\";\n"
	content += "\n"
	content += "function getClientComponentPlugins() {\n"
	content += "    return [\n"
	content += "        isTypescript ? widgetTyping({ sourceDir: join(sourcePath, \"src\") }) : null,\n"
	content += "        clear({ targets: [outDir, mpkDir] }),\n"
	content += "        command([\n"
	content += "            () => {\n"
	content += "                " + xmlLine + "\n"
	content += "                " + pngBlock + "\n"
	content += "            }\n"
	content += "        ]),\n"
	content += "    ];\n"
	content += "}\n"

	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, "node_modules", "@mendix", "pluggable-widgets-tools", "configs")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "rollup.config.mjs")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchRollupCP(tmpDir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)

	if strings.Contains(result, `src/**/*.xml`) {
		t.Error("XML glob pattern should have been replaced, but still present")
	}
	if !strings.Contains(result, `package.xml`) {
		t.Error("patched content should reference package.xml explicitly")
	}
	if !strings.Contains(result, `widgetName+'.xml'`) {
		t.Error("patched content should use widgetName to build widget XML filename")
	}
	if !strings.Contains(result, `[mxcli-patched-xml]`) {
		t.Error("sentinel [mxcli-patched-xml] should be present")
	}
	if !strings.Contains(result, `[mxcli-patched-png]`) {
		t.Error("sentinel [mxcli-patched-png] should be present")
	}
}

func TestPatchRollupCP_idempotent(t *testing.T) {
	// Verify that applying the patch twice doesn't error.
	xmlLine := "                cp(join(sourcePath, \"src/**/*.xml\"), outDir);"
	pngBlock := "if (existsSync(`src/${widgetName}.icon.png`) || existsSync(`src/${widgetName}.tile.png`)) {\n                        cp(join(sourcePath, `src/${widgetName}.@(tile|icon)?(.dark).png`), outDir);\n                    }"
	content := "import { widgetName, sourcePath, outDir } from \"./shared.mjs\";\n"
	content += "function getClientComponentPlugins() {\n"
	content += "    return [\n"
	content += "        command([\n"
	content += "            () => {\n"
	content += "                " + xmlLine + "\n"
	content += "                " + pngBlock + "\n"
	content += "            }\n"
	content += "        ]),\n"
	content += "    ];\n"
	content += "}\n"

	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, "node_modules", "@mendix", "pluggable-widgets-tools", "configs")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "rollup.config.mjs")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// First call — should patch
	if err := patchRollupCP(tmpDir); err != nil {
		t.Fatalf("first patch: %v", err)
	}

	// Second call — should be no-op, no error
	if err := patchRollupCP(tmpDir); err != nil {
		t.Fatalf("second patch (idempotent): %v", err)
	}
}
