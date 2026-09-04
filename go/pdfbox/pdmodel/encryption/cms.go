package encryption

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
)

// The object identifiers a CMS enveloped-data blob carries.
var (
	oidEnvelopedData    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 3}
	oidRSAEncryption    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	oidDESEDE3CBC       = asn1.ObjectIdentifier{1, 2, 840, 113549, 3, 7}
	oidAES128CBC        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 2}
	oidAES192CBC        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 22}
	oidAES256CBC        = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 1, 42}
	oidRC2CBC           = asn1.ObjectIdentifier{1, 2, 840, 113549, 3, 2}
	oidPKCS12KeyBag     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 1}
	oidPKCS12ShroudedKB = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 2}
	oidPKCS12CertBag    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 3}
)

// contentInfo is RFC 5652's ContentInfo.
type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

// algorithmIdentifier is X.509's AlgorithmIdentifier.
type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

// envelopedData is RFC 5652's EnvelopedData.
type envelopedData struct {
	Version              int
	OriginatorInfo       asn1.RawValue   `asn1:"optional,tag:0"`
	RecipientInfos       []asn1.RawValue `asn1:"set"`
	EncryptedContentInfo encryptedContentInfo
	UnprotectedAttrs     asn1.RawValue `asn1:"optional,tag:1"`
}

// encryptedContentInfo is RFC 5652's EncryptedContentInfo.
type encryptedContentInfo struct {
	ContentType                asn1.ObjectIdentifier
	ContentEncryptionAlgorithm algorithmIdentifier
	EncryptedContent           []byte `asn1:"optional,tag:0"`
}

// issuerAndSerialNumber is RFC 5652's IssuerAndSerialNumber.
type issuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

// keyTransRecipientInfo is RFC 5652's KeyTransRecipientInfo.
type keyTransRecipientInfo struct {
	Version                int
	RID                    asn1.RawValue
	KeyEncryptionAlgorithm algorithmIdentifier
	EncryptedKey           []byte
}

// cmsRecipient is one recipient of an enveloped-data blob, in the shape
// PublicKeySecurityHandler asks about it.
//
// Stands for org.bouncycastle.cms.RecipientInformation, of which PDFBox uses
// the recipient identifier and the content it unwraps.
type cmsRecipient struct {
	issuer            []byte
	serialNumber      *big.Int
	subjectKeyID      []byte
	encryptedKey      []byte
	keyEncryptionAlgo asn1.ObjectIdentifier
	envelope          *cmsEnvelopedData
}

// cmsEnvelopedData is a parsed CMS enveloped-data blob.
//
// Stands for org.bouncycastle.cms.CMSEnvelopedData. Only the key transport
// recipient kind is read, which is the only kind a PDF /Recipients entry
// carries.
type cmsEnvelopedData struct {
	recipients                 []*cmsRecipient
	contentEncryptionAlgorithm algorithmIdentifier
	encryptedContent           []byte
}

// newCMSEnvelopedData parses a CMS enveloped-data blob.
func newCMSEnvelopedData(der []byte) (*cmsEnvelopedData, error) {
	var info contentInfo
	if _, err := asn1.Unmarshal(der, &info); err != nil {
		return nil, fmt.Errorf("encryption: reading the CMS content info: %w", err)
	}
	if !info.ContentType.Equal(oidEnvelopedData) {
		return nil, fmt.Errorf("encryption: CMS content type is %v, want enveloped data",
			info.ContentType)
	}
	var enveloped envelopedData
	if _, err := asn1.Unmarshal(info.Content.Bytes, &enveloped); err != nil {
		return nil, fmt.Errorf("encryption: reading the CMS enveloped data: %w", err)
	}

	data := &cmsEnvelopedData{
		contentEncryptionAlgorithm: enveloped.EncryptedContentInfo.ContentEncryptionAlgorithm,
		encryptedContent:           enveloped.EncryptedContentInfo.EncryptedContent,
	}
	for _, raw := range enveloped.RecipientInfos {
		if raw.Class != asn1.ClassUniversal || raw.Tag != asn1.TagSequence {
			// Only KeyTransRecipientInfo is a plain SEQUENCE; the other three
			// kinds are context tagged and PDFBox never writes them.
			continue
		}
		var ktri keyTransRecipientInfo
		if _, err := asn1.Unmarshal(raw.FullBytes, &ktri); err != nil {
			return nil, fmt.Errorf("encryption: reading a CMS recipient: %w", err)
		}
		recipient := &cmsRecipient{
			encryptedKey:      ktri.EncryptedKey,
			keyEncryptionAlgo: ktri.KeyEncryptionAlgorithm.Algorithm,
			envelope:          data,
		}
		if ktri.RID.Class == asn1.ClassContextSpecific && ktri.RID.Tag == 0 {
			recipient.subjectKeyID = ktri.RID.Bytes
		} else {
			var ias issuerAndSerialNumber
			if _, err := asn1.Unmarshal(ktri.RID.FullBytes, &ias); err != nil {
				return nil, fmt.Errorf("encryption: reading a CMS recipient identifier: %w", err)
			}
			recipient.issuer = ias.Issuer.FullBytes
			recipient.serialNumber = ias.SerialNumber
		}
		data.recipients = append(data.recipients, recipient)
	}
	return data, nil
}

// Recipients returns the key transport recipients of the blob.
func (d *cmsEnvelopedData) Recipients() []*cmsRecipient { return d.recipients }

// matches reports whether the recipient identifier names the given certificate,
// which is BouncyCastle's RecipientId.match.
func (r *cmsRecipient) matches(issuerDER []byte, serialNumber *big.Int,
	subjectKeyID []byte) bool {
	if r.subjectKeyID != nil {
		return subjectKeyID != nil && string(r.subjectKeyID) == string(subjectKeyID)
	}
	if r.serialNumber == nil || serialNumber == nil {
		return false
	}
	return r.serialNumber.Cmp(serialNumber) == 0 && string(r.issuer) == string(issuerDER)
}

// content unwraps the content encryption key with the given private key and
// decrypts the enveloped content, which is BouncyCastle's
// RecipientInformation.getContent(new JceKeyTransEnvelopedRecipient(key)).
func (r *cmsRecipient) content(privateKey crypto.PrivateKey) ([]byte, error) {
	if !r.keyEncryptionAlgo.Equal(oidRSAEncryption) {
		return nil, fmt.Errorf("encryption: key encryption algorithm %v is not supported",
			r.keyEncryptionAlgo)
	}
	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("encryption: the private key is not an RSA key")
	}
	contentKey, err := rsa.DecryptPKCS1v15(nil, rsaKey, r.encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("encryption: unwrapping the content key: %w", err)
	}
	return decryptCMSContent(r.envelope.contentEncryptionAlgorithm, contentKey,
		r.envelope.encryptedContent)
}

// decryptCMSContent decrypts the enveloped content with the named algorithm.
//
// The initialisation vector is read per algorithm because the algorithms do not
// agree on how to carry it: AES-CBC and DES-EDE3-CBC carry it on its own, as an
// OCTET STRING, while RC2-CBC wraps it in a SEQUENCE with the version that says
// how many key bits are effective. Reading it once, before the switch, would
// take the OCTET STRING shape for granted and refuse every RC2 envelope --
// which is the shape PDFBox itself writes.
func decryptCMSContent(algorithm algorithmIdentifier, key, encrypted []byte) ([]byte, error) {
	var iv []byte
	var block cipher.Block
	var err error
	switch {
	case algorithm.Algorithm.Equal(oidAES128CBC),
		algorithm.Algorithm.Equal(oidAES192CBC),
		algorithm.Algorithm.Equal(oidAES256CBC):
		if iv, err = cmsOctetStringIV(algorithm.Parameters); err == nil {
			block, err = aes.NewCipher(key)
		}
	case algorithm.Algorithm.Equal(oidDESEDE3CBC):
		if iv, err = cmsOctetStringIV(algorithm.Parameters); err == nil {
			block, err = des.NewTripleDESCipher(key)
		}
	case algorithm.Algorithm.Equal(oidRC2CBC):
		// RC2-CBC-Parameter of RFC 3370 section 5.3: the effective key bits
		// before the IV, both inside a SEQUENCE.
		var rc2Params struct {
			Version int
			IV      []byte
		}
		if _, err = asn1.Unmarshal(algorithm.Parameters.FullBytes, &rc2Params); err != nil {
			return nil, fmt.Errorf("encryption: reading the RC2 parameters: %w", err)
		}
		iv = rc2Params.IV
		block, err = newRC2Cipher(key, rc2EffectiveKeyBits(rc2Params.Version))
	default:
		return nil, fmt.Errorf("encryption: content encryption algorithm %v is not supported",
			algorithm.Algorithm)
	}
	if err != nil {
		return nil, err
	}
	if len(iv) != block.BlockSize() {
		return nil, fmt.Errorf("encryption: the content encryption IV is %d bytes, want %d",
			len(iv), block.BlockSize())
	}
	if len(encrypted) == 0 || len(encrypted)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("encryption: the enveloped content is %d bytes", len(encrypted))
	}
	plain := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, encrypted)
	return pkcs7Unpad(plain, block.BlockSize())
}

// pkcs7Unpad strips the padding CMS uses, which is PKCS#7 over the block size.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("encryption: nothing to unpad")
	}
	padding := int(data[len(data)-1])
	if padding < 1 || padding > blockSize || padding > len(data) {
		return nil, errors.New("encryption: the enveloped content is not properly padded")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return nil, errors.New("encryption: the enveloped content is not properly padded")
		}
	}
	return data[:len(data)-padding], nil
}

// cmsOctetStringIV reads the initialisation vector an AES-CBC or DES-EDE3-CBC
// AlgorithmIdentifier carries, which is an OCTET STRING on its own: RFC 3565
// section 4.1 and RFC 3370 section 5.2. RC2-CBC does not use this shape.
func cmsOctetStringIV(parameters asn1.RawValue) ([]byte, error) {
	if parameters.FullBytes == nil {
		return nil, nil
	}
	var iv []byte
	if _, err := asn1.Unmarshal(parameters.FullBytes, &iv); err != nil {
		return nil, fmt.Errorf("encryption: reading the content encryption IV: %w", err)
	}
	return iv, nil
}
