// Package schema holds the XMP schemas: the base every schema is built on, the
// twelve PDFBox knows by name, and the factory that makes one for a namespace.
//
// Port of org.apache.xmpbox.schema.
package schema

// The base every schema is built on.
//
// Port of org.apache.xmpbox.schema.XMPSchema.

import (
	"errors"
	"fmt"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// XMLNSURI is the namespace of the xml:lang attribute.
//
// Port of javax.xml.XMLConstants.XML_NS_URI, of which XMPSchema uses only this.
const XMLNSURI = "http://www.w3.org/XML/1998/namespace"

// XMPSchema is a set of properties in one namespace.
//
// Port of XMPSchema, which extends AbstractStructuredType.
type XMPSchema struct {
	xmptype.StructuredType

	// properties is the description of the fields this schema declares, which
	// Java reads off the class with reflection. The base schema declares none.
	properties *xmptype.PropertiesDescription

	// typeName is what stands for getClass(): merge refuses two schemas of
	// different classes, and the port has no reflection to ask with.
	typeName string
}

var _ xmptype.AbstractStructuredType = (*XMPSchema)(nil)

// NewXMPSchemaFull returns a schema in the given namespace with the given
// prefix and name.
//
// Port of XMPSchema(XMPMetadata, String, String, String).
func NewXMPSchemaFull(metadata xmptype.MetadataLike, namespaceURI, prefix,
	name string) (*XMPSchema, error) {
	s := &XMPSchema{properties: xmptype.NewPropertiesDescription(), typeName: "XMPSchema"}
	if err := s.InitSchema(metadata, xmptype.StructuredTypeInfo{},
		namespaceURI, prefix, name); err != nil {
		return nil, err
	}
	return s, nil
}

// NewXMPSchema returns a schema with no namespace, prefix or name of its own.
//
// Port of XMPSchema(XMPMetadata), which the base class cannot satisfy without a
// namespace; Java raises IllegalArgumentException and the port an error.
func NewXMPSchema(metadata xmptype.MetadataLike) (*XMPSchema, error) {
	return NewXMPSchemaFull(metadata, "", "", "")
}

// NewXMPSchemaPrefixed returns a schema with the given prefix.
//
// Port of XMPSchema(XMPMetadata, String).
func NewXMPSchemaPrefixed(metadata xmptype.MetadataLike, prefix string) (*XMPSchema, error) {
	return NewXMPSchemaFull(metadata, "", prefix, "")
}

// NewXMPSchemaOfNamespace returns a schema in the given namespace with the
// given prefix.
//
// Port of XMPSchema(XMPMetadata, String, String).
func NewXMPSchemaOfNamespace(metadata xmptype.MetadataLike,
	namespaceURI, prefix string) (*XMPSchema, error) {
	return NewXMPSchemaFull(metadata, namespaceURI, prefix, "")
}

// InitSchema is the body of the four Java constructors, which every schema
// embedding XMPSchema calls with its own @StructuredType annotation and
// property description.
func (s *XMPSchema) InitSchema(metadata xmptype.MetadataLike, info xmptype.StructuredTypeInfo,
	namespaceURI, prefix, name string) error {
	if err := s.InitStructuredType(metadata, info, namespaceURI, prefix, name); err != nil {
		return err
	}
	s.AddNamespace(s.Namespace(), s.Prefix())
	return nil
}

// setDescription records what a schema embedding this one declares, which is
// what instanciateSimple reads.
func (s *XMPSchema) setDescription(properties *xmptype.PropertiesDescription, typeName string) {
	s.properties = properties
	s.typeName = typeName
}

// PropertyDescription returns the fields this schema declares.
func (s *XMPSchema) PropertyDescription() *xmptype.PropertiesDescription { return s.properties }

// TypeName returns the name of the Java class this stands for.
func (s *XMPSchema) TypeName() string { return s.typeName }

// AbstractPropertyOf returns the property of the given qualified name, or nil.
//
// Port of getAbstractProperty.
func (s *XMPSchema) AbstractPropertyOf(qualifiedName string) xmptype.AbstractField {
	for _, child := range s.Container().AllProperties() {
		if child.PropertyName() == qualifiedName {
			return child
		}
	}
	return nil
}

// AboutAttribute returns the rdf:about attribute, or nil.
func (s *XMPSchema) AboutAttribute() *xmptype.Attribute {
	return s.Attribute(xmptype.AboutName)
}

// AboutValue returns what the schema is about.
func (s *XMPSchema) AboutValue() string {
	if prop := s.Attribute(xmptype.AboutName); prop != nil {
		return prop.Value()
	}
	// PDFBOX-1685 : if missing, rdf:about should be considered as empty string
	return ""
}

// SetAbout sets the rdf:about attribute, which must be named that.
func (s *XMPSchema) SetAbout(about *xmptype.Attribute) error {
	if xmptype.RDFNamespace == about.Namespace() && xmptype.AboutName == about.Name() {
		s.SetAttribute(about)
		return nil
	}
	return fmt.Errorf("%w: Attribute 'about' must be named 'rdf:about' or 'about'",
		xmptype.ErrBadFieldValue)
}

// SetAboutAsSimple sets what the schema is about.
//
// Java takes a null to mean removal and sets the attribute for every other
// value, the empty string included; a Go string cannot be null, so the removal
// is RemoveAttribute(xmptype.AboutName).
func (s *XMPSchema) SetAboutAsSimple(about string) {
	s.SetAttribute(xmptype.NewAttribute(xmptype.RDFNamespace, xmptype.AboutName, about))
}

// setSpecifiedSimpleTypeProperty replaces the named property with one of the
// given type and value.
//
// Port of the private setSpecifiedSimpleTypeProperty(Types, String, Object).
// Java takes a null value to mean removal.
func (s *XMPSchema) setSpecifiedSimpleTypeProperty(propertyType xmptype.Types,
	qualifiedName string, propertyValue any) error {
	if propertyValue == nil {
		// Search in properties to erase
		for _, child := range s.Container().AllProperties() {
			if child.PropertyName() == qualifiedName {
				s.Container().RemoveProperty(child)
				return nil
			}
		}
		return nil
	}

	tm := s.Metadata().TypeMapping()
	specifiedTypeProperty, err := tm.InstanciateSimpleProperty(
		"", s.Prefix(), qualifiedName, propertyValue, propertyType)
	if err != nil {
		return fmt.Errorf(
			"Failed to create property with the specified type given in parameters: %w", err)
	}
	s.setSpecifiedSimpleTypePropertyValue(specifiedTypeProperty)
	return nil
}

// setSpecifiedSimpleTypePropertyValue replaces the property of the same name
// with the given one.
//
// Port of the private setSpecifiedSimpleTypeProperty(AbstractSimpleProperty).
func (s *XMPSchema) setSpecifiedSimpleTypePropertyValue(prop xmptype.AbstractSimpleProperty) {
	// attribute placement for simple property has been removed
	// Search in properties to erase
	for _, child := range s.AllProperties() {
		if child.PropertyName() == prop.PropertyName() {
			s.RemoveProperty(child)
			s.AddProperty(prop)
			return
		}
	}
	s.AddProperty(prop)
}

// SetTextProperty replaces the property of the same name with the given text.
func (s *XMPSchema) SetTextProperty(prop *xmptype.TextType) {
	s.setSpecifiedSimpleTypePropertyValue(prop)
}

// SetTextPropertyValue sets the named property to the given text.
func (s *XMPSchema) SetTextPropertyValue(qualifiedName, propertyValue string) error {
	return s.setSpecifiedSimpleTypeProperty(xmptype.Text, qualifiedName, propertyValue)
}

// SetTextPropertyValueAsSimple sets the named property to the given text.
func (s *XMPSchema) SetTextPropertyValueAsSimple(simpleName, propertyValue string) error {
	return s.SetTextPropertyValue(simpleName, propertyValue)
}

// RemoveUnqualifiedProperty removes the first property of the given name.
//
// This is the null path of setSpecifiedSimpleTypeProperty(Types, String,
// Object): Java's four value setters take a reference type and remove the
// property when they are handed null. A Go string, bool, int or time.Time
// cannot be null, so the removal is a method of its own; without it the branch
// would be unreachable.
func (s *XMPSchema) RemoveUnqualifiedProperty(qualifiedName string) {
	// Search in properties to erase
	for _, child := range s.Container().AllProperties() {
		if child.PropertyName() == qualifiedName {
			s.Container().RemoveProperty(child)
			return
		}
	}
}

// UnqualifiedTextProperty returns the named text property, and nil where there
// is none.
func (s *XMPSchema) UnqualifiedTextProperty(name string) (*xmptype.TextType, error) {
	prop := s.AbstractPropertyOf(name)
	if prop != nil {
		if text, isText := prop.(*xmptype.TextType); isText {
			return text, nil
		}
		return nil, fmt.Errorf("%w: Property asked is not a Text Property",
			xmptype.ErrBadFieldValue)
	}
	return nil, nil
}

// UnqualifiedTextPropertyValue returns the value of the named text property,
// and the empty string where there is none.
func (s *XMPSchema) UnqualifiedTextPropertyValue(name string) (string, error) {
	tt, err := s.UnqualifiedTextProperty(name)
	if err != nil || tt == nil {
		return "", err
	}
	return tt.StringValue(), nil
}

// DateProperty returns the named date property, and nil where there is none.
func (s *XMPSchema) DateProperty(qualifiedName string) (*xmptype.DateType, error) {
	prop := s.AbstractPropertyOf(qualifiedName)
	if prop != nil {
		if date, isDate := prop.(*xmptype.DateType); isDate {
			return date, nil
		}
		return nil, fmt.Errorf("%w: Property asked is not a Date Property",
			xmptype.ErrBadFieldValue)
	}
	return nil, nil
}

// DatePropertyValueAsSimple returns the value of the named date property.
func (s *XMPSchema) DatePropertyValueAsSimple(simpleName string) (time.Time, bool, error) {
	return s.DatePropertyValue(simpleName)
}

// DatePropertyValue returns the value of the named date property, the second
// result being false where there is none.
func (s *XMPSchema) DatePropertyValue(qualifiedName string) (time.Time, bool, error) {
	prop := s.AbstractPropertyOf(qualifiedName)
	if prop != nil {
		if date, isDate := prop.(*xmptype.DateType); isDate {
			// Java answers getValue(), which is null for a property that is
			// there and holds no date.
			value, held := date.DateValue()
			return value, held, nil
		}
		return time.Time{}, false, fmt.Errorf("%w: Property asked is not a Date Property",
			xmptype.ErrBadFieldValue)
	}
	return time.Time{}, false, nil
}

// SetDateProperty replaces the property of the same name with the given date.
func (s *XMPSchema) SetDateProperty(date *xmptype.DateType) {
	s.setSpecifiedSimpleTypePropertyValue(date)
}

// SetDatePropertyValueAsSimple sets the named property to the given date.
func (s *XMPSchema) SetDatePropertyValueAsSimple(simpleName string, date time.Time) error {
	return s.SetDatePropertyValue(simpleName, date)
}

// SetDatePropertyValue sets the named property to the given date.
func (s *XMPSchema) SetDatePropertyValue(qualifiedName string, date time.Time) error {
	return s.setSpecifiedSimpleTypeProperty(xmptype.Date, qualifiedName, date)
}

// BooleanProperty returns the named boolean property, and nil where there is
// none.
func (s *XMPSchema) BooleanProperty(qualifiedName string) (*xmptype.BooleanType, error) {
	prop := s.AbstractPropertyOf(qualifiedName)
	if prop != nil {
		if boolean, isBoolean := prop.(*xmptype.BooleanType); isBoolean {
			return boolean, nil
		}
		return nil, fmt.Errorf("%w: Property asked is not a Boolean Property",
			xmptype.ErrBadFieldValue)
	}
	return nil, nil
}

// BooleanPropertyValueAsSimple returns the value of the named boolean property.
func (s *XMPSchema) BooleanPropertyValueAsSimple(simpleName string) (bool, bool, error) {
	return s.BooleanPropertyValue(simpleName)
}

// BooleanPropertyValue returns the value of the named boolean property, the
// second result being false where there is none.
func (s *XMPSchema) BooleanPropertyValue(qualifiedName string) (bool, bool, error) {
	prop := s.AbstractPropertyOf(qualifiedName)
	if prop != nil {
		if boolean, isBoolean := prop.(*xmptype.BooleanType); isBoolean {
			return boolean.BooleanValue(), true, nil
		}
		return false, false, fmt.Errorf("%w: Property asked is not a Boolean Property",
			xmptype.ErrBadFieldValue)
	}
	return false, false, nil
}

// SetBooleanProperty replaces the property of the same name with the given
// boolean.
func (s *XMPSchema) SetBooleanProperty(boolean *xmptype.BooleanType) {
	s.setSpecifiedSimpleTypePropertyValue(boolean)
}

// SetBooleanPropertyValueAsSimple sets the named property to the given boolean.
func (s *XMPSchema) SetBooleanPropertyValueAsSimple(simpleName string, boolean bool) error {
	return s.SetBooleanPropertyValue(simpleName, boolean)
}

// SetBooleanPropertyValue sets the named property to the given boolean.
func (s *XMPSchema) SetBooleanPropertyValue(qualifiedName string, boolean bool) error {
	return s.setSpecifiedSimpleTypeProperty(xmptype.Boolean, qualifiedName, boolean)
}

// IntegerProperty returns the named integer property, and nil where there is
// none.
func (s *XMPSchema) IntegerProperty(qualifiedName string) (*xmptype.IntegerType, error) {
	prop := s.AbstractPropertyOf(qualifiedName)
	if prop != nil {
		if integer, isInteger := prop.(*xmptype.IntegerType); isInteger {
			return integer, nil
		}
		return nil, fmt.Errorf("%w: Property asked is not an Integer Property",
			xmptype.ErrBadFieldValue)
	}
	return nil, nil
}

// IntegerPropertyValueAsSimple returns the value of the named integer property.
func (s *XMPSchema) IntegerPropertyValueAsSimple(simpleName string) (int, bool, error) {
	return s.IntegerPropertyValue(simpleName)
}

// IntegerPropertyValue returns the value of the named integer property, the
// second result being false where there is none.
func (s *XMPSchema) IntegerPropertyValue(qualifiedName string) (int, bool, error) {
	prop := s.AbstractPropertyOf(qualifiedName)
	if prop != nil {
		if integer, isInteger := prop.(*xmptype.IntegerType); isInteger {
			return integer.IntegerValue(), true, nil
		}
		return 0, false, fmt.Errorf("%w: Property asked is not an Integer Property",
			xmptype.ErrBadFieldValue)
	}
	return 0, false, nil
}

// SetIntegerProperty replaces the property of the same name with the given
// integer.
func (s *XMPSchema) SetIntegerProperty(prop *xmptype.IntegerType) {
	s.setSpecifiedSimpleTypePropertyValue(prop)
}

// SetIntegerPropertyValueAsSimple sets the named property to the given integer.
func (s *XMPSchema) SetIntegerPropertyValueAsSimple(simpleName string, intValue int) error {
	return s.SetIntegerPropertyValue(simpleName, intValue)
}

// SetIntegerPropertyValue sets the named property to the given integer.
func (s *XMPSchema) SetIntegerPropertyValue(qualifiedName string, intValue int) error {
	return s.setSpecifiedSimpleTypeProperty(xmptype.Integer, qualifiedName, intValue)
}

// arrayOf returns the named property as an array, or nil.
func (s *XMPSchema) arrayOf(name string) *xmptype.ArrayProperty {
	array, isArray := s.AbstractPropertyOf(name).(*xmptype.ArrayProperty)
	if !isArray {
		return nil
	}
	return array
}

// removeUnqualifiedArrayValue removes every element of the named array whose
// string value is the given one.
func (s *XMPSchema) removeUnqualifiedArrayValue(arrayName, fieldValue string) {
	array := s.arrayOf(arrayName)
	if array == nil {
		return
	}
	var toDelete []xmptype.AbstractField
	for _, abstractField := range array.Container().AllProperties() {
		if tmp, isSimple := abstractField.(xmptype.AbstractSimpleProperty); isSimple &&
			tmp.StringValue() == fieldValue {
			toDelete = append(toDelete, tmp)
		}
	}
	for _, tmp := range toDelete {
		array.Container().RemoveProperty(tmp)
	}
}

// RemoveUnqualifiedBagValue removes the given value from the named bag.
func (s *XMPSchema) RemoveUnqualifiedBagValue(bagName, bagValue string) {
	s.removeUnqualifiedArrayValue(bagName, bagValue)
}

// AddBagValueAsSimple appends the given value to the named bag.
func (s *XMPSchema) AddBagValueAsSimple(simpleName, bagValue string) error {
	return s.internalAddBagValue(simpleName, bagValue)
}

// internalAddBagValue appends the given value to the named bag, making it where
// there is none.
func (s *XMPSchema) internalAddBagValue(qualifiedBagName, bagValue string) error {
	bag := s.arrayOf(qualifiedBagName)
	li, err := s.CreateTextType(xmptype.ListName, bagValue)
	if err != nil {
		return err
	}
	if bag != nil {
		bag.Container().AddProperty(li)
		return nil
	}
	newBag := s.CreateArrayProperty(qualifiedBagName, xmptype.Bag)
	newBag.Container().AddProperty(li)
	s.AddProperty(newBag)
	return nil
}

// AddQualifiedBagValue appends the given value to the named bag.
func (s *XMPSchema) AddQualifiedBagValue(simpleName, bagValue string) error {
	return s.internalAddBagValue(simpleName, bagValue)
}

// UnqualifiedBagValueList returns the values of the named bag, and nil where
// there is none.
func (s *XMPSchema) UnqualifiedBagValueList(bagName string) []string {
	if array := s.arrayOf(bagName); array != nil {
		return array.ElementsAsString()
	}
	return nil
}

// RemoveUnqualifiedSequenceValue removes the given value from the named
// sequence.
func (s *XMPSchema) RemoveUnqualifiedSequenceValue(qualifiedSeqName, seqValue string) {
	s.removeUnqualifiedArrayValue(qualifiedSeqName, seqValue)
}

// RemoveUnqualifiedArrayField removes the given field from the named array.
//
// Port of removeUnqualifiedArrayValue(String, AbstractField), which Java
// overloads on the second argument; Go has no overloading, so the port names
// the field form.
//
// Java casts each element to AbstractSimpleProperty before comparing, so an
// array holding a structured value raises ClassCastException; the port compares
// without the cast. See migration/STATUS.md.
func (s *XMPSchema) RemoveUnqualifiedArrayField(arrayName string,
	fieldValue xmptype.AbstractField) {
	array := s.arrayOf(arrayName)
	if array == nil {
		return
	}
	var toDelete []xmptype.AbstractField
	for _, abstractField := range array.Container().AllProperties() {
		// AbstractField does not override equals, so this is identity.
		if abstractField == fieldValue {
			toDelete = append(toDelete, abstractField)
		}
	}
	for _, tmp := range toDelete {
		array.Container().RemoveProperty(tmp)
	}
}

// RemoveUnqualifiedSequenceField removes the given field from the named
// sequence.
func (s *XMPSchema) RemoveUnqualifiedSequenceField(qualifiedSeqName string,
	seqValue xmptype.AbstractField) {
	s.RemoveUnqualifiedArrayField(qualifiedSeqName, seqValue)
}

// AddUnqualifiedSequenceValue appends the given value to the named sequence,
// making it where there is none.
func (s *XMPSchema) AddUnqualifiedSequenceValue(simpleSeqName, seqValue string) error {
	seq := s.arrayOf(simpleSeqName)
	li, err := s.CreateTextType(xmptype.ListName, seqValue)
	if err != nil {
		return err
	}
	if seq != nil {
		seq.Container().AddProperty(li)
		return nil
	}
	newSeq := s.CreateArrayProperty(simpleSeqName, xmptype.Seq)
	newSeq.Container().AddProperty(li)
	s.AddProperty(newSeq)
	return nil
}

// AddBagField appends the given field to the named bag, making it where there
// is none.
//
// Port of addBagValue(String, AbstractField).
func (s *XMPSchema) AddBagField(qualifiedSeqName string, seqValue xmptype.AbstractField) {
	if bag := s.arrayOf(qualifiedSeqName); bag != nil {
		bag.Container().AddProperty(seqValue)
		return
	}
	newBag := s.CreateArrayProperty(qualifiedSeqName, xmptype.Bag)
	newBag.Container().AddProperty(seqValue)
	s.AddProperty(newBag)
}

// AddUnqualifiedSequenceField appends the given field to the named sequence,
// making it where there is none.
//
// Port of addUnqualifiedSequenceValue(String, AbstractField).
func (s *XMPSchema) AddUnqualifiedSequenceField(seqName string, seqValue xmptype.AbstractField) {
	if seq := s.arrayOf(seqName); seq != nil {
		seq.Container().AddProperty(seqValue)
		return
	}
	newSeq := s.CreateArrayProperty(seqName, xmptype.Seq)
	newSeq.Container().AddProperty(seqValue)
	s.AddProperty(newSeq)
}

// UnqualifiedSequenceValueList returns the values of the named sequence, and
// nil where there is none.
func (s *XMPSchema) UnqualifiedSequenceValueList(seqName string) []string {
	if array := s.arrayOf(seqName); array != nil {
		return array.ElementsAsString()
	}
	return nil
}

// RemoveUnqualifiedSequenceDateValue removes the given date from the named
// sequence.
func (s *XMPSchema) RemoveUnqualifiedSequenceDateValue(seqName string, date time.Time) {
	seq := s.arrayOf(seqName)
	if seq == nil {
		return
	}
	var toDelete []xmptype.AbstractField
	for _, tmp := range seq.Container().AllProperties() {
		dateProperty, isDate := tmp.(*xmptype.DateType)
		if !isDate {
			continue
		}
		// Java calls getValue().equals(date) without a null check, so a
		// sequence holding an empty date raises NullPointerException; the port
		// passes over such an element. See migration/JAVA-BUGS.md.
		if value, held := dateProperty.DateValue(); held && value.Equal(date) {
			toDelete = append(toDelete, tmp)
		}
	}
	for _, tmp := range toDelete {
		seq.Container().RemoveProperty(tmp)
	}
}

// AddSequenceDateValueAsSimple appends the given date to the named sequence.
func (s *XMPSchema) AddSequenceDateValueAsSimple(simpleName string, date time.Time) error {
	return s.AddUnqualifiedSequenceDateValue(simpleName, date)
}

// AddUnqualifiedSequenceDateValue appends the given date to the named sequence.
func (s *XMPSchema) AddUnqualifiedSequenceDateValue(seqName string, date time.Time) error {
	value, err := s.Metadata().TypeMapping().CreateDate(
		"", xmptype.DefaultRDFLocalName, xmptype.ListName, date)
	if err != nil {
		return err
	}
	s.AddUnqualifiedSequenceField(seqName, value)
	return nil
}

// UnqualifiedSequenceDateValueList returns the dates of the named sequence, and
// nil where there is no such array.
func (s *XMPSchema) UnqualifiedSequenceDateValueList(seqName string) []time.Time {
	seq := s.arrayOf(seqName)
	if seq == nil {
		return nil
	}
	retval := []time.Time{}
	for _, child := range seq.Container().AllProperties() {
		if date, isDate := child.(*xmptype.DateType); isDate {
			// Java adds getValue(), so an element holding no date puts a null
			// in the list; a []time.Time cannot hold one, so the zero time
			// stands in and the length is the same either way. See
			// migration/STATUS.md.
			value, _ := date.DateValue()
			retval = append(retval, value)
		}
	}
	return retval
}

// ReorganizeAltOrder moves the x-default alternative to the front, which is
// where the specification requires it.
func (s *XMPSchema) ReorganizeAltOrder(alt *xmptype.ComplexPropertyContainer) {
	all := alt.AllProperties()
	// If alternatives contains x-default in first value
	if len(all) > 0 && languageOf(all[0]) == xmptype.XDefault {
		return
	}
	// Find the xdefault definition
	var xdefault xmptype.AbstractField
	xdefaultFound := false
	for i := 1; i < len(all) && !xdefaultFound; i++ {
		xdefault = all[i]
		if languageOf(xdefault) == xmptype.XDefault {
			alt.RemoveProperty(xdefault)
			xdefaultFound = true
		}
	}
	if !xdefaultFound {
		return
	}
	reordered := []xmptype.AbstractField{xdefault}
	var toDelete []xmptype.AbstractField
	for _, tmp := range alt.AllProperties() {
		reordered = append(reordered, tmp)
		toDelete = append(toDelete, tmp)
	}
	for _, tmp := range toDelete {
		alt.RemoveProperty(tmp)
	}
	for _, tmp := range reordered {
		alt.AddProperty(tmp)
	}
}

// languageOf reads the xml:lang attribute of an alternative.
//
// Java dereferences getAttribute(LANG_NAME) without a null check, which is an
// NPE for an alternative that has none; the port answers the empty string, which
// no language equals. See migration/JAVA-BUGS.md.
func languageOf(field xmptype.AbstractField) string {
	if attribute := field.Attribute(xmptype.LangName); attribute != nil {
		return attribute.Value()
	}
	return ""
}

// SetUnqualifiedLanguagePropertyValue sets the value of the named alternative
// in the given language, and removes it where the value is empty.
func (s *XMPSchema) SetUnqualifiedLanguagePropertyValue(name, language, value string) error {
	if language == "" {
		language = xmptype.XDefault
	}
	property := s.AbstractPropertyOf(name)
	if property != nil {
		// Analyzing content of property
		arrayProp, isArray := property.(*xmptype.ArrayProperty)
		if !isArray {
			return nil
		}
		// Try to find a definition
		for _, child := range arrayProp.Container().AllProperties() {
			// try to find the same lang definition
			if languageOf(child) != language {
				continue
			}
			// the same language has been found
			arrayProp.Container().RemoveProperty(child)
			if value != "" {
				langValue, err := s.createLangValue(value, language)
				if err != nil {
					return err
				}
				arrayProp.Container().AddProperty(langValue)
			}
			s.ReorganizeAltOrder(arrayProp.Container())
			return nil
		}
		// if no definition found, we add a new one
		langValue, err := s.createLangValue(value, language)
		if err != nil {
			return err
		}
		arrayProp.Container().AddProperty(langValue)
		s.ReorganizeAltOrder(arrayProp.Container())
		return nil
	}

	arrayProp := s.CreateArrayProperty(name, xmptype.Alt)
	langValue, err := s.createLangValue(value, language)
	if err != nil {
		return err
	}
	arrayProp.Container().AddProperty(langValue)
	s.AddProperty(arrayProp)
	return nil
}

// createLangValue makes one alternative in the given language.
func (s *XMPSchema) createLangValue(value, language string) (*xmptype.TextType, error) {
	langValue, err := s.CreateTextType(xmptype.ListName, value)
	if err != nil {
		return nil, err
	}
	langValue.SetAttribute(xmptype.NewAttribute(XMLNSURI, xmptype.LangName, language))
	return langValue, nil
}

// UnqualifiedLanguagePropertyValue returns the value of the named alternative
// in the given language, and the empty string where there is none.
func (s *XMPSchema) UnqualifiedLanguagePropertyValue(name,
	expectedLanguage string) (string, error) {
	language := expectedLanguage
	if language == "" {
		language = xmptype.XDefault
	}
	property := s.AbstractPropertyOf(name)
	if property == nil {
		return "", nil
	}
	arrayProp, isArray := property.(*xmptype.ArrayProperty)
	if !isArray {
		return "", fmt.Errorf("%w: The property '%s' is not of Lang Alt type",
			xmptype.ErrBadFieldValue, name)
	}
	for _, child := range arrayProp.Container().AllProperties() {
		text := child.Attribute(xmptype.LangName)
		if text != nil && text.Value() == language {
			if value, isText := child.(*xmptype.TextType); isText {
				return value.StringValue(), nil
			}
		}
	}
	return "", nil
}

// UnqualifiedLanguagePropertyLanguagesValue returns the languages the named
// alternative is written in, and nil where there is no such property.
func (s *XMPSchema) UnqualifiedLanguagePropertyLanguagesValue(name string) ([]string, error) {
	property := s.AbstractPropertyOf(name)
	if property == nil {
		// no property with that name
		return nil, nil
	}
	arrayProp, isArray := property.(*xmptype.ArrayProperty)
	if !isArray {
		return nil, fmt.Errorf("%w: The property '%s' is not of Lang Alt type",
			xmptype.ErrBadFieldValue, name)
	}
	allProperties := arrayProp.Container().AllProperties()
	retval := make([]string, 0, len(allProperties))
	for _, child := range allProperties {
		text := child.Attribute(xmptype.LangName)
		if text != nil {
			retval = append(retval, text.Value())
		} else {
			retval = append(retval, xmptype.XDefault)
		}
	}
	return retval, nil
}

// Merge folds the given schema into this one, which must be of the same class.
func (s *XMPSchema) Merge(xmpSchema *XMPSchema) error {
	if xmpSchema.TypeName() != s.TypeName() {
		return errors.New("Can only merge schemas of the same type.")
	}
	for _, att := range xmpSchema.AllAttributes() {
		if att.Namespace() == s.Namespace() {
			s.SetAttribute(att)
		}
	}
	for _, child := range xmpSchema.Container().AllProperties() {
		if child.Prefix() != s.Prefix() {
			continue
		}
		newArray, isArray := child.(*xmptype.ArrayProperty)
		if !isArray {
			s.AddProperty(child)
			continue
		}
		analyzedPropQualifiedName := child.PropertyName()
		for _, tmpEmbeddedProperty := range s.AllProperties() {
			existing, isArray := tmpEmbeddedProperty.(*xmptype.ArrayProperty)
			if isArray && tmpEmbeddedProperty.PropertyName() == analyzedPropQualifiedName {
				mergeComplexProperty(newArray.Container().AllProperties(), existing)
			}
		}
	}
	return nil
}

// mergeComplexProperty appends to the array every new value it does not
// already hold.
//
// Java returns true from the first duplicate and merge takes that as a reason
// to stop, dropping the rest of the array, every later array and every later
// property. A value that is already there is skipped here and the merge goes
// on. See migration/JAVA-BUGS.md 53.
//
// Java casts each element to TextType without checking, so an array holding
// anything else raises ClassCastException; the port skips such an element. See
// migration/STATUS.md.
func mergeComplexProperty(newValues []xmptype.AbstractField,
	arrayProperty *xmptype.ArrayProperty) {
	for _, newValue := range newValues {
		tmpNewValue, isText := newValue.(*xmptype.TextType)
		if !isText {
			continue
		}
		held := false
		for _, abstractField := range arrayProperty.Container().AllProperties() {
			tmpOldValue, isText := abstractField.(*xmptype.TextType)
			if isText && tmpOldValue.StringValue() == tmpNewValue.StringValue() {
				held = true
				break
			}
		}
		if held {
			continue
		}
		arrayProperty.Container().AddProperty(tmpNewValue)
	}
}

// UnqualifiedArrayList returns the elements of the named array, and nil where
// there is none.
func (s *XMPSchema) UnqualifiedArrayList(name string) ([]xmptype.AbstractField, error) {
	var array *xmptype.ArrayProperty
	for _, child := range s.AllProperties() {
		if child.PropertyName() != name {
			continue
		}
		found, isArray := child.(*xmptype.ArrayProperty)
		if !isArray {
			return nil, fmt.Errorf("%w: Property asked is not an array", xmptype.ErrBadFieldValue)
		}
		array = found
		break
	}
	if array != nil {
		return append([]xmptype.AbstractField(nil), array.Container().AllProperties()...), nil
	}
	return nil, nil
}

// InstanciateSimple returns a simple property of the type this schema declares
// for the named field.
//
// Port of the protected instanciateSimple, which passes getClass(); the port
// passes the description built from that class's annotations.
func (s *XMPSchema) InstanciateSimple(propertyName string,
	value any) (xmptype.AbstractSimpleProperty, error) {
	tm := s.Metadata().TypeMapping()
	return tm.InstanciateSimpleField(s.properties, "", s.Prefix(), propertyName, value)
}

// PropertyAs returns the named property as the type T, and nil where it is
// absent or of another type.
//
// Port of the package-private `<T> T getPropertyAs(String, Class<T>)`, which is
// a generic function here for the same reason it is a generic method there.
func PropertyAs[T xmptype.AbstractField](s *XMPSchema, name string) T {
	property, is := s.Property(name).(T)
	if !is {
		var zero T
		return zero
	}
	return property
}

// TextPropertyOf returns the named property as a text property, and nil where
// it is absent or holds something that is not text.
//
// This is PropertyAs for the TextType.class every Java getter of a text field
// asks for. It cannot be PropertyAs itself: Class.isInstance is true of a
// subclass, so a field declared as a URL or an agent name answers its TextType
// there, and a Go type assertion to *TextType is false for a type that only
// embeds one.
func TextPropertyOf(s *XMPSchema, name string) *xmptype.TextType {
	if valued, is := s.Property(name).(xmptype.TextValued); is {
		return valued.TextValue()
	}
	return nil
}

// TextValueOf returns the string value of the named text property, and the
// empty string where it is absent -- which is Java's null.
//
// Java writes this out per accessor as `TextType tt = getPropertyAs(name,
// TextType.class); return tt == null ? null : tt.getStringValue();`.
func TextValueOf(s *XMPSchema, name string) string {
	if tt := TextPropertyOf(s, name); tt != nil {
		return tt.StringValue()
	}
	return ""
}

// SetTextValue adds a text property of the given name and value, which is
// Java's `addProperty(createTextType(name, value))`.
func SetTextValue(s *XMPSchema, name, value string) error {
	text, err := s.CreateTextType(name, value)
	if err != nil {
		return err
	}
	s.AddProperty(text)
	return nil
}
