package executor

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
)

func connectedMock() *mock.MockBackend {
	return &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
}

func TestExecShowInstalledWidgets_NoWidgetsDir(t *testing.T) {
	tmp := t.TempDir()
	out := &strings.Builder{}
	ctx := &ExecContext{
		Backend:        connectedMock(),
		Output: out,
		MprPath: filepath.Join(tmp, "app.mpr"),
	}
	ctx.initRoles()
	err := execShowInstalledWidgetsFn(ctx, &ast.ShowInstalledWidgetsStmt{}, ctx.Deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "No widget packages found") {
		t.Errorf("expected 'No widget packages found', got:\n%s", out.String())
	}
}

func TestExecShowInstalledWidgets_FindsMpk(t *testing.T) {
	tmp := t.TempDir()
	widgetsDir := filepath.Join(tmp, "widgets")
	if err := os.MkdirAll(widgetsDir, 0755); err != nil {
		t.Fatal(err)
	}
	createMinimalMPKForTest(t,
		filepath.Join(widgetsDir, "TestWidget.mpk"),
		"com.example.TestWidget.TestWidget",
		"TestWidget",
	)

	out := &strings.Builder{}
	ctx := &ExecContext{
		Backend:        connectedMock(),
		Output: out,
		MprPath: filepath.Join(tmp, "app.mpr"),
	}
	ctx.initRoles()

	err := execShowInstalledWidgetsFn(ctx, &ast.ShowInstalledWidgetsStmt{}, ctx.Deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, "com.example.TestWidget.TestWidget") {
		t.Errorf("expected widget ID in output, got:\n%s", result)
	}
	if !strings.Contains(result, "PLUGGABLEWIDGET") {
		t.Errorf("expected MDL usage hint in output, got:\n%s", result)
	}
}

// createMinimalMPKForTest builds a valid .mpk zip: package.xml manifest pointing
// to a widget XML file, plus the widget XML itself (mpk.ParseAll requires both).
func createMinimalMPKForTest(t *testing.T, path, widgetID, widgetName string) {
	t.Helper()
	widgetFile := widgetName + ".xml"
	packageXML := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.mendix.com/package/1.0/">
  <clientModule name="` + widgetName + `" version="1.0.0" xmlns="http://www.mendix.com/clientModule/1.0/">
    <widgetFiles>
      <widgetFile path="` + widgetFile + `"/>
    </widgetFiles>
  </clientModule>
</package>`

	widgetXML := `<?xml version="1.0" encoding="utf-8"?>
<widget id="` + widgetID + `" pluginWidget="true"
        xmlns="http://www.mendix.com/widget/1.0/"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../node_modules/mendix/custom_widget.xsd">
  <name>` + widgetName + `</name>
  <description/>
  <properties/>
</widget>`

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for _, e := range []struct{ name, content string }{
		{"package.xml", packageXML},
		{widgetFile, widgetXML},
	} {
		entry, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, e.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}
