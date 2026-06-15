// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeXML writes content to dir/src/<filename>.
func writeXML(t *testing.T, dir, filename, content string) {
	t.Helper()
	srcDir := filepath.Join(dir, "src")
	os.MkdirAll(srcDir, 0755)
	if err := os.WriteFile(filepath.Join(srcDir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("writeXML %s: %v", filename, err)
	}
}

const validWidgetXML = `<?xml version="1.0" encoding="utf-8"?>
<widget id="com.acme.widget.MySlider.MySlider" pluginWidget="true" offlineCapable="false"
        xmlns="http://www.mendix.com/widget/1.0/">
  <name>My Slider</name>
  <properties>
    <propertyGroup caption="General">
      <property key="value" type="attribute" required="true">
        <caption>Value</caption><description/>
        <attributeTypes><attributeType name="Decimal"/></attributeTypes>
      </property>
    </propertyGroup>
  </properties>
</widget>`

func TestDiscoverWidgets_FindsXML(t *testing.T) {
	dir := t.TempDir()
	writeXML(t, dir, "MySlider.xml", validWidgetXML)

	infos, err := discoverWidgets(dir)
	if err != nil {
		t.Fatalf("discoverWidgets: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(infos))
	}
	if infos[0].Name != "MySlider" || infos[0].WidgetID != "com.acme.widget.MySlider.MySlider" {
		t.Errorf("unexpected widget info: %+v", infos[0])
	}
}

func TestDiscoverWidgets_EmptySrc(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	infos, err := discoverWidgets(dir)
	if err != nil {
		t.Fatalf("discoverWidgets: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 widgets, got %d", len(infos))
	}
}

func TestValidateWidgetInfo_ValidID(t *testing.T) {
	info := widgetInfo{Name: "MySlider", WidgetID: "com.acme.widget.MySlider.MySlider", DisplayName: "My Slider"}
	if err := validateWidgetInfo(info); err != nil {
		t.Errorf("validateWidgetInfo: unexpected error: %v", err)
	}
}

func TestValidateWidgetInfo_BadID(t *testing.T) {
	cases := []struct {
		id      string
		wantMsg string
	}{
		{"com.short", "at least 4"},
		{"one.two", "at least 4"},
		{"", "at least 4"},
	}
	for _, tc := range cases {
		info := widgetInfo{Name: "Foo", WidgetID: tc.id, DisplayName: "Foo"}
		err := validateWidgetInfo(info)
		if err == nil {
			t.Errorf("validateWidgetInfo(%q): expected error", tc.id)
		} else if !strings.Contains(err.Error(), tc.wantMsg) {
			t.Errorf("validateWidgetInfo(%q): error %q does not contain %q", tc.id, err.Error(), tc.wantMsg)
		}
	}
}

func TestValidateWidgetInfo_EmptyName(t *testing.T) {
	info := widgetInfo{Name: "MySlider", WidgetID: "com.a.b.c.MySlider", DisplayName: ""}
	err := validateWidgetInfo(info)
	if err == nil {
		t.Errorf("validateWidgetInfo: expected error for empty display name")
	}
}

func TestDetectToolchain_FindsSomething(t *testing.T) {
	tool, err := detectToolchain()
	if err == nil && tool != "bun" && tool != "npm" {
		t.Errorf("detectToolchain returned unexpected tool %q", tool)
	}
	if err != nil && !strings.Contains(err.Error(), "bun") {
		t.Errorf("error must mention bun: %v", err)
	}
}

func TestPackageMPK_CreatesZip(t *testing.T) {
	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	jsDir := filepath.Join(distDir, "com", "mendix", "widget", "custom", "MySlider")
	os.MkdirAll(jsDir, 0755)
	os.WriteFile(filepath.Join(jsDir, "MySlider.js"), []byte("// bundle"), 0644)
	os.WriteFile(filepath.Join(distDir, "MySlider.xml"), []byte("<widget/>"), 0644)
	os.WriteFile(filepath.Join(distDir, "package.xml"), []byte("<package/>"), 0644)

	mpkPath, err := packageMPK(dir, "MySlider")
	if err != nil {
		t.Fatalf("packageMPK: %v", err)
	}
	if _, err := os.Stat(mpkPath); err != nil {
		t.Errorf("MPK file not created at %s: %v", mpkPath, err)
	}
	if !strings.HasSuffix(mpkPath, "MySlider.mpk") {
		t.Errorf("expected MySlider.mpk, got %s", mpkPath)
	}
}

func TestDiscoverWidgets_WithAbsoluteDir(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	xmlContent := `<?xml version="1.0" encoding="utf-8"?>
<widget id="com.example.MyWidget" pluginWidget="true"
        xmlns="http://www.mendix.com/widget/1.0/"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:schemaLocation="http://www.mendix.com/widget/1.0/ ../node_modules/mendix/custom_widget.xsd">
  <name>MyWidget</name>
  <description/>
  <properties/>
</widget>`
	if err := os.WriteFile(filepath.Join(srcDir, "MyWidget.xml"), []byte(xmlContent), 0644); err != nil {
		t.Fatal(err)
	}
	infos, err := discoverWidgets(tmp)
	if err != nil {
		t.Fatalf("discoverWidgets(%q): %v", tmp, err)
	}
	if len(infos) != 1 || infos[0].Name != "MyWidget" {
		t.Errorf("discoverWidgets: got %v, want [{MyWidget ...}]", infos)
	}
}

func TestWidgetBuildDirResolution(t *testing.T) {
	cases := []string{".", "./SomeWidget", "../../relative/path"}
	for _, rel := range cases {
		abs, err := filepath.Abs(rel)
		if err != nil {
			t.Errorf("filepath.Abs(%q): %v", rel, err)
			continue
		}
		if !filepath.IsAbs(abs) {
			t.Errorf("filepath.Abs(%q) = %q is not absolute", rel, abs)
		}
	}
}

func TestFindMPKInCwd_NoneFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	_, err := findMPKInCwd()
	if err == nil {
		t.Fatal("expected error when no MPK found, got nil")
	}
	if !strings.Contains(err.Error(), "widget build") {
		t.Errorf("error should mention 'widget build', got: %v", err)
	}
}

func TestFindMPKInCwd_OneFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	mpkPath := filepath.Join(dir, "MyWidget.mpk")
	os.WriteFile(mpkPath, []byte("fake"), 0644)

	got, err := findMPKInCwd()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "MyWidget.mpk" {
		t.Errorf("got %q, want %q", got, "MyWidget.mpk")
	}
}

func TestFindMPKInCwd_MultipleFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.WriteFile(filepath.Join(dir, "A.mpk"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(dir, "B.mpk"), []byte("fake"), 0644)

	_, err := findMPKInCwd()
	if err == nil {
		t.Fatal("expected error for multiple MPKs, got nil")
	}
	if !strings.Contains(err.Error(), "--mpk") {
		t.Errorf("error should mention '--mpk', got: %v", err)
	}
}

func TestInstallMPK_CreatesWidgetsDirAndCopiesFile(t *testing.T) {
	srcDir := t.TempDir()
	projectDir := t.TempDir()

	mpkPath := filepath.Join(srcDir, "MyWidget.mpk")
	if err := os.WriteFile(mpkPath, []byte("fake mpk content"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	projectPath := filepath.Join(projectDir, "app.mpr")

	if err := installMPK(mpkPath, projectPath); err != nil {
		t.Fatalf("installMPK: %v", err)
	}

	dst := filepath.Join(projectDir, "widgets", "MyWidget.mpk")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("expected installed file at %s: %v", dst, err)
	}
	if string(data) != "fake mpk content" {
		t.Errorf("installed file content mismatch: %q", data)
	}
}

func TestInstallMPK_OverwritesExistingFile(t *testing.T) {
	srcDir := t.TempDir()
	projectDir := t.TempDir()

	widgetsDir := filepath.Join(projectDir, "widgets")
	os.MkdirAll(widgetsDir, 0755)
	os.WriteFile(filepath.Join(widgetsDir, "MyWidget.mpk"), []byte("old"), 0644)

	mpkPath := filepath.Join(srcDir, "MyWidget.mpk")
	os.WriteFile(mpkPath, []byte("new"), 0644)

	if err := installMPK(mpkPath, filepath.Join(projectDir, "app.mpr")); err != nil {
		t.Fatalf("installMPK: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(widgetsDir, "MyWidget.mpk"))
	if string(data) != "new" {
		t.Errorf("expected overwrite with 'new', got %q", data)
	}
}

func TestRoundTrip_ScaffoldThenDiscover(t *testing.T) {
	dir := t.TempDir()
	props := []PropertySpec{
		{Key: "value", XMLType: "attribute", Subtype: "Decimal"},
		{Key: "label", XMLType: "string"},
	}
	err := scaffoldWidget(dir, "RoundTrip", "com.test.widget.RoundTrip.RoundTrip", "", false, props)
	if err != nil {
		t.Fatalf("scaffoldWidget: %v", err)
	}
	infos, err := discoverWidgets(dir)
	if err != nil {
		t.Fatalf("discoverWidgets: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 widget, got %d", len(infos))
	}
	if err := validateWidgetInfo(infos[0]); err != nil {
		t.Errorf("validateWidgetInfo failed on scaffolded widget: %v", err)
	}
	if infos[0].WidgetID != "com.test.widget.RoundTrip.RoundTrip" {
		t.Errorf("wrong widgetID: %s", infos[0].WidgetID)
	}
}
