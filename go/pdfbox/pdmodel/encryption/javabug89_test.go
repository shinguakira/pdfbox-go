package encryption_test

// JAVA-BUGS 89: `PublicKeySecurityHandler.prepareForDecryption` derives the key
// of a document whose /V is neither 4 nor 5 from a 20-byte SHA-1 digest,
// whatever key length its crypt filter asks for, and copies the key out of it
// with System.arraycopy. A 256-bit key reads past the digest. Carried: the port
// slices the digest to the key length, and panics where PDFBox throws.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// TestVersion6CertificateEncryptionReadsPastTheDigest opens
// TestPublicKeyEncryption's AESkeylength256.pdf -- version 5, a 256-bit key,
// the certificate in PDFBOX-4421-keystore.pfx -- with its one "/V 5" written
// "/V 6", which is the version ISO/TS 32003 gives AES-GCM, and which iText
// writes: kernel/crypto/securityhandler/PubSecHandlerUsingAesGcmTest/externalFile.pdf
// and invalidCryptFilter.pdf reach the same line.
//
// PDFBox, given those bytes, the password and the alias TestReadPubkeyEncryptedAES256
// uses, answers "java.lang.ArrayIndexOutOfBoundsException: arraycopy: last source
// index 32 out of bounds for byte[20]" at PublicKeySecurityHandler.java:288.
// The port panics at the same copy.
func TestVersion6CertificateEncryptionReadsPastTheDigest(t *testing.T) {
	content := fileResourceAsByteArray(t, "AESkeylength256.pdf")
	if bytes.Count(content, []byte("/V 5")) != 1 {
		t.Fatalf("AESkeylength256.pdf does not hold \"/V 5\" exactly once")
	}
	version6 := bytes.Replace(content, []byte("/V 5"), []byte("/V 6"), 1)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		doc, err := pdfbox.LoadPDFFromWithKeyStore(pdfio.NewReadBufferBytes(version6),
			"w!z%C*F-JaNdRgUk", openKeyStore(t, "PDFBOX-4421-keystore.pfx"), "testnutzer")
		if err == nil {
			doc.Close()
		}
		t.Errorf("LoadPDFFromWithKeyStore returned (err = %v); PDFBox throws "+
			"ArrayIndexOutOfBoundsException from the key copy", err)
	}()
	if recovered == nil {
		return
	}
	if message := fmt.Sprint(recovered); !strings.Contains(message, "slice bounds out of range [:32] with capacity 20") {
		t.Errorf("panic %q, want the key copy reading 32 bytes out of 20", message)
	}
}
