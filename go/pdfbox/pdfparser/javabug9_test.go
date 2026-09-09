package pdfparser_test

// JAVA-BUGS 9: `inlineImageDepth` is decremented only on the path that ends in
// a complete inline image, so a malformed one leaves it at 1 and every later
// `BI` in the same stream is refused as "nested".

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
)

// TestMalformedInlineImageDoesNotBlockTheNextOne is the defect: the first BI is
// malformed -- it has no ID -- and the second, which is well formed, must still
// be read. Java refuses it, because the counter never came back down.
//
// The expected behaviour is what the counter is for: it guards against a BI
// nested inside a BI (PDFBOX-6038), and these two are one after the other.
func TestMalformedInlineImageDoesNotBlockTheNextOne(t *testing.T) {
	// A BI whose dictionary ends in a string rather than an ID -- so the loop
	// leaves on a token that is not an operator -- and then a
	// complete one.
	const content = "BI /W 1 /H 1 (unexpected)\n" +
		"BI /W 1 /H 1 ID \x00 EI\n"

	parser, err := pdfparser.NewStreamTokenParser([]byte(content))
	if err != nil {
		t.Fatalf("NewStreamTokenParser: %v", err)
	}
	tokens, err := parser.Parse()
	if err != nil {
		if strings.Contains(err.Error(), "nested") {
			t.Fatalf("the second BI was refused as nested: %v", err)
		}
		t.Fatalf("ParseTokens: %v", err)
	}
	if len(tokens) == 0 {
		t.Fatal("nothing was parsed")
	}
}
