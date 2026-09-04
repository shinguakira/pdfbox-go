package encryption

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// The values the /V entry of an encryption dictionary takes.
const (
	// Version0UndocumentedUnsupported is an undocumented and unsupported
	// algorithm.
	Version0UndocumentedUnsupported = 0
	// Version140BitAlgorithm is 40-bit RC4.
	Version140BitAlgorithm = 1
	// Version2VariableLengthAlgorithm is RC4 with a variable key length.
	Version2VariableLengthAlgorithm = 2
	// Version3UnpublishedAlgorithm is an unpublished algorithm.
	Version3UnpublishedAlgorithm = 3
	// Version4SecurityHandler names a crypt filter in the security handler.
	Version4SecurityHandler = 4

	// DefaultName is the name of the standard security handler.
	DefaultName = "Standard"
	// DefaultLength is the key length an encryption dictionary implies.
	DefaultLength = 40
	// DefaultVersion is the /V an encryption dictionary implies.
	DefaultVersion = Version0UndocumentedUnsupported
)

// PDEncryption is the document encryption dictionary.
//
// Port of org.apache.pdfbox.pdmodel.encryption.PDEncryption.
type PDEncryption struct {
	dictionary      *cos.Dictionary
	securityHandler SecurityHandler
}

// NewPDEncryption creates a new empty encryption dictionary.
func NewPDEncryption() *PDEncryption {
	return &PDEncryption{dictionary: cos.NewDictionary()}
}

// NewPDEncryptionOf wraps the given encryption dictionary, and picks the
// security handler its /Filter names.
func NewPDEncryptionOf(dictionary *cos.Dictionary) *PDEncryption {
	e := &PDEncryption{dictionary: dictionary}
	e.securityHandler = SecurityHandlerFactoryInstance.NewSecurityHandlerForFilter(e.Filter())
	return e
}

// SecurityHandler returns the security handler of the document, reporting an
// error where the /Filter names one this library does not have.
func (e *PDEncryption) SecurityHandler() (SecurityHandler, error) {
	if e.securityHandler == nil {
		// Don't change this text, it's used by Apache Tika (TIKA-4082)
		return nil, fmt.Errorf("No security handler for filter %s", e.Filter())
	}
	return e.securityHandler, nil
}

// SetSecurityHandler sets the security handler of the document.
func (e *PDEncryption) SetSecurityHandler(securityHandler SecurityHandler) {
	e.securityHandler = securityHandler
	// TODO set Filter (currently this is done by the security handlers)
}

// HasSecurityHandler reports whether the document has a security handler.
//
// JAVA-BUGS entry 24: Java returns `securityHandler == null`, so the method
// answers the opposite of its name. Ported as written.
func (e *PDEncryption) HasSecurityHandler() bool { return e.securityHandler == nil }

// COSObject returns the dictionary.
func (e *PDEncryption) COSObject() *cos.Dictionary { return e.dictionary }

// SetFilter sets the filter entry of the encryption dictionary.
func (e *PDEncryption) SetFilter(filter string) {
	e.dictionary.SetItem(cos.Filter, cos.GetPDFName(filter))
}

// Filter returns the filter entry of the encryption dictionary.
func (e *PDEncryption) Filter() string {
	return e.dictionary.GetNameAsString(cos.Filter, "")
}

// SubFilter returns the subfilter entry of the encryption dictionary.
func (e *PDEncryption) SubFilter() string {
	return e.dictionary.GetNameAsString(cos.SubFilter, "")
}

// SetSubFilter sets the subfilter entry of the encryption dictionary.
func (e *PDEncryption) SetSubFilter(subfilter string) {
	e.dictionary.SetName(cos.SubFilter, subfilter)
}

// SetVersion sets the V entry of the encryption dictionary.
func (e *PDEncryption) SetVersion(version int) { e.dictionary.SetInt(cos.V, version) }

// Version returns the V entry of the encryption dictionary.
func (e *PDEncryption) Version() int { return e.dictionary.GetIntDefault(cos.V, 0) }

// SetLength sets the number of bits to use for the encryption algorithm.
func (e *PDEncryption) SetLength(length int) { e.dictionary.SetInt(cos.Length, length) }

// Length returns the number of bits to use for the encryption algorithm.
func (e *PDEncryption) Length() int { return e.dictionary.GetIntDefault(cos.Length, 40) }

// SetRevision sets the R entry of the encryption dictionary.
func (e *PDEncryption) SetRevision(revision int) { e.dictionary.SetInt(cos.R, revision) }

// Revision returns the R entry of the encryption dictionary.
func (e *PDEncryption) Revision() int {
	return e.dictionary.GetIntDefault(cos.R, DefaultVersion)
}

// SetOwnerKey sets the O entry in the standard encryption dictionary.
func (e *PDEncryption) SetOwnerKey(o []byte) {
	e.dictionary.SetItem(cos.O, cos.NewStringObjBytes(o))
}

// OwnerKey returns the O entry of the standard encryption dictionary, padded or
// truncated to the length the revision calls for, or nil where there is none.
func (e *PDEncryption) OwnerKey() []byte {
	var o []byte
	if owner, ok := e.dictionary.GetDictionaryObject(cos.O).(*cos.StringObj); ok {
		o = owner.Bytes()
		r := e.Revision()
		if r <= 4 {
			o = copyOf(o, 32)
		} else if r == 5 || r == 6 {
			o = copyOf(o, 48)
		}
	}
	return o
}

// SetUserKey sets the U entry in the standard encryption dictionary.
func (e *PDEncryption) SetUserKey(u []byte) {
	e.dictionary.SetItem(cos.U, cos.NewStringObjBytes(u))
}

// UserKey returns the U entry of the standard encryption dictionary, padded or
// truncated to the length the revision calls for, or nil where there is none.
func (e *PDEncryption) UserKey() []byte {
	var u []byte
	if user, ok := e.dictionary.GetDictionaryObject(cos.U).(*cos.StringObj); ok {
		u = user.Bytes()
		r := e.Revision()
		if r <= 4 {
			u = copyOf(u, 32)
		} else if r == 5 || r == 6 {
			u = copyOf(u, 48)
		}
	}
	return u
}

// SetOwnerEncryptionKey sets the OE entry in the standard encryption
// dictionary.
func (e *PDEncryption) SetOwnerEncryptionKey(oe []byte) {
	e.dictionary.SetItem(cos.OE, cos.NewStringObjBytes(oe))
}

// OwnerEncryptionKey returns the OE entry of the standard encryption
// dictionary, or nil where there is none.
func (e *PDEncryption) OwnerEncryptionKey() []byte {
	var oe []byte
	if ownerEncryptionKey, ok := e.dictionary.GetDictionaryObject(cos.OE).(*cos.StringObj); ok {
		oe = copyOf(ownerEncryptionKey.Bytes(), 32)
	}
	return oe
}

// SetUserEncryptionKey sets the UE entry in the standard encryption dictionary.
func (e *PDEncryption) SetUserEncryptionKey(ue []byte) {
	e.dictionary.SetItem(cos.UE, cos.NewStringObjBytes(ue))
}

// UserEncryptionKey returns the UE entry of the standard encryption dictionary,
// or nil where there is none.
func (e *PDEncryption) UserEncryptionKey() []byte {
	var ue []byte
	if userEncryptionKey, ok := e.dictionary.GetDictionaryObject(cos.UE).(*cos.StringObj); ok {
		ue = copyOf(userEncryptionKey.Bytes(), 32)
	}
	return ue
}

// SetPermissions sets the access permissions of the document.
func (e *PDEncryption) SetPermissions(permissions int) {
	e.dictionary.SetInt(cos.P, permissions)
}

// Permissions returns the access permissions of the document.
func (e *PDEncryption) Permissions() int { return e.dictionary.GetIntDefault(cos.P, 0) }

// IsEncryptMetaData tells whether the metadata is encrypted.
func (e *PDEncryption) IsEncryptMetaData() bool {
	// default is true (see 7.6.3.2 Standard Encryption Dictionary PDF 32000-1:2008)
	return e.dictionary.GetBoolean(cos.EncryptMetadata, true)
}

// SetRecipients sets the Recipients field of the dictionary. This field
// contains an array of string.
func (e *PDEncryption) SetRecipients(recipients [][]byte) {
	array := cos.NewArray()
	for _, recipient := range recipients {
		recip := cos.NewStringObjBytes(recipient)
		array.Add(recip)
	}
	e.dictionary.SetItem(cos.Recipients, array)
	array.SetDirect(true)
}

// RecipientsLength returns the number of recipients contained in the Recipients
// field of the dictionary.
func (e *PDEncryption) RecipientsLength() int {
	// JAVA-BUGS entry 25: Java casts the item to a COSArray without checking
	// and calls size on it, so a document with no /Recipients -- every
	// password-encrypted one -- gets a NullPointerException rather than zero.
	// Ported as written; the assertion below panics where Java throws.
	array := e.dictionary.GetItem(cos.Recipients).(*cos.Array)
	return array.Size()
}

// RecipientStringAt returns the string at the given index of the Recipients
// field.
func (e *PDEncryption) RecipientStringAt(i int) *cos.StringObj {
	array := e.dictionary.GetItem(cos.Recipients).(*cos.Array)
	return array.Get(i).(*cos.StringObj)
}

// StdCryptFilterDictionary returns the standard crypt filter.
func (e *PDEncryption) StdCryptFilterDictionary() *PDCryptFilterDictionary {
	return e.CryptFilterDictionary(cos.StdCF)
}

// DefaultCryptFilterDictionary returns the default crypt filter.
func (e *PDEncryption) DefaultCryptFilterDictionary() *PDCryptFilterDictionary {
	return e.CryptFilterDictionary(cos.DefaultCryptFilter)
}

// CryptFilterDictionary returns the crypt filter with the given name.
func (e *PDEncryption) CryptFilterDictionary(cryptFilterName *cos.Name) *PDCryptFilterDictionary {
	// See CF in "Table 20 – Entries common to all encryption dictionaries"
	if cfDict := e.dictionary.GetCOSDictionary(cos.CF); cfDict != nil {
		if cryptDict := cfDict.GetCOSDictionary(cryptFilterName); cryptDict != nil {
			return NewPDCryptFilterDictionaryOf(cryptDict)
		}
	}
	return nil
}

// SetCryptFilterDictionary sets the crypt filter with the given name.
func (e *PDEncryption) SetCryptFilterDictionary(cryptFilterName *cos.Name,
	cryptFilterDictionary *PDCryptFilterDictionary) {
	cfDictionary := e.dictionary.GetCOSDictionary(cos.CF)
	if cfDictionary == nil {
		cfDictionary = cos.NewDictionary()
		e.dictionary.SetItem(cos.CF, cfDictionary)
	}
	cfDictionary.SetDirect(true) // PDFBOX-4436 direct obj needed for Adobe Reader on Android
	cfDictionary.SetItem(cryptFilterName, cryptFilterDictionary.COSObject())
}

// SetStdCryptFilterDictionary sets the standard crypt filter.
func (e *PDEncryption) SetStdCryptFilterDictionary(cryptFilterDictionary *PDCryptFilterDictionary) {
	cryptFilterDictionary.COSObject().SetDirect(true) // PDFBOX-4436
	e.SetCryptFilterDictionary(cos.StdCF, cryptFilterDictionary)
}

// SetDefaultCryptFilterDictionary sets the default crypt filter.
func (e *PDEncryption) SetDefaultCryptFilterDictionary(
	defaultFilterDictionary *PDCryptFilterDictionary) {
	defaultFilterDictionary.COSObject().SetDirect(true) // PDFBOX-4436
	e.SetCryptFilterDictionary(cos.DefaultCryptFilter, defaultFilterDictionary)
}

// StreamFilterName returns the name of the filter which is used for de- and
// encrypting streams.
func (e *PDEncryption) StreamFilterName() *cos.Name {
	stmF := e.dictionary.GetCOSName(cos.StmF)
	if stmF == nil {
		return cos.Identity
	}
	return stmF
}

// SetStreamFilterName sets the name of the filter which is used for de- and
// encrypting streams.
func (e *PDEncryption) SetStreamFilterName(streamFilterName *cos.Name) {
	e.dictionary.SetItem(cos.StmF, streamFilterName)
}

// StringFilterName returns the name of the filter which is used for de- and
// encrypting strings.
func (e *PDEncryption) StringFilterName() *cos.Name {
	strF := e.dictionary.GetCOSName(cos.StrF)
	if strF == nil {
		return cos.Identity
	}
	return strF
}

// SetStringFilterName sets the name of the filter which is used for de- and
// encrypting strings.
func (e *PDEncryption) SetStringFilterName(stringFilterName *cos.Name) {
	e.dictionary.SetItem(cos.StrF, stringFilterName)
}

// SetPerms sets the Perms entry in the encryption dictionary.
func (e *PDEncryption) SetPerms(perms []byte) {
	e.dictionary.SetItem(cos.Perms, cos.NewStringObjBytes(perms))
}

// Perms returns the Perms entry of the encryption dictionary, or nil where
// there is none.
func (e *PDEncryption) Perms() []byte {
	var perms []byte
	if permsCosString, ok := e.dictionary.GetDictionaryObject(cos.Perms).(*cos.StringObj); ok {
		perms = permsCosString.Bytes()
	}
	return perms
}

// RemoveV45filters removes CF, StmF, and StrF entries. This is to be called if
// V is not 4 or 5.
func (e *PDEncryption) RemoveV45filters() {
	e.dictionary.SetItem(cos.CF, nil)
	e.dictionary.SetItem(cos.StmF, nil)
	e.dictionary.SetItem(cos.StrF, nil)
}

// copyOf is java.util.Arrays.copyOf: the first newLength bytes, zero-padded
// where the source is shorter.
func copyOf(original []byte, newLength int) []byte {
	copied := make([]byte, newLength)
	copy(copied, original)
	return copied
}
