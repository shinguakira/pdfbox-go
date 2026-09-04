package encryption

// InvalidPasswordError is what a document that cannot be decrypted with the
// password given is reported with.
//
// Port of org.apache.pdfbox.pdmodel.encryption.InvalidPasswordException, which
// extends IOException so that a caller who does not care can treat it as one.
type InvalidPasswordError struct {
	message string
}

func newInvalidPasswordError(message string) *InvalidPasswordError {
	return &InvalidPasswordError{message: message}
}

// Error returns the message.
func (e *InvalidPasswordError) Error() string { return e.message }

// DecryptionMaterial is the material needed to decrypt a document: a password
// for the standard handler, a certificate and a private key for the public key
// one.
//
// Port of the abstract class
// org.apache.pdfbox.pdmodel.encryption.DecryptionMaterial, which declares
// nothing at all; it exists so that the two kinds share a type.
type DecryptionMaterial interface {
	// isDecryptionMaterial is what Java's abstract class does by existing: it
	// keeps anything else from being passed where decryption material belongs.
	isDecryptionMaterial()
}

// StandardDecryptionMaterial represents the necessary information to decrypt a
// document protected by the standard security handler (password protection).
//
// This is only composed of a password.
//
// The following example shows how to decrypt a document protected with the
// standard security handler:
//
//	doc, err := pdfbox.LoadPDF(path)
//	material := encryption.NewStandardDecryptionMaterial("password")
//	err = doc.OpenProtection(material)
//
// Port of org.apache.pdfbox.pdmodel.encryption.StandardDecryptionMaterial.
type StandardDecryptionMaterial struct {
	password string
}

var _ DecryptionMaterial = (*StandardDecryptionMaterial)(nil)

// NewStandardDecryptionMaterial returns the material for the given password.
func NewStandardDecryptionMaterial(pwd string) *StandardDecryptionMaterial {
	return &StandardDecryptionMaterial{password: pwd}
}

func (m *StandardDecryptionMaterial) isDecryptionMaterial() {}

// Password returns the password passed to the constructor.
func (m *StandardDecryptionMaterial) Password() string { return m.password }
