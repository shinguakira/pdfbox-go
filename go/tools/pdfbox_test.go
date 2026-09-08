package tools_test

// The dispatcher.
//
// Its Java tests, PDFBoxHeadlessTest and PDFBoxNonHeadlessTest, check that the
// subcommand list is exactly what PDFBox.main registers -- and both do it
// through picocli's CommandSpec, which has no counterpart here. What they are
// really asserting is that a name a caller types reaches the command it should,
// and that is what these cases assert instead. Recorded in migration/STATUS.md.

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/tools"
)

// TestSubcommandNamesAreJavas checks every name PDFBox.main registers for a
// command this port built.
func TestSubcommandNamesAreJavas(t *testing.T) {
	// The names Java adds, in PDFBox.main's order. "debug" is not here: it is
	// PDFDebugger, a module of its own and out of scope, and Java only adds it
	// when the JVM is not headless.
	built := map[string]bool{
		"decrypt": true, "encrypt": true, "decode": true,
		"export:images": true, "export:xmp": true, "export:text": true,
		"export:fdf": true, "export:xfdf": true,
		"import:fdf": true, "import:xfdf": true,
		"split": true, "fromimage": true, "fromtext": true,
		"version": true,
		"merge":   true,
		"overlay": true,
	}
	// And the ones it registers that this port does not build.
	notBuilt := map[string]bool{
		"print": true, "render": true,
	}

	got := map[string]bool{}
	for _, sub := range tools.Subcommands() {
		got[sub.Name] = true
	}

	for name := range built {
		if !got[name] {
			t.Errorf("the dispatcher does not register %q", name)
		}
	}
	for name := range got {
		if !built[name] {
			t.Errorf("the dispatcher registers %q, which PDFBox.main does not", name)
		}
	}
	// Every name Java has and this port does not must be in NotBuiltCommands,
	// so that the help can say what it waits for.
	recorded := map[string]bool{}
	for _, missing := range tools.NotBuiltCommands {
		if missing.Name != "" {
			recorded[missing.Name] = true
		}
	}
	for name := range notBuilt {
		if !recorded[name] {
			t.Errorf("%q is neither built nor recorded as not built", name)
		}
	}
}

// TestSubcommandNamesAreCaseInsensitive is
// setSubcommandsCaseInsensitive(true).
func TestSubcommandNamesAreCaseInsensitive(t *testing.T) {
	for _, name := range []string{"version", "VERSION", "Version"} {
		code, out, errOut := runPDFBox(name)
		if code != 0 {
			t.Errorf("%q exited %d (%s), want 0", name, code, errOut)
		}
		if !strings.Contains(out, "4.0.0-SNAPSHOT") {
			t.Errorf("%q printed %q, want the version", name, out)
		}
	}
}

// TestNoSubcommandIsAUsageError is PDFBox.run, which throws ParameterException
// with "Error: Subcommand required".
func TestNoSubcommandIsAUsageError(t *testing.T) {
	code, out, errOut := runPDFBox()
	if code != 2 {
		t.Errorf("exited %d, want 2", code)
	}
	if out != "" {
		t.Errorf("the error went to stdout as %q", out)
	}
	if want := "Error: Subcommand required"; !strings.Contains(errOut, want) {
		t.Errorf("stderr is %q, want %q", errOut, want)
	}
}

// TestAnUnknownSubcommandIsAUsageError checks a name that is not registered.
func TestAnUnknownSubcommandIsAUsageError(t *testing.T) {
	code, out, errOut := runPDFBox("nosuchcommand")
	if code != 2 {
		t.Errorf("exited %d, want 2", code)
	}
	if out != "" {
		t.Errorf("the error went to stdout as %q", out)
	}
	if !strings.Contains(errOut, "nosuchcommand") {
		t.Errorf("stderr is %q, want it to name the argument", errOut)
	}
}

// TestHelpNamesWhatIsNotBuilt is B10: a caller who asks for `render` should
// learn what it waits for, not that pdfbox has no such thing. The commands it
// does build are named too.
func TestHelpNamesWhatIsNotBuilt(t *testing.T) {
	_, out, _ := runPDFBox("help")
	for _, want := range []string{
		"merge", "overlay",
		"render", "rendering.Backend",
		"print",
		"export:images",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the help does not mention %q:\n%s", want, out)
		}
	}
}
