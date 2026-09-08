package encryption

// Building a CMS enveloped-data blob, which is what a /Recipients entry holds.
//
// cms.go reads one; this writes one, and the two are the same structures in
// opposite directions. Java gets both from BouncyCastle -- `EnvelopedData`,
// `KeyTransRecipientInfo`, `ContentInfo` -- and `PublicKeySecurityHandler.
// createDERForRecipient` assembles them by hand, which is what this follows.
//
// The recorded reason this was not ported said it "needs a CMS encoder, which
// is slice 7". Slice 7 merged, and the encoder was half here already: the RC2
// cipher and every one of these ASN.1 structures came in with the decrypting
// side. See migration/STATUS.md.

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"math/big"
)

// rc2KeyLength is the content encryption key Java generates, in bytes.
//
// `keygen.init(128)` in createDERForRecipient.
const rc2KeyLength = 16

// rc2ParameterVersion is what RFC 2268 section 6 calls the RC2 version number
// for a 128-bit effective key length, which is what rc2EffectiveKeyBits maps
// back to 128.
const rc2ParameterVersion = 58

// rc2CBCParameter is RFC 3370 section 5.3's RC2-CBC-Parameter, which is the
// shape cms.go reads back.
type rc2CBCParameter struct {
	Version int
	IV      []byte
}

// newCMSEnvelopedDataFor builds a CMS enveloped-data blob that carries the
// given content for the given recipients.
//
// Port of PublicKeySecurityHandler.createDERForRecipient. The content is
// encrypted once, with a fresh RC2 key, and that key is then encrypted to each
// recipient's certificate -- which is what "key transport" means and why a
// document with ten recipients carries the content once and ten wrapped keys.
func newCMSEnvelopedDataFor(content []byte,
	certificates []*x509.Certificate) ([]byte, error) {
	// Java asks a KeyGenerator for a 128-bit RC2 key and an
	// AlgorithmParameterGenerator for the parameters, which for RC2-CBC is the
	// initialisation vector and the version that says how long the key is.
	key := make([]byte, rc2KeyLength)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("encryption: generating an RC2 key: %w", err)
	}
	iv := make([]byte, 8)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("encryption: generating an RC2 initialisation vector: %w", err)
	}

	block, err := newRC2Cipher(key, 128)
	if err != nil {
		return nil, err
	}
	encrypted := encryptCBCPadded(block, iv, content)

	parameters, err := asn1.Marshal(rc2CBCParameter{Version: rc2ParameterVersion, IV: iv})
	if err != nil {
		return nil, fmt.Errorf("encryption: writing the RC2 parameters: %w", err)
	}

	recipients := make([]asn1.RawValue, 0, len(certificates))
	for _, certificate := range certificates {
		recipient, err := keyTransRecipientFor(certificate, key)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, asn1.RawValue{FullBytes: recipient})
	}

	enveloped, err := asn1.Marshal(envelopedData{
		Version:        0,
		RecipientInfos: recipients,
		EncryptedContentInfo: encryptedContentInfo{
			ContentType: oidPKCS7Data,
			ContentEncryptionAlgorithm: algorithmIdentifier{
				Algorithm:  oidRC2CBC,
				Parameters: asn1.RawValue{FullBytes: parameters},
			},
			EncryptedContent: encrypted,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("encryption: writing the enveloped data: %w", err)
	}

	// The content of a ContentInfo is `[0] EXPLICIT`, and the wrapper has to be
	// written by hand: `encoding/asn1` writes a RawValue's FullBytes verbatim
	// and ignores the field's own tagging parameters, so a RawValue holding the
	// enveloped data comes out untagged and nothing can read it back.
	tagged, err := asn1.Marshal(asn1.RawValue{
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
		Bytes:      enveloped,
	})
	if err != nil {
		return nil, fmt.Errorf("encryption: tagging the enveloped data: %w", err)
	}

	blob, err := asn1.Marshal(contentInfo{
		ContentType: oidEnvelopedData,
		Content:     asn1.RawValue{FullBytes: tagged},
	})
	if err != nil {
		return nil, fmt.Errorf("encryption: writing the content info: %w", err)
	}
	return blob, nil
}

// keyTransRecipientFor wraps the content encryption key for one certificate.
//
// Port of computeRecipientInfo. Java reads the issuer and serial number off the
// certificate's TBSCertificate and encrypts the key with the algorithm the
// subject public key info names, which for every certificate a PDF is
// encrypted to is RSA.
func keyTransRecipientFor(certificate *x509.Certificate, key []byte) ([]byte, error) {
	publicKey, isRSA := certificate.PublicKey.(*rsa.PublicKey)
	if !isRSA {
		return nil, fmt.Errorf(
			"encryption: the certificate of %q carries a %T public key; a PDF "+
				"recipient's key has to be RSA",
			certificate.Subject.String(), certificate.PublicKey)
	}
	wrapped, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, key)
	if err != nil {
		return nil, fmt.Errorf("encryption: wrapping the key for %q: %w",
			certificate.Subject.String(), err)
	}

	serial, err := asn1.Marshal(issuerAndSerialNumber{
		Issuer:       asn1.RawValue{FullBytes: certificate.RawIssuer},
		SerialNumber: new(big.Int).Set(certificate.SerialNumber),
	})
	if err != nil {
		return nil, fmt.Errorf("encryption: writing the recipient identifier: %w", err)
	}

	// Java takes the whole AlgorithmIdentifier off the certificate's
	// SubjectPublicKeyInfo, which for an RSA key carries an explicit ASN.1
	// NULL in its parameters. RFC 3370 section 4.2.1 says rsaEncryption's
	// parameters field must be present and NULL, so leaving it out is a
	// difference a strict reader is entitled to refuse.
	recipient, err := asn1.Marshal(keyTransRecipientInfo{
		Version: 0,
		RID:     asn1.RawValue{FullBytes: serial},
		KeyEncryptionAlgorithm: algorithmIdentifier{
			Algorithm:  oidRSAEncryption,
			Parameters: asn1.RawValue{Tag: asn1.TagNull},
		},
		EncryptedKey: wrapped,
	})
	if err != nil {
		return nil, fmt.Errorf("encryption: writing the recipient info: %w", err)
	}
	return recipient, nil
}

// encryptCBCPadded encrypts with PKCS#5 padding, which is what
// `Cipher.getInstance("RC2/CBC/PKCS5Padding")` does and what cms.go strips on
// the way back.
func encryptCBCPadded(block cipher.Block, iv, content []byte) []byte {
	size := block.BlockSize()
	padding := size - len(content)%size
	padded := make([]byte, len(content)+padding)
	copy(padded, content)
	for i := len(content); i < len(padded); i++ {
		padded[i] = byte(padding)
	}
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(padded, padded)
	return padded
}
