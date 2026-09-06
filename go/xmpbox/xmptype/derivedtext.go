package xmptype

// The thirteen text types that add nothing to TextType but a name.
//
// Port of AgentNameType, ChoiceType, GPSCoordinateType, GUIDType, LocaleType,
// MIMEType, PartType, ProperNameType, RationalType, RenditionClassType,
// URIType, URLType and XPathType, each of which Java declares as a subclass of
// TextType whose whole body is a constructor calling super.
//
// The name is not decoration: getFirstEquivalentProperty compares the class and
// toString prints it, so each keeps its own TypeName.
//
// Seven of the Java names collide with a Types constant, which in Go share one
// namespace: MIMEType, PartType, RationalType, URIType, URLType and XPathType
// become MIMEValueType, PartValueType and so on, and TypeName keeps the Java
// name so that everything comparing or printing it is unchanged.

// AgentNameType is a text property naming the software that wrote something.
type AgentNameType struct{ TextType }

// NewAgentNameType returns an agent name property.
func NewAgentNameType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*AgentNameType, error) {
	t := &AgentNameType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *AgentNameType) TypeName() string { return "AgentNameType" }

// String returns the Java toString form.
func (t *AgentNameType) String() string { return simpleString(t) }

// ChoiceType is a text property whose value is one of a closed set.
type ChoiceType struct{ TextType }

// NewChoiceType returns a choice property.
func NewChoiceType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*ChoiceType, error) {
	t := &ChoiceType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *ChoiceType) TypeName() string { return "ChoiceType" }

// String returns the Java toString form.
func (t *ChoiceType) String() string { return simpleString(t) }

// GPSCoordinateType is a text property holding a GPS coordinate.
type GPSCoordinateType struct{ TextType }

// NewGPSCoordinateType returns a GPS coordinate property.
func NewGPSCoordinateType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*GPSCoordinateType, error) {
	t := &GPSCoordinateType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *GPSCoordinateType) TypeName() string { return "GPSCoordinateType" }

// String returns the Java toString form.
func (t *GPSCoordinateType) String() string { return simpleString(t) }

// GUIDType is a text property holding a globally unique identifier.
type GUIDType struct{ TextType }

// NewGUIDType returns a GUID property.
func NewGUIDType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*GUIDType, error) {
	t := &GUIDType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *GUIDType) TypeName() string { return "GUIDType" }

// String returns the Java toString form.
func (t *GUIDType) String() string { return simpleString(t) }

// LocaleType is a text property holding a language tag.
type LocaleType struct{ TextType }

// NewLocaleType returns a locale property.
func NewLocaleType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*LocaleType, error) {
	t := &LocaleType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *LocaleType) TypeName() string { return "LocaleType" }

// String returns the Java toString form.
func (t *LocaleType) String() string { return simpleString(t) }

// MIMEValueType is a text property holding a MIME type.
type MIMEValueType struct{ TextType }

// NewMIMEValueType returns a MIME type property.
func NewMIMEValueType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*MIMEValueType, error) {
	t := &MIMEValueType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *MIMEValueType) TypeName() string { return "MIMEType" }

// String returns the Java toString form.
func (t *MIMEValueType) String() string { return simpleString(t) }

// PartValueType is a text property naming a part of a document.
type PartValueType struct{ TextType }

// NewPartValueType returns a part property.
func NewPartValueType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*PartValueType, error) {
	t := &PartValueType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *PartValueType) TypeName() string { return "PartType" }

// String returns the Java toString form.
func (t *PartValueType) String() string { return simpleString(t) }

// ProperNameType is a text property holding the name of a person or an
// organisation.
type ProperNameType struct{ TextType }

// NewProperNameType returns a proper name property.
func NewProperNameType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*ProperNameType, error) {
	t := &ProperNameType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *ProperNameType) TypeName() string { return "ProperNameType" }

// String returns the Java toString form.
func (t *ProperNameType) String() string { return simpleString(t) }

// RationalValueType is a text property holding a rational number.
type RationalValueType struct{ TextType }

// NewRationalValueType returns a rational property.
func NewRationalValueType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*RationalValueType, error) {
	t := &RationalValueType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *RationalValueType) TypeName() string { return "RationalType" }

// String returns the Java toString form.
func (t *RationalValueType) String() string { return simpleString(t) }

// RenditionClassType is a text property naming a rendition.
type RenditionClassType struct{ TextType }

// NewRenditionClassType returns a rendition class property.
func NewRenditionClassType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*RenditionClassType, error) {
	t := &RenditionClassType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *RenditionClassType) TypeName() string { return "RenditionClassType" }

// String returns the Java toString form.
func (t *RenditionClassType) String() string { return simpleString(t) }

// URIValueType is a text property holding a URI.
type URIValueType struct{ TextType }

// NewURIValueType returns a URI property.
func NewURIValueType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*URIValueType, error) {
	t := &URIValueType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *URIValueType) TypeName() string { return "URIType" }

// String returns the Java toString form.
func (t *URIValueType) String() string { return simpleString(t) }

// URLValueType is a text property holding a URL.
type URLValueType struct{ TextType }

// NewURLValueType returns a URL property.
func NewURLValueType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*URLValueType, error) {
	t := &URLValueType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *URLValueType) TypeName() string { return "URLType" }

// String returns the Java toString form.
func (t *URLValueType) String() string { return simpleString(t) }

// XPathValueType is a text property holding an XPath expression.
type XPathValueType struct{ TextType }

// NewXPathValueType returns an XPath property.
func NewXPathValueType(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (*XPathValueType, error) {
	t := &XPathValueType{}
	if err := t.SetValue(value); err != nil {
		return nil, err
	}
	t.initSimple(metadata, namespaceURI, prefix, propertyName, value)
	return t, nil
}

// TypeName returns the name of the Java class this stands for.
func (t *XPathValueType) TypeName() string { return "XPathType" }

// String returns the Java toString form.
func (t *XPathValueType) String() string { return simpleString(t) }

// init records the constructor of every simple type against the Types constant
// that names it, which is the Class each Java constant carries.
//
// GPSCoordinate and LangAlt name TextType in Java rather than a class of their
// own, and do here too.
func init() {
	simple := func(t Types, construct SimpleConstructor) { RegisterSimpleType(t, construct) }

	simple(Text, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewTextType(m, ns, p, n, v)
	})
	simple(Date, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewDateType(m, ns, p, n, v)
	})
	simple(Boolean, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewBooleanType(m, ns, p, n, v)
	})
	simple(Integer, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewIntegerType(m, ns, p, n, v)
	})
	simple(Real, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewRealType(m, ns, p, n, v)
	})
	simple(GPSCoordinate, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewTextType(m, ns, p, n, v)
	})
	simple(ProperName, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewProperNameType(m, ns, p, n, v)
	})
	simple(Locale, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewLocaleType(m, ns, p, n, v)
	})
	simple(AgentName, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewAgentNameType(m, ns, p, n, v)
	})
	simple(GUID, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewGUIDType(m, ns, p, n, v)
	})
	simple(XPath, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewXPathValueType(m, ns, p, n, v)
	})
	simple(Part, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewPartValueType(m, ns, p, n, v)
	})
	simple(URL, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewURLValueType(m, ns, p, n, v)
	})
	simple(URI, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewURIValueType(m, ns, p, n, v)
	})
	simple(Choice, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewChoiceType(m, ns, p, n, v)
	})
	simple(MIMEType, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewMIMEValueType(m, ns, p, n, v)
	})
	simple(LangAlt, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewTextType(m, ns, p, n, v)
	})
	simple(RenditionClass, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewRenditionClassType(m, ns, p, n, v)
	})
	simple(Rational, func(m MetadataLike, ns, p, n string, v any) (AbstractSimpleProperty, error) {
		return NewRationalValueType(m, ns, p, n, v)
	})
}
