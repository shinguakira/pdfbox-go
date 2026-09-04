package encryption_test

import (
	"errors"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
)

// Port of org.apache.pdfbox.encryption.TestSymmetricKeyEncryption.
//
// Java keeps its encryption tests in org.apache.pdfbox.encryption, a package
// away from the org.apache.pdfbox.pdmodel.encryption they test; the port keeps
// them beside the package, in its external test package, which is what stops
// the loader import from closing a cycle.
//
// Of the seven Java tests, one is ported whole and six are not:
//
//   - testPermissions is here. Its three files were made with Adobe Acrobat
//     rather than with PDFBox, which is what makes them worth reading: a round
//     trip through this port would pass even if both halves were wrong.
//   - testProtection, testProtectionInnerAttachment, testPDFBox4308 and
//     testPDFBox4453 encrypt a document and save it. The writer is slice 7.
//   - testPDFBox5955 and testPDFBox5639 read PDFs the Java build downloads into
//     target/pdfs, which this repository does not carry.
//
// checkPerms also renders page 0, which is slice 9; the port leaves that line
// out and asserts the permissions, which is what the test is named for.

// encryptionFixture is where the Java test resources of this package live,
// relative to it.
const encryptionFixture = "../../../../pdfbox/src/test/resources/org/apache/pdfbox/encryption/"

func fileResourceAsByteArray(t *testing.T, testFileName string) []byte {
	t.Helper()
	content, err := os.ReadFile(encryptionFixture + testFileName)
	if err != nil {
		t.Fatalf("reading %s: %v", testFileName, err)
	}
	return content
}

// TestPermissions checks that permissions work as intended: the user psw
// ("user") is enough to open the PDF with possibly restricted rights, the owner
// psw ("owner") gives full permissions. The 3 files of this test were created
// by Maruan Sahyoun, NOT with PDFBox, but with Adobe Acrobat to ensure "the
// gold standard". The restricted permissions prevent printing and text
// extraction. In the 128 and 256 bit encrypted files, AssembleDocument,
// ExtractForAccessibility and PrintDegraded are also disabled.
func TestPermissions(t *testing.T) {
	fullAP := encryption.NewAccessPermission()
	restrAP := encryption.NewAccessPermission()
	restrAP.SetCanPrint(false)
	restrAP.SetCanExtractContent(false)
	restrAP.SetCanModify(false)

	checkSeveralPerms(t, fileResourceAsByteArray(t, "PasswordSample-40bit.pdf"), fullAP, restrAP)

	restrAP.SetCanAssembleDocument(false)
	restrAP.SetCanExtractForAccessibility(false)
	restrAP.SetCanPrintFaithful(false)

	checkSeveralPerms(t, fileResourceAsByteArray(t, "PasswordSample-128bit.pdf"), fullAP, restrAP)
	checkSeveralPerms(t, fileResourceAsByteArray(t, "PasswordSample-256bit.pdf"), fullAP, restrAP)
}

func checkSeveralPerms(t *testing.T, inputFileAsByteArray1 []byte,
	fullAP, restrAP *encryption.AccessPermission) {
	t.Helper()
	checkPerms(t, inputFileAsByteArray1, "owner", fullAP)
	checkPerms(t, inputFileAsByteArray1, "user", restrAP)

	_, err := pdfbox.LoadPDFBytesWithPassword(inputFileAsByteArray1, "")
	if err == nil {
		t.Fatal("wrong password not detected")
	}
	var invalid *encryption.InvalidPasswordError
	if !errors.As(err, &invalid) {
		t.Fatalf("wrong password not detected: got %T: %v", err, err)
	}
	if invalid.Error() != "Cannot decrypt PDF, the password is incorrect" {
		t.Errorf("message = %q, want %q", invalid.Error(),
			"Cannot decrypt PDF, the password is incorrect")
	}
}

func checkPerms(t *testing.T, inputFileAsByteArray []byte, password string,
	expectedPermissions *encryption.AccessPermission) {
	t.Helper()
	doc, err := pdfbox.LoadPDFBytesWithPassword(inputFileAsByteArray, password)
	if err != nil {
		t.Fatalf("loading with password %q: %v", password, err)
	}
	defer doc.Close()

	currentAccessPermission := doc.CurrentAccessPermission()

	// check permissions
	if got, want := currentAccessPermission.IsOwnerPermission(),
		expectedPermissions.IsOwnerPermission(); got != want {
		t.Errorf("password %q: IsOwnerPermission = %v, want %v", password, got, want)
	}
	if !expectedPermissions.IsOwnerPermission() && !currentAccessPermission.IsReadOnly() {
		t.Errorf("password %q: IsReadOnly = false, want true", password)
	}
	checks := []struct {
		name string
		got  bool
		want bool
	}{
		{"CanAssembleDocument", currentAccessPermission.CanAssembleDocument(),
			expectedPermissions.CanAssembleDocument()},
		{"CanExtractContent", currentAccessPermission.CanExtractContent(),
			expectedPermissions.CanExtractContent()},
		{"CanExtractForAccessibility", currentAccessPermission.CanExtractForAccessibility(),
			expectedPermissions.CanExtractForAccessibility()},
		{"CanFillInForm", currentAccessPermission.CanFillInForm(),
			expectedPermissions.CanFillInForm()},
		{"CanModify", currentAccessPermission.CanModify(),
			expectedPermissions.CanModify()},
		{"CanModifyAnnotations", currentAccessPermission.CanModifyAnnotations(),
			expectedPermissions.CanModifyAnnotations()},
		{"CanPrint", currentAccessPermission.CanPrint(), expectedPermissions.CanPrint()},
		{"CanPrintFaithful", currentAccessPermission.CanPrintFaithful(),
			expectedPermissions.CanPrintFaithful()},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("password %q: %s = %v, want %v", password, c.name, c.got, c.want)
		}
	}
}
