package color

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDOutputIntent describes the colour reproduction characteristics of a
// possible output device or production condition. Output intents provide a
// means for matching the colour characteristics of a PDF document with those of
// a target output device or production environment in which the document will
// be printed.
//
// Port of PDOutputIntent, minus the constructor that builds one from an ICC
// profile: that calls java.awt.color.ICC_Profile.getInstance for the profile's
// data and component count, and Go has no ICC engine -- the same gap PDICCBased
// records. It also takes a PDDocument, which this package cannot name, so it
// would have to be demoted to a function in pdmodel anyway. See
// migration/STATUS.md.
type PDOutputIntent struct {
	dictionary *cos.Dictionary
}

// NewPDOutputIntent returns an output intent over the given dictionary.
func NewPDOutputIntent(dictionary *cos.Dictionary) *PDOutputIntent {
	return &PDOutputIntent{dictionary: dictionary}
}

// COSObject returns the dictionary.
func (o *PDOutputIntent) COSObject() cos.Base { return o.dictionary }

// Dictionary returns the dictionary, which is getCOSObject narrowed the way
// Java declares it.
func (o *PDOutputIntent) Dictionary() *cos.Dictionary { return o.dictionary }

// DestOutputIntent returns the /DestOutputProfile stream, or nil.
func (o *PDOutputIntent) DestOutputIntent() *cos.Stream {
	return o.dictionary.GetCOSStream(cos.DestOutputProfile)
}

// Info returns the /Info entry.
func (o *PDOutputIntent) Info() string {
	return o.dictionary.GetString(cos.Info, "")
}

// SetInfo sets the /Info entry.
func (o *PDOutputIntent) SetInfo(value string) {
	o.dictionary.SetString(cos.Info, value)
}

// OutputCondition returns the /OutputCondition entry.
func (o *PDOutputIntent) OutputCondition() string {
	return o.dictionary.GetString(cos.OutputCondition, "")
}

// SetOutputCondition sets the /OutputCondition entry.
func (o *PDOutputIntent) SetOutputCondition(value string) {
	o.dictionary.SetString(cos.OutputCondition, value)
}

// OutputConditionIdentifier returns the /OutputConditionIdentifier entry.
func (o *PDOutputIntent) OutputConditionIdentifier() string {
	return o.dictionary.GetString(cos.OutputConditionIdentifier, "")
}

// SetOutputConditionIdentifier sets the /OutputConditionIdentifier entry.
func (o *PDOutputIntent) SetOutputConditionIdentifier(value string) {
	o.dictionary.SetString(cos.OutputConditionIdentifier, value)
}

// RegistryName returns the /RegistryName entry.
func (o *PDOutputIntent) RegistryName() string {
	return o.dictionary.GetString(cos.RegistryName, "")
}

// SetRegistryName sets the /RegistryName entry.
func (o *PDOutputIntent) SetRegistryName(value string) {
	o.dictionary.SetString(cos.RegistryName, value)
}
