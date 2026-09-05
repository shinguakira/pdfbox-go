package action

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/action/PDActionURITest.java.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// mojibakeURI is the string the Java test passes to setURI, written out code
// point by code point: it is the UTF-8 bytes of a Japanese domain name read
// back as PDFDocEncoding, and getURI reading them as UTF-8 again is the round
// trip the test is about.
const mojibakeURI = "http://çµ„åŒ¶" +
	"æ›¿ç¶Ž.com/"

// TestUTF8URI is PDFBOX-3913: check that URIs encoded in UTF-8 are also
// supported, and PDFBOX-3946: check that there is no NPE if the URI is missing.
//
// Java asserts null for the missing URI; the port answers the empty string
// there, which is how every string accessor of this port writes null.
func TestUTF8URI(t *testing.T) {
	actionURI := NewPDActionURI()
	if got := actionURI.URI(); got != "" {
		t.Errorf("URI() = %q, want %q", got, "")
	}
	actionURI.SetURI(mojibakeURI)
	if got, want := actionURI.URI(), "http://経営承継.com/"; got != want {
		t.Errorf("URI() = %q, want %q", got, want)
	}
}

// TestUTF16BEURI is PDFBOX-3913: check that URIs encoded in UTF-16 big endian
// are also supported.
func TestUTF16BEURI(t *testing.T) {
	actionURI := NewPDActionURI()

	// found in govdocs file 534948.pdf
	utf16URI, err := cos.ParseHexString("FEFF0068007400740070003A002F002F00770077" +
		"0077002E006E00610070002E006500640075002F0063006100740061006C006F006700" +
		"2F00310031003100340030002E00680074006D006C")
	if err != nil {
		t.Fatal(err)
	}
	actionURI.ActionDictionary().SetItem(cos.URI, utf16URI)
	if got, want := actionURI.URI(), "http://www.nap.edu/catalog/11140.html"; got != want {
		t.Errorf("URI() = %q, want %q", got, want)
	}
}

// TestUTF16LEURI is PDFBOX-3913: check that URIs encoded in UTF-16 little
// endian are also supported.
func TestUTF16LEURI(t *testing.T) {
	actionURI := NewPDActionURI()

	utf16URI, err := cos.ParseHexString("FFFE68007400740070003A00")
	if err != nil {
		t.Fatal(err)
	}
	actionURI.ActionDictionary().SetItem(cos.URI, utf16URI)
	if got, want := actionURI.URI(), "http:"; got != want {
		t.Errorf("URI() = %q, want %q", got, want)
	}
}

func TestUTF7URI(t *testing.T) {
	actionURI := NewPDActionURI()
	actionURI.SetURI("http://pdfbox.apache.org/")
	if got, want := actionURI.URI(), "http://pdfbox.apache.org/"; got != want {
		t.Errorf("URI() = %q, want %q", got, want)
	}
}
