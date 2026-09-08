package encryption_test

// The three cases of TestPublicKeyEncryption that encrypt a document.
//
// `publickey_test.go` ported four of the seven and deferred these three,
// because they "encrypt a document and save it, which needs the writer of slice
// 7 and the CMS encoder that goes with it". Slice 7 merged long ago and
// `track/stale-deferrals` wrote the CMS encoder -- and then left this deferral
// standing, which is the very thing the branch exists to catch. Caught in
// review instead.
//
// These are the end-to-end cases and the synthetic round trip in
// `publickeyencrypt_test.go` does not stand in for them: that one seals and
// opens an envelope, and these protect a real document, save it, and open the
// file again with a keystore this port did not write -- at three key lengths,
// with the wrong certificate, and with two recipients.

import (
	"crypto/x509"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// keyLengths is the Java's @MethodSource("keyLengths").
var keyLengths = []int{40, 128, 256}

// permission1 and permission2 are the two AccessPermissions of the Java's
// setUp, field for field. They differ in one bit: the second may print.
func permission1() *encryption.AccessPermission {
	permission := encryption.NewAccessPermission()
	permission.SetCanAssembleDocument(false)
	permission.SetCanExtractContent(false)
	permission.SetCanExtractForAccessibility(true)
	permission.SetCanFillInForm(false)
	permission.SetCanModify(false)
	permission.SetCanModifyAnnotations(false)
	permission.SetCanPrint(false)
	permission.SetCanPrintFaithful(false)
	return permission
}

func permission2() *encryption.AccessPermission {
	permission := permission1()
	permission.SetCanPrint(true) // it is true now !
	return permission
}

// recipientOf reads one of the two checked-in certificates.
func recipientOf(t *testing.T, name string,
	permission *encryption.AccessPermission) *encryption.PublicKeyRecipient {
	t.Helper()
	der := fileResourceAsByteArray(t, name)
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	recipient := &encryption.PublicKeyRecipient{}
	recipient.SetX509(certificate)
	recipient.SetPermission(permission)
	return recipient
}

// protectAndSave is the Java's `document.protect(policy)` followed by `save`.
//
// The document is `test.pdf` at version 1.7, which is what setUp loads and
// what it sets.
func protectAndSave(t *testing.T, policy *encryption.PublicKeyProtectionPolicy,
	keyLength int) (path, wantText, wantProducer string) {
	t.Helper()
	document, err := pdfbox.LoadPDFBytes(fileResourceAsByteArray(t, "test.pdf"))
	if err != nil {
		t.Fatalf("loading test.pdf: %v", err)
	}
	defer document.Close()
	wantText = extractedTextOf(t, document)
	wantProducer = document.DocumentInformation().Producer()
	document.SetVersion(1.7)

	if err := policy.SetEncryptionKeyLength(keyLength); err != nil {
		t.Fatalf("SetEncryptionKeyLength(%d): %v", keyLength, err)
	}
	if err := document.Protect(policy); err != nil {
		t.Fatalf("Protect: %v", err)
	}

	path = filepath.Join(t.TempDir(), "protected.pdf")
	if err := document.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	return path, wantText, wantProducer
}

// reloadProtected is the Java's reload: open the file with a keystore and
// check the text and the producer survived.
func reloadProtected(t *testing.T, path, password, keyStoreName,
	wantText, wantProducer string) *pdmodel.PDDocument {
	t.Helper()
	document, err := pdfbox.LoadPDFWithKeyStore(path, password,
		openKeyStore(t, keyStoreName), "")
	if err != nil {
		t.Fatalf("reloading with %s: %v", keyStoreName, err)
	}
	if got := extractedTextOf(t, document); got != wantText {
		t.Errorf("Extracted text is different: %q, want %q", got, wantText)
	}
	if got := document.DocumentInformation().Producer(); got != wantProducer {
		t.Errorf("Producer is different: %q, want %q", got, wantProducer)
	}
	return document
}

// extractedTextOf is `new PDFTextStripper().getText(document)`.
func extractedTextOf(t *testing.T, document *pdmodel.PDDocument) string {
	t.Helper()
	stripper := text.NewPDFTextStripper()
	var out strings.Builder
	stripper.SetOutput(&out)
	if err := stripper.ProcessPages(document.Pages()); err != nil {
		t.Fatalf("ProcessPages: %v", err)
	}
	return out.String()
}

// checkPermission1 is the block testProtection and the first half of
// testMultipleRecipients repeat, assertion for assertion.
func checkPermission1(t *testing.T, permission *encryption.AccessPermission) {
	t.Helper()
	checkPermissions(t, permission, false)
}

// checkPermissions asserts the eight bits, with printing as the caller says.
func checkPermissions(t *testing.T, permission *encryption.AccessPermission,
	canPrint bool) {
	t.Helper()
	for _, c := range []struct {
		name string
		got  bool
		want bool
	}{
		{"canAssembleDocument", permission.CanAssembleDocument(), false},
		{"canExtractContent", permission.CanExtractContent(), false},
		{"canExtractForAccessibility", permission.CanExtractForAccessibility(), true},
		{"canFillInForm", permission.CanFillInForm(), false},
		{"canModify", permission.CanModify(), false},
		{"canModifyAnnotations", permission.CanModifyAnnotations(), false},
		{"canPrint", permission.CanPrint(), canPrint},
		{"canPrintFaithful", permission.CanPrintFaithful(), false},
	} {
		if c.got != c.want {
			t.Errorf("%s is %v, want %v", c.name, c.got, c.want)
		}
	}
}

// TestProtection is testProtection: protect with a certificate, save, and open
// it with the matching private key.
func TestProtection(t *testing.T) {
	for _, keyLength := range keyLengths {
		t.Run(keyLengthName(keyLength), func(t *testing.T) {
			policy := encryption.NewPublicKeyProtectionPolicy()
			policy.AddRecipient(recipientOf(t, "test1.der", permission1()))
			path, wantText, wantProducer := protectAndSave(t, policy, keyLength)

			document := reloadProtected(t, path, "test1", "test1.pfx",
				wantText, wantProducer)
			defer document.Close()

			if !document.IsEncrypted() {
				t.Error("the reloaded document is not encrypted")
			}
			checkPermission1(t, document.CurrentAccessPermission())
		})
	}
}

// TestProtectionError is testProtectionError: the wrong certificate does not
// open the document, and says which recipient it did not match.
func TestProtectionError(t *testing.T) {
	for _, keyLength := range keyLengths {
		t.Run(keyLengthName(keyLength), func(t *testing.T) {
			policy := encryption.NewPublicKeyProtectionPolicy()
			policy.AddRecipient(recipientOf(t, "test1.der", permission1()))
			path, _, _ := protectAndSave(t, policy, keyLength)

			_, err := pdfbox.LoadPDFWithKeyStore(path, "test2",
				openKeyStore(t, "test2.pfx"), "")
			if err == nil {
				t.Fatal("No exception when using an incorrect decryption key")
			}
			// The Java asserts this substring, which comes out of
			// appendCertInfo comparing the recipient's serial number with the
			// certificate's.
			const want = "serial-#: rid 2 vs. cert 3"
			if !strings.Contains(err.Error(), want) {
				t.Errorf("not the expected exception: %v", err)
			}
		})
	}
}

// TestMultipleRecipients is testMultipleRecipients: two certificates, and each
// of them opens the document with its own permissions.
func TestMultipleRecipients(t *testing.T) {
	for _, keyLength := range keyLengths {
		t.Run(keyLengthName(keyLength), func(t *testing.T) {
			policy := encryption.NewPublicKeyProtectionPolicy()
			policy.AddRecipient(recipientOf(t, "test1.der", permission1()))
			policy.AddRecipient(recipientOf(t, "test2.der", permission2()))
			path, wantText, wantProducer := protectAndSave(t, policy, keyLength)

			// Open the first time, with the first recipient's key.
			first := reloadProtected(t, path, "test1", "test1.pfx",
				wantText, wantProducer)
			checkPermissions(t, first.CurrentAccessPermission(), false)
			if err := first.Close(); err != nil {
				t.Fatal(err)
			}

			// Open the second time, with the second recipient's key, whose
			// permissions allow printing.
			second := reloadProtected(t, path, "test2", "test2.pfx",
				wantText, wantProducer)
			checkPermissions(t, second.CurrentAccessPermission(), true)
			if err := second.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// keyLengthName names a subtest.
func keyLengthName(keyLength int) string {
	switch keyLength {
	case 40:
		return "40bit"
	case 128:
		return "128bit"
	default:
		return "256bit"
	}
}
