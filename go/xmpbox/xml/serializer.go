package xml

// Writing a packet back out.
//
// Port of XmpSerializer.

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// XmpSerializer writes a packet.
//
// Port of XmpSerializer. Java holds a TransformerFactory and a DocumentBuilder,
// which the port has no need of; what it keeps is the rdf element, which
// serializeFields reaches to declare a namespace at the top.
type XmpSerializer struct {
	rdf *Element
}

// NewXmpSerializer returns a serializer.
//
// Port of the default constructor. Java's second constructor takes the two
// factories to build documents and transformers with; the port has neither.
func NewXmpSerializer() *XmpSerializer { return &XmpSerializer{} }

// Serialize writes the metadata.
//
// Port of serialize(XMPMetadata, OutputStream, boolean).
func (s *XmpSerializer) Serialize(metadata *xmpbox.XMPMetadata, out io.Writer,
	withXpacket bool) error {
	doc := NewDocument()
	// fill document
	s.rdf = s.CreateRdfElement(doc, metadata, withXpacket)
	for _, declared := range metadata.AllSchemas() {
		s.rdf.AppendChild(s.SerializeSchema(doc, declared.Base()))
	}
	// save
	if err := save(doc, out); err != nil {
		return errSerialization(err)
	}
	return nil
}

// SerializeSchema returns the rdf:Description one schema is written as.
//
// Port of serializeSchema(Document, XMPSchema).
func (s *XmpSerializer) SerializeSchema(doc *Document, sc *schema.XMPSchema) *Element {
	// prepare schema
	selem := doc.CreateElementNS(xmptype.RDFNamespace, "rdf:Description")
	selem.SetAttributeNS(xmptype.RDFNamespace, "rdf:about", sc.AboutValue())
	selem.SetAttributeNS(XMLNSNamespace, "xmlns:"+sc.Prefix(), sc.Namespace())
	// the other attributes
	s.fillElementWithAttributes(selem, sc)
	// the content
	fields := sc.AllProperties()
	s.SerializeFields(doc, selem, fields, sc.Prefix(), "", true)
	// return created schema
	return selem
}

// SerializeFields writes a list of properties below the given element.
//
// Port of serializeFields(Document, Element, List, String, String, boolean).
func (s *XmpSerializer) SerializeFields(doc *Document, parent *Element,
	fields []xmptype.AbstractField, resourceNS, prefix string, wrapWithProperty bool) {
	usePrefix := prefix != ""
	for _, field := range fields {
		switch field := field.(type) {
		case xmptype.AbstractSimpleProperty:
			localPrefix := field.Prefix()
			if usePrefix {
				localPrefix = prefix
			}

			esimple := doc.CreateElement(localPrefix + ":" + field.PropertyName())
			esimple.SetTextContent(field.StringValue())
			for _, attribute := range field.AllAttributes() {
				name := attribute.Name()
				// we must add "xml:" to the qualifiedName parameter or it
				// won't appear in the result
				if XMLNamespace == attribute.Namespace() && !strings.Contains(name, ":") {
					esimple.SetAttributeNS(XMLNamespace, XMLNSPrefix+":"+name,
						attribute.Value())
				} else {
					esimple.SetAttributeNS(attribute.Namespace(), name, attribute.Value())
				}
			}

			// PDFBOX-2378: add namespace declaration to the top
			if field.Prefix() != "" && field.Namespace() != "" {
				s.rdf.SetAttributeNS(XMLNSNamespace, "xmlns:"+field.Prefix(),
					field.Namespace())
			}

			parent.AppendChild(esimple)

		case *xmptype.ArrayProperty:
			// property
			asimple := doc.CreateElement(field.Prefix() + ":" + field.PropertyName())
			parent.AppendChild(asimple)
			// attributes
			s.fillElementWithAttributes(asimple, field)
			// the array definition
			econtainer := doc.CreateElement(
				xmptype.DefaultRDFPrefix + ":" + field.ArrayType().String())
			asimple.AppendChild(econtainer)
			// for each element of the array
			s.SerializeFields(doc, econtainer, field.AllProperties(), resourceNS,
				xmptype.DefaultRDFPrefix, false)

		case xmptype.AbstractStructuredType:
			innerFields := field.AllProperties()
			// property name attribute
			listParent := parent
			if wrapWithProperty {
				nstructured := doc.CreateElement(resourceNS + ":" + field.PropertyName())
				parent.AppendChild(nstructured)
				listParent = nstructured
			}

			// element li
			estructured := doc.CreateElement(
				xmptype.DefaultRDFPrefix + ":" + xmptype.ListName)
			listParent.AppendChild(estructured)
			estructured.SetAttribute("rdf:parseType", "Resource")

			// all properties
			s.SerializeFields(doc, estructured, innerFields, resourceNS, "", true)
		}
		// else doesn't happen:
		// AbstractField is only extended by AbstractSimpleProperty and
		// AbstractComplexProperty; AbstractComplexProperty is only extended by
		// AbstractStructuredType and ArrayProperty. And all these are handled
		// here.
	}
}

// fillElementWithAttributes writes a complex property's attributes and its
// namespace declarations onto an element.
func (s *XmpSerializer) fillElementWithAttributes(target *Element,
	property xmptype.AbstractComplexProperty) {
	// normalize the attributes list
	for _, attribute := range normalizeAttributes(property) {
		if xmptype.RDFNamespace == attribute.Namespace() {
			target.SetAttribute(xmptype.DefaultRDFPrefix+":"+attribute.Name(),
				attribute.Value())
		} else {
			target.SetAttribute(attribute.Name(), attribute.Value())
		}
	}

	// Java walks a HashMap, whose order is arbitrary but fixed for a given
	// content; the port sorts so that the same metadata always writes the same
	// bytes.
	namespaces := property.AllNamespacesWithPrefix()
	sorted := make([]string, 0, len(namespaces))
	for namespace := range namespaces {
		sorted = append(sorted, namespace)
	}
	sort.Strings(sorted)
	for _, namespace := range sorted {
		target.SetAttribute(XMLNSAttribute+":"+namespaces[namespace], namespace)
	}
}

// normalizeAttributes returns the attributes that do not match a property,
// because the ones that do are written as child elements.
func normalizeAttributes(property xmptype.AbstractComplexProperty) []*xmptype.Attribute {
	var toSerialize []*xmptype.Attribute
	fields := property.AllProperties()
	for _, attribute := range property.AllAttributes() {
		matchesField := false
		for _, field := range fields {
			if attribute.Name() == field.PropertyName() {
				matchesField = true
				break
			}
		}
		if !matchesField {
			toSerialize = append(toSerialize, attribute)
		}
	}
	return toSerialize
}

// CreateRdfElement builds the packet around the rdf:RDF element and returns
// that element.
//
// Port of createRdfElement(Document, XMPMetadata, boolean).
func (s *XmpSerializer) CreateRdfElement(doc *Document, metadata *xmpbox.XMPMetadata,
	withXpacket bool) *Element {
	// starting xpacket
	if withXpacket {
		beginXPacket := doc.CreateProcessingInstruction("xpacket",
			"begin=\""+metadata.XpacketBegin()+"\" id=\""+metadata.XpacketID()+"\"")
		doc.AppendChild(beginXPacket)
	}
	// meta element
	xmpmeta := doc.CreateElementNS("adobe:ns:meta/", "x:xmpmeta")
	xmpmeta.SetAttributeNS(XMLNSNamespace, "xmlns:x", "adobe:ns:meta/")
	doc.AppendChild(xmpmeta)
	// ending xpacket
	if withXpacket {
		endXPacket := doc.CreateProcessingInstruction("xpacket",
			"end=\""+metadata.EndXPacket()+"\"")
		doc.AppendChild(endXPacket)
	}
	// rdf element
	rdfElement := doc.CreateElementNS(xmptype.RDFNamespace, "rdf:RDF")
	xmpmeta.AppendChild(rdfElement)
	// return the rdf element where all will be put
	return rdfElement
}

// save writes the document.
//
// Port of save(Node, OutputStream, String) as XmpSerializer configures the
// transformer: indent on, two spaces a level, UTF-8, and no XML declaration.
func save(doc *Document, out io.Writer) error {
	w := &indentWriter{out: out}
	for _, child := range doc.ChildNodes() {
		w.writeNode(child, 0, nil)
	}
	// The transformer ends the document with a line separator. Java's is the
	// platform's, which is CRLF on Windows; the port always writes LF, and
	// every caller that compares output normalizes the two the same way. See
	// migration/STATUS.md.
	w.write("\n")
	return w.err
}

// indentWriter writes nodes out with the indenting the transformer does.
type indentWriter struct {
	out io.Writer
	err error
}

// write puts a string out, and remembers the first failure.
func (w *indentWriter) write(text string) {
	if w.err != nil {
		return
	}
	_, w.err = io.WriteString(w.out, text)
}

// writeNode writes one node at the given depth, under the namespace
// declarations in scope around it.
func (w *indentWriter) writeNode(node Node, depth int, namespaces scope) {
	switch node := node.(type) {
	case *ProcessingInstruction:
		w.write("<?" + node.NodeName() + " " + node.Data() + "?>")
	case *Comment:
		w.write("<!--" + node.Data() + "-->")
	case *Text:
		w.write(escapeText(node.Data()))
	case *Element:
		w.writeElement(node, depth, namespaces)
	}
}

// writeElement writes an element and what it holds.
//
// The transformer indents an element that holds only elements, and leaves one
// that holds text alone.
func (w *indentWriter) writeElement(element *Element, depth int, namespaces scope) {
	namespaces = append(namespaces, declaredOn(element))
	missing := missingDeclarations(element, namespaces)
	namespaces[len(namespaces)-1] = mergedWith(namespaces[len(namespaces)-1], missing)

	w.write("<" + element.TagName())
	// A DOM serializer writes an element's namespace declarations before its
	// other attributes, and the ones it had to add itself after both.
	enclosing := namespaces[:len(namespaces)-1]
	for _, declaration := range []bool{true, false} {
		for _, attribute := range element.Attributes() {
			if isDeclaration(attribute) != declaration {
				continue
			}
			if declaration && redundantDeclaration(attribute, enclosing) {
				continue
			}
			w.write(" " + attribute.Name() + "=\"" +
				escapeAttribute(attribute.Value()) + "\"")
		}
	}
	for _, prefix := range sortedKeys(missing) {
		w.write(" " + declarationName(prefix) + "=\"" +
			escapeAttribute(missing[prefix]) + "\"")
	}

	children := element.ChildNodes()
	if len(children) == 0 {
		w.write("/>")
		return
	}
	w.write(">")

	if holdsText(children) {
		for _, child := range children {
			w.writeNode(child, depth+1, namespaces)
		}
		w.write("</" + element.TagName() + ">")
		return
	}

	for _, child := range children {
		w.write("\n" + strings.Repeat("  ", depth+1))
		w.writeNode(child, depth+1, namespaces)
	}
	w.write("\n" + strings.Repeat("  ", depth) + "</" + element.TagName() + ">")
}

// isDeclaration reports whether the attribute is a namespace declaration.
func isDeclaration(attribute *Attr) bool {
	return attribute.Prefix() == XMLNSAttribute ||
		attribute.Prefix() == "" && attribute.LocalName() == XMLNSAttribute
}

// redundantDeclaration reports whether the attribute declares a namespace an
// enclosing element already binds the same prefix to.
//
// A DOM serializer leaves such a declaration out: serializeSchema puts one on
// every rdf:Description, and serializeFields puts the same one on rdf:RDF for
// PDFBOX-2378, so without this every description would carry a declaration its
// parent already made.
func redundantDeclaration(attribute *Attr, enclosing scope) bool {
	switch {
	case attribute.Prefix() == XMLNSAttribute:
		return enclosing.resolve(attribute.LocalName()) == attribute.Value()
	case attribute.Prefix() == "" && attribute.LocalName() == XMLNSAttribute:
		return enclosing.resolve("") == attribute.Value()
	}
	return false
}

// declaredOn reads the namespace declarations an element carries.
func declaredOn(element *Element) map[string]string {
	declared := map[string]string{}
	for _, attribute := range element.Attributes() {
		switch {
		case attribute.Prefix() == XMLNSAttribute:
			declared[attribute.LocalName()] = attribute.Value()
		case attribute.Prefix() == "" && attribute.LocalName() == XMLNSAttribute:
			declared[""] = attribute.Value()
		}
	}
	return declared
}

// missingDeclarations returns the namespace declarations the element and its
// attributes need and no enclosing element made.
//
// This is the namespace fixup a DOM serializer does: an element built with
// createElementNS carries a namespace but no declaration, and the writer has to
// put one out or the name it writes would mean something else.
func missingDeclarations(element *Element, namespaces scope) map[string]string {
	missing := map[string]string{}
	need := func(prefix, namespaceURI string) {
		if namespaceURI == "" || namespaceURI == XMLNSNamespace ||
			namespaceURI == XMLNamespace {
			return
		}
		if namespaces.resolve(prefix) == namespaceURI || missing[prefix] == namespaceURI {
			return
		}
		missing[prefix] = namespaceURI
	}
	need(element.Prefix(), element.NamespaceURI())
	for _, attribute := range element.Attributes() {
		if attribute.Prefix() == "" {
			// An attribute written without a prefix is in no namespace.
			continue
		}
		need(attribute.Prefix(), attribute.NamespaceURI())
	}
	return missing
}

// mergedWith returns the declarations of an element together with the ones the
// writer had to add.
func mergedWith(declared, added map[string]string) map[string]string {
	merged := make(map[string]string, len(declared)+len(added))
	for prefix, namespace := range declared {
		merged[prefix] = namespace
	}
	for prefix, namespace := range added {
		merged[prefix] = namespace
	}
	return merged
}

// declarationName returns how a namespace declaration of the prefix is written.
func declarationName(prefix string) string {
	if prefix == "" {
		return XMLNSAttribute
	}
	return XMLNSAttribute + ":" + prefix
}

// sortedKeys returns a map's keys in order, so that the same document always
// writes the same bytes.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// holdsText reports whether any of the nodes is character data, which stops the
// transformer indenting the element that holds them.
func holdsText(children []Node) bool {
	for _, child := range children {
		if _, isText := child.(*Text); isText {
			return true
		}
	}
	return false
}

// escapeText writes character data.
func escapeText(text string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
}

// escapeAttribute writes an attribute value.
func escapeAttribute(value string) string {
	var escaped strings.Builder
	for _, r := range value {
		switch r {
		case '&':
			escaped.WriteString("&amp;")
		case '<':
			escaped.WriteString("&lt;")
		case '>':
			escaped.WriteString("&gt;")
		case '"':
			escaped.WriteString("&quot;")
		case '\t':
			escaped.WriteString("&#9;")
		case '\n':
			escaped.WriteString("&#10;")
		case '\r':
			escaped.WriteString("&#13;")
		default:
			escaped.WriteRune(r)
		}
	}
	return escaped.String()
}

// errSerialization is what a write failure is reported as.
func errSerialization(cause error) error {
	return fmt.Errorf("%w: %w", ErrXmpSerialization, cause)
}
