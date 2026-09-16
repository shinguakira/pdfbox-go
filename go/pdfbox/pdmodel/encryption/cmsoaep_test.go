package encryption

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"strings"
	"testing"
)

// TestOAEPRecipientUnwraps opens envelopes whose content key is wrapped with
// RSAES-OAEP, as iText wraps it.
//
// PDFBox unwraps a recipient through BouncyCastle's JceKeyTransEnvelopedRecipient,
// which takes the key encryption algorithm the recipient names: RSA with PKCS#1
// v1.5 padding, or RSAES-OAEP with the hash, mask generation hash and label its
// parameters carry. iText writes OAEP with an empty parameter sequence, every
// field left at its default. Four of iText's test PDFs are encrypted that way:
// kernel/crypto/PdfEncryptionManuallyPortedTest/encryptedWithCertificateAes128.pdf
// and kernel/crypto/pdfencryption/PdfEncryptionTest/encryptedWithCertificateAes128.pdf,
// in each of its two repositories. PDFBox opens all four and extracts "Hello
// world!" from them. The port read only the PKCS#1 kind and refused all four
// with "key encryption algorithm 1.2.840.113549.1.1.7 is not supported".
func TestOAEPRecipientUnwraps(t *testing.T) {
	certificate, key := selfSignedCertificate(t)

	mgf1 := func(hash asn1.ObjectIdentifier) algorithmIdentifier {
		inner, err := asn1.Marshal(algorithmIdentifier{Algorithm: hash, Parameters: asn1.RawValue{Tag: asn1.TagNull}})
		if err != nil {
			t.Fatal(err)
		}
		return algorithmIdentifier{Algorithm: oidMGF1, Parameters: asn1.RawValue{FullBytes: inner}}
	}
	marshal := func(params rsaesOAEPParams) []byte {
		der, err := asn1.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		return der
	}
	label, err := asn1.Marshal([]byte("pdfbox-go"))
	if err != nil {
		t.Fatal(err)
	}
	sha256 := algorithmIdentifier{Algorithm: oidSHA256, Parameters: asn1.RawValue{Tag: asn1.TagNull}}

	cases := []struct {
		name       string
		parameters []byte // nil leaves the parameters out
		options    rsa.OAEPOptions
	}{
		{
			name:       "an empty parameter sequence, as iText writes it",
			parameters: []byte{0x30, 0x00},
			options:    rsa.OAEPOptions{Hash: crypto.SHA1, MGFHash: crypto.SHA1},
		},
		{
			name:    "no parameters at all",
			options: rsa.OAEPOptions{Hash: crypto.SHA1, MGFHash: crypto.SHA1},
		},
		{
			name:       "SHA-256 for the hash and the mask",
			parameters: marshal(rsaesOAEPParams{HashAlgorithm: sha256, MaskGenAlgorithm: mgf1(oidSHA256)}),
			options:    rsa.OAEPOptions{Hash: crypto.SHA256, MGFHash: crypto.SHA256},
		},
		{
			name:       "SHA-256 for the hash and the default SHA-1 for the mask",
			parameters: marshal(rsaesOAEPParams{HashAlgorithm: sha256}),
			options:    rsa.OAEPOptions{Hash: crypto.SHA256, MGFHash: crypto.SHA1},
		},
		{
			name: "a label",
			parameters: marshal(rsaesOAEPParams{PSourceAlgorithm: algorithmIdentifier{
				Algorithm: oidPSpecified, Parameters: asn1.RawValue{FullBytes: label}}}),
			options: rsa.OAEPOptions{Hash: crypto.SHA1, MGFHash: crypto.SHA1, Label: []byte("pdfbox-go")},
		},
	}

	// What a /Recipients envelope holds: twenty bytes of seed and four of
	// permissions.
	content := []byte("twenty bytes of seed.four")[:24]
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			blob := oaepEnvelope(t, certificate, content, c.parameters, &c.options)
			envelope, err := newCMSEnvelopedData(blob)
			if err != nil {
				t.Fatalf("the envelope did not parse: %v", err)
			}
			recipients := envelope.Recipients()
			if len(recipients) != 1 {
				t.Fatalf("%d recipients, want 1", len(recipients))
			}
			got, err := recipients[0].content(key)
			if err != nil {
				t.Fatalf("unwrapping: %v", err)
			}
			if !bytes.Equal(got, content) {
				t.Errorf("unwrapped %q, want %q", got, content)
			}
		})
	}

	// A mask generation function other than MGF1 is not one BouncyCastle's
	// cipher has either; it is refused, not guessed at.
	t.Run("a mask generation function that is not MGF1", func(t *testing.T) {
		parameters := marshal(rsaesOAEPParams{MaskGenAlgorithm: algorithmIdentifier{
			Algorithm: asn1.ObjectIdentifier{1, 2, 3, 4}, Parameters: asn1.RawValue{Tag: asn1.TagNull}}})
		blob := oaepEnvelope(t, certificate, content, parameters,
			&rsa.OAEPOptions{Hash: crypto.SHA1, MGFHash: crypto.SHA1})
		envelope, err := newCMSEnvelopedData(blob)
		if err != nil {
			t.Fatalf("the envelope did not parse: %v", err)
		}
		_, err = envelope.Recipients()[0].content(key)
		if err == nil || !strings.Contains(err.Error(), "mask generation function 1.2.3.4") {
			t.Errorf("err = %v, want the mask generation function refused", err)
		}
	})
}

// oaepEnvelope builds a CMS enveloped-data blob whose AES-256-CBC content key is
// wrapped for the certificate with RSAES-OAEP under the given options, naming
// the given parameters.
func oaepEnvelope(t *testing.T, certificate *x509.Certificate, content, parameters []byte,
	options *rsa.OAEPOptions) []byte {
	t.Helper()
	contentKey := make([]byte, 32)
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(contentKey); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(iv); err != nil {
		t.Fatal(err)
	}
	block, err := aes.NewCipher(contentKey)
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := rsa.EncryptOAEPWithOptions(rand.Reader, certificate.PublicKey.(*rsa.PublicKey),
		contentKey, options)
	if err != nil {
		t.Fatal(err)
	}

	serial, err := asn1.Marshal(issuerAndSerialNumber{
		Issuer:       asn1.RawValue{FullBytes: certificate.RawIssuer},
		SerialNumber: certificate.SerialNumber,
	})
	if err != nil {
		t.Fatal(err)
	}
	algorithm := algorithmIdentifier{Algorithm: oidRSAESOAEP}
	if parameters != nil {
		algorithm.Parameters = asn1.RawValue{FullBytes: parameters}
	}
	recipient, err := asn1.Marshal(keyTransRecipientInfo{
		RID:                    asn1.RawValue{FullBytes: serial},
		KeyEncryptionAlgorithm: algorithm,
		EncryptedKey:           wrapped,
	})
	if err != nil {
		t.Fatal(err)
	}
	ivDER, err := asn1.Marshal(iv)
	if err != nil {
		t.Fatal(err)
	}
	enveloped, err := asn1.Marshal(envelopedData{
		RecipientInfos: []asn1.RawValue{{FullBytes: recipient}},
		EncryptedContentInfo: encryptedContentInfo{
			ContentType: oidPKCS7Data,
			ContentEncryptionAlgorithm: algorithmIdentifier{
				Algorithm: oidAES256CBC, Parameters: asn1.RawValue{FullBytes: ivDER}},
			EncryptedContent: encryptCBCPadded(block, iv, content),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tagged, err := asn1.Marshal(asn1.RawValue{
		Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: enveloped})
	if err != nil {
		t.Fatal(err)
	}
	blob, err := asn1.Marshal(contentInfo{ContentType: oidEnvelopedData, Content: asn1.RawValue{FullBytes: tagged}})
	if err != nil {
		t.Fatal(err)
	}
	return blob
}
