package xml_test

// What the serializer does with a property whose element had no prefix.
//
// Not a case in DomXmpParserTest, which only parses PDFBOX-5835. Serializing
// what it parses is where Java and the port part company, and this pins it. See
// migration/JAVA-BUGS.md 59 and the track's section in migration/STATUS.md.

import (
	"strings"
	"testing"
)

// TestSerializingAnUnprefixedPropertyWhereJavaFails writes back the packet
// PDFBOX-5835 is about, whose pdfaExtension properties are written without a
// prefix.
//
// Java's serializeFields reads `field.getPrefix().isEmpty()` on a prefix that
// is null for such a property and raises NullPointerException, so the packet
// cannot be written at all. The port's prefix is the empty string, so it writes
// the element under its local name.
func TestSerializingAnUnprefixedPropertyWhereJavaFails(t *testing.T) {
	metadata := parseFixture(t, "org/apache/xmpbox/xml/PDFBOX-5835.xml")
	written := string(serialized(t, metadata))

	for _, want := range []string{
		"<schemas>",
		"<schema>Some Schema</schema>",
	} {
		if !strings.Contains(written, want) {
			t.Errorf("serialized packet does not hold %q:\n%s", want, written)
		}
	}
	// The prefixed properties of the same packet are unaffected.
	if !strings.Contains(written, "<pdfaid:part>3</pdfaid:part>") {
		t.Errorf("serialized packet does not hold the pdfaid:part property:\n%s", written)
	}
}
