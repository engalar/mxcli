package scss

import (
	"testing"
)

func TestParse_BasicSassVar(t *testing.T) {
	input := "$brand-primary: #264ae5;"
	doc, err := Parse("test.scss", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	v := doc.Vars[0]
	if v.Name != "$brand-primary" {
		t.Errorf("Name = %q, want $brand-primary", v.Name)
	}
	if v.Value != "#264ae5" {
		t.Errorf("Value = %q, want #264ae5", v.Value)
	}
	if v.IsCSSVar {
		t.Error("IsCSSVar should be false for SASS var")
	}
}

func TestParse_CssCustomProperty(t *testing.T) {
	input := ":root {\n  --brand-primary: #1565C0;\n}"
	doc, err := Parse("test.scss", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	v := doc.Vars[0]
	if v.Name != "--brand-primary" {
		t.Errorf("Name = %q, want --brand-primary", v.Name)
	}
	if v.Value != "#1565C0" {
		t.Errorf("Value = %q, want #1565C0", v.Value)
	}
	if !v.IsCSSVar {
		t.Error("IsCSSVar should be true")
	}
	if !v.IsInRoot {
		t.Error("IsInRoot should be true")
	}
}

func TestParse_DefaultFlag(t *testing.T) {
	input := "$spacing-small: 8px !default;"
	doc, _ := Parse("test.scss", input)
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	if !doc.Vars[0].IsDefault {
		t.Error("IsDefault should be true")
	}
}

func TestParse_CommentedVar(t *testing.T) {
	input := "// $brand-primary: #264ae5;"
	doc, _ := Parse("test.scss", input)
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	if doc.Vars[0].IsActive {
		t.Error("commented var should be IsActive=false")
	}
}

func TestParse_MixedContent(t *testing.T) {
	input := `// Theme options
$brand-logo: false;

:root {
  --brand-primary: #264ae5;
  // --brand-success: #16aa16;
}

@import "variables";`
	doc, _ := Parse("test.scss", input)
	if len(doc.Vars) != 3 {
		t.Fatalf("Vars = %d, want 3 ($brand-logo, --brand-primary, --brand-success)", len(doc.Vars))
	}
	if len(doc.Lines) != 9 {
		t.Errorf("Lines = %d, want 9 (preserve all lines)", len(doc.Lines))
	}
}

func TestParse_CommentPreserved(t *testing.T) {
	input := "$brand-primary: #264ae5; // primary brand color"
	doc, _ := Parse("test.scss", input)
	if doc.Vars[0].Comment != "primary brand color" {
		t.Errorf("Comment = %q, want 'primary brand color'", doc.Vars[0].Comment)
	}
}

func TestSetVar_NewVar(t *testing.T) {
	doc, _ := Parse("test.scss", "@import \"variables\";")
	err := doc.SetVar("--brand-primary", "#1565C0", ScssVarOpts{InsertBefore: "@import"})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Vars) != 1 {
		t.Fatalf("Vars = %d, want 1", len(doc.Vars))
	}
	if doc.Vars[0].Value != "#1565C0" {
		t.Errorf("Value = %q, want #1565C0", doc.Vars[0].Value)
	}
}

func TestSetVar_UpdateExisting(t *testing.T) {
	doc, _ := Parse("test.scss", "$brand-primary: #264ae5;")
	err := doc.SetVar("$brand-primary", "#1565C0", ScssVarOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Vars[0].Value != "#1565C0" {
		t.Errorf("Value = %q, want #1565C0", doc.Vars[0].Value)
	}
	v := doc.findVar("$brand-primary")
	if v == nil {
		t.Fatal("findVar returned nil")
	}
	if v.Value != "#1565C0" {
		t.Errorf("findVar Value = %q", v.Value)
	}
}

func TestWrite_RoundTrip(t *testing.T) {
	input := `// Theme options
$brand-logo: false;

:root {
  --brand-primary: #264ae5;
}

@import "variables";`
	doc, err := Parse("test.scss", input)
	if err != nil {
		t.Fatal(err)
	}
	output := doc.Write()
	if output != input {
		t.Errorf("Write() != original input\n--- got:\n%s\n--- want:\n%s", output, input)
	}
}

func TestWrite_AfterSetVar(t *testing.T) {
	doc, _ := Parse("test.scss", "@import \"variables\";")
	doc.SetVar("--brand-primary", "#1565C0", ScssVarOpts{InsertBefore: "@import"})
	output := doc.Write()
	if output != "  --brand-primary: #1565C0;\n@import \"variables\";" {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestWrite_AfterUpdate(t *testing.T) {
	doc, _ := Parse("test.scss", "$brand-primary: #264ae5;")
	doc.SetVar("$brand-primary", "#1565C0", ScssVarOpts{})
	output := doc.Write()
	if output != "$brand-primary: #1565C0;" {
		t.Errorf("unexpected output: %q", output)
	}
}
