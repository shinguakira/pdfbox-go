package encryption_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// What the slice 5 adversarial review asked, written down as tests.
//
// D8 asks what happens on the wrong password: Java's behaviour is specific, and
// the danger is not that the port reports something different but that it
// reports nothing and hands back garbage. Every case below has to fail, and
// fail with something a caller can act on.

// TestWrongDocumentPassword is the case testPermissions already covers, kept
// here beside the others so that the four failures read together.
func TestWrongDocumentPassword(t *testing.T) {
	_, err := pdfbox.LoadPDFBytesWithPassword(
		fileResourceAsByteArray(t, "PasswordSample-256bit.pdf"), "not the password")
	if err == nil {
		t.Fatal("a wrong password opened the document")
	}
	var invalid *encryption.InvalidPasswordError
	if !errors.As(err, &invalid) {
		t.Fatalf("a wrong password gave %T: %v", err, err)
	}
	if invalid.Error() != "Cannot decrypt PDF, the password is incorrect" {
		t.Errorf("message = %q", invalid.Error())
	}
}

// TestWrongKeyStorePassword checks that a keystore that will not open is
// reported rather than read as rubbish. Java's KeyStore.load throws where the
// MAC does not verify; the port checks the same MAC.
func TestWrongKeyStorePassword(t *testing.T) {
	_, err := pdfbox.LoadPDFFromWithKeyStore(
		pdfio.NewReadBufferBytes(fileResourceAsByteArray(t, "AESkeylength128.pdf")),
		"not the keystore password", openKeyStore(t, "PDFBOX-4421-keystore.pfx"), "testnutzer")
	if err == nil {
		t.Fatal("a wrong keystore password opened the document")
	}
	if !strings.Contains(err.Error(), "MAC") {
		t.Errorf("a wrong keystore password gave %q, want the MAC check to catch it",
			err.Error())
	}
}

// TestAliasIgnoredForASingleEntryKeyStore pins something surprising that the
// review turned up rather than assumed: PublicKeyDecryptionMaterial does not
// look at the alias at all when the keystore holds exactly one entry -- both
// getCertificate and getPrivateKey take the first one -- so a wrong alias opens
// the document. All four checked-in keystores hold one entry, so this is the
// path the ported tests take. It is what Java does, and the port does it too.
func TestAliasIgnoredForASingleEntryKeyStore(t *testing.T) {
	doc, err := pdfbox.LoadPDFFromWithKeyStore(
		pdfio.NewReadBufferBytes(fileResourceAsByteArray(t, "AES128ExposedMeta.pdf")),
		"", openKeyStore(t, "PDFBOX-5249.p12"), "no such alias")
	if err != nil {
		t.Fatalf("a one-entry keystore should ignore the alias: %v", err)
	}
	doc.Close()
}

// TestPublicKeyDocumentWithoutKeyStore checks the material mismatch: a document
// encrypted for a certificate, opened with a password.
func TestPublicKeyDocumentWithoutKeyStore(t *testing.T) {
	_, err := pdfbox.LoadPDFBytesWithPassword(
		fileResourceAsByteArray(t, "AESkeylength128.pdf"), "w!z%C*F-JaNdRgUk")
	if err == nil {
		t.Fatal("a public key document opened with a password")
	}
	if !strings.Contains(err.Error(), "not compatible") {
		t.Errorf("the material mismatch gave %q", err.Error())
	}
}

// TestPasswordDocumentWithKeyStore checks the mismatch the other way round. It
// never reaches the handler: Java loads the keystore with the same password
// before it builds the material, so the document password opens the keystore or
// nothing happens at all.
func TestPasswordDocumentWithKeyStore(t *testing.T) {
	_, err := pdfbox.LoadPDFFromWithKeyStore(
		pdfio.NewReadBufferBytes(fileResourceAsByteArray(t, "PasswordSample-128bit.pdf")),
		"user", openKeyStore(t, "PDFBOX-5249.p12"), "test")
	if err == nil {
		t.Fatal("a password document opened with a keystore")
	}
	if !strings.Contains(err.Error(), "MAC") {
		t.Errorf("the keystore should refuse the document password, got %q", err.Error())
	}
}
