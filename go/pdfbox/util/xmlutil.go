package util

import (
	"io"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/w3c/dom"
)

// XMLParse reads the given XML into a DOM tree, with the parser not namespace
// aware.
//
// Port of the static XMLUtil.parse(InputStream). Java declares the class final
// with a private constructor and everything static; Go writes those as package
// functions, and the names carry the XML prefix because this package holds more
// than one utility class.
func XMLParse(is io.Reader) (*dom.Document, error) { return XMLParseNamespaceAware(is, false) }

// XMLParseNamespaceAware reads the given XML into a DOM tree.
//
// Port of the static XMLUtil.parse(InputStream, boolean). The parser of Java has
// every doorway to an external entity shut: DOCTYPE declarations are disallowed,
// general and parameter entities are not read, no external DTD is loaded, and
// XInclude is off. The port shuts the same doors; see w3c/dom.Parse.
func XMLParseNamespaceAware(is io.Reader, nsAware bool) (*dom.Document, error) {
	return dom.Parse(is, nsAware)
}

// XMLNodeValue returns the text of the children of the given element, with
// everything that is not character data left out.
//
// Port of the static XMLUtil.getNodeValue(Element).
func XMLNodeValue(node *dom.Element) string {
	sb := &strings.Builder{}
	children := node.ChildNodes()
	numNodes := children.Length()
	for i := 0; i < numNodes; i++ {
		next := children.Item(i)
		if text, isText := next.(*dom.Text); isText {
			sb.WriteString(text.NodeValue())
		}
	}
	return sb.String()
}
