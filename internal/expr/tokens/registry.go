package tokens

import "strings"

// Kind categorises a Mendix system token by its semantic type.
type Kind int

const (
	KindDateTime  Kind = iota // time-point tokens (BeginOfCurrentDay, etc.)
	KindDuration              // time-length tokens (DayLength, etc.) — integer milliseconds
	KindObjectRef             // object GUID tokens (CurrentUser, CurrentObject)
	KindBoolean               // True, False
	KindEmpty                 // Null
)

// Token is a single Mendix system token descriptor.
type Token struct {
	Name   string
	Kind   Kind
	HasUTC bool // a UTC variant of this token exists (only set on the non-UTC variant)
	IsUTC  bool // this IS the UTC variant
}

// All is the complete static token list derived from the Mendix documentation:
// https://docs.mendix.com/refguide/xpath-keywords-and-system-variables/
//
// Total: 2 object + 1 CurrentDateTime + 16 base time-point (HasUTC) +
//
//	16 UTC variants + 7 duration + 2 boolean + 1 null = 45.
//
// Time-points cover 8 buckets (Minute/Hour/Day/Yesterday/Tomorrow/Week/Month/Year)
// × Begin/End = 16, each with a UTC variant.
var All = []Token{
	// Object-related
	{Name: "CurrentUser", Kind: KindObjectRef},
	{Name: "CurrentObject", Kind: KindObjectRef},

	// Time-point: no UTC variant
	{Name: "CurrentDateTime", Kind: KindDateTime},

	// Time-point: base (HasUTC=true) + UTC variants
	{Name: "BeginOfCurrentMinute", Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentMinute", Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentHour", Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentHour", Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentDay", Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentDay", Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfYesterday", Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfYesterday", Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfTomorrow", Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfTomorrow", Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentWeek", Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentWeek", Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentMonth", Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentMonth", Kind: KindDateTime, HasUTC: true},
	{Name: "BeginOfCurrentYear", Kind: KindDateTime, HasUTC: true},
	{Name: "EndOfCurrentYear", Kind: KindDateTime, HasUTC: true},

	// UTC variants (18)
	{Name: "BeginOfCurrentMinuteUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "EndOfCurrentMinuteUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "BeginOfCurrentHourUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "EndOfCurrentHourUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "BeginOfCurrentDayUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "EndOfCurrentDayUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "BeginOfYesterdayUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "EndOfYesterdayUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "BeginOfTomorrowUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "EndOfTomorrowUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "BeginOfCurrentWeekUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "EndOfCurrentWeekUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "BeginOfCurrentMonthUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "EndOfCurrentMonthUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "BeginOfCurrentYearUTC", Kind: KindDateTime, IsUTC: true},
	{Name: "EndOfCurrentYearUTC", Kind: KindDateTime, IsUTC: true},

	// Duration (time-length, value in milliseconds — integer semantics)
	{Name: "SecondLength", Kind: KindDuration},
	{Name: "MinuteLength", Kind: KindDuration},
	{Name: "HourLength", Kind: KindDuration},
	{Name: "DayLength", Kind: KindDuration},
	{Name: "WeekLength", Kind: KindDuration},
	{Name: "MonthLength", Kind: KindDuration},
	{Name: "YearLength", Kind: KindDuration},

	// Boolean / Null
	{Name: "True", Kind: KindBoolean},
	{Name: "False", Kind: KindBoolean},
	{Name: "Null", Kind: KindEmpty},
}

// index is built once from All for O(1) lookups.
var index map[string]Token

func init() {
	index = make(map[string]Token, len(All))
	for _, t := range All {
		index[t.Name] = t
	}
}

// Lookup returns the Token for an exact static token name.
// Returns (Token{}, false) for unknown names or UserRole_* patterns.
func Lookup(name string) (Token, bool) {
	t, ok := index[name]
	return t, ok
}

// LookupUserRole matches a "UserRole_<RoleName>" prefix pattern.
// Returns the role name portion and true when the pattern matches and
// the role name part is non-empty.
func LookupUserRole(name string) (roleName string, ok bool) {
	const prefix = "UserRole_"
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	role := name[len(prefix):]
	if role == "" {
		return "", false
	}
	return role, true
}
