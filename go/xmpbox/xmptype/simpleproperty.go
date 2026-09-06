package xmptype

// The properties that hold one value.
//
// Port of org.apache.xmpbox.type.AbstractSimpleProperty and the five concrete
// classes over it: BooleanType, DateType, IntegerType, RealType and TextType.
//
// Java's setValue takes an Object and throws IllegalArgumentException for a
// value of the wrong shape. The port takes an any and returns an error, which
// is what migration/conventions/java-to-go.md says an IllegalArgumentException
// out of a constructor becomes: the value comes from a document, so it is the
// document that is wrong rather than the program.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shinguakira/pdfbox-go/go/internal/javafmt"
)

// AbstractSimpleProperty is a property holding a single value.
//
// Port of the abstract AbstractSimpleProperty.
type AbstractSimpleProperty interface {
	AbstractField

	// SetValue sets the value of the property.
	SetValue(value any) error

	// StringValue returns the value as the string XMP writes it as.
	StringValue() string

	// Value returns the value, in whatever Go type the property holds.
	Value() any

	// RawValue returns the value the property was constructed with, before it
	// was interpreted.
	RawValue() any
}

// SimpleProperty is the state and the concrete methods of
// AbstractSimpleProperty, which each of the five embeds.
type SimpleProperty struct {
	Field

	namespace string
	prefix    string
	rawValue  any
}

// initSimple is Java's protected AbstractSimpleProperty constructor, minus the
// setValue it calls: Go cannot dispatch into the embedder from here, so each
// concrete constructor calls its own SetValue and then this.
//
// Java sets rawValue after setValue, so a value setValue rejected never becomes
// a rawValue -- the constructor throws first.
func (p *SimpleProperty) initSimple(metadata MetadataLike, namespaceURI, prefix,
	propertyName string, value any) {
	p.InitField(metadata, propertyName)
	p.namespace = namespaceURI
	p.prefix = prefix
	p.rawValue = value
}

// RawValue returns the value the property was constructed with.
func (p *SimpleProperty) RawValue() any { return p.rawValue }

// Namespace returns the namespace URI of the property.
func (p *SimpleProperty) Namespace() string { return p.namespace }

// Prefix returns the namespace prefix of the property.
func (p *SimpleProperty) Prefix() string { return p.prefix }

// simpleString is Java's AbstractSimpleProperty.toString, which every concrete
// property inherits; getClass().getSimpleName() is TypeName here.
func simpleString(p AbstractSimpleProperty) string {
	return "[" + p.PropertyName() + "=" + p.TypeName() + ":" + p.StringValue() + "]"
}

// The two strings a boolean is written as.
//
// Port of BooleanType.TRUE and BooleanType.FALSE.
const (
	// True is the string a true boolean is written as.
	True = "True"
	// False is the string a false boolean is written as.
	False = "False"
)

// BooleanType is a property holding a boolean.
type BooleanType struct {
	SimpleProperty

	booleanValue bool
}

var _ AbstractSimpleProperty = (*BooleanType)(nil)

// NewBooleanType returns a boolean property with the given value, which may be
// a bool or a string.
func NewBooleanType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*BooleanType, error) {
	b := &BooleanType{}
	if err := b.SetValue(value); err != nil {
		return nil, err
	}
	b.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return b, nil
}

// Value returns the boolean.
func (b *BooleanType) Value() any { return b.booleanValue }

// BooleanValue returns the boolean, typed.
func (b *BooleanType) BooleanValue() bool { return b.booleanValue }

// SetValue sets the boolean from a bool or from the strings "True" and "False",
// in any case.
func (b *BooleanType) SetValue(value any) error {
	switch v := value.(type) {
	case bool:
		b.booleanValue = v
	case string:
		// NumberFormatException is thrown (sub of InvalidArgumentException)
		s := strings.ToUpper(strings.TrimSpace(v))
		switch s {
		case "TRUE":
			b.booleanValue = true
		case "FALSE":
			b.booleanValue = false
		default:
			// unknown value
			return fmt.Errorf("Not a valid boolean value : '%v'", value)
		}
	default:
		// invalid type of value
		return fmt.Errorf("Value given is not allowed for the Boolean type.")
	}
	return nil
}

// StringValue returns "True" or "False".
func (b *BooleanType) StringValue() string {
	if b.booleanValue {
		return True
	}
	return False
}

// TypeName returns the name of the Java class this stands for.
func (b *BooleanType) TypeName() string { return "BooleanType" }

// String returns the Java toString form.
func (b *BooleanType) String() string { return simpleString(b) }

// IntegerType is a property holding an integer.
type IntegerType struct {
	SimpleProperty

	integerValue int
}

var _ AbstractSimpleProperty = (*IntegerType)(nil)

// NewIntegerType returns an integer property with the given value, which may be
// an int or a string.
func NewIntegerType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*IntegerType, error) {
	i := &IntegerType{}
	if err := i.SetValue(value); err != nil {
		return nil, err
	}
	i.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return i, nil
}

// Value returns the integer.
func (i *IntegerType) Value() any { return i.integerValue }

// IntegerValue returns the integer, typed.
func (i *IntegerType) IntegerValue() int { return i.integerValue }

// SetValue sets the integer from an int or a decimal string.
func (i *IntegerType) SetValue(value any) error {
	switch v := value.(type) {
	case int:
		i.integerValue = v
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			// NumberFormatException is thrown (sub of InvalidArgumentException)
			return fmt.Errorf("For input string: %q", v)
		}
		i.integerValue = parsed
	default:
		// invalid type of value
		return fmt.Errorf("Value given is not allowed for the Integer type: %v", value)
	}
	return nil
}

// StringValue returns the integer in decimal.
func (i *IntegerType) StringValue() string { return strconv.Itoa(i.integerValue) }

// TypeName returns the name of the Java class this stands for.
func (i *IntegerType) TypeName() string { return "IntegerType" }

// String returns the Java toString form.
func (i *IntegerType) String() string { return simpleString(i) }

// RealType is a property holding a floating point number.
type RealType struct {
	SimpleProperty

	realValue float32
}

var _ AbstractSimpleProperty = (*RealType)(nil)

// NewRealType returns a real property with the given value, which may be a
// float32 or a string.
func NewRealType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*RealType, error) {
	r := &RealType{}
	if err := r.SetValue(value); err != nil {
		return nil, err
	}
	r.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return r, nil
}

// Value returns the number.
func (r *RealType) Value() any { return r.realValue }

// RealValue returns the number, typed.
func (r *RealType) RealValue() float32 { return r.realValue }

// SetValue sets the number from a float32 or a string.
func (r *RealType) SetValue(value any) error {
	switch v := value.(type) {
	case float32:
		r.realValue = v
	case string:
		// NumberFormatException is thrown (sub of InvalidArgumentException)
		parsed, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return fmt.Errorf("For input string: %q", v)
		}
		r.realValue = float32(parsed)
	default:
		// invalid type of value
		return fmt.Errorf("Value given is not allowed for the Real type: %v", value)
	}
	return nil
}

// StringValue returns the number the way Java's Float.toString writes it.
func (r *RealType) StringValue() string { return javafmt.Float32(r.realValue) }

// TypeName returns the name of the Java class this stands for.
func (r *RealType) TypeName() string { return "RealType" }

// String returns the Java toString form.
func (r *RealType) String() string { return simpleString(r) }

// TextType is a property holding a string.
type TextType struct {
	SimpleProperty

	textValue string
}

var _ AbstractSimpleProperty = (*TextType)(nil)

// NewTextType returns a text property with the given value, which must be a
// string.
func NewTextType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*TextType, error) {
	t := &TextType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// SetValue sets the string.
func (t *TextType) SetValue(value any) error {
	text, isText := value.(string)
	if !isText {
		return fmt.Errorf("Value given is not allowed for the Text type : '%v'", value)
	}
	t.textValue = text
	return nil
}

// StringValue returns the string.
func (t *TextType) StringValue() string { return t.textValue }

// Value returns the string.
func (t *TextType) Value() any { return t.textValue }

// TypeName returns the name of the Java class this stands for.
func (t *TextType) TypeName() string { return "TextType" }

// String returns the Java toString form.
func (t *TextType) String() string { return simpleString(t) }

// DateType is a property holding a date.
type DateType struct {
	SimpleProperty

	dateValue time.Time
	hasValue  bool
}

var _ AbstractSimpleProperty = (*DateType)(nil)

// NewDateType returns a date property with the given value, which may be a
// time.Time or a string.
func NewDateType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*DateType, error) {
	d := &DateType{}
	if err := d.SetValue(value); err != nil {
		return nil, err
	}
	d.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return d, nil
}

// Value returns the date, and nil where the property holds none.
//
// Java's getValue answers the Calendar, which is null for a property built from
// a blank string; see PDFBOX-6029 and setValueFromString below.
func (d *DateType) Value() any {
	if !d.hasValue {
		return nil
	}
	return d.dateValue
}

// DateValue returns the date, typed, the second result being false where the
// property holds none -- which is Java's null getValue().
//
// Every caller has to answer that null onwards: a date property that is there
// and holds nothing is not the same as one holding the epoch.
func (d *DateType) DateValue() (time.Time, bool) { return d.dateValue, d.hasValue }

// isGoodType reports whether the given value is one this property can hold.
func isGoodDate(value any) bool {
	switch v := value.(type) {
	case time.Time:
		return true
	case string:
		_, err := ToCalendar(v)
		return err == nil
	}
	return false
}

// SetValue sets the date from a time.Time or from a string ToCalendar accepts.
func (d *DateType) SetValue(value any) error {
	if !isGoodDate(value) {
		if value == nil {
			return fmt.Errorf("Value null is not allowed for the Date type")
		}
		return fmt.Errorf("Value given is not allowed for the Date type: %T, value: %v",
			value, value)
	}
	// if string object
	if text, isText := value.(string); isText {
		return d.setValueFromString(text)
	}
	// if Calendar
	d.setValueFromCalendar(value.(time.Time))
	return nil
}

// setValueFromCalendar sets the date outright.
func (d *DateType) setValueFromCalendar(value time.Time) {
	d.dateValue = value
	d.hasValue = true
}

// setValueFromString sets the date from its string form.
func (d *DateType) setValueFromString(value string) error {
	parsed, err := ToCalendar(value)
	if err != nil {
		// SHOULD NEVER HAPPEN
		// STRING HAS BEEN CHECKED BEFORE
		return err
	}
	if IsBlankDate(value) {
		// DateConverter.toCalendar answers null for a blank string, which
		// leaves the property with no date at all: getStringValue then answers
		// null and the serializer writes an empty element. See PDFBOX-6029.
		d.dateValue, d.hasValue = time.Time{}, false
		return nil
	}
	d.setValueFromCalendar(parsed)
	return nil
}

// StringValue returns the date in ISO 8601, and the empty string where there is
// none, which is Java's null.
func (d *DateType) StringValue() string {
	if !d.hasValue {
		return ""
	}
	return ToISO8601(d.dateValue)
}

// TypeName returns the name of the Java class this stands for.
func (d *DateType) TypeName() string { return "DateType" }

// String returns the Java toString form.
func (d *DateType) String() string { return simpleString(d) }

// TextValued is what a TextType answers, and so does every type derived from
// it: the text property itself.
//
// Java's getPropertyAs asks Class.isInstance, which is true of a subclass, so a
// getter declared to answer a TextType answers a URLType or an AgentNameType
// where the field holds one. A Go type assertion to *TextType is false for a
// type that embeds it, so the accessors ask for this instead.
type TextValued interface {
	AbstractSimpleProperty

	// TextValue returns the text property this is, or the one it embeds.
	TextValue() *TextType
}

// TextValue returns this property, so that a TextType and everything embedding
// one is TextValued.
func (t *TextType) TextValue() *TextType { return t }
