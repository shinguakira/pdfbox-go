package encryption

import "github.com/shinguakira/pdfbox-go/go/pdfbox/cos"

// PDCryptFilterDictionary is the crypt filter dictionary of an encrypted
// document.
//
// Port of org.apache.pdfbox.pdmodel.encryption.PDCryptFilterDictionary.
type PDCryptFilterDictionary struct {
	cryptFilterDictionary *cos.Dictionary
}

// NewPDCryptFilterDictionary creates a new empty crypt filter dictionary.
func NewPDCryptFilterDictionary() *PDCryptFilterDictionary {
	return &PDCryptFilterDictionary{cryptFilterDictionary: cos.NewDictionary()}
}

// NewPDCryptFilterDictionaryOf wraps the given dictionary.
func NewPDCryptFilterDictionaryOf(d *cos.Dictionary) *PDCryptFilterDictionary {
	return &PDCryptFilterDictionary{cryptFilterDictionary: d}
}

// COSObject returns the dictionary.
func (d *PDCryptFilterDictionary) COSObject() *cos.Dictionary { return d.cryptFilterDictionary }

// SetLength sets the length of the secret key.
func (d *PDCryptFilterDictionary) SetLength(length int) {
	d.cryptFilterDictionary.SetInt(cos.Length, length)
}

// Length returns the length of the secret key.
func (d *PDCryptFilterDictionary) Length() int {
	return d.cryptFilterDictionary.GetIntDefault(cos.Length, 40)
}

// SetCryptFilterMethod sets the method used by the crypt filter.
func (d *PDCryptFilterDictionary) SetCryptFilterMethod(cfm *cos.Name) {
	d.cryptFilterDictionary.SetItem(cos.CFM, cfm)
}

// CryptFilterMethod returns the method used by the crypt filter.
func (d *PDCryptFilterDictionary) CryptFilterMethod() *cos.Name {
	return d.cryptFilterDictionary.GetCOSName(cos.CFM)
}

// IsEncryptMetaData tells whether the metadata is encrypted.
func (d *PDCryptFilterDictionary) IsEncryptMetaData() bool {
	value := d.COSObject().GetDictionaryObject(cos.EncryptMetadata)
	if boolean, ok := value.(*cos.Boolean); ok {
		return boolean.Value()
	}
	// default is true (see 7.6.3.2 Standard Encryption Dictionary PDF 32000-1:2008)
	return true
}

// SetEncryptMetaData sets whether the metadata is encrypted.
func (d *PDCryptFilterDictionary) SetEncryptMetaData(encryptMetaData bool) {
	d.COSObject().SetBoolean(cos.EncryptMetadata, encryptMetaData)
}
