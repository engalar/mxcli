// SPDX-License-Identifier: Apache-2.0

package hints

type Entry struct {
	Code     string
	Slug     string
	Severity Severity
	Trigger  string
	WhyWrong string
	HowToFix string
	Examples []ExampleFix
}

type ExampleFix struct {
	Wrong string
	Right string
	Note  string
}

type registry struct {
	byCode map[string]Entry
}

func (r *registry) Lookup(code string) (Entry, bool) {
	e, ok := r.byCode[code]
	return e, ok
}

func (r *registry) All() []Entry {
	out := make([]Entry, 0, len(r.byCode))
	for _, e := range r.byCode {
		out = append(out, e)
	}
	return out
}

var Registry = &registry{byCode: map[string]Entry{
	"E001": {
		Code:     "E001",
		Slug:     "enum-string-mismatch",
		Severity: SeverityError,
		Trigger: "Your MDL has a comparison or assignment where one side is " +
			"an Enumeration attribute (or Enumeration parameter) and the " +
			"other side is a quoted string literal.",
		WhyWrong: "Mendix expressions cannot compare an Enumeration value " +
			"to a String. The comparison would always be false at runtime, " +
			"or trigger CE0109 in Studio Pro.",
		HowToFix: "Replace the string literal with the fully-qualified " +
			"enumeration value: 'NewAlert' → FraudDetection.AlertStatus.NewAlert",
		Examples: []ExampleFix{
			{
				Wrong: "CHANGE $Alert (Status = 'NewAlert')",
				Right: "CHANGE $Alert (Status = FraudDetection.AlertStatus.NewAlert)",
				Note:  "CREATE/CHANGE assignment",
			},
			{
				Wrong: "IF $Alert/Status = 'NewAlert' THEN ...",
				Right: "IF $Alert/Status = FraudDetection.AlertStatus.NewAlert THEN ...",
				Note:  "IF condition",
			},
			{
				Wrong: "CALL Mf($Status = 'Validated')",
				Right: "CALL Mf($Status = FraudDetection.AlertStatus.Validated)",
				Note:  "CALL parameter",
			},
		},
	},
}}
