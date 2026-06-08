// mdl/executor/widget_fmt_test_helpers_test.go
package executor

import (
	"bytes"
	"strings"
	"testing"
)

func newTestFormatCtx(t *testing.T) (*FormatContext, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	d := newDefaultDispatcher()
	return &FormatContext{Output: &buf, Indent: 0, Dispatcher: d}, &buf
}

func assertOutput(t *testing.T, buf *bytes.Buffer, want string) {
	t.Helper()
	got := buf.String()
	if !fmtContainsStr(got, want) {
		t.Errorf("output missing %q\nfull output:\n%s", want, got)
	}
}

func assertNotOutput(t *testing.T, buf *bytes.Buffer, notWant string) {
	t.Helper()
	if fmtContainsStr(buf.String(), notWant) {
		t.Errorf("output should not contain %q\nfull output:\n%s", notWant, buf.String())
	}
}

func fmtContainsStr(s, sub string) bool {
	return len(sub) == 0 || strings.Contains(s, sub)
}
