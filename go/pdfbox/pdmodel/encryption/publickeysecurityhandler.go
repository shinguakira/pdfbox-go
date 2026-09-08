package encryption

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PublicKeySecurityHandlerFilter is the /Filter name of this handler.
//
// Port of PublicKeySecurityHandler.FILTER.
const PublicKeySecurityHandlerFilter = "Adobe.PubSec"

// PublicKeySecurityHandler is the public key security handler, which protects
// a document for one or more recipients named by their X509 certificates.
//
// Port of org.apache.pdfbox.pdmodel.encryption.PublicKeySecurityHandler.
type PublicKeySecurityHandler struct {
	securityHandlerBase

	policy *PublicKeyProtectionPolicy
}

var _ SecurityHandler = (*PublicKeySecurityHandler)(nil)

// NewPublicKeySecurityHandler creates a handler for reading an encrypted
// document.
func NewPublicKeySecurityHandler() *PublicKeySecurityHandler {
	h := &PublicKeySecurityHandler{securityHandlerBase: newSecurityHandlerBase()}
	h.self = h
	return h
}

// NewPublicKeySecurityHandlerOfPolicy creates a handler for encrypting a
// document with the given policy.
func NewPublicKeySecurityHandlerOfPolicy(
	publicKeyProtectionPolicy *PublicKeyProtectionPolicy) *PublicKeySecurityHandler {
	h := NewPublicKeySecurityHandler()
	h.policy = publicKeyProtectionPolicy
	h.hasPolicy = true
	h.keyLength = int16(publicKeyProtectionPolicy.EncryptionKeyLength())
	return h
}

// PrepareForDecryption prepares everything to decrypt the document.
func (h *PublicKeySecurityHandler) PrepareForDecryption(encryption *PDEncryption,
	documentIDArray *cos.Array, decryptionMaterial DecryptionMaterial) error {
	material, ok := decryptionMaterial.(*PublicKeyDecryptionMaterial)
	if !ok {
		return errors.New("Provided decryption material is not compatible with the document - " +
			"did you pass a null keyStore?")
	}

	defaultCryptFilterDictionary := encryption.DefaultCryptFilterDictionary()
	if defaultCryptFilterDictionary != nil && defaultCryptFilterDictionary.Length() != 0 {
		h.SetKeyLength(defaultCryptFilterDictionary.Length())
		h.setDecryptMetadata(defaultCryptFilterDictionary.IsEncryptMetaData())
	} else {
		encryptionLength := encryption.Length()
		if encryptionLength != 0 {
			h.SetKeyLength(encryptionLength)
			h.setDecryptMetadata(encryption.IsEncryptMetaData())
		}
	}

	foundRecipient := false

	certificate, err := material.Certificate()
	if err != nil {
		return err
	}
	var materialIssuer []byte
	var materialSerial *big.Int
	var materialSubjectKeyID []byte
	if certificate != nil {
		materialIssuer = certificate.RawIssuer
		materialSerial = certificate.SerialNumber
		materialSubjectKeyID = certificate.SubjectKeyId
	}

	// the decrypted content of the enveloped data that match
	// the certificate in the decryption material provided
	var envelopedData []byte

	// the bytes of each recipient in the recipients array
	array := encryption.COSObject().GetCOSArray(cos.Recipients)
	if array == nil && defaultCryptFilterDictionary != nil {
		array = defaultCryptFilterDictionary.COSObject().GetCOSArray(cos.Recipients)
	}
	if array == nil {
		return errors.New("/Recipients entry is missing in encryption dictionary")
	}

	recipientFieldsBytes := make([][]byte, array.Size())
	// TODO encryption.getRecipientsLength() and getRecipientStringAt() should be deprecated

	recipientFieldsLength := 0
	var extraInfo strings.Builder
	for i := 0; i < array.Size(); i++ {
		recipientFieldString := array.GetObject(i).(*cos.StringObj)
		recipientBytes := recipientFieldString.Bytes()
		data, err := newCMSEnvelopedData(recipientBytes)
		if err != nil {
			return err
		}
		j := 0
		for _, ri := range data.Recipients() {
			// Impl: if a matching certificate was previously found it is an
			// error, here we just don't care about it
			if !foundRecipient && ri.matches(materialIssuer, materialSerial, materialSubjectKeyID) {
				foundRecipient = true
				privateKey, err := material.PrivateKey()
				if err != nil {
					return err
				}
				envelopedData, err = ri.content(privateKey)
				if err != nil {
					return err
				}
				break
			}
			j++
			if certificate != nil {
				extraInfo.WriteByte('\n')
				fmt.Fprintf(&extraInfo, "%d", j)
				extraInfo.WriteString(": ")
				appendCertInfo(&extraInfo, ri, certificate)
			}
		}
		recipientFieldsBytes[i] = recipientBytes
		recipientFieldsLength += len(recipientBytes)
	}
	if !foundRecipient || envelopedData == nil {
		return fmt.Errorf("The certificate matches none of %d recipient entries%s",
			array.Size(), extraInfo.String())
	}

	if len(envelopedData) != 24 {
		return errors.New("The enveloped data does not contain 24 bytes")
	}

	// now envelopedData contains:
	// - the 20 bytes seed
	// - the 4 bytes of permission for the current user

	accessBytes := make([]byte, 4)
	copy(accessBytes, envelopedData[20:24])

	currentAccessPermission := NewAccessPermissionFromBytes(accessBytes)
	currentAccessPermission.SetReadOnly()
	h.SetCurrentAccessPermission(currentAccessPermission)

	// what we will put in the SHA1 = the seed + each byte contained in the
	// recipients array
	sha1Input := make([]byte, recipientFieldsLength+20)

	// put the seed in the sha1 input
	copy(sha1Input, envelopedData[:20])

	// put each bytes of the recipients array in the sha1 input
	sha1InputOffset := 20
	for _, recipientFieldsByte := range recipientFieldsBytes {
		copy(sha1Input[sha1InputOffset:], recipientFieldsByte)
		sha1InputOffset += len(recipientFieldsByte)
	}

	var mdResult []byte
	encryptionVersion := encryption.Version()
	if encryptionVersion == 4 || encryptionVersion == 5 {
		if !h.IsDecryptMetadata() {
			// "4 bytes with the value 0xFF if the key being generated is
			// intended for use in document-level encryption and the document
			// metadata is being left as plaintext"
			sha1Input = copyOf(sha1Input, len(sha1Input)+4)
			copy(sha1Input[len(sha1Input)-4:], []byte{0xff, 0xff, 0xff, 0xff})
		}
		if encryptionVersion == 4 {
			sum := sha1.Sum(sha1Input)
			mdResult = sum[:]
		} else {
			sum := sha256.Sum256(sha1Input)
			mdResult = sum[:]
		}

		// detect whether AES encryption is used. This assumes that the
		// encryption algo is stored in the PDCryptFilterDictionary
		// However, crypt filters are used only when V is 4 or 5.
		if defaultCryptFilterDictionary != nil {
			cryptFilterMethod := defaultCryptFilterDictionary.CryptFilterMethod()
			h.SetAES(cos.AESV2.Equals(cryptFilterMethod) || cos.AESV3.Equals(cryptFilterMethod))
		}
	} else {
		sum := sha1.Sum(sha1Input)
		mdResult = sum[:]
	}

	// we have the encryption key ...
	h.SetEncryptionKey(make([]byte, h.KeyLength()/8))
	copy(h.EncryptionKey(), mdResult[:h.KeyLength()/8])
	return nil
}

// appendCertInfo writes why a recipient did not match, which is what Java's
// appendCertInfo does with the KeyTransRecipientId.
func appendCertInfo(extraInfo *strings.Builder, ri *cmsRecipient, certificate *x509.Certificate) {
	if ri.serialNumber != nil {
		certSerial := "unknown"
		if certificate.SerialNumber != nil {
			certSerial = certificate.SerialNumber.Text(16)
		}
		extraInfo.WriteString("serial-#: rid ")
		extraInfo.WriteString(ri.serialNumber.Text(16))
		extraInfo.WriteString(" vs. cert ")
		extraInfo.WriteString(certSerial)
		extraInfo.WriteString(" issuer: rid '")
		extraInfo.WriteString(distinguishedNameText(ri.issuer))
		extraInfo.WriteString("' vs. cert '")
		extraInfo.WriteString(certificate.Issuer.String())
		extraInfo.WriteString("' ")
	}
}

// distinguishedNameText renders a DER encoded X.501 name the way BouncyCastle's
// X500Name.toString does, as far as Go can: it parses it back into a
// pkix.RDNSequence and prints that.
func distinguishedNameText(der []byte) string {
	var rdns pkix.RDNSequence
	if _, err := asn1.Unmarshal(der, &rdns); err != nil {
		return fmt.Sprintf("%x", der)
	}
	var name pkix.Name
	name.FillFromRDNSequence(&rdns)
	return name.String()
}

// PrepareDocumentForEncryption prepares everything to encrypt the document.
//
// Port of prepareDocumentForEncryption. Java generates a 20-byte seed, wraps it
// with each recipient's permissions into a CMS enveloped-data blob per
// recipient, and derives the encryption key by digesting the seed followed by
// every one of those blobs -- so a recipient who can unwrap any one of them
// recovers the seed and, with the blobs from the document, the same key.
func (h *PublicKeySecurityHandler) PrepareDocumentForEncryption(doc PDDocumentLike) error {
	if h.policy == nil || h.policy.NumberOfRecipients() == 0 {
		return fmt.Errorf("encryption: a public key policy needs at least one recipient")
	}

	dictionary := doc.Encryption()
	if dictionary == nil {
		dictionary = NewPDEncryption()
	}
	dictionary.SetFilter(PublicKeySecurityHandlerFilter)
	dictionary.SetLength(h.KeyLength())
	version := h.computeVersionNumber()
	dictionary.SetVersion(version)

	// Remove CF, StmF and StrF entries that may be left from a previous
	// encryption.
	dictionary.RemoveV45filters()

	// Java asks a KeyGenerator for a 192-bit AES key and takes the first 20
	// bytes of it, which is a roundabout way of asking for 20 random bytes.
	seed := make([]byte, 20)
	if _, err := rand.Read(seed); err != nil {
		return fmt.Errorf("encryption: generating the seed: %w", err)
	}

	recipientsFields, err := h.computeRecipientsField(seed)
	if err != nil {
		return err
	}

	shaInput := make([]byte, 0, len(seed))
	shaInput = append(shaInput, seed...)
	for _, field := range recipientsFields {
		shaInput = append(shaInput, field...)
	}

	var digest []byte
	switch version {
	case 4:
		dictionary.SetSubFilter(publicKeySubFilter5)
		sum := sha1.Sum(shaInput)
		digest = sum[:]
		h.prepareEncryptionDictAES(dictionary, cos.AESV2, recipientsFields)
	case 5:
		dictionary.SetSubFilter(publicKeySubFilter5)
		sum := sha256.Sum256(shaInput)
		digest = sum[:]
		h.prepareEncryptionDictAES(dictionary, cos.AESV3, recipientsFields)
	default:
		dictionary.SetSubFilter(publicKeySubFilter4)
		sum := sha1.Sum(shaInput)
		digest = sum[:]
		dictionary.SetRecipients(recipientsFields)
	}

	key := make([]byte, h.KeyLength()/8)
	copy(key, digest)
	h.SetEncryptionKey(key)

	doc.SetEncryptionDictionary(dictionary)
	doc.COSDocument().SetEncryptionDictionary(dictionary.COSObject())
	return nil
}

// The two subfilters a public key encryption dictionary carries, which say
// which of the two layouts the recipients are in.
const (
	publicKeySubFilter4 = "adbe.pkcs7.s4"
	publicKeySubFilter5 = "adbe.pkcs7.s5"
)

// computeVersionNumber is the base SecurityHandler.computeVersionNumber, which
// the port has on StandardSecurityHandler and needs here as well.
func (h *PublicKeySecurityHandler) computeVersionNumber() int {
	switch {
	case h.KeyLength() == 40:
		return 1
	case h.KeyLength() == 128 && h.policy.IsPreferAES():
		return 4
	case h.KeyLength() == 256:
		return 5
	}
	return 2
}

// computeRecipientsField builds one CMS enveloped-data blob per recipient.
//
// Port of computeRecipientsField. What is enveloped is the seed followed by
// that recipient's permission bits, big-endian, which is what ties a
// recipient's permissions to a key only they can unwrap.
func (h *PublicKeySecurityHandler) computeRecipientsField(seed []byte) ([][]byte, error) {
	recipientsField := make([][]byte, 0, h.policy.NumberOfRecipients())
	for _, recipient := range h.policy.Recipients() {
		permission := recipient.Permission().PermissionBytesForPublicKey()

		pkcs7input := make([]byte, 24)
		copy(pkcs7input, seed)
		pkcs7input[20] = byte(uint32(permission) >> 24)
		pkcs7input[21] = byte(uint32(permission) >> 16)
		pkcs7input[22] = byte(uint32(permission) >> 8)
		pkcs7input[23] = byte(permission)

		blob, err := newCMSEnvelopedDataFor(pkcs7input,
			[]*x509.Certificate{recipient.X509()})
		if err != nil {
			return nil, err
		}
		recipientsField = append(recipientsField, blob)
	}
	return recipientsField, nil
}

// prepareEncryptionDictAES writes the crypt filter an AES public key document
// carries, with the recipients on the filter rather than on the dictionary.
//
// Port of the private prepareEncryptionDictAES of this class, which is not the
// StandardSecurityHandler one of the same name: that one has no recipients to
// put anywhere.
func (h *PublicKeySecurityHandler) prepareEncryptionDictAES(dictionary *PDEncryption,
	aesVName *cos.Name, recipients [][]byte) {
	cryptFilterDictionary := NewPDCryptFilterDictionary()
	cryptFilterDictionary.SetCryptFilterMethod(aesVName)
	cryptFilterDictionary.SetLength(h.KeyLength())

	array := cos.NewArray()
	for _, recipient := range recipients {
		array.Add(cos.NewStringObjBytes(recipient))
	}
	cryptFilterDictionary.COSObject().SetItem(cos.Recipients, array)
	array.SetDirect(true)

	dictionary.SetDefaultCryptFilterDictionary(cryptFilterDictionary)
	dictionary.SetStreamFilterName(cos.DefaultCryptFilter)
	dictionary.SetStringFilterName(cos.DefaultCryptFilter)
	cryptFilterDictionary.COSObject().SetDirect(true)
	h.SetAES(true)
}
