package generator

import (
	"fmt"
	"strings"
)

// codeWriter accumulates generated Go source (or another short in-memory
// string result) as the generator renders it. It wraps strings.Builder,
// whose Write and WriteString methods are documented to always return a
// nil error, so callers of codeWriter never need to check one: the
// justification for ignoring that error lives here, once, instead of at
// every call site.
type codeWriter struct {
	buf strings.Builder
}

// String returns everything written so far.
func (w *codeWriter) String() string {
	return w.buf.String()
}

// writeString appends s.
func (w *codeWriter) writeString(s string) {
	w.buf.WriteString(s)
}

// printf appends a formatted string.
func (w *codeWriter) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(&w.buf, format, args...)
}
