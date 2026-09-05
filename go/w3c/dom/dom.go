// Package dom is the small part of the W3C DOM that PDFBox reads XML through.
//
// Port of the org.w3c.dom interfaces the port needs: Node, NodeList,
// NamedNodeMap, Document, Element, Attr, Text and CDATASection. It is a reading
// DOM only, which is all PDFBox uses one for: XFDF is written out by hand with
// a Writer. The pieces that are not needed -- entities, notations, processing
// instructions, namespaces beyond the parsed name, mutation, ranges, traversal
// -- are not here.
package dom

// NodeType is what kind of node a node is.
//
// Port of the nodeType constants of org.w3c.dom.Node. Only the kinds this DOM
// builds are named.
type NodeType int

// The node kinds this DOM builds.
const (
	ElementNode               NodeType = 1
	AttributeNode             NodeType = 2
	TextNode                  NodeType = 3
	CDATASectionNode          NodeType = 4
	ProcessingInstructionNode NodeType = 7
	CommentNode               NodeType = 8
	DocumentNode              NodeType = 9
)

// Node is one node of the tree.
//
// Port of the interface org.w3c.dom.Node.
type Node interface {
	// NodeName returns the name of the node: the tag name of an element, the
	// name of an attribute, and "#text", "#cdata-section", "#comment" or
	// "#document" for the rest.
	NodeName() string

	// NodeValue returns the value of the node: the text of a text or CDATA
	// node, the value of an attribute, and the empty string for an element or
	// the document, where Java answers null.
	NodeValue() string

	// NodeType returns what kind of node this is.
	NodeType() NodeType

	// ChildNodes returns the children of the node, which is empty where it has
	// none.
	ChildNodes() NodeList

	// FirstChild returns the first child of the node, or nil where it has
	// none.
	FirstChild() Node

	// Attributes returns the attributes of an element, and nil for every other
	// kind of node, which is what Java answers.
	Attributes() NamedNodeMap

	// OwnerDocument returns the document the node belongs to, and nil for the
	// document itself.
	OwnerDocument() *Document
}

// NodeList is an ordered collection of nodes.
//
// Port of the interface org.w3c.dom.NodeList.
type NodeList []Node

// Length returns how many nodes the list holds.
func (l NodeList) Length() int { return len(l) }

// Item returns the node at the given index, or nil where the index is out of
// range, which is what Java answers.
func (l NodeList) Item(index int) Node {
	if index < 0 || index >= len(l) {
		return nil
	}
	return l[index]
}

// NamedNodeMap is a collection of nodes reachable by name.
//
// Port of the interface org.w3c.dom.NamedNodeMap. It holds the attributes of an
// element, in the order they were written, which is the order Xerces keeps them
// in too.
type NamedNodeMap []*Attr

// Length returns how many nodes the map holds.
func (m NamedNodeMap) Length() int { return len(m) }

// Item returns the node at the given index, or nil where the index is out of
// range.
func (m NamedNodeMap) Item(index int) Node {
	if index < 0 || index >= len(m) {
		return nil
	}
	return m[index]
}

// GetNamedItem returns the node with the given name, or nil where the map holds
// none.
func (m NamedNodeMap) GetNamedItem(name string) Node {
	for _, attr := range m {
		if attr.name == name {
			return attr
		}
	}
	return nil
}

// node is the state every node shares. Java puts it in the abstract NodeImpl of
// Xerces; the concrete nodes here embed it.
type node struct {
	children NodeList
	owner    *Document
}

// ChildNodes returns the children of the node.
func (n *node) ChildNodes() NodeList { return n.children }

// FirstChild returns the first child of the node, or nil where it has none.
func (n *node) FirstChild() Node {
	if len(n.children) == 0 {
		return nil
	}
	return n.children[0]
}

// OwnerDocument returns the document the node belongs to.
func (n *node) OwnerDocument() *Document { return n.owner }

// Attributes returns nil: only an element has attributes.
func (n *node) Attributes() NamedNodeMap { return nil }

// Document is the root of the tree.
//
// Port of the interface org.w3c.dom.Document.
type Document struct {
	node
	xmlEncoding   string
	inputEncoding string
}

var _ Node = (*Document)(nil)

// NodeName returns "#document".
func (d *Document) NodeName() string { return "#document" }

// NodeValue returns the empty string, which is the null Java answers for a
// document.
func (d *Document) NodeValue() string { return "" }

// NodeType returns DocumentNode.
func (d *Document) NodeType() NodeType { return DocumentNode }

// OwnerDocument returns nil: the document belongs to no other document.
func (d *Document) OwnerDocument() *Document { return nil }

// DocumentElement returns the root element, or nil where the document has none.
func (d *Document) DocumentElement() *Element {
	for _, child := range d.children {
		if element, isElement := child.(*Element); isElement {
			return element
		}
	}
	return nil
}

// XmlEncoding returns the encoding the XML declaration named, or the empty
// string where it named none, which is the null Java answers.
func (d *Document) XmlEncoding() string { return d.xmlEncoding }

// InputEncoding returns the encoding the document was actually read in, or the
// empty string where the parser could not say.
func (d *Document) InputEncoding() string { return d.inputEncoding }

// Element is one element of the tree.
//
// Port of the interface org.w3c.dom.Element.
type Element struct {
	node
	tagName    string
	attributes NamedNodeMap
}

var _ Node = (*Element)(nil)

// NodeName returns the tag name.
func (e *Element) NodeName() string { return e.tagName }

// NodeValue returns the empty string, which is the null Java answers for an
// element.
func (e *Element) NodeValue() string { return "" }

// NodeType returns ElementNode.
func (e *Element) NodeType() NodeType { return ElementNode }

// TagName returns the name of the element.
func (e *Element) TagName() string { return e.tagName }

// Attributes returns the attributes of the element.
func (e *Element) Attributes() NamedNodeMap { return e.attributes }

// GetAttribute returns the value of the named attribute, or the empty string
// where the element has none -- which is what Java answers here, rather than
// null.
func (e *Element) GetAttribute(name string) string {
	if attr := e.attributes.GetNamedItem(name); attr != nil {
		return attr.NodeValue()
	}
	return ""
}

// HasAttribute reports whether the element has the named attribute.
func (e *Element) HasAttribute(name string) bool {
	return e.attributes.GetNamedItem(name) != nil
}

// Attr is one attribute of an element.
//
// Port of the interface org.w3c.dom.Attr.
type Attr struct {
	node
	name  string
	value string
}

var _ Node = (*Attr)(nil)

// NodeName returns the name of the attribute.
func (a *Attr) NodeName() string { return a.name }

// NodeValue returns the value of the attribute.
func (a *Attr) NodeValue() string { return a.value }

// NodeType returns AttributeNode.
func (a *Attr) NodeType() NodeType { return AttributeNode }

// Name returns the name of the attribute.
func (a *Attr) Name() string { return a.name }

// Value returns the value of the attribute.
func (a *Attr) Value() string { return a.value }

// Text is a run of character data.
//
// Port of the interfaces org.w3c.dom.Text and org.w3c.dom.CharacterData.
type Text struct {
	node
	data     string
	isCDATA  bool
	nodeName string
}

var _ Node = (*Text)(nil)

// NodeName returns "#text", or "#cdata-section" for a CDATA section.
func (t *Text) NodeName() string { return t.nodeName }

// NodeValue returns the character data.
func (t *Text) NodeValue() string { return t.data }

// NodeType returns TextNode, or CDATASectionNode for a CDATA section.
func (t *Text) NodeType() NodeType {
	if t.isCDATA {
		return CDATASectionNode
	}
	return TextNode
}

// Data returns the character data.
func (t *Text) Data() string { return t.data }

// IsCDATASection reports whether this run was written as a CDATA section.
//
// Java asks `instanceof CDATASection`, which Go cannot, because a CDATA section
// and a text node are the same struct here: the parser reports them as the same
// token kind and only says which it was.
func (t *Text) IsCDATASection() bool { return t.isCDATA }

// CDATASection returns t where it is a CDATA section, and nil otherwise, which
// is the `instanceof CDATASection` of Java written as a cast.
func (t *Text) CDATASection() *Text {
	if t.isCDATA {
		return t
	}
	return nil
}

// Comment is a comment node, which the FDF reading skips over.
//
// Port of the interface org.w3c.dom.Comment.
type Comment struct {
	node
	data string
}

var _ Node = (*Comment)(nil)

// NodeName returns "#comment".
func (c *Comment) NodeName() string { return "#comment" }

// NodeValue returns the text of the comment.
func (c *Comment) NodeValue() string { return c.data }

// NodeType returns CommentNode.
func (c *Comment) NodeType() NodeType { return CommentNode }

// Data returns the text of the comment.
func (c *Comment) Data() string { return c.data }

// ProcessingInstruction is a processing instruction node.
//
// Port of the interface org.w3c.dom.ProcessingInstruction.
type ProcessingInstruction struct {
	node
	target string
	data   string
}

var _ Node = (*ProcessingInstruction)(nil)

// NodeName returns the target of the instruction.
func (p *ProcessingInstruction) NodeName() string { return p.target }

// NodeValue returns the data of the instruction.
func (p *ProcessingInstruction) NodeValue() string { return p.data }

// NodeType returns ProcessingInstructionNode.
func (p *ProcessingInstruction) NodeType() NodeType { return ProcessingInstructionNode }

// Target returns the target of the instruction.
func (p *ProcessingInstruction) Target() string { return p.target }

// Data returns the data of the instruction.
func (p *ProcessingInstruction) Data() string { return p.data }

// TextContent returns the text of the node and of everything below it, with
// every element and attribute left out.
//
// Port of Node.getTextContent. Java puts it on the interface; Go writes it as a
// function over one, because the nodes here share no base struct that could
// carry it.
func TextContent(n Node) string {
	switch typed := n.(type) {
	case *Text:
		return typed.data
	case *Comment, *ProcessingInstruction:
		// getTextContent answers null for a document, a document type and a
		// notation, and the data itself for a comment or a processing
		// instruction.
		return n.NodeValue()
	case *Attr:
		return typed.value
	}
	out := ""
	for _, child := range n.ChildNodes() {
		switch child.(type) {
		case *Comment, *ProcessingInstruction:
			// getTextContent leaves comments and processing instructions out
			// when it walks the children of an element or a document.
			continue
		}
		out += TextContent(child)
	}
	return out
}

// FirstElementByTagName returns the first child element of the given node with
// the given name, or nil where it has none.
//
// The XPath expressions PDFBox evaluates over an XFDF annotation are
// "contents[1]" and "contents-richtext[1]", both of which are this. Go has no
// XPath engine in its standard library, and the two steps are the whole of what
// PDFBox asks for, so the port answers them directly.
func FirstElementByTagName(n Node, tagName string) *Element {
	for _, child := range n.ChildNodes() {
		if element, isElement := child.(*Element); isElement && element.tagName == tagName {
			return element
		}
	}
	return nil
}

// ElementsByPath returns every element the given path of child element names
// reaches from the given node, in document order.
//
// The XPath expressions PDFBox evaluates as node sets are "inklist/gesture" and
// "OnActivation/Action/URI", both of which are this. Go has no XPath engine in
// its standard library, and these two steps are the whole of what PDFBox asks
// for, so the port answers them directly.
func ElementsByPath(n Node, path ...string) []*Element {
	current := []Node{n}
	for _, name := range path {
		next := []Node{}
		for _, node := range current {
			for _, child := range node.ChildNodes() {
				if element, isElement := child.(*Element); isElement && element.tagName == name {
					next = append(next, element)
				}
			}
		}
		current = next
	}
	out := make([]*Element, 0, len(current))
	for _, node := range current {
		out = append(out, node.(*Element))
	}
	return out
}
