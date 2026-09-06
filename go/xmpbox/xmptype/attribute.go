package xmptype

// An XML attribute of a property.
//
// Port of org.apache.xmpbox.type.Attribute.

import "fmt"

// Attribute is a namespace, a local name and a value.
type Attribute struct {
	nsURI string
	name  string
	value string
}

// NewAttribute returns an attribute with the given namespace, local name and
// value.
func NewAttribute(nsURI, localName, value string) *Attribute {
	return &Attribute{nsURI: nsURI, name: localName, value: value}
}

// Name returns the local name of the attribute.
func (a *Attribute) Name() string { return a.name }

// SetName sets the local name of the attribute.
func (a *Attribute) SetName(lname string) { a.name = lname }

// Namespace returns the namespace URI of the attribute.
func (a *Attribute) Namespace() string { return a.nsURI }

// SetNsURI sets the namespace URI of the attribute.
func (a *Attribute) SetNsURI(nsURI string) { a.nsURI = nsURI }

// Value returns the value of the attribute.
func (a *Attribute) Value() string { return a.value }

// SetValue sets the value of the attribute.
func (a *Attribute) SetValue(value string) { a.value = value }

// String returns the Java toString form.
func (a *Attribute) String() string {
	return fmt.Sprintf("[attr:{%s}%s=%s]", a.nsURI, a.name, a.value)
}
