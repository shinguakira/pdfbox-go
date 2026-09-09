package xml_test

// JAVA-BUGS 56: `DomXmpParser.parseEndPacket` checks the four characters of
// `end=` and then reads the sixth, so a shorter instruction raises
// StringIndexOutOfBoundsException -- which is unchecked, and so escapes the
// XmpParsingException a caller catches.

import (
	"testing"

	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
)

// TestShortEndPacketIsRefused is the defect.
//
// Both packets below are malformed, and the answer a parser owes malformed
// input is its own error: the same one the sixth character gets when it is
// present and is neither 'r' nor 'w'.
func TestShortEndPacketIsRefused(t *testing.T) {
	const head = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + "\uFEFF" + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description rdf:about=""/>
	</rdf:RDF>
</x:xmpmeta>`

	for _, end := range []string{`<?xpacket end=?>`, `<?xpacket end="?>`} {
		_, err := xmpxml.NewDomXmpParser().ParseBytes([]byte(head + end))
		if err == nil {
			t.Errorf("the packet ending %q was accepted", end)
			continue
		}
		const want = "Expected xpacket 'end' attribute with value 'r' or 'w' "
		if err.Error() != want {
			t.Errorf("the packet ending %q was refused with %q, want %q", end, err, want)
		}
	}
}

// TestWellFormedEndPacketIsStillRead keeps the instruction the length check
// must not turn away.
func TestWellFormedEndPacketIsStillRead(t *testing.T) {
	const packet = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + "\uFEFF" + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description rdf:about=""/>
	</rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`

	xmp, err := xmpxml.NewDomXmpParser().ParseBytes([]byte(packet))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := xmp.EndXPacket(); got != "w" {
		t.Errorf("EndXPacket() = %q, want %q", got, "w")
	}
}
