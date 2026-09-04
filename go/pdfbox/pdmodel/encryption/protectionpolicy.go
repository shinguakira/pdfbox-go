package encryption

import "fmt"

// defaultProtectionKeyLength is the key length a policy starts with.
const defaultProtectionKeyLength int16 = 40

// ProtectionPolicy represents the protection policy to apply to a document.
//
// Objects implementing this interface can be passed to the protect method of
// PDDocument to protect a document.
//
// Port of the abstract class
// org.apache.pdfbox.pdmodel.encryption.ProtectionPolicy, whose two fields the
// concrete policies get by embedding protectionPolicyBase.
type ProtectionPolicy interface {
	// EncryptionKeyLength returns the encryption key length, in bits.
	EncryptionKeyLength() int

	// SetEncryptionKeyLength sets the length in bits of the secret key that
	// will be used to encrypt document data. The default value is 40, which
	// gives a low security level but is compatible with old versions of Acrobat
	// Reader.
	SetEncryptionKeyLength(l int) error

	// IsPreferAES reports whether AES is preferred where the key length allows
	// a choice.
	IsPreferAES() bool

	// SetPreferAES sets whether AES is preferred, which only has an effect for
	// a key length of 128 bits.
	SetPreferAES(preferAES bool)

	// policyKey names the policy type, which is what Java's factory keys its
	// registry on with policy.getClass().
	policyKey() string
}

// protectionPolicyBase holds the two fields ProtectionPolicy declares.
type protectionPolicyBase struct {
	encryptionKeyLength int16
	preferAES           bool
}

func newProtectionPolicyBase() protectionPolicyBase {
	return protectionPolicyBase{encryptionKeyLength: defaultProtectionKeyLength}
}

// SetEncryptionKeyLength sets the length in bits of the secret key.
//
// Java throws IllegalArgumentException for anything but 40, 128 and 256; the
// port returns the error, since the caller is asking for something a document
// cannot carry rather than making a programming mistake it cannot see.
func (p *protectionPolicyBase) SetEncryptionKeyLength(l int) error {
	if l != 40 && l != 128 && l != 256 {
		return fmt.Errorf("Invalid key length '%d' value must be 40, 128 or 256!", l)
	}
	p.encryptionKeyLength = int16(l)
	return nil
}

// EncryptionKeyLength returns the encryption key length, in bits.
func (p *protectionPolicyBase) EncryptionKeyLength() int { return int(p.encryptionKeyLength) }

// IsPreferAES reports whether AES is preferred.
func (p *protectionPolicyBase) IsPreferAES() bool { return p.preferAES }

// SetPreferAES sets whether AES is preferred, which only has an effect for a
// key length of 128 bits.
func (p *protectionPolicyBase) SetPreferAES(preferAES bool) { p.preferAES = preferAES }

// StandardProtectionPolicy represents the protection policy to use to protect a
// document with the standard security handler.
//
// PDF supports the following combination of passwords:
//
//   - <b>no password</b>: In that case, the document can be opened by anyone
//     and every operation is allowed on the document
//   - <b>only a user password</b>: In that case, the document can be opened
//     only if the user password is provided, and every operation is allowed on
//     the document
//   - <b>only an owner password</b>: In that case, the document can be opened
//     by anyone, but the operations are limited to those set in the access
//     permissions
//   - <b>both a user and an owner password</b>: In that case, the document can
//     be opened with either password, and the operations are limited to those
//     set in the access permissions where the user password was given
//
// Port of org.apache.pdfbox.pdmodel.encryption.StandardProtectionPolicy.
type StandardProtectionPolicy struct {
	protectionPolicyBase

	permissions   *AccessPermission
	ownerPassword string
	userPassword  string
}

var _ ProtectionPolicy = (*StandardProtectionPolicy)(nil)

// NewStandardProtectionPolicy creates a policy with the given passwords and
// permissions.
func NewStandardProtectionPolicy(ownerPassword, userPassword string,
	permissions *AccessPermission) *StandardProtectionPolicy {
	return &StandardProtectionPolicy{
		protectionPolicyBase: newProtectionPolicyBase(),
		ownerPassword:        ownerPassword,
		userPassword:         userPassword,
		permissions:          permissions,
	}
}

func (p *StandardProtectionPolicy) policyKey() string { return "StandardProtectionPolicy" }

// Permissions returns the access permissions granted when the document is
// decrypted with the user password.
func (p *StandardProtectionPolicy) Permissions() *AccessPermission { return p.permissions }

// SetPermissions sets the access permissions granted when the document is
// decrypted with the user password.
func (p *StandardProtectionPolicy) SetPermissions(permissions *AccessPermission) {
	p.permissions = permissions
}

// OwnerPassword returns the owner password.
func (p *StandardProtectionPolicy) OwnerPassword() string { return p.ownerPassword }

// SetOwnerPassword sets the owner password.
func (p *StandardProtectionPolicy) SetOwnerPassword(ownerPassword string) {
	p.ownerPassword = ownerPassword
}

// UserPassword returns the user password.
func (p *StandardProtectionPolicy) UserPassword() string { return p.userPassword }

// SetUserPassword sets the user password.
func (p *StandardProtectionPolicy) SetUserPassword(userPassword string) {
	p.userPassword = userPassword
}
