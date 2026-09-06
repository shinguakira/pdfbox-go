package dom

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrDoctypeNotAllowed is what Parse reports for a document holding a DOCTYPE
// declaration.
//
// XMLUtil turns "http://apache.org/xml/features/disallow-doctype-decl" on, so
// Xerces rejects such a document; the port rejects it here.
var ErrDoctypeNotAllowed = errors.New("dom: DOCTYPE is disallowed")

// cdataOpen is what a CDATA section starts with in the source.
var cdataOpen = []byte("<![CDATA[")

// Parse reads the given XML into a tree.
//
// Java builds a DocumentBuilder with every external entity turned off, and so
// does this: encoding/xml resolves no external entity of its own, and a DOCTYPE
// declaration is rejected outright, which is the disallow-doctype-decl of
// XMLUtil.
//
// nsAware says whether the parser is namespace aware, as the builder factory
// flag of Java does. With it off a node keeps the qualified name it was written
// with, prefix and all, which is what the FDF reading matches against; with it
// on the prefix is resolved and the node keeps its local name.
//
// The whole input is read into memory first. The parser of Go reports a CDATA
// section as ordinary character data, so the only way to tell the two apart --
// which the rich contents of an FDF annotation needs -- is to look at the bytes
// the token was read from.
func Parse(r io.Reader, nsAware bool) (*Document, error) {
	source, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(bytes.NewReader(source))
	decoder.Strict = true

	document := &Document{}
	stack := []Node{Node(document)}
	// namespaces holds the prefix to URI bindings in scope, innermost last.
	namespaces := []map[string]string{}

	appendChild := func(child Node) {
		switch typed := stack[len(stack)-1].(type) {
		case *Document:
			typed.children = append(typed.children, child)
		case *Element:
			typed.children = append(typed.children, child)
		}
	}

	for {
		start := decoder.InputOffset()
		// RawToken does not resolve namespaces, so a name keeps the prefix it
		// was written with, which is what a namespace-unaware Xerces answers.
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch typed := token.(type) {
		case xml.StartElement:
			bindings := map[string]string{}
			for _, attr := range typed.Attr {
				switch {
				case attr.Name.Space == "" && attr.Name.Local == "xmlns":
					bindings[""] = attr.Value
				case attr.Name.Space == "xmlns":
					bindings[attr.Name.Local] = attr.Value
				}
			}
			namespaces = append(namespaces, bindings)

			element := &Element{
				node:    node{owner: document},
				tagName: nodeName(typed.Name, nsAware),
			}
			for _, attr := range typed.Attr {
				element.attributes = append(element.attributes, &Attr{
					node:  node{owner: document},
					name:  nodeName(attr.Name, nsAware),
					value: attr.Value,
				})
			}
			appendChild(element)
			stack = append(stack, element)
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
			if len(namespaces) > 0 {
				namespaces = namespaces[:len(namespaces)-1]
			}
		case xml.CharData:
			text := &Text{
				node:     node{owner: document},
				data:     string(typed),
				nodeName: "#text",
			}
			if isCDATASection(source, start) {
				text.isCDATA = true
				text.nodeName = "#cdata-section"
			}
			appendChild(text)
		case xml.Comment:
			appendChild(&Comment{node: node{owner: document}, data: string(typed)})
		case xml.ProcInst:
			if typed.Target == "xml" {
				document.xmlEncoding = declaredEncoding(string(typed.Inst))
				continue
			}
			appendChild(&ProcessingInstruction{
				node:   node{owner: document},
				target: typed.Target,
				data:   string(typed.Inst),
			})
		case xml.Directive:
			if isDoctype(string(typed)) {
				return nil, ErrDoctypeNotAllowed
			}
		}
	}
	// Xerces answers the encoding it actually decoded in; encoding/xml decodes
	// UTF-8 unless a CharsetReader is given, and none is.
	document.inputEncoding = "UTF-8"
	return document, nil
}

// NewCDATASection returns a CDATA section node.
func NewCDATASection(owner *Document, data string) *Text {
	return &Text{node: node{owner: owner}, data: data, isCDATA: true, nodeName: "#cdata-section"}
}

// NewText returns a text node.
func NewText(owner *Document, data string) *Text {
	return &Text{node: node{owner: owner}, data: data, nodeName: "#text"}
}

// nodeName returns the name of a node the way the parser of Java would.
//
// RawToken leaves the prefix in Space, so the qualified name is prefix:local,
// which is what a namespace-unaware parser answers. A namespace-aware one
// answers the local name.
func nodeName(name xml.Name, nsAware bool) string {
	if nsAware || name.Space == "" {
		return name.Local
	}
	return name.Space + ":" + name.Local
}

// isCDATASection reports whether the character data token that starts at the
// given offset of the source was written as a CDATA section.
func isCDATASection(source []byte, offset int64) bool {
	if offset < 0 || offset >= int64(len(source)) {
		return false
	}
	return bytes.HasPrefix(source[offset:], cdataOpen)
}

// declaredEncoding pulls the encoding out of an XML declaration, and answers the
// empty string where it names none.
func declaredEncoding(inst string) string {
	const key = "encoding="
	index := strings.Index(inst, key)
	if index == -1 {
		return ""
	}
	rest := inst[index+len(key):]
	if rest == "" {
		return ""
	}
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return ""
	}
	end := strings.IndexByte(rest[1:], quote)
	if end == -1 {
		return ""
	}
	return rest[1 : 1+end]
}

// isDoctype reports whether the directive is a DOCTYPE declaration.
func isDoctype(directive string) bool {
	return strings.HasPrefix(strings.TrimSpace(directive), "DOCTYPE")
}

// String renders the element the way a debugger would, which Java gets from
// Object.toString and never relies on.
func (e *Element) String() string { return fmt.Sprintf("<%s>", e.tagName) }
