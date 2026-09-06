package digitalsignature

import (
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// The bits of the /Ff flags of a seed value certificate.
//
// Port of the FLAG_ constants of PDSeedValueCertificate. Java leaves 1 << 4
// out: the specification has no constraint there.
const (
	FlagSubject   = 1
	FlagIssuer    = 1 << 1
	FlagOID       = 1 << 2
	FlagSubjectDN = 1 << 3
	FlagKeyUsage  = 1 << 5
	FlagURL       = 1 << 6
)

// PDSeedValueCertificate is the /Cert dictionary of a seed value: what the
// certificate that signs has to look like.
//
// Port of PDSeedValueCertificate.
type PDSeedValueCertificate struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDSeedValueCertificate)(nil)

// NewPDSeedValueCertificate returns a new certificate seed value dictionary.
func NewPDSeedValueCertificate() *PDSeedValueCertificate {
	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.Type, cos.SVCert)
	dictionary.SetDirect(true)
	return &PDSeedValueCertificate{dictionary: dictionary}
}

// NewPDSeedValueCertificateOf returns the certificate seed value the given
// dictionary holds.
func NewPDSeedValueCertificateOf(dict *cos.Dictionary) *PDSeedValueCertificate {
	dict.SetDirect(true)
	return &PDSeedValueCertificate{dictionary: dict}
}

// COSObject returns the dictionary.
func (c *PDSeedValueCertificate) COSObject() cos.Base { return c.dictionary }

// Dictionary returns the dictionary, typed.
func (c *PDSeedValueCertificate) Dictionary() *cos.Dictionary { return c.dictionary }

// IsSubjectRequired reports whether the subject is a required constraint.
func (c *PDSeedValueCertificate) IsSubjectRequired() bool {
	return c.dictionary.GetFlag(cos.Ff, FlagSubject)
}

// SetSubjectRequired sets whether the subject is a required constraint.
func (c *PDSeedValueCertificate) SetSubjectRequired(flag bool) {
	c.dictionary.SetFlag(cos.Ff, FlagSubject, flag)
}

// IsIssuerRequired reports whether the issuer is a required constraint.
func (c *PDSeedValueCertificate) IsIssuerRequired() bool {
	return c.dictionary.GetFlag(cos.Ff, FlagIssuer)
}

// SetIssuerRequired sets whether the issuer is a required constraint.
func (c *PDSeedValueCertificate) SetIssuerRequired(flag bool) {
	c.dictionary.SetFlag(cos.Ff, FlagIssuer, flag)
}

// IsOIDRequired reports whether the object identifier is a required constraint.
func (c *PDSeedValueCertificate) IsOIDRequired() bool {
	return c.dictionary.GetFlag(cos.Ff, FlagOID)
}

// SetOIDRequired sets whether the object identifier is a required constraint.
func (c *PDSeedValueCertificate) SetOIDRequired(flag bool) {
	c.dictionary.SetFlag(cos.Ff, FlagOID, flag)
}

// IsSubjectDNRequired reports whether the subject distinguished name is a
// required constraint.
func (c *PDSeedValueCertificate) IsSubjectDNRequired() bool {
	return c.dictionary.GetFlag(cos.Ff, FlagSubjectDN)
}

// SetSubjectDNRequired sets whether the subject distinguished name is a
// required constraint.
func (c *PDSeedValueCertificate) SetSubjectDNRequired(flag bool) {
	c.dictionary.SetFlag(cos.Ff, FlagSubjectDN, flag)
}

// IsKeyUsageRequired reports whether the key usage is a required constraint.
func (c *PDSeedValueCertificate) IsKeyUsageRequired() bool {
	return c.dictionary.GetFlag(cos.Ff, FlagKeyUsage)
}

// SetKeyUsageRequired sets whether the key usage is a required constraint.
func (c *PDSeedValueCertificate) SetKeyUsageRequired(flag bool) {
	c.dictionary.SetFlag(cos.Ff, FlagKeyUsage, flag)
}

// IsURLRequired reports whether the URL is a required constraint.
func (c *PDSeedValueCertificate) IsURLRequired() bool {
	return c.dictionary.GetFlag(cos.Ff, FlagURL)
}

// SetURLRequired sets whether the URL is a required constraint.
func (c *PDSeedValueCertificate) SetURLRequired(flag bool) {
	c.dictionary.SetFlag(cos.Ff, FlagURL, flag)
}

// Subject returns the certificate subjects, or nil where there are none.
func (c *PDSeedValueCertificate) Subject() [][]byte {
	array := c.dictionary.GetCOSArray(cos.Subject)
	if array != nil {
		return listOfByteArraysFromCOSArray(array)
	}
	return nil
}

// SetSubject sets the certificate subjects.
func (c *PDSeedValueCertificate) SetSubject(subjects [][]byte) {
	c.dictionary.SetItem(cos.Subject, convertListOfByteArraysToCOSArray(subjects))
}

// AddSubject adds a certificate subject.
func (c *PDSeedValueCertificate) AddSubject(subject []byte) {
	array := c.dictionary.GetCOSArray(cos.Subject)
	if array == nil {
		array = cos.NewArray()
	}
	array.Add(cos.NewStringObjBytes(subject))
	c.dictionary.SetItem(cos.Subject, array)
}

// RemoveSubject removes a certificate subject.
func (c *PDSeedValueCertificate) RemoveSubject(subject []byte) {
	array := c.dictionary.GetCOSArray(cos.Subject)
	if array != nil {
		array.Remove(cos.NewStringObjBytes(subject))
	}
}

// SubjectDN returns the subject distinguished names, or nil where there are
// none.
//
// Java answers a List<Map<String, String>>; Go maps iterate in a random order,
// which does not matter here because the caller looks entries up by key.
func (c *PDSeedValueCertificate) SubjectDN() []map[string]string {
	cosArray := c.dictionary.GetCOSArray(cos.SubjectDN)
	if cosArray == nil {
		return nil
	}
	subjectDNList := cosArray.ToList()
	result := []map[string]string{}
	for _, subjectDNItem := range subjectDNList {
		if subjectDNItemDict, isDictionary := subjectDNItem.(*cos.Dictionary); isDictionary {
			subjectDNMap := map[string]string{}
			for _, key := range subjectDNItemDict.KeySet() {
				subjectDNMap[key.Name()] = subjectDNItemDict.GetString(key, "")
			}
			result = append(result, subjectDNMap)
		}
	}
	return result
}

// SetSubjectDN sets the subject distinguished names.
//
// The entries of each map go into its dictionary in the order the map yields
// them; Java's HashMap has no order of its own either.
func (c *PDSeedValueCertificate) SetSubjectDN(subjectDN []map[string]string) {
	subjectDNDict := make([]*cos.Dictionary, 0, len(subjectDN))
	for _, subjectDNItem := range subjectDN {
		dict := cos.NewDictionary()
		for key, value := range subjectDNItem {
			dict.SetItem(cos.GetPDFName(key), cos.NewStringObj(value))
		}
		subjectDNDict = append(subjectDNDict, dict)
	}
	c.dictionary.SetItem(cos.SubjectDN, common.NewCOSArrayOfObjectables(subjectDNDict))
}

// KeyUsage returns the key usage extensions, or nil where there are none.
func (c *PDSeedValueCertificate) KeyUsage() []string {
	array := c.dictionary.GetCOSArray(cos.KeyUsage)
	if array == nil {
		return nil
	}
	keyUsageExtensions := []string{}
	for _, item := range array.ToList() {
		if str, isString := item.(*cos.StringObj); isString {
			keyUsageExtensions = append(keyUsageExtensions, str.Value())
		}
	}
	return keyUsageExtensions
}

// SetKeyUsage sets the key usage extensions.
func (c *PDSeedValueCertificate) SetKeyUsage(keyUsageExtensions []string) {
	c.dictionary.SetItem(cos.KeyUsage, cos.ArrayOfStrings(keyUsageExtensions))
}

// AddKeyUsage adds a key usage extension.
//
// Java throws IllegalArgumentException for a character other than 0, 1 and X,
// which is unchecked, so the port panics.
func (c *PDSeedValueCertificate) AddKeyUsage(keyUsageExtension string) {
	const allowedChars = "01X"
	for _, ch := range []rune(keyUsageExtension) {
		if !strings.ContainsRune(allowedChars, ch) {
			panic("characters can only be 0, 1, X")
		}
	}
	array := c.dictionary.GetCOSArray(cos.KeyUsage)
	if array == nil {
		array = cos.NewArray()
	}
	array.Add(cos.NewStringObj(keyUsageExtension))
	c.dictionary.SetItem(cos.KeyUsage, array)
}

// AddKeyUsageOfFlags adds a key usage extension built out of its nine flags,
// each of which is 0, 1 or X.
//
// Port of addKeyUsage(char, char, char, char, char, char, char, char, char).
func (c *PDSeedValueCertificate) AddKeyUsageOfFlags(digitalSignature, nonRepudiation,
	keyEncipherment, dataEncipherment, keyAgreement, keyCertSign, cRLSign,
	encipherOnly, decipherOnly rune) {
	builder := &strings.Builder{}
	builder.WriteRune(digitalSignature)
	builder.WriteRune(nonRepudiation)
	builder.WriteRune(keyEncipherment)
	builder.WriteRune(dataEncipherment)
	builder.WriteRune(keyAgreement)
	builder.WriteRune(keyCertSign)
	builder.WriteRune(cRLSign)
	builder.WriteRune(encipherOnly)
	builder.WriteRune(decipherOnly)
	c.AddKeyUsage(builder.String())
}

// RemoveKeyUsage removes a key usage extension.
func (c *PDSeedValueCertificate) RemoveKeyUsage(keyUsageExtension string) {
	array := c.dictionary.GetCOSArray(cos.KeyUsage)
	if array != nil {
		array.Remove(cos.NewStringObj(keyUsageExtension))
	}
}

// Issuer returns the certificate issuers, or nil where there are none.
func (c *PDSeedValueCertificate) Issuer() [][]byte {
	array := c.dictionary.GetCOSArray(cos.Issuer)
	if array != nil {
		return listOfByteArraysFromCOSArray(array)
	}
	return nil
}

// SetIssuer sets the certificate issuers.
func (c *PDSeedValueCertificate) SetIssuer(issuers [][]byte) {
	c.dictionary.SetItem(cos.Issuer, convertListOfByteArraysToCOSArray(issuers))
}

// AddIssuer adds a certificate issuer.
func (c *PDSeedValueCertificate) AddIssuer(issuer []byte) {
	array := c.dictionary.GetCOSArray(cos.Issuer)
	if array == nil {
		array = cos.NewArray()
	}
	array.Add(cos.NewStringObjBytes(issuer))
	c.dictionary.SetItem(cos.Issuer, array)
}

// RemoveIssuer removes a certificate issuer.
func (c *PDSeedValueCertificate) RemoveIssuer(issuer []byte) {
	array := c.dictionary.GetCOSArray(cos.Issuer)
	if array != nil {
		array.Remove(cos.NewStringObjBytes(issuer))
	}
}

// OID returns the object identifiers of acceptable certificate policies, or nil
// where there are none.
func (c *PDSeedValueCertificate) OID() [][]byte {
	array := c.dictionary.GetCOSArray(cos.OID)
	if array != nil {
		return listOfByteArraysFromCOSArray(array)
	}
	return nil
}

// SetOID sets the object identifiers of acceptable certificate policies.
func (c *PDSeedValueCertificate) SetOID(oidByteStrings [][]byte) {
	c.dictionary.SetItem(cos.OID, convertListOfByteArraysToCOSArray(oidByteStrings))
}

// AddOID adds an object identifier of an acceptable certificate policy.
func (c *PDSeedValueCertificate) AddOID(oid []byte) {
	array := c.dictionary.GetCOSArray(cos.OID)
	if array == nil {
		array = cos.NewArray()
	}
	array.Add(cos.NewStringObjBytes(oid))
	c.dictionary.SetItem(cos.OID, array)
}

// RemoveOID removes an object identifier of an acceptable certificate policy.
func (c *PDSeedValueCertificate) RemoveOID(oid []byte) {
	array := c.dictionary.GetCOSArray(cos.OID)
	if array != nil {
		array.Remove(cos.NewStringObjBytes(oid))
	}
}

// URL returns the URL the certificate is fetched from, or the empty string
// where there is none.
func (c *PDSeedValueCertificate) URL() string { return c.dictionary.GetString(cos.URL, "") }

// SetURL sets the URL the certificate is fetched from.
func (c *PDSeedValueCertificate) SetURL(url string) { c.dictionary.SetString(cos.URL, url) }

// URLType returns what kind of URL the /URL entry is, or the empty string where
// there is none.
func (c *PDSeedValueCertificate) URLType() string {
	return c.dictionary.GetNameAsString(cos.URLType, "")
}

// SetURLType sets what kind of URL the /URL entry is.
func (c *PDSeedValueCertificate) SetURLType(urlType string) {
	c.dictionary.SetName(cos.URLType, urlType)
}

// listOfByteArraysFromCOSArray returns the bytes of every string of the array.
// Java declares it private static.
func listOfByteArraysFromCOSArray(array *cos.Array) [][]byte {
	result := [][]byte{}
	for _, item := range array.ToList() {
		if str, isString := item.(*cos.StringObj); isString {
			result = append(result, str.Bytes())
		}
	}
	return result
}

// convertListOfByteArraysToCOSArray returns an array of strings holding the
// given bytes. Java declares it private static.
func convertListOfByteArraysToCOSArray(strings [][]byte) *cos.Array {
	array := cos.NewArray()
	for _, s := range strings {
		array.Add(cos.NewStringObjBytes(s))
	}
	return array
}
