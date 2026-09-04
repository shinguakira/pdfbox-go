package markedcontent

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDPropertyList is a property list, which a marked content operator names.
//
// Port of
// org.apache.pdfbox.pdmodel.documentinterchange.markedcontent.PDPropertyList.
type PDPropertyList struct {
	// Dict is the protected `dict` field, which the two optionalcontent
	// subclasses write through.
	Dict *cos.Dictionary
}

// propertyListFactories maps a /Type to the constructor that builds its
// property list.
//
// Java's PDPropertyList.create names PDOptionalContentGroup and
// PDOptionalContentMembershipDictionary, which both extend PDPropertyList, so
// the two packages import each other. Go forbids that, and the port's
// convention for a factory that names its own subclasses is a registry the
// subclass package fills from its init; see
// migration/conventions/java-to-go.md. graphics/optionalcontent registers OCG
// and OCMD, so Create dispatches exactly as Java's chain of ifs does.
var propertyListFactories = map[*cos.Name]func(dict *cos.Dictionary) PropertyList{}

// PropertyList is what Create returns: a PDPropertyList or one of the
// optionalcontent types that extend it.
type PropertyList interface {
	COSObject() cos.Base
	// PropertyDictionary returns the dictionary, which is getCOSObject
	// narrowed the way Java declares it.
	PropertyDictionary() *cos.Dictionary
}

// RegisterPropertyList records the constructor for a /Type. It is called from
// the init of the package that defines the type.
func RegisterPropertyList(typeName *cos.Name, factory func(dict *cos.Dictionary) PropertyList) {
	propertyListFactories[typeName] = factory
}

// CreatePropertyList returns the property list the given dictionary holds.
//
// Port of the static PDPropertyList.create(COSDictionary). It is not called
// Create because PDMarkedContent.create is already that name in this package:
// Java tells the two apart by the class they hang off, and Go has only the
// package.
func CreatePropertyList(dict *cos.Dictionary) PropertyList {
	item := dict.GetItem(cos.Type)
	if name, ok := item.(*cos.Name); ok {
		if factory := propertyListFactories[name]; factory != nil {
			return factory(dict)
		}
	}
	// todo: more types
	return NewPDPropertyListOf(dict)
}

// NewPDPropertyList creates a property list over a fresh dictionary.
//
// Java's constructor is protected; Go has no such level, and the two
// optionalcontent types need it from another package.
func NewPDPropertyList() *PDPropertyList {
	return &PDPropertyList{Dict: cos.NewDictionary()}
}

// NewPDPropertyListOf creates a property list over the given dictionary.
func NewPDPropertyListOf(dict *cos.Dictionary) *PDPropertyList {
	return &PDPropertyList{Dict: dict}
}

// COSObject returns the dictionary.
func (p *PDPropertyList) COSObject() cos.Base { return p.Dict }

// PropertyDictionary returns the dictionary, typed.
func (p *PDPropertyList) PropertyDictionary() *cos.Dictionary { return p.Dict }
