package encryption

import (
	"crypto"
	"crypto/x509"
	"errors"
)

// PublicKeyRecipient represents a recipient in the public key protection
// policy.
//
// A recipient is an X509 certificate and its access permissions.
//
// Port of org.apache.pdfbox.pdmodel.encryption.PublicKeyRecipient.
type PublicKeyRecipient struct {
	x509       *x509.Certificate
	permission *AccessPermission
}

// X509 returns the X509 certificate of the recipient.
func (r *PublicKeyRecipient) X509() *x509.Certificate { return r.x509 }

// SetX509 sets the X509 certificate of the recipient.
func (r *PublicKeyRecipient) SetX509(aX509 *x509.Certificate) { r.x509 = aX509 }

// Permission returns the access permissions granted to the recipient.
func (r *PublicKeyRecipient) Permission() *AccessPermission { return r.permission }

// SetPermission sets the access permissions granted to the recipient.
func (r *PublicKeyRecipient) SetPermission(permissions *AccessPermission) {
	r.permission = permissions
}

// KeyStore is the store of certificates and private keys a public key
// decryption reads from.
//
// Java takes a java.security.KeyStore, which the caller has already loaded from
// a PKCS#12 file; Go has no such type, so the port declares the four operations
// PublicKeyDecryptionMaterial uses. LoadPKCS12 returns one.
type KeyStore interface {
	// Size returns how many entries the store holds.
	Size() int

	// Aliases returns the names of the entries, in the order the store lists
	// them.
	Aliases() []string

	// Certificate returns the certificate of the named entry, or nil.
	Certificate(alias string) *x509.Certificate

	// Key returns the private key of the named entry, or nil.
	Key(alias string, password string) (crypto.PrivateKey, error)

	// ContainsAlias reports whether the store holds an entry of that name.
	ContainsAlias(alias string) bool
}

// PublicKeyDecryptionMaterial represents the necessary information to decrypt a
// PDF document protected by the public key security handler. To decrypt such a
// document, we need:
//
//   - a valid X509 certificate which correspond to one of the recipients of the
//     document
//   - the private key corresponding to this certificate
//   - the password to decrypt the private key if necessary
//
// Objects of this class can be used with the openProtection method of
// PDDocument.
//
// The following example shows how to decrypt a document using a private key
// stored in a PKCS#12 keystore:
//
//	doc, err := pdfbox.LoadPDF(path)
//	keyStore, err := encryption.LoadPKCS12(file, "keyStorePassword")
//	material := encryption.NewPublicKeyDecryptionMaterial(keyStore, "certAlias", "password")
//	err = doc.OpenProtection(material)
//
// Port of org.apache.pdfbox.pdmodel.encryption.PublicKeyDecryptionMaterial.
type PublicKeyDecryptionMaterial struct {
	password string
	keyStore KeyStore
	alias    string
}

var _ DecryptionMaterial = (*PublicKeyDecryptionMaterial)(nil)

// NewPublicKeyDecryptionMaterial creates a new decryption material for the
// given key store, alias and password.
func NewPublicKeyDecryptionMaterial(keystore KeyStore, a, pwd string) *PublicKeyDecryptionMaterial {
	return &PublicKeyDecryptionMaterial{keyStore: keystore, alias: a, password: pwd}
}

func (m *PublicKeyDecryptionMaterial) isDecryptionMaterial() {}

// Certificate returns the certificate contained in the material.
func (m *PublicKeyDecryptionMaterial) Certificate() (*x509.Certificate, error) {
	if m.keyStore.Size() == 1 {
		aliases := m.keyStore.Aliases()
		keyStoreAlias := aliases[0]
		return m.keyStore.Certificate(keyStoreAlias), nil
	}
	if m.keyStore.ContainsAlias(m.alias) {
		return m.keyStore.Certificate(m.alias), nil
	}
	return nil, errors.New("the keystore does not contain the given alias")
}

// Password returns the password given to the constructor.
func (m *PublicKeyDecryptionMaterial) Password() string { return m.password }

// PrivateKey returns the private key contained in the material.
func (m *PublicKeyDecryptionMaterial) PrivateKey() (crypto.PrivateKey, error) {
	if m.keyStore.Size() == 1 {
		aliases := m.keyStore.Aliases()
		keyStoreAlias := aliases[0]
		return m.keyStore.Key(keyStoreAlias, m.password)
	}
	if m.keyStore.ContainsAlias(m.alias) {
		return m.keyStore.Key(m.alias, m.password)
	}
	return nil, errors.New("the keystore does not contain the given alias")
}

// PublicKeyProtectionPolicy represents the protection policy to use to protect
// a document with the public key security handler.
//
// PDF documents are encrypted so that they can be decrypted by one or more
// recipients. Each recipient have its own access permission.
//
// Port of org.apache.pdfbox.pdmodel.encryption.PublicKeyProtectionPolicy.
type PublicKeyProtectionPolicy struct {
	protectionPolicyBase

	recipients            []*PublicKeyRecipient
	decryptionCertificate *x509.Certificate
}

var _ ProtectionPolicy = (*PublicKeyProtectionPolicy)(nil)

// NewPublicKeyProtectionPolicy creates a new empty public key protection
// policy.
func NewPublicKeyProtectionPolicy() *PublicKeyProtectionPolicy {
	return &PublicKeyProtectionPolicy{protectionPolicyBase: newProtectionPolicyBase()}
}

func (p *PublicKeyProtectionPolicy) policyKey() string { return "PublicKeyProtectionPolicy" }

// AddRecipient adds a new recipient to the recipients list.
func (p *PublicKeyProtectionPolicy) AddRecipient(recipient *PublicKeyRecipient) {
	p.recipients = append(p.recipients, recipient)
}

// RemoveRecipient removes a recipient from the recipients list, reporting
// whether it was there.
func (p *PublicKeyProtectionPolicy) RemoveRecipient(recipient *PublicKeyRecipient) bool {
	for i, candidate := range p.recipients {
		if candidate == recipient {
			p.recipients = append(p.recipients[:i], p.recipients[i+1:]...)
			return true
		}
	}
	return false
}

// Recipients returns the recipients of the policy, in the order they were
// added, which is what Java's iterator walks.
func (p *PublicKeyProtectionPolicy) Recipients() []*PublicKeyRecipient { return p.recipients }

// DecryptionCertificate returns the certificate used to open the document.
func (p *PublicKeyProtectionPolicy) DecryptionCertificate() *x509.Certificate {
	return p.decryptionCertificate
}

// SetDecryptionCertificate sets the certificate used to open the document.
func (p *PublicKeyProtectionPolicy) SetDecryptionCertificate(
	decryptionCertificate *x509.Certificate) {
	p.decryptionCertificate = decryptionCertificate
}

// NumberOfRecipients returns the number of recipients.
func (p *PublicKeyProtectionPolicy) NumberOfRecipients() int { return len(p.recipients) }
