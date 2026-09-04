// Package encryption opens an encrypted PDF.
//
// Port of org.apache.pdfbox.pdmodel.encryption.
package encryption

// The bits of the permission integer, numbered as the PDF specification numbers
// them: from 1, not from 0.
const (
	defaultPermissions         int32 = ^3 // bits 0 & 1 need to be zero
	printBit                         = 3
	modificationBit                  = 4
	extractBit                       = 5
	modifyAnnotationsBit             = 6
	fillInFormBit                    = 9
	extractForAccessibilityBit       = 10
	assembleDocumentBit              = 11
	faithfulPrintBit                 = 12
)

// AccessPermission represents the access permissions to a document. These
// permissions are specified in the PDF format specifications, they include:
//
//   - print the document
//   - modify the content of the document
//   - copy or extract content of the document
//   - add or modify annotations
//   - fill in interactive form fields
//   - extract text and graphics for accessibility to visually impaired people
//   - assemble the document
//   - print in degraded quality
//
// This class can be used to protect a document by assigning access permissions
// to recipients. In this case, it must be used with a specific ProtectionPolicy.
//
// When a document is decrypted, it has a currentAccessPermission property which
// is the access permissions granted to the user who decrypted the document.
//
// Port of org.apache.pdfbox.pdmodel.encryption.AccessPermission. The permission
// word is an int32 here because Java's is an int: bit 32 is the sign bit, and
// the shifts that reach it have to wrap the way Java's do.
type AccessPermission struct {
	bytes    int32
	readOnly bool
}

// NewAccessPermission creates a new access permission object. By default, all
// permissions are granted.
func NewAccessPermission() *AccessPermission {
	return &AccessPermission{bytes: defaultPermissions}
}

// NewAccessPermissionFromBytes creates a new access permission object from a
// byte array. Bytes are ordered most significant byte first.
func NewAccessPermissionFromBytes(b []byte) *AccessPermission {
	var bytes int32
	bytes |= int32(b[0]) & 0xFF
	bytes <<= 8
	bytes |= int32(b[1]) & 0xFF
	bytes <<= 8
	bytes |= int32(b[2]) & 0xFF
	bytes <<= 8
	bytes |= int32(b[3]) & 0xFF
	return &AccessPermission{bytes: bytes}
}

// NewAccessPermissionOf creates a new access permission object from a single
// integer.
func NewAccessPermissionOf(permissions int32) *AccessPermission {
	return &AccessPermission{bytes: permissions}
}

func (p *AccessPermission) isPermissionBitOn(bit int) bool {
	return p.bytes&(int32(1)<<uint(bit-1)) != 0
}

func (p *AccessPermission) setPermissionBit(bit int, value bool) bool {
	permissions := p.bytes
	if value {
		permissions |= int32(1) << uint(bit-1)
	} else {
		permissions &= ^(int32(1) << uint(bit-1))
	}
	p.bytes = permissions

	return p.bytes&(int32(1)<<uint(bit-1)) != 0
}

// IsOwnerPermission tells whether the access permission corresponds to owner
// access permission (no restriction).
func (p *AccessPermission) IsOwnerPermission() bool {
	return p.CanAssembleDocument() &&
		p.CanExtractContent() &&
		p.CanExtractForAccessibility() &&
		p.CanFillInForm() &&
		p.CanModify() &&
		p.CanModifyAnnotations() &&
		p.CanPrint() &&
		p.CanPrintFaithful()
}

// OwnerAccessPermission returns an access permission object for a document
// owner.
func OwnerAccessPermission() *AccessPermission {
	ret := NewAccessPermission()
	ret.SetCanAssembleDocument(true)
	ret.SetCanExtractContent(true)
	ret.SetCanExtractForAccessibility(true)
	ret.SetCanFillInForm(true)
	ret.SetCanModify(true)
	ret.SetCanModifyAnnotations(true)
	ret.SetCanPrint(true)
	ret.SetCanPrintFaithful(true)
	return ret
}

// PermissionBytesForPublicKey returns an integer representing the access
// permissions. This integer can be used for public key encryption. This format
// is not documented in the PDF specifications but is necessary for
// compatibility with Adobe Acrobat and Adobe Reader.
func (p *AccessPermission) PermissionBytesForPublicKey() int32 {
	p.setPermissionBit(1, true)
	p.setPermissionBit(7, false)
	p.setPermissionBit(8, false)
	for i := 13; i <= 32; i++ {
		p.setPermissionBit(i, false)
	}
	return p.bytes
}

// PermissionBytes returns an integer representing the access permissions. This
// integer can be used for standard PDF encryption as specified in the PDF
// specifications.
func (p *AccessPermission) PermissionBytes() int32 { return p.bytes }

// CanPrint tells whether the user can print.
func (p *AccessPermission) CanPrint() bool { return p.isPermissionBitOn(printBit) }

// SetCanPrint sets whether the user can print.
//
// This method will have no effect if the object is in read only mode.
func (p *AccessPermission) SetCanPrint(allowPrinting bool) {
	if !p.readOnly {
		p.setPermissionBit(printBit, allowPrinting)
	}
}

// CanModify tells whether the user can modify contents of the document.
func (p *AccessPermission) CanModify() bool { return p.isPermissionBitOn(modificationBit) }

// SetCanModify sets whether the user can modify the document.
//
// This method will have no effect if the object is in read only mode.
func (p *AccessPermission) SetCanModify(allowModifications bool) {
	if !p.readOnly {
		p.setPermissionBit(modificationBit, allowModifications)
	}
}

// CanExtractContent tells whether the user can extract text and images from the
// PDF document.
func (p *AccessPermission) CanExtractContent() bool { return p.isPermissionBitOn(extractBit) }

// SetCanExtractContent sets whether the user can extract content from the
// document.
//
// This method will have no effect if the object is in read only mode.
func (p *AccessPermission) SetCanExtractContent(allowExtraction bool) {
	if !p.readOnly {
		p.setPermissionBit(extractBit, allowExtraction)
	}
}

// CanModifyAnnotations tells whether the user can add or modify text
// annotations and fill in interactive forms fields and, if CanModify returns
// true, create or modify interactive form fields (including signature fields).
// Note that if CanFillInForm returns true, it is still possible to fill in
// interactive forms (including signature fields) even if this method here
// returns false.
func (p *AccessPermission) CanModifyAnnotations() bool {
	return p.isPermissionBitOn(modifyAnnotationsBit)
}

// SetCanModifyAnnotations sets whether the user can add or modify text
// annotations and fill in interactive forms fields.
//
// This method will have no effect if the object is in read only mode.
func (p *AccessPermission) SetCanModifyAnnotations(allowAnnotationModification bool) {
	if !p.readOnly {
		p.setPermissionBit(modifyAnnotationsBit, allowAnnotationModification)
	}
}

// CanFillInForm tells whether the user can fill in interactive form fields
// (including signature fields) even if CanModifyAnnotations returns false.
func (p *AccessPermission) CanFillInForm() bool { return p.isPermissionBitOn(fillInFormBit) }

// SetCanFillInForm sets whether the user can fill in interactive form fields
// (including signature fields) even if CanModifyAnnotations returns false.
// Therefore, if you want to prevent a user from filling in interactive form
// fields, you need to call SetCanModifyAnnotations(false) as well.
//
// This method will have no effect if the object is in read only mode.
func (p *AccessPermission) SetCanFillInForm(allowFillingInForm bool) {
	if !p.readOnly {
		p.setPermissionBit(fillInFormBit, allowFillingInForm)
	}
}

// CanExtractForAccessibility tells whether the user can extract text and images
// from the PDF document for accessibility purposes.
func (p *AccessPermission) CanExtractForAccessibility() bool {
	return p.isPermissionBitOn(extractForAccessibilityBit)
}

// SetCanExtractForAccessibility sets whether the user can extract content from
// the document for accessibility purposes.
//
// This method will have no effect if the object is in read only mode.
func (p *AccessPermission) SetCanExtractForAccessibility(allowExtraction bool) {
	if !p.readOnly {
		p.setPermissionBit(extractForAccessibilityBit, allowExtraction)
	}
}

// CanAssembleDocument tells whether the user can insert, rotate or delete
// pages.
func (p *AccessPermission) CanAssembleDocument() bool {
	return p.isPermissionBitOn(assembleDocumentBit)
}

// SetCanAssembleDocument sets whether the user can insert, rotate or delete
// pages.
//
// This method will have no effect if the object is in read only mode.
func (p *AccessPermission) SetCanAssembleDocument(allowAssembly bool) {
	if !p.readOnly {
		p.setPermissionBit(assembleDocumentBit, allowAssembly)
	}
}

// CanPrintFaithful tells whether the user can print the document in a faithful
// format or in a degraded format (if print is enabled).
func (p *AccessPermission) CanPrintFaithful() bool { return p.isPermissionBitOn(faithfulPrintBit) }

// SetCanPrintFaithful sets whether the user can print the document in a
// faithful format or in a degraded format (if print is enabled). The PDF
// version must be 1.5 or higher.
//
// This method will have no effect if the object is in read only mode.
func (p *AccessPermission) SetCanPrintFaithful(canPrintFaithful bool) {
	if !p.readOnly {
		p.setPermissionBit(faithfulPrintBit, canPrintFaithful)
	}
}

// SetReadOnly locks the access permission read only, so that the setters have
// no effect. After that, the object cannot be unlocked. This method is used for
// the currentAccessPermission of a document to stop users changing the access
// permission.
func (p *AccessPermission) SetReadOnly() { p.readOnly = true }

// IsReadOnly tells whether the object has been set as read only.
func (p *AccessPermission) IsReadOnly() bool { return p.readOnly }

// hasAnyRevision3PermissionSet reports whether any revision 3 access permission
// is set.
func (p *AccessPermission) hasAnyRevision3PermissionSet() bool {
	if p.CanFillInForm() {
		return true
	}
	if p.CanExtractForAccessibility() {
		return true
	}
	if p.CanAssembleDocument() {
		return true
	}
	return p.CanPrintFaithful()
}
