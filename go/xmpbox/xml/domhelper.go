package xml

// The four questions the parser asks of an element.
//
// Port of DomHelper.

import (
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// UniqueElementChild returns the one element child, and reports a problem where
// there are two.
//
// Java walks the child nodes and remembers the position of the one element it
// found; where it found none the position stays -1 and NodeList.item(-1)
// answers null, so an element with no element child answers null rather than
// failing. Ported as written.
//
// Port of DomHelper.getUniqueElementChild(Element).
func UniqueElementChild(description *Element) (*Element, error) {
	nodes := description.ChildNodes()
	pos := -1
	for i, node := range nodes {
		if _, isElement := node.(*Element); isElement {
			if pos >= 0 {
				// invalid : found two child elements
				return nil, NewXmpParsingError(Undefined,
					"Found two child elements in "+description.String())
			}
			pos = i
		}
	}
	if pos < 0 {
		return nil, nil
	}
	return nodes[pos].(*Element), nil
}

// FirstChildElement returns the first element child, and nil where there is
// none.
//
// Port of DomHelper.getFirstChildElement(Element).
func FirstChildElement(description *Element) *Element {
	for _, node := range description.ChildNodes() {
		if element, isElement := node.(*Element); isElement {
			return element
		}
	}
	return nil
}

// ElementChildren returns the element children.
//
// Port of DomHelper.getElementChildren(Element).
func ElementChildren(description *Element) []*Element {
	nodes := description.ChildNodes()
	children := make([]*Element, 0, len(nodes))
	for _, node := range nodes {
		if element, isElement := node.(*Element); isElement {
			children = append(children, element)
		}
	}
	return children
}

// QNameOf returns the element's qualified name.
//
// Port of DomHelper.getQName(Element). PDFBOX-5835: Java builds the two-argument
// QName where the element has no prefix, because the three-argument one refuses
// a null prefix; the port's QName holds the empty string for both.
func QNameOf(element *Element) xmptype.QName {
	return xmptype.QName{
		NamespaceURI: element.NamespaceURI(),
		LocalPart:    element.LocalName(),
		Prefix:       element.Prefix(),
	}
}

// IsParseTypeResource reports whether the element is written as a resource.
//
// Port of DomHelper.isParseTypeResource(Element).
func IsParseTypeResource(element *Element) bool {
	parseType := element.AttributeNodeNS(xmptype.RDFNamespace, xmptype.ParseType)
	return parseType != nil && xmptype.ResourceName == parseType.Value()
}
