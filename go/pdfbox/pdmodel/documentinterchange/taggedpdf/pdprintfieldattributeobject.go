package taggedpdf

import (
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// OwnerPrintField is the /O owner of a print field attribute object.
const OwnerPrintField = "PrintField"

// The attributes of a print field attribute object. Java declares them private.
const (
	role    = "Role"
	checked = "checked"
	desc    = "Desc"
)

// The roles a print field can have.
const (
	RoleRB = "rb"
	RoleCB = "cb"
	RolePB = "pb"
	RoleTV = "tv"
)

// The checked states a print field can have.
const (
	CheckedStateOn      = "on"
	CheckedStateOff     = "off"
	CheckedStateNeutral = "neutral"
)

func init() {
	logicalstructure.RegisterAttributeObject(OwnerPrintField,
		func(dictionary *cos.Dictionary) logicalstructure.PDAttributeObject {
			return NewPDPrintFieldAttributeObjectOf(dictionary)
		})
}

// PDPrintFieldAttributeObject is the print field attribute object of a
// structure element.
//
// Port of PDPrintFieldAttributeObject.
type PDPrintFieldAttributeObject struct {
	PDStandardAttributeObject
}

var _ logicalstructure.PDAttributeObject = (*PDPrintFieldAttributeObject)(nil)

// NewPDPrintFieldAttributeObject builds an empty print field attribute object.
func NewPDPrintFieldAttributeObject() *PDPrintFieldAttributeObject {
	o := &PDPrintFieldAttributeObject{}
	o.InitStandardAttributeObject(o)
	o.SetOwner(OwnerPrintField)
	return o
}

// NewPDPrintFieldAttributeObjectOf builds one over the given dictionary.
func NewPDPrintFieldAttributeObjectOf(dictionary *cos.Dictionary) *PDPrintFieldAttributeObject {
	o := &PDPrintFieldAttributeObject{}
	o.InitStandardAttributeObjectOf(o, dictionary)
	return o
}

// Role returns the /Role of the field.
func (o *PDPrintFieldAttributeObject) Role() string {
	return o.GetName(role)
}

// SetRole sets the /Role of the field.
func (o *PDPrintFieldAttributeObject) SetRole(value string) {
	o.SetName(role, value)
}

// CheckedState returns the /checked state, which defaults to off.
func (o *PDPrintFieldAttributeObject) CheckedState() string {
	return o.GetNameDefault(checked, CheckedStateOff)
}

// SetCheckedState sets the /checked state.
func (o *PDPrintFieldAttributeObject) SetCheckedState(checkedState string) {
	o.SetName(checked, checkedState)
}

// AlternateName returns the /Desc alternate name of the field.
func (o *PDPrintFieldAttributeObject) AlternateName() string {
	return o.GetString(desc)
}

// SetAlternateName sets the /Desc alternate name of the field.
func (o *PDPrintFieldAttributeObject) SetAlternateName(alternateName string) {
	o.SetString(desc, alternateName)
}

// String renders the attribute object the way Java's toString does.
func (o *PDPrintFieldAttributeObject) String() string {
	sb := strings.Builder{}
	sb.WriteString(o.PDAttributeObjectBase.String())
	if o.IsSpecified(role) {
		sb.WriteString(", Role=")
		sb.WriteString(o.Role())
	}
	if o.IsSpecified(checked) {
		sb.WriteString(", Checked=")
		sb.WriteString(o.CheckedState())
	}
	if o.IsSpecified(desc) {
		sb.WriteString(", Desc=")
		sb.WriteString(o.AlternateName())
	}
	return sb.String()
}
