package taggedpdf

import (
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// The /O owners of an export format attribute object, one per export format.
const (
	OwnerXML100  = "XML-1.00"
	OwnerHTML320 = "HTML-3.2"
	OwnerHTML401 = "HTML-4.01"
	OwnerOEB100  = "OEB-1.00"
	OwnerRTF105  = "RTF-1.05"
	OwnerCSS100  = "CSS-1.00"
	OwnerCSS200  = "CSS-2.00"
)

func init() {
	for _, owner := range []string{
		OwnerXML100, OwnerHTML320, OwnerHTML401, OwnerOEB100, OwnerRTF105,
		OwnerCSS100, OwnerCSS200,
	} {
		logicalstructure.RegisterAttributeObject(owner,
			func(dictionary *cos.Dictionary) logicalstructure.PDAttributeObject {
				return NewPDExportFormatAttributeObjectOf(dictionary)
			})
	}
}

// PDExportFormatAttributeObject is the export format attribute object of a
// structure element: the layout attributes, and the list and table ones an
// export needs beside them.
//
// Port of PDExportFormatAttributeObject.
type PDExportFormatAttributeObject struct {
	PDLayoutAttributeObject
}

var _ logicalstructure.PDAttributeObject = (*PDExportFormatAttributeObject)(nil)

// NewPDExportFormatAttributeObject builds an empty one for the given export
// format.
func NewPDExportFormatAttributeObject(owner string) *PDExportFormatAttributeObject {
	o := &PDExportFormatAttributeObject{}
	o.InitStandardAttributeObject(o)
	o.SetOwner(owner)
	return o
}

// NewPDExportFormatAttributeObjectOf builds one over the given dictionary.
func NewPDExportFormatAttributeObjectOf(dictionary *cos.Dictionary) *PDExportFormatAttributeObject {
	o := &PDExportFormatAttributeObject{}
	o.InitStandardAttributeObjectOf(o, dictionary)
	return o
}

// ListNumbering returns the /ListNumbering, which defaults to none.
func (o *PDExportFormatAttributeObject) ListNumbering() string {
	return o.GetNameDefault(listNumbering, ListNumberingNone)
}

// SetListNumbering sets the /ListNumbering.
func (o *PDExportFormatAttributeObject) SetListNumbering(numbering string) {
	o.SetName(listNumbering, numbering)
}

// RowSpan returns the /RowSpan, which defaults to 1.
func (o *PDExportFormatAttributeObject) RowSpan() int {
	return o.GetInteger(rowSpan, 1)
}

// SetRowSpan sets the /RowSpan.
func (o *PDExportFormatAttributeObject) SetRowSpan(value int) {
	o.SetInteger(rowSpan, value)
}

// ColSpan returns the /ColSpan, which defaults to 1.
func (o *PDExportFormatAttributeObject) ColSpan() int {
	return o.GetInteger(colSpan, 1)
}

// SetColSpan sets the /ColSpan.
func (o *PDExportFormatAttributeObject) SetColSpan(value int) {
	o.SetInteger(colSpan, value)
}

// Headers returns the /Headers of the cell, or nil.
func (o *PDExportFormatAttributeObject) Headers() []string {
	return o.GetArrayOfString(headers)
}

// SetHeaders sets the /Headers of the cell.
func (o *PDExportFormatAttributeObject) SetHeaders(value []string) {
	o.SetArrayOfString(headers, value)
}

// Scope returns the /Scope of the header cell.
func (o *PDExportFormatAttributeObject) Scope() string {
	return o.GetName(scope)
}

// SetScope sets the /Scope of the header cell.
func (o *PDExportFormatAttributeObject) SetScope(value string) {
	o.SetName(scope, value)
}

// Summary returns the /Summary of the table.
func (o *PDExportFormatAttributeObject) Summary() string {
	return o.GetString(summary)
}

// SetSummary sets the /Summary of the table.
func (o *PDExportFormatAttributeObject) SetSummary(value string) {
	o.SetString(summary, value)
}

// String renders the attribute object the way Java's toString does.
func (o *PDExportFormatAttributeObject) String() string {
	sb := strings.Builder{}
	sb.WriteString(o.PDLayoutAttributeObject.String())
	if o.IsSpecified(listNumbering) {
		sb.WriteString(", ListNumbering=")
		sb.WriteString(o.ListNumbering())
	}
	if o.IsSpecified(rowSpan) {
		sb.WriteString(", RowSpan=")
		sb.WriteString(strconv.Itoa(o.RowSpan()))
	}
	if o.IsSpecified(colSpan) {
		sb.WriteString(", ColSpan=")
		sb.WriteString(strconv.Itoa(o.ColSpan()))
	}
	if o.IsSpecified(headers) {
		sb.WriteString(", Headers=")
		sb.WriteString(logicalstructure.ArrayToString(stringsAsAny(o.Headers())))
	}
	if o.IsSpecified(scope) {
		sb.WriteString(", Scope=")
		sb.WriteString(o.Scope())
	}
	if o.IsSpecified(summary) {
		sb.WriteString(", Summary=")
		sb.WriteString(o.Summary())
	}
	return sb.String()
}
