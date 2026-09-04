package encryption_test

import (
	"os"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.pdfbox.encryption.TestPublicKeyEncryption.
//
// Four of its seven tests are ported: the four that read a file encrypted by
// something other than PDFBox and open it with a certificate from a checked-in
// keystore. testProtection, testProtectionError and testMultipleRecipients
// encrypt a document and save it, which needs the writer of slice 7 and the CMS
// encoder that goes with it.
//
// The four here are the evidence that matters for this branch: neither the
// files nor the keystores were made by this port, so nothing about them can
// pass by agreeing with itself.

func openKeyStore(t *testing.T, name string) *os.File {
	t.Helper()
	file, err := os.Open(encryptionFixture + name)
	if err != nil {
		t.Fatalf("opening %s: %v", name, err)
	}
	t.Cleanup(func() { file.Close() })
	return file
}

// checkPublicKeyRead is the body the four tests share: open the file with the
// keystore, and check the handler, the key length and the text.
func checkPublicKeyRead(t *testing.T, pdfName, password, keyStoreName, alias string,
	wantKeyLength int, wantText string) {
	t.Helper()
	doc, err := pdfbox.LoadPDFFromWithKeyStore(
		pdfio.NewReadBufferBytes(fileResourceAsByteArray(t, pdfName)),
		password, openKeyStore(t, keyStoreName), alias)
	if err != nil {
		t.Fatalf("loading %s: %v", pdfName, err)
	}
	defer doc.Close()

	handler, err := doc.Encryption().SecurityHandler()
	if err != nil {
		t.Fatalf("SecurityHandler: %v", err)
	}
	if _, ok := handler.(*encryption.PublicKeySecurityHandler); !ok {
		t.Errorf("the security handler is %T, want a PublicKeySecurityHandler", handler)
	}
	if got := handler.KeyLength(); got != wantKeyLength {
		t.Errorf("KeyLength = %d, want %d", got, wantKeyLength)
	}

	stripper := text.NewPDFTextStripper()
	stripper.SetLineSeparator("\n")
	var builder strings.Builder
	stripper.SetOutput(&builder)
	if err := stripper.ProcessPages(doc.Pages()); err != nil {
		t.Fatalf("ProcessPages: %v", err)
	}
	if got := strings.TrimSpace(builder.String()); got != wantText {
		t.Errorf("text = %q, want %q", got, wantText)
	}
}

// TestReadPubkeyEncryptedAES128 is PDFBOX-4421: read a file encrypted with
// AES128 but not with PDFBox, and with a missing /Length entry.
func TestReadPubkeyEncryptedAES128(t *testing.T) {
	checkPublicKeyRead(t, "AESkeylength128.pdf", "w!z%C*F-JaNdRgUk",
		"PDFBOX-4421-keystore.pfx", "testnutzer", 128, "Key length: 128")
}

// TestReadPubkeyEncryptedAES256 is PDFBOX-4421 for the 256-bit file.
func TestReadPubkeyEncryptedAES256(t *testing.T) {
	checkPublicKeyRead(t, "AESkeylength256.pdf", "w!z%C*F-JaNdRgUk",
		"PDFBOX-4421-keystore.pfx", "testnutzer", 256, "Key length: 256")
}

// TestReadPubkeyEncryptedAES128withMetadataExposed is PDFBOX-5249: read a file
// encrypted with AES128 but not with PDFBox, and with exposed Metadata.
func TestReadPubkeyEncryptedAES128withMetadataExposed(t *testing.T) {
	checkPublicKeyRead(t, "AES128ExposedMeta.pdf", "", "PDFBOX-5249.p12", "test", 128,
		"AES key length: 128\nwith exposed Metadata")
}

// TestReadPubkeyEncryptedAES256withMetadataExposed is PDFBOX-5249 for the
// 256-bit file.
func TestReadPubkeyEncryptedAES256withMetadataExposed(t *testing.T) {
	checkPublicKeyRead(t, "AES256ExposedMeta.pdf", "", "PDFBOX-5249.p12", "test", 256,
		"AES key length: 256 \nwith exposed Metadata")
}
