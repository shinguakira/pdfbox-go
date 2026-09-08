package encryption

// Encrypting to a public key, which this port could not do.
//
// `PrepareDocumentForEncryption` returned an error saying it "needs a CMS
// encoder, which is slice 7". Slice 7 merged, and the CMS encoder was half
// here already: `rc2.go` encrypts as well as decrypts, and every ASN.1
// structure a CMS enveloped-data blob is made of came in with the decrypting
// side. What was missing was the encoding direction, and nothing went back to
// look.
//
// The case is a round trip, because that is the only assertion worth making
// about an envelope: what one half seals, the other half has to open.

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestPublicKeyEnvelopeRoundTrips seals the seed and permissions for a
// certificate and opens them again with its private key.
//
// It goes through `computeRecipientsField`, which is what the document's
// /Recipients entry carries, and back through `newCMSEnvelopedData`, which is
// what `PrepareForDecryption` reads.
func TestPublicKeyEnvelopeRoundTrips(t *testing.T) {
	certificate, key := selfSignedCertificate(t)

	policy := NewPublicKeyProtectionPolicy()
	recipient := &PublicKeyRecipient{}
	recipient.SetX509(certificate)
	// Permissions with printing taken away, so the four bytes carry something
	// that is not all ones.
	permission := NewAccessPermission()
	permission.SetCanPrint(false)
	recipient.SetPermission(permission)
	policy.AddRecipient(recipient)

	handler := NewPublicKeySecurityHandlerOfPolicy(policy)
	seed := make([]byte, 20)
	for i := range seed {
		seed[i] = byte(i * 7)
	}
	fields, err := handler.computeRecipientsField(seed)
	if err != nil {
		t.Fatalf("computeRecipientsField: %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("built %d recipient fields for one recipient", len(fields))
	}

	// Read it back the way PrepareForDecryption does.
	envelope, err := newCMSEnvelopedData(fields[0])
	if err != nil {
		t.Fatalf("the blob this port wrote did not parse: %v", err)
	}
	recipients := envelope.Recipients()
	if len(recipients) != 1 {
		t.Fatalf("the envelope holds %d recipients, want 1", len(recipients))
	}
	if !recipients[0].matches(certificate.RawIssuer, certificate.SerialNumber,
		certificate.SubjectKeyId) {
		t.Error("the recipient does not name the certificate it was written for")
	}

	content, err := recipients[0].content(key)
	if err != nil {
		t.Fatalf("unwrapping the envelope: %v", err)
	}
	if len(content) != 24 {
		t.Fatalf("the envelope holds %d bytes, want 24: twenty of seed and four "+
			"of permissions", len(content))
	}
	for i := range seed {
		if content[i] != seed[i] {
			t.Fatalf("byte %d of the seed came back as %#x, want %#x", i,
				content[i], seed[i])
		}
	}

	// The four permission bytes are big-endian, which is the order
	// computeRecipientsField writes them in and AccessPermission reads them in.
	got := NewAccessPermissionFromBytes(content[20:24])
	if got.CanPrint() {
		t.Error("the permissions came back allowing printing, which was taken away")
	}
	if !got.CanExtractContent() {
		t.Error("the permissions came back denying extraction, which was allowed")
	}
}

// TestPrepareDocumentForEncryption checks the whole method: the dictionary it
// writes and the key it derives.
func TestPrepareDocumentForEncryption(t *testing.T) {
	certificate, _ := selfSignedCertificate(t)

	policy := NewPublicKeyProtectionPolicy()
	recipient := &PublicKeyRecipient{}
	recipient.SetX509(certificate)
	recipient.SetPermission(NewAccessPermission())
	policy.AddRecipient(recipient)

	handler := NewPublicKeySecurityHandlerOfPolicy(policy)
	document := &fakeDocument{}
	if err := handler.PrepareDocumentForEncryption(document); err != nil {
		t.Fatalf("PrepareDocumentForEncryption: %v", err)
	}

	dictionary := document.encryption
	if dictionary == nil {
		t.Fatal("no encryption dictionary was stored on the document")
	}
	if got := dictionary.Filter(); got != PublicKeySecurityHandlerFilter {
		t.Errorf("the filter is %q, want %q", got, PublicKeySecurityHandlerFilter)
	}
	if got := dictionary.SubFilter(); got != publicKeySubFilter4 &&
		got != publicKeySubFilter5 {
		t.Errorf("the subfilter is %q, want one of the two adbe.pkcs7 ones", got)
	}
	if got := len(handler.EncryptionKey()); got != handler.KeyLength()/8 {
		t.Errorf("the encryption key is %d bytes and the key length is %d bits",
			got, handler.KeyLength())
	}
	if allZero(handler.EncryptionKey()) {
		t.Error("the encryption key is all zeros")
	}

	// The recipients are on the dictionary for version 2 and on the crypt
	// filter for 4 and 5, which is what the two subfilters mean.
	recipients := dictionary.COSObject().GetCOSArray(cos.Recipients)
	if recipients == nil {
		filter := dictionary.DefaultCryptFilterDictionary()
		if filter != nil {
			recipients = filter.COSObject().GetCOSArray(cos.Recipients)
		}
	}
	if recipients == nil {
		t.Fatal("the dictionary carries no /Recipients")
	}
	if got := recipients.Size(); got != 1 {
		t.Errorf("the dictionary carries %d recipients, want 1", got)
	}
}

// TestPrepareDocumentForEncryptionNeedsARecipient is the guard: a policy that
// names nobody cannot encrypt to anybody.
func TestPrepareDocumentForEncryptionNeedsARecipient(t *testing.T) {
	handler := NewPublicKeySecurityHandlerOfPolicy(NewPublicKeyProtectionPolicy())
	if err := handler.PrepareDocumentForEncryption(&fakeDocument{}); err == nil {
		t.Error("a policy with no recipients was accepted")
	}
}

// allZero reports whether every byte is zero.
func allZero(bytes []byte) bool {
	for _, b := range bytes {
		if b != 0 {
			return false
		}
	}
	return true
}

// selfSignedCertificate makes a certificate and its key, which is all a
// recipient is.
func selfSignedCertificate(t *testing.T) (*x509.Certificate, crypto.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating a key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(20260908),
		Subject:      pkix.Name{CommonName: "pdfbox-go recipient"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template,
		&key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating a certificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parsing the certificate back: %v", err)
	}
	return certificate, key
}

// fakeDocument is the least a PDDocumentLike can be, which is all
// PrepareDocumentForEncryption asks of one.
type fakeDocument struct {
	encryption *PDEncryption
	cosDoc     fakeCOSDocument
}

func (d *fakeDocument) Encryption() *PDEncryption { return d.encryption }

func (d *fakeDocument) SetEncryptionDictionary(encryption *PDEncryption) {
	d.encryption = encryption
}

func (d *fakeDocument) COSDocument() COSDocumentLike { return &d.cosDoc }

// fakeCOSDocument records the dictionary it is handed.
type fakeCOSDocument struct {
	dictionary *cos.Dictionary
	id         *cos.Array
}

func (d *fakeCOSDocument) DocumentID() *cos.Array { return d.id }

func (d *fakeCOSDocument) SetDocumentID(id *cos.Array) { d.id = id }

func (d *fakeCOSDocument) SetEncryptionDictionary(dictionary *cos.Dictionary) {
	d.dictionary = dictionary
}

func (d *fakeCOSDocument) String() string { return "fakeCOSDocument" }

// TestPublicKeyVersionNumber walks the four arms of computeVersionNumber,
// which is the base SecurityHandler method the port has on
// StandardSecurityHandler and which this handler needed as well.
//
// The values are the Java's, from SecurityHandler.computeVersionNumber.
func TestPublicKeyVersionNumber(t *testing.T) {
	for _, c := range []struct {
		keyLength int
		preferAES bool
		want      int
	}{
		{40, false, 1},
		{128, true, 4},
		{128, false, 2},
		{256, false, 5},
		{256, true, 5},
	} {
		policy := NewPublicKeyProtectionPolicy()
		if err := policy.SetEncryptionKeyLength(c.keyLength); err != nil {
			t.Fatalf("SetEncryptionKeyLength(%d): %v", c.keyLength, err)
		}
		policy.SetPreferAES(c.preferAES)
		handler := NewPublicKeySecurityHandlerOfPolicy(policy)
		if got := handler.computeVersionNumber(); got != c.want {
			t.Errorf("%d bits with preferAES %v gives version %d, want %d",
				c.keyLength, c.preferAES, got, c.want)
		}
	}
}
