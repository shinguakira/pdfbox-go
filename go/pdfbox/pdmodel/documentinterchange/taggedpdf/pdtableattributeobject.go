package taggedpdf

import (
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// OwnerTable is the /O owner of a table attribute object.
const OwnerTable = "Table"

// The attributes of a table attribute object. Java declares them protected,
// because PDExportFormatAttributeObject reads them too.
const (
	rowSpan = "RowSpan"
	colSpan = "ColSpan"
	headers = "Headers"
	scope   = "Scope"
	summary = "Summary"
)

// The scopes a table header can have.
const (
	ScopeBoth   = "Both"
	ScopeColumn = "Column"
	ScopeRow    = "Row"
)

func init() {
	logicalstructure.RegisterAttributeObject(OwnerTable,
		func(dictionary *cos.Dictionary) logicalstructure.PDAttributeObject {
			return NewPDTableAttributeObjectOf(dictionary)
		})
}

// PDTableAttributeObject is the table attribute object of a structure element.
//
// Port of PDTableAttributeObject.
type PDTableAttributeObject struct {
	PDStandardAttributeObject
}

var _ logicalstructure.PDAttributeObject = (*PDTableAttributeObject)(nil)

// NewPDTableAttributeObject builds an empty table attribute object.
func NewPDTableAttributeObject() *PDTableAttributeObject {
	o := &PDTableAttributeObject{}
	o.InitStandardAttributeObject(o)
	o.SetOwner(OwnerTable)
	return o
}

// NewPDTableAttributeObjectOf builds one over the given dictionary.
func NewPDTableAttributeObjectOf(dictionary *cos.Dictionary) *PDTableAttributeObject {
	o := &PDTableAttributeObject{}
	o.InitStandardAttributeObjectOf(o, dictionary)
	return o
}

// RowSpan returns the /RowSpan, which defaults to 1.
func (o *PDTableAttributeObject) RowSpan() int {
	return o.GetInteger(rowSpan, 1)
}

// SetRowSpan sets the /RowSpan.
func (o *PDTableAttributeObject) SetRowSpan(value int) {
	o.SetInteger(rowSpan, value)
}

// ColSpan returns the /ColSpan, which defaults to 1.
func (o *PDTableAttributeObject) ColSpan() int {
	return o.GetInteger(colSpan, 1)
}

// SetColSpan sets the /ColSpan.
func (o *PDTableAttributeObject) SetColSpan(value int) {
	o.SetInteger(colSpan, value)
}

// Headers returns the /Headers of the cell, or nil.
func (o *PDTableAttributeObject) Headers() []string {
	return o.GetArrayOfString(headers)
}

// SetHeaders sets the /Headers of the cell.
func (o *PDTableAttributeObject) SetHeaders(value []string) {
	o.SetArrayOfString(headers, value)
}

// Scope returns the /Scope of the header cell.
func (o *PDTableAttributeObject) Scope() string {
	return o.GetName(scope)
}

// SetScope sets the /Scope of the header cell.
func (o *PDTableAttributeObject) SetScope(value string) {
	o.SetName(scope, value)
}

// Summary returns the /Summary of the table.
func (o *PDTableAttributeObject) Summary() string {
	return o.GetString(summary)
}

// SetSummary sets the /Summary of the table.
func (o *PDTableAttributeObject) SetSummary(value string) {
	o.SetString(summary, value)
}

// String renders the attribute object the way Java's toString does.
func (o *PDTableAttributeObject) String() string {
	sb := strings.Builder{}
	sb.WriteString(o.PDAttributeObjectBase.String())
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

// stringsAsAny widens a slice of strings for arrayToString, whose Java
// parameter is Object[].
func stringsAsAny(values []string) []any {
	widened := make([]any, len(values))
	for i, value := range values {
		widened[i] = value
	}
	return widened
}
