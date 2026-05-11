// SPDX-License-Identifier: Apache-2.0

package hints

type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

type Hint struct {
	Code      string
	Slug      string
	Severity  Severity
	Where     Location
	YouWrote  string
	Problem   string
	Fix       string
	Reference *Reference
}

type Location struct {
	File      string
	Line      int
	Column    int
	Microflow string
	Context   string
}

type Reference struct {
	Enum            string
	EnumValues      []string
	FunctionName    string
	FunctionArgs    []string
	FunctionReturns string
	AttributeName   string
	AttributeType   string
	EntityType      string
}
