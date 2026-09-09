package xml

// Reading a packet.
//
// Port of DomXmpParser.

import (
	"bytes"
	"io"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// DomXmpParser reads an XMP packet into metadata.
//
// Port of DomXmpParser. Java holds the DocumentBuilder it was configured with;
// the port's Parse configures its own, because there is nothing to keep.
type DomXmpParser struct {
	nsFinder      *namespaceFinder
	strictParsing bool
}

// NewDomXmpParser returns a parser in strict mode.
//
// Port of the DomXmpParser() constructor, whose ParserConfigurationException
// the port cannot raise.
func NewDomXmpParser() *DomXmpParser {
	return &DomXmpParser{nsFinder: &namespaceFinder{}, strictParsing: true}
}

// IsStrictParsing tells whether strict parsing mode is enabled.
func (p *DomXmpParser) IsStrictParsing() bool { return p.strictParsing }

// SetStrictParsing enables or disables strict parsing mode.
//
// True (the default) means that malformed XMP results in an error, false
// (lenient) means that if malformed content is encountered, the parser
// continues its work if possible. Use strict mode to work with PDF/A files, and
// lenient mode to care more about getting metadata.
func (p *DomXmpParser) SetStrictParsing(strictParsing bool) { p.strictParsing = strictParsing }

// ParseBytes reads a packet from a byte slice.
//
// Port of parse(byte[]).
func (p *DomXmpParser) ParseBytes(xmp []byte) (*xmpbox.XMPMetadata, error) {
	return p.Parse(bytes.NewReader(xmp))
}

// Parse reads a packet.
//
// Port of parse(InputStream).
func (p *DomXmpParser) Parse(input io.Reader) (*xmpbox.XMPMetadata, error) {
	document, err := Parse(input)
	if err != nil {
		return nil, NewXmpParsingErrorCause(Undefined, "Failed to parse: "+err.Error(), err)
	}

	var xmp *xmpbox.XMPMetadata

	// Start reading
	removeCommentsAndBlanks(document)
	nodes := document.ChildNodes()
	at := 0
	node := nodeAt(nodes, at)

	// expect xpacket processing instruction
	pi, isPI := node.(*ProcessingInstruction)
	if !isPI {
		if p.strictParsing {
			return nil, NewXmpParsingError(XpacketBadStart,
				"xmp should start with a processing instruction")
		}
		xmp = xmpbox.CreateXMPMetadataOfXpacket(xmpbox.DefaultXpacketBegin,
			xmpbox.DefaultXpacketID, xmpbox.DefaultXpacketBytes,
			xmpbox.DefaultXpacketEncoding)
	} else {
		if xmp, err = parseInitialXpacket(pi); err != nil {
			return nil, err
		}
		at++
		node = nodeAt(nodes, at)
	}
	// forget other processing instruction
	for {
		if _, isPI := node.(*ProcessingInstruction); !isPI {
			break
		}
		at++
		node = nodeAt(nodes, at)
	}
	// expect root element
	root, isElement := node.(*Element)
	if !isElement {
		return nil, NewXmpParsingError(NoRootElement, "xmp should contain a root element")
	}
	// use this element as root
	at++
	node = nodeAt(nodes, at)
	// expect xpacket end
	if endPI, isPI := node.(*ProcessingInstruction); !isPI {
		if p.strictParsing {
			return nil, NewXmpParsingError(XpacketBadEnd,
				"xmp should end with a processing instruction")
		}
		xmp.SetEndXPacket(xmpbox.DefaultXpacketEnd)
	} else {
		if err := parseEndPacket(xmp, endPI); err != nil {
			return nil, err
		}
		at++
		node = nodeAt(nodes, at)
	}
	// should be null
	if node != nil {
		return nil, NewXmpParsingError(XpacketBadEnd,
			"xmp should end after xpacket end processing instruction")
	}
	// xpacket is OK and there are no more nodes
	// Now, parse the content of root
	p.nsFinder.push(root) // PDFBOX-6138: push namespaces in root
	rdfRdf, err := p.findDescriptionsParent(root)
	if err != nil {
		return nil, err
	}
	p.nsFinder.push(rdfRdf) // PDFBOX-6099: push namespaces in rdf:RDF

	// PDFBOX-6127: look for non standard namespaces (similar to PDFBOX-2378)
	if !p.strictParsing {
		for _, attr := range rdfRdf.Attributes() {
			if XMLNSAttribute == attr.Prefix() {
				maybeAddNonStandardNamespace(xmp, attr)
			}
		}
	}

	descriptions := ElementChildren(rdfRdf)
	for _, description := range descriptions {
		if err := p.parseSchemaExtensions(xmp, description); err != nil {
			return nil, err
		}
	}

	// find schema description
	if err := PopulateSchemaMapping(xmp, p.strictParsing); err != nil {
		return nil, err
	}

	// parse data description
	for _, description := range descriptions {
		if err := p.parseDescriptionRoot(xmp, description); err != nil {
			return nil, err
		}
	}

	p.nsFinder.pop()
	p.nsFinder.pop()

	return xmp, nil
}

// nodeAt returns the node at the given position, and nil past the end -- which
// is what getNextSibling answers for the last node.
func nodeAt(nodes []Node, at int) Node {
	if at < 0 || at >= len(nodes) {
		return nil
	}
	return nodes[at]
}

// maybeAddNonStandardNamespace records a namespace declaration whose schema the
// module does not know.
func maybeAddNonStandardNamespace(xmp *xmpbox.XMPMetadata, attr *Attr) {
	// xmlns:prefix="namespace"
	tm := xmp.TypeMapping()
	namespace := attr.Value()
	if xmptype.RDFNamespace != namespace && !tm.IsStructuredTypeNamespace(namespace) &&
		xmp.Schema(namespace) == nil && tm.SchemaFactory(namespace) == nil {
		// PDFBOX-5128 / PDFBOX-6127: Add the schema on the fly if it can't be
		// found. PDFBOX-5649: But only if the namespace isn't already known,
		// because this adds a namespace without property descriptions.
		// PDFBOX-6127: never rdf
		tm.AddNewNameSpace(namespace, attr.LocalName())
	}
}

// isSchemaExtensionProperty reports whether the element belongs to a PDF/A
// extension schema.
func isSchemaExtensionProperty(element *Element) bool {
	return element != nil && schema.PDFAExtensionPreferedPrefix == element.Prefix()
}

// parseSchemaExtensions reads the PDF/A extension schemas out of one
// description.
func (p *DomXmpParser) parseSchemaExtensions(xmp *xmpbox.XMPMetadata,
	description *Element) error {
	tm := xmp.TypeMapping()
	p.nsFinder.push(description)
	defer p.nsFinder.pop()

	var schemaExtensions []*Element
	for _, child := range ElementChildren(description) {
		if isSchemaExtensionProperty(child) {
			schemaExtensions = append(schemaExtensions, child)
		}
	}
	if len(schemaExtensions) != 0 {
		if err := ValidateNaming(description); err != nil {
			return err
		}
	}
	for _, schemaExtension := range schemaExtensions {
		namespace := schemaExtension.NamespaceURI()
		if !tm.IsDefinedSchema(namespace) {
			return NewXmpParsingError(NoSchema,
				"This namespace is not from a schema: "+namespace)
		}
		propertyType, hasType, err := p.checkPropertyDefinition(tm,
			QNameOf(schemaExtension), "")
		if err != nil {
			return err
		}
		factory, isFactory := tm.SchemaFactory(namespace).(*schema.XMPSchemaFactory)
		if !isFactory {
			continue
		}
		created, err := factory.CreateXMPSchema(xmp, schemaExtension.Prefix())
		if err != nil {
			return NewXmpParsingErrorCause(Undefined, "Parsing failed", err)
		}
		loadAttributes(created, description)
		container := created.Container()
		if err := p.createProperty(xmp, schemaExtension, propertyType, hasType,
			container); err != nil {
			return err
		}
	}
	return nil
}

// parseDescriptionRoot reads one rdf:Description into the schemas it names.
func (p *DomXmpParser) parseDescriptionRoot(xmp *xmpbox.XMPMetadata,
	description *Element) error {
	p.nsFinder.push(description)
	defer p.nsFinder.pop()
	tm := xmp.TypeMapping()

	properties := ElementChildren(description)
	// parse attributes as properties
	for _, attr := range description.Attributes() {
		switch {
		case xmptype.AboutName == attr.LocalName() && attr.Prefix() == "" ||
			xmptype.DefaultRDFPrefix == attr.Prefix():
			// do nothing
		case XMLNSAttribute == attr.Prefix():
			if !p.strictParsing {
				maybeAddNonStandardNamespace(xmp, attr)
			}
		default:
			if err := p.parseDescriptionRootAttr(xmp, description, attr, tm); err != nil {
				return err
			}
		}
	}
	return p.parseChildrenAsProperties(xmp, properties, tm, description)
}

// parseDescriptionRootAttr reads one attribute of a description as a property
// of the schema its namespace names.
func (p *DomXmpParser) parseDescriptionRootAttr(xmp *xmpbox.XMPMetadata,
	description *Element, attr *Attr, tm *xmptype.TypeMapping) error {
	namespace := attr.NamespaceURI()
	var found schema.Schema = xmp.Schema(namespace)
	if isNilSchema(found) {
		factory, isFactory := tm.SchemaFactory(namespace).(*schema.XMPSchemaFactory)
		if !isFactory {
			// Only process when a schema was successfully found
			return nil
		}
		created, err := factory.CreateXMPSchema(xmp, attr.Prefix())
		if err != nil {
			return NewXmpParsingErrorCause(Undefined, "Parsing failed", err)
		}
		found = created
		loadAttributes(found, description)
	}
	container := found.Container()
	propertyType, hasType, err := p.checkPropertyDefinition(tm, xmptype.QName{
		NamespaceURI: attr.NamespaceURI(),
		LocalPart:    attr.LocalName(),
		Prefix:       attr.Prefix(),
	}, "")
	if err != nil {
		return err
	}

	if !hasType {
		if p.strictParsing {
			return NewXmpParsingError(InvalidType,
				"No type defined for {"+attr.NamespaceURI()+"}"+attr.LocalName())
		}
		// PDFBOX-2318, PDFBOX-6106: Default to text if no type is found
		propertyType = xmptype.CreatePropertyType(xmptype.Text, xmptype.Simple)
	} else if !propertyType.Type.IsSimple() || propertyType.Card.IsArray() ||
		propertyType.Type == xmptype.LangAlt {
		if p.strictParsing {
			return NewXmpParsingError(InvalidType, "The type '"+propertyType.Type.String()+
				"' in '"+attr.Prefix()+":"+attr.LocalName()+"="+attr.Value()+
				"' is a structured or array type, but attributes are simple types")
		}
		// PDFBOX-6125: Default to text or skip
		if attr.Value() == "" {
			found.RemoveAttribute(attr.LocalName())
			return nil
		}
		propertyType = xmptype.CreatePropertyType(xmptype.Text, xmptype.Simple)
	}

	sp, err := tm.InstanciateSimpleProperty(namespace, found.Prefix(), attr.LocalName(),
		attr.Value(), propertyType.Type)
	if err != nil {
		return NewXmpParsingErrorCause(Format,
			err.Error()+" in "+found.Prefix()+":"+attr.LocalName(), err)
	}
	container.AddProperty(sp)
	return nil
}

// parseChildrenAsProperties reads the children of a description as properties
// of the schemas they name.
func (p *DomXmpParser) parseChildrenAsProperties(xmp *xmpbox.XMPMetadata,
	properties []*Element, tm *xmptype.TypeMapping, description *Element) error {
	// parse children elements as properties
	for _, property := range properties {
		p.nsFinder.push(property)
		namespace := property.NamespaceURI()
		propertyType, hasType, err := p.checkPropertyDefinition(tm, QNameOf(property), "")
		if err != nil {
			return err
		}
		// create the container
		if !tm.IsDefinedSchema(namespace) {
			return NewXmpParsingError(NoSchema,
				"This namespace is not from a schema: "+namespace)
		}
		if isSchemaExtensionProperty(property) {
			continue
		}
		var found schema.Schema = xmp.Schema(namespace)
		if isNilSchema(found) {
			factory, isFactory := tm.SchemaFactory(namespace).(*schema.XMPSchemaFactory)
			if !isFactory {
				continue
			}
			created, err := factory.CreateXMPSchema(xmp, property.Prefix())
			if err != nil {
				return NewXmpParsingErrorCause(Undefined, "Parsing failed", err)
			}
			found = created
			loadAttributes(found, description)
		}
		container := found.Container()
		// create property
		if err := p.createProperty(xmp, property, propertyType, hasType, container); err != nil {
			return err
		}
		p.nsFinder.pop()
	}
	return nil
}

// createProperty reads one element as a property of the given kind.
func (p *DomXmpParser) createProperty(xmp *xmpbox.XMPMetadata, property *Element,
	propertyType xmptype.PropertyType, hasType bool,
	container *xmptype.ComplexPropertyContainer) error {
	prefix := property.Prefix()
	name := property.LocalName()
	namespace := property.NamespaceURI()
	// create property
	p.nsFinder.push(property)
	defer p.nsFinder.pop()

	var err error
	switch {
	case !hasType:
		if p.strictParsing {
			return NewXmpParsingError(InvalidType, "No type defined for {"+namespace+"}"+name)
		}
		// use it as string
		err = p.manageSimpleType(xmp, property, xmptype.Text, container)
	case propertyType.Type == xmptype.LangAlt:
		err = p.manageLangAlt(xmp, property, container)
	case propertyType.Card.IsArray():
		err = p.manageArray(xmp, property, propertyType, container)
	case propertyType.Type.IsSimple():
		err = p.manageSimpleType(xmp, property, propertyType.Type, container)
	case propertyType.Type.IsStructured():
		err = p.manageStructuredType(xmp, property, prefix, container)
	case propertyType.Type == xmptype.DefinedType:
		err = p.manageDefinedType(xmp, property, prefix, container)
	}
	if err != nil {
		return formatted(err, prefix, name)
	}
	return nil
}

// formatted is Java's catch of IllegalArgumentException around createProperty,
// which rewrites it as a Format problem naming the property.
func formatted(err error, prefix, name string) error {
	var parsing *XmpParsingError
	if asParsing(err, &parsing) {
		return err
	}
	return NewXmpParsingErrorCause(Format, err.Error()+" in "+prefix+":"+name, err)
}

// manageDefinedType reads a property whose type a PDF/A extension schema
// defined.
func (p *DomXmpParser) manageDefinedType(xmp *xmpbox.XMPMetadata, property *Element,
	prefix string, container *xmptype.ComplexPropertyContainer) error {
	if IsParseTypeResource(property) {
		ast, err := p.parseLiDescription(xmp, QNameOf(property), property)
		if err != nil {
			return err
		}
		if ast == nil {
			return NewXmpParsingError(Format,
				"property should contain child elements : "+property.String())
		}
		ast.SetPrefix(prefix)
		container.AddProperty(ast)
		return nil
	}
	inner := FirstChildElement(property)
	if inner == nil {
		return NewXmpParsingError(Format,
			"property should contain child element : "+property.String())
	}
	ast, err := p.parseLiDescription(xmp, QNameOf(property), inner)
	if err != nil {
		return err
	}
	if ast == nil {
		return NewXmpParsingError(Format,
			"inner element should contain child elements : "+inner.String())
	}
	ast.SetPrefix(prefix)
	container.AddProperty(ast)
	return nil
}

// manageStructuredType reads a property whose type the module knows.
func (p *DomXmpParser) manageStructuredType(xmp *xmpbox.XMPMetadata, property *Element,
	prefix string, container *xmptype.ComplexPropertyContainer) error {
	if IsParseTypeResource(property) {
		ast, err := p.parseLiDescription(xmp, QNameOf(property), property)
		if err != nil {
			return err
		}
		if ast != nil {
			ast.SetPrefix(prefix)
			container.AddProperty(ast)
		}
		return nil
	}
	inner := FirstChildElement(property)
	if inner == nil {
		return nil
	}
	p.nsFinder.push(inner)
	defer p.nsFinder.pop()
	ast, err := p.parseLiDescription(xmp, QNameOf(property), inner)
	if err != nil {
		return err
	}
	if ast == nil {
		return NewXmpParsingError(Format,
			"inner element should contain child elements : "+inner.String())
	}
	ast.SetPrefix(prefix)
	container.AddProperty(ast)
	return nil
}

// manageSimpleType reads a property holding one value.
func (p *DomXmpParser) manageSimpleType(xmp *xmpbox.XMPMetadata, property *Element,
	simpleType xmptype.Types, container *xmptype.ComplexPropertyContainer) error {
	tm := xmp.TypeMapping()
	prefix := property.Prefix()
	name := property.LocalName()
	namespace := property.NamespaceURI()
	sp, err := tm.InstanciateSimpleProperty(namespace, prefix, name,
		property.TextContent(), simpleType)
	if err != nil {
		return err
	}
	loadAttributes(sp, property)
	container.AddProperty(sp)
	return nil
}

// manageArray reads a property holding an array.
func (p *DomXmpParser) manageArray(xmp *xmpbox.XMPMetadata, property *Element,
	propertyType xmptype.PropertyType, container *xmptype.ComplexPropertyContainer) error {
	tm := xmp.TypeMapping()
	prefix := property.Prefix()
	name := property.LocalName()
	namespace := property.NamespaceURI()
	bagOrSeq, err := UniqueElementChild(property)
	if err != nil {
		return err
	}
	// ensure this is the good type of array
	if bagOrSeq == nil {
		// not an array
		firstChild := property.FirstChild()
		if !p.strictParsing {
			if firstChild == nil {
				// PDFBOX-6125: ignore
				return nil
			}
			if _, isText := firstChild.(*Text); isText {
				// PDFBOX-6125: Default to text in lenient mode
				// Improvement idea in the future: create an array and add the
				// text item.
				return p.manageSimpleType(xmp, property, xmptype.Text, container)
			}
		}
		whatFound := "nothing"
		if firstChild != nil {
			whatFound = nodeClassName(firstChild)
		}
		return NewXmpParsingError(Format, "Invalid array definition, expecting "+
			propertyType.Card.String()+" and found "+whatFound+
			" [prefix="+prefix+"; name="+name+"]")
	}
	if p.strictParsing && bagOrSeq.LocalName() != propertyType.Card.String() {
		// not the good array type
		return NewXmpParsingError(Format, "Invalid array type, expecting "+
			propertyType.Card.String()+" and found "+bagOrSeq.LocalName()+
			" [prefix="+prefix+"; name="+name+"]")
	}
	array := tm.CreateArrayProperty(namespace, prefix, name, propertyType.Card)
	container.AddProperty(array)

	for _, element := range ElementChildren(bagOrSeq) {
		propertyQName := xmptype.QName{LocalPart: element.LocalName()}
		ast, err := p.parseLiElement(xmp, propertyQName, element, propertyType.Type)
		if err != nil {
			return err
		}
		if ast != nil {
			array.AddProperty(ast)
		}
	}
	return nil
}

// manageLangAlt reads a property holding an alternative of languages.
func (p *DomXmpParser) manageLangAlt(xmp *xmpbox.XMPMetadata, property *Element,
	container *xmptype.ComplexPropertyContainer) error {
	return p.manageArray(xmp, property,
		xmptype.CreatePropertyType(xmptype.LangAlt, xmptype.Alt), container)
}

// parseDescriptionInner reads the properties of a structured value into the
// given container.
func (p *DomXmpParser) parseDescriptionInner(xmp *xmpbox.XMPMetadata, description *Element,
	parentContainer *xmptype.ComplexPropertyContainer) error {
	p.nsFinder.push(description)
	defer p.nsFinder.pop()
	tm := xmp.TypeMapping()

	for _, property := range ElementChildren(description) {
		name := property.LocalName()
		dtype, hasDType, err := p.checkPropertyDefinition(tm, QNameOf(property), "")
		if err != nil {
			return err
		}
		if !hasDType {
			// Java reaches through the null it was handed and raises
			// NullPointerException; the port reports the property instead. See
			// migration/JAVA-BUGS.md.
			return NewXmpParsingError(NoType, "No type defined for {"+
				property.NamespaceURI()+"}"+name)
		}
		mapping := tm.StructuredPropMapping(dtype.Type)
		ptype, hasPType := mapping.PropertyType(name)
		// create property
		if err := p.createProperty(xmp, property, ptype, hasPType, parentContainer); err != nil {
			return err
		}
	}
	return nil
}

// parseLiElement reads one array item.
func (p *DomXmpParser) parseLiElement(xmp *xmpbox.XMPMetadata, descriptor xmptype.QName,
	liElement *Element, itemType xmptype.Types) (xmptype.AbstractField, error) {
	if IsParseTypeResource(liElement) {
		p.nsFinder.push(liElement)
		defer p.nsFinder.pop()
		ast, err := p.parseLiDescription(xmp, descriptor, liElement)
		if ast == nil {
			return nil, err
		}
		return ast, err
	}
	// will find rdf:Description
	liChild, err := UniqueElementChild(liElement)
	if err != nil {
		return nil, err
	}
	if liChild != nil {
		p.nsFinder.push(liElement)
		p.nsFinder.push(liChild)
		defer func() {
			p.nsFinder.pop()
			p.nsFinder.pop()
		}()
		ast, err := p.parseLiDescription(xmp, descriptor, liChild)
		if ast == nil {
			return nil, err
		}
		return ast, err
	}
	// no child
	text := liElement.TextContent()
	tm := xmp.TypeMapping()
	if itemType.IsSimple() {
		af, err := tm.InstanciateSimpleProperty(descriptor.NamespaceURI, descriptor.Prefix,
			descriptor.LocalPart, text, itemType)
		if err != nil {
			return nil, err
		}
		loadAttributes(af, liElement)
		return af, nil
	}
	// PDFBOX-4325: assume it is structured
	af, err := tm.InstanciateStructuredType(itemType, descriptor.LocalPart)
	if err != nil {
		return nil, NewXmpParsingErrorCause(InvalidType,
			"Parsing of structured type failed", err)
	}
	loadAttributes(af, liElement)
	var pm *xmptype.PropertiesDescription
	if itemType.IsStructured() {
		pm = tm.StructuredPropMapping(itemType)
	} else {
		pm = tm.DefinedDescriptionByNamespace(liElement.NamespaceURI(), liElement.LocalName())
	}
	structured, err := p.tryParseAttributesAsProperties(tm, liElement, af, pm, xmptype.QName{})
	if structured == nil {
		return nil, err
	}
	return structured, err
}

// loadAttributes copies the attributes the module keeps as attributes onto the
// property.
func loadAttributes(sp xmptype.AbstractField, element *Element) {
	for _, attr := range element.Attributes() {
		switch {
		case XMLNSAttribute == attr.Prefix():
			// do nothing
		case xmptype.DefaultRDFPrefix == attr.Prefix() && xmptype.AboutName == attr.LocalName():
			// set about
			if xmpSchema, isSchema := sp.(schema.Schema); isSchema {
				xmpSchema.Base().SetAboutAsSimple(attr.Value())
			}
		case XMLNamespace == attr.NamespaceURI():
			// This part was the fallback before PDFBOX-6130, now restricted:
			// Do not load "ordinary" attributes here because these will be
			// handled by tryParseAttributesAsProperties() and
			// parseDescriptionRootAttr()
			sp.SetAttribute(xmptype.NewAttribute(XMLNamespace, attr.LocalName(), attr.Value()))
		}
	}
}

// parseLiDescription reads a structured value out of the element that describes
// it.
func (p *DomXmpParser) parseLiDescription(xmp *xmpbox.XMPMetadata, parentQName xmptype.QName,
	liDescriptionElement *Element) (xmptype.AbstractStructuredType, error) {
	tm := xmp.TypeMapping()
	children := ElementChildren(liDescriptionElement)
	if len(children) == 0 {
		// The list is empty
		return p.tryParseAttributesAsProperties(tm, liDescriptionElement, nil, nil, parentQName)
	}
	firstChild := children[0]
	if "rdf:Description" == firstChild.TagName() {
		// PDFBOX-6126: "<rdf:Description" as child of "<rdf:li"
		return p.parseLiDescription(xmp, parentQName, firstChild)
	}
	// Instantiate abstract structured type with hint from first element
	p.nsFinder.push(firstChild)
	firstChildQName := QNameOf(firstChild)
	ctype, hasCType, err := p.checkPropertyDefinition(tm, firstChildQName,
		parentQName.LocalPart)
	if err != nil {
		return nil, err
	}
	if !hasCType {
		// PDFBOX-5649
		return nil, NewXmpParsingError(NoType, "Property '"+firstChildQName.Prefix+":"+
			firstChildQName.LocalPart+"' not defined in "+firstChildQName.NamespaceURI)
	}
	tt := ctype.Type
	ast, err := p.instanciateStructured(tm, tt, parentQName.LocalPart,
		firstChild.NamespaceURI())
	if err != nil {
		return nil, err
	}

	ast.SetNamespace(firstChild.NamespaceURI())
	ast.SetPrefix(firstChild.Prefix())

	var pm *xmptype.PropertiesDescription
	if tt.IsStructured() {
		pm = tm.StructuredPropMapping(tt)
	} else {
		pm = tm.DefinedDescriptionByNamespace(firstChild.NamespaceURI(), firstChild.LocalName())
	}
	for _, child := range children {
		prefix := child.Prefix()
		name := child.LocalName()
		namespace := child.NamespaceURI()
		propertyType, hasType := pm.PropertyType(name)
		if !hasType {
			if p.strictParsing {
				return nil, NewXmpParsingError(NoType, "Type '"+prefix+":"+name+
					"' not defined in "+child.NamespaceURI())
			}
			// PDFBOX-6135: Default to text if no type is found
			propertyType = xmptype.CreatePropertyType(xmptype.Text, xmptype.Simple)
		}
		switch {
		case propertyType.Card.IsArray():
			array := tm.CreateArrayProperty(namespace, prefix, name, propertyType.Card)
			ast.Container().AddProperty(array)
			bagOrSeq, err := UniqueElementChild(child)
			if err != nil {
				return nil, err
			}
			for _, element2 := range ElementChildren(bagOrSeq) {
				ast2, err := p.parseLiElement(xmp, parentQName, element2, propertyType.Type)
				if err != nil {
					return nil, err
				}
				if ast2 != nil {
					array.AddProperty(ast2)
				}
			}
		case propertyType.Type.IsSimple():
			sp, err := tm.InstanciateSimpleProperty(namespace, prefix, name,
				child.TextContent(), propertyType.Type)
			if err != nil {
				return nil, err
			}
			loadAttributes(sp, child)
			ast.Container().AddProperty(sp)
		case propertyType.Type.IsStructured():
			// create a new structured type
			inner, err := p.instanciateStructured(tm, propertyType.Type, name, "")
			if err != nil {
				return nil, err
			}
			inner.SetNamespace(namespace)
			inner.SetPrefix(prefix)
			ast.Container().AddProperty(inner)
			cpc := inner.Container()
			if IsParseTypeResource(child) {
				if err := p.parseDescriptionInner(xmp, child, cpc); err != nil {
					return nil, err
				}
			} else if descElement := FirstChildElement(child); descElement != nil {
				if err := p.parseDescriptionInner(xmp, descElement, cpc); err != nil {
					return nil, err
				}
			}
		default:
			return nil, NewXmpParsingError(NoType, "Unidentified element to parse "+
				child.String()+" (type="+propertyType.String()+")")
		}
	}
	ast, err = p.tryParseAttributesAsProperties(tm, liDescriptionElement, ast, pm, parentQName)
	p.nsFinder.pop()
	return ast, err
}

// parseInitialXpacket reads the xpacket instruction a packet opens with.
func parseInitialXpacket(pi *ProcessingInstruction) (*xmpbox.XMPMetadata, error) {
	if "xpacket" != pi.NodeName() {
		return nil, NewXmpParsingError(XpacketBadStart,
			"Bad processing instruction name : "+pi.NodeName())
	}
	data := pi.Data()
	var id, begin, encoding string
	var bytesValue *string
	for _, token := range strings.FieldsFunc(data, func(r rune) bool { return r == ' ' }) {
		if !strings.HasSuffix(token, "\"") && !strings.HasSuffix(token, "'") {
			return nil, NewXmpParsingError(XpacketBadStart,
				"Cannot understand PI data part : '"+token+"' in '"+data+"'")
		}
		quote := token[len(token)-1:]
		pos := strings.Index(token, "="+quote)
		if pos <= 0 {
			return nil, NewXmpParsingError(XpacketBadStart,
				"Cannot understand PI data part : '"+token+"' in '"+data+"'")
		}
		name := token[:pos]
		if len(token)-1 < pos+2 {
			return nil, NewXmpParsingError(XpacketBadStart,
				"Cannot understand PI data part : '"+token+"' in '"+data+"'")
		}
		value := token[pos+2 : len(token)-1]
		switch name {
		case "id":
			id = value
		case "begin":
			begin = value
		case "bytes":
			held := value
			bytesValue = &held
		case "encoding":
			encoding = value
		default:
			return nil, NewXmpParsingError(XpacketBadStart,
				"Unknown attribute in xpacket PI : '"+token+"'")
		}
	}
	return xmpbox.CreateXMPMetadataOfXpacket(begin, id, bytesValue, encoding), nil
}

// parseEndPacket reads the xpacket instruction a packet closes with.
func parseEndPacket(metadata *xmpbox.XMPMetadata, pi *ProcessingInstruction) error {
	xpackData := pi.Data()
	// end attribute must be present and placed in first
	// xmp spec says Other unrecognized attributes can follow, but
	// should be ignored
	if !strings.HasPrefix(xpackData, "end=") {
		// should find end='r/w'
		return NewXmpParsingError(XpacketBadEnd,
			"Expected xpacket 'end' attribute (must be present and placed in first)")
	}
	// Java indexes the sixth character without checking the length, which
	// raises StringIndexOutOfBoundsException for a shorter instruction --
	// unchecked, so it escapes the XmpParsingException a caller catches. A
	// shorter instruction gets the same answer as a sixth character that is
	// neither 'r' nor 'w'. See migration/JAVA-BUGS.md 56.
	if len(xpackData) < 6 {
		return NewXmpParsingError(XpacketBadEnd,
			"Expected xpacket 'end' attribute with value 'r' or 'w' ")
	}
	end := xpackData[5]
	// check value (5 for end='X')
	if end != 'r' && end != 'w' {
		return NewXmpParsingError(XpacketBadEnd,
			"Expected xpacket 'end' attribute with value 'r' or 'w' ")
	}
	metadata.SetEndXPacket(string(end))
	return nil
}

// findDescriptionsParent returns the rdf:RDF element the descriptions are
// under, the x:xmpmeta wrapper around it being optional.
func (p *DomXmpParser) findDescriptionsParent(root *Element) (*Element, error) {
	var rdfRdf *Element
	// check if already rdf element, as xmpmeta wrapper can be optional
	if xmptype.RDFNamespace != root.NamespaceURI() {
		// always <x:xmpmeta xmlns:x="adobe:ns:meta/">
		if !p.strictParsing && "xapmeta" == root.LocalName() {
			// older XMP content
			if err := expectNaming(root, "adobe:ns:meta/", "x", "xapmeta"); err != nil {
				return nil, err
			}
		} else if err := expectNaming(root, "adobe:ns:meta/", "x", "xmpmeta"); err != nil {
			return nil, err
		}
		// should only have one child
		children := root.ChildNodes()
		switch {
		case len(children) == 0:
			// empty description
			return nil, NewXmpParsingError(Format, "No rdf description found in xmp")
		case len(children) > 1:
			// only expect one element
			return nil, NewXmpParsingError(Format, "More than one element found in x:xmpmeta")
		}
		firstChild, isElement := children[0].(*Element)
		if !isElement {
			// should be an element
			return nil, NewXmpParsingError(Format,
				"x:xmpmeta does not contains rdf:RDF element but "+nodeString(children[0]))
		} // else let's parse
		rdfRdf = firstChild
	} else {
		rdfRdf = root
	}
	// always <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
	if err := expectNaming(rdfRdf, xmptype.RDFNamespace, xmptype.DefaultRDFPrefix,
		xmptype.DefaultRDFLocalName); err != nil {
		return nil, err
	}
	// return description parent
	return rdfRdf, nil
}

// expectNaming reports an element that is not named as it should be.
//
// The three messages concatenate a value that is null in Java where the element
// has no namespace or no prefix, and Java writes a null as "null".
func expectNaming(element *Element, ns, prefix, ln string) error {
	switch {
	case ns != "" && ns != element.NamespaceURI():
		return NewXmpParsingError(Format, "Expecting namespace '"+ns+"' and found '"+
			orNull(element.NamespaceURI())+"'")
	case prefix != "" && prefix != element.Prefix():
		return NewXmpParsingError(Format, "Expecting prefix '"+prefix+"' and found '"+
			orNull(element.Prefix())+"'")
	case ln != "" && ln != element.LocalName():
		return NewXmpParsingError(Format, "Expecting local name '"+ln+"' and found '"+
			element.LocalName()+"'")
	}
	// else OK
	return nil
}

// orNull writes an absent string the way Java's string concatenation writes a
// null.
func orNull(value string) string {
	if value == "" {
		return "null"
	}
	return value
}

// removeCommentsAndBlanks removes every comment and blank node below the node.
func removeCommentsAndBlanks(root Node) {
	// will hold the nodes which are to be deleted
	var forDeletion []Node

	var children []Node
	switch root := root.(type) {
	case *Document:
		children = root.ChildNodes()
	case *Element:
		children = root.ChildNodes()
	default:
		return
	}

	if _, isDocument := root.(*Document); !isDocument && len(children) <= 1 {
		// There is only one node so we're done, except when Document
		return
	}

	for _, node := range children {
		switch node := node.(type) {
		case *Comment:
			// comments to be deleted
			forDeletion = append(forDeletion, node)
		case *Text:
			if isBlank(node.Data()) {
				// empty text nodes to be deleted
				forDeletion = append(forDeletion, node)
			}
		case *Element:
			// clean child
			removeCommentsAndBlanks(node)
		} // else do nothing
	}

	// now remove the child nodes
	for _, node := range forDeletion {
		switch root := root.(type) {
		case *Document:
			root.RemoveChild(node)
		case *Element:
			root.RemoveChild(node)
		}
	}
}

// isBlank is Java's String.isBlank: empty, or every character a Java whitespace
// character.
//
// Java's Character.isWhitespace is not Go's unicode.IsSpace: it leaves out the
// three non-breaking space separators and takes in the four file separators.
func isBlank(text string) bool {
	for _, r := range text {
		if !isJavaWhitespace(r) {
			return false
		}
	}
	return true
}

// isJavaWhitespace is Character.isWhitespace.
func isJavaWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', 0x1C, 0x1D, 0x1E, 0x1F:
		return true
	case 0x00A0, 0x2007, 0x202F:
		// the three space separators that are not whitespace, because they do
		// not break a line
		return false
	}
	return isSpaceSeparator(r) || r == 0x1680 || r == 0x2028 || r == 0x2029
}

// isSpaceSeparator is the Unicode category Zs.
func isSpaceSeparator(r rune) bool {
	switch {
	case r == 0x0020, r == 0x00A0, r == 0x1680, r == 0x202F, r == 0x205F, r == 0x3000:
		return true
	case r >= 0x2000 && r <= 0x200A:
		return true
	}
	return false
}

// instanciateStructured returns a new value of a structured or defined type.
func (p *DomXmpParser) instanciateStructured(tm *xmptype.TypeMapping, structuredType xmptype.Types,
	name, structuredNamespace string) (xmptype.AbstractStructuredType, error) {
	switch {
	case structuredType.IsStructured():
		ast, err := tm.InstanciateStructuredType(structuredType, name)
		if err != nil {
			return nil, NewXmpParsingErrorCause(InvalidType, "Parsing failed", err)
		}
		return ast, nil
	case structuredType.IsDefined():
		return tm.InstanciateDefinedType(name, structuredNamespace), nil
	default:
		return nil, NewXmpParsingError(InvalidType,
			"Type not structured : "+structuredType.String())
	}
}

// checkPropertyDefinition returns the type declared for the given name, the
// second result being false where none is -- which is Java's null.
func (p *DomXmpParser) checkPropertyDefinition(tm *xmptype.TypeMapping, qName xmptype.QName,
	parentTypeName string) (xmptype.PropertyType, bool, error) {
	// test if namespace is set in xml
	nsuri := qName.NamespaceURI
	if !p.nsFinder.containsNamespace(nsuri) {
		return xmptype.PropertyType{}, false, NewXmpParsingError(NoSchema,
			"Schema is not set in this document : "+nsuri+", property: "+
				qName.Prefix+":"+qName.LocalPart)
	}
	// test if namespace is defined
	if !tm.IsDefinedNamespace(nsuri) {
		return xmptype.PropertyType{}, false, NewXmpParsingError(NoSchema,
			"Cannot find a definition for the namespace "+nsuri+", property: "+
				qName.Prefix+":"+qName.LocalPart)
	}
	propertyType, found, err := tm.SpecifiedPropertyType(qName, parentTypeName)
	if err != nil {
		return xmptype.PropertyType{}, false, NewXmpParsingErrorCause(InvalidType,
			"Failed to retrieve property definition for "+qName.String(), err)
	}
	return propertyType, found, nil
}

// tryParseAttributesAsProperties runs the same logic as parseLiDescription but
// with simple attributes that are treated like children.
//
// This is inspired by loadAttributes and parseDescriptionRootAttr, and solves
// the problem in PDFBOX-3882 where properties appear as attributes in places
// lower than the descriptor root. ast may be nil, in which case it is created
// here; pm and qName must be set where ast is not nil.
func (p *DomXmpParser) tryParseAttributesAsProperties(tm *xmptype.TypeMapping, liElement *Element,
	ast xmptype.AbstractStructuredType, pm *xmptype.PropertiesDescription,
	qName xmptype.QName) (xmptype.AbstractStructuredType, error) {
	for _, attr := range liElement.Attributes() {
		if XMLNSAttribute == attr.Prefix() || XMLNamespace == attr.NamespaceURI() ||
			xmptype.DefaultRDFPrefix == attr.Prefix() {
			// do nothing
			continue
		}
		if ast == nil && attr.NamespaceURI() != "" {
			// What to do if attr.getNamespaceURI() is null?
			// like in parseLiDescription():
			// Instantiate abstract structured type with hint from first element
			attrQName := xmptype.QName{
				NamespaceURI: attr.NamespaceURI(),
				LocalPart:    attr.LocalName(),
				Prefix:       attr.Prefix(),
			}
			ctype, hasCType, err := p.checkPropertyDefinition(tm, attrQName, "")
			if err != nil {
				return nil, err
			}
			// this is the type of the AbstractStructuredType, not of the
			// element(s)
			if !hasCType {
				return nil, NewXmpParsingError(NoType, "Property '"+attrQName.LocalPart+
					"' not defined in "+attrQName.NamespaceURI)
			}
			tt := ctype.Type
			if ast, err = p.instanciateStructured(tm, tt, qName.LocalPart,
				attr.NamespaceURI()); err != nil {
				return nil, err
			}
			if tt.IsStructured() {
				pm = tm.StructuredPropMapping(tt)
			} else {
				pm = tm.DefinedDescriptionByNamespace(attr.NamespaceURI(), attr.LocalName())
			}
		}
		if ast != nil && pm != nil && attr.NamespaceURI() != "" {
			propertyType, hasType := pm.PropertyType(attr.LocalName())
			if !hasType {
				if p.strictParsing {
					return nil, NewXmpParsingError(InvalidType, "No type defined for {"+
						attr.NamespaceURI()+"}"+attr.LocalName())
				}
				// PDFBOX-2318, PDFBOX-6106: Default to text if no type is found
				propertyType = xmptype.CreatePropertyType(xmptype.Text, xmptype.Simple)
			} else if !propertyType.Type.IsSimple() || propertyType.Card.IsArray() ||
				propertyType.Type == xmptype.LangAlt {
				if p.strictParsing {
					return nil, NewXmpParsingError(InvalidType, "The type '"+
						propertyType.Type.String()+"' in '"+attr.Prefix()+":"+
						attr.LocalName()+"="+attr.Value()+
						"' is a structured or array type, but attributes are simple types")
				}
				// PDFBOX-6125: Default to text or skip
				if attr.Value() == "" {
					continue
				}
				propertyType = xmptype.CreatePropertyType(xmptype.Text, xmptype.Simple)
			}
			asp, err := tm.InstanciateSimpleProperty(attr.NamespaceURI(), attr.Prefix(),
				attr.LocalName(), attr.Value(), propertyType.Type)
			if err != nil {
				return nil, err
			}
			ast.Container().AddProperty(asp)
		}
	}
	return ast, nil
}

// namespaceFinder is the namespace declarations of every description the parser
// is inside.
//
// Port of DomXmpParser.NamespaceFinder.
type namespaceFinder struct {
	stack []map[string]string
}

// push records the namespaces the description declares.
func (n *namespaceFinder) push(description *Element) {
	declared := map[string]string{}
	if description != nil {
		for _, attr := range description.Attributes() {
			// if ns definition add it
			if XMLNSNamespace == attr.NamespaceURI() {
				declared[attr.LocalName()] = attr.Value()
			}
		}
	}
	n.stack = append(n.stack, declared)
}

// pop forgets the innermost description's namespaces.
func (n *namespaceFinder) pop() map[string]string {
	if len(n.stack) == 0 {
		return nil
	}
	popped := n.stack[len(n.stack)-1]
	n.stack = n.stack[:len(n.stack)-1]
	return popped
}

// containsNamespace reports whether any description in scope declared the
// namespace.
func (n *namespaceFinder) containsNamespace(namespace string) bool {
	for _, declared := range n.stack {
		for _, value := range declared {
			if value == namespace {
				return true
			}
		}
	}
	return false
}

// nodeClassName is what Java's node.getClass().getName() answers, which the
// array message writes for anything that is not a Text.
func nodeClassName(node Node) string {
	switch node.(type) {
	case *Text:
		return "Text"
	case *Element:
		return "com.sun.org.apache.xerces.internal.dom.ElementNSImpl"
	case *Comment:
		return "com.sun.org.apache.xerces.internal.dom.CommentImpl"
	case *ProcessingInstruction:
		return "com.sun.org.apache.xerces.internal.dom.ProcessingInstructionImpl"
	}
	return "org.w3c.dom.Node"
}

// nodeString is what a node reads as inside a message, which is what Xerces
// writes for the node kind: the name in brackets, and the data after it.
func nodeString(node Node) string {
	switch node := node.(type) {
	case *Element:
		return node.String()
	case *Text:
		return "[#text: " + node.Data() + "]"
	case *Comment:
		return "[#comment: " + node.Data() + "]"
	case *ProcessingInstruction:
		return "[" + node.NodeName() + ": " + node.Data() + "]"
	}
	return "[" + nodeClassName(node) + "]"
}

// isNilSchema reports whether the interface holds no schema, which a typed nil
// out of a lookup would otherwise hide.
func isNilSchema(s schema.Schema) bool { return s == nil }

// asParsing is errors.As for a parsing error, kept here so that formatted reads
// as the catch it ports.
func asParsing(err error, target **XmpParsingError) bool {
	parsing, is := err.(*XmpParsingError)
	if is {
		*target = parsing
	}
	return is
}
