package logicalstructure

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDAttributeObject is what an attribute object of a structure element is.
//
// Java's PDAttributeObject is an abstract class; the port splits it into this
// interface for the contract and the embedded struct below for the state.
type PDAttributeObject interface {
	common.COSObjectable

	// Dictionary returns the dictionary, which PDDictionaryWrapper holds.
	Dictionary() *cos.Dictionary

	// Owner returns the /O owner of this attribute object.
	Owner() string

	// SetStructureElement sets the structure element this attribute object
	// belongs to. Java declares it protected.
	SetStructureElement(structureElement *PDStructureElement)

	// IsEmpty reports whether this attribute object holds nothing but its
	// owner.
	IsEmpty() bool
}

// attributeObjectFactories holds the constructor of each owner an attribute
// object can have.
//
// Java's PDAttributeObject.create switches on the owner and names the
// taggedpdf subclasses, which extend it; in Go that import cannot run both
// ways, so taggedpdf registers its own from its init instead.
var attributeObjectFactories = map[string]func(dictionary *cos.Dictionary) PDAttributeObject{}

// RegisterAttributeObject records the constructor of one owner. The package
// that defines the subclass calls it from its init.
func RegisterAttributeObject(owner string, factory func(dictionary *cos.Dictionary) PDAttributeObject) {
	attributeObjectFactories[owner] = factory
}

// CreateAttributeObject builds the attribute object the given dictionary
// describes, which is a default one where the owner is unknown.
//
// Port of the static PDAttributeObject.create. The port cannot call it Create,
// because PDStructureNode.create already owns that name in this package; Java
// tells the two apart by the class they hang off.
func CreateAttributeObject(dictionary *cos.Dictionary) PDAttributeObject {
	owner := dictionary.GetNameAsString(cos.O, "")
	if owner != "" {
		if factory := attributeObjectFactories[owner]; factory != nil {
			return factory(dictionary)
		}
	}
	return NewPDDefaultAttributeObjectOf(dictionary)
}

// PDAttributeObjectBase carries the state and the concrete methods of an
// attribute object.
//
// Port of the non-abstract half of PDAttributeObject.
type PDAttributeObjectBase struct {
	common.PDDictionaryWrapper
	self             PDAttributeObject
	structureElement *PDStructureElement
}

// InitAttributeObject is the protected PDAttributeObject() constructor. A
// concrete attribute object calls it from its own constructor with itself as
// self, since Go embedding does not dispatch.
func (a *PDAttributeObjectBase) InitAttributeObject(self PDAttributeObject) {
	a.self = self
	a.PDDictionaryWrapper = *common.NewPDDictionaryWrapper()
}

// InitAttributeObjectOf is the protected PDAttributeObject(COSDictionary)
// constructor.
func (a *PDAttributeObjectBase) InitAttributeObjectOf(self PDAttributeObject, dictionary *cos.Dictionary) {
	a.self = self
	a.PDDictionaryWrapper = *common.NewPDDictionaryWrapperOf(dictionary)
}

// StructureElement returns the structure element this attribute object belongs
// to. Java declares it private.
func (a *PDAttributeObjectBase) StructureElement() *PDStructureElement {
	return a.structureElement
}

// SetStructureElement sets the structure element this attribute object belongs
// to. Java declares it protected.
func (a *PDAttributeObjectBase) SetStructureElement(structureElement *PDStructureElement) {
	a.structureElement = structureElement
}

// Owner returns the /O owner of this attribute object.
func (a *PDAttributeObjectBase) Owner() string {
	return a.Dictionary().GetNameAsString(cos.O, "")
}

// SetOwner sets the /O owner of this attribute object. Java declares it
// protected.
func (a *PDAttributeObjectBase) SetOwner(owner string) {
	a.Dictionary().SetName(cos.O, owner)
}

// IsEmpty reports whether this attribute object holds nothing but its owner.
func (a *PDAttributeObjectBase) IsEmpty() bool {
	// only entry is the owner?
	return a.Dictionary().Size() == 1 && a.Owner() != ""
}

// PotentiallyNotifyChanged tells the structure element when a value really
// changed. Java declares it protected.
func (a *PDAttributeObjectBase) PotentiallyNotifyChanged(oldBase, newBase cos.Base) {
	if isValueChanged(oldBase, newBase) {
		a.NotifyChanged()
	}
}

// isValueChanged reports whether the two values differ, the way Java's equals
// does.
func isValueChanged(oldValue, newValue cos.Base) bool {
	if oldValue == nil {
		return newValue != nil
	}
	return !cos.Equal(oldValue, newValue)
}

// NotifyChanged tells the structure element that this attribute object changed.
// Java declares it protected.
func (a *PDAttributeObjectBase) NotifyChanged() {
	if a.StructureElement() != nil {
		a.StructureElement().AttributeChanged(a.self)
	}
}

// String renders the attribute object the way Java's toString does.
func (a *PDAttributeObjectBase) String() string {
	return "O=" + a.Owner()
}

// ArrayToString renders an array the way Java's protected static
// arrayToString(Object[]) does.
func ArrayToString(array []any) string {
	parts := make([]string, len(array))
	for i, o := range array {
		parts[i] = fmt.Sprintf("%v", o)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ArrayToStringFloats renders an array the way Java's protected static
// arrayToString(float[]) does, which is Float.toString of each entry.
func ArrayToStringFloats(array []float32) string {
	parts := make([]string, len(array))
	for i, f := range array {
		parts[i] = strconv.FormatFloat(float64(f), 'g', -1, 32)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// PDDefaultAttributeObject is an attribute object whose owner PDFBox has no
// class for, and which therefore reads and writes raw entries.
//
// Port of PDDefaultAttributeObject.
type PDDefaultAttributeObject struct {
	PDAttributeObjectBase
}

var _ PDAttributeObject = (*PDDefaultAttributeObject)(nil)

// NewPDDefaultAttributeObject builds an empty default attribute object.
func NewPDDefaultAttributeObject() *PDDefaultAttributeObject {
	o := &PDDefaultAttributeObject{}
	o.InitAttributeObject(o)
	return o
}

// NewPDDefaultAttributeObjectOf builds one over the given dictionary.
func NewPDDefaultAttributeObjectOf(dictionary *cos.Dictionary) *PDDefaultAttributeObject {
	o := &PDDefaultAttributeObject{}
	o.InitAttributeObjectOf(o, dictionary)
	return o
}

// AttributeNames returns the names of the entries other than the owner, in the
// order the dictionary holds them.
func (o *PDDefaultAttributeObject) AttributeNames() []string {
	attrNames := []string{}
	for _, key := range o.Dictionary().KeySet() {
		if key == cos.O {
			continue
		}
		attrNames = append(attrNames, key.Name())
	}
	return attrNames
}

// AttributeValue returns the value of one entry, or nil.
func (o *PDDefaultAttributeObject) AttributeValue(attrName string) cos.Base {
	return o.Dictionary().GetDictionaryObject(cos.GetPDFName(attrName))
}

// AttributeValueDefault returns the value of one entry, or defaultValue. Java
// declares it protected.
func (o *PDDefaultAttributeObject) AttributeValueDefault(attrName string, defaultValue cos.Base) cos.Base {
	value := o.Dictionary().GetDictionaryObject(cos.GetPDFName(attrName))
	if value == nil {
		return defaultValue
	}
	return value
}

// SetAttribute sets one entry, and tells the structure element when the value
// really changed.
func (o *PDDefaultAttributeObject) SetAttribute(attrName string, attrValue cos.Base) {
	old := o.AttributeValue(attrName)
	o.Dictionary().SetItem(cos.GetPDFName(attrName), attrValue)
	o.PotentiallyNotifyChanged(old, attrValue)
}

// String renders the attribute object the way Java's toString does.
func (o *PDDefaultAttributeObject) String() string {
	sb := strings.Builder{}
	sb.WriteString(o.PDAttributeObjectBase.String())
	sb.WriteString(", attributes={")
	for i, name := range o.AttributeNames() {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(name)
		sb.WriteString("=")
		sb.WriteString(fmt.Sprintf("%v", o.AttributeValue(name)))
	}
	sb.WriteString("}")
	return sb.String()
}

// OwnerUserProperties is the /O owner of a user attribute object.
//
// Port of PDUserAttributeObject.OWNER_USER_PROPERTIES.
const OwnerUserProperties = "UserProperties"

func init() {
	RegisterAttributeObject(OwnerUserProperties, func(dictionary *cos.Dictionary) PDAttributeObject {
		return NewPDUserAttributeObjectOf(dictionary)
	})
}

// PDUserAttributeObject is an attribute object holding user properties.
//
// Port of PDUserAttributeObject.
type PDUserAttributeObject struct {
	PDAttributeObjectBase
}

var _ PDAttributeObject = (*PDUserAttributeObject)(nil)

// NewPDUserAttributeObject builds an empty user attribute object.
func NewPDUserAttributeObject() *PDUserAttributeObject {
	o := &PDUserAttributeObject{}
	o.InitAttributeObject(o)
	o.SetOwner(OwnerUserProperties)
	return o
}

// NewPDUserAttributeObjectOf builds one over the given dictionary.
func NewPDUserAttributeObjectOf(dictionary *cos.Dictionary) *PDUserAttributeObject {
	o := &PDUserAttributeObject{}
	o.InitAttributeObjectOf(o, dictionary)
	return o
}

// OwnerUserProperties returns the /P user properties.
//
// JAVA BUG: it reads /P without checking for it, so a user attribute object
// with no /P throws a NullPointerException instead of answering an empty list.
// See migration/JAVA-BUGS.md entry 38. The port keeps it: the nil array panics.
func (o *PDUserAttributeObject) OwnerUserProperties() []*PDUserProperty {
	p := o.Dictionary().GetCOSArray(cos.P)
	properties := make([]*PDUserProperty, 0, p.Size())
	for i := 0; i < p.Size(); i++ {
		dictionary, _ := asDictionary(p.GetObject(i))
		properties = append(properties, NewPDUserPropertyOf(dictionary, o))
	}
	return properties
}

// SetUserProperties sets the /P user properties.
func (o *PDUserAttributeObject) SetUserProperties(userProperties []*PDUserProperty) {
	p := cos.NewArray()
	for _, userProperty := range userProperties {
		p.Add(userProperty.COSObject())
	}
	o.Dictionary().SetItem(cos.P, p)
}

// AddUserProperty appends one user property.
//
// JAVA BUG: like OwnerUserProperties, it reads /P without checking for it.
// See migration/JAVA-BUGS.md entry 38.
func (o *PDUserAttributeObject) AddUserProperty(userProperty *PDUserProperty) {
	p := o.Dictionary().GetCOSArray(cos.P)
	p.Add(userProperty.COSObject())
	o.NotifyChanged()
}

// RemoveUserProperty removes one user property.
//
// JAVA BUG: like OwnerUserProperties, it reads /P without checking for it.
// See migration/JAVA-BUGS.md entry 38.
func (o *PDUserAttributeObject) RemoveUserProperty(userProperty *PDUserProperty) {
	if userProperty == nil {
		return
	}
	p := o.Dictionary().GetCOSArray(cos.P)
	if p.Remove(userProperty.COSObject()) {
		o.NotifyChanged()
	}
}

// UserPropertyChanged is called when one of the user properties changed. Java's
// body is empty.
func (o *PDUserAttributeObject) UserPropertyChanged(userProperty *PDUserProperty) {
}

// String renders the attribute object the way Java's toString does.
func (o *PDUserAttributeObject) String() string {
	properties := o.OwnerUserProperties()
	parts := make([]string, len(properties))
	for i, property := range properties {
		parts[i] = property.String()
	}
	return o.PDAttributeObjectBase.String() + ", userProperties=[" + strings.Join(parts, ", ") + "]"
}

// PDUserProperty is one user property of a user attribute object.
//
// Port of PDUserProperty.
type PDUserProperty struct {
	common.PDDictionaryWrapper
	userAttributeObject *PDUserAttributeObject
}

var _ common.COSObjectable = (*PDUserProperty)(nil)

// NewPDUserProperty builds an empty user property of the given attribute
// object.
func NewPDUserProperty(userAttributeObject *PDUserAttributeObject) *PDUserProperty {
	return &PDUserProperty{
		PDDictionaryWrapper: *common.NewPDDictionaryWrapper(),
		userAttributeObject: userAttributeObject,
	}
}

// NewPDUserPropertyOf builds one over the given dictionary.
func NewPDUserPropertyOf(dictionary *cos.Dictionary,
	userAttributeObject *PDUserAttributeObject) *PDUserProperty {
	return &PDUserProperty{
		PDDictionaryWrapper: *common.NewPDDictionaryWrapperOf(dictionary),
		userAttributeObject: userAttributeObject,
	}
}

// Name returns the /N name of this property.
func (p *PDUserProperty) Name() string {
	return p.Dictionary().GetNameAsString(cos.N, "")
}

// SetName sets the /N name of this property.
func (p *PDUserProperty) SetName(name string) {
	p.potentiallyNotifyChanged(p.Name(), name)
	p.Dictionary().SetName(cos.N, name)
}

// Value returns the /V value of this property.
func (p *PDUserProperty) Value() cos.Base {
	return p.Dictionary().GetDictionaryObject(cos.V)
}

// SetValue sets the /V value of this property.
func (p *PDUserProperty) SetValue(value cos.Base) {
	p.potentiallyNotifyChanged(p.Value(), value)
	p.Dictionary().SetItem(cos.V, value)
}

// FormattedValue returns the /F formatted value of this property.
func (p *PDUserProperty) FormattedValue() string {
	return p.Dictionary().GetString(cos.F, "")
}

// SetFormattedValue sets the /F formatted value of this property.
func (p *PDUserProperty) SetFormattedValue(formattedValue string) {
	p.potentiallyNotifyChanged(p.FormattedValue(), formattedValue)
	p.Dictionary().SetString(cos.F, formattedValue)
}

// IsHidden reports the /H flag of this property.
func (p *PDUserProperty) IsHidden() bool {
	return p.Dictionary().GetBoolean(cos.H, false)
}

// SetHidden sets the /H flag of this property.
func (p *PDUserProperty) SetHidden(hidden bool) {
	p.potentiallyNotifyChanged(p.IsHidden(), hidden)
	p.Dictionary().SetBoolean(cos.H, hidden)
}

// String renders the property the way Java's toString does.
func (p *PDUserProperty) String() string {
	return "Name=" + p.Name() +
		", Value=" + fmt.Sprintf("%v", p.Value()) +
		", FormattedValue=" + p.FormattedValue() +
		", Hidden=" + strconv.FormatBool(p.IsHidden())
}

// potentiallyNotifyChanged tells the attribute object when an entry really
// changed.
//
// Java compares with equals over Object, which for a COSBase value is the
// COSBase equals and for the others is String or Boolean equality; the port
// compares the same way through the empty interface, where a COS value falls to
// pointer equality unless its type defines more. Only cos.Base values that are
// names, numbers, strings or booleans differ from that, and those are compared
// by their own Equals below.
func (p *PDUserProperty) potentiallyNotifyChanged(oldEntry, newEntry any) {
	if isEntryChanged(oldEntry, newEntry) {
		p.userAttributeObject.UserPropertyChanged(p)
	}
}

// isEntryChanged reports whether the two entries differ, the way Java's equals
// does.
func isEntryChanged(oldEntry, newEntry any) bool {
	oldBase, oldIsBase := oldEntry.(cos.Base)
	newBase, newIsBase := newEntry.(cos.Base)
	if oldIsBase && newIsBase {
		return !cos.Equal(oldBase, newBase)
	}
	if oldEntry == nil {
		return newEntry != nil
	}
	return oldEntry != newEntry
}

// Equals reports whether two user properties wrap the same dictionary and hang
// off the same attribute object, which is Java's equals.
func (p *PDUserProperty) Equals(other *PDUserProperty) bool {
	if p == other {
		return true
	}
	if other == nil {
		return false
	}
	if !p.PDDictionaryWrapper.Equals(&other.PDDictionaryWrapper) {
		return false
	}
	return p.userAttributeObject == other.userAttributeObject
}
