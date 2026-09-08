package pdfparser_test

// The `pdfparser` half of `track/java-bug-fixes`.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// TestParseCOSNameKeepsAPrematureHash is JAVA-BUGS 8.
//
// A name that ends `/A#2` at the end of input loses both the `#` and the `2`,
// and parses as `A`. Every other malformed-escape path in the same loop keeps
// the `#` as a literal character -- the branch three lines below writes it --
// so the parser answers a name that is not what is on disk, silently.
//
// The expected value is the specification's: before PDF 1.2 the `#` was an
// ordinary character, and the parser's own comment says tools still emit it
// that way. So a `#` that is not followed by two hex digits is a `#`.
func TestParseCOSNameKeepsAPrematureHash(t *testing.T) {
	for _, c := range []struct{ input, want string }{
		{"/A#2", "A#2"},
		{"/A#", "A#"},
		// The complete escape still resolves: #41 is 'A'.
		{"/B#41", "BA"},
		// And a malformed escape that is not at the end already kept its hash.
		{"/A#zz ", "A#zz"},
	} {
		parser := pdfparser.NewObjectParser(pdfio.NewReadBufferBytes([]byte(c.input)), nil)
		name, err := parser.ParseCOSName()
		if err != nil {
			t.Errorf("%q: ParseCOSName: %v", c.input, err)
			continue
		}
		if got := name.Name(); got != c.want {
			t.Errorf("%q parses as /%s, want /%s", c.input, got, c.want)
		}
	}
}
