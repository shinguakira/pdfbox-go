package tools_test

// The two encryption commands.
//
// **Neither has a Java test.** A2 asks for a decision either way, and both are
// worth one: each is a chain of library calls, but the *order* of its checks
// and the exit code each answers are the command's own, and the round trip --
// encrypt, then decrypt, then read the result -- is the only thing that says
// the pair works.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/tools"
)

// TestEncryptThenDecrypt is the round trip.
func TestEncryptThenDecrypt(t *testing.T) {
	dir := t.TempDir()
	encrypted := filepath.Join(dir, "encrypted.pdf")
	decrypted := filepath.Join(dir, "decrypted.pdf")

	code, _, errOut := runCommand(tools.NewEncrypt(),
		"-i", testFile2, "-o", encrypted, "-O", "owner", "-U", "user")
	if code != 0 {
		t.Fatalf("encrypt exited %d (%s), want 0", code, errOut)
	}

	// The result is encrypted, and needs the password.
	if _, err := pdfbox.LoadPDF(encrypted); err == nil {
		t.Error("the encrypted document opened with no password")
	}
	document, err := pdfbox.LoadPDFWithPassword(encrypted, "owner")
	if err != nil {
		t.Fatalf("opening the encrypted document with the owner password: %v", err)
	}
	if !document.IsEncrypted() {
		t.Error("the document is not encrypted")
	}
	document.Close()

	code, _, errOut = runCommand(tools.NewDecrypt(),
		"-i", encrypted, "-o", decrypted, "-password", "owner")
	if code != 0 {
		t.Fatalf("decrypt exited %d (%s), want 0", code, errOut)
	}

	// And it opens with no password again.
	roundTripped, err := pdfbox.LoadPDF(decrypted)
	if err != nil {
		t.Fatalf("opening the decrypted document: %v", err)
	}
	defer roundTripped.Close()
	if roundTripped.IsEncrypted() {
		t.Error("the decrypted document is still encrypted")
	}
}

// TestDecryptRefusesAnUnencryptedDocument is the exit code and the message of
// the branch Java takes when there is nothing to decrypt.
func TestDecryptRefusesAnUnencryptedDocument(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.pdf")
	code, stdout, stderr := runCommand(tools.NewDecrypt(), "-i", testFile2, "-o", out)
	if code != 1 {
		t.Errorf("exited %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("the error went to stdout as %q", stdout)
	}
	if want := "Error: Document is not encrypted."; !strings.Contains(stderr, want) {
		t.Errorf("stderr is %q, want %q", stderr, want)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("the output file was written; the command refused before saving")
	}
}

// TestEncryptRefusesAnEncryptedDocument is the other side, which Java reports
// and then answers 0 for.
//
// The document has to be one that *opens* with no password, because Encrypt
// loads with `Loader.loadPDF(infile)` and nothing else: a document with a user
// password throws InvalidPasswordException there and never reaches the branch.
// An owner password with an empty user password is the shape that does -- the
// document is encrypted, and anyone may open it.
func TestEncryptRefusesAnEncryptedDocument(t *testing.T) {
	dir := t.TempDir()
	encrypted := filepath.Join(dir, "encrypted.pdf")
	if code, _, errOut := runCommand(tools.NewEncrypt(),
		"-i", testFile2, "-o", encrypted, "-O", "owner", "-U", ""); code != 0 {
		t.Fatalf("encrypt exited %d (%s)", code, errOut)
	}

	code, _, stderr := runCommand(tools.NewEncrypt(),
		"-i", encrypted, "-o", filepath.Join(dir, "again.pdf"), "-O", "o", "-U", "u")
	if code != 0 {
		t.Errorf("exited %d, want 0: Java prints the error and answers 0", code)
	}
	if want := "Error: Document is already encrypted."; !strings.Contains(stderr, want) {
		t.Errorf("stderr is %q, want %q", stderr, want)
	}
}

// TestEncryptOnAUserEncryptedDocumentFailsToLoad is the branch that hides the
// one above: Encrypt loads with no password, so a document with a user password
// fails there and answers 4 rather than reporting that it is already encrypted.
func TestEncryptOnAUserEncryptedDocumentFailsToLoad(t *testing.T) {
	dir := t.TempDir()
	encrypted := filepath.Join(dir, "encrypted.pdf")
	if code, _, errOut := runCommand(tools.NewEncrypt(),
		"-i", testFile2, "-o", encrypted, "-O", "owner", "-U", "user"); code != 0 {
		t.Fatalf("encrypt exited %d (%s)", code, errOut)
	}

	code, _, stderr := runCommand(tools.NewEncrypt(),
		"-i", encrypted, "-o", filepath.Join(dir, "again.pdf"), "-O", "o", "-U", "u")
	if code != 4 {
		t.Errorf("exited %d, want 4", code)
	}
	if want := "Error encrypting PDF"; !strings.Contains(stderr, want) {
		t.Errorf("stderr is %q, want %q", stderr, want)
	}
	if strings.Contains(stderr, "already encrypted") {
		t.Errorf("stderr is %q; the load fails before that branch is reached", stderr)
	}
}

// TestEncryptPermissionsReachTheDocument checks the eight -can options are not
// merely parsed: -canExtractContent=false has to make it into the file.
func TestEncryptPermissionsReachTheDocument(t *testing.T) {
	encrypted := filepath.Join(t.TempDir(), "encrypted.pdf")
	code, _, errOut := runCommand(tools.NewEncrypt(),
		"-i", testFile2, "-o", encrypted, "-O", "owner", "-U", "user",
		"-canExtractContent=false", "-canPrint=false")
	if code != 0 {
		t.Fatalf("encrypt exited %d (%s), want 0", code, errOut)
	}

	// Opening with the *user* password gives the permissions that were set;
	// the owner password gives everything.
	document, err := pdfbox.LoadPDFWithPassword(encrypted, "user")
	if err != nil {
		t.Fatalf("opening with the user password: %v", err)
	}
	defer document.Close()
	permission := document.CurrentAccessPermission()
	if permission.CanExtractContent() {
		t.Error("-canExtractContent=false did not reach the document")
	}
	if permission.CanPrint() {
		t.Error("-canPrint=false did not reach the document")
	}
	if !permission.CanFillInForm() {
		t.Error("-canFillInForm defaults to true and was not set")
	}
}

// TestEncryptRefusesACertificate records the one branch that is not ported.
func TestEncryptRefusesACertificate(t *testing.T) {
	code, _, stderr := runCommand(tools.NewEncrypt(),
		"-i", testFile2, "-o", filepath.Join(t.TempDir(), "out.pdf"),
		"-certFile", "nosuch.cer")
	if code != 4 {
		t.Errorf("exited %d, want 4", code)
	}
	if want := "certFile"; !strings.Contains(stderr, want) {
		t.Errorf("stderr is %q, want it to name the option that is not supported", stderr)
	}
}

// runCommand runs one command on its own.
func runCommand(command tools.Command, args ...string) (int, string, string) {
	var stdout, stderr strings.Builder
	code := tools.Execute(command, args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
