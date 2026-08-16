package generator //nolint:testpackage // codeWriter's fields/methods are unexported; this test needs white-box access

import "testing"

func TestCodeWriter(t *testing.T) {
	t.Parallel()

	var w codeWriter

	w.writeString("package main\n\n")
	w.printf("func %s() {}\n", "main")

	want := "package main\n\nfunc main() {}\n"
	if got := w.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
