package testresource

import "testing"

func TestClassify_Uncategorized(t *testing.T) {
	p := Profile{Name: "T", DurationMs: 1000, CPUTimeMs: 100, ReadBytes: 0, WriteBytes: 0}
	if c := Classify(p); c != CategoryUncategorized {
		t.Errorf("Classify = %v, want CategoryUncategorized", c)
	}
}

func TestClassify_IOHeavy_ByRead(t *testing.T) {
	p := Profile{Name: "T", DurationMs: 1000, CPUTimeMs: 100, ReadBytes: 50_000_000, WriteBytes: 0}
	if c := Classify(p); c != CategoryIOHeavy {
		t.Errorf("Classify = %v, want CategoryIOHeavy", c)
	}
}

func TestClassify_IOHeavy_ByWrite(t *testing.T) {
	p := Profile{Name: "T", DurationMs: 1000, CPUTimeMs: 100, ReadBytes: 0, WriteBytes: 5_000_000}
	if c := Classify(p); c != CategoryIOHeavy {
		t.Errorf("Classify = %v, want CategoryIOHeavy", c)
	}
}

func TestClassify_CPUHeavy(t *testing.T) {
	p := Profile{Name: "T", DurationMs: 1000, CPUTimeMs: 800, ReadBytes: 0, WriteBytes: 0}
	if c := Classify(p); c != CategoryCPUHeavy {
		t.Errorf("Classify = %v, want CategoryCPUHeavy", c)
	}
}

func TestClassify_Mixed(t *testing.T) {
	p := Profile{Name: "T", DurationMs: 1000, CPUTimeMs: 800, ReadBytes: 50_000_000, WriteBytes: 0}
	if c := Classify(p); c != CategoryMixed {
		t.Errorf("Classify = %v, want CategoryMixed", c)
	}
}

func TestClassify_ZeroDuration(t *testing.T) {
	p := Profile{Name: "T", DurationMs: 0, CPUTimeMs: 100, ReadBytes: 0, WriteBytes: 0}
	if c := Classify(p); c != CategoryUncategorized {
		t.Errorf("Classify = %v, want CategoryUncategorized (zero duration guard)", c)
	}
}

func TestClassify_EdgeThreshold(t *testing.T) {
	p := Profile{Name: "T", DurationMs: 1000, CPUTimeMs: 500, ReadBytes: 0, WriteBytes: 0}
	if c := Classify(p); c != CategoryUncategorized {
		t.Errorf("Classify = %v, want CategoryUncategorized (CPUTime/Duration not strictly > 0.5)", c)
	}
}
