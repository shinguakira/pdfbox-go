// Package xml reads and writes the XML an XMP packet is written in.
//
// Port of org.apache.xmpbox.xml.
//
// Java reaches the XML through org.w3c.dom: DocumentBuilderFactory parses into
// a namespace-aware Document, and a Transformer writes one back out. Go has no
// DOM in its standard library, so this file is one: the node kinds the port
// uses, a parser that fills them, and, in serializer.go, a writer that puts
// them back. Everything the two helpers and the parser ask of a node is here
// and nothing else is.
//
// The parser is built on encoding/xml's RawToken, which reports names as they
// were written -- prefix and all -- and leaves namespace resolution to the
// caller. That is what the port needs: DomXmpParser reads an element's prefix
// as often as it reads its namespace. Where this differs from Xerces the
// difference is recorded in migration/STATUS.md.
//
// go/w3c/dom is a second DOM, and the two do not fold together. That one is the
// subset PDFBox reads XFDF through: read only, because XFDF is written out by
// hand with a Writer, and it resolves a prefix away when it is namespace aware,
// because that is what the FDF reading wants. This one has to build a document
// to serialize it, and has to keep the prefix on every name. Each is a faithful
// port of what its own Java call site asks for, and widening either to cover
// both would make it a port of neither.
package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// The two namespaces XML gives a fixed meaning.
//
// Port of the XMLConstants fields the module uses.
const (
	// XMLNamespace is the namespace of the xml: prefix, which needs no
	// declaration.
	XMLNamespace = "http://www.w3.org/XML/1998/namespace"

	// XMLNSNamespace is the namespace of a namespace declaration.
	XMLNSNamespace = "http://www.w3.org/2000/xmlns/"

	// XMLNSAttribute is the name a namespace declaration is written with.
	XMLNSAttribute = "xmlns"

	// XMLNSPrefix is the prefix the xml namespace is bound to.
	XMLNSPrefix = "xml"
)

// Node is one node of a document.
//
// Port of org.w3c.dom.Node, narrowed to what the module asks of it: the type of
// a node, which Java tests with instanceof.
type Node interface {
	// TextContent returns the text below this node.
	TextContent() string

	isNode()
}

// Document is a parsed or built document: the processing instructions and the
// one root element it holds.
//
// Port of org.w3c.dom.Document.
type Document struct {
	children []Node
}

// Element is an element: its name, its attributes and what it contains.
//
// Port of org.w3c.dom.Element.
type Element struct {
	// namespaceURI is the namespace the name resolved to, and the empty
	// string where the name is in no namespace -- which is Java's null.
	namespaceURI string

	// prefix is the prefix the name was written with, and the empty string
	// where it was written without one -- which is Java's null.
	prefix string

	// localName is the part of the name after the prefix.
	localName string

	attributes []*Attr
	children   []Node
}

// Attr is one attribute of an element.
//
// Port of org.w3c.dom.Attr.
type Attr struct {
	namespaceURI string
	prefix       string
	localName    string
	value        string
}

// Text is character data.
//
// Port of org.w3c.dom.Text.
type Text struct{ data string }

// Comment is a comment.
//
// Port of org.w3c.dom.Comment.
type Comment struct{ data string }

// ProcessingInstruction is a processing instruction.
//
// Port of org.w3c.dom.ProcessingInstruction.
type ProcessingInstruction struct {
	target string
	data   string
}

func (*Document) isNode()              {}
func (*Element) isNode()               {}
func (*Text) isNode()                  {}
func (*Comment) isNode()               {}
func (*ProcessingInstruction) isNode() {}

// NewDocument returns an empty document.
//
// Port of DocumentBuilder.newDocument().
func NewDocument() *Document { return &Document{} }

// ChildNodes returns the nodes this document holds.
func (d *Document) ChildNodes() []Node { return d.children }

// FirstChild returns the first node this document holds, and nil where it holds
// none.
func (d *Document) FirstChild() Node {
	if len(d.children) == 0 {
		return nil
	}
	return d.children[0]
}

// AppendChild adds a node to the end of this document.
func (d *Document) AppendChild(child Node) { d.children = append(d.children, child) }

// RemoveChild takes a node out of this document.
func (d *Document) RemoveChild(child Node) { d.children = removeNode(d.children, child) }

// TextContent returns the text below this document.
func (d *Document) TextContent() string { return textOf(d.children) }

// DocumentElement returns the root element, and nil where there is none.
func (d *Document) DocumentElement() *Element {
	for _, child := range d.children {
		if element, isElement := child.(*Element); isElement {
			return element
		}
	}
	return nil
}

// CreateElement returns an element of the given qualified name in no namespace.
//
// Port of Document.createElement(String), which the serializer uses for every
// element whose namespace is already declared on an ancestor.
func (d *Document) CreateElement(qualifiedName string) *Element {
	prefix, localName := splitQName(qualifiedName)
	return &Element{prefix: prefix, localName: localName}
}

// CreateElementNS returns an element of the given namespace and qualified name.
//
// Port of Document.createElementNS(String, String).
func (d *Document) CreateElementNS(namespaceURI, qualifiedName string) *Element {
	element := d.CreateElement(qualifiedName)
	element.namespaceURI = namespaceURI
	return element
}

// CreateProcessingInstruction returns a processing instruction.
//
// Port of Document.createProcessingInstruction(String, String).
func (d *Document) CreateProcessingInstruction(target, data string) *ProcessingInstruction {
	return &ProcessingInstruction{target: target, data: data}
}

// NamespaceURI returns the namespace of this element's name, and the empty
// string where it is in none.
func (e *Element) NamespaceURI() string { return e.namespaceURI }

// Prefix returns the prefix this element's name was written with, and the empty
// string where it was written without one.
func (e *Element) Prefix() string { return e.prefix }

// LocalName returns the part of this element's name after the prefix.
func (e *Element) LocalName() string { return e.localName }

// TagName returns this element's name as it was written.
//
// Port of Element.getTagName().
func (e *Element) TagName() string { return qualify(e.prefix, e.localName) }

// SetPrefix sets the prefix this element's name is written with.
func (e *Element) SetPrefix(prefix string) { e.prefix = prefix }

// Attributes returns this element's attributes, in the order they were written
// or set.
//
// Port of Element.getAttributes(), whose NamedNodeMap the module only ever
// walks by index.
func (e *Element) Attributes() []*Attr { return e.attributes }

// AttributeNodeNS returns the attribute of the given namespace and local name,
// and nil where the element has none.
//
// Port of Element.getAttributeNodeNS(String, String).
func (e *Element) AttributeNodeNS(namespaceURI, localName string) *Attr {
	for _, attribute := range e.attributes {
		if attribute.namespaceURI == namespaceURI && attribute.localName == localName {
			return attribute
		}
	}
	return nil
}

// SetAttribute sets an attribute of the given qualified name in no namespace.
//
// Port of Element.setAttribute(String, String), which does not resolve the
// prefix: the name it stores is the whole of what it was given, and it finds an
// attribute already there by that whole name -- so a declaration set with
// setAttributeNS is replaced rather than written twice.
func (e *Element) SetAttribute(qualifiedName, value string) {
	for _, attribute := range e.attributes {
		if attribute.Name() == qualifiedName {
			attribute.value = value
			return
		}
	}
	prefix, localName := splitQName(qualifiedName)
	e.addAttribute(&Attr{prefix: prefix, localName: localName, value: value})
}

// SetAttributeNS sets an attribute of the given namespace and qualified name.
//
// Port of Element.setAttributeNS(String, String, String), which finds an
// attribute already there by its namespace and local name.
func (e *Element) SetAttributeNS(namespaceURI, qualifiedName, value string) {
	prefix, localName := splitQName(qualifiedName)
	for _, attribute := range e.attributes {
		if attribute.namespaceURI == namespaceURI && attribute.localName == localName {
			attribute.prefix = prefix
			attribute.value = value
			return
		}
	}
	e.addAttribute(&Attr{
		namespaceURI: namespaceURI,
		prefix:       prefix,
		localName:    localName,
		value:        value,
	})
}

// addAttribute puts an attribute in the list, in order of its written name.
//
// Xerces keeps an element's attributes in a NamedNodeMap it searches by name
// with a binary search, so they are held -- and walked, and written out -- in
// that order rather than the order they were set or read in. Both the parser
// and the serializer walk this list, so the order belongs here.
func (e *Element) addAttribute(attribute *Attr) {
	name := attribute.Name()
	at := len(e.attributes)
	for i, held := range e.attributes {
		if held.Name() > name {
			at = i
			break
		}
	}
	e.attributes = append(e.attributes, nil)
	copy(e.attributes[at+1:], e.attributes[at:])
	e.attributes[at] = attribute
}

// ChildNodes returns the nodes this element holds.
func (e *Element) ChildNodes() []Node { return e.children }

// FirstChild returns the first node this element holds, and nil where it holds
// none.
func (e *Element) FirstChild() Node {
	if len(e.children) == 0 {
		return nil
	}
	return e.children[0]
}

// AppendChild adds a node to the end of this element.
func (e *Element) AppendChild(child Node) { e.children = append(e.children, child) }

// RemoveChild takes a node out of this element.
func (e *Element) RemoveChild(child Node) { e.children = removeNode(e.children, child) }

// TextContent returns the text below this element.
func (e *Element) TextContent() string { return textOf(e.children) }

// SetTextContent replaces what this element holds with one text node.
//
// Port of Node.setTextContent(String), which removes the children and adds the
// text node only where the string is not empty -- so an element set to the
// empty string holds nothing and is written as an empty tag.
func (e *Element) SetTextContent(text string) {
	e.children = nil
	if text != "" {
		e.children = []Node{&Text{data: text}}
	}
}

// String returns what Java's Element.toString would put in a message.
//
// Xerces answers the tag name in brackets, and the parser puts elements into
// its messages with string concatenation.
func (e *Element) String() string { return "[" + e.TagName() + ": null]" }

// NamespaceURI returns the namespace of this attribute's name, and the empty
// string where it is in none.
//
// An attribute written without a prefix is in no namespace: unlike an element,
// it does not take the default one.
func (a *Attr) NamespaceURI() string { return a.namespaceURI }

// Prefix returns the prefix this attribute's name was written with, and the
// empty string where it was written without one.
func (a *Attr) Prefix() string { return a.prefix }

// LocalName returns the part of this attribute's name after the prefix.
func (a *Attr) LocalName() string { return a.localName }

// Name returns this attribute's name as it was written.
func (a *Attr) Name() string { return qualify(a.prefix, a.localName) }

// Value returns this attribute's value.
func (a *Attr) Value() string { return a.value }

// Data returns the characters.
func (t *Text) Data() string { return t.data }

// TextContent returns the characters.
func (t *Text) TextContent() string { return t.data }

// Data returns the comment.
func (c *Comment) Data() string { return c.data }

// TextContent returns the comment.
func (c *Comment) TextContent() string { return c.data }

// NodeName returns the target of the processing instruction, which is what
// Java's getNodeName answers for one.
func (p *ProcessingInstruction) NodeName() string { return p.target }

// Data returns the instruction after the target.
func (p *ProcessingInstruction) Data() string { return p.data }

// TextContent returns the instruction after the target.
func (p *ProcessingInstruction) TextContent() string { return p.data }

// textOf is Node.getTextContent for a node with children: the text of every
// descendant, run together, with comments and processing instructions left out.
func textOf(children []Node) string {
	var text strings.Builder
	for _, child := range children {
		switch child := child.(type) {
		case *Text:
			text.WriteString(child.data)
		case *Element:
			text.WriteString(child.TextContent())
		}
	}
	return text.String()
}

// removeNode takes the first identical node out of the list.
func removeNode(children []Node, child Node) []Node {
	for i, declared := range children {
		if declared == child {
			return append(children[:i], children[i+1:]...)
		}
	}
	return children
}

// splitQName splits a written name into its prefix and its local part.
func splitQName(qualifiedName string) (prefix, localName string) {
	if colon := strings.IndexByte(qualifiedName, ':'); colon >= 0 {
		return qualifiedName[:colon], qualifiedName[colon+1:]
	}
	return "", qualifiedName
}

// qualify writes a prefix and a local part back into a name.
func qualify(prefix, localName string) string {
	if prefix == "" {
		return localName
	}
	return prefix + ":" + localName
}

// scope is the namespace declarations in force, innermost last.
type scope []map[string]string

// resolve returns the namespace a prefix is bound to, and the empty string
// where it is bound to none.
func (s scope) resolve(prefix string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if namespace, bound := s[i][prefix]; bound {
			return namespace
		}
	}
	if prefix == XMLNSPrefix {
		return XMLNamespace
	}
	return ""
}

// Parse reads a document.
//
// Port of DocumentBuilder.parse(InputStream) as DomXmpParser configures it: a
// namespace-aware builder that keeps comments, expands no entity but the five
// XML declares, and refuses a document type declaration.
func Parse(input io.Reader) (*Document, error) {
	decoder := xml.NewDecoder(input)
	decoder.Strict = true

	document := NewDocument()
	// open is the elements the parser is inside, innermost last, and
	// namespaces is what each of them declared.
	var open []*Element
	var namespaces scope

	// first says whether nothing has been read yet, which is the only place
	// an XML declaration may appear.
	first := true

	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch token := token.(type) {
		case xml.StartElement:
			declared := declarationsOf(token.Attr)
			namespaces = append(namespaces, declared)

			element := &Element{
				prefix:       token.Name.Space,
				localName:    token.Name.Local,
				namespaceURI: namespaces.resolve(token.Name.Space),
			}
			for _, attribute := range token.Attr {
				element.addAttribute(attributeOf(attribute, namespaces))
			}
			appendTo(document, open, element)
			open = append(open, element)

		case xml.EndElement:
			if len(open) == 0 {
				return nil, fmt.Errorf("unexpected end element </%s>",
					qualify(token.Name.Space, token.Name.Local))
			}
			innermost := open[len(open)-1]
			if innermost.prefix != token.Name.Space ||
				innermost.localName != token.Name.Local {
				return nil, fmt.Errorf("element <%s> closed by </%s>",
					innermost.TagName(),
					qualify(token.Name.Space, token.Name.Local))
			}
			open = open[:len(open)-1]
			namespaces = namespaces[:len(namespaces)-1]

		case xml.CharData:
			appendTo(document, open, &Text{data: string(token)})

		case xml.Comment:
			appendTo(document, open, &Comment{data: string(token)})

		case xml.ProcInst:
			// The XML declaration is not a processing instruction, and a
			// DOM does not hold one.
			if first && token.Target == "xml" {
				break
			}
			appendTo(document, open,
				&ProcessingInstruction{target: token.Target, data: string(token.Inst)})

		case xml.Directive:
			directive := strings.TrimSpace(string(token))
			if strings.HasPrefix(directive, "DOCTYPE") {
				return nil, fmt.Errorf(
					"DOCTYPE is disallowed when the feature " +
						"\"http://apache.org/xml/features/disallow-doctype-decl\" " +
						"set to true.")
			}
		}
		first = false
	}

	if len(open) != 0 {
		return nil, fmt.Errorf("element <%s> is not closed", open[len(open)-1].TagName())
	}
	if document.DocumentElement() == nil {
		return nil, fmt.Errorf("premature end of file")
	}
	return document, nil
}

// declarationsOf reads the namespace declarations an element carries.
//
// encoding/xml reports xmlns:p as a name whose space is "xmlns", and xmlns as a
// name with no space and the local part "xmlns".
func declarationsOf(attributes []xml.Attr) map[string]string {
	declared := map[string]string{}
	for _, attribute := range attributes {
		switch {
		case attribute.Name.Space == XMLNSAttribute:
			declared[attribute.Name.Local] = attribute.Value
		case attribute.Name.Space == "" && attribute.Name.Local == XMLNSAttribute:
			declared[""] = attribute.Value
		}
	}
	return declared
}

// attributeOf turns a written attribute into a node of the tree.
func attributeOf(attribute xml.Attr, namespaces scope) *Attr {
	prefix, localName := attribute.Name.Space, attribute.Name.Local
	namespaceURI := ""
	switch {
	case prefix == XMLNSAttribute:
		namespaceURI = XMLNSNamespace
	case prefix == "" && localName == XMLNSAttribute:
		namespaceURI = XMLNSNamespace
	case prefix != "":
		namespaceURI = namespaces.resolve(prefix)
	}
	return &Attr{
		namespaceURI: namespaceURI,
		prefix:       prefix,
		localName:    localName,
		value:        attribute.Value,
	}
}

// appendTo adds a node to the innermost open element, or to the document where
// there is none.
func appendTo(document *Document, open []*Element, child Node) {
	if len(open) == 0 {
		document.AppendChild(child)
		return
	}
	open[len(open)-1].AppendChild(child)
}
