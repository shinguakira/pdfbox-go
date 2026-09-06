package digitalsignature

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDPropBuild is the build dictionary of a signature, which says which
// signature handler was used.
//
// Port of PDPropBuild.
type PDPropBuild struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDPropBuild)(nil)

// NewPDPropBuild returns a new build dictionary.
func NewPDPropBuild() *PDPropBuild {
	dictionary := cos.NewDictionary()
	dictionary.SetDirect(true) // the specification claim to use direct objects
	return &PDPropBuild{dictionary: dictionary}
}

// NewPDPropBuildOf returns the build dictionary the given dictionary holds.
func NewPDPropBuildOf(dict *cos.Dictionary) *PDPropBuild {
	dict.SetDirect(true) // the specification claim to use direct objects
	return &PDPropBuild{dictionary: dict}
}

// COSObject returns the dictionary.
func (b *PDPropBuild) COSObject() cos.Base { return b.dictionary }

// Dictionary returns the dictionary, typed.
func (b *PDPropBuild) Dictionary() *cos.Dictionary { return b.dictionary }

// Filter returns the build data of the signature handler that was used to sign,
// or nil where there is none.
func (b *PDPropBuild) Filter() *PDPropBuildDataDict {
	var filter *PDPropBuildDataDict
	filterDic := b.dictionary.GetCOSDictionary(cos.Filter)
	if filterDic != nil {
		filter = NewPDPropBuildDataDictOf(filterDic)
	}
	return filter
}

// SetPDPropBuildFilter sets the build data of the signature handler.
func (b *PDPropBuild) SetPDPropBuildFilter(filter *PDPropBuildDataDict) {
	b.dictionary.SetItem(cos.Filter, common.COSObjectOrNil(filter))
}

// PubSec returns the build data of the public key security handler, or nil
// where there is none.
func (b *PDPropBuild) PubSec() *PDPropBuildDataDict {
	var pubSec *PDPropBuildDataDict
	pubSecDic := b.dictionary.GetCOSDictionary(cos.PubSec)
	if pubSecDic != nil {
		pubSec = NewPDPropBuildDataDictOf(pubSecDic)
	}
	return pubSec
}

// SetPDPropBuildPubSec sets the build data of the public key security handler.
func (b *PDPropBuild) SetPDPropBuildPubSec(pubSec *PDPropBuildDataDict) {
	b.dictionary.SetItem(cos.PubSec, common.COSObjectOrNil(pubSec))
}

// App returns the build data of the software that created the signature, or nil
// where there is none.
func (b *PDPropBuild) App() *PDPropBuildDataDict {
	var app *PDPropBuildDataDict
	appDic := b.dictionary.GetCOSDictionary(cos.App)
	if appDic != nil {
		app = NewPDPropBuildDataDictOf(appDic)
	}
	return app
}

// SetPDPropBuildApp sets the build data of the software that created the
// signature.
func (b *PDPropBuild) SetPDPropBuildApp(app *PDPropBuildDataDict) {
	b.dictionary.SetItem(cos.App, common.COSObjectOrNil(app))
}

// PDPropBuildDataDict is one build data dictionary of a signature.
//
// Port of PDPropBuildDataDict.
type PDPropBuildDataDict struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDPropBuildDataDict)(nil)

// NewPDPropBuildDataDict returns a new build data dictionary.
func NewPDPropBuildDataDict() *PDPropBuildDataDict {
	dictionary := cos.NewDictionary()
	// the specification claim to use direct objects
	dictionary.SetDirect(true)
	return &PDPropBuildDataDict{dictionary: dictionary}
}

// NewPDPropBuildDataDictOf returns the build data the given dictionary holds.
func NewPDPropBuildDataDictOf(dict *cos.Dictionary) *PDPropBuildDataDict {
	// the specification claim to use direct objects
	dict.SetDirect(true)
	return &PDPropBuildDataDict{dictionary: dict}
}

// COSObject returns the dictionary.
func (d *PDPropBuildDataDict) COSObject() cos.Base { return d.dictionary }

// Dictionary returns the dictionary, typed.
func (d *PDPropBuildDataDict) Dictionary() *cos.Dictionary { return d.dictionary }

// Name returns the name of the software module that was used to create the
// signature, or the empty string where there is none.
func (d *PDPropBuildDataDict) Name() string {
	return d.dictionary.GetNameAsString(cos.NameKey, "")
}

// SetName sets the name of the software module that was used to create the
// signature.
func (d *PDPropBuildDataDict) SetName(name string) { d.dictionary.SetName(cos.NameKey, name) }

// Date returns the build date of the software module, or the empty string where
// there is none. This string is normally an unformatted text string that
// represents the date and time when the software was built.
func (d *PDPropBuildDataDict) Date() string { return d.dictionary.GetString(cos.Date, "") }

// SetDate sets the build date of the software module.
func (d *PDPropBuildDataDict) SetDate(date string) { d.dictionary.SetString(cos.Date, date) }

// SetVersion sets the version of the software module that was used to create
// the signature.
func (d *PDPropBuildDataDict) SetVersion(applicationVersion string) {
	d.dictionary.SetString(cos.GetPDFName("REx"), applicationVersion)
}

// Version returns the version of the software module that was used to create
// the signature, or the empty string where there is none.
func (d *PDPropBuildDataDict) Version() string {
	return d.dictionary.GetString(cos.GetPDFName("REx"), "")
}

// Revision returns the revision of the software module.
func (d *PDPropBuildDataDict) Revision() int64 { return d.dictionary.GetLong(cos.R) }

// SetRevision sets the revision of the software module.
func (d *PDPropBuildDataDict) SetRevision(revision int64) { d.dictionary.SetLong(cos.R, revision) }

// MinimumRevision returns the minimum software revision needed to validate the
// signature.
func (d *PDPropBuildDataDict) MinimumRevision() int64 { return d.dictionary.GetLong(cos.V) }

// SetMinimumRevision sets the minimum software revision needed to validate the
// signature.
func (d *PDPropBuildDataDict) SetMinimumRevision(revision int64) {
	d.dictionary.SetLong(cos.V, revision)
}

// PreRelease reports whether the software module is a pre-release.
func (d *PDPropBuildDataDict) PreRelease() bool {
	return d.dictionary.GetBoolean(cos.PreRelease, false)
}

// SetPreRelease sets whether the software module is a pre-release.
func (d *PDPropBuildDataDict) SetPreRelease(preRelease bool) {
	d.dictionary.SetBoolean(cos.PreRelease, preRelease)
}

// OS returns the operating system the software module ran on, or the empty
// string where there is none.
func (d *PDPropBuildDataDict) OS() string {
	osArray := d.dictionary.GetCOSArray(cos.OS)
	// PDF v1.5 style
	if osArray != nil {
		return osArray.GetName(0, "")
	}
	return d.dictionary.GetString(cos.OS, "")
}

// SetOS sets the operating system the software module ran on, and removes it
// for the empty string, which is the null Java takes.
func (d *PDPropBuildDataDict) SetOS(os string) {
	if os == "" {
		d.dictionary.RemoveItem(cos.OS)
		return
	}
	osArray := d.dictionary.GetCOSArray(cos.OS)
	if osArray == nil {
		osArray = cos.NewArray()
		osArray.SetDirect(true)
		d.dictionary.SetItem(cos.OS, osArray)
	}
	osArray.AddAt(0, cos.GetPDFName(os))
}

// NonEFontNoWarn reports whether the software module warns when a font is not
// embedded. It is true by default.
func (d *PDPropBuildDataDict) NonEFontNoWarn() bool {
	return d.dictionary.GetBoolean(cos.NonEFontNoWarn, true)
}

// SetNonEFontNoWarn sets whether the software module warns when a font is not
// embedded.
func (d *PDPropBuildDataDict) SetNonEFontNoWarn(noEmbedFontWarning bool) {
	d.dictionary.SetBoolean(cos.NonEFontNoWarn, noEmbedFontWarning)
}

// TrustedMode reports whether the software module was in trusted mode when the
// signature was made.
func (d *PDPropBuildDataDict) TrustedMode() bool {
	return d.dictionary.GetBoolean(cos.TrustedMode, false)
}

// SetTrustedMode sets whether the software module was in trusted mode when the
// signature was made.
func (d *PDPropBuildDataDict) SetTrustedMode(trustedMode bool) {
	d.dictionary.SetBoolean(cos.TrustedMode, trustedMode)
}
