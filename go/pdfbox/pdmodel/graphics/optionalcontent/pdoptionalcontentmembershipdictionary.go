package optionalcontent

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
)

// PDOptionalContentMembershipDictionary is an optional content membership
// dictionary, which makes visibility depend on several groups at once.
//
// Port of PDOptionalContentMembershipDictionary, which extends PDPropertyList.
type PDOptionalContentMembershipDictionary struct {
	markedcontent.PDPropertyList
}

var _ markedcontent.PropertyList = (*PDOptionalContentMembershipDictionary)(nil)

// NewPDOptionalContentMembershipDictionary creates an empty membership
// dictionary.
func NewPDOptionalContentMembershipDictionary() *PDOptionalContentMembershipDictionary {
	d := &PDOptionalContentMembershipDictionary{
		PDPropertyList: *markedcontent.NewPDPropertyList(),
	}
	d.Dict.SetItem(cos.Type, cos.OCMD)
	return d
}

// NewPDOptionalContentMembershipDictionaryOf creates a membership dictionary
// over the given dictionary.
//
// Java throws IllegalArgumentException where the /Type is not /OCMD, which is
// unchecked, so the port panics.
func NewPDOptionalContentMembershipDictionaryOf(dict *cos.Dictionary) *PDOptionalContentMembershipDictionary {
	if dict.GetDictionaryObject(cos.Type) != cos.Base(cos.OCMD) {
		panic(fmt.Sprintf("Provided dictionary is not of type '%v'", cos.OCMD))
	}
	return &PDOptionalContentMembershipDictionary{
		PDPropertyList: *markedcontent.NewPDPropertyListOf(dict),
	}
}

// OCGs returns the groups this dictionary depends on.
func (d *PDOptionalContentMembershipDictionary) OCGs() []markedcontent.PropertyList {
	base := d.Dict.GetDictionaryObject(cos.OCGs)
	switch value := base.(type) {
	case *cos.Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		return []markedcontent.PropertyList{markedcontent.CreatePropertyList(&value.Dictionary)}
	case *cos.Dictionary:
		return []markedcontent.PropertyList{markedcontent.CreatePropertyList(value)}
	case *cos.Array:
		var list []markedcontent.PropertyList
		for i := 0; i < value.Size(); i++ {
			elem := value.GetObject(i)
			if dictionary, ok := asDictionary(elem); ok {
				list = append(list, markedcontent.CreatePropertyList(dictionary))
			}
		}
		return list
	}
	return nil
}

// asDictionary is Java's `instanceof COSDictionary`, which a COSStream also
// satisfies.
func asDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// SetOCGs sets the groups this dictionary depends on.
func (d *PDOptionalContentMembershipDictionary) SetOCGs(ocgs []markedcontent.PropertyList) {
	ar := cos.NewArray()
	for _, prop := range ocgs {
		ar.Add(prop.COSObject())
	}
	d.Dict.SetItem(cos.OCGs, ar)
}

// VisibilityPolicy returns the /P entry, which defaults to /AnyOn.
func (d *PDOptionalContentMembershipDictionary) VisibilityPolicy() *cos.Name {
	return d.Dict.GetCOSNameDefault(cos.P, cos.AnyOn)
}

// SetVisibilityPolicy sets the /P entry.
func (d *PDOptionalContentMembershipDictionary) SetVisibilityPolicy(visibilityPolicy *cos.Name) {
	d.Dict.SetItem(cos.P, visibilityPolicy)
}

// The /VE support Java marks with a TODO is not here either.
