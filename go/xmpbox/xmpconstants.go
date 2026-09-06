package xmpbox

// The two root classes that had to move down a layer, aliased back to the Java
// name in the Java place.
//
// Port of org.apache.xmpbox.XmpConstants and org.apache.xmpbox.DateConverter,
// both declared in xmpbox/xmptype for the reason the package comment gives.
// This is the same device pdmodel.ResourceCache uses for pdmodel/font's.

import (
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// The constants XmpConstants declares.
const (
	// RDFNamespace is the RDF namespace every XMP packet is written in.
	RDFNamespace = xmptype.RDFNamespace

	// DefaultXpacketBegin is the byte order mark an xpacket opens with.
	DefaultXpacketBegin = xmptype.DefaultXpacketBegin

	// DefaultXpacketID is the packet identifier the specification fixes.
	DefaultXpacketID = xmptype.DefaultXpacketID

	// DefaultXpacketEncoding is the encoding an xpacket declares.
	DefaultXpacketEncoding = xmptype.DefaultXpacketEncoding

	// DefaultXpacketEnd says the packet may be written into.
	DefaultXpacketEnd = xmptype.DefaultXpacketEnd

	// DefaultRDFPrefix is the prefix the RDF namespace is bound to.
	DefaultRDFPrefix = xmptype.DefaultRDFPrefix

	// DefaultRDFLocalName is the local name of the RDF root element.
	DefaultRDFLocalName = xmptype.DefaultRDFLocalName

	// ListName is the element name of an array item.
	ListName = xmptype.ListName

	// LangName is the attribute name of a language qualifier.
	LangName = xmptype.LangName

	// AboutName is the attribute naming what a description is about.
	AboutName = xmptype.AboutName

	// DescriptionName is the element name of an RDF description.
	DescriptionName = xmptype.DescriptionName

	// ResourceName is the parse type of a structured value.
	ResourceName = xmptype.ResourceName

	// ParseType is the attribute naming how a value is parsed.
	ParseType = xmptype.ParseType

	// XDefault is the language of the default alternative.
	XDefault = xmptype.XDefault
)

// DefaultXpacketBytes is the byte count an xpacket declares, which Java
// declares as a null String.
var DefaultXpacketBytes = xmptype.DefaultXpacketBytes

// ToCalendar converts a string to a date.
//
// Port of DateConverter.toCalendar.
func ToCalendar(date string) (time.Time, error) { return xmptype.ToCalendar(date) }

// ToISO8601 converts a date to its ISO 8601 string, without milliseconds.
//
// Port of DateConverter.toISO8601(Calendar).
func ToISO8601(cal time.Time) string { return xmptype.ToISO8601(cal) }

// ToISO8601Millis converts a date to its ISO 8601 string, printMillis saying
// whether the milliseconds are written.
//
// Port of DateConverter.toISO8601(Calendar, boolean).
func ToISO8601Millis(cal time.Time, printMillis bool) string {
	return xmptype.ToISO8601Millis(cal, printMillis)
}
