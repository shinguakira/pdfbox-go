// Package xmptype holds the XMP type system: the fields, the simple and
// complex properties, and the mapping from a type name to the thing that
// implements it.
//
// Port of org.apache.xmpbox.type. The package is named xmptype rather than type
// because type is a Go keyword; migration/mapping/packages.tsv records the
// rename.
//
// Two classes Java keeps in org.apache.xmpbox are declared here instead:
// XmpConstants and DateConverter. Java's root package holds XMPMetadata, which
// every field points back at, and this package would have to import it for
// them; Go forbids the cycle, so they sit at the bottom of the stack where
// everything can reach them and the root package aliases both back to the Java
// name in the Java place. See migration/STATUS.md.
package xmptype

// The constants org.apache.xmpbox.XmpConstants declares.
//
// Port of XmpConstants, which Java declares final with a private constructor.
const (
	// RDFNamespace is the RDF namespace every XMP packet is written in.
	RDFNamespace = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"

	// DefaultXpacketBegin is the byte order mark an xpacket opens with, which
	// Java writes as a literal U+FEFF in the source.
	DefaultXpacketBegin = "\uFEFF"

	// DefaultXpacketID is the packet identifier the specification fixes.
	DefaultXpacketID = "W5M0MpCehiHzreSzNTczkc9d"

	// DefaultXpacketEncoding is the encoding an xpacket declares.
	DefaultXpacketEncoding = "UTF-8"

	// DefaultXpacketEnd says the packet may be written into.
	DefaultXpacketEnd = "w"

	// DefaultRDFPrefix is the prefix the RDF namespace is bound to.
	DefaultRDFPrefix = "rdf"

	// DefaultRDFLocalName is the local name of the RDF root element.
	DefaultRDFLocalName = "RDF"

	// ListName is the element name of an array item.
	ListName = "li"

	// LangName is the attribute name of a language qualifier.
	LangName = "lang"

	// AboutName is the attribute naming what a description is about.
	AboutName = "about"

	// DescriptionName is the element name of an RDF description.
	DescriptionName = "Description"

	// ResourceName is the parse type of a structured value.
	ResourceName = "Resource"

	// ParseType is the attribute naming how a value is parsed.
	ParseType = "parseType"

	// XDefault is the language of the default alternative.
	XDefault = "x-default"
)

// DefaultXpacketBytes is the byte count an xpacket declares, which Java
// declares as a null String rather than a value.
//
// It is a var because Go has no null string; nil means the same "no value the
// serialiser should write" Java's null does.
var DefaultXpacketBytes *string
