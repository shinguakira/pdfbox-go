package digitalsignature

import (
	"fmt"
	"slices"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// allowedDigestNames is the set of digests setDigestMethod accepts. Java holds
// it in a private static list.
var allowedDigestNames = []string{
	cos.SHA1.Name(),
	cos.SHA256.Name(),
	cos.SHA384.Name(),
	cos.SHA512.Name(),
	cos.RIPEMD160.Name(),
}

// The bits of the /Ff flags of a seed value.
//
// Port of the FLAG_ constants of PDSeedValue.
const (
	FlagFilter           = 1
	FlagSubFilter        = 1 << 1
	FlagV                = 1 << 2
	FlagReason           = 1 << 3
	FlagLegalAttestation = 1 << 4
	FlagAddRevInfo       = 1 << 5
	FlagDigestMethod     = 1 << 6
)

// PDSeedValue is the seed value dictionary of a signature field: what the
// signature made in that field has to look like.
//
// Port of PDSeedValue.
type PDSeedValue struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDSeedValue)(nil)

// NewPDSeedValue returns a new seed value dictionary.
func NewPDSeedValue() *PDSeedValue {
	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.Type, cos.SV)
	dictionary.SetDirect(true) // the specification claim to use direct objects
	return &PDSeedValue{dictionary: dictionary}
}

// NewPDSeedValueOf returns the seed value the given dictionary holds.
func NewPDSeedValueOf(dict *cos.Dictionary) *PDSeedValue {
	dict.SetDirect(true) // the specification claim to use direct objects
	return &PDSeedValue{dictionary: dict}
}

// COSObject returns the dictionary.
func (v *PDSeedValue) COSObject() cos.Base { return v.dictionary }

// Dictionary returns the dictionary, typed.
func (v *PDSeedValue) Dictionary() *cos.Dictionary { return v.dictionary }

// IsFilterRequired reports whether the filter is a required constraint.
func (v *PDSeedValue) IsFilterRequired() bool { return v.dictionary.GetFlag(cos.Ff, FlagFilter) }

// SetFilterRequired sets whether the filter is a required constraint.
func (v *PDSeedValue) SetFilterRequired(flag bool) {
	v.dictionary.SetFlag(cos.Ff, FlagFilter, flag)
}

// IsSubFilterRequired reports whether the subfilter is a required constraint.
func (v *PDSeedValue) IsSubFilterRequired() bool {
	return v.dictionary.GetFlag(cos.Ff, FlagSubFilter)
}

// SetSubFilterRequired sets whether the subfilter is a required constraint.
func (v *PDSeedValue) SetSubFilterRequired(flag bool) {
	v.dictionary.SetFlag(cos.Ff, FlagSubFilter, flag)
}

// IsDigestMethodRequired reports whether the digest method is a required
// constraint.
func (v *PDSeedValue) IsDigestMethodRequired() bool {
	return v.dictionary.GetFlag(cos.Ff, FlagDigestMethod)
}

// SetDigestMethodRequired sets whether the digest method is a required
// constraint.
func (v *PDSeedValue) SetDigestMethodRequired(flag bool) {
	v.dictionary.SetFlag(cos.Ff, FlagDigestMethod, flag)
}

// IsVRequired reports whether the minimum required capability is a required
// constraint.
func (v *PDSeedValue) IsVRequired() bool { return v.dictionary.GetFlag(cos.Ff, FlagV) }

// SetVRequired sets whether the minimum required capability is a required
// constraint.
func (v *PDSeedValue) SetVRequired(flag bool) { v.dictionary.SetFlag(cos.Ff, FlagV, flag) }

// IsReasonRequired reports whether the reason is a required constraint.
func (v *PDSeedValue) IsReasonRequired() bool { return v.dictionary.GetFlag(cos.Ff, FlagReason) }

// SetReasonRequired sets whether the reason is a required constraint.
func (v *PDSeedValue) SetReasonRequired(flag bool) {
	v.dictionary.SetFlag(cos.Ff, FlagReason, flag)
}

// IsLegalAttestationRequired reports whether the legal attestation is a
// required constraint.
func (v *PDSeedValue) IsLegalAttestationRequired() bool {
	return v.dictionary.GetFlag(cos.Ff, FlagLegalAttestation)
}

// SetLegalAttestationRequired sets whether the legal attestation is a required
// constraint.
func (v *PDSeedValue) SetLegalAttestationRequired(flag bool) {
	v.dictionary.SetFlag(cos.Ff, FlagLegalAttestation, flag)
}

// IsAddRevInfoRequired reports whether the revocation information is a required
// constraint.
func (v *PDSeedValue) IsAddRevInfoRequired() bool {
	return v.dictionary.GetFlag(cos.Ff, FlagAddRevInfo)
}

// SetAddRevInfoRequired sets whether the revocation information is a required
// constraint.
func (v *PDSeedValue) SetAddRevInfoRequired(flag bool) {
	v.dictionary.SetFlag(cos.Ff, FlagAddRevInfo, flag)
}

// Filter returns the filter that shall be used by the signature handler, or the
// empty string where there is none.
func (v *PDSeedValue) Filter() string { return v.dictionary.GetNameAsString(cos.Filter, "") }

// SetFilter sets the filter that shall be used by the signature handler.
func (v *PDSeedValue) SetFilter(filter *cos.Name) { v.dictionary.SetItem(cos.Filter, filter) }

// SubFilter returns the subfilters that may be used by the signature handler,
// which is empty where there are none.
func (v *PDSeedValue) SubFilter() []string {
	fields := v.dictionary.GetCOSArray(cos.SubFilter)
	if fields != nil {
		return fields.ToNameStringList()
	}
	return []string{}
}

// SetSubFilter sets the subfilters that may be used by the signature handler.
func (v *PDSeedValue) SetSubFilter(subfilter []string) {
	v.dictionary.SetItem(cos.SubFilter, cos.ArrayOfNames(subfilter))
}

// DigestMethod returns the digest methods that may be used, which is empty
// where there are none.
func (v *PDSeedValue) DigestMethod() []string {
	fields := v.dictionary.GetCOSArray(cos.DigestMethod)
	if fields != nil {
		return fields.ToNameStringList()
	}
	return []string{}
}

// SetDigestMethod sets the digest methods that may be used.
//
// Java throws IllegalArgumentException for a digest that is not one of SHA1,
// SHA256, SHA384, SHA512 and RIPEMD160, which is unchecked, so the port panics.
func (v *PDSeedValue) SetDigestMethod(digestMethod []string) {
	// integrity check
	for _, digestName := range digestMethod {
		if !slices.Contains(allowedDigestNames, digestName) {
			panic(fmt.Sprintf("Specified digest %s isn't allowed.", digestName))
		}
	}
	v.dictionary.SetItem(cos.DigestMethod, cos.ArrayOfNames(digestMethod))
}

// V returns the minimum required capability of the signature field seed value
// dictionary parser.
func (v *PDSeedValue) V() float32 { return v.dictionary.GetFloat(cos.V, -1) }

// SetV sets the minimum required capability of the signature field seed value
// dictionary parser.
func (v *PDSeedValue) SetV(minimumRequiredCapability float32) {
	v.dictionary.SetFloat(cos.V, minimumRequiredCapability)
}

// Reasons returns the reasons that may be used for the signing, which is empty
// where there are none.
//
// SetReasons writes strings and this reads names, so it panics on anything
// SetReasons wrote and on any conforming file. That is the Java, which throws
// ClassCastException there; see migration/JAVA-BUGS.md.
func (v *PDSeedValue) Reasons() []string {
	fields := v.dictionary.GetCOSArray(cos.Reasons)
	if fields != nil {
		return fields.ToNameStringList()
	}
	return []string{}
}

// SetReasons sets the reasons that may be used for the signing.
func (v *PDSeedValue) SetReasons(reasons []string) {
	v.dictionary.SetItem(cos.Reasons, cos.ArrayOfStrings(reasons))
}

// MDP returns the /MDP dictionary, or nil where there is none.
func (v *PDSeedValue) MDP() *PDSeedValueMDP {
	dict := v.dictionary.GetCOSDictionary(cos.MDP)
	if dict != nil {
		return NewPDSeedValueMDPOf(dict)
	}
	return nil
}

// SetMPD sets the /MDP dictionary, and does nothing for a nil one.
//
// Java spells the name setMPD.
func (v *PDSeedValue) SetMPD(mdp *PDSeedValueMDP) {
	if mdp != nil {
		v.dictionary.SetItem(cos.MDP, mdp.COSObject())
	}
}

// SeedValueCertificate returns the /Cert dictionary, or nil where there is
// none.
func (v *PDSeedValue) SeedValueCertificate() *PDSeedValueCertificate {
	certificate := v.dictionary.GetCOSDictionary(cos.Cert)
	if certificate != nil {
		return NewPDSeedValueCertificateOf(certificate)
	}
	return nil
}

// SetSeedValueCertificate sets the /Cert dictionary.
func (v *PDSeedValue) SetSeedValueCertificate(certificate *PDSeedValueCertificate) {
	v.dictionary.SetItem(cos.Cert, common.COSObjectOrNil(certificate))
}

// TimeStamp returns the /TimeStamp dictionary, or nil where there is none.
func (v *PDSeedValue) TimeStamp() *PDSeedValueTimeStamp {
	dict := v.dictionary.GetCOSDictionary(cos.TimeStamp)
	if dict != nil {
		return NewPDSeedValueTimeStampOf(dict)
	}
	return nil
}

// SetTimeStamp sets the /TimeStamp dictionary, and does nothing for a nil one.
func (v *PDSeedValue) SetTimeStamp(timestamp *PDSeedValueTimeStamp) {
	if timestamp != nil {
		v.dictionary.SetItem(cos.TimeStamp, timestamp.COSObject())
	}
}

// LegalAttestation returns the legal attestations that may be used, which is
// empty where there are none.
//
// SetLegalAttestation writes strings and this reads names, so it panics on
// anything SetLegalAttestation wrote and on any conforming file. That is the
// Java, which throws ClassCastException there; see migration/JAVA-BUGS.md.
func (v *PDSeedValue) LegalAttestation() []string {
	fields := v.dictionary.GetCOSArray(cos.LegalAttestation)
	if fields != nil {
		return fields.ToNameStringList()
	}
	return []string{}
}

// SetLegalAttestation sets the legal attestations that may be used.
func (v *PDSeedValue) SetLegalAttestation(legalAttestation []string) {
	v.dictionary.SetItem(cos.LegalAttestation, cos.ArrayOfStrings(legalAttestation))
}

// PDSeedValueMDP is the /MDP dictionary of a seed value: what a document may
// have done to it after it is signed.
//
// Port of PDSeedValueMDP, which implements no interface.
type PDSeedValueMDP struct {
	dictionary *cos.Dictionary
}

// NewPDSeedValueMDP returns a new /MDP dictionary.
func NewPDSeedValueMDP() *PDSeedValueMDP {
	dictionary := cos.NewDictionary()
	dictionary.SetDirect(true)
	return &PDSeedValueMDP{dictionary: dictionary}
}

// NewPDSeedValueMDPOf returns the /MDP dictionary the given dictionary holds.
func NewPDSeedValueMDPOf(dict *cos.Dictionary) *PDSeedValueMDP {
	dict.SetDirect(true)
	return &PDSeedValueMDP{dictionary: dict}
}

// COSObject returns the dictionary.
func (m *PDSeedValueMDP) COSObject() cos.Base { return m.dictionary }

// Dictionary returns the dictionary, typed.
func (m *PDSeedValueMDP) Dictionary() *cos.Dictionary { return m.dictionary }

// P returns the /P entry: which changes are allowed after signing.
func (m *PDSeedValueMDP) P() int { return m.dictionary.GetInt(cos.P) }

// SetP sets the /P entry.
//
// Java throws IllegalArgumentException outside 0 to 3, which is unchecked, so
// the port panics.
func (m *PDSeedValueMDP) SetP(p int) {
	if p < 0 || p > 3 {
		panic("Only values between 0 and 3 nare allowed.")
	}
	m.dictionary.SetInt(cos.P, p)
}

// PDSeedValueTimeStamp is the /TimeStamp dictionary of a seed value: the time
// stamp authority the signature is to be stamped by.
//
// Port of PDSeedValueTimeStamp, which implements no interface.
type PDSeedValueTimeStamp struct {
	dictionary *cos.Dictionary
}

// NewPDSeedValueTimeStamp returns a new /TimeStamp dictionary.
func NewPDSeedValueTimeStamp() *PDSeedValueTimeStamp {
	dictionary := cos.NewDictionary()
	dictionary.SetDirect(true)
	return &PDSeedValueTimeStamp{dictionary: dictionary}
}

// NewPDSeedValueTimeStampOf returns the /TimeStamp dictionary the given
// dictionary holds.
func NewPDSeedValueTimeStampOf(dict *cos.Dictionary) *PDSeedValueTimeStamp {
	dict.SetDirect(true)
	return &PDSeedValueTimeStamp{dictionary: dict}
}

// COSObject returns the dictionary.
func (t *PDSeedValueTimeStamp) COSObject() cos.Base { return t.dictionary }

// Dictionary returns the dictionary, typed.
func (t *PDSeedValueTimeStamp) Dictionary() *cos.Dictionary { return t.dictionary }

// URL returns the URL of the time stamp server, or the empty string where there
// is none.
func (t *PDSeedValueTimeStamp) URL() string { return t.dictionary.GetString(cos.URL, "") }

// SetURL sets the URL of the time stamp server.
func (t *PDSeedValueTimeStamp) SetURL(url string) { t.dictionary.SetString(cos.URL, url) }

// IsTimestampRequired reports whether the signature has to be time stamped.
func (t *PDSeedValueTimeStamp) IsTimestampRequired() bool {
	return t.dictionary.GetIntDefault(cos.Ff, 0) != 0
}

// SetTimestampRequired sets whether the signature has to be time stamped.
func (t *PDSeedValueTimeStamp) SetTimestampRequired(flag bool) {
	value := 0
	if flag {
		value = 1
	}
	t.dictionary.SetInt(cos.Ff, value)
}
