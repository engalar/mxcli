package tokens_test

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/expr/tokens"
)

func TestAll_Count(t *testing.T) {
	// 2 对象 + 1 CurrentDateTime + 16 基础时间点(HasUTC) + 16 UTC变体 + 7 长度 + 2 布尔 + 1 Null = 45
	// 时间点 8 段（Minute/Hour/Day/Yesterday/Tomorrow/Week/Month/Year）× Begin/End = 16
	if got := len(tokens.All); got != 45 {
		t.Errorf("All has %d tokens, want 45", got)
	}
}

func TestAll_NoUTCDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, tok := range tokens.All {
		if seen[tok.Name] {
			t.Errorf("duplicate token name: %s", tok.Name)
		}
		seen[tok.Name] = true
	}
}

func TestLookup_StaticToken(t *testing.T) {
	cases := []struct {
		name string
		kind tokens.Kind
	}{
		{"CurrentDateTime", tokens.KindDateTime},
		{"BeginOfCurrentMinute", tokens.KindDateTime},
		{"BeginOfCurrentMinuteUTC", tokens.KindDateTime},
		{"BeginOfYesterday", tokens.KindDateTime},
		{"BeginOfTomorrow", tokens.KindDateTime},
		{"DayLength", tokens.KindDuration},
		{"SecondLength", tokens.KindDuration},
		{"HourLength", tokens.KindDuration},
		{"WeekLength", tokens.KindDuration},
		{"MonthLength", tokens.KindDuration},
		{"YearLength", tokens.KindDuration},
		{"CurrentUser", tokens.KindObjectRef},
		{"CurrentObject", tokens.KindObjectRef},
		{"True", tokens.KindBoolean},
		{"False", tokens.KindBoolean},
		{"Null", tokens.KindEmpty},
	}
	for _, tc := range cases {
		tok, ok := tokens.Lookup(tc.name)
		if !ok {
			t.Errorf("Lookup(%q): not found", tc.name)
			continue
		}
		if tok.Kind != tc.kind {
			t.Errorf("Lookup(%q).Kind = %v, want %v", tc.name, tok.Kind, tc.kind)
		}
	}
}

func TestLookup_Unknown(t *testing.T) {
	_, ok := tokens.Lookup("NotAToken")
	if ok {
		t.Error("Lookup(NotAToken) should return false")
	}
}

func TestLookupUserRole_Match(t *testing.T) {
	name, ok := tokens.LookupUserRole("UserRole_Administrator")
	if !ok {
		t.Error("LookupUserRole(UserRole_Administrator): expected ok")
	}
	if name != "Administrator" {
		t.Errorf("LookupUserRole role name = %q, want %q", name, "Administrator")
	}
}

func TestLookupUserRole_NoMatch(t *testing.T) {
	_, ok := tokens.LookupUserRole("CurrentUser")
	if ok {
		t.Error("LookupUserRole(CurrentUser): should not match")
	}
	_, ok = tokens.LookupUserRole("UserRole_") // empty role name
	if ok {
		t.Error("LookupUserRole(UserRole_) empty role: should not match")
	}
}

func TestAll_UTCPairing(t *testing.T) {
	// Each HasUTC=true token must have a corresponding UTC variant in All.
	names := map[string]bool{}
	for _, tok := range tokens.All {
		names[tok.Name] = true
	}
	for _, tok := range tokens.All {
		if tok.HasUTC && !tok.IsUTC {
			utcName := tok.Name + "UTC"
			if !names[utcName] {
				t.Errorf("token %q has HasUTC=true but %q not found in All", tok.Name, utcName)
			}
		}
	}
}

func TestAll_DurationKinds(t *testing.T) {
	want := []string{"SecondLength", "MinuteLength", "HourLength", "DayLength", "WeekLength", "MonthLength", "YearLength"}
	for _, name := range want {
		tok, ok := tokens.Lookup(name)
		if !ok {
			t.Errorf("Lookup(%q): not found", name)
			continue
		}
		if tok.Kind != tokens.KindDuration {
			t.Errorf("Lookup(%q).Kind = %v, want KindDuration", name, tok.Kind)
		}
		if strings.HasSuffix(name, "UTC") {
			t.Errorf("duration token %q should not be UTC variant", name)
		}
	}
}
