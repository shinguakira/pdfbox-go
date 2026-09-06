// Package xmpbox reads and writes XMP metadata.
//
// Port of org.apache.xmpbox. PDFBox hands back a raw metadata stream and this
// module parses it separately; migration/PLAN.md gives it a parallel track of
// its own, because it depends on nothing else in the build.
//
// Two of the module's root classes are declared in xmpbox/xmptype rather than
// here: XmpConstants and DateConverter. Java's root package holds XMPMetadata,
// which every field points back at, and xmptype would have to import this
// package for them; Go forbids the cycle. Both are aliased below, so the Java
// names sit in the Java place.
package xmpbox

import (
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// XMPMetadata is an XMP packet: the schemas it carries and the xpacket
// processing instruction that wraps them.
//
// Port of org.apache.xmpbox.XMPMetadata.
type XMPMetadata struct {
	xpacketID       string
	xpacketBegin    string
	xpacketBytes    *string
	xpacketEncoding string
	xpacketEndData  string

	schemas     []schema.Schema
	typeMapping *xmptype.TypeMapping
}

var _ xmptype.MetadataLike = (*XMPMetadata)(nil)

// CreateXMPMetadata returns an empty packet with the default xpacket
// declaration.
//
// Port of the static createXMPMetadata() and the protected XMPMetadata()
// constructor it calls.
func CreateXMPMetadata() *XMPMetadata {
	return CreateXMPMetadataOfXpacket(DefaultXpacketBegin, DefaultXpacketID,
		DefaultXpacketBytes, DefaultXpacketEncoding)
}

// CreateXMPMetadataOfXpacket returns an empty packet with the given xpacket
// declaration.
//
// Port of the static createXMPMetadata(String, String, String, String) and the
// protected constructor it calls.
func CreateXMPMetadataOfXpacket(xpacketBegin, xpacketID string, xpacketBytes *string,
	xpacketEncoding string) *XMPMetadata {
	m := &XMPMetadata{
		xpacketBegin:    xpacketBegin,
		xpacketID:       xpacketID,
		xpacketBytes:    xpacketBytes,
		xpacketEncoding: xpacketEncoding,
		xpacketEndData:  DefaultXpacketEnd,
	}
	m.typeMapping = xmptype.NewTypeMapping(m)
	return m
}

// TypeMapping returns the mapping from a type name to what implements it.
func (m *XMPMetadata) TypeMapping() *xmptype.TypeMapping { return m.typeMapping }

// XpacketBytes returns the byte count the xpacket declares, or nil.
func (m *XMPMetadata) XpacketBytes() *string { return m.xpacketBytes }

// XpacketEncoding returns the encoding the xpacket declares.
func (m *XMPMetadata) XpacketEncoding() string { return m.xpacketEncoding }

// XpacketBegin returns the byte order mark the xpacket opens with.
func (m *XMPMetadata) XpacketBegin() string { return m.xpacketBegin }

// XpacketID returns the packet identifier.
func (m *XMPMetadata) XpacketID() string { return m.xpacketID }

// SetEndXPacket sets what the closing xpacket instruction says.
func (m *XMPMetadata) SetEndXPacket(data string) { m.xpacketEndData = data }

// EndXPacket returns what the closing xpacket instruction says.
func (m *XMPMetadata) EndXPacket() string { return m.xpacketEndData }
