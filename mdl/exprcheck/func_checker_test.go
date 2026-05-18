// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestFuncChecker_ArityMismatch(t *testing.T) {
	p := NewParser()
	// substring requires min 2 args — 1 arg should still fail
	_, hs := p.Parse("substring('hi')", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Fatalf("expected E006, got %+v", hs)
	}
}

func TestFuncChecker_KnownArityOK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("length('hi')", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("unexpected E006: %+v", hs)
	}
}

// Optional argument tests — verify that Mendix functions with optional args
// do not fire E006 when called with the minimum required argument count.

func TestFuncChecker_Round_OneArg_OK(t *testing.T) {
	p := NewParser()
	// round(x) with 1 arg is valid in Mendix (rounds to 0 decimal places)
	_, hs := p.Parse("round(random())", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("round(x) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_Round_TwoArgs_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("round(1.567, 2)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("round(x, d) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_Round_ThreeArgs_Fail(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("round(1.0, 2, 3)", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Fatalf("round() with 3 args must fire E006, got %+v", hs)
	}
}

func TestFuncChecker_Substring_TwoArgs_OK(t *testing.T) {
	p := NewParser()
	// substring(string, startPos) is valid in Mendix (extracts to end of string)
	_, hs := p.Parse("substring('hello', 2)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("substring(s, pos) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_Substring_ThreeArgs_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("substring('hello', 0, 3)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("substring(s, pos, len) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_Substring_OneArg_Fail(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("substring('hello')", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Fatalf("substring with 1 arg must fire E006, got %+v", hs)
	}
}

func hasCode(hs []Hint, code string) bool {
	for _, h := range hs {
		if h.Code == code {
			return true
		}
	}
	return false
}

// ── Function table completeness tests ──────────────────────────────────────
// These tests guard against the func table containing wrong names or missing
// officially-documented Mendix expression functions.
// Reference: https://docs.mendix.com/refguide/expressions/

// TestFuncTable_CorrectCalendarBetweenNames verifies the correct official names
// are registered and the historical wrong names are absent.
func TestFuncTable_CorrectCalendarBetweenNames(t *testing.T) {
	table := PublicFuncTable()

	correct := []string{"calendarMonthsBetween", "calendarYearsBetween"}
	for _, name := range correct {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table (official Mendix name)", name)
		}
	}

	spurious := []string{"monthsBetween", "yearsBetween"}
	for _, name := range spurious {
		if _, ok := table[name]; ok {
			t.Errorf("function %q must NOT be in func table — official name is calendar-prefixed", name)
		}
	}
}

// TestFuncTable_SubtractDateFunctions verifies all subtract-date functions are registered.
func TestFuncTable_SubtractDateFunctions(t *testing.T) {
	table := PublicFuncTable()
	want := []string{
		"subtractMilliseconds", "subtractSeconds", "subtractMinutes", "subtractHours",
		"subtractDays", "subtractDaysUTC",
		"subtractWeeks", "subtractWeeksUTC",
		"subtractMonths", "subtractMonthsUTC",
		"subtractQuarters", "subtractQuartersUTC",
		"subtractYears", "subtractYearsUTC",
	}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// TestFuncTable_AddDateUTCAndMissing verifies addMilliseconds and UTC add variants.
func TestFuncTable_AddDateUTCAndMissing(t *testing.T) {
	table := PublicFuncTable()
	want := []string{
		"addMilliseconds",
		"addDaysUTC", "addWeeksUTC", "addMonthsUTC",
		"addQuarters", "addQuartersUTC",
		"addYearsUTC",
	}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// TestFuncTable_BeginOfDateFunctions verifies beginOf* functions are registered.
func TestFuncTable_BeginOfDateFunctions(t *testing.T) {
	table := PublicFuncTable()
	want := []string{"beginOfDay", "beginOfWeek", "beginOfMonth", "beginOfYear"}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// TestFuncTable_EndOfDateFunctions verifies endOf* functions are registered.
func TestFuncTable_EndOfDateFunctions(t *testing.T) {
	table := PublicFuncTable()
	want := []string{"endOfDay", "endOfWeek", "endOfMonth", "endOfYear"}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// TestFuncTable_TrimToDateFunctions verifies trimTo* functions are registered.
func TestFuncTable_TrimToDateFunctions(t *testing.T) {
	table := PublicFuncTable()
	want := []string{
		"trimToSeconds", "trimToMinutes",
		"trimToHours", "trimToHoursUTC",
		"trimToDays", "trimToDaysUTC",
		"trimToMonths", "trimToMonthsUTC",
		"trimToYears", "trimToYearsUTC",
	}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// TestFuncTable_BetweenDateFunctions verifies all between-date functions including milliseconds.
func TestFuncTable_BetweenDateFunctions(t *testing.T) {
	table := PublicFuncTable()
	want := []string{
		"millisecondsBetween",
		"secondsBetween", "minutesBetween", "hoursBetween",
		"daysBetween", "weeksBetween",
		"calendarMonthsBetween", "calendarYearsBetween",
	}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// TestFuncTable_DateCreationAndFormatting verifies UTC date creation and formatting functions.
func TestFuncTable_DateCreationAndFormatting(t *testing.T) {
	table := PublicFuncTable()
	want := []string{
		"dateTimeUTC",
		"formatDate", "formatDateUTC",
		"formatTime", "formatTimeUTC",
		"dateTimeToEpoch", "epochToDateTime",
		"formatDecimal",
	}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// TestFuncTable_StringFunctions verifies findLast and replaceFirst are registered.
func TestFuncTable_StringFunctions(t *testing.T) {
	table := PublicFuncTable()
	want := []string{"findLast", "replaceFirst"}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// TestFuncTable_EnumerationFunctions verifies getCaption and getKey are registered.
func TestFuncTable_EnumerationFunctions(t *testing.T) {
	table := PublicFuncTable()
	want := []string{"getCaption", "getKey"}
	for _, name := range want {
		if _, ok := table[name]; !ok {
			t.Errorf("function %q must be in func table", name)
		}
	}
}

// ── Arity enforcement tests ──────────────────────────────────────────────────
// These tests verify that once the functions are in the table, arity errors fire
// correctly. They will be RED before the functions are added to funcTable and
// GREEN after.

func TestFuncChecker_CalendarMonthsBetween_WrongArity_E006(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("calendarMonthsBetween($d)", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Error("calendarMonthsBetween with 1 arg must fire E006 (requires 2)")
	}
}

func TestFuncChecker_CalendarMonthsBetween_CorrectArity_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("calendarMonthsBetween($d1, $d2)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("calendarMonthsBetween($d1, $d2) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_CalendarYearsBetween_WrongArity_E006(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("calendarYearsBetween($d)", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Error("calendarYearsBetween with 1 arg must fire E006 (requires 2)")
	}
}

func TestFuncChecker_SubtractDays_WrongArity_E006(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("subtractDays($d)", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Error("subtractDays with 1 arg must fire E006 (requires 2)")
	}
}

func TestFuncChecker_SubtractDays_CorrectArity_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("subtractDays($d, 3)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("subtractDays($d, 3) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_BeginOfDay_WrongArity_E006(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("beginOfDay()", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Error("beginOfDay() with 0 args must fire E006 (requires 1)")
	}
}

func TestFuncChecker_BeginOfDay_CorrectArity_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("beginOfDay($d)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("beginOfDay($d) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_TrimToHours_WrongArity_E006(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("trimToHours()", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Error("trimToHours() with 0 args must fire E006 (requires 1)")
	}
}

func TestFuncChecker_FormatDate_WrongArity_E006(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("formatDate()", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Error("formatDate() with 0 args must fire E006 (requires 1)")
	}
}

func TestFuncChecker_FormatDate_CorrectArity_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("formatDate($d)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("formatDate($d) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_GetCaption_CorrectArity_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("getCaption(MyMod.Status.Active)", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("getCaption(enum) must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_FindLast_WrongArity_E006(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("findLast('hello')", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Error("findLast with 1 arg must fire E006 (requires 2)")
	}
}

func TestFuncChecker_FindLast_CorrectArity_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("findLast('hello world', 'o')", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("findLast('s','sub') must not fire E006: %+v", hs)
	}
}

func TestFuncChecker_FormatDecimal_WrongArity_E006(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("formatDecimal()", Context{Microflow: "M.F"})
	if !hasCode(hs, "E006") {
		t.Error("formatDecimal() with 0 args must fire E006 (requires at least 1)")
	}
}

func TestFuncChecker_FormatDecimal_CorrectArity_OK(t *testing.T) {
	p := NewParser()
	_, hs := p.Parse("formatDecimal($x, '#,###.00')", Context{Microflow: "M.F"})
	if hasCode(hs, "E006") {
		t.Errorf("formatDecimal(x, mask) must not fire E006: %+v", hs)
	}
}

func TestFuncReturnKind_DateTimeExtraction(t *testing.T) {
	intFuncs := []string{"year", "month", "dayOfYear", "dayOfMonth",
		"weekOfYear", "dayOfWeek", "hour", "minute", "second", "millisecond"}
	for _, name := range intFuncs {
		k, ok := FuncReturnKind(name)
		if !ok {
			t.Errorf("FuncReturnKind(%q): not found", name)
			continue
		}
		if k != KindInteger {
			t.Errorf("FuncReturnKind(%q) = %v, want KindInteger", name, k)
		}
	}
}

func TestFuncReturnKind_Unknown(t *testing.T) {
	_, ok := FuncReturnKind("nonExistentFunction")
	if ok {
		t.Error("expected false for unknown function")
	}
}
