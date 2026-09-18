package xml_test

// What the parser makes of a packet by the encoding it is written in.
//
// Java's DocumentBuilder is Xerces, which reads the first bytes of a document
// to tell its encoding before it reads any markup: a byte order mark for UTF-8
// or UTF-16, the pattern of "<?" in UTF-16 or UCS-4 without one, and after that
// the encoding the XML declaration names. Found by the corpus comparison of
// XMP schemas: 69 veraPDF files whose packet PDFBox reads and the Go version
// did not, 56 of them starting with a UTF-8 byte order mark and 13 written in
// UTF-16 or UCS-4.
//
// The expected values are PDFBox's, printed by DomXmpParser.parse for each of
// these byte sequences, built from the same packet.

import (
	"encoding/binary"
	"sort"
	"strings"
	"testing"
	"unicode/utf16"

	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
)

const encodedPacket = "<?xpacket begin='" + string(rune(0xFEFF)) + "' id='W5M0MpCehiHzreSzNTczkc9d'?>\n" +
	"<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">\n" +
	"<rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n" +
	"<rdf:Description rdf:about=\"\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n" +
	"<dc:format>application/pdf</dc:format>\n" +
	"</rdf:Description>\n" +
	"</rdf:RDF>\n" +
	"</x:xmpmeta>\n" +
	"<?xpacket end='w'?>"

func utf16Of(s string, order binary.AppendByteOrder) []byte {
	var out []byte
	for _, unit := range utf16.Encode([]rune(s)) {
		out = order.AppendUint16(out, unit)
	}
	return out
}

func ucs4Of(s string, order binary.AppendByteOrder) []byte {
	var out []byte
	for _, r := range s {
		out = order.AppendUint32(out, uint32(r))
	}
	return out
}

func joined(parts ...[]byte) []byte {
	var out []byte
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func TestParseDetectsEncoding(t *testing.T) {
	bomUTF8 := []byte{0xEF, 0xBB, 0xBF}
	bomBE := []byte{0xFE, 0xFF}
	bomLE := []byte{0xFF, 0xFE}
	declared16 := `<?xml version="1.0" encoding="UTF-16"?>` + encodedPacket
	declared8 := `<?xml version="1.0" encoding="UTF-8"?>` + encodedPacket
	be, le := binary.BigEndian, binary.LittleEndian

	// what PDFBox reads out of every packet it parses
	dc := []string{"http://purl.org/dc/elements/1.1/"}

	for _, c := range []struct {
		what  string
		bytes []byte
		want  []string // nil where PDFBox refuses the packet
	}{
		{"UTF-8", []byte(encodedPacket), dc},
		{"UTF-8 after a byte order mark", joined(bomUTF8, []byte(encodedPacket)), dc},
		// "Content is not allowed in prolog": only the first mark is one
		{"UTF-8 after two byte order marks", joined(bomUTF8, bomUTF8, []byte(encodedPacket)), nil},
		{"UTF-8 after a byte order mark, declared UTF-8", joined(bomUTF8, []byte(declared8)), dc},
		{"UTF-16BE after a byte order mark", joined(bomBE, utf16Of(encodedPacket, be)), dc},
		{"UTF-16LE after a byte order mark", joined(bomLE, utf16Of(encodedPacket, le)), dc},
		{"UTF-16BE", utf16Of(encodedPacket, be), dc},
		{"UTF-16LE", utf16Of(encodedPacket, le), dc},
		{"UTF-16BE after a byte order mark, declared UTF-16", joined(bomBE, utf16Of(declared16, be)), dc},
		{"UTF-16LE after a byte order mark, declared UTF-16", joined(bomLE, utf16Of(declared16, le)), dc},
		{"UTF-16LE declared UTF-16", utf16Of(declared16, le), dc},
		// "Content is not allowed in prolog": the declaration switches to UTF-8
		{"UTF-16BE declared UTF-8", utf16Of(declared8, be), nil},
		{"UCS-4BE", ucs4Of(encodedPacket, be), dc},
		{"UCS-4LE", ucs4Of(encodedPacket, le), dc},
		// "Invalid byte 1 of 1-byte UTF-8 sequence": no mark is looked for in UCS-4
		{"UCS-4BE after a byte order mark", joined([]byte{0, 0, 0xFE, 0xFF}, ucs4Of(encodedPacket, be)), nil},
		// "Content is not allowed in prolog": FF FE is read as UTF-16LE's mark
		{"UCS-4LE after a byte order mark", joined([]byte{0xFF, 0xFE, 0, 0}, ucs4Of(encodedPacket, le)), nil},
		{"UTF-8 declared ISO-8859-1", []byte(`<?xml version="1.0" encoding="ISO-8859-1"?>` + encodedPacket), dc},
		// "Content is not allowed in prolog"
		{"UTF-8 after a byte order mark, declared UTF-16", joined(bomUTF8, []byte(declared16)), nil},
	} {
		metadata, err := xmpxml.NewDomXmpParser().ParseBytes(c.bytes)
		if c.want == nil {
			if err == nil {
				t.Errorf("%s parsed, and PDFBox refuses it", c.what)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v, and PDFBox reads %v", c.what, err, c.want)
			continue
		}
		var got []string
		for _, schema := range metadata.AllSchemas() {
			got = append(got, schema.Namespace())
		}
		sort.Strings(got)
		if strings.Join(got, " ") != strings.Join(c.want, " ") {
			t.Errorf("%s holds the schemas %v, want %v", c.what, got, c.want)
		}
	}
}
