package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
)

// The encryption half of PDDocument.
//
// Port of the encryption members of org.apache.pdfbox.pdmodel.PDDocument, kept
// in a file of their own so that the reading half stays readable.

// Encryption returns the encryption dictionary for this document, or nil where
// the document is not encrypted.
func (d *PDDocument) Encryption() *encryption.PDEncryption {
	if d.encryption == nil && d.IsEncrypted() {
		d.encryption = encryption.NewPDEncryptionOf(d.document.EncryptionDictionary())
	}
	return d.encryption
}

// SetEncryptionDictionary sets the encryption dictionary for this document.
func (d *PDDocument) SetEncryptionDictionary(encryptionDictionary *encryption.PDEncryption) {
	d.encryption = encryptionDictionary
}

// COSDocument returns the COS document below this one, as a security handler
// sees it.
func (d *PDDocument) COSDocument() encryption.COSDocumentLike { return d.document }

// CurrentAccessPermission returns the access permissions granted when the
// document was decrypted. If the document was not decrypted this method returns
// the access permission for a document owner.
func (d *PDDocument) CurrentAccessPermission() *encryption.AccessPermission {
	if d.accessPermission == nil {
		d.accessPermission = encryption.OwnerAccessPermission()
	}
	return d.accessPermission
}

// SetAccessPermission records what the parser's security handler granted.
//
// Java hands this to the PDDocument constructor; the port's reading
// constructor is NewPDDocumentOf, which the loader calls before it knows.
func (d *PDDocument) SetAccessPermission(accessPermission *encryption.AccessPermission) {
	d.accessPermission = accessPermission
}

// IsAllSecurityToBeRemoved tells whether the document will be saved without
// encryption.
func (d *PDDocument) IsAllSecurityToBeRemoved() bool { return d.allSecurityToBeRemoved }

// SetAllSecurityToBeRemoved sets whether the document will be saved without
// encryption.
func (d *PDDocument) SetAllSecurityToBeRemoved(removeAllSecurity bool) {
	d.allSecurityToBeRemoved = removeAllSecurity
}

// ensure the document satisfies what a security handler asks of it.
var _ encryption.PDDocumentLike = (*PDDocument)(nil)

// String is Java's Object.toString, which the revision 2 to 4 document ID
// digest feeds on.
func (d *PDDocument) String() string { return "PDDocument" }
