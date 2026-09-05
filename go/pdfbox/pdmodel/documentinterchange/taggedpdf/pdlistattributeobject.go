package taggedpdf

import (
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// OwnerList is the /O owner of a list attribute object.
const OwnerList = "List"

// listNumbering is the /ListNumbering attribute. Java declares it protected,
// because PDExportFormatAttributeObject reads it too.
const listNumbering = "ListNumbering"

// The list numberings.
const (
	ListNumberingCircle     = "Circle"
	ListNumberingDecimal    = "Decimal"
	ListNumberingDisc       = "Disc"
	ListNumberingLowerAlpha = "LowerAlpha"
	ListNumberingLowerRoman = "LowerRoman"
	ListNumberingNone       = "None"
	ListNumberingSquare     = "Square"
	ListNumberingUpperAlpha = "UpperAlpha"
	ListNumberingUpperRoman = "UpperRoman"
)

func init() {
	logicalstructure.RegisterAttributeObject(OwnerList,
		func(dictionary *cos.Dictionary) logicalstructure.PDAttributeObject {
			return NewPDListAttributeObjectOf(dictionary)
		})
}

// PDListAttributeObject is the list attribute object of a structure element.
//
// Port of PDListAttributeObject.
type PDListAttributeObject struct {
	PDStandardAttributeObject
}

var _ logicalstructure.PDAttributeObject = (*PDListAttributeObject)(nil)

// NewPDListAttributeObject builds an empty list attribute object.
func NewPDListAttributeObject() *PDListAttributeObject {
	o := &PDListAttributeObject{}
	o.InitStandardAttributeObject(o)
	o.SetOwner(OwnerList)
	return o
}

// NewPDListAttributeObjectOf builds one over the given dictionary.
func NewPDListAttributeObjectOf(dictionary *cos.Dictionary) *PDListAttributeObject {
	o := &PDListAttributeObject{}
	o.InitStandardAttributeObjectOf(o, dictionary)
	return o
}

// ListNumbering returns the /ListNumbering, which defaults to none.
func (o *PDListAttributeObject) ListNumbering() string {
	return o.GetNameDefault(listNumbering, ListNumberingNone)
}

// SetListNumbering sets the /ListNumbering.
func (o *PDListAttributeObject) SetListNumbering(numbering string) {
	o.SetName(listNumbering, numbering)
}

// String renders the attribute object the way Java's toString does.
func (o *PDListAttributeObject) String() string {
	sb := strings.Builder{}
	sb.WriteString(o.PDAttributeObjectBase.String())
	if o.IsSpecified(listNumbering) {
		sb.WriteString(", ListNumbering=")
		sb.WriteString(o.ListNumbering())
	}
	return sb.String()
}
