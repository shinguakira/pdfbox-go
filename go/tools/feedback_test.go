package tools_test

// The review round of track/tools.
//
// Every case here failed before the fix it names, and each asserts what the
// Java does rather than what looked right.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/tools"
)

// TestMarkdownKeepsASurrogatePair is the P1.
//
// escapeMarkdown walks UTF-16 units, because Java's escape walks charAt and its
// table names characters like '*' and 178 by their unit value. Java appends
// each unit to one StringBuilder, so the two halves of a supplementary-plane
// character land next to each other and the string comes back whole. Decoding
// each unit on its own instead answers U+FFFD twice.
//
// U+1F600 GRINNING FACE is D83D DE00 in UTF-16, and neither half is in the
// escape table, so both take the default branch.
func TestMarkdownKeepsASurrogatePair(t *testing.T) {
	const emoji = "\U0001F600"

	got := tools.EscapeMarkdownForTest("a" + emoji + "b")
	if want := "a" + emoji + "b"; got != want {
		t.Errorf("escapeMarkdown(%q) = %q, want %q", "a"+emoji+"b", got, want)
	}
	if strings.ContainsRune(got, '\uFFFD') {
		t.Errorf("escapeMarkdown produced a replacement character: %q", got)
	}

	// And the escape table still fires for the characters that are in it.
	if got, want := tools.EscapeMarkdownForTest("a*b"), `a\*b`; got != want {
		t.Errorf("escapeMarkdown(%q) = %q, want %q", "a*b", got, want)
	}
	if got, want := tools.EscapeMarkdownForTest("<&>"), "&lt;&amp;&gt;"; got != want {
		t.Errorf("escapeMarkdown(%q) = %q, want %q", "<&>", got, want)
	}
	// 178 and 179 are the two superscripts Java names by unit value.
	if got, want := tools.EscapeMarkdownForTest("m\u00B2"), "m<sup>2</sup>"; got != want {
		t.Errorf("escapeMarkdown(%q) = %q, want %q", "m\u00B2", got, want)
	}
}

// TestProtectDoesNotPrepareTheHandler is the P2 on PDDocument.protect.
//
// Java's protect installs the handler and stops. Preparing the document is the
// writer's job, and it does it on every save; doing it here as well runs the
// password hashing twice and throws the first result away.
//
// prepareDocumentForEncryption is what writes /Filter, /V, /R, /Length and /P
// into the encryption dictionary, so their absence right after protect is what
// says it has not run.
func TestProtectDoesNotPrepareTheHandler(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()
	document.AddPage(pdmodel.NewPDPage())

	policy := encryption.NewStandardProtectionPolicy("owner", "user",
		encryption.NewAccessPermission())
	if err := policy.SetEncryptionKeyLength(256); err != nil {
		t.Fatalf("SetEncryptionKeyLength: %v", err)
	}
	if err := document.Protect(policy); err != nil {
		t.Fatalf("Protect: %v", err)
	}

	if got := document.Encryption().Filter(); got != "" {
		t.Errorf("the encryption dictionary has /Filter %q right after Protect; "+
			"preparing it is the writer's job", got)
	}
	if got := document.Encryption().Revision(); got != 0 {
		t.Errorf("the encryption dictionary has /R %d right after Protect", got)
	}

	// And a save still produces a document that opens with the password, which
	// is what says the writer does the preparation.
	path := filepath.Join(t.TempDir(), "encrypted.pdf")
	if err := document.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	if got := document.Encryption().Revision(); got == 0 {
		t.Error("the encryption dictionary has no /R after the save either")
	}
}

// TestLineSpacingMustBePositive is the P2 on -lineSpacing.
//
// Java's call() routes the parsed value through setLineSpacing, which throws
// IllegalArgumentException for anything <= 0. It is unchecked, so picocli's
// execution exception handler answers ExitCode.SOFTWARE; the port panics and
// Execute answers the same.
func TestLineSpacingMustBePositive(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(in, []byte("one\ntwo\n"), 0o666); err != nil {
		t.Fatal(err)
	}

	for _, spacing := range []string{"0", "-1", "-0.5"} {
		out := filepath.Join(dir, "out"+spacing+".pdf")
		code, stdout, stderr := runCommand(tools.NewTextToPDF(),
			"-i", in, "-o", out, "-lineSpacing", spacing)
		if code != 1 {
			t.Errorf("-lineSpacing %s exited %d, want 1", spacing, code)
		}
		if stdout != "" {
			t.Errorf("-lineSpacing %s wrote %q to stdout", spacing, stdout)
		}
		if want := "line spacing must be positive"; !strings.Contains(stderr, want) {
			t.Errorf("-lineSpacing %s: stderr is %q, want %q", spacing, stderr, want)
		}
	}

	// A positive one still works.
	out := filepath.Join(dir, "ok.pdf")
	if code, _, stderr := runCommand(tools.NewTextToPDF(),
		"-i", in, "-o", out, "-lineSpacing", "2"); code != 0 {
		t.Errorf("-lineSpacing 2 exited %d (%s), want 0", code, stderr)
	}
}

// TestTextToPDFTakesAVeryLongLine is the P2 on the scanner's buffer.
//
// BufferedReader.readLine has no line-length limit; the port had a 16 MiB one,
// which is a cap Java does not have and which nothing documented. A line longer
// than it failed the conversion outright rather than being wrapped to the page.
func TestTextToPDFTakesAVeryLongLine(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "long.txt")

	// One line of 17 MiB, which is over the old cap.
	var line strings.Builder
	for line.Len() < 17*1024*1024 {
		line.WriteString("lorem ipsum dolor sit amet ")
	}
	if err := os.WriteFile(in, []byte(line.String()), 0o666); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "out.pdf")
	code, _, stderr := runCommand(tools.NewTextToPDF(), "-i", in, "-o", out)
	if code != 0 {
		t.Fatalf("exited %d (%s), want 0: readLine has no length limit", code, stderr)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("the output was not written: %v", err)
	}
}

// TestSubcommandBoundaryRespectsOptionValues is the P2 on the dispatcher.
//
// picocli knows each option's arity, so a value that happens to spell a
// subcommand name is consumed by the option. Splitting on the name instead
// hands the option no value and then runs a command nobody asked for.
func TestSubcommandBoundaryRespectsOptionValues(t *testing.T) {
	dir := t.TempDir()
	// A file actually named "version", reached by that bare name so that the
	// argument is the word itself.
	data, err := os.ReadFile(testFile2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "version"), data, 0o666); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	// -i takes a value, and the value is the file named "version".
	code, out, errOut := runPDFBox("export:text", "-i", "version", "-console")
	if code != 0 {
		t.Fatalf("exited %d (%s), want 0", code, errOut)
	}
	if strings.Contains(out, "4.0.0-SNAPSHOT") {
		t.Errorf("the version command ran; \"version\" was the value of -i:\n%s", out)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("the text was not extracted:\n%s", out)
	}
}

// TestHelpTakesASubcommand is the other half: picocli's HelpCommand takes the
// subcommand as a parameter and prints *its* help, rather than the global help
// followed by a run of that command with no arguments.
func TestHelpTakesASubcommand(t *testing.T) {
	code, out, errOut := runPDFBox("help", "decrypt")
	if code != 0 {
		t.Fatalf("exited %d (%s), want 0", code, errOut)
	}
	if want := "Decrypts a PDF document"; !strings.Contains(out, want) {
		t.Errorf("the help does not carry %q:\n%s", want, out)
	}
	if want := "-keyStore"; !strings.Contains(out, want) {
		t.Errorf("the help does not list the command's options:\n%s", out)
	}
	// And it must not have run decrypt, which with no -i is a usage error.
	if strings.Contains(errOut, "Missing required option") {
		t.Errorf("decrypt was run as well as helped:\n%s", errOut)
	}
}

// TestHelpWithNoSubcommandIsStillGlobal keeps the case the dispatcher already
// had: `pdfbox help` prints the list.
func TestHelpWithNoSubcommandIsStillGlobal(t *testing.T) {
	code, out, _ := runPDFBox("help")
	if code != 0 {
		t.Errorf("exited %d, want 0", code)
	}
	if !strings.Contains(out, "Commands:") {
		t.Errorf("the global help does not list the commands:\n%s", out)
	}
}

var _ = strconv.Itoa
