package tools_test

// JAVA-BUGS 79: `Encrypt` builds the recipient once, outside the loop over
// `-certFile`, and the loop overwrites its certificate and adds the same
// object again -- so a document encrypted for several certificates is
// encrypted for the last one, several times over.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/tools"
)

// encryptionFixtureDir holds the two certificates and the two keystores that
// go with them, which the encryption tests already use and which were not made
// by this port.
const encryptionFixtureDir = "../../pdfbox/src/test/resources/org/apache/pdfbox/encryption/"

// TestEncryptToTwoCertificatesReachesBoth is the defect.
//
// `-certFile` is documented as repeatable -- "path to a certificate file, can
// be used multiple times" -- so a document named with two of them opens with
// either one's key. Java's policy holds two references to one recipient
// carrying the second certificate, and the first key opens nothing.
func TestEncryptToTwoCertificatesReachesBoth(t *testing.T) {
	for _, name := range []string{"test1.der", "test2.der", "test1.pfx", "test2.pfx"} {
		if _, err := os.Stat(encryptionFixtureDir + name); err != nil {
			t.Skipf("%s is not in this repository: %v", name, err)
		}
	}
	out := filepath.Join(t.TempDir(), "out.pdf")

	code, _, stderr := runCommand(tools.NewEncrypt(), "-i", testFile2, "-o", out,
		"-certFile", encryptionFixtureDir+"test1.der",
		"-certFile", encryptionFixtureDir+"test2.der")
	if code != 0 {
		t.Fatalf("exited %d, want 0; stderr is %q", code, stderr)
	}

	for _, recipient := range []struct{ password, keyStore string }{
		{"test1", "test1.pfx"},
		{"test2", "test2.pfx"},
	} {
		keyStore, err := os.Open(encryptionFixtureDir + recipient.keyStore)
		if err != nil {
			t.Fatalf("opening %s: %v", recipient.keyStore, err)
		}
		document, err := pdfbox.LoadPDFWithKeyStore(out, recipient.password, keyStore, "")
		keyStore.Close()
		if err != nil {
			t.Errorf("%s does not open the document: %v -- the certificate it "+
				"belongs to was named on the command line", recipient.keyStore, err)
			continue
		}
		document.Close()
	}
}
