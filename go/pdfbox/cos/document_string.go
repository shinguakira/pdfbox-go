package cos

import "fmt"

// String is Java's Object.toString for a COSDocument, which the class does not
// override: the class name and an identity hash.
//
// StandardSecurityHandler.prepareEncryptionDictRev234 feeds it to the MD5 that
// makes a document ID where the file has none, so it has to be there; it is
// arbitrary by design, and the pointer is as arbitrary as an identity hash.
func (d *Document) String() string {
	return fmt.Sprintf("org.apache.pdfbox.cos.COSDocument@%p", d)
}
