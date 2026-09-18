package xml_test

// Characters XML does not allow, where encoding/xml does not look for them.
//
// encoding/xml refuses a character outside XML's Char production in text and
// in attribute values, and reads a processing instruction or a comment through
// without looking. Xerces refuses one anywhere. Found by the corpus comparison of
// XMP schemas: cabinet-of-horrors/balloon_a1b_jp2k.pdf writes its packet header
// as <?xpacket begin="\0" ...?>, with a NUL byte, and PDFBox refuses the packet
// where the Go version read it.
//
// The expected values are PDFBox's, printed by DomXmpParser.parse for each of
// these byte sequences; the messages are Xerces's.

import (
	"sort"
	"strings"
	"testing"

	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
)

func packetWith(begin, comment, end string) []byte {
	var out strings.Builder
	out.WriteString(`<?xpacket begin="` + begin + `" id="W5M0MpCehiHzreSzNTczkc9d"?>` + "\n")
	out.WriteString(`<x:xmpmeta xmlns:x="adobe:ns:meta/">` + "\n")
	if comment != "" {
		out.WriteString("<!--" + comment + "-->\n")
	}
	out.WriteString(`<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` + "\n" +
		`<rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/">` + "\n" +
		"<dc:format>application/pdf</dc:format>\n" +
		"</rdf:Description>\n</rdf:RDF>\n</x:xmpmeta>\n" +
		`<?xpacket end="` + end + `"?>`)
	return []byte(out.String())
}

func TestParseRefusesCharactersXMLDoesNotAllow(t *testing.T) {
	bom := "\xEF\xBB\xBF"
	for _, c := range []struct {
		what  string
		bytes []byte
		err   string // empty where PDFBox reads the packet
	}{
		{"U+FEFF in a processing instruction", packetWith(bom, "", "w"), ""},
		{"U+0000 in a processing instruction", packetWith("\x00", "", "w"),
			"An invalid XML character (Unicode: 0x0) was found in the processing instruction."},
		{"U+0008 in a processing instruction", packetWith("\x08", "", "w"),
			"An invalid XML character (Unicode: 0x8) was found in the processing instruction."},
		{"U+0009 in a processing instruction", packetWith("\x09", "", "w"), ""},
		{"U+001F in a processing instruction", packetWith("\x1F", "", "w"),
			"An invalid XML character (Unicode: 0x1f) was found in the processing instruction."},
		{"U+FFFE in a processing instruction", packetWith("\xEF\xBF\xBE", "", "w"),
			"An invalid XML character (Unicode: 0xfffe) was found in the processing instruction."},
		{"U+FFFF in a processing instruction", packetWith("\xEF\xBF\xBF", "", "w"),
			"An invalid XML character (Unicode: 0xffff) was found in the processing instruction."},
		{"U+10000 in a processing instruction", packetWith("\xF0\x90\x80\x80", "", "w"), ""},
		// "Invalid byte 1 of 1-byte UTF-8 sequence."
		{"the byte FF in a processing instruction", packetWith("\xFF", "", "w"), "UTF-8"},
		// "Invalid byte 2 of 2-byte UTF-8 sequence."
		{"a lone C3 in a processing instruction", packetWith("\xC3", "", "w"), "UTF-8"},
		// "Invalid byte 2 of 3-byte UTF-8 sequence."
		{"U+D800 written as UTF-8 in a processing instruction", packetWith("\xED\xA0\x80", "", "w"), "UTF-8"},
		{"U+0000 in the closing processing instruction", packetWith(bom, "", "\x00"),
			"An invalid XML character (Unicode: 0x0) was found in the processing instruction."},
		{"a comment", packetWith(bom, "a", "w"), ""},
		{"U+0000 in a comment", packetWith(bom, "a\x00b", "w"),
			"An invalid XML character (Unicode: 0x0) was found in the comment."},
		{"U+0001 in a comment", packetWith(bom, "\x01", "w"),
			"An invalid XML character (Unicode: 0x1) was found in the comment."},
		{"U+FFFE in a comment", packetWith(bom, "\xEF\xBF\xBE", "w"),
			"An invalid XML character (Unicode: 0xfffe) was found in the comment."},
		// "Invalid byte 1 of 1-byte UTF-8 sequence."
		{"the byte FF in a comment", packetWith(bom, "\xFF", "w"), "UTF-8"},
		{"U+0085 in a comment", packetWith(bom, "\xC2\x85", "w"), ""},
		{"U+007F in a comment", packetWith(bom, "\x7F", "w"), ""},
	} {
		metadata, err := xmpxml.NewDomXmpParser().ParseBytes(c.bytes)
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%s gave %v, want PDFBox's %q", c.what, err, c.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v, and PDFBox reads the packet", c.what, err)
			continue
		}
		var got []string
		for _, schema := range metadata.AllSchemas() {
			got = append(got, schema.Namespace())
		}
		sort.Strings(got)
		if want := "http://purl.org/dc/elements/1.1/"; strings.Join(got, " ") != want {
			t.Errorf("%s holds the schemas %v, want %s", c.what, got, want)
		}
	}
}
